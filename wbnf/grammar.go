package wbnf

import (
	"fmt"

	"github.com/arr-ai/wbnf/parser"
	"github.com/arr-ai/wbnf/parser/diff"
)

// Build the grammar grammar from grammarGrammarSrc and check that it matches
// grammarGrammar.
var core = func() parser.Parsers {
	g := MustCompile(grammarGrammarSrc, nil)
	newGrammarGrammar := g.Grammar()

	a := Grammar().Grammar()
	b := newGrammarGrammar
	if diff := diff.Grammars(a, b); !diff.Equal() {
		panic(fmt.Errorf(
			"mismatch between parsed and hand-crafted core grammar"+
				"\nold: %v"+
				"\nnew: %v"+
				"\ndiff: %#v",
			a, b, diff,
		))
	}
	return newGrammarGrammar.Compile(g.Node())
}()

func Core() parser.Parsers {
	return core
}
