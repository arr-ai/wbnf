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

func makeCOMMENT(s string) *COMMENT { x := COMMENT(s); return &x }
func makeIDENT(s string) *IDENT     { x := IDENT(s); return &x }
func makeINT(s string) *INT         { x := INT(s); return &x }
func makeSTR(s string) *STR         { x := STR(s); return &x }
func makeRE(s string) *RE           { x := RE(s); return &x }

func compileAtomNode(node parser.Node) isGenNode {
	atom := Atom{
		children:  nil,
		choice:    node.Extra.(int),
		numTokens: 0,
	}
	switch node.Extra.(int) {
	case 0:
		atom.children = []isGenNode{makeIDENT(node.GetString(0))}
	case 1:
		atom.children = []isGenNode{makeSTR(parseString(node.GetString(0)))}
	case 2:
		atom.children = []isGenNode{makeRE(strings.ReplaceAll(node.GetString(0), `\/`, `/`))}
	case 3:
		node := node.GetNode(0)
		atom.children = []isGenNode{
			&Token{v: node.GetString(0)},
			compileTermStackNode(node.GetNode(1)),
			&Token{v: node.GetString(2)},
		}
	case 4:
		node := node.GetNode(0)
		atom.children = []isGenNode{
			&Token{v: node.GetString(0)},
			&Token{v: node.GetString(1)},
		}
	default:
		panic("foo")
	}
	return &atom
}

func compileQuantNode(node parser.Node) isGenNode {
	quant := Quant{
		choice: node.Extra.(int),
	}
	switch node.Extra.(int) {
	case 0:
		quant.op = &Token{node.GetString(0)}
	case 1:
		if node.Count() != 5 {
			panic("ooops")
		}
		quant.children = []isGenNode{
			&Token{node.GetString(0)},
			makeINT(node.GetString(1)),
			&Token{node.GetString(2)},
			makeINT(node.GetString(3)),
			&Token{node.GetString(4)},
		}
		quant.min = quant.children[1]
		quant.max = quant.children[2]
	case 2:
		node = node.GetNode(0)
		if node.Count() != 4 {
			panic("ooops")
		}
		quant.op = &Token{node.GetString(0)}
		quant.children = []isGenNode{
			quant.op,
			compileNamedNode(node.GetNode(2)),
		}
		// FIXME: These 2 are not present in the wbnf grammar
		if node.GetNode(1).Count() != 0 {
			quant.lbang = &Token{node.GetString(1)}
		}
		if node.GetNode(3).Count() != 0 {
			quant.rbang = &Token{node.GetString(3)}
		}

	}
	return &quant
}

func compileNamedNode(node parser.Node) isGenNode {
	named := Named{
		children: nil,
		op:       nil,
	}
	if x := node.GetNode(0); x.Count() != 0 {
		x := x.GetNode(0)
		named.children = append(named.children, makeIDENT(x.GetString(0)))
		named.op = &Token{v: x.GetString(1)}
	}

	named.children = append(named.children, compileAtomNode(node.GetNode(1)))

	return &named
}

func compileTermStackNode(node parser.Node) isGenNode {
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
				term.children = append(term.children, compileTermStackNode(x))
				term.termCount++
			case strings.HasPrefix(x.Tag, "named"):
				term.children = append(term.children, compileNamedNode(x))
			case strings.HasPrefix(x.Tag, "?") && x.Count() > 0:
				term.children = append(term.children, compileQuantNode(x.GetNode(0)))
				term.quantCount++
			}
		case parser.Scanner:
			switch term.Choice() {
			case 0, 1:
				term.op = &Token{v: x.String()}
			default:
			}
		}
	}

	return &term
}

func compileProdNode(node parser.Node) isGenNode {
	prod := Prod{
		children: []isGenNode{
			makeIDENT(node.GetString(0)),
			&Token{node.GetString(1)},
		},
	}

	terms := node.GetNode(2)
	switch terms.Tag {
	case "?":

	}
	children := terms.Children
	prod.termCount = len(children)
	for _, child := range children {
		prod.children = append(prod.children, compileTermStackNode(child.(parser.Node)))
	}

	prod.children = append(prod.children, &Token{node.GetString(3)})

	return &prod
}

func FromNodes(node parser.Node) *Grammar {
	g := Grammar{
		children:  []isGenNode{},
		stmtCount: len(node.Children),
	}
	for _, v := range node.Children {
		stmt := v.(parser.Node)
		var child Stmt
		switch stmt.Extra.(int) {
		case 0:
			child.choice = 0
			child.children = []isGenNode{
				makeCOMMENT(v.(parser.Node).GetString(0)),
			}
		case 1:
			prod := stmt.GetNode(0)
			child.choice = 1
			child.children = []isGenNode{compileProdNode(prod)}
		default:
			panic("")
		}
		g.children = append(g.children, &child)
	}
	return &g
}
