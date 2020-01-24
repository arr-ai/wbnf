package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/arr-ai/wbnf/wbnf"
	"github.com/arr-ai/wbnf/parser"
	"github.com/urfave/cli"
)

var inFile string
var codegenCommand = cli.Command{
	Name:    "codegen",
	Aliases: []string{"gen"},
	Usage:   "Generate go code from a grammar",
	Action:  codegen,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:        "input",
			Usage:       "input grammar file to dump",
			Required:    true,
			TakesFile:   true,
			Destination: &inFile,
		},
	},
}

func dumpNode(term parser.Node, o io.Writer) error {
	tag := strings.Split(term.Tag, "\\")
	switch len(tag) {
	case 0:
		return nil // probably an error
	}
	switch tag[0] {
	case "grammar":

	}

	return nil
}

func codegen(c *cli.Context) error {
	buf, err := ioutil.ReadFile(inFile)
	if err != nil {
		return err
	}

	scanner := parser.NewScanner(string(buf))
	g := wbnf.Core()

	tree, err := g.Parse(wbnf.RootRule, scanner)
	if err != nil {
		return err
	}

	if err := g.ValidateParse(tree); err != nil {
		return err
	}

	ast := wbnf.NewFromNode2(tree.(parser.Node))
	if ast == nil {
		return nil
	} /*
		text := ast.Dump()

		newg := wbnf.NewFromNode(tree.(parser.Node)).Compile()

		scanner = parser.NewScanner(text)
		_, err := newg.Parse(wbnf.RootRule, scanner)
		if err != nil {
			return err
		}
	*/
	fmt.Print(wbnf.Codegen(ast))

	return nil
}
