package cmd

import (
	"fmt"
	"go/format"
	"os"
	"strings"

	"github.com/arr-ai/wbnf/parser"

	"github.com/arr-ai/wbnf/ast"
	"github.com/arr-ai/wbnf/wbnf"
	"github.com/urfave/cli"
)

var pkgName string
var rootRuleName string
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
			Name:        "rootrule",
			Usage:       "grammar rule to being parseing at",
			Required:    true,
			TakesFile:   false,
			Destination: &rootRuleName,
		},
	},
}

func gen(c *cli.Context) error {
	g := loadTestGrammar()

	core := wbnf.Core()
	tree := ast.FromParserNode(core.Grammar(), *g.Node())

	idents := wbnf.IdentMap(tree)
	fmt.Print(idents)

	root := goNode{name: "parser.Grammar", scope: squiglyScope}

	for _, stmt := range tree.Many("stmt") {
		if p := stmt.One("prod"); p != nil {
			root.Add(*makeProd(p))
		}
	}

	text := fmt.Sprintf(`// Code generated by "ωBNF gen" DO NOT EDIT.
// $ wbnf %s
package %s

import (
	"github.com/arr-ai/wbnf/ast"
	"github.com/arr-ai/wbnf/parser"
)

func Grammar() parser.Grammar {
	return %s
}

func Parse(input *parser.Scanner) (ast.Node, error) {
	p := Grammar().Compile(nil)
	tree, err := p.Parse("%s", input)
	if err != nil {
		return nil, err
	}
    return ast.FromParserNode(p.Grammar(), tree), nil
}

func ParseString(input string) (ast.Node, error) {
	return Parse(parser.NewScanner(input))
}

`, strings.Join(os.Args[1:], " "), pkgName, root.String(), rootRuleName)

	out, err := format.Source([]byte(text))
	if err != nil {
		fmt.Println(err, root.String())

	}

	os.Stdout.Write(out)

	return nil
}

const (
	noScope int = iota
	bracesScope
	squiglyScope
	mapScope
)

type goNode struct {
	name     string
	children []goNode
	scope    int
}

func (g *goNode) String() string {
	x := map[int]struct {
		open  string
		close string
	}{
		noScope:      {"", ""},
		mapScope:     {":", ""},
		bracesScope:  {"(", ")"},
		squiglyScope: {"{", "}"},
	}[g.scope]
	var children []string
	for _, c := range g.children {
		children = append(children, c.String())
	}
	return strings.Join([]string{g.name, x.open, strings.Join(children, ",\n"), x.close}, "")
}

func (g *goNode) Add(n goNode) {
	g.children = append(g.children, n)
}

func safeString(src string) string {
	r := strings.NewReplacer("`", "`+\"`\"+`", " ", "", "\n", "")
	return r.Replace(src)
}

func makeAtom(node ast.Node) *goNode {
	atom := node.(ast.Branch)
	x, _ := ast.Which(atom, "RE", "STR", "IDENT", "REF", "term")
	name := ""
	switch x {
	case "term", "":
	case "REF":
		name = safeString(atom.One(x).One("IDENT").Scanner().String())
	default:
		name = safeString(atom.One(x).Scanner().String())
	}
	switch x {
	case "IDENT":
		return &goNode{name: fmt.Sprintf("parser.Rule(`%s`)", name)}
	case "STR":
		return &goNode{name: fmt.Sprintf("parser.S(%s)", name)}
	case "RE":
		return &goNode{name: fmt.Sprintf("parser.RE(`%s`)", name)}
	case "REF":
		return &goNode{name: fmt.Sprintf("parser.REF(`%s`)", name)}
	case "term":
		return makeTerm(atom.One(x))
	}
	return &goNode{name: "todo"}
}
func makeNamed(node ast.Node) *goNode {
	named := node.(ast.Branch)
	atom := makeAtom(named.One("atom"))

	if named.One("IDENT") != nil {
		val := &goNode{name: "parser.Eq",
			scope:    bracesScope,
			children: []goNode{{name: "\"" + named.One("IDENT").Scanner().String() + "\""}, *atom},
		}
		return val
	}
	return atom
}
func makeQuant(node ast.Node, term goNode) *goNode {
	switch node.Many(ast.ChoiceTag)[0].(ast.Extra).Data.(parser.Choice) {
	case 0:
		switch node.One("op").Scanner().String() {
		case "*":
			return &goNode{name: "parser.Any", scope: bracesScope, children: []goNode{term}}
		case "?":
			return &goNode{name: "parser.Opt", scope: bracesScope, children: []goNode{term}}
		case "+":
			return &goNode{name: "parser.Some", scope: bracesScope, children: []goNode{term}}
		}
	case 1:
		min := "0"
		max := "0"
		if x := node.One("min"); x != nil {
			min = x.Scanner().String()
		}
		if x := node.One("max"); x != nil {
			max = x.Scanner().String()
		}
		term.name = "Term: " + term.name
		return &goNode{name: "parser.Quant", scope: squiglyScope, children: []goNode{term, {name: "Min:" + min}, {name: "Max:" + max}}}
	case 2:
		delim := &goNode{name: "parser.Delim", scope: squiglyScope}
		var assoc string
		switch node.One("op").Scanner().String() {
		case "<:":
			assoc = "Assoc: parser.RightToLeft"
		case ":>":
			assoc = "Assoc: parser.LeftToRight"
		default:
			assoc = "Assoc: parser.NonAssociative"
		}
		term.name = "Term: " + term.name
		sep := *makeNamed(node.One("named"))
		sep.name = "Sep: " + sep.name
		delim.children = []goNode{term, sep, {name: assoc}}
		if node.One("opt_leading") != nil {
			delim.children = append(delim.children, goNode{name: "CanStartWithSep: true"})
		}
		if node.One("opt_trailing") != nil {
			delim.children = append(delim.children, goNode{name: "CanEndWithSep: true"})
		}
		return delim
	}
	return &goNode{name: "todo"}
}

func makeTerm(node ast.Node) *goNode {
	term := node.(ast.Branch)
	x, _ := ast.Which(term, "term", "atom", "named")
	switch x {
	case "term":
		var next *goNode
		if ops := term.Many("op"); len(ops) > 0 {
			switch ops[0].Scanner().String() {
			case "|":
				next = &goNode{name: "parser.Oneof", scope: squiglyScope}
			case ">":
				next = &goNode{name: "parser.Stack", scope: squiglyScope}
			}
		} else {
			next = &goNode{name: "parser.Seq", scope: squiglyScope}
		}
		for _, t := range term.Many("term") {
			next.Add(*makeTerm(t))
		}
		if len(next.children) == 1 {
			return &next.children[0]
		}
		return next
	case "named":
		// named and quants need to be added backwards
		// "a":","*     ->   Any(Delim(... S("a")))
		next := makeNamed(term.One("named"))
		quants := term.Many("quant")
		for i := range quants {
			next = makeQuant(quants[len(quants)-1-i], *next)
		}
		return next

	}
	return &goNode{name: "todo"}
}

func makeProd(tree ast.Node) *goNode {
	terms := tree.Many("term")

	p := &goNode{
		name: fmt.Sprintf(`"%s"`,
			tree.One("IDENT").Scanner().String()),
		children: nil,
		scope:    mapScope,
	}
	for _, t := range terms {
		p.Add(*makeTerm(t))
	}
	return p
}
