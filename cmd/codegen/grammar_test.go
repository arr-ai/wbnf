package codegen

import (
	"testing"

	"github.com/arr-ai/wbnf/parser"
	"github.com/stretchr/testify/assert"
)

// TestWalkTermTerminal tests the the output of terminal nodes in AST (those
// with no children (noScope).
func TestWalkTermTerminal(t *testing.T) {
	// Note: These tests are far from comprehensive. They include enough
	// to test fixes for #59 (Generated Go code removes spaces from regexes)
	tests := map[string]struct {
		input  parser.Term
		output string
	}{
		"S simple":  {parser.S("hello"), "parser.S(`hello`)"},
		"S newline": {parser.S("\n"), "parser.S(`\n`)"},
		"RE simple": {parser.RE("[A-Z]+"), "parser.RE(`[A-Z]+`)"},
		"RE space":  {parser.RE("[ ]+"), "parser.RE(`[ ]+`)"},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.output, walkTerm(tt.input).name)
		})
	}
}
