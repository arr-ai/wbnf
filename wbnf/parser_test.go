//nolint:lll
package wbnf

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/arr-ai/wbnf/parser"
	"github.com/stretchr/testify/assert"
)

func newCache() cache {
	return cache{
		parsers:    map[Rule]parser.Parser{},
		grammar:    Grammar{},
		rulePtrses: map[Rule][]*parser.Parser{},
	}
}

type data struct {
	name      string
	term      Term
	input     string
	success   bool
	nextSlice int
}

func testParser(t *testing.T, test data) {
	p := test.term.Parser("rule", newCache())
	var v interface{}
	scanner := parser.NewScanner(test.input)
	err := p.Parse(scanner, &v)
	if test.success {
		require.NoError(t, err)
		require.NotNil(t, v)
		switch res := v.(type) {
		case parser.Scanner:
			assert.Equal(t, test.input[:test.nextSlice], res.String())
		case parser.Node:
			// todo
		default:
			assert.Fail(t, "unhandled return type")
		}
	} else {
		assert.Error(t, err)
		assert.Nil(t, v)
	}
	assert.Equal(t, test.input[test.nextSlice:], scanner.String())
}

func Test_sParser(t *testing.T) {
	for _, test := range []data{
		{name: "simple-pass", input: "test", success: true},
		{name: "simple-fail", input: "blaa", success: false},
		{name: "pass-with-more-text", input: "test1234", success: true},
		{name: "pass-with-whitespace-text", input: "test  1234", success: true},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			var s S = "test"
			test.term = s
			if test.success {
				test.nextSlice = len(string(s))
			}
			testParser(t, test)
		})
	}
}

func Test_reParser(t *testing.T) {
	for _, test := range []data{
		{name: "simple-pass", input: "123abc", success: true, nextSlice: len("123abc")},
		{name: "simple-fail", input: "BLAA", success: false},
		{name: "pass-with-more-text", input: "123abcTest1234", success: true, nextSlice: len("123abc")},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			var s RE = `\d*[a-z]+`
			test.term = s
			testParser(t, test)
		})
	}
}

func Test_seqParser(t *testing.T) {
	for _, test := range []data{
		{name: "simple-pass", term: Seq{S("1"), S("2")}, input: "12", success: true, nextSlice: 2},
		{name: "simple-fail", term: Seq{S("1"), S("2")}, input: "BLAA", success: false},
		{name: "pass-with-more-text", term: Seq{S("1"), S("2")}, input: "123abcTest1234", success: true, nextSlice: 2},
		{name: "partial-success", term: Seq{S("1"), S("2")}, input: "1BLAA", success: false, nextSlice: 1},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			testParser(t, test)
		})
	}
}

func Test_oneOfParser(t *testing.T) {
	for _, test := range []data{
		{name: "pass-choose-first", term: Oneof{S("1"), S("2")}, input: "12", success: true, nextSlice: 1},
		{name: "pass-choose-second", term: Oneof{S("1"), S("2")}, input: "21", success: true, nextSlice: 1},
		{name: "simple-fail", term: Oneof{S("1"), S("2")}, input: "BLAA", success: false},
		{name: "pass-with-more-text", term: Oneof{S("1"), S("2")}, input: "123abcTest1234", success: true, nextSlice: 1},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			testParser(t, test)
		})
	}
}

