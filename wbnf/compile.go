package wbnf

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
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
	case "term", "REF", "":
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
		if strings.HasPrefix(s, "/{") {
			s = s[2 : len(s)-1]
		}
		return parser.RE(s)
	case "REF":
		refNode := atom.OneRef()
		ref := parser.REF{
			Ident:   refNode.OneIdent().String(),
			Default: nil,
		}
		if defTerm := refNode.OneDefault().String(); defTerm != "" {
			ref.Default = parser.S(parseString(defTerm))
		}
		return ref
	case "term":
		return buildTerm(*atom.OneTerm())
	}
	// Must be the empty term '()'
	return parser.Seq{}
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
		sep := buildNamed(*q.OneNamed())
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
	atom := buildAtom(*n.OneAtom())
	if ident := n.OneIdent().String(); ident != "" {
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
		var sg *parser.ScopedGrammar
		if g := t.AllGrammar(); len(g) == 1 {
			nested := NewFromAst(g[0].Node)
			sg = &parser.ScopedGrammar{
				Grammar: nested,
			}
		}
		if len(terms) == 1 {
			if sg != nil {
				sg.Term = terms[0]
				return *sg
			}
			return terms[0]
		}
		if sg != nil {
			sg.Term = append(parser.Seq{}, terms...)
			return *sg
		}
		return append(parser.Seq{}, terms...)
	}
	// named and quants need to be added backwards
	// "a":","*     ->   Any(Delim(... S("a")))
	next := buildNamed(*t.OneNamed())
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

func NewFromAst(node ast.Node) parser.Grammar {
	g := parser.Grammar{}
	tree := NewGrammarNode(node)
	for _, stmt := range tree.AllStmt() {
		if prod := stmt.OneProd(); prod != nil {
			g[parser.Rule(prod.OneIdent().String())] = buildProd(*prod)
		}
	}
	return g
}

func mergeGrammarNodes(a, b ast.Branch) ast.Node {
	a["stmt"] = append(a["stmt"].(ast.Many), b.Many("stmt")...)
	return a
}

type ImportResolver interface {
	// Resolve returns the absolute path of the file at the location 'path' relative to the 'from' path
	Resolve(from, path string) string
}

type compiler struct {
	imports  map[string]GrammarNode
	resolver ImportResolver
}

func (c *compiler) makeGrammar(filename, text string) (GrammarNode, error) {
	node, err := ParseString(text)
	if err != nil {
		return GrammarNode{}, err
	}
	WalkerOps{
		EnterPragmaImportNode: func(impNode PragmaImportNode) Stopper {
			importPath := filepath.Join(impNode.OnePath().AllToken()...)
			if c.resolver != nil {
				importPath = c.resolver.Resolve(filename, importPath)
			}
			nested, nestedErr := c.loadGrammarFile(importPath)
			if nestedErr != nil {
				err = nestedErr
				return &aborter{}
			}
			if nested.Node != nil {
				x := mergeGrammarNodes(node.Node.(ast.Branch), nested.Node.(ast.Branch))
				node = GrammarNode{Node: x}
			}
			return nil
		},
	}.Walk(node)
	return node, nil
}

func (c *compiler) loadGrammarFile(filename string) (GrammarNode, error) {
	filename = filepath.Clean(filename)
	if _, has := c.imports[filename]; !has {
		text, err := ioutil.ReadFile(filename)
		if err != nil {
			return GrammarNode{}, err
		}
		g, err := c.makeGrammar(filename, string(text))
		if err != nil {
			return GrammarNode{}, err
		}
		c.imports[filename] = g
	}
	return c.imports[filename], nil
}

func Compile(grammar string, resolver ImportResolver) (parser.Parsers, error) {
	c := compiler{
		imports:  map[string]GrammarNode{},
		resolver: resolver,
	}
	node, err := c.makeGrammar("", grammar)
	if err != nil {
		return parser.Parsers{}, err
	}
	if err := validate(node); err != nil {
		return parser.Parsers{}, err
	}
	return NewFromAst(node).Compile(node), nil
}

func MustCompile(grammar string, resolver ImportResolver) parser.Parsers {
	p, err := Compile(grammar, resolver)
	if err != nil {
		panic(err)
	}
	return p
}
