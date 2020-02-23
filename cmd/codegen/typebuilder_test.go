package codegen

import (
	"fmt"
	"testing"

	"github.com/arr-ai/wbnf/wbnf"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

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
			assert.IsType(t, choice{}, child, errMessage)
			continue
		}
		assert.IsType(t, val.t, child, errMessage)
		switch x := child.(type) {
		case namedRule:
			assert.Equal(t, val.quant, x.count, errMessage)
			assert.Equal(t, val.returnType, x.returnType, errMessage)
		case namedToken:
			assert.Equal(t, val.quant, x.count, errMessage)
			assert.Equal(t, "", val.returnType, "Test mis-configured"+errMessage)

		}
	}
}

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

func TestTypeBuilder2_ScopedGrammarRulesComplex(t *testing.T) {
	types := initTypeBuilderTest(t, `pragma  -> import {
                import -> ".import" path=((".."|"."|[a-zA-Z0-9.:]+):,"/") ";"?;
            };`)

	assert.NotEmpty(t, types)
	assert.IsType(t, rule{}, types["PragmaNode"])
	assert.IsType(t, rule{}, types["PragmaImportNode"])
	assert.IsType(t, rule{}, types["PragmaImportPathNode"])
	assert.IsType(t, nil, types["PragmaImportPathDelimNode"])

	testChildren(t, types["PragmaNode"].Children(), childrenTestData{
		"import": {t: namedRule{}, quant: wantOneGetter,
			returnType: "PragmaImportNode"},
	})
	testChildren(t, types["PragmaImportNode"].Children(), childrenTestData{
		"path":  {t: namedRule{}, quant: wantOneGetter, returnType: "PragmaImportPathNode"},
		"Token": {t: unnamedToken{}, quant: wantAllGetter},
	})

}

func TestTypeBuilder2_Delims(t *testing.T) {
	types := initTypeBuilderTest(t, "a -> 'hello':op=',';"+
		"b -> a:,/{[asd]};"+
		"c->'a':a;"+
		"x -> b:foo=/{[abcd]*};")

	assert.NotEmpty(t, types)
	assert.IsType(t, rule{}, types["ANode"])
	assert.IsType(t, rule{}, types["BNode"])
	assert.IsType(t, nil, types["BDelimNode"])
	assert.IsType(t, rule{}, types["CNode"])
	assert.IsType(t, rule{}, types["CaNode"])
	testChildren(t, types["ANode"].Children(), childrenTestData{
		"op":    {t: namedToken{}, quant: wantOneGetter},
		"Token": {t: unnamedToken{}, quant: wantAllGetter},
	})
}

func TestTypeBuilder2_RuleWithQuant(t *testing.T) {
	types := initTypeBuilderTest(t, "a -> a? ;")

	assert.NotEmpty(t, types)
	assert.IsType(t, rule{}, types["ANode"])
	testChildren(t, types["ANode"].Children(), childrenTestData{
		"a": {t: namedRule{}, quant: wantOneGetter, returnType: "ANode"},
	})
}

func TestTypeBuilder2_RuleWithUnnamedTerm(t *testing.T) {
	types := initTypeBuilderTest(t, "a -> 'a' /{[123]*};")
	assert.NotEmpty(t, types)
	assert.IsType(t, rule{}, types["ANode"])

	testChildren(t, types["ANode"].Children(), childrenTestData{
		"Token": {t: unnamedToken{}, quant: wantAllGetter},
	})
}

func TestTypeBuilder_RuleWithNamedTerm(t *testing.T) {
	types := initTypeBuilderTest(t, "a -> val=('a' /{[123]*});")

	assert.NotEmpty(t, types)
	assert.IsType(t, rule{}, types["ANode"])

	testChildren(t, types["ANode"].Children(), childrenTestData{
		"val": {t: namedRule{}, quant: wantOneGetter, returnType: "AvalNode"},
	})
	testChildren(t, types["AValNode"].Children(), childrenTestData{
		"Token": {t: unnamedToken{}, quant: wantAllGetter},
	})
}

func TestTypeBuilder2_RuleWithStack(t *testing.T) {
	types := initTypeBuilderTest(t, "a -> @ \\s? > 'hello';")
	assert.NotEmpty(t, types)
	assert.IsType(t, rule{}, types["ANode"])

	testChildren(t, types["ANode"].Children(), childrenTestData{
		"a":     {t: stackBackRef{}},
		"Token": {t: unnamedToken{}, quant: wantAllGetter},
	})
}

