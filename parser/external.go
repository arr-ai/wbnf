package parser

type externalRef string

type External func(scope Scope, input *Scanner, end bool) (ast TreeElement, subgrammar Node, _ error)
type Externals map[string]External

type SubGrammar struct {
	Node
}

func (SubGrammar) IsExtra() {}
