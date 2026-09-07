package diff

import (
	"testing"

	"github.com/arr-ai/wbnf/parser"
	"github.com/stretchr/testify/assert"
)

func TestEmptyGrammarsEqual(t *testing.T) {
	t.Parallel()
	g := parser.Grammar{}
	assert.True(t, Grammars(g, g).Equal())
}

func TestSameSTermsEqual(t *testing.T) {
	t.Parallel()
	assert.True(t, Terms(parser.S("x"), parser.S("x")).Equal())
	assert.False(t, Terms(parser.S("x"), parser.S("y")).Equal())
}

func TestTermTypesDiffer(t *testing.T) {
	t.Parallel()
	assert.False(t, Terms(parser.S("x"), parser.Rule("x")).Equal())
}
