package parser

import "github.com/arr-ai/wbnf/errors"

func (t Rule) Resolve(oldRule, newRule Rule) Term {
	if t == oldRule {
		return newRule
	}
	return t
}

func (t S) Resolve(oldRule, newRule Rule) Term {
	return t
}

func (t RE) Resolve(oldRule, newRule Rule) Term {
	return t
}
func (t REF) Resolve(oldRule, newRule Rule) Term {
	return t
}

func (t Seq) Resolve(oldRule, newRule Rule) Term {
	result := make(Seq, 0, len(t))
	for _, term := range t {
		result = append(result, term.Resolve(oldRule, newRule))
	}
	return result
}

func (t Oneof) Resolve(oldRule, newRule Rule) Term {
	result := make(Oneof, 0, len(t))
	for _, term := range t {
		result = append(result, term.Resolve(oldRule, newRule))
	}
	return result
}

func (t Stack) Resolve(oldRule, newRule Rule) Term {
	panic(errors.Inconceivable)
}

func (t Delim) Resolve(oldRule, newRule Rule) Term {
	t.Term = t.Term.Resolve(oldRule, newRule)
	t.Sep = t.Sep.Resolve(oldRule, newRule)
	return t
}

func (t Quant) Resolve(oldRule, newRule Rule) Term {
	t.Term = t.Term.Resolve(oldRule, newRule)
	return t
}

func (t Named) Resolve(oldRule, newRule Rule) Term {
	t.Term = t.Term.Resolve(oldRule, newRule)
	return t
}

func (t ScopedGrammar) Resolve(oldRule, newRule Rule) Term {
	panic(errors.Inconceivable)
}
