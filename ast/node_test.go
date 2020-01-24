package ast

import (
	"fmt"
	"testing"

	"github.com/arr-ai/wbnf/wbnf"
	"github.com/arr-ai/wbnf/parser"
	"github.com/stretchr/testify/assert"
)

func TestParserNodeToNode(t *testing.T) {
	p := wbnf.Core()
	v := p.MustParse(wbnf.GrammarRule, parser.NewScanner(`expr -> @:op="+" ^ @:op="*" ^ /{\d+};`)).(parser.Node)
	g := p.Grammar()
	n := ParserNodeToNode(g, v)
	u := NodeToParserNode(g, n).(parser.Node)
	parser.AssertEqualNodes(t, v, u)

	p = wbnf.NewFromNode(v).Compile()
	v = p.MustParse(wbnf.Rule("expr"), parser.NewScanner(`1+2*3`)).(parser.Node)
	g = p.Grammar()
	n = ParserNodeToNode(g, v)
	u = NodeToParserNode(g, n).(parser.Node)
	parser.AssertEqualNodes(t, v, u)
}

func TestTinyXmlGrammar(t *testing.T) {
	t.Parallel()

	v, err := wbnf.Core().Parse(wbnf.GrammarRule, parser.NewScanner(`
		xml  -> s "<" s NAME attr* s ">" xml* "</" s NAME s ">" | CDATA=/{[^<]+};
		attr -> s NAME s "=" s value=/{"[^"]*"};
		NAME -> /{[A-Za-z_:][-A-Za-z0-9._:]*};
		s    -> /{\s*};
	`))
	assert.NoError(t, err)

	xmlParser := wbnf.NewFromNode(v.(parser.Node)).Compile()

	src := parser.NewScanner(`<a x="1">hello <b>world!</b></a>`)
	orig := *src
	s := func(offset int, expected string) Leaf {
		end := offset + len(expected)
		slice := orig.Slice(offset, end).String()
		if slice != expected {
			panic(fmt.Errorf("expecting %q, got %q", expected, slice))
		}
		return Leaf(*orig.Slice(offset, end))
	}

	xml, err := xmlParser.Parse(wbnf.Rule("xml"), src)
	assert.NoError(t, err)

	ast := ParserNodeToNode(xmlParser.Grammar(), xml)

	assert.EqualValues(t,
		Branch{
			"@rule":   One{Extra{wbnf.Rule("xml")}},
			"@choice": Many{Extra{0}},
			"":        Many{s(0, `<`), s(8, `>`), s(28, `</`), s(31, `>`)},
			"s":       Many{s(0, ``), s(1, ``), s(8, ``), s(30, ``), s(31, ``)},
			"NAME":    Many{s(1, `a`), s(30, `a`)},
			"attr": Many{Branch{
				"":      One{s(4, `=`)},
				"NAME":  One{s(3, `x`)},
				"s":     Many{s(2, ` `), s(4, ``), s(5, ``)},
				"value": One{s(5, `"1"`)},
			}},
			"xml": Many{
				Branch{
					"@choice": Many{Extra{1}},
					"CDATA":   One{s(9, `hello `)},
				},
				Branch{
					"@choice": Many{Extra{0}},
					"":        Many{s(15, `<`), s(17, `>`), s(24, `</`), s(27, `>`)},
					"s":       Many{s(15, ``), s(16, ``), s(17, ``), s(26, ``), s(27, ``)},
					"NAME":    Many{s(16, `b`), s(26, `b`)},
					"xml": Many{
						Branch{
							"@choice": Many{Extra{1}},
							"CDATA":   One{s(18, `world!`)},
						},
					},
				},
			},
		},
		ast,
	)
}
