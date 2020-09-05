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
	return Scope{m: s.m.With(ident, v)}
}

func (s Scope) Has(ident string) bool {
	return s.m.Has(ident)
}

func (s Scope) Merge(t Scope) Scope {
	return Scope{m: s.m.Update(t.m)}
}

type scopeBuilder struct {
	mb frozen.MapBuilder
}

func (b *scopeBuilder) Put(ident string, v interface{}) {
	b.mb.Put(ident, v)
}

func (b *scopeBuilder) Finish() Scope {
	return Scope{m: b.mb.Finish()}
}

type scopeVal struct {
	p   Parser
	val TreeElement
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

type Cutpointdata int32

func (c Cutpointdata) valid() bool { return c >= 0 }

const invalidCutpoint = Cutpointdata(-1)

func (s Scope) ReplaceCutPoint(force bool) (newScope Scope, prev, replacement Cutpointdata) {
	prev = s.GetCutPoint()
	replacement = invalidCutpoint
	if prev.valid() || force {
		// TODO: What's with the random number?
		replacement = Cutpointdata(rand.Int31()) //nolint:gosec
		return s.With(cutpointkey, replacement), prev, replacement
	}
	return s, invalidCutpoint, invalidCutpoint
}

func (s Scope) GetCutPoint() Cutpointdata {
	return s.m.GetElse(cutpointkey, invalidCutpoint).(Cutpointdata)
}

const externalsKey = ".Externals-key."
const parseEscapeKey = ".ParseEscape-key."

func (s Scope) WithExternals(extRefs ExternalRefs) Scope {
	var e *escape
	var sb scopeBuilder
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
			sb.Put(parseEscapeKey, e)
		}
	}
	sb.Put(externalsKey, extRefs)
	return s.Merge(sb.Finish())
}

func (s Scope) GetExternal(ident string) ExternalRef {
	if e, has := s.m.GetElse(externalsKey, ExternalRefs{}).(ExternalRefs)[ident]; has {
		return e
	}
	return nil
}

func (s Scope) getParserEscape() *escape {
	if e, has := s.m.Get(parseEscapeKey); has {
		return e.(*escape)
	}
	return nil
}

type call struct {
	ident string
	term  Term
	next  *call
}

func (c *call) Error() string {
	var parts []string
	for ; c != nil; c = c.next {
		parts = append(parts, fmt.Sprintf("%+v", *c))
	}
	return strings.Join(parts, "\n")
}

func (c *call) push(ident string, t Term) *call {
	return &call{ident, t, c}
}
