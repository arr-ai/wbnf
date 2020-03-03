package parser

// Func will be called when the parser hits a %%ref term during a parse.

type ExternalRef func(scope Scope, input *Scanner) (TreeElement, error)

type ExternalRefs map[string]ExternalRef
