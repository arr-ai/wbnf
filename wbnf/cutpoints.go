package wbnf

import (
	"fmt"

	"github.com/arr-ai/frozen"
	"github.com/arr-ai/wbnf/parser"
)

/* Attempt to find and insert cutpoints into the generated parser.Grammar

What is a cutpoint?
	A point in a rule which if the parse was successful past this point but fails at any point later the whole
	parse should fail instead of trying different branches.

	example:    (using the symbol ! to denote a cutpoint)
			a -> "hello" ! c;
		If the token "hello" was parsed successfully then if the parse in `c` fails the whole parse should abort.

			a ->  ("hello" ! z) | "goodbye";

		In the above example without a cutpoint a failure to parse the `z` rule would cause the parser to backtrack
		and attempt to then parse `goodbye`. With the cutpoint a failure to parse `z` would abort and not attempt
		backtracking.


Where can cutpoints be added?
	At any point where it is guaranteed that no other branch could possibly successfully parse the previously
	consumed text. This will have to be heuristically determined.

What does this code attempt to add?
	1) Any S() token which appears only once in the entire grammar can be safely considered a cutpoint
	2) .... TBD

The intention of this file is to provide good-enough cutpoints that it is not necessary for grammar authors
to add their own

*/

func insertCutPoints(g parser.Grammar) parser.Grammar {
	strings := findUniqueStrings(g)
	s := strings.Elements()
	_ = s
	var callback func(t parser.Term) parser.Term
	callback = func(t parser.Term) parser.Term {
		switch t := t.(type) {
		case parser.S:
			if strings.Has(string(t)) {
				return parser.CutPoint{Term: t}
			}
		case parser.ScopedGrammar:
			t.Grammar = rebuildGrammar(t.Grammar, callback)
			return t
		}
		return t
	}
	return rebuildGrammar(g, callback)
}

func findUniqueStrings(g parser.Grammar) frozen.Set {
	mergeFn := func(_ interface{}, a, b interface{}) interface{} {
		return a.(int) + b.(int)
	}
	var forTerm func(t parser.Term) frozen.Map
	forTerm = func(t parser.Term) frozen.Map {
		out := frozen.NewMap()
		if t == nil {
			return out
		}
		switch t := t.(type) {
		case parser.S:
			return out.With(string(t), 1)
		case parser.Seq:
			for _, t := range t {
				out = out.Merge(forTerm(t), mergeFn)
			}
		case parser.Stack:
			for _, t := range t {
				out = out.Merge(forTerm(t), mergeFn)
			}
		case parser.Oneof:
			for _, t := range t {
				out = out.Merge(forTerm(t), mergeFn)
			}
		case parser.Delim:
			out = out.Merge(forTerm(t.Term), mergeFn)
			out = out.Merge(forTerm(t.Sep), mergeFn)
			if t.CanStartWithSep {
				// ensure that a starting sep will never be a cutpoint
				out = out.Merge(forTerm(t.Sep), mergeFn)
			}
		case parser.Quant:
			out = out.Merge(forTerm(t.Term), mergeFn)
		case parser.Named:
			out = out.Merge(forTerm(t.Term), mergeFn)
		case parser.ScopedGrammar:
			out = out.Merge(forTerm(t.Term), mergeFn)
			incoming := frozen.NewMapFromKeys(findUniqueStrings(t.Grammar), func(key interface{}) interface{} {
				return 1
			})
			out = out.Merge(incoming, mergeFn)
		case parser.REF:
			out = out.Merge(forTerm(t.Default), mergeFn)
		case parser.CutPoint:
			out = out.Merge(forTerm(t.Term), mergeFn)
		case parser.RE, parser.Rule, parser.ExtRef: // do nothing
		default:
			panic(fmt.Errorf("findUniqueStrings: unexpected term type: %v %[1]T", t))
		}
		return out
	}

	strings := frozen.NewMap()
	for _, t := range g {
		strings = strings.Merge(forTerm(t), mergeFn)
	}
	return strings.Where(func(key, val interface{}) bool {
		return val.(int) == 1
	}).Keys()
}

func rebuildGrammar(input parser.Grammar, callback func(t parser.Term) parser.Term) parser.Grammar {
	out := parser.Grammar{}
	for rule, term := range input {
		out[rule] = fixTerm(term, callback)
	}
	return out
}

func fixTerm(term parser.Term, callback func(t parser.Term) parser.Term) parser.Term {
	switch t := term.(type) {
	case parser.Seq:
		out := parser.Seq{}
		for _, t := range t {
			out = append(out, fixTerm(t, callback))
		}
		return out
	case parser.Stack:
		out := parser.Stack{}
		for _, t := range t {
			out = append(out, fixTerm(t, callback))
		}
		return callback(out)
	case parser.Oneof:
		out := parser.Oneof{}
		for _, t := range t {
			out = append(out, fixTerm(t, callback))
		}
		return callback(out)
	case parser.Delim:
		t.Term = fixTerm(t.Term, callback)
		t.Sep = fixTerm(t.Sep, callback)
		return callback(t)
	case parser.Quant:
		t.Term = fixTerm(t.Term, callback)
		return callback(t)
	case parser.Named:
		t.Term = fixTerm(t.Term, callback)
		return callback(t)
	case parser.ScopedGrammar:
		t.Term = fixTerm(t.Term, callback)
		return callback(t)
	case parser.CutPoint:
		t.Term = fixTerm(t.Term, callback)
		return callback(t)
	case parser.S, parser.REF, parser.RE, parser.Rule, parser.ExtRef:
		return callback(term)
	default:
		panic(fmt.Errorf("fixTerm: unexpected term type: %v %[1]T", t))
	}
}
