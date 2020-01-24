package wbnf

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/arr-ai/wbnf/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertUnparse(t *testing.T, expected string, parsers Parsers, v interface{}) bool { //nolint:unparam
	var sb strings.Builder
	_, err := parsers.Unparse(v, &sb)
	return assert.NoError(t, err) && assert.Equal(t, expected, sb.String())
}

var expr = Rule("expr")

var exprGrammarSrc = `
// Simple expression grammar
expr -> @:/{([-+])}
      ^ @:/{([*/])}
      ^ "-"? @
	  ^ /{(\d+)} | @
	  ^ @<:"**"
      ^ "(" @ ")";
`

var exprGrammar = Grammar{
	expr: Stack{
		Delim{Term: at, Sep: RE(`([-+])`)},
		Delim{Term: at, Sep: RE(`([*/])`)},
		Seq{Opt(S("-")), at},
		Oneof{RE(`(\d+)`), at},
		R2L(at, S("**")),
		Seq{S("("), at, S(")")},
	},
}

func assertEqualObjects(t *testing.T, expected, actual interface{}) bool { //nolint:unparam
	if assert.True(t, reflect.DeepEqual(expected, actual)) {
		return true
	}
	t.Logf("raw expected: %#v", expected)
	t.Logf("raw actual:   %#v", actual)

	expectedJSON, err := json.Marshal(expected)
	require.NoError(t, err)
	actualJSON, err := json.Marshal(actual)
	require.NoError(t, err)
	t.Log("JSON(expected): ", string(expectedJSON))
	t.Log("JSON(actual):   ", string(actualJSON))

	assert.JSONEq(t, string(expectedJSON), string(actualJSON))

	return false
}

func assertParseToNode(t *testing.T, expected parser.Node, rule Rule, input *parser.Scanner) bool { //nolint:unparam
	parsers := Core()
	v, err := parsers.Parse(rule, input)
	if assert.NoError(t, err) {
		if assert.NoError(t, parsers.ValidateParse(v)) {
			return parser.AssertEqualNodes(t, expected, v.(parser.Node))
		}
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

var stackNamePrefixRE = regexp.MustCompile(`^([a-z\.]*)(?:` + regexp.QuoteMeta(StackDelim) + `(\d+))?\\`)

func (s *stackBuilder) a(name string, extras ...interface{}) *stackBuilder {
	var extra interface{}
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

func (s *stackBuilder) z(children ...interface{}) parser.Node {
	if children == nil {
		children = []interface{}{}
	}
	s.stack[len(s.stack)-1].Children = children
	for i := len(s.stack) - 1; i > 0; i-- {
		s.stack[i-1].Children = []interface{}{*s.stack[i]}
	}
	return *s.stack[0]
}

func stack(name string, extras ...interface{}) *stackBuilder {
	return (&stackBuilder{}).a(name, extras...)
}

func TestParseNamedTerm(t *testing.T) {
	r := parser.NewScanner(`opt=""`)
	x := stack(`term`, NonAssociative).a(`term@1`, NonAssociative).a(`term@2`).a(`term@3`).z(
		stack(`named`).z(
			stack(`?`).a(`_`).z(*r.Slice(0, 3), *r.Slice(3, 4)),
			stack(`atom`, 1).z(*r.Slice(4, 6)),
		),
		stack(`?`).z(),
	)
	assertParseToNode(t, x, term, r)
}

func TestParseNamedTermInDelim(t *testing.T) {
	r := parser.NewScanner(`"1":op=","`)
	x := stack(`term`, NonAssociative).a(`term@1`, NonAssociative).a(`term@2`).a(`term@3`).z(
		stack(`named`).z(
			stack(`?`).z(),
			stack(`atom`, 1).z(*r.Slice(0, 3)),
		),
		stack(`?`).a(`quant`, 2).a(`_`).z(
			*r.Slice(3, 4),
			stack(`?`).z(),
			stack(`named`).z(
				stack(`?`).a(`_`).z(*r.Slice(4, 6), *r.Slice(6, 7)),
				stack(`atom`, 1).z(*r.Slice(7, 10)),
			),
			stack(`?`).z(),
		),
	)
	assertParseToNode(t, x, term, r)
}

func TestGrammarParser(t *testing.T) {
	t.Parallel()

	parsers := exprGrammar.Compile()

	r := parser.NewScanner("1+2*3")
	v, err := parsers.Parse(expr, r)
	require.NoError(t, err)
	assert.NoError(t, parsers.ValidateParse(v))
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
	assert.NoError(t, parsers.ValidateParse(v))
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
	v, err := parsers.Parse(GrammarRule, r)
	require.NoError(t, err, "r=%v\nv=%v", r.Context(), v)
	require.Equal(t, len(exprGrammarSrc), r.Offset(), "r=%v\nv=%v", r.Context(), v)
	assert.NoError(t, parsers.ValidateParse(v))
	assertUnparse(t,
		`// Simple expression grammar`+
			`expr->@:([-+])`+
			`^@:([*/])`+
			`^"-"?@`+
			`^(\d+)|@`+
			`^@<:"**"`+
			`^"("@")";`,
		parsers,
		v,
	)
}

func TestGrammarSnippet(t *testing.T) {
	t.Parallel()

	parsers := Core()
	r := parser.NewScanner(`prod+`)
	v, err := parsers.Parse(term, r)
	require.NoError(t, err)
	assert.Equal(t,
		`term║:[term@1║:[term@2[term@3[named[?[], atom║0[prod]], ?[quant║0[+]]]]]]`,
		fmt.Sprintf("%v", v),
	)
	assert.NoError(t, parsers.ValidateParse(v))
	assertUnparse(t, "prod+", parsers, v)
}

func TestTinyGrammarGrammarGrammar(t *testing.T) {
	t.Parallel()

	tiny := Rule("tiny")
	tinyGrammar := Grammar{tiny: S("x")}
	tinyGrammarSrc := `tiny -> "x";`

	parsers := Core()
	r := parser.NewScanner(tinyGrammarSrc)
	v, err := parsers.Parse(GrammarRule, r)
	require.NoError(t, err)
	e := v.(parser.Node)
	assert.NoError(t, parsers.ValidateParse(v))

	grammar2 := NewFromNode(e)
	assertEqualObjects(t, tinyGrammar, grammar2)
}

func TestExprGrammarGrammarGrammar(t *testing.T) {
	t.Parallel()

	parsers := Core()
	r := parser.NewScanner(exprGrammarSrc)
	v, err := parsers.Parse(GrammarRule, r)
	require.NoError(t, err)
	e := v.(parser.Node)
	assert.NoError(t, parsers.ValidateParse(v))

	grammar2 := NewFromNode(e)
	assertEqualObjects(t, exprGrammar, grammar2)
}

func TestBacktrackGrammar(t *testing.T) {
	t.Parallel()

	parsers := MustCompile(`a -> ("x" ":" "x"+ ";"?)+;`)
	_, err := parsers.Parse(Rule("a"), parser.NewScanner(`x:x;x:x`))
	assert.NoError(t, err)

	// TODO: Make this work. Probably requires an LL(k) or LL(*) parser.
	// _, err = parsers.Parse(Rule("a"), parser.NewScanner(`x:xx:x`))
	// assert.NoError(t, err)
}
