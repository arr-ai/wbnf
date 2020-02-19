package codegen

import (
	"fmt"
	"sort"
	"strconv"
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
func DropNodeSuffix(typename string) string {
	return strings.TrimSuffix(typename, "Node")
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
		count        int
	}
	unnamedToken struct {
		parent string
		count  int
	}
	namedRule struct {
		name, parent, returnType string
		count                    int
	}
	rule struct {
		name   string
		childs []grammarType
	} // used for the common rules
)

const (
	wantOneGetter int = 1 << iota
	wantAllGetter
)

func wantOneFn(count int) bool { return count&wantOneGetter != 0 }
func wantAllFn(count int) bool { return count&wantAllGetter != 0 }

func (t basicRule) TypeName() string        { return GoTypeName(string(t)) }
func (t basicRule) Children() []grammarType { return nil }
func (t basicRule) String() string {
	return fmt.Sprintf(`
func (c %s) String() string {
	if c.Node == nil { return "" }
	return c.Node.Scanner().String()
}
`, t.TypeName())
}

func (t choice) TypeName() string        { return "" }
func (t choice) Children() []grammarType { return nil }
func (t choice) String() string {
	return fmt.Sprintf(`
func (c %s) Choice() int {
	return ast.Choice(c.Node)
}
`, GoTypeName(string(t)))
}

func (t namedToken) TypeName() string        { return "" /* not exported */ }
func (t namedToken) Children() []grammarType { return nil }
func (t namedToken) String() string {
	replacer := strings.NewReplacer("{{parent}}", GoTypeName(t.parent),
		"{{childtype}}", strcase.ToCamel(t.name),
		"{{name}}", t.name,
	)
	out := ""
	if wantOneFn(t.count) {
		out += replacer.Replace(`
func (c {{parent}}) One{{childtype}}() string {
	if child := ast.First(c.Node, "{{name}}"); child != nil {
		return ast.First(child, "").Scanner().String()
	}
	return ""
}
`)
	}
	if wantAllFn(t.count) {
		out += replacer.Replace(`
func (c {{parent}}) All{{childtype}}() []string {
	var out []string
	for _, child := range ast.All(c.Node, "{{name}}") {
		out = append(out, ast.First(child, "").Scanner().String())
	}
	return out
}
`)
	}
	return out
}

func (t unnamedToken) TypeName() string        { return "" /* not exported */ }
func (t unnamedToken) Children() []grammarType { return nil }
func (t unnamedToken) String() string {
	replacer := strings.NewReplacer("{{parent}}", GoTypeName(t.parent))
	out := ""
	if wantOneFn(t.count) {
		out += replacer.Replace(`
func (c {{parent}}) OneToken() string {
	if child := ast.First(c.Node, ""); child != nil {
		return child.Scanner().String()
	}
	return ""
}
`)
	}
	if wantAllFn(t.count) {
		out += replacer.Replace(`
func (c {{parent}}) AllToken() []string {
	var out []string
	for _, child := range ast.All(c.Node, "") {
		out = append(out, child.Scanner().String())
	}
	return out
}
`)
	}
	return out
}

func (t namedRule) TypeName() string        { return "" /* not exported */ }
func (t namedRule) Children() []grammarType { return nil }
func (t namedRule) String() string {
	replacer := strings.NewReplacer("{{parent}}", GoTypeName(t.parent),
		"{{child}}", strcase.ToCamel(t.name),
		"{{returnType}}", t.returnType,
		"{{name}}", t.name,
	)
	out := ""
	if wantOneFn(t.count) {
		out += replacer.Replace(`
func (c {{parent}}) One{{child}}() {{returnType}} {
	if child := ast.First(c.Node, "{{name}}"); child != nil {
		return {{returnType}}{child}
	}
	return ""
}
`)
	}
	if wantAllFn(t.count) {
		out += replacer.Replace(`func (c {{parent}}) All{{child}}() []{{returnType}} {
	var out []{{returnType}}
	for _, child := range ast.All(c.Node, "{{name}}") {
		out = append(out, {{returnType}}{child})
	}
	return out
}

`)
	}
	return out
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

	prodName   string
	prodIdents []grammarType
}

func (d *data) Get() []fmt.Stringer {
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

const (
	tokenTarget int = iota
	ruleTarget
	termTarget
)

func nameFromAtom(atom wbnf.AtomNode) (string, int) {
	x, _ := ast.Which(atom.Node.(ast.Branch), "RE", "STR", "IDENT", "REF", "term")
	name := ""
	targetType := tokenTarget
	switch x {
	case "REF", "IDENT":
		name = atom.One("IDENT").Scanner().String()
		targetType = ruleTarget
	case "term":
		name = x
		targetType = termTarget
	}
	return name, targetType
}

func (d *data) handleProd(prod wbnf.ProdNode) wbnf.Stopper {
	name := prod.OneIdent().String()
	d.prodName = d.prefix + strcase.ToCamel(DropCaps(name))

	if len(prod.AllTerm()) == 1 {
		newTypes, _ := MakeTypesForTerms(d.prodName, prod.AllTerm()[0])
		for k, v := range newTypes {
			d.types[k] = v
		}
	} else {
		for i, term := range prod.AllTerm() {
			newTypes, _ := MakeTypesForTerms(d.prodName+strconv.Itoa(i), term)
			for k, v := range newTypes {
				d.types[k] = v
			}
		}
	}

	return nil
}

func MakeTypes(prefix string, node wbnf.GrammarNode) *data {
	d := &data{prefix: prefix, types: map[string]grammarType{}}
	wbnf.WalkerOps{EnterProdNode: d.handleProd}.Walk(node)

	return d
}
