package ast

import (
	"fmt"
	"strings"

	"github.com/arr-ai/wbnf/errors"
	"github.com/arr-ai/wbnf/parser"
	"github.com/arr-ai/wbnf/wbnf"
)

var coreNode = func() Node {
	return FromParserNode(wbnf.Core().Grammar(), *wbnf.Core().Node())
}()

func CoreNode() Node {
	return coreNode
}

func FromParserNode(g wbnf.Grammar, e parser.TreeElement) Branch {
	rule := wbnf.NodeRule(e.(parser.Node))
	term := g[rule]
	result := Branch{}
	result.one("@rule", Extra{rule})
	ctrs := newCounters(term)
	result.fromParserNode(g, term, ctrs, e)
	return result.collapse(0).(Branch)
}

func (n Branch) collapse(level int) Node {
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

func (n Branch) fromParserNode(g wbnf.Grammar, term wbnf.Term, ctrs counters, e parser.TreeElement) {
	var tag string
	defer enterf("fromParserNode(term=%T(%[1]v), ctrs=%v, v=%v)", term, ctrs, e).exitf("tag=%q, n=%v", &tag, &n)
	switch t := term.(type) {
	case wbnf.S, wbnf.RE, wbnf.REF:
		n.add("", Leaf(e.(parser.Scanner)), ctrs[""])
	case wbnf.Rule:
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
		node = node.collapse(level)
		n.add(unleveled, node, ctrs[string(t)])
	case wbnf.Seq:
		node := e.(parser.Node)
		tag = node.Tag
		for i, child := range node.Children {
			n.fromParserNode(g, t[i], ctrs, child)
		}
	case wbnf.Oneof:
		node := e.(parser.Node)
		tag = node.Tag
		n.many(ChoiceTag, Extra{Data: node.Extra.(wbnf.Choice)})
		n.fromParserNode(g, t[node.Extra.(wbnf.Choice)], ctrs, node.Children[0])
	case wbnf.Delim:
		node := e.(parser.Node)
		tag = node.Tag
		if node.Extra.(wbnf.Associativity) != wbnf.NonAssociative {
			panic(errors.Unfinished)
		}
		L, R := t.LRTerms(node)
		terms := [2]wbnf.Term{L, t.Sep}
		for i, child := range node.Children {
			if _, ok := child.(wbnf.Empty); ok {
				// TODO: round-trip trailing commas
			} else {
				n.fromParserNode(g, terms[i%2], ctrs, child)
				terms[0] = R
			}
			terms[0] = R
		}
	case wbnf.Quant:
		node := e.(parser.Node)
		tag = node.Tag
		for _, child := range node.Children {
			n.fromParserNode(g, t.Term, ctrs, child)
		}
	case wbnf.Named:
		childCtrs := newCounters(t.Term)
		b := Branch{}
		b.fromParserNode(g, t.Term, childCtrs, e)
		var node Node = b
		// if name := childCtrs.singular(); name != nil {
		// 	node = b[*name].(One).Node
		// 	// TODO: zeroOrOne
		// }
		n.add(t.Name, node, ctrs[t.Name])
	default:
		panic(fmt.Errorf("unexpected term type: %v %[1]T", t))
	}
}
