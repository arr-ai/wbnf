package wbnf

import (
	"fmt"

	"github.com/arr-ai/wbnf/parser"
	"github.com/arr-ai/wbnf/parser/diff"
)

// Build the grammar grammar from grammarGrammarSrc and check that it matches
// grammarGrammar.
var core = func() parser.Parsers {
	parsers := Grammar().Compile(nil)

	r := parser.NewScanner(grammarGrammarSrc)
	v, err := parsers.Parse("grammar", r)
	if err != nil {
		panic(err)
	}
	coreNode := v.(parser.Node)

	newGrammarGrammar := NewFromNode(coreNode)

	a := Grammar()
	b := newGrammarGrammar
	if diff := diff.DiffGrammars(a, b); !diff.Equal() {
		panic(fmt.Errorf(
			"mismatch between parsed and hand-crafted core grammar"+
				"\nold: %v"+
				"\nnew: %v"+
				"\ndiff: %#v",
			a, b, diff,
		))
	}
	return newGrammarGrammar.Compile(&coreNode)
}()

func Core() parser.Parsers {
	return core
}
