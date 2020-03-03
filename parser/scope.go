package parser

import (
	"math/rand"

	"github.com/arr-ai/frozen"
)

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

func (s Scope) WithExternals(e ExternalRefs) Scope {
	return s.With(externalsKey, e)
}

func (s Scope) GetExternal(ident string) ExternalRef {
	if e, has := s.m.GetElse(externalsKey, ExternalRefs{}).(ExternalRefs)[ident]; has {
		return e
	}
	return nil
}
