package codegen

import (
	"fmt"
	"strings"

	"github.com/arr-ai/wbnf/ast"
	"github.com/arr-ai/wbnf/parser"
	"github.com/arr-ai/wbnf/wbnf"
)

const (
	noScope int = iota
	bracesScope
	squigglyScope
	mapScope
)

type goNode struct {
	name     string
	children []goNode
	scope    int
}

func (g *goNode) String() string {
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

func (g *goNode) Add(n goNode) {
	g.children = append(g.children, n)
}

func safeString(src string) string {
	r := strings.NewReplacer("`", "`+\"`\"+`", " ", "", "\n", "")
	return r.Replace(src)
}

func makeAtom(node ast.Node) *goNode {
	atom := node.(ast.Branch)
	x, _ := ast.Which(atom, wbnf.IdentRE, wbnf.IdentSTR, wbnf.IdentIDENT, wbnf.IdentREF, wbnf.IdentTerm)
	name := ""
	switch x {
	case wbnf.IdentTerm, "":
	case wbnf.IdentREF:
		name = safeString(atom.One(x).One(wbnf.IdentIDENT).Scanner().String())
	default:
		name = safeString(atom.One(x).Scanner().String())
	}
	switch x {
	case wbnf.IdentIDENT:
		return &goNode{name: fmt.Sprintf("parser.Rule(`%s`)", name)}
	case wbnf.IdentSTR:
		return &goNode{name: fmt.Sprintf("parser.S(%s)", name)}
	case wbnf.IdentRE:
		if strings.HasPrefix(name, "/{") {
			name = name[2 : len(name)-1]
		}
		return &goNode{name: fmt.Sprintf("parser.RE(`%s`)", name)}
	case wbnf.IdentREF:
		return &goNode{name: fmt.Sprintf("parser.REF(`%s`)", name)}
	case wbnf.IdentTerm:
		return makeTerm(atom.One(x))
	}
	return &goNode{name: "todo"}
}
func makeNamed(node ast.Node) *goNode {
	named := node.(ast.Branch)
	atom := makeAtom(named.One("atom"))

	if named.One(wbnf.IdentIDENT) != nil {
		val := &goNode{name: "parser.Eq",
			scope:    bracesScope,
			children: []goNode{{name: "\"" + named.One(wbnf.IdentIDENT).Scanner().String() + "\""}, *atom},
		}
		return val
	}
	return atom
}
func makeQuant(node ast.Node, term goNode) *goNode {
	switch node.Many(ast.ChoiceTag)[0].(ast.Extra).Data.(parser.Choice) {
	case 0:
		switch node.One("op").Scanner().String() {
		case "*":
			return &goNode{name: "parser.Any", scope: bracesScope, children: []goNode{term}}
		case "?":
			return &goNode{name: "parser.Opt", scope: bracesScope, children: []goNode{term}}
		case "+":
			return &goNode{name: "parser.Some", scope: bracesScope, children: []goNode{term}}
		}
	case 1:
		min := "0"
		max := "0"
		if x := node.One("min"); x != nil {
			min = x.Scanner().String()
		}
		if x := node.One("max"); x != nil {
			max = x.Scanner().String()
		}
		term.name = "Term: " + term.name
		return &goNode{name: "parser.Quant", scope: squigglyScope, children: []goNode{term, {name: "Min:" + min}, {name: "Max:" + max}}}
	case 2:
		delim := &goNode{name: "parser.Delim", scope: squigglyScope}
		var assoc string
		switch node.One("op").Scanner().String() {
		case "<:":
			assoc = "Assoc: parser.RightToLeft"
		case ":>":
			assoc = "Assoc: parser.LeftToRight"
		default:
			assoc = "Assoc: parser.NonAssociative"
		}
		term.name = "Term: " + term.name
		sep := *makeNamed(node.One("named"))
		sep.name = "Sep: " + sep.name
		delim.children = []goNode{term, sep, {name: assoc}}
		if node.One("opt_leading") != nil {
			delim.children = append(delim.children, goNode{name: "CanStartWithSep: true"})
		}
		if node.One("opt_trailing") != nil {
			delim.children = append(delim.children, goNode{name: "CanEndWithSep: true"})
		}
		return delim
	}
	return &goNode{name: "todo"}
}

func makeTerm(node ast.Node) *goNode {
	term := node.(ast.Branch)
	x, _ := ast.Which(term, "term", "atom", "named")
	switch x {
	case "term":
		var next *goNode
		if ops := term.Many("op"); len(ops) > 0 {
			switch ops[0].Scanner().String() {
			case "|":
				next = &goNode{name: "parser.Oneof", scope: squigglyScope}
			case ">":
				next = &goNode{name: "parser.Stack", scope: squigglyScope}
			}
		} else {
			next = &goNode{name: "parser.Seq", scope: squigglyScope}
		}
		for _, t := range term.Many("term") {
			next.Add(*makeTerm(t))
		}
		if len(next.children) == 1 {
			next = &next.children[0]
		}
		if sg := ast.First(term, "grammar"); sg != nil {
			next = &goNode{
				name: "parser.ScopedGrammar",
				children: []goNode{
					{name: "Term: ", children: []goNode{*next}},
					{name: "Grammar: ", children: []goNode{*MakeGrammar(sg)}},
				},
				scope: squigglyScope,
			}
		}
		return next
	case "named":
		// named and quants need to be added backwards
		// "a":","*     ->   Any(Delim(... S("a")))
		next := makeNamed(term.One("named"))
		quants := term.Many("quant")
		for i := range quants {
			next = makeQuant(quants[len(quants)-1-i], *next)
		}
		return next
	}
	return &goNode{name: "todo"}
}

func makeProd(tree ast.Node) *goNode {
	terms := tree.Many("term")

	p := &goNode{
		name: fmt.Sprintf(`"%s"`,
			tree.One("IDENT").Scanner().String()),
		children: nil,
		scope:    mapScope,
	}
	for _, t := range terms {
		p.Add(*makeTerm(t))
	}
	return p
}

func MakeGrammar(tree ast.Node) *goNode {
	root := goNode{name: "parser.Grammar", scope: squigglyScope}

	for _, stmt := range tree.Many("stmt") {
		if p := stmt.One("prod"); p != nil {
			root.Add(*makeProd(p))
		}
	}
	return &root
}
