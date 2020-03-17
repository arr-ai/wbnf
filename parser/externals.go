package parser

import "github.com/arr-ai/wbnf/parse"

// Func will be called when the parser hits a %%ref term during a parse.

type ExternalRef func(scope Scope, input *parse.Scanner) (parse.TreeElement, error)

type ExternalRefs map[string]ExternalRef
