package wbnf

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/arr-ai/wbnf/errors"
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

var whitespaceRE = regexp.MustCompile(`\s`)
var escapedSpaceRE = regexp.MustCompile(`((?:\A|[^\\])(?:\\\\)*)\\_`)

func compileAtomNode(node parser.Node) parser.Term {
	switch node.Extra.(parser.Choice) {
	case 0:
		return parser.Rule(node.GetString(0))
	case 1:
		return parser.S(parseString(node.GetString(0)))
	case 2:
		s := whitespaceRE.ReplaceAllString(node.GetString(0), "")
		// Do this twice to cover adjacent escaped spaces `\_\_`.
		s = escapedSpaceRE.ReplaceAllString(s, "$1 ")
		s = escapedSpaceRE.ReplaceAllString(s, "$1 ")
		return parser.RE(s)
	case 3:
		return compileRefNode(node.GetNode(0))
	case 4:
		return compileTermNode(node.GetNode(0, 1))
	case 5:
		return parser.Seq{}
	default:
		panic(errors.BadInput)
	}
}

func compileRefNode(node parser.Node) parser.Term {
	ref := parser.REF{Ident: node.GetString(1)}

	term := node.GetNode(2)
	if term.Count() != 0 {
		ref.Default = parser.S(parseString(term.GetString(0, 1)))
	}

	return ref
}

func compileTermNamedNode(node parser.Node) parser.Term {
	term := compileAtomNode(node.GetNode(1))
	if quant := node.GetNode(0); quant.Count() == 1 {
		return parser.Named{
			Name: quant.GetString(0, 0),
			Term: term,
		}
	}
	return term
}

func compileTermQuantNode(node parser.Node) parser.Term {
	term := compileTermNamedNode(node.GetNode(0))
	opt := node.GetNode(1)
	for _, child := range opt.Children {
		quant := child.(parser.Node)
		switch quant.Extra.(parser.Choice) {
		case 0:
			switch quant.GetString(0) {
			case "?":
				term = parser.Opt(term)
			case "*":
				term = parser.Any(term)
			case "+":
				term = parser.Some(term)
			default:
				panic(errors.BadInput)
			}
		case 1:
			seq := quant.GetNode(0)
			min, max := 0, 0
			minOpt := seq.GetNode(1)
			if minOpt.Count() == 1 {
				var err error
				min, err = strconv.Atoi(minOpt.GetString(0))
				if err != nil {
					panic(err)
				}
			}
			maxOpt := seq.GetNode(3)
			if maxOpt.Count() == 1 {
				var err error
				max, err = strconv.Atoi(maxOpt.GetString(0))
				if err != nil {
					panic(err)
				}
			}
			term = parser.Quant{Term: term, Min: min, Max: max}
		case 2:
			seq := quant.GetNode(0)
			assoc := parser.NewAssociativity(seq.GetString(0))
			term = parser.Delim{
				Term:            term,
				Sep:             compileTermNamedNode(seq.GetNode(2)),
				Assoc:           assoc,
				CanStartWithSep: seq.GetNode(1).Count() == 1,
				CanEndWithSep:   seq.GetNode(3).Count() == 1,
			}
		default:
			panic(errors.BadInput)
		}
	}
	return term
}

func compileTermSeqNode(node parser.Node) parser.Term {
	n := node.Count()
	if n == 1 {
		return compileTermQuantNode(node.Children[0].(parser.Node))
	}
	seq := make(parser.Seq, 0, node.Count())
	for _, child := range node.Children {
		seq = append(seq, compileTermQuantNode(child.(parser.Node)))
	}
	return seq
}

func compileTermOneofNode(node parser.Node) parser.Term {
	n := node.Count()
	if n == 1 {
		return compileTermSeqNode(node.GetNode(0))
	}
	oneof := make(parser.Oneof, 0, n/2+1)
	for i := 0; i < n; i += 2 {
		oneof = append(oneof, compileTermSeqNode(node.GetNode(i)))
	}
	return oneof
}

func compileTermStackNode(node parser.Node) parser.Term {
	if node.Count() == 1 {
		return compileTermOneofNode(node.GetNode(0))
	}
	var stack parser.Stack
	for i := 0; i < node.Count(); i += 2 {
		stack = append(stack, compileTermOneofNode(node.GetNode(i)))
	}
	return stack
}

func compileTermNode(node parser.Node) parser.Term {
	return compileTermStackNode(node)
}

func compileProdNode(node parser.Node) parser.Term {
	children := node.GetNode(2).Children
	if len(children) == 1 {
		return compileTermNode(children[0].(parser.Node))
	}
	seq := make(parser.Seq, 0, node.Count())
	for _, child := range children {
		seq = append(seq, compileTermNode(child.(parser.Node)))
	}
	return seq
}

// NewFromNode converts the output from parsing an input via GrammarGrammar into
// a parser.Grammar, which can then be used to generate parsers.
func NewFromNode(node parser.Node) parser.Grammar {
	g := parser.Grammar{}
	for _, v := range node.Children {
		stmt := v.(parser.Node)
		switch stmt.Extra.(parser.Choice) {
		case 0:
		// 	comment := v.(parse.Node).GetString(0)
		case 1:
			prod := stmt.GetNode(0)
			g[parser.Rule(prod.GetString(0))] = compileProdNode(prod)
		default:
			panic(errors.BadInput)
		}
	}
	return g
}

func Compile(grammar string) (parser.Parsers, error) {
	node, err := Core().Parse(GrammarRule, parser.NewScanner(grammar))
	if err != nil {
		return parser.Parsers{}, err
	}
	n := node.(parser.Node)
	return NewFromNode(n).Compile(&n), nil
}

func MustCompile(grammar string) parser.Parsers {
	node := Core().MustParse(GrammarRule, parser.NewScanner(grammar)).(parser.Node)

	return NewFromNode(node).Compile(&node)
}
