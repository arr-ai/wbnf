package bootstrap

import (
	"fmt"
	"strings"
)

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
	return p.walkErrors(0)
}

func prefix(depth int) string {
	if depth == 0 {
		return ""
	}
	return fmt.Sprintf("\t\\%s ", strings.Repeat("-", depth))
}

func (p ParseError) walkErrors(depth int) string {
	lines := []string{
		fmt.Sprintf(`%srule(%s) - %s`, prefix(depth), p.rule, p.msg),
	}
	for _, err := range p.children {
		if pe, ok := err.(*ParseError); ok {
			lines = append(lines, pe.walkErrors(depth+1))
		} else {
			lines = append(lines, fmt.Sprintf(`%s	 %s`, prefix(depth), err.Error()))
		}
	}
	return strings.Join(lines, "\n")
}
