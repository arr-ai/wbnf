package ast

import (
	"fmt"

	"github.com/arr-ai/wbnf/errors"
	"github.com/arr-ai/wbnf/parser"
)

func ToParserNode(g parser.Grammar, branch Branch) parser.TreeElement {
	branch = branch.clone().(Branch)
	rule := branch.pullFromOne(RuleTag).(Extra).Data.(parser.Rule)
	term := g[rule]
	ctrs := newCounters(term)
	branch = expandNode(branch, string(rule)).(Branch)
	return relabelNode(string(rule), branch.toParserNode(g, term, ctrs))
}

func relabelNode(name string, e parser.TreeElement) parser.TreeElement {
	if n, ok := e.(parser.Node); ok {
		n.Tag = name
		return n
	}
	return e
}

func expandNode(n Node, name string) Node {
	return n
	// switch n := n.(type) {
	// case Branch:
	// 	if i := n.dec(SkipTag); i <= 0 {
	// 		return n
	// 	}
	// 	return Branch{name: Many{n}}
	// case Leaf:
	// 	return n
	// default:
	// 	panic("wat?")
	// }
}

func (b Branch) pull(name string, ctr counter) Node {
	switch ctr {
	case counter{}:
		panic(errors.Inconceivable)
	case zeroOrOne, oneOne:
		return b.pullFromOne(name)
	default:
		return b.pullFromMany(name)
	}
}

func (b Branch) pullFromOne(name string) Node {
	if child, has := b[name]; has {
		delete(b, name)
		return child.(One).Node
	}
	return nil
}

func (b Branch) pullFromMany(name string) Node {
	if node, has := b[name]; has {
		many := node.(Many)
		if len(many) > 0 {
			result := many[0]
			if len(many) > 1 {
				b[name] = many[1:]
			} else {
				delete(b, name)
			}
			return result
		}
	}
	return nil
}

func (b Branch) toParserNode(g parser.Grammar, term parser.Term, ctrs counters) (out parser.TreeElement) {
	// defer enterf("%v.toParserNode(g, term=%T(%[2]v), ctrs=%v)", n, term, ctrs).exitf("%v", &out)
	switch t := term.(type) {
	case parser.S, parser.RE:
		if node := b.pull("", ctrs[""]); node != nil {
			return parser.Scanner(node.(Leaf))
		}
		return nil
	case parser.Rule:
		name := string(t)
		term := g[t]
		unleveled, level := unlevel(name, g)
		if node := b.pull(unleveled, ctrs[name]); node != nil {
			if level > 0 {
				node = expandNode(node, unleveled)
			}
			childCtrs := newCounters(term)
			// if name := childCtrs.singular(); name != nil {
			// 	node = Branch{*name: One{Node: node}}
			// }
			switch node := node.(type) {
			case Branch:
				return relabelNode(string(t), node.toParserNode(g, term, childCtrs))
			case Leaf:
				return parser.Scanner(node)
			default:
				panic(fmt.Errorf("wrong node type: %v", node))
			}
		}
		return nil
	case parser.ScopedGrammar:
		gcopy := g
		for rule, terms := range t.Grammar {
			gcopy[rule] = terms
		}
		return b.toParserNode(gcopy, t.Term, ctrs)
	case parser.Seq:
		result := parser.Node{Tag: seqTag}
		for _, child := range t {
			if node := b.toParserNode(g, child, ctrs); node != nil {
				result.Children = append(result.Children, node)
			} else {
				return nil
			}
		}
		return result
	case parser.Oneof:
		if choice := b.pullFromMany(ChoiceTag); choice != nil {
			extra := choice.(Extra).Data.(parser.Choice)
			return parser.Node{
				Tag:      oneofTag,
				Extra:    extra,
				Children: []parser.TreeElement{b.toParserNode(g, t[extra], ctrs)},
			}
		}
		return nil
	case parser.Delim:
		v := parser.Node{
			Tag:   delimTag,
			Extra: parser.NonAssociative,
		}
		terms := [2]parser.Term{t.Term, t.Sep}
		i := 0
		for ; ; i++ {
			if child := b.toParserNode(g, terms[i%2], ctrs); child != nil {
				v.Children = append(v.Children, child)
			} else {
				break
			}
		}
		if i%2 == 0 {
			panic(errors.Inconceivable)
		}
		return v
	case parser.Quant:
		result := parser.Node{Tag: quantTag}
		for i := 0; !t.MaxLessThan(i); i++ {
			if v := b.toParserNode(g, t.Term, ctrs); v != nil {
				result.Children = append(result.Children, v)
			} else {
				break
			}
		}
		if !t.Contains(len(result.Children)) {
			panic(errors.Inconceivable)
		}
		return result
	case parser.Named:
		childCtrs := newCounters(t.Term)
		if node := b.pull(t.Name, ctrs[t.Name]); node != nil {
			// if name := childCtrs.singular(); name != nil {
			// 	node = Branch{*name: One{Node: node}}
			// }
			switch node := node.(type) {
			case Branch:
				return relabelNode(t.Name, node.toParserNode(g, t.Term, childCtrs))
			case Leaf:
				return parser.Scanner(node)
			default:
				panic(fmt.Errorf("wrong node type: %v", node))
			}
		}
		return nil
	case parser.CutPoint:
		return b.toParserNode(g, t.Term, ctrs)
	default:
		panic(fmt.Errorf("branch.toParserNode: unexpected term type: %v %[1]T", t))
	}
}
