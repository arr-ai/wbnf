package errors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorMessages(t *testing.T) {
	t.Parallel()
	assert.EqualError(t, Inconceivable, "How did this happen!?")
	assert.EqualError(t, Unfinished, "not yet implemented")
	assert.EqualError(t, BadInput, "bad input")
}
