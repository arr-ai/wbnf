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
		MiddleSection: MakeTypes("", g),
	}))

	assert.EqualValues(t, "hello", buf.String())
}
