package codegen

import (
	"math/rand"

	"github.com/arr-ai/frozen"
	"github.com/arr-ai/wbnf/parser"
	"github.com/iancoleman/strcase"
)

func MakeTypesFromGrammar(g parser.Grammar) map[string]grammarType {
	tm := &TypeMap{}
	return tm.walkGrammar("", g, mergeGrammarRules("", g, frozen.NewMap()))
}

type TypeMap map[string]grammarType

func (tm *TypeMap) merge(other TypeMap) TypeMap {
	for k, v := range other {
		(*tm)[k] = v
	}
	return *tm
}

type stackInfo struct{ ident, parentName string }

func pushRuleNameForStack(ident, parentName string, knownRules frozen.Map) frozen.Map {
	return knownRules.With(parser.At, stackInfo{
		ident:      ident,
		parentName: parentName,
	})
}

func mergeGrammarRules(prefix string, g parser.Grammar, knownRules frozen.Map) frozen.Map {
	mb := frozen.NewMapBuilder(len(g))
	for k := range g {
		mb.Put(k.String(), prefix+strcase.ToCamel(k.String()))
	}
	return knownRules.Update(mb.Finish())
}

func (tm *TypeMap) walkGrammar(prefix string, g parser.Grammar, knownRules frozen.Map) TypeMap {
	result := map[string]grammarType{}
	for r, term := range g {
		typeName := prefix + strcase.ToCamel(r.String())
		tm.walkTerm(term, typeName, setWantOneGetter(), pushRuleNameForStack(r.String(), typeName, knownRules), rand.Int())
	}

	return tm.merge(result)
}

func (tm *TypeMap) handleSeq(terms []parser.Term, parentName string, quant countManager, knownRules frozen.Map, termId int) {
	for _, t := range terms {
		tm.walkTerm(t, parentName, quant, knownRules, termId)
	}
}

func (tm *TypeMap) makeLeafType(term parser.Term, parentName string, quant countManager, knownRules frozen.Map) {
	var val grammarType
	switch t := term.(type) {
	case parser.Rule:
		if t == parser.At {
			si := knownRules.MustGet(t).(stackInfo)
			val = stackBackRef{
				name:   si.ident,
				parent: si.parentName,
			}
		} else {
			val = namedRule{
				name:       t.String(),
				parent:     GoTypeName(parentName),
				returnType: GoTypeName(knownRules.MustGet(t.String()).(string)),
				count:      quant,
			}
		}
	case parser.S, parser.RE:
		val = unnamedToken{GoTypeName(parentName), quant}
	default:
		panic("Should not have got here")
	}
	tm.pushType("", parentName, val)
}

func (tm *TypeMap) walkTerm(term parser.Term, parentName string, quant countManager, knownRules frozen.Map, termId int) {
	switch t := term.(type) {
	case parser.S, parser.RE, parser.Rule:
		tm.makeLeafType(term, parentName, quant.pushSingleNode(termId), knownRules)
	case parser.REF:
		tm.pushType("", parentName, backRef{
			name:   t.Ident,
			parent: GoTypeName(parentName),
		})
	case parser.ScopedGrammar:
		knownRules = mergeGrammarRules(parentName, t.Grammar, knownRules)
		scoped := tm.walkGrammar(parentName, t.Grammar, knownRules)
		scoped.walkTerm(t.Term, parentName, quant, knownRules, termId)
		*tm = tm.merge(scoped)
	case parser.Seq:
		tm.handleSeq(t, parentName, quant, knownRules, termId)
	case parser.Oneof:
		tm.pushType("", parentName, choice{parent: parentName})
		for _, t := range t {
			tm.walkTerm(t, parentName, quant, knownRules, rand.Int())
		}
	case parser.Stack:
		tm.handleSeq(t, parentName, quant, knownRules, termId)
	case parser.Delim:
		tm.walkTerm(t.Term, parentName, setWantAllGetter(), knownRules, termId)
		switch delim := t.Sep.(type) {
		case parser.Named:
			childName := parentName + strcase.ToCamel(DropCaps(delim.Name))
			if _, ok := delim.Term.(parser.S); ok {
				tm.pushType(childName, parentName, namedToken{
					name:   delim.Name,
					parent: parentName,
					count:  quant,
				})
			} else {
				tm.walkTerm(t.Sep, parentName, setWantAllGetter(), knownRules, termId)
			}
		case parser.Rule:
			childName := parentName + strcase.ToCamel(DropCaps(delim.String()))
			tm.walkTerm(t.Sep, childName, setWantAllGetter(), knownRules, termId)
		case parser.S: // ignore the delim
		default:
			childName := parentName + "Delim"
			tm.walkTerm(t.Sep, childName, setWantAllGetter(), knownRules, termId)
		}
	case parser.Named:
		childName := parentName + strcase.ToCamel(DropCaps(t.Name))
		switch term := t.Term.(type) {
		case parser.Rule:
			tm.pushType(childName, parentName, namedRule{
				name:       t.Name,
				parent:     parentName,
				returnType: GoTypeName(term.String()),
				count:      quant,
			})
		case parser.RE, parser.S:
			tm.pushType(childName, parentName, namedToken{
				name:   t.Name,
				parent: parentName,
				count:  quant,
			})
		default:
			tm.walkTerm(t.Term, childName, quant, knownRules, termId)
			tm.pushType(childName, parentName, namedRule{
				name:       t.Name,
				parent:     parentName,
				returnType: GoTypeName(childName),
				count:      quant,
			})
		}
	case parser.Quant:
		if t.Max != 1 {
			quant = setWantAllGetter()
		}
		tm.walkTerm(t.Term, parentName, quant, knownRules, termId)
	default:
		panic("unknown type")
	}
}
