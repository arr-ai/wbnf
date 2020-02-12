package parser

type externalRef string

type External func(term Term, elt TreeElement, input *Scanner) (ast TreeElement, subgrammar Node, _ error)
type Externals map[string]External

type SubGrammar struct {
	Node
}

func (SubGrammar) IsExtra() {}
