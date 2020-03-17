package parser

import (
	"fmt"
	"github.com/arr-ai/wbnf/ast"
	"github.com/arr-ai/wbnf/parse"

	"github.com/arr-ai/wbnf/errors"
)

func ToParserNode(g Grammar, branch ast.Branch) parse.TreeElement {
	branch = branch.Clone().(ast.Branch)
	rule := pullFromOne(branch, ast.RuleTag).(ast.Extra).Data.(Rule)
	term := g[rule]
	ctrs := newCounters(term)
	branch = expandNode(branch, string(rule)).(ast.Branch)
	return relabelNode(string(rule), toParserNode(branch, g, term, ctrs))
}

func relabelNode(name string, e parse.TreeElement) parse.TreeElement {
	if n, ok := e.(Node); ok {
		n.Tag = name
		return n
	}
	return e
}

func expandNode(n ast.Node, name string) ast.Node {
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

func pull(n ast.Branch, name string, ctr counter) ast.Node {
	switch ctr {
	case counter{}:
		panic(errors.Inconceivable)
	case zeroOrOne, oneOne:
		return pullFromOne(n, name)
	default:
		return pullFromMany(n, name)
	}
}

func pullFromOne(n ast.Branch, name string) ast.Node {
	if child, has := n[name]; has {
		delete(n, name)
		return child.(ast.One).Node
	}
	return nil
}

func dec(n ast.Branch, name string) int {
	if child, has := n[name]; has {
		i := child.(ast.One).Node.(ast.Extra).Data.(int)
		if i == 1 {
			delete(n, name)
		}
		return i
	}
	return 0
}

func pullFromMany(n ast.Branch, name string) ast.Node {
	if node, has := n[name]; has {
		many := node.(ast.Many)
		if len(many) > 0 {
			result := many[0]
			if len(many) > 1 {
				n[name] = many[1:]
			} else {
				delete(n, name)
			}
			return result
		}
	}
	return nil
}

func toParserNode(n ast.Branch, g Grammar, term Term, ctrs counters) (out parse.TreeElement) {
	defer enterf("%v.toParserNode(g, term=%T(%[2]v), ctrs=%v)", n, term, ctrs).exitf("%v", &out)
	switch t := term.(type) {
	case S, RE:
		if node := pull(n, "", ctrs[""]); node != nil {
			return parse.Scanner(node.(ast.Leaf))
		}
		return nil
	case Rule:
		name := string(t)
		term := g[t]
		unleveled, level := unlevel(name, g)
		if node := pull(n, unleveled, ctrs[name]); node != nil {
			if level > 0 {
				node = expandNode(node, unleveled)
			}
			childCtrs := newCounters(term)
			// if name := childCtrs.singular(); name != nil {
			// 	node = Branch{*name: One{Node: node}}
			// }
			switch node := node.(type) {
			case ast.Branch:
				return relabelNode(string(t), toParserNode(node, g, term, childCtrs))
			case ast.Leaf:
				return parse.Scanner(node)
			default:
				panic(fmt.Errorf("wrong node type: %v", node))
			}
		}
		return nil
	case ScopedGrammar:
		gcopy := g
		for rule, terms := range t.Grammar {
			gcopy[rule] = terms
		}
		return toParserNode(n, gcopy, t.Term, ctrs)
	case Seq:
		result := Node{Tag: seqTag}
		for _, child := range t {
			if node := toParserNode(n, g, child, ctrs); node != nil {
				result.Children = append(result.Children, node)
			} else {
				return nil
			}
		}
		return result
	case Oneof:
		if choice := pullFromMany(n, ast.ChoiceTag); choice != nil {
			extra := choice.(ast.Extra).Data.(Choice)
			return Node{
				Tag:      oneofTag,
				Extra:    extra,
				Children: []parse.TreeElement{toParserNode(n, g, t[extra], ctrs)},
			}
		}
		return nil
	case Delim:
		v := Node{
			Tag:   delimTag,
			Extra: NonAssociative,
		}
		terms := [2]Term{t.Term, t.Sep}
		i := 0
		for ; ; i++ {
			if child := toParserNode(n, g, terms[i%2], ctrs); child != nil {
				v.Children = append(v.Children, child)
			} else {
				break
			}
		}
		if i%2 == 0 {
			panic(errors.Inconceivable)
		}
		return v
	case Quant:
		result := Node{Tag: quantTag}
		for i := 0; !t.MaxLessThan(i); i++ {
			if v := toParserNode(n, g, t.Term, ctrs); v != nil {
				result.Children = append(result.Children, v)
			} else {
				break
			}
		}
		if !t.Contains(len(result.Children)) {
			panic(errors.Inconceivable)
		}
		return result
	case Named:
		childCtrs := newCounters(t.Term)
		if node := pull(n, t.Name, ctrs[t.Name]); node != nil {
			// if name := childCtrs.singular(); name != nil {
			// 	node = Branch{*name: One{Node: node}}
			// }
			switch node := node.(type) {
			case ast.Branch:
				return relabelNode(t.Name, toParserNode(node, g, t.Term, childCtrs))
			case ast.Leaf:
				return parse.Scanner(node)
			default:
				panic(fmt.Errorf("wrong node type: %v", node))
			}
		}
		return nil
	case CutPoint:
		return toParserNode(n, g, t.Term, ctrs)
	default:
		panic(fmt.Errorf("unexpected term type: %v %[1]T", t))
	}
}
