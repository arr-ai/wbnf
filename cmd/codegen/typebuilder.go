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

	prefix   string
	ident    string
	children map[string]grammarType
}

func MakeTypesForTerms(prefix string, ident string, term wbnf.TermNode) map[string]grammarType {
	t := &typeBuilder{
		types:    map[string]grammarType{},
		children: map[string]grammarType{},
		prefix:   prefix,
		ident:    ident,
	}
	wbnf.WalkTermNode(term, wbnf.WalkerOps{EnterTermNode: t.handleTerm})

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
	return t.types
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

func (t *typeBuilder) handleNamed(named wbnf.NamedNode, quant int) wbnf.Stopper {
	name := named.OneIdent().String()
	target, targetType := nameFromAtom(named.OneAtom())

	var child grammarType

	if name == "" {
		name = target
	}

	switch targetType {
	case termTarget:
		childTerms := named.OneAtom().AllTerm()
		for i, child := range childTerms {
			prefix := t.prefix + strcase.ToCamel(name)
			typename := prefix
			if i > 0 {
				typename += strconv.Itoa(i)
			}
			newtypes := MakeTypesForTerms(prefix, name, child)
			for k, v := range newtypes {
				t.types[k] = v
			}
			child := t.makeMultiOrFixName(typename, namedRule{name: name, parent: t.prefix, count: quant, returnType: GoTypeName(typename)})
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
		if target == "@" {
			child = namedRule{name: t.ident, parent: t.prefix, count: quant, returnType: GoTypeName(t.prefix)}
		} else {
			child = namedRule{name: name, parent: t.prefix, count: quant, returnType: GoTypeName(DropCaps(target))}
		}
	}

	if child != nil {
		t.children[name] = t.makeMultiOrFixName(name, child)
	}
	return nil
}

func (t *typeBuilder) handleTerm(term wbnf.TermNode) wbnf.Stopper {
	if term.OneOp() == "|" {
		t.children[ast.ChoiceTag] = choice(t.prefix)
	}

	// And catch any terms from the quants
	for _, q := range term.AllQuant() {
		wbnf.WalkQuantNode(q, wbnf.WalkerOps{EnterNamedNode: func(node wbnf.NamedNode) wbnf.Stopper {
			t.handleNamed(node, wantOneGetter)
			return nil
		}})
	}

	if named := term.OneNamed(); named.Node != nil {
		quant := countFromQuant(term.AllQuant())
		return t.handleNamed(named, quant)
	}

	return nil
}
