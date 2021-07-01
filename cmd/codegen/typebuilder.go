package codegen

import (
	"math/rand"

	"github.com/arr-ai/frozen"
	"github.com/arr-ai/wbnf/parser"
)

func makeTypesFromGrammar(g parser.Grammar) map[string]GrammarType {
	tm := &TypeMap{}
	return tm.walkGrammar("", g, mergeGrammarRules("", g, frozen.NewMap()))
}

type TypeMap map[string]GrammarType

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
		mb.Put(k.String(), prefix+GoName(k.String()))
	}
	return knownRules.Update(mb.Finish())
}

func (tm *TypeMap) walkGrammar(prefix string, g parser.Grammar, knownRules frozen.Map) TypeMap {
	result := map[string]GrammarType{}
	for r, term := range g {
		typeName := prefix + GoName(r.String())
		tm.walkTerm(term, typeName, setWantOneGetter(),
			pushRuleNameForStack(r.String(), typeName, knownRules), rand.Int()) //nolint:gosec
		// Now we need to check if stack terms were used, if they were we need to ensure that unnamed rule child
		// was added, otherwise a rule like `f -> foo=@ > BAR;` would not generate a working walker api
		newT := tm.findType(GoTypeName(typeName))
		var needStackAddition bool
		for _, child := range newT.Children() {
			if x, ok := child.(stackBackRef); ok {
				if x.name == r.String() {
					needStackAddition = false
					break
				}
				needStackAddition = true
			}
		}
		if needStackAddition {
			tm.pushType("", typeName, stackBackRef{name: r.String(), parent: typeName})
		}
	}

	return tm.merge(result)
}

func (tm *TypeMap) handleSeq(
	terms []parser.Term,
	parentName string,
	quant countManager,
	knownRules frozen.Map,
	termID int,
) {
	for _, t := range terms {
		tm.walkTerm(t, parentName, quant, knownRules, termID)
	}
}

func (tm *TypeMap) makeLeafType(term parser.Term, parentName string, quant countManager, knownRules frozen.Map) {
	var val GrammarType
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
				parent:     parentName,
				returnType: knownRules.MustGet(t.String()).(string),
				count:      quant,
			}
		}
	case parser.S, parser.RE:
		val = unnamedToken{parentName, quant}
	default:
		panic("Should not have got here")
	}
	tm.pushType("", parentName, val)
}

func (tm *TypeMap) walkTerm(
	term parser.Term,
	parentName string,
	quant countManager,
	knownRules frozen.Map,
	termID int,
) {
	switch t := term.(type) {
	case parser.S, parser.RE, parser.Rule:
		tm.makeLeafType(term, parentName, quant.pushSingleNode(termID), knownRules)
	case parser.REF:
		tm.pushType("", parentName, backRef{
			name:   t.Ident,
			parent: parentName,
		})
	case parser.ScopedGrammar:
		knownRules = mergeGrammarRules(parentName, t.Grammar, knownRules)
		scoped := tm.walkGrammar(parentName, t.Grammar, knownRules)
		scoped.walkTerm(t.Term, parentName, quant, knownRules, termID)
		*tm = tm.merge(scoped)
	case parser.Seq:
		tm.handleSeq(t, parentName, quant, knownRules, termID)
	case parser.Oneof:
		tm.pushType("", parentName, choice{parent: parentName})
		for _, t := range t {
			tm.walkTerm(t, parentName, quant, knownRules, rand.Int()) //nolint:gosec
		}
	case parser.Stack:
		tm.handleSeq(t, parentName, quant, knownRules, termID)
	case parser.Delim:
		tm.walkTerm(t.Term, parentName, setWantAllGetter(), knownRules, termID)
		switch delim := t.Sep.(type) {
		case parser.Named:
			childName := parentName + GoName(delim.Name)
			switch delim.Term.(type) {
			case parser.S, parser.CutPoint: //fixme: This will only work as long as cutpoints are s() only
				tm.pushType(childName, parentName, namedToken{
					name:   delim.Name,
					parent: parentName,
					count:  quant,
				})
			default:
				tm.walkTerm(t.Sep, parentName, setWantAllGetter(), knownRules, termID)
			}
		case parser.Rule:
			childName := parentName + GoName(delim.String())
			tm.walkTerm(t.Sep, childName, setWantAllGetter(), knownRules, termID)
		case parser.CutPoint, parser.S: // ignore the delim
		default:
			childName := parentName + "Delim"
			tm.walkTerm(t.Sep, childName, setWantAllGetter(), knownRules, termID)
		}
	case parser.Named:
		childName := parentName + GoName(t.Name)
		switch term := t.Term.(type) {
		case parser.Rule:
			var val GrammarType
			if term == parser.At {
				si := knownRules.MustGet(term).(stackInfo)
				val = stackBackRef{
					name:   t.Name,
					parent: si.parentName,
				}
			} else {
				val = namedRule{
					name:       t.Name,
					parent:     parentName,
					returnType: term.String(),
					count:      quant,
				}
			}
			tm.pushType(childName, parentName, val)
		case parser.RE, parser.S, parser.CutPoint:
			tm.pushType(childName, parentName, namedToken{
				name:   t.Name,
				parent: parentName,
				count:  quant,
			})
		default:
			tm.walkTerm(t.Term, childName, quant, knownRules, termID)
			tm.pushType(childName, parentName, namedRule{
				name:       t.Name,
				parent:     parentName,
				returnType: childName,
				count:      quant,
			})
		}
	case parser.Quant:
		if t.Max != 1 {
			quant = setWantAllGetter()
		}
		tm.walkTerm(t.Term, parentName, quant, knownRules, termID)
	case parser.CutPoint:
		tm.walkTerm(t.Term, parentName, quant, knownRules, termID)
	case parser.LookAhead:
		tm.walkTerm(t.Term, parentName, quant, knownRules, termID)
	case parser.ExtRef:
		// nothing yet
	default:
		panic("unknown type")
	}
}
