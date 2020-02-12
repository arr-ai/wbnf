package wbnf

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/arr-ai/wbnf/ast"

	"github.com/arr-ai/wbnf/parser"
)

// unfakeBackquote replaces reversed prime with grave accent (backquote) in
// order to make the grammar below more readable.
func unfakeBackquote(s string) string {
	return strings.ReplaceAll(s, "‵", "`")
}

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

var whitespaceRE = regexp.MustCompile(`\s`)
var escapedSpaceRE = regexp.MustCompile(`((?:\A|[^\\])(?:\\\\)*)\\_`)

func buildAtom(atom AtomNode) parser.Term {
	x, _ := ast.Which(atom.Node.(ast.Branch), "RE", "STR", "IDENT", "REF", "term")
	name := ""
	switch x {
	case "term", "":
	case "REF":
		name = atom.OneIdent().String()
	default:
		name = atom.One(x).Scanner().String()
	}
	switch x {
	case "IDENT":
		return parser.Rule(name)
	case "STR":
		return parser.S(parseString(name))
	case "RE":
		s := whitespaceRE.ReplaceAllString(name, "")
		// Do this twice to cover adjacent escaped spaces `\_\_`.
		s = escapedSpaceRE.ReplaceAllString(s, "$1 ")
		s = escapedSpaceRE.ReplaceAllString(s, "$1 ")
		return parser.RE(s)
	case "REF":
		ref := parser.REF{
			Ident:   name,
			Default: nil,
		}
		defTerm := atom.OneRef().OneDefault().String()
		if defTerm != "" {
			ref.Default = parser.S(parseString(defTerm))
		}
		return ref
	case "term":
		return buildTerm(atom.OneTerm())
	}
	panic("bad input")
}

func buildQuant(q QuantNode, term parser.Term) parser.Term {
	switch q.Choice() {
	case 0:
		switch q.OneOp() {
		case "*":
			return parser.Any(term)
		case "?":
			return parser.Opt(term)
		case "+":
			return parser.Some(term)
		}
	case 1:
		min := 0
		max := 0
		if x := q.OneMin().String(); x != "" {
			val, err := strconv.Atoi(x)
			if err != nil {
				panic(err)
			}
			min = val
		}
		if x := q.OneMax().String(); x != "" {
			val, err := strconv.Atoi(x)
			if err != nil {
				panic(err)
			}
			max = val
		}
		return parser.Quant{Term: term, Min: min, Max: max}
	case 2:
		assoc := parser.NewAssociativity(q.OneOp())
		sep := buildNamed(q.OneNamed())
		delim := parser.Delim{Term: term, Sep: sep, Assoc: assoc}
		if q.OneOptLeading() != "" {
			delim.CanStartWithSep = true
		}
		if q.OneOptTrailing() != "" {
			delim.CanEndWithSep = true
		}
		return delim
	}
	panic("bad input")
}

func buildNamed(n NamedNode) parser.Term {
	atom := buildAtom(n.OneAtom())
	ident := n.OneIdent().String()
	if ident != "" {
		return parser.Eq(ident, atom)
	}
	return atom
}

func buildTerm(t TermNode) parser.Term {
	if len(t.AllTerm()) > 0 {
		var terms []parser.Term
		for _, t := range t.AllTerm() {
			terms = append(terms, buildTerm(t))
		}
		switch t.OneOp() {
		case "|":
			return append(parser.Oneof{}, terms...)
		case ">":
			return append(parser.Stack{}, terms...)
		}
		if len(terms) == 1 {
			return terms[0]
		}
		return append(parser.Seq{}, terms...)
	}
	// named and quants need to be added backwards
	// "a":","*     ->   Any(Delim(... S("a")))
	next := buildNamed(t.OneNamed())
	quants := t.AllQuant()
	for i := range quants {
		next = buildQuant(quants[len(quants)-1-i], next)
	}
	return next
}

func buildProd(p ProdNode) parser.Term {
	children := p.AllTerm()
	if len(children) == 1 {
		return buildTerm(children[0])
	}
	seq := make(parser.Seq, 0, len(children))
	for _, child := range children {
		seq = append(seq, buildTerm(child))
	}
	return seq
}

func buildGrammar(node ast.Node) parser.Grammar {
	g := parser.Grammar{}
	tree := NewGrammarNode(node)
	for _, stmt := range tree.AllStmt() {
		for _, prod := range stmt.AllProd() {
			g[parser.Rule(prod.OneIdent().String())] = buildProd(prod)
		}
	}
	return g
}

func Compile(grammar string) (parser.Parsers, error) {
	node, err := ParseString(grammar)
	if err != nil {
		return parser.Parsers{}, err
	}
	g := buildGrammar(node)
	return g.Compile(node), nil
}

func MustCompile(grammar string) parser.Parsers {
	p, err := Compile(grammar)
	if err != nil {
		panic(err)
	}
	return p
}
