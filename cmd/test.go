package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/arr-ai/wbnf/bootstrap"
	"github.com/arr-ai/wbnf/parser"

	"github.com/urfave/cli"
)

var testCommand = cli.Command{
	Name:    "test",
	Aliases: []string{"t"},
	Usage:   "Test a grammar",
	Action:  test,
}

func test(c *cli.Context) error {
	source := c.Args().Get(0)

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
	scanner := parser.NewScanner(input)
	g := bootstrap.Core()

	tree, err := g.Parse(bootstrap.RootRule, scanner)
	if err != nil {
		return err
	}
	if err := g.ValidateParse(tree); err != nil {
		return err
	}

	data, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))

	return nil
}
