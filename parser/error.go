package parser

import (
	"fmt"

	"github.com/arr-ai/wbnf/gotree"
)

type ParseError struct {
	rule     Rule
	msg      string
	children []error
}

type FatalError struct {
	ParseError
	cutpointdata
}

func isFatal(err error) bool {
	_, ok := err.(FatalError)
	return ok
}
func isNotMyFatalError(err error, cp cutpointdata) bool {
	fe, ok := err.(FatalError)
	return ok && fe.cutpointdata != cp
}

func newParseError(rule Rule, msg string, fatal cutpointdata, errors ...error) error {
	err := ParseError{
		rule:     rule,
		msg:      msg,
		children: errors,
	}
	if fatal.valid() {
		return FatalError{err, fatal}
	}
	return err
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

type UnconsumedInputError struct {
	residue Scanner
	tree    TreeElement
}

// UnconsumedInput is returned by a successful parse that didn't fully
// consume the input.
func UnconsumedInput(residue Scanner, result TreeElement) UnconsumedInputError {
	return UnconsumedInputError{residue: residue, tree: result}
}

func (e UnconsumedInputError) Error() string {
	return fmt.Sprintf("unconsumed input: %v", e.residue)
}
