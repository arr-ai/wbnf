package codegen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/iancoleman/strcase"

	"github.com/arr-ai/wbnf/wbnf"
)

var gotypemap map[string]string

func GoTypeName(rule string) string {
	return GoName(rule) + "Node"
}

func GoName(rule string) string {
	if strings.HasSuffix(rule, "Node") {
		return strings.TrimSuffix(rule, "Node")
	}
	if gotypemap == nil {
		gotypemap = map[string]string{}
	}

	if val, has := gotypemap[rule]; has {
		return val
	}

	res := strcase.ToCamel(DropCaps(rule))
	gotypemap[rule] = res
	return res
}

func DropCaps(rule string) string {
	isCaps := func(r uint8) bool { return r >= 'A' && r <= 'Z' }
	out := make([]string, 0, len(rule))
	for i := 0; i < len(rule); i++ {
		out = append(out, string(rule[i]))
		if isCaps(rule[i]) {
			for i+1 < len(rule) && isCaps(rule[i+1]) {
				i++
				out = append(out, strings.ToLower(string(rule[i])))
			}
		}
	}

	return strings.Join(out, "")
}

type (
	callbackData struct {
		getter, walker string
		isMany         bool
	}
	grammarType interface {
		TypeName() string
		Ident() string
		String() string
		Children() []grammarType
		CallbackData() *callbackData
	}
	basicRule string // Used for rules which only return an unnamed string (i.e foo -> /{[a-z]*}; )
	choice    struct {
		parent string
	}
	stackBackRef struct {
		name, parent string
	}
	backRef struct {
		name, parent string
	}
	namedToken struct {
		name, parent string
		count        countManager
	}
	unnamedToken struct {
		parent string
		count  countManager
	}
	namedRule struct {
		name, parent, returnType string
		count                    countManager
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

// This little struct is used to figure out if any particular ident requires the AllName() func or just the OneName()
// all the following rules clearly can only ever have a single value for x, so only the One*() is needed
// 		a -> x=IDENT;      				obvious
//		a -> x="hello" | x="goodbye";	only one x on either side of the |
//		a -> FOO:x=",";					even though there will be many delims, they must all be the same value
// Other combinations will require the AllName() because it is ambigous which term the user is asking for.
//      a -> x="a"+ | x="b"				one option wants only one, the other wants at least one.
//  etc.
// To track this, each countManager keeps a set of identifiers for each branch of a term the ident is used in,
// If each term is unique, and they all want just One, then the result should only require a One.
type countManager struct {
	int
	nodes []int
}

func (c countManager) forceMany() countManager {
	c.int |= wantAllGetter
	return c
}
func (c countManager) pushSingleNode(id int) countManager {
	if !c.wantAll() {
		for _, x := range c.nodes {
			if x == id {
				return c.forceMany()
			}
		}
		c.nodes = append(c.nodes, id)
	}
	return c
}

func (c countManager) merge(other countManager) countManager {
	if other.wantAll() {
		return c.forceMany()
	}
	for _, id := range other.nodes {
		c = c.pushSingleNode(id)
	}
	return c
}
func (c countManager) wantAll() bool { return c.int&wantAllGetter != 0 }
func (c countManager) wantOne() bool { return c.int&wantOneGetter != 0 }

func setWantOneGetter() countManager { return countManager{int: wantOneGetter} }
func setWantAllGetter() countManager { return countManager{int: wantAllGetter} }

func (t basicRule) TypeName() string        { return string(t) }
func (t basicRule) Ident() string           { return "String" }
func (t basicRule) Children() []grammarType { return nil }
func (t basicRule) String() string {
	return strings.ReplaceAll(`
type %s struct { ast.Node }
func (%s) isWalkableType() {}
func (c *%s) String() string {
	if c == nil || c.Node == nil { return "" }
	return c.Node.Scanner().String()
}
`, "%s", GoTypeName(string(t)))
}
func (t basicRule) CallbackData() *callbackData { return nil }
func (t basicRule) Upgrade() unnamedToken {
	return unnamedToken{
		parent: string(t),
		count:  countManager{int: wantOneGetter},
	}
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
func (t choice) CallbackData() *callbackData { return nil }

func (t stackBackRef) TypeName() string        { return "" }
func (t stackBackRef) Ident() string           { return t.name }
func (t stackBackRef) Children() []grammarType { return nil }
func (t stackBackRef) toNamedRule() grammarType {
	return namedRule{
		name:       t.name,
		parent:     t.parent,
		returnType: t.parent,
		count:      countManager{int: wantAllGetter},
	}
}
func (t stackBackRef) String() string {
	return t.toNamedRule().String()
}
func (t stackBackRef) CallbackData() *callbackData {
	return t.toNamedRule().CallbackData()
}

func (t backRef) TypeName() string        { return "" }
func (t backRef) Ident() string           { return t.name }
func (t backRef) Children() []grammarType { return nil }
func (t backRef) String() string {
	return fmt.Sprintf(`func (c %s) %sRef() ast.Node { return ast.First(c.Node, "%s") }
`,
		GoTypeName(t.parent), GoName(t.name), t.name)
}
func (t backRef) CallbackData() *callbackData { return nil }

func (t namedToken) TypeName() string        { return "" /* not exported */ }
func (t namedToken) Ident() string           { return t.name }
func (t namedToken) Children() []grammarType { return nil }
func (t namedToken) String() string {
	replacer := strings.NewReplacer("{{parent}}", GoTypeName(t.parent),
		"{{childtype}}", GoName(t.name),
		"{{name}}", IdentName(t.name),
	)
	out := ""
	if t.count.wantOne() {
		out += replacer.Replace(`
func (c {{parent}}) One{{childtype}}() string {
	if child := ast.First(c.Node, {{name}}); child != nil {
		return ast.First(child, "").Scanner().String()
	}
	return ""
}
`)
	}
	if t.count.wantAll() {
		out += replacer.Replace(`
func (c {{parent}}) All{{childtype}}() []string {
	var out []string
	for _, child := range ast.All(c.Node, {{name}}) {
		out = append(out, ast.First(child, "").Scanner().String())
	}
	return out
}
`)
	}
	return out
}
func (t namedToken) CallbackData() *callbackData { return nil }

func (t unnamedToken) TypeName() string        { return "" /* not exported */ }
func (t unnamedToken) Ident() string           { return "Token" }
func (t unnamedToken) Children() []grammarType { return nil }
func (t unnamedToken) String() string {
	replacer := strings.NewReplacer("{{parent}}", GoTypeName(t.parent))
	out := ""
	// a rule like: x -> 'a' | 'b' | 'c'; would expect to have an @choice rule and a single unnamedToken() rule
	// Because the options all only have a single token the code will generate a getter for the "" named tree node
	// However, this will fail because in this case the ast not would look like [@choice: 0, RULE_NAME: 'a']
	// instead of simply ["": 'a'] which is what would be expected.
	// To solve this problem the second if block is added to catch this.
	// A better solution would be to do `ast.First(ast.First(c.Node, "TheRule").Node, "") but the codegen loses
	// the rule name, this solution is Good Enough(TM)
	if t.count.wantOne() {
		out += replacer.Replace(`
func (c {{parent}}) OneToken() string {
	if child := ast.First(c.Node, ""); child != nil {
		return child.Scanner().String()
	}
	if b, ok := c.Node.(ast.Branch); ok && len(b) == 1 {
		for _, c := range b {
			if child := ast.First(c.(ast.One).Node, ""); child != nil {
				return child.Scanner().String()
			}
		}
	}
	return ""
}
`)
	}
	if t.count.wantAll() {
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
func (t unnamedToken) CallbackData() *callbackData { return nil }

func (t namedRule) TypeName() string        { return "" /* not exported */ }
func (t namedRule) Ident() string           { return t.name }
func (t namedRule) Children() []grammarType { return nil }
func (t namedRule) String() string {
	replacer := strings.NewReplacer("{{parent}}", GoTypeName(t.parent),
		"{{child}}", GoName(t.name),
		"{{returnType}}", GoTypeName(t.returnType),
		"{{name}}", IdentName(t.name),
	)
	out := ""
	if t.count.wantOne() {
		out += replacer.Replace(`
func (c {{parent}}) One{{child}}() *{{returnType}} {
	if child := ast.First(c.Node, {{name}}); child != nil {
		return &{{returnType}}{child}
	}
	return nil
}
`)
	}
	if t.count.wantAll() {
		out += replacer.Replace(`func (c {{parent}}) All{{child}}() []{{returnType}} {
	var out []{{returnType}}
	for _, child := range ast.All(c.Node, {{name}}) {
		out = append(out, {{returnType}}{child})
	}
	return out
}

`)
	}
	return out
}
func (t namedRule) CallbackData() *callbackData {
	return &callbackData{getter: GoName(t.name), walker: GoTypeName(t.returnType), isMany: t.count.wantAll()}
}

func (t rule) TypeName() string        { return t.name }
func (t rule) Ident() string           { return t.name }
func (t rule) Children() []grammarType { return t.childs }
func (t rule) String() string {
	out := fmt.Sprintf("type %s struct { ast.Node}\n func (%s) isWalkableType() {}\n",
		GoTypeName(t.name), GoTypeName(t.name))
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
		out += strings.Join(funcs, "\n")
	}
	return out
}
func (t rule) CallbackData() *callbackData { return nil }

type TypesData struct {
	types map[string]grammarType
}

func (d *TypesData) Get() []fmt.Stringer {
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

func (d *TypesData) Types() map[string]grammarType {
	return d.types
}

func MakeTypes(node wbnf.GrammarNode) *TypesData {
	return &TypesData{types: MakeTypesFromGrammar(wbnf.NewFromAst(node))}
}
