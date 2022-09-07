package parser

import (
	"fmt"

	"github.com/arr-ai/wbnf/gotree"
)

type ParseError struct {
	rule      Rule
	msgFormat string
	msgArgs   []any
	children  []func() error
}

type FatalError struct {
	ParseError
	Cutpointdata
}

type StopError interface {
	IsStopError()
}

func isFatal(err error) bool {
	switch err.(type) {
	case FatalError, StopError:
		return true
	}
	return false
}

func isNotMyFatalError(err error, cp Cutpointdata) bool {
	switch err := err.(type) {
	case FatalError:
		return err.Cutpointdata != cp
	case StopError:
		return true
	}
	return false
}

func newParseError(
	rule Rule,
	format string,
	args ...any,
) func(fatal Cutpointdata, errors ...func() error) error {
	return func(fatal Cutpointdata, errors ...func() error) error {
		err := ParseError{
			rule:      rule,
			msgFormat: format,
			msgArgs:   args,
			children:  errors,
		}
		if fatal.valid() {
			return FatalError{err, fatal}
		}
		return err
	}
}

func (p ParseError) Error() string {
	tree := gotree.New("parse failed")
	p.walkErrors(tree)

	return "\n" + tree.Print()
}

func (p ParseError) walkErrors(parent gotree.Tree) {
	x := gotree.New(fmt.Sprintf(`rule(%s) - %s`, p.rule, fmt.Sprintf(p.msgFormat, p.msgArgs...)))
	for _, errf := range p.children {
		err := errf()
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
	return fmt.Sprintf("unconsumed input\n %v", e.residue.Context(DefaultLimit))
}

func (e UnconsumedInputError) Result() TreeElement { return e.tree }
func (e UnconsumedInputError) Residue() *Scanner   { return &e.residue }
