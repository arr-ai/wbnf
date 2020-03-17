package wbnf

import (
	"fmt"
	"github.com/arr-ai/wbnf/parse"
	"regexp"
	"strings"
	"testing"

	ast2 "github.com/arr-ai/wbnf/ast"
	"github.com/arr-ai/wbnf/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type nodeParseScenario struct {
	expected, grammar, rule, input string
	reversible                     bool
}

func (s nodeParseScenario) String() string {
	return fmt.Sprintf("%s/%s",
		strings.TrimRight(s.grammar, " "),
		strings.TrimRight(s.input, " "))
}

var endIndentRE = regexp.MustCompile(`([([])\n *|,\n *([)\]])|(,)\n( ) *`)

func assertNodeParsesAs(
	t *testing.T,
	expected, grammar, rule, input string,
	reversible bool,
) bool {
	return assertNodeParsesAsScenario(t, nodeParseScenario{expected, grammar, rule, input, reversible})
}

func assertNodeParsesAsScenario(t *testing.T, s nodeParseScenario) bool { //nolint:unparam
	p, err := Compile(s.grammar, nil)
	g := p.Grammar()
	require.NoError(t, err)

	src := parse.NewScanner(strings.TrimRight(s.input, " "))

	node, err := p.Parse(parser.Rule(s.rule), src)
	require.NoError(t, err)
	require.Empty(t, src.String())
	// log.Print(node)

	ast := ast2.FromParserNode(g, node)
	// log.Print(ast)
	reversalOK := true
	if s.reversible {
		node2 := ast2.ToParserNode(g, ast)
		// log.Print(node2)
		ok := parser.AssertEqualNodes(t, node.(parser.Node), node2.(parser.Node))
		if !ok {
			t.Error(s)
			ast2.ToParserNode(g, ast)
		}
	}
	if assert.Equal(t, parser.Rule(s.rule), ast[ast2.RuleTag].(ast2.One).Node.(ast2.Extra).Data) {
		delete(ast, ast2.RuleTag)
	}
	expectedOK := s.expected == "" || assert.Equal(t,
		strings.TrimRight(s.expected, " "),
		endIndentRE.ReplaceAllString(ast.String(), "$1$2$3$4"),
		"%v", s,
	)
	return reversalOK && expectedOK
}

func assertNodeFailsToParse(
	t *testing.T,
	grammar, rule, input string,
) bool {
	return assertNodeFailsToParseScenario(t, nodeParseScenario{``, grammar, rule, input, false})
}

func assertNodeFailsToParseScenario(t *testing.T, s nodeParseScenario) bool { //nolint:unparam
	p, err := Compile(s.grammar, nil)
	require.NoError(t, err)

	src := parse.NewScanner(strings.TrimRight(s.input, " "))

	node, err := p.Parse(parser.Rule(s.rule), src)
	return assert.Error(t, err, "%v", node)
}

