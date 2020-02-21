package codegen

import (
	"bytes"
	"testing"

	"github.com/arr-ai/wbnf/wbnf"

	"github.com/stretchr/testify/assert"
)

func TestWrite(t *testing.T) {
	var buf bytes.Buffer

	g, _ := wbnf.ParseString("hard->asd=('a' | fff=('b' | 'hello')*);")
	types := MakeTypes(g)
	assert.NoError(t, Write(&buf, TemplateData{
		CommandLine:       "foo bar baz",
		PackageName:       "testpackage",
		StartRule:         "IdentStartRule",
		StartRuleTypeName: "StartRuleNode",
		Grammar: &goNode{name: "parser.Grammar", scope: squigglyScope, children: []goNode{{
			name:     "parser.Foooooo",
			children: nil,
			scope:    bracesScope,
		}}},
		MiddleSection: append(types.Get(), VisitorWriter{startRule: "IdentStartRule", types: types.types}),
	}))

	assert.EqualValues(t, "hello", buf.String())
}
