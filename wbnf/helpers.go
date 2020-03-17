package wbnf

import (
	"github.com/arr-ai/wbnf/ast"
	"github.com/arr-ai/wbnf/parser"
)

// Choice attempts to check for the @choice tag, and if found returns the value
// The return is the 0-based index of the chosen option. -1 means there was no @choice tag
func Choice(n ast.Node) int {
	if n == nil {
		return -1
	}
	if choice := ast.First(n, ast.ChoiceTag); choice != nil {
		return int(choice.(ast.Extra).Data.(parser.Choice))
	}
	return -1
}

