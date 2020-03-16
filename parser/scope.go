package parser

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"

	"github.com/arr-ai/frozen"
)

type escape struct {
	openDelim  *regexp.Regexp
	closeDelim *regexp.Regexp
	external   ExternalRef
}
type Scope struct {
	m frozen.Map
}

func (s Scope) String() string {
	return s.m.String()
}

func (s Scope) Keys() frozen.Set {
	return s.m.Keys()
}

func (s Scope) With(ident string, v interface{}) Scope {
	s.m = s.m.With(ident, v)
	return s
}

func (s Scope) Has(ident string) bool {
	return s.m.Has(ident)
}

type scopeVal struct {
	p   Parser
	val TreeElement
	cp  cutpointdata
}

func (s Scope) WithVal(ident string, p Parser, val TreeElement) Scope {
	if ident != "" {
		s.m = s.m.With(ident, &scopeVal{p: p, val: val})
	}
	return s
}

func (s Scope) GetVal(ident string) (Parser, TreeElement, bool) {
	if val, ok := s.m.Get(ident); ok {
		sv := val.(*scopeVal)
		return sv.p, sv.val, ok
	}
	return nil, nil, false
}

const cutpointkey = ".Cutpoint-key."

type cutpointdata int32

func (c cutpointdata) valid() bool { return c >= 0 }

const invalidCutpoint = cutpointdata(-1)

func (s Scope) ReplaceCutPoint(force bool) (newScope Scope, prev, replacement cutpointdata) {
	prev = s.GetCutPoint()
	replacement = invalidCutpoint
	if prev.valid() || force {
		replacement = cutpointdata(rand.Int31())
		return s.With(cutpointkey, replacement), prev, replacement
	}
	return s, invalidCutpoint, invalidCutpoint
}

func (s Scope) GetCutPoint() cutpointdata {
	return s.m.GetElse(cutpointkey, invalidCutpoint).(cutpointdata)
}

const externalsKey = ".Externals-key."
const parseEscapeKey = ".ParseEscape-key."

func (s Scope) WithExternals(extRefs ExternalRefs) Scope {
	var e *escape
	for name, external := range extRefs {
		if strings.HasPrefix(name, "*") {
			if e != nil {
				panic(fmt.Errorf("too many escapes"))
			}
			openClose := strings.Split(name[1:], "()")
			e = &escape{
				openDelim:  regexp.MustCompile(`(?m)\A` + openClose[0]),
				closeDelim: regexp.MustCompile(`(?m)\A` + openClose[1]),
				external:   external,
			}
			s = s.With(parseEscapeKey, e)
		}
	}
	return s.With(externalsKey, extRefs)
}

func (s Scope) GetExternal(ident string) ExternalRef {
	if e, has := s.m.GetElse(externalsKey, ExternalRefs{}).(ExternalRefs)[ident]; has {
		return e
	}
	return nil
}

func (s Scope) GetParserEscape() *escape {
	if e, has := s.m.Get(parseEscapeKey); has {
		return e.(*escape)
	}
	return nil
}

type call struct {
	ident string
	term  Term
	scope Scope
}
type CallStack struct {
	stack []call
}

func (c CallStack) Error() string {
	var parts []string
	for _, call := range c.stack {
		parts = append(parts, fmt.Sprintf("%+v", call))
	}
	return strings.Join(parts, "\n")
}

const callStackKey = ".CallStack-key."

func (s Scope) PushCall(ident string, t Term) Scope {
	cs := s.GetCallStack()
	cs.stack = append(cs.stack, call{ident, t, s})
	return s.With(callStackKey, cs)
}

func (s Scope) GetCallStack() CallStack {
	if e, has := s.m.Get(callStackKey); has {
		return e.(CallStack)
	}
	return CallStack{}
}
