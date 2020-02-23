package wbnf

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"testing"

	"github.com/arr-ai/wbnf/ast"

	"github.com/arr-ai/wbnf/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertUnparse(t *testing.T, expected string, parsers parser.Parsers, v parser.TreeElement) bool { //nolint:unparam
	var sb strings.Builder
	_, err := parsers.Unparse(v, &sb)
	return assert.NoError(t, err) && assert.Equal(t, expected, sb.String())
}

var expr = parser.Rule("expr")

var exprGrammarSrc = `
// Simple expression grammar
expr -> @:[-+]
      > @:[*/]
      > "-"? @
      > \d+ | @
      > @<:"**"
      > "(" @ ")";
`

var exprGrammar = parser.Grammar{
	expr: parser.Stack{
		parser.Delim{Term: parser.At, Sep: parser.RE(`[-+]`)},
		parser.Delim{Term: parser.At, Sep: parser.RE(`[*/]`)},
		parser.Seq{parser.Opt(parser.S("-")), parser.At},
		parser.Oneof{parser.RE(`\d+`), parser.At},
		parser.R2L(parser.At, parser.S("**")),
		parser.Seq{parser.S("("), parser.At, parser.S(")")},
	},
}

func assertParseToNode(t *testing.T, expected parser.Node, rule parser.Rule, input *parser.Scanner) bool { //nolint:unparam
	parsers := Core()
	v, err := parsers.Parse(rule, input)
	if assert.NoError(t, err) {
		return parser.AssertEqualNodes(t, expected, v.(parser.Node))
	} else {
		t.Logf("input: %s", input.Context())
	}
	return false
}

type stackBuilder struct {
	stack  []*parser.Node
	prefix string
	level  int
}

var stackNamePrefixRE = regexp.MustCompile(`^([a-z\.]*)(?:` + regexp.QuoteMeta(parser.StackDelim) + `(\d+))?\\`)

func (s *stackBuilder) a(name string, extras ...parser.Extra) *stackBuilder {
	var extra parser.Extra
	switch len(extras) {
	case 0:
	case 1:
		extra = extras[0]
	default:
		panic("Too many extras")
	}
	if prefixMatch := stackNamePrefixRE.FindStringSubmatch(name); prefixMatch != nil {
		if prefix := prefixMatch[1]; prefix != "" {
			s.prefix = prefix
			s.level = 0
		} else {
			s.level++
			name = fmt.Sprintf("%s#%d%s", s.prefix, s.level, name)
		}
	}
	s.stack = append(s.stack, parser.NewNode(name, extra))
	return s
}

func (s *stackBuilder) z(children ...parser.TreeElement) parser.Node {
	if children == nil {
		children = []parser.TreeElement{}
	}
	s.stack[len(s.stack)-1].Children = children
	for i := len(s.stack) - 1; i > 0; i-- {
		s.stack[i-1].Children = []parser.TreeElement{*s.stack[i]}
	}
	return *s.stack[0]
}

func stack(name string, extras ...parser.Extra) *stackBuilder {
	return (&stackBuilder{}).a(name, extras...)
}

func TestParseNamedTerm(t *testing.T) {
	r := parser.NewScanner(`opt=""`)
	x := stack(`term`, parser.NonAssociative).z(
		stack(`_`).z(stack(`term@1`, parser.NonAssociative).a(`term@2`).a(`term@3`).z(
			stack(`named`).z(
				stack(`?`).a(`_`).z(*r.Slice(0, 3), *r.Slice(3, 4)),
				stack(`atom`, parser.Choice(1)).z(*r.Slice(4, 6)),
			), stack(`?`).z(),
		), stack(`?`).z(),
		))
	assertParseToNode(t, x, "term", r)
}

func TestParseNamedTermInDelim(t *testing.T) {
	r := parser.NewScanner(`"1":op=","`)
	x := stack(`term`, parser.NonAssociative).z(
		stack(`_`).z(stack(`term@1`, parser.NonAssociative).a(`term@2`).a(`term@3`).z(
			stack(`named`).z(
				stack(`?`).z(),
				stack(`atom`, parser.Choice(1)).z(*r.Slice(0, 3)),
			),
			stack(`?`).a(`quant`, parser.Choice(2)).a(`_`).z(
				*r.Slice(3, 4),
				stack(`?`).z(),
				stack(`named`).z(
					stack(`?`).a(`_`).z(*r.Slice(4, 6), *r.Slice(6, 7)),
					stack(`atom`, parser.Choice(1)).z(*r.Slice(7, 10)),
				),
				stack(`?`).z(),
			),
		), stack(`?`).z(),
		))
	assertParseToNode(t, x, "term", r)
}

func TestGrammarParser(t *testing.T) {
	t.Parallel()

	parsers := exprGrammar.Compile(nil)

	r := parser.NewScanner("1+2*3")
	v, err := parsers.Parse(expr, r)
	require.NoError(t, err)
	assertUnparse(t, "1+2*3", parsers, v)
	assert.Equal(t,
		`expr║:[expr@1║:[expr@2[?[], expr@3║0[1]]], `+
			`+, `+
			`expr@1║:[expr@2[?[], expr@3║0[2]], *, expr@2[?[], expr@3║0[3]]]]`,
		fmt.Sprintf("%v", v),
	)

	r = parser.NewScanner("1+(2-3/4)")
	v, err = parsers.Parse(expr, r)
	assert.NoError(t, err)
	assertUnparse(t, "1+(2-3/4)", parsers, v)
	assert.Equal(t,
		`expr║:[`+
			`expr@1║:[expr@2[?[], expr@3║0[1]]], `+
			`+, `+
			`expr@1║:[expr@2[?[], expr@3║1[expr@4║:[expr@5[(, `+
			`expr║:[expr@1║:[expr@2[?[], expr@3║0[2]]], `+
			`-, `+
			`expr@1║:[expr@2[?[], expr@3║0[3]], `+
			`/, `+
			`expr@2[?[], expr@3║0[4]]]], `+
			`)]]]]]]`,
		fmt.Sprintf("%v", v),
	)
}

