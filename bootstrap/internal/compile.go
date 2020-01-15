package internal

import (
	"strings"

	"github.com/arr-ai/wbnf/parser"
)

func compileAtomNode(node parser.Node) parser.BaseNode {
	atom := Atom{
		choice: node.Extra.(int),
	}
	switch node.Extra.(int) {
	case 0:
		atom.AddAndSet(IDENT{}.New(node.GetString(0), parser.NoTag), &atom.ident)
	case 1:
		atom.AddAndSet(STR{}.New(node.GetString(0), parser.NoTag), &atom.str)
	case 2:
		atom.AddAndSet(RE{}.New(strings.ReplaceAll(node.GetString(0), `\/`, `/`), parser.NoTag), &atom.re)
	case 3:
		node := node.GetNode(0)
		atom.Add(
			parser.Terminal{}.New(node.GetString(0), parser.NoTag),
			compileTermStackNode(node.GetNode(1)),
			parser.Terminal{}.New(node.GetString(2), parser.NoTag),
		)
		atom.term = atom.AllChildren()[1]
		atom.tokenCount = 2
	case 4:
		node := node.GetNode(0)
		atom.Add(
			parser.Terminal{}.New(node.GetString(0), parser.NoTag),
			parser.Terminal{}.New(node.GetString(1), parser.NoTag),
		)
		atom.tokenCount = 2
	default:
		panic("foo")
	}
	return &atom
}

func compileQuantNode(node parser.Node) parser.BaseNode {
	quant := Quant{
		choice: node.Extra.(int),
	}
	switch node.Extra.(int) {
	case 0:
		op := parser.Terminal{}.New(node.GetString(0), parser.Tag("op"))
		quant.Add(op)
		quant.opCount++
	case 1:
		if node.Count() != 5 {
			panic("ooops")
		}
		quant.Add(
			parser.Terminal{}.New(node.GetString(0), parser.NoTag),
			INT{}.New(node.GetString(1), parser.Tag("min")),
			parser.Terminal{}.New(node.GetString(2), parser.NoTag),
			INT{}.New(node.GetString(3), parser.Tag("max")),
			parser.Terminal{}.New(node.GetString(4), parser.NoTag),
		)
		quant.min = quant.AllChildren()[1]
		quant.max = quant.AllChildren()[3]
	case 2:
		node = node.GetNode(0)
		if node.Count() != 4 {
			panic("ooops")
		}
		quant.AddAndCount(parser.Terminal{}.New(node.GetString(0), parser.Tag("op")), &quant.opCount)
		if node.GetNode(1).Count() != 0 {
			quant.AddAndSet(parser.Terminal{}.New(node.GetString(1), parser.Tag("lbang")),
				&quant.lbang)
		}
		quant.AddAndSet(compileNamedNode(node.GetNode(2)), &quant.named)
		if node.GetNode(3).Count() != 0 {
			quant.AddAndSet(parser.Terminal{}.New(node.GetString(3), parser.Tag("rbang")),
				&quant.rbang)
		}
	}
	return &quant
}

func compileNamedNode(node parser.Node) parser.BaseNode {
	named := Named{}
	if x := node.GetNode(0); x.Count() != 0 {
		x := x.GetNode(0)
		named.Add(
			IDENT{}.New(x.GetString(0), parser.NoTag),
			parser.Terminal{}.New(x.GetString(1), parser.Tag("op")),
		)
		named.ident = named.AllChildren()[0]
		named.op = named.AllChildren()[1]
	}

	atom := compileAtomNode(node.GetNode(1))
	named.Add(atom)
	named.atom = atom

	return &named
}

func compileTermStackNode(node parser.Node) parser.BaseNode {
	var child interface{}
	if node.Count() > 1 {
		child = node
	} else {
		child = node.Get(0)
		for {
			if x, ok := child.(parser.Node); ok && x.Count() == 1 {
				child = x.Get(0)
			} else {
				break
			}
		}
	}
	choicefn := func(name string) int {
		if !strings.HasPrefix(name, "term") {
			panic("oops")
		}
		prefix := strings.Split(name, "\\")[0]
		vals := map[string]int{"m": 0, "1": 1, "2": 2, "3": 3}
		return vals[string(prefix[len(prefix)-1])]
	}
	node = child.(parser.Node)
	term := Term{
		choice: choicefn(node.Tag),
	}
	for _, x := range node.Children {
		switch x := x.(type) {
		case parser.Node:
			switch {
			case strings.HasPrefix(x.Tag, "term"):
				term.Add(compileTermStackNode(x))
				term.termCount++
			case strings.HasPrefix(x.Tag, "named"):
				term.named = compileNamedNode(x)
				term.Add(term.named)
			case strings.HasPrefix(x.Tag, "?") && x.Count() > 0:
				term.Add(compileQuantNode(x.GetNode(0)))
				term.quantCount++
			}
		case parser.Scanner:
			switch term.Choice() {
			case 0, 1:
				op := parser.Terminal{}.New(x.String(), parser.Tag("op"))
				term.Add(op)
				term.opCount++
			default:
			}
		}
	}

	return &term
}

func compileProdNode(node parser.Node) parser.BaseNode {
	prod := Prod{}
	prod.Add(
		IDENT{}.New(node.GetString(0), parser.NoTag),
		parser.Terminal{}.New(node.GetString(1), parser.NoTag),
	)
	prod.ident = prod.AllChildren()[0]

	terms := node.GetNode(2)
	switch terms.Tag {
	case "?":

	}
	children := terms.Children
	prod.termCount = len(children)
	for _, child := range children {
		prod.Add(compileTermStackNode(child.(parser.Node)))
	}

	prod.Add(parser.Terminal{}.New(node.GetString(3), parser.NoTag))
	prod.tokenCount++
	return &prod
}

func FromNodes(node parser.Node) *Grammar {
	g := Grammar{
		stmtCount: len(node.Children),
	}
	for _, v := range node.Children {
		stmt := v.(parser.Node)
		var child Stmt
		switch stmt.Extra.(int) {
		case 0:
			child.choice = 0
			child.Add(
				COMMENT{}.New(v.(parser.Node).GetString(0), parser.NoTag),
			)
			child.comment = child.AllChildren()[0]
		case 1:
			prod := stmt.GetNode(0)
			child.choice = 1
			child.Add(compileProdNode(prod))
			child.prod = child.AllChildren()[0]
		default:
			panic("")
		}
		g.Add(&child)
	}
	return &g
}
