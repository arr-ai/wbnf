package cmd

import (
	"bytes"
	"go/format"
	"io/ioutil"
	"os"
	"strings"

	"github.com/arr-ai/wbnf/cmd/codegen"

	"github.com/arr-ai/wbnf/wbnf"
	"github.com/urfave/cli"
)

var pkgName string
var outFile string
var genCommand = cli.Command{
	Name:    "gen",
	Aliases: []string{"g"},
	Usage:   "Generate a grammar",
	Action:  gen,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:        "grammar",
			Usage:       "input grammar file",
			Required:    true,
			TakesFile:   true,
			Destination: &inGrammarFile,
		},
		cli.StringFlag{
			Name:        "pkg",
			Usage:       "name of the generated package",
			Required:    true,
			TakesFile:   false,
			Destination: &pkgName,
		},
		cli.StringFlag{
			Name:        "start",
			Usage:       "grammar rule to being parsing at",
			Required:    true,
			TakesFile:   false,
			Destination: &startingRule,
		},
		cli.StringFlag{
			Name:        "output",
			Usage:       "filename to write the output to",
			Required:    false,
			TakesFile:   false,
			Destination: &outFile,
		},
	},
}

func gen(c *cli.Context) error {
	g := loadTestGrammar()
	tree := g.Node().(wbnf.GrammarNode)

	types := codegen.MakeTypes(tree)
	tmpldata := codegen.TemplateData{
		CommandLine:       strings.Join(os.Args[1:], " "),
		PackageName:       pkgName,
		StartRule:         codegen.IdentName(startingRule),
		StartRuleTypeName: codegen.GoTypeName(startingRule),
		Grammar:           codegen.MakeGrammarString(g.Grammar()),
		MiddleSection: append(
			types.Get(),
			codegen.IdentsWriter{GrammarNode: tree},
			codegen.GetVisitorWriter(types.Types(), startingRule)),
	}
	var buf bytes.Buffer
	if err := codegen.Write(&buf, tmpldata); err != nil {
		return err
	}

	out, err := format.Source(buf.Bytes())
	if err != nil {
		return err
	}

	switch outFile {
	case "", "-":
		os.Stdout.Write(out)
	default:
		ioutil.WriteFile(outFile, out, 0644) //nolint:errcheck
	}

	return nil
}
