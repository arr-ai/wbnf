package parser

import (
	"fmt"
	"github.com/arr-ai/wbnf/ast"
	"github.com/arr-ai/wbnf/parse"
	"regexp"
	"strconv"
	"strings"

	"github.com/arr-ai/wbnf/errors"
)

func NewExtRefTreeElement(g Grammar, node parse.TreeElement) parse.TreeElement {
	return Node{
		Tag:      "extref",
		Extra:    FromParserNode(g, node),
		Children: nil,
	}
}

func FromParserNode(g Grammar, e parse.TreeElement) ast.Branch {
	if s, ok := e.(parse.Scanner); ok {
		result := ast.Branch{}
		one(result, "", ast.Leaf(s))
		return result
	}
	rule := NodeRule(e.(Node))
	term := g[rule]
	result := ast.Branch{}
	one(result, "@rule", ast.Extra{rule})
	ctrs := newCounters(term)
	fromParserNode(result, g, term, ctrs, e)
	return result
}

func inc(n ast.Branch, name string) int {
	i := 0
	if child, has := n[name]; has {
		i = child.(ast.One).Node.(ast.Extra).Data.(int)
	}
	n[name] = ast.One{Node: ast.Extra{Data: i + 1}}
	return i
}

func add(n ast.Branch, name string, node ast.Node, ctr counter) {
	switch ctr {
	case counter{}:
		panic(errors.Inconceivable)
	case zeroOrOne, oneOne:
		one(n, name, node)
	default:
		many(n, name, node)
	}
}

func one(n ast.Branch, name string, node ast.Node) {
	if _, has := n[name]; has {
		panic(errors.Inconceivable)
	}
	n[name] = ast.One{Node: node}
}

func many(n ast.Branch, name string, node ast.Node) {
	if many, has := n[name]; has {
		n[name] = append(many.(ast.Many), node)
	} else {
		n[name] = ast.Many([]ast.Node{node})
	}
}

func fromParserNode(n ast.Branch, g Grammar, term Term, ctrs counters, e parse.TreeElement) {
	var tag string
	defer enterf("fromParserNode(term=%T(%[1]v), ctrs=%v, v=%v)", term, ctrs, e).exitf("tag=%q, n=%v", &tag, &n)
	switch t := term.(type) {
	case S, RE:
		add(n, "", ast.Leaf(e.(parse.Scanner)), ctrs[""])
	case Rule:
		term := g[t]
		childCtrs := newCounters(term)
		b := ast.Branch{}
		unleveled, _ := unlevel(string(t), g)
		fromParserNode(b, g, term, childCtrs, e)
		var node ast.Node = b
		// if name := childCtrs.singular(); name != nil {
		// 	node = b[*name].(One).Node
		// 	// TODO: zeroOrOne
		// }
		add(n, unleveled, node, ctrs[string(t)])
	case ScopedGrammar:
		gcopy := g
		for rule, terms := range t.Grammar {
			gcopy[rule] = terms
		}
		fromParserNode(n, gcopy, t.Term, ctrs, e)
	case Seq:
		node := e.(Node)
		tag = node.Tag
		for i, child := range node.Children {
			fromParserNode(n, g, t[i], ctrs, child)
		}
	case Oneof:
		node := e.(Node)
		tag = node.Tag
		many(n, ast.ChoiceTag, ast.Extra{Data: node.Extra.(Choice)})
		fromParserNode(n, g, t[node.Extra.(Choice)], ctrs, node.Children[0])
	case Delim:
		node := e.(Node)
		tag = node.Tag
		tgen := t.LRTerms(node)
		for i, child := range node.Children {
			term := tgen.Next()
			if _, ok := child.(Empty); ok {
				many(n, "@empty", ast.Extra{map[bool]string{true: "@prefix", false: "@suffix"}[i == 0]})
			} else {
				if term == t {
					if _, ok := child.(Node); ok {
						childCtrs := newCounters(term)
						b := ast.Branch{}
						childCtrs.termCountChildren(t, ctrs[""])
						fromParserNode(b, g, term, childCtrs, child)
						one(n, tag, b)
					} else {
						fromParserNode(n, g, t.Term, ctrs, child)
					}
				} else {
					fromParserNode(n, g, term, ctrs, child)
				}
			}
		}
	case Quant:
		node := e.(Node)
		tag = node.Tag
		for _, child := range node.Children {
			fromParserNode(n, g, t.Term, ctrs, child)
		}
	case Named:
		childCtrs := newCounters(t.Term)
		b := ast.Branch{}
		fromParserNode(b, g, t.Term, childCtrs, e)
		var node ast.Node = b
		// if name := childCtrs.singular(); name != nil {
		// 	node = b[*name].(One).Node
		// 	// TODO: zeroOrOne
		// }
		add(n, t.Name, node, ctrs[t.Name])
	case REF:
		switch e := e.(type) {
		case parse.Scanner:
			add(n, t.Ident, ast.Leaf(e), ctrs[t.Ident])
		case Node:
			b := ast.Branch{}
			for _, child := range e.Children {
				fromParserNode(b, g, term, ctrs, child)
			}
			add(n, t.Ident, b, ctrs[t.Ident])
		}
	case CutPoint:
		fromParserNode(n, g, t.Term, ctrs, e)
	case ExtRef:
		if node, ok := e.(Node); ok {
			if b, ok := node.Extra.(ast.Branch); ok {
				ident := t.String()
				add(n, ident, b, ctrs[ident])
			}
		}
	default:
		panic(fmt.Errorf("unexpected term type: %v %[1]T", t))
	}
}

var stackLevelRE = regexp.MustCompile(`^(\w+)@(\d+)$`)

func unlevel(name string, g Grammar) (string, int) {
	if m := stackLevelRE.FindStringSubmatch(name); m != nil {
		i, err := strconv.Atoi(m[2])
		if err != nil {
			panic(errors.Inconceivable)
		}
		return m[1], i
	}
	if !strings.Contains(name, StackDelim) {
		if _, has := g[Rule(name+"@1")]; has {
			return name, 0
		}
	}
	return name, -1
}