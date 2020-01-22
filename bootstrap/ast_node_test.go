package bootstrap

import (
	"testing"

	"github.com/arr-ai/wbnf/parser"
)

func TestParserNodeToASTNode(t *testing.T) {
	p := Core()
	v := p.MustParse(grammarR, parser.NewScanner(`x -> x:op="+" ^ x:op="*" ^ /{\d+};`)).(parser.Node)
	g := p.Grammar()
	n := ParserNodeToASTNode(g, v)
	u := ASTNodeToParserNode(g, n).(parser.Node)
	assertEqualNodes(t, v, u)

	p = NewFromNode(v).Compile()
	v = p.MustParse(Rule("x"), parser.NewScanner(`1+2*3`)).(parser.Node)
	g = p.Grammar()
	n = ParserNodeToASTNode(g, v)
	u = ASTNodeToParserNode(g, n).(parser.Node)
	assertEqualNodes(t, v, u)
}
