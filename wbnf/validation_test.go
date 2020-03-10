package wbnf

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

type testData struct {
	name, grammar string
	ekind         validationErrorKind
}

func TestValidationErrors(t *testing.T) {
	for _, test := range []testData{
		{"undefined rule", "a -> b;", UnknownRule},
		{"redefined rule", "a -> 'b';  a -> 'f';", DuplicatedRule},
		{"invalid regex", "a -> /{[};", InvalidRegex},
		{"named term clashes with rule", "a -> a='hello';", NameClashesWithRule},
		{"min/max switch", "a -> 'a'{10,1};", MinMaxQuantError},

		{"multiple uses of name", "a -> op='|' op='b';", MultipleTermsWithSameName},
		{"legal multiple uses of name 1", "a -> (op='|' | op='b');", NoError},
		{"legal multiple uses of name 2", "a -> x=(op='|') y=(op='b');", NoError},

		{"macro valid", "a -> 'a'; .macro Foo(b) { b }", NoError},
		{"macro arg clashes with rule", "a -> 'a'; .macro Foo(a) { a }", NameClashesWithRule},
		{"macro name clashes with rule", "a -> 'a'; .macro a(b) { b }", DuplicatedRule},
		{"calling a rule", "a -> 'a'; x -> %!a('a');", NotAMacro},
		{"macro arg count", "a -> %!Foo('a', 'b'); .macro Foo(b) { b };", IncorrectMacroArgCount},
		{"macro arg count", "a -> %!Foo(); .macro Foo(b) { b };", IncorrectMacroArgCount},
		// Wish-list validity checks:

		// Should fail because op would return different types
		// {"redefined term name", "a -> (op='|' | op=x); x-> 'a'*", NamedTermWithConflictingTypes},

	} {
		test := test
		t.Run("TestValidationErrors-"+test.name, func(t *testing.T) {
			node, err := ParseString(test.grammar)
			assert.NoError(t, err)
			assert.NotNil(t, node.Node)
			err = validate(node)
			if test.ekind != NoError {
				require.Error(t, err)
				switch err := err.(type) {
				case *validator:
					assert.Len(t, err.err, 1)
					assert.Equal(t, test.ekind, err.err[0].(validationError).kind)
				case validationError:
					assert.Equal(t, test.ekind, err.kind)
				}
				assert.NotPanics(t, func() {
					_ = err.Error()
				})
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
