package parser

import (
	"fmt"

	"github.com/arr-ai/frozen"
)

type tracepoint struct {
	detail Detail
	offset int
}

type Scope struct {
	data  frozen.Map
	trace []tracepoint
}

func (s Scope) clone() Scope {
	return Scope{
		data:  s.data,
		trace: append(s.trace[:]),
	}
}

func (s Scope) Mark(tp tracepoint) Scope {
	clone := s.clone()
	clone.trace = append(clone.trace, tp)
	if clone.isCycle() {
		panic(fmt.Errorf("non-progressing cycle detected: %+v", s.trace))
	}
	return clone
}

func (s Scope) WithIdent(ident string, p Parser, val TreeElement) Scope {
	clone := s.clone()
	if ident == "" {
		return s
	}
	clone.data = clone.data.With(ident, &scopeVal{p, val})
	return clone
}

/* Test is there appears to be a cycle in the parse trace where no progress has been made.

 */
func (s Scope) isCycle() bool {
	testCycle := func(front, back []tracepoint) bool {
		for i := range front {
			if front[i].offset != back[i].offset ||
				front[i].detail.Id != back[i].detail.Id {
				return false
			}
		}
		return true
	}

	for testSize := len(s.trace) / 2; testSize > 1; testSize-- {
		front := s.trace[len(s.trace)-testSize*2 : len(s.trace)-testSize]
		back := s.trace[len(s.trace)-testSize:]
		if testCycle(front, back) {
			return true
		}
	}
	return false
}
