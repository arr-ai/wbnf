package bootstrap

import (
	"fmt"
	"strings"
)

const (
	// Inconceivable indicates that a function should never have been called.
	Inconceivable Error = "How did this happen!?"

	// Unfinished indicates that a function isn't ready for use yet.
	Unfinished Error = "not yet implemented"

	// BadInput indicates that a function was given bad inputs.
	BadInput Error = "bad input"
)

type Error string

var _ error = Error("")

func (p Error) Error() string {
	return string(p)
}

type ParseError struct {
	rule Rule
	msgs []string
}

func (p ParseError) Error() string {
	return fmt.Sprintf("rule(%s) -\n  %s", p.rule, strings.Join(p.msgs, "\n  "))
}

func newParseError(rule Rule, msgs ...string) error {
	return &ParseError{
		rule: rule,
		msgs: msgs,
	}
}
