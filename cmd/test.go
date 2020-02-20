package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"

	"github.com/arr-ai/wbnf/ast"
	"github.com/arr-ai/wbnf/parser"
	"github.com/arr-ai/wbnf/wbnf"

	"github.com/urfave/cli"
)

var inFile string
var inGrammarFile string
var startingRule string
var verboseMode bool
var printTree bool
var testCommand = cli.Command{
	Name:    "test",
	Aliases: []string{"t"},
	Usage:   "Test a grammar",
	Action:  test,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:        "grammar",
			Usage:       "input grammar file",
			Required:    false,
			TakesFile:   true,
			Destination: &inGrammarFile,
		},
		cli.StringFlag{
			Name:        "start",
			Usage:       "starting rule to process the input text",
			Required:    false,
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
		cli.BoolFlag{
			Name:        "v",
			Usage:       "verbose logging",
			EnvVar:      "",
			FilePath:    "",
			Required:    false,
			Hidden:      false,
			Destination: &verboseMode,
		},
		cli.BoolFlag{
			Name:        "tree",
			Usage:       "pretty print the AST",
			EnvVar:      "",
			FilePath:    "",
			Required:    false,
			Hidden:      false,
			Destination: &printTree,
		},
	},
}

type resolver struct {
	base string
}

func (r resolver) Resolve(from, file string) string {
	if from == "" {
		return filepath.Clean(filepath.Join(r.base, file))
	}
	return filepath.Clean(filepath.Join(r.base, from, file))
}

func makeResolver(firstFilename string) resolver {
	return resolver{filepath.Dir(firstFilename)}
}

func loadTestGrammar() parser.Parsers {
	text, err := ioutil.ReadFile(inGrammarFile)
	if err != nil {
		panic(err)
	}
	if startingRule == "" {
		panic(fmt.Errorf("--start missing"))
	}
	return wbnf.MustCompile(string(text), makeResolver(inGrammarFile))
}

func testWbnfFile(filename, grammar string) error {
	g := wbnf.MustCompile(grammar, makeResolver(filename))
	if printTree {
		fmt.Println(ast.BuildTreeView("grammar", g.Node().(wbnf.GrammarNode).Node, true))
	} else {
		fmt.Println(g.Node().(ast.Node))
	}

	return nil
}

func test(c *cli.Context) error {
	source := inFile

	defer func() {
		if r := recover(); r != nil {
			fmt.Print(r)
			os.Exit(1)
		}
	}()

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
	if inGrammarFile == "" {
		return testWbnfFile(source, input)
	}
	g := loadTestGrammar()

	if verboseMode {
		logrus.SetLevel(logrus.TraceLevel)
	}
	if !g.HasRule(parser.Rule(startingRule)) {
		return fmt.Errorf("starting rule '%s' not in test grammar", startingRule)
	}
	tree, err := g.Parse(parser.Rule(startingRule), parser.NewScanner(input))
	if err != nil {
		return err
	}
	a := ast.FromParserNode(g.Grammar(), tree)
	if printTree {
		fmt.Println(ast.BuildTreeView(startingRule, a, true))
	} else {
		fmt.Println(a)
	}

	return nil
}
