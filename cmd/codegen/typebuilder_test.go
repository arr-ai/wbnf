package codegen

import (
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

func TestTypeBuilder_RuleWithOnlyUnnamedTokenVal(t *testing.T) {
	terms := initTest(t, "a -> 'hello';", "a")

	types, deps := MakeTypesForTerms("a", terms[0])
	assert.NotEmpty(t, types)
	assert.Empty(t, deps)
	assert.IsType(t, basicRule(""), types["ANode"])
}

func TestTypeBuilder_RuleWithOnlyNamedTokenVal(t *testing.T) {
	terms := initTest(t, "a -> val='hello';", "a")

	types, deps := MakeTypesForTerms("a", terms[0])
	assert.NotEmpty(t, types)
	assert.Empty(t, deps)
	assert.IsType(t, rule{}, types["ANode"])
	assert.Len(t, types["ANode"].Children(), 1)
	assert.IsType(t, namedToken{}, types["ANode"].Children()[0])
}

func TestTypeBuilder_RuleWithUnnamedRuleVal(t *testing.T) {
	terms := initTest(t, "a -> x; x -> 'a''b';", "a")

	types, deps := MakeTypesForTerms("a", terms[0])
	assert.NotEmpty(t, types)
	assert.Empty(t, deps)
	assert.IsType(t, rule{}, types["ANode"])

	assert.Len(t, types["ANode"].Children(), 1)
	assert.IsType(t, namedRule{}, types["ANode"].Children()[0])
	child := types["ANode"].Children()[0].(namedRule)
	assert.Equal(t, "XNode", child.returnType)
	assert.Equal(t, "x", child.name)
}

func TestTypeBuilder_RuleWithNamedRuleVal(t *testing.T) {
	terms := initTest(t, "a -> val=x; x -> 'a''b';", "a")

	types, deps := MakeTypesForTerms("a", terms[0])
	assert.NotEmpty(t, types)
	assert.Empty(t, deps)
	assert.IsType(t, rule{}, types["ANode"])

	assert.Len(t, types["ANode"].Children(), 1)
	assert.IsType(t, namedRule{}, types["ANode"].Children()[0])
	child := types["ANode"].Children()[0].(namedRule)
	assert.Equal(t, "XNode", child.returnType)
	assert.Equal(t, "val", child.name)
}

func TestTypeBuilder_RuleWithUnnamedTerm(t *testing.T) {
	terms := initTest(t, "a -> 'a' /{[123]*};", "a")

	types, deps := MakeTypesForTerms("a", terms[0])
	assert.NotEmpty(t, types)
	assert.Empty(t, deps)
	assert.IsType(t, rule{}, types["ANode"])

	assert.Len(t, types["ANode"].Children(), 1)
	assert.IsType(t, unnamedToken{}, types["ANode"].Children()[0])
	child := types["ANode"].Children()[0].(unnamedToken)
	assert.Equal(t, -1, child.count)
}

func TestTypeBuilder_RuleWithUnnamedChoiceTerm(t *testing.T) {
	terms := initTest(t, "a -> 'a' | /{[123]*};", "a")

	types, deps := MakeTypesForTerms("a", terms[0])
	assert.NotEmpty(t, types)
	assert.Empty(t, deps)
	assert.IsType(t, rule{}, types["ANode"])

	assert.Len(t, types["ANode"].Children(), 2)
	assert.IsType(t, unnamedToken{}, types["ANode"].Children()[0])
	child := types["ANode"].Children()[0].(unnamedToken)
	assert.Equal(t, -1, child.count)
}

func TestTypeBuilder_RuleWithNamedTerm(t *testing.T) {
	terms := initTest(t, "a -> val=('a' /{[123]*});", "a")

	types, deps := MakeTypesForTerms("a", terms[0])
	assert.NotEmpty(t, types)
	assert.Empty(t, deps)
	assert.IsType(t, rule{}, types["ANode"])

	assert.IsType(t, namedRule{}, types["ANode"].Children()[0])
	child := types["ANode"].Children()[0].(namedRule)
	assert.Equal(t, "AValNode", child.returnType)
	assert.Equal(t, "val", child.name)
	assert.IsType(t, rule{}, types["AValNode"])
}

func TestTypeBuilder_RuleWithStack(t *testing.T) {
	terms := initTest(t, "a -> @ \\s? > 'hello';", "a")

	types, deps := MakeTypesForTerms("a", terms[0])
	assert.NotEmpty(t, types)
	assert.Empty(t, deps)
	assert.IsType(t, rule{}, types["ANode"])

	assert.Len(t, types["ANode"].Children(), 2)

	assert.IsType(t, namedRule{}, types["ANode"].Children()[0])
	child := types["ANode"].Children()[0].(namedRule)
	assert.Equal(t, "ANode", child.returnType)
	assert.Equal(t, "a", child.name)

	assert.IsType(t, unnamedToken{}, types["ANode"].Children()[1])
	child2 := types["ANode"].Children()[1].(unnamedToken)
	assert.Equal(t, -1, child2.count)
}
