package ast

import (
	"fmt"
	"strings"

	"github.com/arr-ai/wbnf/errors"
	"github.com/arr-ai/wbnf/parser"
)

func NewExtRefTreeElement(g parser.Grammar, node parser.TreeElement) parser.TreeElement {
	return parser.Node{
		Tag:      "extref",
		Extra:    FromParserNode(g, node),
		Children: nil,
	}
}

func (Branch) IsExtra() {}

func FromParserNode(g parser.Grammar, e parser.TreeElement) Branch {
	if s, ok := e.(parser.Scanner); ok {
		result := Branch{}
		result.one("", Leaf(s))
		return result
	}
	rule := parser.NodeRule(e.(parser.Node))
	term := g[rule]
	result := Branch{}
	result.one("@rule", Extra{rule})
	ctrs := newCounters(term)
	result.fromParserNode(g, term, ctrs, e)
	return result.collapse(0).(Branch)
}

func (b Branch) collapse(level int) Node {
	if false && level > 0 {
		switch oneChild := b.oneChild().(type) {
		case Branch:
			oneBranch := oneChild
			oneBranch.inc(SkipTag)
			if choice, has := b[ChoiceTag]; has {
				if oChoice, has := oneBranch[ChoiceTag]; has {
					oneBranch[ChoiceTag] = append(choice.(Many), oChoice.(Many)...)
				} else {
					oneBranch[ChoiceTag] = choice
				}
			}
			if rule, has := b[RuleTag]; has {
				oneBranch[RuleTag] = rule
			}
			return oneBranch
			// case Leaf:
			// 	return oneChild
		}
	}
	return b
}

func (b Branch) oneChild() Node {
	var oneChildren Children
	for childrenName, children := range b {
		if !strings.HasPrefix(childrenName, "@") {
			if oneChildren != nil {
				return nil
			}
			oneChildren = children
		}
	}
	if oneChildren != nil {
		switch c := oneChildren.(type) {
		case One:
			return c.Node
		case Many:
			if len(c) == 1 {
				return c[0]
			}
		}
	}
	return nil
}

func (b Branch) inc(name string) int {
	i := 0
	if child, has := b[name]; has {
		i = child.(One).Node.(Extra).Data.(int)
	}
	b[name] = One{Node: Extra{Data: i + 1}}
	return i
}

func (b Branch) add(name string, node Node, ctr counter) {
	switch ctr {
	case counter{}:
		panic(errors.Inconceivable)
	case zeroOrOne, oneOne:
		b.one(name, node)
	default:
		b.many(name, node)
	}
}

func (b Branch) one(name string, node Node) {
	if _, has := b[name]; has {
		panic(errors.Inconceivable)
	}
	b[name] = One{Node: node}
}

func (b Branch) many(name string, node Node) {
	if many, has := b[name]; has {
		b[name] = append(many.(Many), node)
	} else {
		b[name] = Many([]Node{node})
	}
}

func (b Branch) fromParserNode(g parser.Grammar, term parser.Term, ctrs counters, e parser.TreeElement) {
	var tag string
	// defer enterf("fromParserNode(term=%T(%[1]v), ctrs=%v, v=%v)", term, ctrs, e).exitf("tag=%q, n=%v", &tag, &n)
	switch t := term.(type) {
	case parser.S, parser.RE:
		b.add("", Leaf(e.(parser.Scanner)), ctrs[""])
	case parser.Rule:
		term := g[t]
		childCtrs := newCounters(term)
		b2 := Branch{}
		unleveled, level := unlevel(string(t), g)
		b2.fromParserNode(g, term, childCtrs, e)
		var node Node = b2
		// if name := childCtrs.singular(); name != nil {
		// 	node = b2[*name].(One).Node
		// 	// TODO: zeroOrOne
		// }
		node = node.collapse(level)
		b.add(unleveled, node, ctrs[string(t)])
	case parser.ScopedGrammar:
		gcopy := g
		for rule, terms := range t.Grammar {
			gcopy[rule] = terms
		}
		b.fromParserNode(gcopy, t.Term, ctrs, e)
	case parser.Seq:
		node := e.(parser.Node)
		for i, child := range node.Children {
			b.fromParserNode(g, t[i], ctrs, child)
		}
	case parser.Oneof:
		node := e.(parser.Node)
		b.many(ChoiceTag, Extra{Data: node.Extra.(parser.Choice)})
		b.fromParserNode(g, t[node.Extra.(parser.Choice)], ctrs, node.Children[0])
	case parser.Delim:
		node := e.(parser.Node)
		tag = node.Tag
		tgen := t.LRTerms(node)
		for i, child := range node.Children {
			term := tgen.Next()
			if _, ok := child.(parser.Empty); ok {
				b.many("@empty", Extra{map[bool]string{true: "@prefix", false: "@suffix"}[i == 0]})
			} else {
				if term == t {
					if _, ok := child.(parser.Node); ok {
						childCtrs := newCounters(term)
						b2 := Branch{}
						childCtrs.termCountChildren(t, ctrs[""])
						b2.fromParserNode(g, term, childCtrs, child)
						b.one(tag, b2)
					} else {
						b.fromParserNode(g, t.Term, ctrs, child)
					}
				} else {
					b.fromParserNode(g, term, ctrs, child)
				}
			}
		}
	case parser.Quant:
		node := e.(parser.Node)
		for _, child := range node.Children {
			b.fromParserNode(g, t.Term, ctrs, child)
		}
	case parser.Named:
		childCtrs := newCounters(t.Term)
		b2 := Branch{}
		b2.fromParserNode(g, t.Term, childCtrs, e)
		var node Node = b2
		// if name := childCtrs.singular(); name != nil {
		// 	node = b2[*name].(One).Node
		// 	// TODO: zeroOrOne
		// }
		b.add(t.Name, node, ctrs[t.Name])
	case parser.REF:
		switch e := e.(type) {
		case parser.Scanner:
			b.add(t.Ident, Leaf(e), ctrs[t.Ident])
		case parser.Node:
			b2 := Branch{}
			for _, child := range e.Children {
				b2.fromParserNode(g, term, ctrs, child)
			}
			b.add(t.Ident, b2, ctrs[t.Ident])
		}
	case parser.CutPoint:
		b.fromParserNode(g, t.Term, ctrs, e)
	case parser.ExtRef:
		if node, ok := e.(parser.Node); ok {
			if b2, ok := node.Extra.(Branch); ok {
				ident := t.String()
				b.add(ident, b2, ctrs[ident])
			}
		}
	default:
		panic(fmt.Errorf("branch.fromParserNode: unexpected term type: %v %[1]T", t))
	}
}
