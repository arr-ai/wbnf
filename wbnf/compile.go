package wbnf

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
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

var whitespaceRE = regexp.MustCompile(`\s`)
var escapedSpaceRE = regexp.MustCompile(`((?:\A|[^\\])(?:\\\\)*)\\_`)

func compileAtomNode(node Node) Term {
	atom := node.(Branch)
	x, _ := Which(atom, "RE", "STR", "IDENT", "REF", "term")
	name := ""
	switch x {
	case "term", "":
	case "REF":
		name = atom.MustOne(x).MustOne("IDENT").Scanner().String()
	default:
		name = atom.MustOne(x).Scanner().String()
	}
	switch x {
	case "IDENT":
		return Rule(name)
	case "STR":
		return S(parseString(name))
	case "RE":
		s := whitespaceRE.ReplaceAllString(name, "")
		// Do this twice to cover adjacent escaped spaces `\_\_`.
		s = escapedSpaceRE.ReplaceAllString(s, "$1 ")
		s = escapedSpaceRE.ReplaceAllString(s, "$1 ")
		return RE(s)
	case "REF":
		return REF(name)
	case "term":
		return compileTermNode(atom.MustOne(x))
	}
	panic("bad input")
}

func compileTermNamedNode(node Node) Term {
	named := node.(Branch)
	atom := compileAtomNode(named.MustOne("atom"))

	if named.One("IDENT") != nil {
		return Eq(named.MustOne("IDENT").Scanner().String(), atom)
	}
	return atom
}

func compileQuantNode(node Node, term Term) Term {
	switch node.MustMany(ChoiceTag)[0].(Extra).Data.(int) {
	case 0:
		switch node.MustOne("op").Scanner().String() {
		case "*":
			return Any(term)
		case "?":
			return Opt(term)
		case "+":
			return Some(term)
		}
	case 1:
		min := 0
		max := 0
		if x := node.One("min"); x != nil {
			val, err := strconv.Atoi(x.Scanner().String())
			if err != nil {
				panic(err)
			}
			min = val
		}
		if x := node.One("max"); x != nil {
			val, err := strconv.Atoi(x.Scanner().String())
			if err != nil {
				panic(err)
			}
			max = val
		}
		return Quant{Term: term, Min: min, Max: max}
	case 2:
		assoc := NewAssociativity(node.MustOne("op").Scanner().String())
		sep := compileTermNamedNode(node.MustOne("named"))
		delim := Delim{Term: term, Sep: sep, Assoc: assoc}
		if node.One("opt_leading") != nil {
			delim.CanStartWithSep = true
		}
		if node.One("opt_trailing") != nil {
			delim.CanEndWithSep = true
		}
		return delim
	}
	panic("bad input")
}

func compileTermNode(node Node) Term {
	term := node.(Branch)
	x, _ := Which(term, "term", "atom", "named")
	switch x {
	case "term":
		var terms []Term
		for _, t := range term.MustMany("term") {
			terms = append(terms, compileTermNode(t))
		}
		if ops := term.Many("op"); len(ops) > 0 {
			switch ops[0].Scanner().String() {
			case "|":
				return append(Oneof{}, terms...)
			case ">":
				return append(Stack{}, terms...)
			}
		}
		return append(Seq{}, terms...)

	case "atom": // FIXME: This shouldnt actually be here,
		// there is a bug in the node collapse() which causes the `named` term to be @skip-eed
		return compileTermNamedNode(term)
	case "named":
		// named and quants need to be added backwards
		// "a":","*     ->   Any(Delim(... S("a")))
		next := compileTermNamedNode(term.MustOne("named"))
		quants := term.Many("quant")
		for i := range quants {
			next = compileQuantNode(quants[len(quants)-1-i], next)
		}
		return next

	}
	return nil
}

func compileProdNode(node Node) Term {
	prod := Seq{}
	terms := node.MustMany("term")
	for _, t := range terms {
		prod = append(prod, compileTermNode(t))
	}
	if len(prod) == 1 {
		return prod[0]
	}
	return prod
}

// NewFromNode converts the output from parsing an input via GrammarGrammar into
// a Grammar, which can then be used to generate parsers.
func NewFromAst(node Node) Grammar {
	g := Grammar{}
	for _, stmt := range node.MustMany("stmt") {
		if p := stmt.One("prod"); p != nil {
			g[Rule(p.MustOne("IDENT").Scanner().String())] = compileProdNode(p)
		}
	}
	return g
}
