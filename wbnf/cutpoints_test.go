package wbnf

import (
	"testing"

	"github.com/arr-ai/frozen"
	"github.com/stretchr/testify/assert"

	"github.com/arr-ai/wbnf/parser"
)

func TestStringCutpoints(t *testing.T) {
	g := parser.Grammar{"a": parser.Seq{parser.S("hello"), parser.S("A"), parser.S("b"), parser.S("A")}}

	idents := findUniqueStrings(g)

	assert.EqualValues(
		t,
		frozen.NewSet[string]("hello", "b").OrderedElements(stringLess),
		idents.OrderedElements(stringLess),
	)
}

func TestStringCutpointsInNestedGrammar(t *testing.T) {
	// a -> foo "->"  { foo -> "a" "->"; };
	// "a" is unique, "->" is not
	g := parser.Grammar{"a": parser.ScopedGrammar{
		Term:    parser.Seq{parser.Rule("foo"), parser.S("->")},
		Grammar: parser.Grammar{"foo": parser.Seq{parser.S("a"), parser.S("->")}},
	}}

	idents := findUniqueStrings(g)

	assert.EqualValues(t, frozen.NewSet[string]("a").Elements(), idents.Elements())
}

func stringLess(a, b string) bool {
	return a < b
}
