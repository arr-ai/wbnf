package parser

type externalRef string

type External func(elt TreeElement, input *Scanner, end bool) (ast TreeElement, subgrammar Node, _ error)
type Externals map[string]External

type SubGrammar struct {
	Node
}

func (SubGrammar) IsExtra() {}
