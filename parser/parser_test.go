package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// These test all verify the parser behaviour when the tested Term fails when inside a CutPoint scope
// the test will automatically add a Seq{CutPoint{S(":")}} around the test term and prefix the input data so it passes
func TestParserWithCutpointScope(t *testing.T) {
	for _, test := range []struct {
		name  string
		term  Term
		err   error // what type of error is expected when not in a cutpoint scope
		input string
	}{
		{name: "seq ok", term: Seq{S("a"), S("b")}, err: nil, input: "ab"},
		{name: "seq fail non fatal child", term: Seq{S("a"), S("b")}, err: ParseError{}, input: "b"},
		{name: "seq fail fatal child", term: Seq{CutPoint{S("a")}, S("b")}, err: FatalError{}, input: "a"},

		{name: "oneof ok", term: Oneof{S("1"), Seq{S("a"), CutPoint{S("b")}, S("c")}}, err: nil, input: "abc"},
		{name: "oneof fail not fatal", term: Oneof{S("1"), Seq{S("a"), CutPoint{S("b")}, S("c")}}, err: ParseError{}, input: "z"},
		{name: "oneof fail not fatal child", term: Oneof{S("1"), Seq{S("a"), CutPoint{S("b")}, S("c")}}, err: FatalError{}, input: "abd"},

		{name: "quant min = 1 ok", term: Some(S("1")), err: nil, input: "11"},
		{name: "quant min = 1 fail", term: Some(S("1")), err: ParseError{}, input: "2"},
		{name: "quant min = 1 fail seq", term: Some(Seq{CutPoint{S("1")}, S("2")}), err: FatalError{}, input: "1"},

		{name: "quant min = 0 ok", term: Any(S("1")), err: nil, input: ""},
		{name: "quant min = 0 fail", term: Any(S("1")), err: nil, input: ""},
		{name: "quant min = 0 fail seq", term: Any(Seq{CutPoint{S("1")}, S("2")}), err: FatalError{}, input: "1"},

		{name: "delim 'a':,'1',; ok", term: Delim{
			Term:            S("a"),
			Sep:             S("1"),
			CanStartWithSep: true,
			CanEndWithSep:   true,
		}, err: nil, input: "1a"},
		{name: "delim 'a':'1'; fail1", term: Delim{
			Term:            S("a"),
			Sep:             S("1"),
			CanStartWithSep: false,
			CanEndWithSep:   false,
		}, err: ParseError{}, input: "1"},
		{name: "delim 'a':,'1'; fail2", term: Delim{
			Term:            S("a"),
			Sep:             CutPoint{S("1")},
			CanStartWithSep: true,
			CanEndWithSep:   false,
		}, err: FatalError{}, input: "a1b"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Run("with cutpoint", func(t *testing.T) {
				p := Grammar{"a": Seq{CutPoint{S(":")}, test.term}}.Compile(nil)
				te, err := p.Parse("a", NewScanner(":"+test.input))
				if test.err == nil {
					assert.NotNil(t, te)
					assert.NoError(t, err)
				} else {
					assert.Nil(t, te)
					assert.Error(t, err)
					assert.IsType(t, FatalError{}, err)
				}
			})
			t.Run("without cutpoint", func(t *testing.T) {
				p := Grammar{"a": test.term}.Compile(nil)
				te, err := p.Parse("a", NewScanner(test.input))
				if test.err == nil {
					assert.NotNil(t, te)
					assert.NoError(t, err)
				} else {
					assert.Nil(t, te)
					assert.Error(t, err)
					assert.IsType(t, test.err, err)
				}
			})
		})
	}
}

func TestParserEscaping(t *testing.T) {
	g := Grammar{
		"x":       Seq{S("a"), RE("\\w+"), S("c")},
		".wrapRE": RE(`\s*()\s*`),
	}.Compile(nil)
	input := "a {:haha:} c"

	expected, err := g.Parse("x", NewScanner("a haha c"))
	assert.NoError(t, err)

	actual, err := g.ParseWithExternals("x", NewScanner(input), ExternalRefs{
		"*{:():}": func(scope Scope, input *Scanner) (element TreeElement, err error) {
			assert.EqualValues(t, "haha:} c", input.String())
			var eaten Scanner
			input = input.Eat(4, &eaten)
			return eaten, nil
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, expected.(Node).String(), actual.(Node).String())

}

func TestParserEscaping2LevelGrammar(t *testing.T) {
	g := Grammar{
		"x": Seq{S("a"), Named{
			Name: "ref",
			Term: S("haha"),
		}, S("c")},
		".wrapRE": RE(`\s*()\s*`),
	}.Compile(nil)

	g2 := Grammar{"x": S("haha")}.Compile(nil)
	input := "a {:haha     :} c"

	expected, err := g.Parse("x", NewScanner("a haha c"))
	assert.NoError(t, err)

	actual, err := g.ParseWithExternals("x", NewScanner(input), ExternalRefs{
		"*{:\\s*()\\s*:}": func(scope Scope, input *Scanner) (element TreeElement, err error) {
			return g2.Parse("x", input)
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, expected.(Node).String(), actual.(Node).String())
}
