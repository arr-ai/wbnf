package wbnf

import (
	"fmt"
	"strings"

	"github.com/arr-ai/wbnf/errors"
	"github.com/arr-ai/wbnf/parser"
)

func FromParserNode(g parser.Grammar, e parser.TreeElement) Branch {
	rule := parser.NodeRule(e.(parser.Node))
	term := g[rule]
	result := Branch{}
	result.one("@rule", Extra{rule})
	ctrs := newCounters(term)
	result.fromParserNode(g, term, ctrs, e)
	return result.Collapse(0).(Branch)
}

func (n Branch) Collapse(level int) Node {
	if false && level > 0 {
		switch oneChild := n.oneChild().(type) {
		case Branch:
			oneBranch := oneChild
			oneBranch.inc(SkipTag)
			if choice, has := n[ChoiceTag]; has {
				if oChoice, has := oneBranch[ChoiceTag]; has {
					oneBranch[ChoiceTag] = append(choice.(Many), oChoice.(Many)...)
				} else {
					oneBranch[ChoiceTag] = choice
				}
			}
			if rule, has := n[RuleTag]; has {
				oneBranch[RuleTag] = rule
			}
			return oneBranch
			// case Leaf:
			// 	return oneChild
		}
	}
	return n
}

func (n Branch) oneChild() Node {
	var oneChildren Children
	for childrenName, children := range n {
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

func (n Branch) inc(name string) int {
	i := 0
	if child, has := n[name]; has {
		i = child.(One).Node.(Extra).Data.(int)
	}
	n[name] = One{Node: Extra{Data: i + 1}}
	return i
}

func (n Branch) add(name string, node Node, ctr counter) {
	switch ctr {
	case counter{}:
		panic(errors.Inconceivable)
	case zeroOrOne, oneOne:
		n.one(name, node)
	default:
		n.many(name, node)
	}
}

func (n Branch) one(name string, node Node) {
	if _, has := n[name]; has {
		panic(errors.Inconceivable)
	}
	n[name] = One{Node: node}
}

func (n Branch) many(name string, node Node) {
	if many, has := n[name]; has {
		n[name] = append(many.(Many), node)
	} else {
		n[name] = Many([]Node{node})
	}
}

func (n Branch) fromParserNode(g parser.Grammar, term parser.Term, ctrs counters, e parser.TreeElement) {
	var tag string
	defer enterf("fromParserNode(term=%T(%[1]v), ctrs=%v, v=%v)", term, ctrs, e).exitf("tag=%q, n=%v", &tag, &n)
	switch t := term.(type) {
	case parser.S, parser.RE:
		n.add("", Leaf(e.(parser.Scanner)), ctrs[""])
	case parser.Rule:
		term := g[t]
		childCtrs := newCounters(term)
		b := Branch{}
		unleveled, level := unlevel(string(t), g)
		b.fromParserNode(g, term, childCtrs, e)
		var node Node = b
		// if name := childCtrs.singular(); name != nil {
		// 	node = b[*name].(One).Node
		// 	// TODO: zeroOrOne
		// }
		node = node.Collapse(level)
		n.add(unleveled, node, ctrs[string(t)])
	case parser.Seq:
		node := e.(parser.Node)
		tag = node.Tag
		for i, child := range node.Children {
			n.fromParserNode(g, t[i], ctrs, child)
		}
	case parser.Oneof:
		node := e.(parser.Node)
		tag = node.Tag
		n.many(ChoiceTag, Extra{Data: node.Extra.(parser.Choice)})
		n.fromParserNode(g, t[node.Extra.(parser.Choice)], ctrs, node.Children[0])
	case parser.Delim:
		node := e.(parser.Node)
		tag = node.Tag
		tgen := t.LRTerms(node)
		for _, child := range node.Children {
			term := tgen.Next()
			if _, ok := child.(parser.Empty); ok {
				n.one("@empty", Extra{})
			} else {
				if term == t {
					if _, ok := child.(parser.Node); ok {
						childCtrs := newCounters(term)
						b := Branch{}
						childCtrs.termCountChildren(t, ctrs[""])
						b.fromParserNode(g, term, childCtrs, child)
						n.one(tag, b)
					} else {
						n.fromParserNode(g, t.Term, ctrs, child)
					}
				} else {
					n.fromParserNode(g, term, ctrs, child)
				}
			}
		}
	case parser.Quant:
		node := e.(parser.Node)
		tag = node.Tag
		for _, child := range node.Children {
			n.fromParserNode(g, t.Term, ctrs, child)
		}
	case parser.Named:
		childCtrs := newCounters(t.Term)
		b := Branch{}
		b.fromParserNode(g, t.Term, childCtrs, e)
		var node Node = b
		// if name := childCtrs.singular(); name != nil {
		// 	node = b[*name].(One).Node
		// 	// TODO: zeroOrOne
		// }
		n.add(t.Name, node, ctrs[t.Name])
	case parser.REF:
		if t.External {
			node := e.(*parser.Node)
			subgrammarNode := node.Extra.(parser.SubGrammar).Node
			subgrammar := FromParserNode(Core().Grammar(), subgrammarNode)
			g := NewFromNode(subgrammarNode)
			g.ResolveStacks()
			subnode := FromParserNode(g, node.Children[0])
			n.add("", Branch{"@grammar": One{Node: subgrammar}, "@node": One{subnode}}, oneOne)
		} else {
			switch e := e.(type) {
			case parser.Scanner:
				n.add(t.Ident, Leaf(e), ctrs[t.Ident])
			case parser.Node:
				b := Branch{}
				for _, child := range e.Children {
					b.fromParserNode(g, term, ctrs, child)
				}
				n.add(t.Ident, b, ctrs[t.Ident])
			}
		}
	default:
		panic(fmt.Errorf("unexpected term type: %v %[1]T", t))
	}
}
