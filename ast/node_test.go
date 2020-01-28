package ast

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/arr-ai/wbnf/parser"
	"github.com/arr-ai/wbnf/wbnf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type nodeParseScenario struct {
	expected, grammar, rule, input string
}

func (s nodeParseScenario) String() string {
	return fmt.Sprintf("%s/%s",
		strings.TrimRight(s.grammar, " "),
		strings.TrimRight(s.input, " "))
}

var endIndentRE = regexp.MustCompile(`(\()\n *|,\n *(\))|(,)\n( ) *`)

func assertNodeParsesAs(t *testing.T, s nodeParseScenario) bool { //nolint:unparam
	parsers, err := wbnf.Compile(s.grammar)
	require.NoError(t, err)

	src := parser.NewScanner(strings.TrimRight(s.input, " "))

	node, err := parsers.Parse(wbnf.Rule(s.rule), src)
	require.NoError(t, err)
	require.Empty(t, src.String())

	ast := ParserNodeToNode(parsers.Grammar(), node)
	if assert.Equal(t, wbnf.Rule(s.rule), ast[RuleTag].(One).Node.(Extra).Data) {
		delete(ast, RuleTag)
	}
	return assert.Equal(t,
		strings.TrimRight(s.expected, " "),
		endIndentRE.ReplaceAllString(ast.String(), "$1$2$3$4"))
}

func TestNodeOneRule(t *testing.T) {
	t.Parallel()

	for _, s := range []nodeParseScenario{
		// {`('': 0‣1)                      `, `a -> "1";       `, `a`, `1    `},
		{`()                             `, `a -> "1"?;      `, `a`, `     `},
		{`('': 0‣1)                      `, `a -> "1"?;      `, `a`, `1    `},

		{`()                             `, `a -> "1"*;      `, `a`, `     `},
		{`('': [0‣1])                    `, `a -> "1"*;      `, `a`, `1    `},
		{`('': [0‣1, 1‣1])               `, `a -> "1"*;      `, `a`, `11   `},

		{`('': [0‣1, 1‣2])               `, `a -> "1" "2";   `, `a`, `12   `},
		{`('': [0‣1])                    `, `a -> "1" "2"?;  `, `a`, `1    `},
		{`('': [0‣1, 1‣2, 2‣2])          `, `a -> "1" "2"*;  `, `a`, `122  `},

		{`()                             `, `a -> "1"? "2"?; `, `a`, `     `},
		{`('': [0‣1])                    `, `a -> "1"? "2"?; `, `a`, `1    `},
		{`('': [0‣2])                    `, `a -> "1"? "2"?; `, `a`, `2    `},
		{`('': [0‣1, 1‣2])               `, `a -> "1"? "2"?; `, `a`, `12   `},

		{`()                             `, `a -> "1"* "2"*; `, `a`, `     `},
		{`('': [0‣1, 1‣1, 2‣2, 3‣2])     `, `a -> "1"* "2"*; `, `a`, `1122 `},

		{`('': [0‣1])                    `, `a -> "1":"2";   `, `a`, `1    `},
		{`('': [0‣1, 1‣2, 2‣1])          `, `a -> "1":"2";   `, `a`, `121  `},
		{`('': [0‣1, 1‣2, 2‣1, 3‣2, 4‣1])`, `a -> "1":"2";   `, `a`, `12121`},
	} {
		s := s
		t.Run(s.String(), func(t *testing.T) {
			t.Parallel()
			assertNodeParsesAs(t, s)
		})
	}
}

func TestNodeOneRuleWithNames(t *testing.T) {
	t.Parallel()

	for _, s := range []nodeParseScenario{
		{`(x: [0‣1, 1‣1], y: [2‣2, 3‣2])                       `, `a -> x="1"* y="2"*; `, `a`, `1122`},
		{`(@choice: [0, 1, 0, 1], x: [0‣1, 2‣1], y: [1‣2, 3‣2])`, `a -> (x="1"|y="2")*;`, `a`, `1212`},
		{`(@choice: [1, 0, 1, 0], x: [1‣1, 3‣1], y: [0‣2, 2‣2])`, `a -> (x="1"|y="2")*;`, `a`, `2121`},
		{`(x: [0‣1, 2‣1, 4‣1], y: [1‣2, 3‣2])                  `, `a -> x="1":y="2";   `, `a`, `12121`},
	} {
		s := s
		t.Run(s.String(), func(t *testing.T) {
			t.Parallel()
			assertNodeParsesAs(t, s)
		})
	}
}

func TestNodeStack(t *testing.T) {
	t.Parallel()

	for _, s := range []nodeParseScenario{
		{`(a: [0‣1])`, `a -> @:"+" > "1";`, `a`, `1`},

		// {Branch{"a": many(0, `1`, 1, `1`)},
		// 	`a -> @:op="+" > @:op="*" > "1";`, `a`, `1+1`},

		// {`('': Many{many(0, `1`)}},
		// 	`a -> @:op="+" > @:op="*" > "1";`, `a`, `1+1`},
	} {
		s := s
		t.Run(s.String(), func(t *testing.T) {
			t.Parallel()
			assertNodeParsesAs(t, s)
		})
	}
}

func TestNodeInnerOuterName(t *testing.T) {
	t.Parallel()

	for _, s := range []nodeParseScenario{
		{`('': [0‣"(", 2‣")"], sum: ('': [1‣1]))`, `a -> "(" sum=("1":"+") ")";`, `a`, `(1)`},
		{`('': [0‣"(", 4‣")"], sum: ('': [1‣1, 2‣+, 3‣1]))`, `a -> "(" sum=("1":"+") ")";`, `a`, `(1+1)`},

		// {Branch{"a": many(0, `1`, 1, `1`)},
		// 	`a -> @:op="+" > @:op="*" > "1";`, `a`, `1+1`},

		// {`('': Many{many(0, `1`)}},
		// 	`a -> @:op="+" > @:op="*" > "1";`, `a`, `1+1`},
	} {
		s := s
		t.Run(s.String(), func(t *testing.T) {
			t.Parallel()
			assertNodeParsesAs(t, s)
		})
	}
}
