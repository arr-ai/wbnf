package wbnf

import (
	"testing"

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
	} {
		test := test
		t.Run("TestValidationErrors-"+test.name, func(t *testing.T) {
			node, err := ParseString(test.grammar)
			assert.NoError(t, err)
			assert.NotNil(t, node.Node)
			err = validate(node)
			assert.Error(t, err)
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
		})
	}
}
