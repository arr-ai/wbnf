package wbnf

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/arr-ai/wbnf/parser"
)

func findDefinedRules(tree GrammarNode) (map[string]struct{}, error) {
	var dupeRules []string
	out := map[string]struct{}{}
	ops := WalkerOps{
		EnterProdNode: func(node ProdNode) Stopper {
			ident := node.OneIdent().String()
			if _, has := out[ident]; has {
				dupeRules = append(dupeRules, ident)
			}
			out[ident] = struct{}{}
			return &nodeExiter{}
		},
	}
	ops.Walk(tree)
	if len(dupeRules) == 0 {
		return out, nil
	}
	return nil, validationError{msg: "the following rule(s) are defined multiple times: %s",
		kind: DuplicatedRule, args: []interface{}{dupeRules}}
}

func validate(tree GrammarNode) error {
	rules, err := findDefinedRules(tree)
	if err != nil {
		return err
	}
	v := validator{
		knownRules: rules,
	}

	ops := WalkerOps{
		EnterAtomNode:  v.validateAtom,
		EnterQuantNode: v.validateQuant,
		EnterNamedNode: v.validateNamed,
		EnterTermNode:  v.validateTerm,
	}
	ops.Walk(tree)

	if len(v.err) == 0 {
		return nil
	}
	return &v
}

type validationErrorKind int

const (
	NoError validationErrorKind = iota
	UnknownRule
	DuplicatedRule
	InvalidRegex
	NameClashesWithRule
	MinMaxQuantError
	MultipleTermsWithSameName // something like `term -> foo op="*" op="|";`, likely missing a separator
)

type validationError struct {
	s    parser.Scanner
	msg  string
	args []interface{}
	kind validationErrorKind
}

func (v validationError) Error() string {
	var args []interface{}
	args = append(args, v.s.String())
	if len(v.args) > 0 {
		args = append(args, v.args...)
	}
	args = append(args, v.s.Offset())

	return fmt.Sprintf(v.msg+"@ %d", args...)
}

type validator struct {
	knownRules map[string]struct{}
	err        []error
}

func (v *validator) Error() string {
	return fmt.Sprint(v.err)
}

func (v *validator) validateTerm(tree TermNode) Stopper {
	if len(tree.AllGrammar()) != 0 {
		//fixme: This doesnt work for scoped grammars yet, abort!
		return NodeExiter
	}
	if len(tree.AllOp()) == 0 || tree.AllOp()[0] == "" {
		names := map[string]bool{}
		for _, child := range tree.AllTerm() {
			if name := child.OneNamed(); name != nil {
				if x := name.OneIdent(); x != nil {
					if _, has := names[x.String()]; has {
						v.err = append(v.err, validationError{s: name.OneIdent().Scanner(),
							msg: "identifier '%s' is being used multiple times in a single term", kind: MultipleTermsWithSameName})
					}
					names[x.String()] = true
				}
			}
		}
	}
	return nil
}

func (v *validator) validateNamed(tree NamedNode) Stopper {
	if x := tree.OneIdent(); x != nil {
		if _, has := v.knownRules[x.String()]; has {
			v.err = append(v.err, validationError{s: tree.OneIdent().Scanner(),
				msg: "identifier '%s' clashes with a defined rule", kind: NameClashesWithRule})
		}
	}
	return nil
}

func (v *validator) validateAtom(tree AtomNode) Stopper {
	if ident := tree.OneIdent(); ident != nil {
		if ident.String() != "@" {
			if _, has := v.knownRules[ident.String()]; !has {
				v.err = append(v.err, validationError{s: tree.OneIdent().Scanner(),
					msg: "identifier '%s' is not a defined rule", kind: UnknownRule})
			}
		}
	} else if x := tree.OneRe(); x != nil {
		if _, err := regexp.Compile(x.String()); err != nil {
			v.err = append(v.err, validationError{s: tree.OneRe().Scanner(),
				msg: "regex '%s' is not valid, %s", kind: InvalidRegex, args: []interface{}{err}})
		}
	}
	return nil
}

func (v *validator) validateQuant(tree QuantNode) Stopper {
	switch tree.Choice() {
	case 0:
	case 1:
		min := 0
		max := 0
		if x := tree.OneMin().String(); x != "" {
			val, err := strconv.Atoi(x)
			if err != nil {
				panic("should not get here")
			}
			min = val
		}
		if x := tree.OneMax().String(); x != "" {
			val, err := strconv.Atoi(x)
			if err != nil {
				panic("should not get here")
			}
			max = val
		}
		if min != 0 && max != 0 {
			if max < min {
				v.err = append(v.err, validationError{
					msg: "quant: min (%d) > max (%d)", kind: MinMaxQuantError, args: []interface{}{min, max}})
			}
		}
	case 2:
	}
	return nil
}