func TestNodeOneRule(t *testing.T) {
	t.Parallel()

	for _, s := range []nodeParseScenario{
		// {`('': 0‣1)                      `, `a -> "1";       `, `a`, `1    `, true},
		// {`()                             `, `a -> "1"?;      `, `a`, `     `, true},
		// {`('': 0‣1)                      `, `a -> "1"?;      `, `a`, `1    `, true},

		{`()                             `, `a -> "1"*;      `, `a`, `     `, true},
		{`('': [0‣1])                    `, `a -> "1"*;      `, `a`, `1    `, true},
		{`('': [0‣1, 1‣1])               `, `a -> "1"*;      `, `a`, `11   `, true},

		{`('': [0‣1, 1‣2])               `, `a -> "1" "2";   `, `a`, `12   `, true},
		{`('': [0‣1])                    `, `a -> "1" "2"?;  `, `a`, `1    `, true},
		{`('': [0‣1, 1‣2, 2‣2])          `, `a -> "1" "2"*;  `, `a`, `122  `, true},

		{`()                             `, `a -> "1"* "2"*; `, `a`, `     `, true},

		// Reversibility needs reparsing of quant children.
		{`()                             `, `a -> "1"? "2"?; `, `a`, `     `, false},
		{`('': [0‣1])                    `, `a -> "1"? "2"?; `, `a`, `1    `, false},
		{`('': [0‣2])                    `, `a -> "1"? "2"?; `, `a`, `2    `, false},
		{`('': [0‣1, 1‣2])               `, `a -> "1"? "2"?; `, `a`, `12   `, false},
		{`('': [0‣1, 1‣1, 2‣2, 3‣2])     `, `a -> "1"* "2"*; `, `a`, `1122 `, false},

		{`('': [0‣1])                    `, `a -> "1":"2";   `, `a`, `1    `, false},
		{`('': [0‣1, 1‣2, 2‣1])          `, `a -> "1":"2";   `, `a`, `121  `, false},
		{`('': [0‣1, 1‣2, 2‣1, 3‣2, 4‣1])`, `a -> "1":"2";   `, `a`, `12121`, false},
	} {
		s := s
		t.Run(s.String(), func(t *testing.T) {
			// assert.NotPanics(t, func() {
			assertNodeParsesAsScenario(t, s)
			// })
		})
	}
}

func TestNodeOneRuleWithNames(t *testing.T) {
	t.Parallel()

	for _, s := range []nodeParseScenario{
		{`(x: ('': 0‣1), y: ('': 1‣2))`,
			`a -> x="1" y="2";`, `a`, `12`, true},
		{`(x: [('': 0‣1), ('': 1‣1)], y: [('': 2‣2), ('': 3‣2)])`,
			`a -> x="1"* y="2"*;`, `a`, `1122`, true},
		{`(@choice: [0, 1, 0, 1], x: [('': 0‣1), ('': 2‣1)], y: [('': 1‣2), ('': 3‣2)])`,
			`a -> (x="1"|y="2")*;`, `a`, `1212`, true},
		{`(@choice: [1, 0, 1, 0], x: [('': 1‣1), ('': 3‣1)], y: [('': 0‣2), ('': 2‣2)])`,
			`a -> (x="1"|y="2")*;`, `a`, `2121`, true},
		{`(x: [('': 0‣1), ('': 2‣1), ('': 4‣1)], y: [('': 1‣2), ('': 3‣2)])`,
			`a -> x="1":y="2";`, `a`, `12121`, true},
	} {
		s := s
		t.Run(s.String(), func(t *testing.T) {
			// assert.NotPanics(t, func() {
			assertNodeParsesAsScenario(t, s)
			// })
		})
	}
}

func TestNodeStack(t *testing.T) {
	t.Parallel()

	exprGrammar2 := `a -> @:op="+" > @:op="*" > \d;`
	exprGrammar3 := `a -> @:op="+" > @:op="*" > @:"**" > \d;`

	for _, s := range []nodeParseScenario{
		{`(a: [('': 0‣1)])`, `a -> @:op="+" > \d;`, `a`, `1`, true},
		{`(a: [('': 0‣4), ('': 2‣5)], op: [('': 1‣+)])`, `a -> @:op="+" > \d;`, `a`, `4+5`, true},
		{`(a: [(a: [('': 0‣5)])])`, exprGrammar2, `a`, `5`, true},
		{`(a: [(a: [('': 0‣6), ('': 2‣7)], op: [('': 1‣*)])])`, exprGrammar2, `a`, `6*7`, true},
		{`(a: [(a: [('': 0‣5)]), (a: [('': 2‣6), ('': 4‣7)], op: [('': 3‣*)])], op: [('': 1‣+)])`,
			exprGrammar2, `a`, `5+6*7`, true},
		{`(a: [(a: [(a: [('': 0‣5)])])])`, exprGrammar3, `a`, `5`, true},
		{`(a: [(a: [(a: [('': 0‣6)]), (a: [('': 2‣7)])], op: [('': 1‣*)])])`, exprGrammar3, `a`, `6*7`, true},
		{`(a: [(a: [(a: [('': 0‣5)])]), (a: [(a: [('': 2‣6)]), (a: [('': 4‣7)])], op: [('': 3‣*)])], op: [('': 1‣+)])`,
			exprGrammar3, `a`, `5+6*7`, true},

		// {`('': Many{many(0, `1`)}},
		// 	`a -> @:op="+" > @:op="*" > "1";`, `a`, `1+1`, true},
	} {
		s := s
		t.Run(s.String(), func(t *testing.T) {
			// assert.NotPanics(t, func() {
			assertNodeParsesAsScenario(t, s)
			// })
		})
	}
}

