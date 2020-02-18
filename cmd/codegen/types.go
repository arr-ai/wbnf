package codegen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/arr-ai/wbnf/ast"

	"github.com/iancoleman/strcase"

	"github.com/arr-ai/wbnf/wbnf"
)

func GoTypeName(rule string) string {
	return strcase.ToCamel(rule + "Node")
}
func DropCaps(rule string) string {
	return strings.ToLower(rule)
}

type (
	grammarType interface {
		TypeName() string
		String() string
		Children() []grammarType
	}
	basicRule  string // Used for rules which only return an unnamed string (i.e foo -> /{[a-z]*}; )
	choice     string // val is parent name
	namedToken struct {
		name, parent string
		oneOnly      bool
	}
	namedRule struct {
		name, parent string
		oneOnly      bool
	}
	rule struct {
		name   string
		childs []grammarType
	} // used for the common rules
)

func (t basicRule) TypeName() string        { return "" }
func (t basicRule) Children() []grammarType { return nil }
func (t basicRule) String() string {
	return fmt.Sprintf(`
func (c %s) String() string {
	if c.Node == nil { return "" }
	return c.Node.Scanner().String()
}
`, string(t))
}

func (t choice) TypeName() string        { return "" }
func (t choice) Children() []grammarType { return nil }
func (t choice) String() string {
	return fmt.Sprintf(`
func (c %s) Choice() int {
	return ast.Choice(c.Node)
}
`, string(t))
}

func (t namedToken) TypeName() string        { return "" /* not exported */ }
func (t namedToken) Children() []grammarType { return nil }
func (t namedToken) String() string {
	replacer := strings.NewReplacer("{{parent}}", t.parent,
		"{{childtype}}", strcase.ToCamel(t.name),
		"{{name}}", t.name,
	)
	if t.oneOnly {
		return replacer.Replace(`
func (c {{parent}}) One{{childtype}}() string {
	if child := ast.First(c.Node, "{{name}}"); child != nil {
		return ast.First(child, "").Scanner().String()
	}
	return ""
}
`)
	}
	return replacer.Replace(`
func (c {{parent}}) All{{childtype}}() []string {
	var out []string
	for _, child := range ast.All(c.Node, "{{name}}") {
		out = append(out, ast.First(child, "").Scanner().String())
	}
	return out
}
`)
}

func (t namedRule) TypeName() string        { return "" /* not exported */ }
func (t namedRule) Children() []grammarType { return nil }
func (t namedRule) String() string {
	replacer := strings.NewReplacer("{{parent}}", t.parent,
		"{{child}}", strcase.ToCamel(t.name),
		"{{childtype}}", GoTypeName(t.name),
		"{{name}}", t.name,
	)
	if t.oneOnly {
		return replacer.Replace(`
func (c {{parent}}) One{{child}}() {{childtype}} {
	if child := ast.First(c.Node, "{{name}}"); child != nil {
		return {{childtype}}{child}
	}
	return ""
}
`)
	}
	return replacer.Replace(`func (c {{parent}}) All{{child}}() []{{childtype}} {
	var out []string
	for _, child := range ast.All(c.Node, "{{name}}") {
		out = append(out, {{childtype}}{child})
	}
	return out
}

`)
}

func (t rule) TypeName() string        { return t.name }
func (t rule) Children() []grammarType { return t.childs }
func (t rule) String() string {
	out := fmt.Sprintf(`
type %s struct { ast.Node}
`, t.TypeName())

	if len(t.Children()) > 0 {
		funcs := make([]string, 0, len(t.Children()))
		for _, child := range t.Children() {
			funcs = append(funcs, child.String())
		}
		sort.Strings(funcs)

		walker := strings.ReplaceAll(`func Walk{{.CtxName}}(node {{.CtxName}}, ops WalkerOps) Stopper {
	if fn := ops.Enter{{.CtxName}}; fn != nil {
		s := fn(node)
		switch {
			case s == nil:
			case s.ExitNode():
				return nil
			case s.Abort():
				return s
		}
}
`, "{{.CtxName}}", t.TypeName())
		for _, child := range t.Children() {
			if name := child.TypeName(); name != "" {
				walker += strings.ReplaceAll(`
for _, child := range node.All{{}}() {
	s := Walk{{}}(child, ops)
		switch {
			case s == nil:
			case s.ExitNode():
				return nil
			case s.Abort():
				return s
		}
}`, "{{}}",
					name)
			}
		}
		walker += strings.ReplaceAll(`
if fn := ops.Exit{{.CtxName}}; fn != nil {
	if s := fn(node); s != nil && s.Abort() { return s }
}
	return nil
}
`,
			"{{.CtxName}}", t.TypeName())
		return out + strings.Join(funcs, "\n") + walker
	}
	return out
}

