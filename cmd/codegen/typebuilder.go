package codegen

import (
	"reflect"

	"github.com/arr-ai/frozen"
	"github.com/arr-ai/wbnf/parser"
	"github.com/iancoleman/strcase"
)

func MakeTypesFromGrammar(g parser.Grammar) map[string]grammarType {
	tm := &TypeMap{}
	return tm.walkGrammar("", g, mergeGrammarRules("", g, frozen.NewMap()))
}

type TypeMap map[string]grammarType

func (t *TypeMap) pushType(name, parent string, child grammarType) grammarType {
	if child == nil {
		return nil
	}

	typename := GoTypeName(name)

	var val grammarType
	val = rule{name: typename, childs: []grammarType{child}}
	if v, ok := child.(unnamedToken); ok && v.count == wantOneGetter {
		val = basicRule(typename)
	}

	if oldval, has := (*t)[typename]; has {
		if reflect.TypeOf(oldval) != reflect.TypeOf(val) {
			switch oldval := oldval.(type) {
			case rule:
				if _, ok := val.(basicRule); !ok {
					panic("cont replace a rule with this type")
				}
			case basicRule:
				switch v := val.(type) {
				case unnamedToken:
				case namedRule:
				case namedToken:
				case rule:
					if oldval.TypeName() == v.TypeName() {
						(*t)[val.TypeName()] = val
						return val
					}
				default:
					panic("This should not have happened")
				}
			default:
				panic("This should not have happened")
			}
		}
		switch oldval := oldval.(type) {
		case basicRule:
			val = rule{name: GoTypeName(name), childs: []grammarType{
				unnamedToken{parent: oldval.TypeName(), count: wantAllGetter}}}
		case rule:
			checkForDupes := func(children []grammarType, next grammarType) []grammarType {
				result := make([]grammarType, 0, len(children)+1)
				appendNext := true
				for _, c := range children {
					switch child := c.(type) {
					case unnamedToken:
						child.count = wantAllGetter
						result = append(result, child)
						appendNext = false
					case namedToken:
						if next.Ident() == child.Ident() {
							child.count = wantAllGetter
							appendNext = false
						}
						result = append(result, child)
					case namedRule:
						if next.Ident() == child.Ident() {
							child.count = wantAllGetter
							appendNext = false
						}
						result = append(result, child)
					case stackBackRef:
						if _, ok := next.(stackBackRef); ok {
							return children
						}
						result = append(result, child)
					default:
						result = append(result, child)
					}
				}
				if appendNext {
					return append(result, next)
				}
				return result
			}
			val = rule{name: GoTypeName(name), childs: checkForDupes(oldval.childs, child)}
		case namedRule:
			newval := val.(namedRule)
			newval.count = wantAllGetter
			val = newval
		case namedToken:
			newval := val.(namedToken)
			newval.count = wantAllGetter
			val = newval
		case choice:
		default:
			panic("oops")
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
		tm.walkTerm(term, typeName, wantOneGetter, pushRuleNameForStack(r.String(), typeName, knownRules))
	}

	return tm.merge(result)
}

func (tm *TypeMap) handleSeq(terms []parser.Term, parentName string, quant int, knownRules frozen.Map) {
	for _, t := range terms {
		tm.walkTerm(t, parentName, quant, knownRules)
	}
}

func (tm *TypeMap) makeLeafType(term parser.Term, parentName string, quant int, knownRules frozen.Map) {
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
	tm.pushType(parentName, "", val)
}

func (tm *TypeMap) walkTerm(term parser.Term, parentName string, quant int, knownRules frozen.Map) {
	switch t := term.(type) {
	case parser.S, parser.RE, parser.Rule:
		tm.makeLeafType(term, parentName, quant, knownRules)
	case parser.REF:
		tm.pushType(parentName, "", backRef{
			name:   t.Ident,
			parent: GoTypeName(parentName),
		})
	case parser.ScopedGrammar:
		knownRules = mergeGrammarRules(parentName, t.Grammar, knownRules)
		scoped := tm.walkGrammar(parentName, t.Grammar, knownRules)
		scoped.walkTerm(t.Term, parentName, quant, knownRules)
		*tm = tm.merge(scoped)
		tm.walkTerm(t.Term, parentName, quant, knownRules)
	case parser.Seq:
		tm.handleSeq(t, parentName, quant, knownRules)
	case parser.Oneof:
		tm.pushType(parentName, "", choice{parent: parentName})
		for _, t := range t {
			tm.walkTerm(t, parentName, quant, knownRules)
		}
	case parser.Stack:
		tm.handleSeq(t, parentName, quant, knownRules)
	case parser.Delim:
		tm.walkTerm(t.Term, parentName, wantAllGetter, knownRules)
		switch delim := t.Sep.(type) {
		case parser.Named:
			childName := parentName + strcase.ToCamel(DropCaps(delim.Name))
			if _, ok := delim.Term.(parser.S); ok {
				tm.pushType(childName, parentName, unnamedToken{childName, quant})
				tm.pushType(parentName, "", namedToken{
					name:   delim.Name,
					parent: parentName,
					count:  quant,
				})
			} else {
				tm.walkTerm(t.Sep, childName, wantAllGetter, knownRules)
			}

		case parser.Rule:
			childName := parentName + strcase.ToCamel(DropCaps(delim.String()))
			tm.walkTerm(t.Sep, childName, wantAllGetter, knownRules)
		default:
			childName := parentName + "Delim"
			tm.walkTerm(t.Sep, childName, wantAllGetter, knownRules)
		}
	case parser.Named:
		childName := parentName + strcase.ToCamel(DropCaps(t.Name))
		switch term := t.Term.(type) {
		case parser.Rule:
			tm.pushType(parentName, "", namedRule{
				name:       t.Name,
				parent:     parentName,
				returnType: GoTypeName(DropCaps(term.String())),
				count:      quant,
			})
		case parser.RE, parser.S:
			tm.pushType(parentName, "", namedToken{
				name:   t.Name,
				parent: parentName,
				count:  quant,
			})
		default:
			tm.walkTerm(t.Term, childName, quant, knownRules)
			tm.pushType(parentName, "", namedRule{
				name:       t.Name,
				parent:     parentName,
				returnType: GoTypeName(childName),
				count:      quant,
			})
		}
	case parser.Quant:
		if quant == wantAllGetter {
			tm.walkTerm(t.Term, parentName, wantAllGetter, knownRules)
		} else {
			if t.Min == 0 && t.Max == 1 {
				quant = wantOneGetter
			} else {
				quant = wantAllGetter
			}
			tm.walkTerm(t.Term, parentName, quant, knownRules)
		}
	default:
		panic("unknown type")
	}
}
