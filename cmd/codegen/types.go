package codegen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/iancoleman/strcase"

	"github.com/arr-ai/wbnf/wbnf"
)

func GoTypeName(rule string) string {
	return strcase.ToCamel(strings.TrimSuffix(rule, "Node") + "Node")
}
func DropCaps(rule string) string {
	return strings.ToLower(rule)
}

type (
	grammarType interface {
		TypeName() string
		Ident() string
		String() string
		Children() []grammarType
	}
	basicRule string // Used for rules which only return an unnamed string (i.e foo -> /{[a-z]*}; )
	choice    struct {
		parent      string
		returnTypes []string
	}
	stackBackRef struct {
		name, parent string
	}
	backRef struct {
		name, parent string
	}
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

func (t basicRule) TypeName() string        { return string(t) }
func (t basicRule) Ident() string           { return "String" }
func (t basicRule) Children() []grammarType { return nil }
func (t basicRule) String() string {
	return fmt.Sprintf(`
type %s struct { ast.Node }
func (c %s) String() string {
	if c.Node == nil { return "" }
	return c.Node.Scanner().String()
}
`, t.TypeName(), t.TypeName())
}

func (t choice) TypeName() string        { return "" }
func (t choice) Ident() string           { return "@choice" }
func (t choice) Children() []grammarType { return nil }
func (t choice) String() string {
	parentType := GoTypeName(t.parent)
	/*
		var buf = bytes.Buffer{}
		fmt.Fprintf(&buf, "type %sChoiceOptions interface { is%sOption() }\n", parentType, parentType)
		for _, c := range t.returnTypes {
			fmt.Fprintf(&buf, "func (%s) is%sOption(){}\n", c, parentType)
		}

		fmt.Fprintf(&buf, "func (c %s) Choice() %sChoiceOptions {\n"+
			"switch ast.Choice(c.Node) {\n", parentType, parentType)

		for i, c := range t.returnTypes {
			fmt.Fprintf(&buf,"\tcase %d: "
		}*/
	return fmt.Sprintf("func (c %s) Choice() int { return ast.Choice(c.Node) }\n", parentType)
}

func (t stackBackRef) TypeName() string        { return "" }
func (t stackBackRef) Ident() string           { return t.name }
func (t stackBackRef) Children() []grammarType { return nil }
func (t stackBackRef) String() string {
	return namedRule{
		name:       t.name,
		parent:     t.parent,
		returnType: GoTypeName(t.parent),
		count:      wantAllGetter,
	}.String()
}

func (t backRef) TypeName() string        { return "" }
func (t backRef) Ident() string           { return t.name }
func (t backRef) Children() []grammarType { return nil }
func (t backRef) String() string {
	return fmt.Sprintf(`func (c %s) %sRef() ast.Node { return ast.First(c.Node, %s) }
`,
		GoTypeName(t.parent), strcase.ToCamel(t.name), t.name)
}

func (t namedToken) TypeName() string        { return "" /* not exported */ }
func (t namedToken) Ident() string           { return t.name }
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
func (t unnamedToken) Ident() string           { return "Token" }
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
func (t namedRule) Ident() string           { return t.name }
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
func (t rule) Ident() string           { return t.name }
func (t rule) Children() []grammarType { return t.childs }
func (t rule) String() string {
	out := fmt.Sprintf(`
type %s struct { ast.Node}
`, t.TypeName())

	if len(t.Children()) > 0 {
		orderedChildren := t.Children()
		sort.Slice(orderedChildren, func(i, j int) bool {
			return strings.Compare(strings.ToUpper(orderedChildren[i].Ident()),
				strings.ToUpper(orderedChildren[j].Ident())) < 0
		})
		funcs := make([]string, 0, len(t.Children()))
		for _, child := range orderedChildren {
			funcs = append(funcs, child.String())
		}

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
	types map[string]grammarType
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

func MakeTypes(node wbnf.GrammarNode) *data {
	return &data{types: MakeTypesFromGrammar(wbnf.NewFromAst(node))}
}