type data struct {
	prefix string
	types  map[string]grammarType

	prodName    string
	prodIdents  []grammarType
	choiceCount int
}

func (d *data) get() []fmt.Stringer {
	keys := make([]string, 0, len(d.types))
	for rule := range d.types {
		keys = append(keys, rule)
	}
	sort.Strings(keys)

	result := make([]fmt.Stringer, 0, len(keys))
	for _, k := range keys {
		result = append(result, d.types[k])
	}
	return result
}

func nameFromAtom(atom wbnf.AtomNode) string {
	x, _ := ast.Which(atom.Node.(ast.Branch), "RE", "STR", "IDENT", "REF", "term")
	name := ""
	switch x {
	case "REF":
		name = atom.One("IDENT").Scanner().String()
	case "IDENT":
		name = atom.One(x).Scanner().String()
	case "term":
		name = x
	}
	return name
}

func (d *data) handleTerm(term wbnf.TermNode) wbnf.Stopper {
	if term.OneOp() == "|" {
		if d.choiceCount == 0 {
			d.prodIdents = append(d.prodIdents, choice(d.prodName))
		} else {
			d.prodIdents = append(d.prodIdents, choice(fmt.Sprintf("%s%d", d.prodName, d.choiceCount)))
		}
		d.choiceCount++
	}
	if named := term.OneNamed(); named.Node != nil {
		quant := len(term.AllQuant())
		target := nameFromAtom(named.OneAtom())
		if target != "" {
			target = GoTypeName(DropCaps(d.prefix + target))
		}
		if ident := named.OneIdent().String(); ident != "" {
			d.prodIdents = append(d.prodIdents, namedRule{
				oneOnly: quant == 0,
				name:    target,
				parent:  d.prodName,
			})
			if _, has := d.types[target]; !has {
				d.types[target] = rule{name: target}
			}
		} else if target == "" {
			d.prodIdents = append(d.prodIdents, basicRule(d.prodName))
		} else {
			d.prodIdents = append(d.prodIdents, namedToken{
				oneOnly: quant == 0,
				name:    target,
				parent:  d.prodName,
			})
		}
	}
	return nil
}

func (d *data) handleProd(prod wbnf.ProdNode) wbnf.Stopper {
	name := prod.OneIdent().String()
	d.prodName = GoTypeName(DropCaps(d.prefix + name))
	d.prodIdents = []grammarType{}
	d.choiceCount = 0

	return nil
}

func (d *data) finishProd(prod wbnf.ProdNode) wbnf.Stopper {
	var val grammarType
	switch len(d.prodIdents) {
	case 0:
	case 1:
		switch x := d.prodIdents[0].(type) {
		case basicRule:
			val = x
		}
		fallthrough
	default:
		val = rule{name: d.prodName, childs: d.prodIdents}
	}

	if _, has := d.types[d.prodName]; !has {
		d.types[d.prodName] = val
	} else {
		panic("oops")
	}
	return nil
}

func MakeTypes(prefix string, node wbnf.GrammarNode) []fmt.Stringer {
	d := &data{prefix: prefix, types: map[string]grammarType{}}
	wbnf.WalkerOps{EnterProdNode: d.handleProd,
		ExitProdNode:  d.finishProd,
		EnterTermNode: d.handleTerm}.Walk(node)

	return d.get()
}
