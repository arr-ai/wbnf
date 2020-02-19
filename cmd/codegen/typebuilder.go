package codegen

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/iancoleman/strcase"

	"github.com/arr-ai/wbnf/ast"
	"github.com/arr-ai/wbnf/wbnf"
)

type typeBuilder struct {
	types map[string]grammarType // new types created in this builder
	deps  map[string]struct{}    // types which are being used by the type in this builder but not found

	prefix   string
	children map[string]grammarType
}

func MakeTypesForTerms(prefix string, term wbnf.TermNode) (map[string]grammarType, []string) {
	t := &typeBuilder{
		types:    map[string]grammarType{},
		deps:     map[string]struct{}{},
		children: map[string]grammarType{},
		prefix:   prefix,
	}
	wbnf.WalkTermNode(term, wbnf.WalkerOps{EnterTermNode: t.handleTerm})
	deps := make([]string, 0, len(t.deps))
	for d := range t.deps {
		deps = append(deps, d)
	}

	var val grammarType
	switch len(t.children) {
	case 0:
	case 1:
		if v, ok := t.getChildren()[0].(unnamedToken); ok && v.count == wantOneGetter {
			t.types[GoTypeName(prefix)] = basicRule(GoTypeName(prefix))
			break
		}
		fallthrough
	default:
		val = rule{name: GoTypeName(prefix), childs: t.getChildren()}
		t.types[val.TypeName()] = val
	}
	return t.types, deps
}

func (t typeBuilder) getChildren() []grammarType {
	out := make([]grammarType, 0, len(t.children))
	for _, c := range t.children {
		out = append(out, c)
	}
	return out
}

func fixCount(old, new int) int {
	if old&wantOneGetter != 0 {
		new = wantAllGetter
	}
	if (old&wantAllGetter != 0) && (new&wantOneGetter != 0) {
		new = old
	}
	if new == (wantOneGetter | wantAllGetter) {
		new = wantAllGetter
	}
	return new
}

func (t *typeBuilder) makeMultiOrFixName(name string, expected grammarType) grammarType {
	if val, has := t.children[name]; has {
		if reflect.TypeOf(val) == reflect.TypeOf(expected) {
			switch t := val.(type) {
			case namedToken:
				t.count = fixCount(t.count, expected.(namedToken).count)
				return t
			case unnamedToken:
				t.count = fixCount(t.count, expected.(unnamedToken).count)
				return t
			case namedRule:
				if t.returnType == expected.(namedRule).returnType {
					t.count = fixCount(t.count, expected.(namedRule).count)
					return t
				}
			}
		}
		for i := 1; ; i++ {
			name := fmt.Sprintf("%s%d", name, i)
			if _, has := t.children[name]; !has {
				switch t := expected.(type) {
				case namedToken:
					t.name = name
					expected = t
				case namedRule:
					t.name = name
					expected = t
				}
				return expected
			}
		}
	}
	return expected
}

func countFromQuant(quants []wbnf.QuantNode) int {
	count := wantOneGetter
	for _, q := range quants {
		switch q.OneOp() {
		case "*", "+":
			count = wantAllGetter
		case "?":
			count |= wantOneGetter
		}
		if q.Choice() != 0 {
			count = wantAllGetter
		}
	}
	return count
}

func (t *typeBuilder) handleTerm(term wbnf.TermNode) wbnf.Stopper {
	if term.OneOp() == "|" {
		t.children[ast.ChoiceTag] = choice(t.prefix)
	}

	if named := term.OneNamed(); named.Node != nil {
		quant := countFromQuant(term.AllQuant())
		name := named.OneIdent().String()
		target, targetType := nameFromAtom(named.OneAtom())

		var child grammarType

		if target == "@" {
			target = t.prefix // FIXME
		}
		if name == "" {
			name = target
		}

		switch targetType {
		case termTarget:
			childTerms := term.OneNamed().OneAtom().AllTerm()
			for i, child := range childTerms {
				prefix := t.prefix + strcase.ToCamel(name)
				typename := prefix
				if i > 0 {
					typename += strconv.Itoa(i)
				}
				newtypes, deps := MakeTypesForTerms(prefix, child)
				for k, v := range newtypes {
					t.types[k] = v
				}
				for _, d := range deps {
					t.deps[d] = struct{}{}
				}
				child := t.makeMultiOrFixName(typename, namedRule{name: name, parent: t.prefix, returnType: GoTypeName(typename)})
				t.children[child.(namedRule).returnType] = child
			}
			return wbnf.NodeExiter
		case tokenTarget:
			if name == "" {
				name = "Token"
				child = unnamedToken{parent: t.prefix, count: quant}
			} else {
				child = namedToken{name: name, parent: t.prefix, count: quant}
			}

		case ruleTarget:
			if name == "" {
				name = target
			}
			child = namedRule{name: name, parent: t.prefix, count: quant, returnType: GoTypeName(target)}
		}

		if child != nil {
			t.children[name] = t.makeMultiOrFixName(name, child)
		}
	}
	return nil
}
