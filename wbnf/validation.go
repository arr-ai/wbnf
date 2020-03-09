package wbnf

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/arr-ai/frozen"

	"github.com/arr-ai/wbnf/parser"
)

func findDefinedRules(tree GrammarNode) (frozen.Set, map[string]PragmaMacrodefNode, error) {
	var dupeRules []string
	out := frozen.NewSet()
	macros := map[string]PragmaMacrodefNode{}
	adder := func(ident string) {
		if out.Has(ident) {
			dupeRules = append(dupeRules, ident)
		}
		out = out.With(ident)
	}
	ops := WalkerOps{
		EnterProdNode: func(node ProdNode) Stopper {
			adder(node.OneIdent().String())
			return NodeExiter
		},
		EnterPragmaMacrodefNode: func(node PragmaMacrodefNode) Stopper {
			ident := node.OneName().String()
			adder(ident)
			macros[ident] = node
			return NodeExiter
		},
	}
	ops.Walk(tree)
	if len(dupeRules) == 0 {
		return out, macros, nil
	}
	return frozen.Set{}, nil, validationError{
		msg:  fmt.Sprintf("the following rule(s) are defined multiple times: %s", dupeRules),
		kind: DuplicatedRule}
}

func validate(tree GrammarNode) error {
	rules, macros, err := findDefinedRules(tree)
	if err != nil {
		return err
	}
	v := validator{
		knownRules: rules,
		macros:     macros,
	}
	v.walk(tree)

	if cycles := checkForRecursion(tree); cycles != nil {
		v.err = append(v.err, cycles)
	}

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
	PossibleCycleDetected
	NotAMacro
	IncorrectMacroArgCount
)

type validationError struct {
	s    parser.Scanner
	msg  string
	args []interface{}
	kind validationErrorKind
}

func (v validationError) Error() string {
	var args []interface{}
	if v.s.String() != "" {
		args = append(args, v.s.String())
		if len(v.args) > 0 {
			args = append(args, v.args...)
		}
		args = append(args, v.s.Offset())

		return fmt.Sprintf(v.msg+"@ %d", args...)
	}
	return fmt.Sprintf(v.msg, args...)
}

type validator struct {
	knownRules frozen.Set
	macros     map[string]PragmaMacrodefNode
	err        []error
}

func (v *validator) walk(node IsWalkableType) {
	ops := WalkerOps{
		EnterAtomNode:           v.validateAtom,
		EnterQuantNode:          v.validateQuant,
		EnterNamedNode:          v.validateNamed,
		EnterTermNode:           v.validateTerm,
		EnterPragmaMacrodefNode: v.validateMacro,
		EnterMacrocallNode:      v.validateMacroCall,
	}
	ops.Walk(node)
}

func (v *validator) Error() string {
	return fmt.Sprint(v.err)
}

func (v *validator) validateTerm(tree TermNode) Stopper {
	if len(tree.AllGrammar()) != 0 {
		//fixme: This doesnt work for scoped grammars yet, abort!
		return NodeExiter
	}
	if tree.OneOp() == "" {
		names := map[string]bool{}
		for _, child := range tree.AllTerm() {
			if name := child.OneNamed(); name != nil {
				if x := name.OneIdent().String(); x != "" {
					if _, has := names[x]; has {
						v.err = append(v.err, validationError{s: name.OneIdent().Scanner(),
							msg: "identifier '%s' is being used multiple times in a single term", kind: MultipleTermsWithSameName})
					}
					names[x] = true
				}
			}
		}
	}
	return nil
}

func (v *validator) validateNamed(tree NamedNode) Stopper {
	if x := tree.OneIdent(); x != nil {
		if v.knownRules.Has(x.String()) {
			v.err = append(v.err, validationError{s: tree.OneIdent().Scanner(),
				msg: "identifier '%s' clashes with a defined rule", kind: NameClashesWithRule})
		}
	}
	return nil
}

func (v *validator) validateAtom(tree AtomNode) Stopper {
	if ident := tree.OneIdent(); ident != nil {
		if ident.String() != "@" {
			if !v.knownRules.Has(ident.String()) {
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
					msg: fmt.Sprintf("quant: min (%d) > max (%d)", min, max), kind: MinMaxQuantError})
			}
		}
	case 2:
	}
	return nil
}

func (v *validator) validateMacro(node PragmaMacrodefNode) Stopper {
	prevRules := v.knownRules
	defer func() { v.knownRules = prevRules }()

	for _, arg := range node.AllArgs() {
		if v.knownRules.Has(arg.String()) {
			v.err = append(v.err, validationError{s: arg.Scanner(),
				msg: "macro arg '%s' clashes with a defined rule", kind: NameClashesWithRule})
		} else {
			v.knownRules = v.knownRules.With(arg.String())
		}
	}
	v.walk(node.OneTerm())
	return NodeExiter
}

func (v *validator) validateMacroCall(node MacrocallNode) Stopper {
	macro, has := v.macros[node.OneName().String()]
	if !has {
		v.err = append(v.err, validationError{s: node.OneName().Scanner(),
			msg: "Attempting to call %s which is not a macro", kind: NotAMacro})
	} else {
		if len(macro.AllArgs()) != len(node.AllTerm()) {
			v.err = append(v.err, validationError{
				msg: fmt.Sprintf("Macro %s expected %d args, given %d",
					node.OneName().String(), len(macro.AllArgs()), len(node.AllTerm())), kind: IncorrectMacroArgCount})
		}
	}
	return nil
}