func TestExprGrammarGrammar(t *testing.T) {
	t.Parallel()

	parsers := Core()
	r := parser.NewScanner(exprGrammarSrc)
	v, err := parsers.Parse("grammar", r)
	require.NoError(t, err, "r=%v\nv=%v", r.Context(), v)
	require.Equal(t, len(exprGrammarSrc), r.Offset(), "r=%v\nv=%v", r.Context(), v)
	assertUnparse(t,
		`// Simple expression grammar`+
			`expr->@:[-+]`+
			`>@:[*/]`+
			`>"-"?@`+
			`>\d+|@`+
			`>@<:"**"`+
			`>"("@")";`,
		parsers,
		v,
	)
}

func TestGrammarSnippet(t *testing.T) {
	t.Parallel()

	parsers := Core()
	r := parser.NewScanner(`prod+`)
	v, err := parsers.Parse("term", r)
	require.NoError(t, err)
	assert.Equal(t,
		`term║:[_[term@1║:[term@2[term@3[named[?[], atom║0[prod]], ?[quant║0[+]]]]], ?[]]]`,
		fmt.Sprintf("%v", v),
	)
	assertUnparse(t, "prod+", parsers, v)
}

func TestTinyGrammarGrammarGrammar(t *testing.T) {
	t.Parallel()

	tiny := parser.Rule("tiny")
	tinyGrammar := parser.Grammar{tiny: parser.S("x")}
	tinyGrammarSrc := `tiny -> "x";`

	parsers := Core()
	r := parser.NewScanner(tinyGrammarSrc)
	v, err := parsers.Parse("grammar", r)
	require.NoError(t, err)
	e := v.(parser.Node)

	grammar2 := NewFromAst(ast.FromParserNode(parsers.Grammar(), e))
	assert.EqualValues(t, tinyGrammar, grammar2)
}

func TestExprGrammarGrammarGrammar(t *testing.T) {
	t.Parallel()

	parsers := Core()
	r := parser.NewScanner(exprGrammarSrc)
	v, err := parsers.Parse("grammar", r)
	require.NoError(t, err)
	e := v.(parser.Node)

	grammar2 := NewFromAst(ast.FromParserNode(parsers.Grammar(), e))
	assert.EqualValues(t, exprGrammar, grammar2)
}

func TestBacktrackGrammar(t *testing.T) {
	t.Parallel()

	parsers := MustCompile(`a -> ("x" ":" "x"+ ";"?)+;`, nil)
	_, err := parsers.Parse(parser.Rule("a"), parser.NewScanner(`x:x;x:x`))
	assert.NoError(t, err)

	// TODO: Make this work. Probably requires an LL(k) or LL(*) parser.
	// _, err = parsers.Parse(Rule("a"), parser.NewScanner(`x:xx:x`))
	// assert.NoError(t, err)
}

func TestCombo1(t *testing.T) {
	t.Parallel()

	p, err := Compile(`x -> tuple=("(" "1":",",? ")");`, nil)
	assert.NoError(t, err)
	log.Print(p.Grammar())
}

func TestCombo2(t *testing.T) {
	t.Parallel()

	p, err := Compile(`x -> rel=("{" names ("(" @:",", ")"):",",? "}"); names -> "";`, nil)
	assert.NoError(t, err)
	log.Print(p.Grammar())
}

func TestEmptyNamedTerm(t *testing.T) {
	t.Parallel()

	p, err := Compile(`x -> rel=();`, nil)
	assert.NoError(t, err)
	log.Print(p.Grammar())
}

func TestScopeGrammar(t *testing.T) {
	t.Parallel()
	g := parser.Grammar{
		"a": parser.ScopedGrammar{
			Term: parser.Seq{parser.S("a"), parser.Rule("b"), parser.Rule("c")},
			Grammar: parser.Grammar{
				"b": parser.S("c"),
				"c": parser.S("C"),
			},
		},
		"c": parser.S("foo"),
	}
	p := g.Compile(nil)

	te := p.MustParse("a", parser.NewScanner("acC"))
	tree := ast.FromParserNode(g, te)
	te2 := ast.ToParserNode(g, tree)

	parser.AssertEqualNodes(t, te.(parser.Node), te2.(parser.Node))
}

func TestScopeGrammarwithWrapping(t *testing.T) {
	t.Parallel()
	g := parser.Grammar{
		".wrapRE": parser.RE(`\s*()\s*`),
		"pragma": parser.ScopedGrammar{Term: parser.Oneof{parser.Rule(`import`)},
			Grammar: parser.Grammar{"import": parser.Seq{parser.S(".import"),
				parser.Eq("path",
					parser.Delim{Term: parser.Oneof{parser.S(".."),
						parser.S("."),
						parser.RE(`[a-zA-Z0-9]+`)},
						Sep:             parser.S("/"),
						Assoc:           parser.NonAssociative,
						CanStartWithSep: true})}}},
	}
	p := g.Compile(nil)

	te := p.MustParse("pragma", parser.NewScanner(".import foowbnf"))
	tree := ast.FromParserNode(g, te)
	te2 := ast.ToParserNode(g, tree)

	parser.AssertEqualNodes(t, te.(parser.Node), te2.(parser.Node))
}
