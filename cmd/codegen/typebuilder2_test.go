package codegen

import (
	"testing"

	"github.com/arr-ai/wbnf/wbnf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func initTypeBuilderTest(t *testing.T, grammar string) map[string]grammarType {
	tree, err := wbnf.ParseString(grammar)
	require.NoError(t, err)

	return MakeTypesFromGrammar(wbnf.NewFromAst(tree))
}

func TestTypeBuilder2_RuleWithOnlyUnnamedTokenVal(t *testing.T) {
	types := initTypeBuilderTest(t, "a -> 'hello';")

	assert.NotEmpty(t, types)
	assert.IsType(t, basicRule(""), types["ANode"])
}

func TestTypeBuilder2_ScopedGrammarRules(t *testing.T) {
	types := initTypeBuilderTest(t, `a -> foo { foo -> "a";};`)

	assert.NotEmpty(t, types)
	assert.IsType(t, rule{}, types["ANode"])
	assert.IsType(t, basicRule(""), types["AFooNode"])
}

func TestTypeBuilder2_Delims(t *testing.T) {
	types := initTypeBuilderTest(t, "a -> 'hello':op=',';"+
		"b -> a:',';"+"c->'a':a;")

	assert.NotEmpty(t, types)
	assert.IsType(t, rule{}, types["ANode"])
	assert.IsType(t, basicRule(""), types["AOpNode"])
	assert.IsType(t, rule{}, types["BNode"])
	assert.IsType(t, rule{}, types["BDelimNode"])
	assert.IsType(t, rule{}, types["CNode"])
	assert.IsType(t, rule{}, types["CANode"])
}

func TestTypeBuilder2_RuleWithQuant(t *testing.T) {
	types := initTypeBuilderTest(t, "a -> a? ;")

	assert.NotEmpty(t, types)
	assert.IsType(t, rule{}, types["ANode"])
	testChildren(t, types["ANode"].Children(), childrenTestData{
		"a": {t: namedRule{}, quant: wantOneGetter, returnType: "ANode"},
	})
}
