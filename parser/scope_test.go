package parser

import (
	"testing"

	"github.com/magiconair/properties/assert"
)

func makeTracepoints(vals ...int) []tracepoint {
	out := make([]tracepoint, 0, len(vals)/2)
	for i := 0; i < len(vals); i += 2 {
		out = append(out, tracepoint{detail: Detail{Id: int32(vals[i])}, offset: vals[i+1]})
	}
	return out
}

func TestScopeCycle(t *testing.T) {
	for _, test := range []struct {
		tp    []tracepoint
		cycle bool
	}{{tp: makeTracepoints(0, 10, 1, 11, 2, 12), cycle: false},
		{tp: makeTracepoints(0, 10, 1, 11, 0, 10, 1, 11), cycle: true},
		{tp: makeTracepoints(0, 10, 1, 11, 0, 10, 1, 12, 1, 12, 1, 12, 1, 12), cycle: true},
	} {
		test := test
		t.Run("TestScopeCycle", func(t *testing.T) {
			assert.Equal(t, test.cycle, Scope{trace: test.tp}.isCycle())
		})
	}
}
