package internal

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/arr-ai/wbnf/parser"
)

func parseString(s string) string {
	var sb strings.Builder
	quote, s := s[0], s[1:len(s)-1]
	if quote == '`' {
		return strings.ReplaceAll(s, "``", "`")
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '\\':
			i++
			switch s[i] {
			case 'x':
				n, err := strconv.ParseInt(s[i:i+2], 16, 8)
				if err != nil {
					panic(err)
				}
				sb.WriteByte(uint8(n))
				i++
			case 'u':
				n, err := strconv.ParseInt(s[i:i+4], 16, 16)
				if err != nil {
					panic(err)
				}
				sb.WriteByte(uint8(n))
				i += 2
			case 'U':
				n, err := strconv.ParseInt(s[i:i+8], 16, 32)
				if err != nil {
					panic(err)
				}
				sb.WriteByte(uint8(n))
				i += 4
			case '0', '1', '2', '3', '4', '5', '6', '7':
				n, err := strconv.ParseInt(s[i:i+3], 8, 8)
				if err != nil {
					panic(err)
				}
				sb.WriteByte(uint8(n))
				i++
			case 'a':
				sb.WriteByte('\a')
			case 'b':
				sb.WriteByte('\b')
			case 'f':
				sb.WriteByte('\f')
			case 'n':
				sb.WriteByte('\n')
			case 'r':
				sb.WriteByte('\r')
			case 't':
				sb.WriteByte('\t')
			case 'v':
				sb.WriteByte('\v')
			case '\\':
				sb.WriteByte('\\')
			case '\'':
				sb.WriteByte('\'')
			case quote:
				sb.WriteByte(quote)
			default:
				panic(fmt.Errorf("unrecognized \\-escape: %q", s[i]))
			}
		default:
			sb.WriteByte(c)
		}
	}
	return sb.String()
}

func compileAtomNode(node parser.Node) parser.BaseNode {
	atom := Atom{
		choice:    node.Extra.(int),
		numTokens: 0,
	}
	switch node.Extra.(int) {
	case 0:
		atom.Add(IDENT{}.New(node.GetString(0), parser.NoTag))
	case 1:
		atom.Add(STR{}.New(node.GetString(0), parser.NoTag))
	case 2:
		atom.Add(RE{}.New(strings.ReplaceAll(node.GetString(0), `\/`, `/`), parser.NoTag))
	case 3:
		node := node.GetNode(0)
		atom.Add(
			parser.Terminal{}.New(node.GetString(0), parser.NoTag),
			compileTermStackNode(node.GetNode(1)),
			parser.Terminal{}.New(node.GetString(2), parser.NoTag),
		)
	case 4:
		node := node.GetNode(0)
		atom.Add(
			parser.Terminal{}.New(node.GetString(0), parser.NoTag),
			parser.Terminal{}.New(node.GetString(1), parser.NoTag),
		)
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
		quant.op = op
		quant.Add(op)
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
		op := parser.Terminal{}.New(node.GetString(0), parser.Tag("op"))
		quant.op = op
		quant.Add(
			op,
			compileNamedNode(node.GetNode(2)))
		/*
			// FIXME: These 2 are not present in the wbnf grammar
			if node.GetNode(1).Count() != 0 {
				quant.lbang = &Token{node.GetString(1)}
			}
			if node.GetNode(3).Count() != 0 {
				quant.rbang = &Token{node.GetString(3)}
			}*/

	}
	return &quant
}

func compileNamedNode(node parser.Node) parser.BaseNode {
	named := Named{
		op: nil,
	}
	if x := node.GetNode(0); x.Count() != 0 {
		x := x.GetNode(0)
		named.Add(
			IDENT{}.New(x.GetString(0), parser.NoTag),
			parser.Terminal{}.New(x.GetString(1), parser.Tag("op")),
		)
		named.op = named.AllChildren()[1]
	}

	named.Add(compileAtomNode(node.GetNode(1)))

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
				term.Add(compileNamedNode(x))
			case strings.HasPrefix(x.Tag, "?") && x.Count() > 0:
				term.Add(compileQuantNode(x.GetNode(0)))
				term.quantCount++
			}
		case parser.Scanner:
			switch term.Choice() {
			case 0, 1:
				op := parser.Terminal{}.New(x.String(), parser.Tag("op"))
				term.Add(op)
				term.op = op
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
		case 1:
			prod := stmt.GetNode(0)
			child.choice = 1
			child.Add(compileProdNode(prod))
		default:
			panic("")
		}
		g.Add(&child)
	}
	return &g
}
