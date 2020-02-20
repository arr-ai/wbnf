package codegen

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/arr-ai/wbnf/wbnf"
	"github.com/stretchr/testify/require"
)

func initTest(t *testing.T, grammar string, prodname string) []wbnf.TermNode {
	tree, err := wbnf.ParseString(grammar)
	require.NoError(t, err)

	var out []wbnf.TermNode
	wbnf.WalkerOps{EnterProdNode: func(node wbnf.ProdNode) wbnf.Stopper {
		if node.OneIdent().String() == prodname {
			out = node.AllTerm()
			return wbnf.Aborter
		}
		return wbnf.NodeExiter
	}}.Walk(tree)

	return out
}

func findChild(needle string, haystack []grammarType) grammarType {
	for _, child := range haystack {
		if child.Ident() == needle {
			return child
		}
	}
	return nil
}

type childrenTestData map[string]struct {
	t          interface{}
	quant      int
	returnType string
}

func testChildren(t *testing.T, children []grammarType, tests childrenTestData) {
	assert.Len(t, children, len(tests))
	for name, val := range tests {
		errMessage := fmt.Sprintf("Looking up ident: %s", name)
		child := findChild(name, children)
		assert.NotNil(t, child, errMessage)
		if name == "@choice" {
			assert.IsType(t, choice(""), child, errMessage)
			return
		}
		assert.IsType(t, val.t, child, errMessage)
		switch x := child.(type) {
		case namedRule:
			assert.Equal(t, val.quant, x.count, errMessage)
			assert.Equal(t, val.returnType, x.returnType, errMessage)
		case namedToken:
			assert.Equal(t, val.quant, x.count, errMessage)

		}
	}
}

func TestTypeBuilder_RuleWithOnlyUnnamedTokenVal(t *testing.T) {
	terms := initTest(t, "a -> 'hello';", "a")

	types := MakeTypesForTerms("a", "a", terms[0])
	assert.NotEmpty(t, types)
	assert.IsType(t, basicRule(""), types["ANode"])
}

func TestTypeBuilder_RulePointingToOnlyUnnamedTokenVal(t *testing.T) {
	terms := initTest(t, "a -> b; b -> 'hello';", "a")

	types := MakeTypesForTerms("a", "a", terms[0])
	assert.NotEmpty(t, types)
	assert.IsType(t, rule{}, types["ANode"])
	testChildren(t, types["ANode"].Children(), childrenTestData{
		"b": {t: namedRule{}, quant: wantOneGetter, returnType: "BNode"},
	})
}

func TestTypeBuilder_RuleWithQuant(t *testing.T) {
	terms := initTest(t, "a -> a? ;", "a")

	types := MakeTypesForTerms("a", "a", terms[0])
	assert.NotEmpty(t, types)
	assert.IsType(t, rule{}, types["ANode"])
	testChildren(t, types["ANode"].Children(), childrenTestData{
		"a": {t: namedRule{}, quant: wantOneGetter, returnType: "ANode"},
	})
}

func TestTypeBuilder_RuleWithOnlyNamedTokenVal(t *testing.T) {
	terms := initTest(t, "a -> val='hello';", "a")

	types := MakeTypesForTerms("a", "a", terms[0])
	assert.NotEmpty(t, types)
	assert.IsType(t, rule{}, types["ANode"])
	testChildren(t, types["ANode"].Children(), childrenTestData{
		"val": {t: namedToken{}, quant: wantOneGetter},
	})
}

func TestTypeBuilder_RuleWithUnnamedRuleVal(t *testing.T) {
	terms := initTest(t, "a -> x; x -> 'a''b';", "a")

	types := MakeTypesForTerms("a", "a", terms[0])
	assert.NotEmpty(t, types)
	assert.IsType(t, rule{}, types["ANode"])
	testChildren(t, types["ANode"].Children(), childrenTestData{
		"x": {t: namedRule{}, quant: wantOneGetter, returnType: "XNode"},
	})
}