func TestTypeBuilder2_RuleWithStackComplicated(t *testing.T) {
	types := initTypeBuilderTest(t, `term    -> @:op=">"
         > @:op="|"
         > @+
         > named quant*;
		named -> [a-z]*; quant -> [*?+];`)

	assert.NotEmpty(t, types)
	assert.IsType(t, rule{}, types["TermNode"])
	assert.IsType(t, nil, types["TermOpNode"])

	testChildren(t, types["TermNode"].Children(), childrenTestData{
		"term":  {t: stackBackRef{}},
		"op":    {t: namedToken{}, quant: wantAllGetter},
		"named": {t: namedRule{}, quant: wantOneGetter, returnType: "NamedNode"},
		"quant": {t: namedRule{}, quant: wantAllGetter, returnType: "QuantNode"},
	})
}

func TestTypeBuilder_RulePointingToOnlyUnnamedTokenVal(t *testing.T) {
	types := initTypeBuilderTest(t, "a -> b; b -> 'hello';")
	assert.NotEmpty(t, types)
	assert.IsType(t, rule{}, types["ANode"])
	testChildren(t, types["ANode"].Children(), childrenTestData{
		"b": {t: namedRule{}, quant: wantOneGetter, returnType: "BNode"},
	})
}

func TestTypeBuilder_RuleWithOnlyNamedTokenVal(t *testing.T) {
	types := initTypeBuilderTest(t, "a -> val='hello';")
	assert.NotEmpty(t, types)
	assert.IsType(t, rule{}, types["ANode"])
	testChildren(t, types["ANode"].Children(), childrenTestData{
		"val": {t: namedToken{}, quant: wantOneGetter},
	})
}

func TestTypeBuilder_RuleWithUnnamedRuleVal(t *testing.T) {
	types := initTypeBuilderTest(t, "a -> x; x -> 'a''b';")
	assert.NotEmpty(t, types)
	assert.IsType(t, rule{}, types["ANode"])
	testChildren(t, types["ANode"].Children(), childrenTestData{
		"x": {t: namedRule{}, quant: wantOneGetter, returnType: "XNode"},
	})
}

func TestTypeBuilder_RuleWithNamedRuleVal(t *testing.T) {
	types := initTypeBuilderTest(t, "a -> val=x; x -> 'a''b';")
	assert.NotEmpty(t, types)
	assert.IsType(t, rule{}, types["ANode"])

	testChildren(t, types["ANode"].Children(), childrenTestData{
		"val": {t: namedRule{}, quant: wantOneGetter, returnType: "XNode"},
	})
}

func TestTypeBuilder_RuleWithUnnamedChoiceTerm(t *testing.T) {
	types := initTypeBuilderTest(t, "a -> 'a' | /{[123]*};")
	assert.NotEmpty(t, types)
	assert.IsType(t, rule{}, types["ANode"])

	testChildren(t, types["ANode"].Children(), childrenTestData{
		"@choice": {},
		"Token":   {t: unnamedToken{}, quant: wantAllGetter},
	})
}

func TestTypeBuilder_RuleWithRefs(t *testing.T) {
	types := initTypeBuilderTest(t, "a -> foo='a' %foo; b -> %a;")
	assert.NotEmpty(t, types)
	assert.IsType(t, rule{}, types["ANode"])
	assert.IsType(t, rule{}, types["BNode"])

	testChildren(t, types["ANode"].Children(), childrenTestData{
		"foo": {t: namedToken{}, quant: wantAllGetter},
	})
	testChildren(t, types["BNode"].Children(), childrenTestData{
		"a": {t: backRef{}, quant: wantAllGetter, returnType: "ANode"},
	})
}

func TestTypeBuilder_RuleWithOptsAndNames(t *testing.T) {
	types := initTypeBuilderTest(t, `quant   -> op=[?*+]
         | "{" min=INT? "," max=INT? "}"
         | op=/{<:|:>?} opt_leading=","? "named" opt_trailing=","?;
INT     -> \d+;`)
	assert.NotEmpty(t, types)
	assert.IsType(t, rule{}, types["QuantNode"])
	assert.IsType(t, basicRule(""), types["INTNode"])

	testChildren(t, types["QuantNode"].Children(), childrenTestData{
		"@choice":      {},
		"op":           {t: namedToken{}, quant: wantAllGetter}, // TODO: make this wantOneGetter
		"min":          {t: namedRule{}, returnType: GoTypeName("INT"), quant: wantOneGetter},
		"max":          {t: namedRule{}, returnType: GoTypeName("INT"), quant: wantOneGetter},
		"opt_leading":  {t: namedToken{}, quant: wantOneGetter},
		"opt_trailing": {t: namedToken{}, quant: wantOneGetter},
		"Token":        {t: unnamedToken{}, quant: wantAllGetter},
	})
}

func TestDropCaps(t *testing.T) {
	tests := []string{
		"A", "A",
		"a", "a",
		"Bar", "Bar",
		"INT", "Int",
	}
	for i := 0; i < len(tests); i += 2 {
		t.Run(tests[i], func(t *testing.T) {
			in := tests[i]
			expected := tests[i+1]
			assert.EqualValues(t, expected, DropCaps(in))
		})
	}
}