func Test_quantParser(t *testing.T) {
	for _, test := range []data{
		{name: "a+ none", term: Quant{Term: S("a"), Min: 1}, input: "2", success: false},
		{name: "a+ single", term: Quant{Term: S("a"), Min: 1}, input: "a2", success: true, nextSlice: 1},
		{name: "a+ many", term: Quant{Term: S("a"), Min: 1}, input: "aaaaaaaaaaaa2", success: true, nextSlice: 12},

		{name: "a{2,5} not enough", term: Quant{Term: S("a"), Min: 2, Max: 5}, input: "a2", success: false, nextSlice: 1},
		{name: "a{2,5} pass bottom", term: Quant{Term: S("a"), Min: 2, Max: 5}, input: "aa2", success: true, nextSlice: 2},
		{name: "a{2,5} pass top", term: Quant{Term: S("a"), Min: 2, Max: 5}, input: "aaaaa2", success: true, nextSlice: 5},
		{name: "a{2,5} too many", term: Quant{Term: S("a"), Min: 2, Max: 5}, input: "aaaaaaaa", success: true, nextSlice: 5},

		{name: "a* none", term: Quant{Term: S("a")}, input: "2", success: true},
		{name: "a* single", term: Quant{Term: S("a")}, input: "a2", success: true, nextSlice: 1},
		{name: "a* many", term: Quant{Term: S("a")}, input: "aaaaaaaaaaaa2", success: true, nextSlice: 12},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			testParser(t, test)
		})
	}
}

func Test_delimParser(t *testing.T) {
	for _, test := range []data{
		{name: "a:b-pass", term: Delim{Term: S("a"), Sep: S("b")}, input: "ababa", success: true, nextSlice: 5},
		{name: "a:b-pass-no-b", term: Delim{Term: S("a"), Sep: S("b")}, input: "a", success: true, nextSlice: 1},
		{name: "a:b-fail-leading-b", term: Delim{Term: S("a"), Sep: S("b")}, input: "bababa", success: false},
		{name: "a:b-fail-trailing-b", term: Delim{Term: S("a"), Sep: S("b")}, input: "ababab", success: false, nextSlice: 6},

		{name: "a:,b-pass", term: Delim{Term: S("a"), Sep: S("b"), CanStartWithSep: true}, input: "ababa", success: true, nextSlice: 5},
		{name: "a:,b-pass-no-b", term: Delim{Term: S("a"), Sep: S("b"), CanStartWithSep: true}, input: "a", success: true, nextSlice: 1},
		{name: "a:,b-pass-leading-b", term: Delim{Term: S("a"), Sep: S("b"), CanStartWithSep: true}, input: "bababa", success: true, nextSlice: 6},
		{name: "a:,b-fail-trailing-b", term: Delim{Term: S("a"), Sep: S("b"), CanStartWithSep: true}, input: "ababab", success: false, nextSlice: 6},

		{name: "a:b,-pass", term: Delim{Term: S("a"), Sep: S("b"), CanEndWithSep: true}, input: "ababa", success: true, nextSlice: 5},
		{name: "a:b,-pass-no-b", term: Delim{Term: S("a"), Sep: S("b"), CanEndWithSep: true}, input: "a", success: true, nextSlice: 1},
		{name: "a:b,-fail-leading-b", term: Delim{Term: S("a"), Sep: S("b"), CanEndWithSep: true}, input: "bababa", success: false},
		{name: "a:b,-pass-trailing-b", term: Delim{Term: S("a"), Sep: S("b"), CanEndWithSep: true}, input: "ababab", success: true, nextSlice: 6},

		{name: "a:,b,-pass", term: Delim{Term: S("a"), Sep: S("b"), CanStartWithSep: true, CanEndWithSep: true}, input: "ababa", success: true, nextSlice: 5},
		{name: "a:,b,-pass-no-b", term: Delim{Term: S("a"), Sep: S("b"), CanStartWithSep: true, CanEndWithSep: true}, input: "a", success: true, nextSlice: 1},
		{name: "a:,b,-pass-leading-b", term: Delim{Term: S("a"), Sep: S("b"), CanStartWithSep: true, CanEndWithSep: true}, input: "bababa", success: true, nextSlice: 6},
		{name: "a:,b,-pass-trailing-b", term: Delim{Term: S("a"), Sep: S("b"), CanStartWithSep: true, CanEndWithSep: true}, input: "ababab", success: true, nextSlice: 6},
		{name: "a:,b,-pass-surrounding-b", term: Delim{Term: S("a"), Sep: S("b"), CanStartWithSep: true, CanEndWithSep: true}, input: "bababab", success: true, nextSlice: 7},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			testParser(t, test)
		})
	}
}