func TestTypeBuilder_RuleWithNamedRuleVal(t *testing.T) {
	terms := initTest(t, "a -> val=x; x -> 'a''b';", "a")

	types := MakeTypesForTerms("a", "a", terms[0])
	assert.NotEmpty(t, types)
	assert.IsType(t, rule{}, types["ANode"])

	testChildren(t, types["ANode"].Children(), childrenTestData{
		"val": {t: namedRule{}, quant: wantOneGetter, returnType: "XNode"},
	})
}

func TestTypeBuilder_RuleWithUnnamedTerm(t *testing.T) {
	terms := initTest(t, "a -> 'a' /{[123]*};", "a")

	types := MakeTypesForTerms("a", "a", terms[0])
	assert.NotEmpty(t, types)
	assert.IsType(t, rule{}, types["ANode"])

	testChildren(t, types["ANode"].Children(), childrenTestData{
		"Token": {t: unnamedToken{}, quant: wantAllGetter},
	})
}

func TestTypeBuilder_RuleWithUnnamedChoiceTerm(t *testing.T) {
	terms := initTest(t, "a -> 'a' | /{[123]*};", "a")

	types := MakeTypesForTerms("a", "a", terms[0])
	assert.NotEmpty(t, types)
	assert.IsType(t, rule{}, types["ANode"])

	testChildren(t, types["ANode"].Children(), childrenTestData{
		"@choice": {},
		"Token":   {t: unnamedToken{}, quant: wantAllGetter},
	})
}

func TestTypeBuilder_RuleWithNamedTerm(t *testing.T) {
	terms := initTest(t, "a -> val=('a' /{[123]*});", "a")

	types := MakeTypesForTerms("a", "a", terms[0])
	assert.NotEmpty(t, types)
	assert.IsType(t, rule{}, types["ANode"])

	testChildren(t, types["ANode"].Children(), childrenTestData{
		"val": {t: namedRule{}, quant: wantOneGetter, returnType: "AValNode"},
	})
	testChildren(t, types["AValNode"].Children(), childrenTestData{
		"Token": {t: unnamedToken{}, quant: wantAllGetter},
	})
}

func TestTypeBuilder_RuleWithStack(t *testing.T) {
	terms := initTest(t, "a -> @ \\s? > 'hello';", "a")

	types := MakeTypesForTerms("a", "a", terms[0])
	assert.NotEmpty(t, types)
	assert.IsType(t, rule{}, types["ANode"])

	testChildren(t, types["ANode"].Children(), childrenTestData{
		"a":     {t: namedRule{}, quant: wantOneGetter, returnType: "ANode"},
		"Token": {t: unnamedToken{}, quant: wantAllGetter, returnType: "ANode"},
	})
}

func TestTypeBuilder_RuleWithStackComplicated(t *testing.T) {
	terms := initTest(t, `term    -> @:op=">"
         > @:op="|"
         > @+
         > named quant*;
		named -> [a-z]*; quant -> [*?+];`, "term")

	types := MakeTypesForTerms("term", "term", terms[0])
	assert.NotEmpty(t, types)
	assert.IsType(t, rule{}, types["TermNode"])

	testChildren(t, types["TermNode"].Children(), childrenTestData{
		"term":  {t: namedRule{}, quant: wantAllGetter, returnType: "TermNode"},
		"op":    {t: namedToken{}, quant: wantAllGetter},
		"named": {t: namedRule{}, quant: wantOneGetter, returnType: "NamedNode"},
		"quant": {t: namedRule{}, quant: wantAllGetter, returnType: "QuantNode"},
	})
}

func TestTypeBuilder_ScopedGrammarRules(t *testing.T) {
	terms := initTest(t, `a -> foo { foo -> "a" | "b";};`, "a")

	types := MakeTypesForTerms("a", "a", terms[0])
	assert.NotEmpty(t, types)
	assert.IsType(t, rule{}, types["ANode"])

}
