//nolint:lll
package wbnf

import (
	"testing"

	"github.com/arr-ai/frozen"

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

func testParser(t *testing.T, test data, scope frozen.Map) {
	p := test.term.Parser("rule", newCache())
	var v parser.TreeElement
	scanner := parser.NewScanner(test.input)
	err := p.Parse(scope, scanner, &v)
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
			testParser(t, test, frozen.NewMap())
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
			testParser(t, test, frozen.NewMap())
		})
	}
}

func Test_seqParser(t *testing.T) {
	for _, test := range []data{
		{name: "simple-pass", term: Seq{S("1"), S("2")}, input: "12", success: true, nextSlice: 2},
		{name: "simple-fail", term: Seq{S("1"), S("2")}, input: "BLAA", success: false},
		{name: "pass-with-more-text", term: Seq{S("1"), S("2")}, input: "123abcTest1234", success: true, nextSlice: 2},
		{name: "partial-success", term: Seq{S("1"), S("2")}, input: "1BLAA", success: false, nextSlice: 1},
		{name: "simple-back-ref", term: Seq{Eq("a", S("1")), REF{Ident: "a"}}, input: "11", success: true, nextSlice: 2},
		{name: "default-ref-val", term: Seq{Eq("a", S("1")), REF{Ident: "b", Default: S("2")}}, input: "123", success: true, nextSlice: 2},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			testParser(t, test, frozen.NewMap())
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
			testParser(t, test, frozen.NewMap())
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
			testParser(t, test, frozen.NewMap())
		})
	}
}

func Test_delimParser(t *testing.T) {
	ab := Delim{Term: S("a"), Sep: S("b")}
	acb := Delim{Term: S("a"), Sep: S("b"), CanStartWithSep: true}
	abc := Delim{Term: S("a"), Sep: S("b"), CanEndWithSep: true}
	acbc := Delim{Term: S("a"), Sep: S("b"), CanStartWithSep: true, CanEndWithSep: true}

	for _, test := range []data{
		{name: "a:b-pass", term: ab, input: "ababa", success: true, nextSlice: 5},
		{name: "a:b-pass-no-b", term: ab, input: "a", success: true, nextSlice: 1},
		{name: "a:b-fail-leading-b", term: ab, input: "bababa", success: false},
		{name: "a:b-ignore-trailing-b", term: ab, input: "ababab", success: true, nextSlice: 5},

		{name: "a:,b-pass", term: acb, input: "ababa", success: true, nextSlice: 5},
		{name: "a:,b-pass-no-b", term: acb, input: "a", success: true, nextSlice: 1},
		{name: "a:,b-pass-leading-b", term: acb, input: "bababa", success: true, nextSlice: 6},
		{name: "a:,b-ignore-trailing-b", term: acb, input: "ababab", success: true, nextSlice: 5},

		{name: "a:b,-pass", term: abc, input: "ababa", success: true, nextSlice: 5},
		{name: "a:b,-pass-no-b", term: abc, input: "a", success: true, nextSlice: 1},
		{name: "a:b,-fail-leading-b", term: abc, input: "bababa", success: false},
		{name: "a:b,-pass-trailing-b", term: abc, input: "ababab", success: true, nextSlice: 6},

		{name: "a:,b,-pass", term: acbc, input: "ababa", success: true, nextSlice: 5},
		{name: "a:,b,-pass-no-b", term: acbc, input: "a", success: true, nextSlice: 1},
		{name: "a:,b,-pass-leading-b", term: acbc, input: "bababa", success: true, nextSlice: 6},
		{name: "a:,b,-pass-trailing-b", term: acbc, input: "ababab", success: true, nextSlice: 6},
		{name: "a:,b,-pass-surrounding-b", term: acbc, input: "bababab", success: true, nextSlice: 7},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			testParser(t, test, frozen.NewMap())
		})
	}
}

func Test_refParser(t *testing.T) {
	// Need to build up some data in the test scope
	scope := frozen.NewMap()
	for _, val := range []struct {
		ident string
		t     Term
		input string
	}{
		{ident: "a", t: S("a"), input: "a"},
		{ident: "b", t: Seq{S("a"), S("x")}, input: "ax"},
		{ident: "z", t: Delim{Term: S("a"), Sep: S("x")}, input: "axaxaxa"},
	} {
		p := val.t.Parser("rule", newCache())
		var v parser.TreeElement
		scanner := parser.NewScanner(val.input)
		err := p.Parse(scope, scanner, &v)
		require.NoError(t, err)

		scope = NewScopeWith(scope, val.ident, p, v)
	}
	for _, test := range []data{
		{name: "ref-a-pass", term: REF{Ident: "a", Default: nil}, input: "ababa", success: true, nextSlice: 1},
		{name: "ref-b-pass", term: REF{Ident: "b", Default: nil}, input: "axaba", success: true, nextSlice: 2},
		{name: "ref-z-pass", term: REF{Ident: "z", Default: nil}, input: "axaxaxaaxaba", success: true, nextSlice: 7},
		{name: "ref-a-fail", term: REF{Ident: "a", Default: nil}, input: "xaba", success: false},
		{name: "ref-b-fail", term: REF{Ident: "b", Default: nil}, input: "abxaba", success: false, nextSlice: 1},
		{name: "ref-b-fail-dont-take-default", term: REF{Ident: "b", Default: S("hello")}, input: "abxaba", success: false, nextSlice: 1},
		{name: "missing-ref-no-default", term: REF{Ident: "c", Default: nil}, input: "abxaba", success: false},
		{name: "missing-ref-default", term: REF{Ident: "c", Default: S("foo")}, input: "fooabxaba", success: true, nextSlice: 3},
	} {
		test := test

		t.Run(test.name, func(t *testing.T) {
			testParser(t, test, scope)
		})
	}
}
