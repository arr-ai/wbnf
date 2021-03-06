package codegen

import (
	"fmt"
	"io"
	"text/template"
)

type TemplateData struct {
	CommandLine string
	PackageName string

	StartRule         string
	StartRuleTypeName string

	Grammar *GoNode

	MiddleSection []fmt.Stringer
}

const outFileTemplate = `// Code generated by "ωBNF gen" DO NOT EDIT.
// $ wbnf {{.CommandLine}}
package {{.PackageName}}

import (
	"github.com/arr-ai/wbnf/ast"
	"github.com/arr-ai/wbnf/parser"
)

func Grammar() parser.Parsers {
	return {{.Grammar}}.Compile(nil)
}

type Stopper interface {
	 ExitNode() bool
	 Abort() bool
}
type nodeExiter struct{}
func (n *nodeExiter) ExitNode() bool {return true }
func (n *nodeExiter) Abort() bool {return false }

type aborter struct{}
func (n *aborter) ExitNode() bool {return true }
func (n *aborter) Abort() bool {return true }
var (
	NodeExiter = &nodeExiter{}
	Aborter    = &aborter{}
)

type IsWalkableType interface { isWalkableType() }

{{ range .MiddleSection }} {{.}} {{end}}

func (c {{.StartRuleTypeName}}) GetAstNode() ast.Node { return c.Node }

func New{{.StartRuleTypeName}}(from ast.Node) {{.StartRuleTypeName}} { return {{.StartRuleTypeName}}{ from } }

func Parse(input *parser.Scanner) ({{.StartRuleTypeName}}, error) {
	p := Grammar()
	tree, err := p.Parse({{.StartRule}}, input)
	if err != nil {
		return {{.StartRuleTypeName}}{nil}, err
	}
	return {{.StartRuleTypeName}}{ast.FromParserNode(p.Grammar(), tree)}, nil
}

func ParseString(input string) ({{.StartRuleTypeName}}, error) {
	return Parse(parser.NewScanner(input))
}
`

func Write(w io.Writer, data TemplateData) error {
	tmpl, err := template.New("output").Parse(outFileTemplate)
	if err != nil {
		panic(err)
	}

	return tmpl.Execute(w, data)
}
