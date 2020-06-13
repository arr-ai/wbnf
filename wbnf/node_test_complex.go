package wbnf

import (
	"fmt"
	"testing"

	"github.com/arr-ai/wbnf/ast"

	"github.com/arr-ai/wbnf/parser"
	"github.com/stretchr/testify/assert"
)

func TestParserNodeToNode(t *testing.T) {
	p := Core()
	v := p.MustParse("grammar", parser.NewScanner(`expr -> @:op="+" > @:op="*" > \d+;`)).(parser.Node)
	g := p.Grammar()
	n := ast.FromParserNode(g, v)
	u := ast.ToParserNode(g, n).(parser.Node)
	parser.AssertEqualNodes(t, v, u)

	p = NewFromAst(n).Compile(u)
	v = p.MustParse(parser.Rule("expr"), parser.NewScanner(`1+2*3`)).(parser.Node)
	g = p.Grammar()
	n = ast.FromParserNode(g, v)
	u = ast.ToParserNode(g, n).(parser.Node)
	parser.AssertEqualNodes(t, v, u)
}

func TestTinyXMLGrammar(t *testing.T) {
	t.Parallel()

	v, err := Core().Parse("grammar", parser.NewScanner(`
		xml  -> s "<" s NAME attr* s ">" xml* "</" s NAME s ">" | CDATA=[^<]+;
		attr -> s NAME s "=" s value=/{"[^"]*"};
		NAME -> [A-Za-z_:][-A-Za-z0-9._:]*;
		s    -> \s*;
	`))
	assert.NoError(t, err)

	node := v.(parser.Node)
	xmlParser := NewFromAst(ast.FromParserNode(Core().Grammar(), node)).Compile(&node)

	src := parser.NewScanner(`<a x="1">hello <b>world!</b></a>`)
	orig := *src
	s := func(offset int, expected string) ast.Leaf {
		end := offset + len(expected)
		slice := orig.Slice(offset, end).String()
		if slice != expected {
			panic(fmt.Errorf("expecting %q, got %q", expected, slice))
		}
		return ast.Leaf(*orig.Slice(offset, end))
	}

	xml, err := xmlParser.Parse(parser.Rule("xml"), src)
	assert.NoError(t, err)

	a := ast.FromParserNode(xmlParser.Grammar(), xml)

	assert.EqualValues(t,
		ast.Branch{
			ast.RuleTag:   ast.One{Node: ast.Extra{Data: parser.Rule("xml")}},
			ast.ChoiceTag: ast.Many{ast.Extra{Data: 0}},
			"":            ast.Many{s(0, `<`), s(8, `>`), s(28, `</`), s(31, `>`)},
			"s":           ast.Many{s(0, ``), s(1, ``), s(8, ``), s(30, ``), s(31, ``)},
			"NAME":        ast.Many{s(1, `a`), s(30, `a`)},
			"attr": ast.Many{ast.Branch{
				"":      ast.One{Node: s(4, `=`)},
				"NAME":  ast.One{Node: s(3, `x`)},
				"s":     ast.Many{s(2, ` `), s(4, ``), s(5, ``)},
				"value": ast.One{Node: s(5, `"1"`)},
			}},
			"xml": ast.Many{
				ast.Branch{
					ast.ChoiceTag: ast.Many{ast.Extra{Data: 1}},
					"CDATA":       ast.One{Node: s(9, `hello `)},
				},
				ast.Branch{
					ast.ChoiceTag: ast.Many{ast.Extra{Data: 0}},
					"":            ast.Many{s(15, `<`), s(17, `>`), s(24, `</`), s(27, `>`)},
					"s":           ast.Many{s(15, ``), s(16, ``), s(17, ``), s(26, ``), s(27, ``)},
					"NAME":        ast.Many{s(16, `b`), s(26, `b`)},
					"xml": ast.Many{
						ast.Branch{
							ast.ChoiceTag: ast.Many{ast.Extra{Data: 1}},
							"CDATA":       ast.One{Node: s(18, `world!`)},
						},
					},
				},
			},
		},
		a,
	)
}
