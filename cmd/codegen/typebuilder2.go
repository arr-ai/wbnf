package codegen

import (
	"github.com/arr-ai/frozen"
	"github.com/arr-ai/wbnf/parser"
	"github.com/iancoleman/strcase"
)

func MakeTypesFromGrammar(g parser.Grammar) map[string]grammarType {
	tm := &TypeMap{}
	return tm.walkGrammar("", g, frozen.NewMap())
}

type TypeMap map[string]grammarType

func (t *TypeMap) pushType(name string, children []grammarType) grammarType {
	if children == nil {
		return nil
	}
	if _, has := (*t)[name]; has {
		panic("oops")
	}
	var val grammarType
	val = rule{name: GoTypeName(name), childs: children}
	if len(children) == 1 {
		if v, ok := children[0].(unnamedToken); ok && v.count == wantOneGetter {
			val = basicRule(GoTypeName(name))
		}
	}
	(*t)[val.TypeName()] = val
	return val
}

func (t *TypeMap) merge(other TypeMap) TypeMap {
	for k, v := range other {
		(*t)[k] = v
	}
	return *t
}

func mergeGrammarRules(prefix string, g parser.Grammar, knownRules frozen.Map) frozen.Map {
	mb := frozen.NewMapBuilder(len(g))
	for k := range g {
		mb.Put(k.String(), prefix+strcase.ToCamel(k.String()))
	}
	return knownRules.Update(mb.Finish())
}

func (tm *TypeMap) walkGrammar(prefix string, g parser.Grammar, knownRules frozen.Map) TypeMap {
	knownRules = mergeGrammarRules(prefix, g, knownRules)

	result := map[string]grammarType{}
	for r, term := range g {
		typeName := prefix + strcase.ToCamel(r.String())
		children := tm.walkTerm(term, typeName, wantOneGetter, knownRules)

		if children != nil {
			tm.pushType(typeName, children)
		}
	}

	return tm.merge(result)
}

func (tm *TypeMap) walkTerm(term parser.Term, parentName string, quant int, knownRules frozen.Map) []grammarType {
	switch t := term.(type) {
	case parser.Rule:
		return []grammarType{namedRule{
			name:       t.String(),
			parent:     GoTypeName(parentName),
			returnType: GoTypeName(knownRules.MustGet(t.String()).(string)),
			count:      quant,
		}}
	case parser.S, parser.RE:
		return []grammarType{unnamedToken{GoTypeName(parentName), quant}}
	case parser.REF:
	case parser.ScopedGrammar:
		knownRules = mergeGrammarRules(parentName, t.Grammar, knownRules)
		scoped := tm.walkGrammar(parentName, t.Grammar, knownRules)
		scoped.pushType(parentName, scoped.walkTerm(t.Term, parentName, quant, knownRules))
		*tm = tm.merge(scoped)
	case parser.Seq:
	case parser.Oneof:
	case parser.Stack:
	case parser.Delim:
		tm.pushType(parentName, tm.walkTerm(t.Term, parentName, wantAllGetter, knownRules))
		switch delim := t.Sep.(type) {
		case parser.Named:
			childName := parentName + strcase.ToCamel(DropCaps(delim.Name))
			if _, ok := delim.Term.(parser.S); ok {
				tm.pushType(childName, []grammarType{unnamedToken{childName, quant}})
			} else {
				tm.pushType(childName, tm.walkTerm(t.Sep, childName, wantAllGetter, knownRules))
			}
		case parser.Rule:
			childName := parentName + strcase.ToCamel(DropCaps(delim.String()))
			tm.pushType(childName, tm.walkTerm(t.Sep, childName, wantAllGetter, knownRules))
		default:
			childName := parentName + "Delim"
			tm.pushType(childName, tm.walkTerm(t.Sep, childName, wantAllGetter, knownRules))
		}
	case parser.Named:
		childName := parentName + strcase.ToCamel(DropCaps(t.Name))
		tm.pushType(childName, tm.walkTerm(t.Term, childName, wantAllGetter, knownRules))
	case parser.Quant:
		if quant == wantAllGetter {
			tm.pushType(parentName, tm.walkTerm(t.Term, parentName, wantAllGetter, knownRules))
		} else {
			if t.Min == 0 && t.Max == 1 {
				quant = wantOneGetter
			} else {
				quant = wantAllGetter
			}
			tm.pushType(parentName, tm.walkTerm(t.Term, parentName, quant, knownRules))
		}
	default:
		panic("unknown type")
	}
	return nil
}
