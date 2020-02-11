package parser

import (
	"fmt"

	"github.com/arr-ai/wbnf/gotree"
)

type errorNode struct {
	TreeElement
}

func (e errorNode) Error() string {
	es := ""
	switch t := e.TreeElement.(type) {
	case Node:
		es = t.String()
	case Scanner:
		es = t.String()
	}
	if es != "" {
		return "partial parse nodes:  " + getErrorStrings(NewScanner(es))
	}
	return ""
}

type possibleFixup string

func (p possibleFixup) Error() string {
	return string(p)
}

type ParseError struct {
	rule     Rule
	msg      string
	children []error
}

func newParseError(rule Rule, msg string, errors ...error) error {
	return &ParseError{
		rule:     rule,
		msg:      msg,
		children: errors,
	}
}

func (p ParseError) Error() string {
	tree := gotree.New("parse failed")
	p.walkErrors(tree)

	return "\n" + tree.Print()
}

func (p ParseError) walkErrors(parent gotree.Tree) {
	x := gotree.New(fmt.Sprintf(`rule(%s) - %s`, p.rule, p.msg))
	for _, err := range p.children {
		if pe, ok := err.(*ParseError); ok {
			pe.walkErrors(x)
		} else {
			x.Add(err.Error())
		}
	}
	parent.AddTree(x)
}
