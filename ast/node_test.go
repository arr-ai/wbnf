package ast

import (
	"fmt"
	"testing"

	"github.com/arr-ai/wbnf/parser"
	"github.com/arr-ai/wbnf/wbnf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertNodeParsesAs(t *testing.T, expected Node, grammar, rule, input string) bool { //nolint:unparam
	parsers, err := wbnf.Compile(grammar)
	require.NoError(t, err)

	src := parser.NewScanner(input)
	orig := *src

	node, err := parsers.Parse(wbnf.Rule(rule), src)
	require.NoError(t, err)
	require.Empty(t, src.String())

	expected = resolveSrc(expected, orig)
	if b, ok := expected.(Branch); ok {
		b[RuleTag] = One{Extra{wbnf.Rule(rule)}}
	}
	var ast Branch
	ast = ParserNodeToNode(parsers.Grammar(), node)
	return assert.EqualValues(t, expected, ast)
}

func resolveSrc(node Node, src parser.Scanner) Node {
	switch n := node.(type) {
	case Branch:
		result := make(Branch, len(n))
		for name, children := range n {
			switch c := children.(type) {
			case Many:
				array := make(Many, 0, len(c))
				for _, child := range c {
					array = append(array, resolveSrc(child, src))
				}
				result[name] = array
			case One:
				result[name] = One{resolveSrc(c.Node, src)}
			}
		}
		return result
	case Leaf:
		scanner := n.Scanner()
		offset := scanner.Offset()
		expected := scanner.String()
		end := offset + len(expected)
		slice := src.Slice(offset, end).String()
		if slice != expected {
			panic(fmt.Errorf("expecting %q, got %q", expected, slice))
		}
		return Leaf(*src.Slice(offset, end))
	default:
		return n
	}
}

func slicePlaceholder(offset int, expected string) One {
	return One{Leaf(*parser.NewBareScanner(offset, expected))}
}

func slicesPlaceholder(offset int, expected ...interface{}) Many {
	many := make(Many, 0, len(expected))
	for _, e := range expected {
		switch e := e.(type) {
		case string:
			many = append(many, Leaf(*parser.NewBareScanner(offset, e)))
			offset += len(e)
		case int:
			offset += e
		default:
			panic("wat?")
		}
	}
	return many
}

func TestNodeOneRule(t *testing.T) {
	t.Parallel()

	one := slicePlaceholder
	many := slicesPlaceholder

	for i, s := range []struct {
		expected             Node
		grammar, rule, input string
	}{
		// {Branch{"": one(0, `1`)} /*                      */, `a -> "1";      `, `a`, `1`}, // 0
		{Branch{} /*                                     */, `a -> "1"?;     `, `a`, ``},
		{Branch{"": one(0, `1`)} /*                      */, `a -> "1"?;     `, `a`, `1`},

		{Branch{} /*                                     */, `a -> "1"*;     `, `a`, ``}, // 3
		{Branch{"": many(0, `1`)} /*                     */, `a -> "1"*;     `, `a`, `1`},
		{Branch{"": many(0, `1`, `1`)} /*                */, `a -> "1"*;     `, `a`, `11`},

		{Branch{"": many(0, `1`, `2`)} /*                */, `a -> "1" "2";  `, `a`, `12`}, // 6
		{Branch{"": many(0, `1`)} /*                     */, `a -> "1" "2"?; `, `a`, `1`},
		{Branch{"": many(0, `1`, `2`, `2`)} /*           */, `a -> "1" "2"*; `, `a`, `122`},

		{Branch{} /*                                     */, `a -> "1"? "2"?;`, `a`, ``}, // 9
		{Branch{"": many(0, `1`)} /*                     */, `a -> "1"? "2"?;`, `a`, `1`},
		{Branch{"": many(0, `2`)} /*                     */, `a -> "1"? "2"?;`, `a`, `2`},
		{Branch{"": many(0, `1`, `2`)} /*                */, `a -> "1"? "2"?;`, `a`, `12`},

		{Branch{} /*                                     */, `a -> "1"* "2"*;`, `a`, ``}, // 13
		{Branch{"": many(0, `1`, `1`, `2`, `2`)} /*      */, `a -> "1"* "2"*;`, `a`, `1122`},

		{Branch{"": many(0, `1`)} /*                     */, `a -> "1":"2";  `, `a`, `1`}, // 15
		{Branch{"": many(0, `1`, `2`, `1`)} /*           */, `a -> "1":"2";  `, `a`, `121`},
		{Branch{"": many(0, `1`, `2`, `1`, `2`, `1`)} /* */, `a -> "1":"2";  `, `a`, `12121`},
	} {
		s := s
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()
			assertNodeParsesAs(t, s.expected, s.grammar, s.rule, s.input)
		})
	}
}

func TestNodeOneRuleWithNames(t *testing.T) {
	t.Parallel()

	// one := slicePlaceholder
	many := slicesPlaceholder

	for i, s := range []struct {
		expected             Node
		grammar, rule, input string
	}{
		{Branch{"x": many(0, `1`, `1`), "y": many(2, `2`, `2`)}, // 18
			`a -> x="1"* y="2"*;`, `a`, `1122`},
		{
			Branch{
				"@choice": Many{Extra{0}, Extra{1}, Extra{0}, Extra{1}},
				"x":       many(0, `1`, 1, `1`),
				"y":       many(1, `2`, 1, `2`),
			},
			`a -> (x="1"|y="2")*;`, `a`, `1212`},
		{
			Branch{
				"x": many(0, `1`, 1, `1`, 1, `1`),
				"y": many(1, `2`, 1, `2`),
			},
			`a -> (x="1"):y="2";`, `a`, `12121`},
	} {
		s := s
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()
			assertNodeParsesAs(t, s.expected, s.grammar, s.rule, s.input)
		})
	}
}

func TestNodeStack(t *testing.T) {
	t.Parallel()

	// one := slicePlaceholder
	many := slicesPlaceholder

	for i, s := range []struct {
		expected             Node
		grammar, rule, input string
	}{
		{Branch{"a": many(0, `1`)}, `a -> @:"+" > "1";`, `a`, `1`},

		// {Branch{"a": many(0, `1`, 1, `1`)},
		// 	`a -> @:op="+" > @:op="*" > "1";`, `a`, `1+1`},

		// {Branch{"": Many{many(0, `1`)}},
		// 	`a -> @:op="+" > @:op="*" > "1";`, `a`, `1+1`},
	} {
		s := s
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()
			assertNodeParsesAs(t, s.expected, s.grammar, s.rule, s.input)
		})
	}
}
