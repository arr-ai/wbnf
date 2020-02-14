package wbnf

import (
	"fmt"
	"regexp"
	"strconv"
)

func findDefinedRules(tree GrammarNode) map[string]struct{} {
	out := map[string]struct{}{}
	ops := WalkerOps{
		EnterProdNode: func(node ProdNode) Stopper {
			out[node.OneIdent().String()] = struct{}{}
			return &nodeExiter{}
		},
	}
	ops.Walk(tree)
	return out
}

func validate(tree GrammarNode) error {
	v := validator{
		knownRules: findDefinedRules(tree),
	}

	ops := WalkerOps{
		EnterAtomNode:  v.validateAtom,
		EnterQuantNode: v.validateQuant,
	}
	ops.Walk(tree)

	if len(v.err) == 0 {
		return nil
	}
	return &v
}

type validator struct {
	knownRules map[string]struct{}
	err        []error
}

func (v *validator) Error() string {
	return fmt.Sprint(v.err)
}

func (v *validator) validateAtom(tree AtomNode) Stopper {
	if x := tree.OneIdent(); x.Node != nil {
		ident := x.String()
		if ident != "" && ident != "@" {
			if _, has := v.knownRules[ident]; !has {
				v.err = append(v.err, fmt.Errorf("identifier '%s' is not a defined rule", ident))
			}
		}
	} else if x := tree.OneRe(); x.Node != nil {
		if _, err := regexp.Compile(x.String()); err != nil {
			v.err = append(v.err, fmt.Errorf("regex '%s' is not valid, %s", x.String(), err))
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
				v.err = append(v.err, fmt.Errorf("quant: min (%d) > max (%d)", min, max))
			}
		}
	case 2:
	}
	return nil
}