func TestNodeInnerOuterName(t *testing.T) {
	t.Parallel()

	for _, s := range []nodeParseScenario{
		{`('': [0‣"(", 2‣")"], sum: ('': [1‣1]))`, `a -> "(" sum=("1":"+") ")";`, `a`, `(1)`, true},
		{`('': [0‣"(", 4‣")"], sum: ('': [1‣1, 2‣+, 3‣1]))`, `a -> "(" sum=("1":"+") ")";`, `a`, `(1+1)`, true},
	} {
		s := s
		t.Run(s.String(), func(t *testing.T) {
			t.Parallel()
			assertNodeParsesAsScenario(t, s)
		})
	}
}

func TestNodeCoreSubsets(t *testing.T) {
	t.Parallel()

	assertNodeParsesAs(t, `(a: [(b: ('': 0‣1))])`, `a -> @+ > b; b -> "1";`, "a", `1`, true)
	assertNodeParsesAs(t, `(b: ('': [0‣1]))`, `a -> b; b -> "1"+;`, "a", `1`, true)
	assertNodeParsesAs(t, `(a: [(b: [('': [0‣5])])])`, `a -> @+ > b+; b -> "5"+;`, "a", `5`, true)
}

func TestNodeBacktrack(t *testing.T) {
	t.Parallel()

	assertNodeParsesAs(t, ``, `p -> @ ("->" @)* > @:"-" > "i";`, "p", `i->i`, true)
}

func TestNodeControlledWrapRE(t *testing.T) {
	t.Parallel()

	assertNodeParsesAs(t, ``, `p -> "1" "2";`, "p", `12`, true)

	assertNodeParsesAs(t, ``, `p -> "1" "2"; .wrapRE -> "1" | /{\s*()};`, "p", `12`, true)
	assertNodeParsesAs(t, ``, `p -> "1" "2"; .wrapRE -> "1" | /{\s*()};`, "p", `1 2`, true)
	assertNodeFailsToParse(t, `p -> "1" "2"; .wrapRE -> "1" | /{\s*()};`, "p", ` 12`)

	assertNodeParsesAs(t, ``, `p -> "1" "2"; .wrapRE -> "2" | /{\s*()};`, "p", `12`, true)
	assertNodeParsesAs(t, ``, `p -> "1" "2"; .wrapRE -> "2" | /{\s*()};`, "p", ` 12`, true)
	assertNodeFailsToParse(t, `p -> "1" "2"; .wrapRE -> "2" | /{\s*()};`, "p", `1 2`)
}

func TestNodeCoreGrammarTrivial(t *testing.T) {
	t.Parallel()

	assertNodeParsesAsScenario(t, nodeParseScenario{
		grammar:    grammarGrammarSrc,
		rule:       "term",
		input:      `a`,
		reversible: true,
	})
}

func TestNodeCoreGrammarCoreGrammar(t *testing.T) {
	t.Parallel()

	assertNodeParsesAsScenario(t, nodeParseScenario{
		grammar:    grammarGrammarSrc,
		rule:       "grammar",
		input:      grammarGrammarSrc,
		reversible: true,
	})
}
