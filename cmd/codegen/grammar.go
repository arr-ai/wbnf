package codegen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/arr-ai/frozen"

	"github.com/arr-ai/wbnf/parser"
)

const (
	noScope int = iota
	bracesScope
	squigglyScope
	mapScope
)

type GoNode struct {
	name     string
	children []GoNode
	scope    int
}

func (g *GoNode) String() string {
	x := map[int]struct {
		open  string
		close string
	}{
		noScope:       {"", ""},
		mapScope:      {":", ""},
		bracesScope:   {"(", ")"},
		squigglyScope: {"{", "}"},
	}[g.scope]
	children := make([]string, 0, len(g.children))
	for _, c := range g.children {
		children = append(children, c.String())
	}
	return strings.Join([]string{g.name, x.open, strings.Join(children, ",\n"), x.close}, "")
}

func (g *GoNode) Add(n GoNode) {
	g.children = append(g.children, n)
}

func safeString(src string) string {
	r := strings.NewReplacer("`", "`+\"`\"+`")
	return r.Replace(src)
}

func prefixName(prefix string, node GoNode) GoNode {
	return GoNode{
		name:     prefix + node.name,
		children: node.children,
		scope:    node.scope,
	}
}
func stringNode(fmtString string, args ...interface{}) GoNode {
	return GoNode{name: fmt.Sprintf(fmtString, args...)}
}

func walkTerm(term parser.Term) GoNode {
	node := GoNode{}
	switch t := term.(type) {
	case parser.Seq:
		node.name = "parser.Seq"
		node.scope = squigglyScope
		for _, t := range t {
			node.children = append(node.children, walkTerm(t))
		}
	case parser.Stack:
		node.name = "parser.Stack"
		node.scope = squigglyScope
		for _, t := range t {
			node.children = append(node.children, walkTerm(t))
		}
	case parser.Oneof:
		node.name = "parser.Oneof"
		node.scope = squigglyScope
		for _, t := range t {
			node.children = append(node.children, walkTerm(t))
		}
	case parser.S:
		node.name = fmt.Sprintf("parser.S(`%s`)", safeString(string(t)))
	case parser.Delim:
		node.name = "parser.Delim"
		node.scope = squigglyScope
		node.children = []GoNode{
			prefixName("Term: ", walkTerm(t.Term)),
			prefixName("Sep: ", walkTerm(t.Sep)),
		}
		if t.CanStartWithSep {
			node.Add(stringNode("CanStartWithSep: true"))
		}
		if t.CanEndWithSep {
			node.Add(stringNode("CanEndWithSep: true"))
		}
		switch t.Assoc {
		case parser.LeftToRight:
			node.Add(stringNode("Assoc: parser.LeftToRight"))
		case parser.RightToLeft:
			node.Add(stringNode("Assoc: parser.RightToLeft"))
		}
	case parser.Quant:
		node.Add(walkTerm(t.Term))
		if t.Min == 0 && t.Max == 0 {
			node.name = "parser.Any"
			node.scope = bracesScope
		} else if t.Min == 1 && t.Max == 0 {
			node.name = "parser.Some"
			node.scope = bracesScope
		} else if t.Min == 0 && t.Max == 1 {
			node.name = "parser.Opt"
			node.scope = bracesScope
		} else {
			node.name = "parser.Quant"
			node.scope = squigglyScope
			node.Add(stringNode("Min: %d", t.Min))
			node.Add(stringNode("Max: %d", t.Max))
		}
	case parser.Named:
		node.name = "parser.Eq"
		node.scope = bracesScope
		node.children = []GoNode{
			stringNode("`%s`", t.Name),
			walkTerm(t.Term),
		}
	case parser.ScopedGrammar:
		node.name = "parser.ScopedGrammar"
		node.children = []GoNode{
			prefixName("Term: ", walkTerm(t.Term)),
			prefixName("Grammar: ", *MakeGrammarString(t.Grammar)),
		}
		node.scope = squigglyScope
	case parser.REF:
		node.name = "parser.REF"
		node.scope = squigglyScope
		node.Add(stringNode("Ident: `%s`", t.Ident))
		if t.Default != nil {
			node.Add(prefixName("Default: ", walkTerm(t.Default)))
		}
	case parser.RE:
		node = stringNode("parser.RE(`%s`)", safeString(string(t)))
	case parser.Rule:
		if strings.Contains(string(t), parser.StackDelim) {
			node = stringNode("parser.At")
		} else {
			node = stringNode("parser.Rule(`%s`)", string(t))
		}
	case parser.CutPoint:
		node.name = "parser.CutPoint"
		node.scope = squigglyScope
		node.Add(walkTerm(t.Term))
	case parser.ExtRef:
		node = stringNode("parser.ExtRef(`%s`)", safeString(string(t)))
	default:
		panic(fmt.Errorf("walkTerm: unexpected term type: %v %[1]T", t))
	}

	return node
}

func MakeGrammarString(g parser.Grammar) *GoNode {
	root := GoNode{name: "parser.Grammar", scope: squigglyScope}
	keys := make([]string, 0, len(g))
	for rule := range g {
		keys = append(keys, string(rule))
	}
	sort.Strings(keys)
	rules := map[string]GoNode{}
	stackPrefixes := frozen.NewSet()
	for rule, t := range g {
		r := string(rule)
		rules[r] = walkTerm(t)
		if strings.Contains(r, parser.StackDelim) {
			stackPrefixes = stackPrefixes.With(strings.Split(r, parser.StackDelim)[0])
		}
	}

	for _, rule := range keys {
		if stackPrefixes.Has(rule) {
			stack := GoNode{
				name:     "parser.Stack",
				children: []GoNode{rules[rule]},
				scope:    squigglyScope,
			}
			for i := 1; ; i++ {
				stackname := fmt.Sprintf("%s@%d", rule, i)
				if node, ok := rules[stackname]; ok {
					stack.Add(node)
					delete(rules, stackname)
				} else {
					break
				}
			}
			root.Add(GoNode{
				name:     fmt.Sprintf(`"%s"`, rule),
				children: []GoNode{stack},
				scope:    mapScope,
			})
		} else if node, ok := rules[rule]; ok {
			root.Add(GoNode{
				name:     fmt.Sprintf(`"%s"`, rule),
				children: []GoNode{node},
				scope:    mapScope,
			})
		}
	}
	return &root
}
