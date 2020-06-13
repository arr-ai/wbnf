package wbnf

import (
	"testing"

	"github.com/arr-ai/frozen"
	"github.com/stretchr/testify/assert"
)

type rtd struct {
	name, grammar, rule string
	dangers             []string
}

func TestRecursionTermDangerNodes(t *testing.T) {
	for _, test := range []rtd{
		{name: "trivial", rule: "a", grammar: "a -> a;", dangers: []string{"a"}},
		{name: "trivial safe", rule: "a", grammar: "a -> '@' a;", dangers: []string{}},
		{name: "opt preceding", rule: "a", grammar: "a -> '@'? a;", dangers: []string{"a"}},
		{name: "trivial unsafe opt", rule: "a", grammar: "a -> '@' | a;", dangers: []string{"a"}},
		{name: "unsafe opt", rule: "a", grammar: "a -> '@'+ | a? | d | e:':'? | f{0,1};",
			dangers: []string{"a", "d", "f"}},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			node, err := ParseString(test.grammar)
			assert.NoError(t, err)
			assert.NotNil(t, node.Node)
			WalkerOps{EnterProdNode: func(node ProdNode) Stopper {
				if node.OneIdent().String() == test.rule {
					dangers := frozen.NewSet()
					for _, t := range node.AllTerm() {
						dangers = dangers.Union(getSequenceDangerTerms(t))
					}
					expected := frozen.NewSetFromStrings(test.dangers...)

					assert.True(t, dangers.EqualSet(expected),
						"expect:%s actual:%s", expected.Elements(), dangers.Elements())
					return Aborter
				}
				return nil
			}}.Walk(node)
		})
	}
}

func TestGnodeWalkRule(t *testing.T) {
	rules := map[string]frozen.Set{
		"a": frozen.NewSetFromStrings("b", "c"),
		"b": frozen.NewSetFromStrings("c"),
		"c": frozen.NewSetFromStrings("a", "d"),
	}

	gn := &gnode{next: map[string]*gnode{}}
	for k := range rules {
		gn.walkRule(k, &rules)
	}

	err := findPaths("", gn, frozen.NewSet(), nil)
	assert.NotNil(t, err)
}

func TestCheckForRecursion(t *testing.T) {
	for _, test := range []testData{
		{"simple", "a -> 'a';", NoError},
		{"simple", "a -> a;", PossibleCycleDetected},
		{"harder", "a -> ('a'? | b); b -> c; c-> a;", PossibleCycleDetected},
	} {
		test := test
		t.Run("TestValidationErrors-"+test.name, func(t *testing.T) {
			node, err := ParseString(test.grammar)
			assert.NoError(t, err)
			assert.NotNil(t, node.Node)

			err = checkForRecursion(node)
			if test.ekind == NoError {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
