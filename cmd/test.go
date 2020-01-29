package cmd

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/arr-ai/wbnf/ast"
	"github.com/arr-ai/wbnf/parser"
	"github.com/arr-ai/wbnf/wbnf"

	"github.com/urfave/cli"
)

var inGrammarFile string
var startingRule string
var testCommand = cli.Command{
	Name:    "test",
	Aliases: []string{"t"},
	Usage:   "Test a grammar",
	Action:  test,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:        "grammar",
			Usage:       "input grammar file",
			Required:    true,
			TakesFile:   true,
			Destination: &inGrammarFile,
		},
		cli.StringFlag{
			Name:        "start",
			Usage:       "starting rule to process the input text",
			Required:    true,
			TakesFile:   false,
			Destination: &startingRule,
		},
		cli.StringFlag{
			Name:        "input",
			Usage:       "input test file",
			Required:    false,
			TakesFile:   true,
			Destination: &inFile,
		},
	},
}

func loadTestGrammar() wbnf.Parsers {
	text, err := ioutil.ReadFile(inGrammarFile)
	if err != nil {
		panic(err)
	}
	return wbnf.MustCompile(string(text))
}

func test(c *cli.Context) error {
	source := inFile

	g := loadTestGrammar()

	var input string
	switch source {
	case "", "-":
		buf, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		input = string(buf)
	default:
		buf, err := ioutil.ReadFile(source)
		if err != nil {
			return err
		}
		input = string(buf)
	}

	tree, err := g.Parse(wbnf.Rule(startingRule), parser.NewScanner(input))
	if err != nil {
		return err
	}
	ast := ast.ParserNodeToNode(g.Grammar(), tree)
	fmt.Println(ast)

	return nil
}
