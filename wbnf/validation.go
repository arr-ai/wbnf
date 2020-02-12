package wbnf

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/arr-ai/wbnf/parser"
)

func findDefinedRules(tree GrammarNode) map[string]struct{} {
	out := map[string]struct{}{}
	for _, stmt := range tree.AllStmt() {
		for _, prod := range stmt.AllProd() {
			out[prod.OneIdent().Scanner().String()] = struct{}{}
		}
	}
	return out
}

func validate(tree GrammarNode) error {
	v := validator{}
	return v.Validate(tree)
}

type validator struct {
	knownRules map[string]struct{}
	err        []error
}

func (v *validator) Error() string {
	return fmt.Sprint(v.err)
}

func (v *validator) Validate(tree GrammarNode) error {
	v.knownRules = findDefinedRules(tree)

	for _, stmt := range tree.AllStmt() {
		v.validateStmt(stmt)
	}

	if len(v.err) > 0 {
		return v
	}
	return nil
}

func (v *validator) validateStmt(tree StmtNode) {
	for _, prod := range tree.AllProd() {
		v.validateProd(prod)
	}
}

func (v *validator) validateProd(tree ProdNode) {
	name := tree.OneIdent().Scanner().String()
	if name == parser.WrapRE.String() {
		// todo: validate the .wrapRE rule
	} else {
		for _, term := range tree.AllTerm() {
			v.validateTerm(term)
		}
	}
}

func (v *validator) validateTerm(tree TermNode) {
	for _, term := range tree.AllTerm() {
		v.validateTerm(term)
	}
	for _, named := range tree.AllNamed() {
		v.validateNamed(named)
	}
	for _, quant := range tree.AllQuant() {
		v.validateQuant(quant)
	}
}

func (v *validator) validateNamed(tree NamedNode) {
	if atom := tree.OneAtom(); atom.Node != nil {
		v.validateAtom(atom)
	}
}

func (v *validator) validateAtom(tree AtomNode) {
	if x := tree.OneIdent(); x.Node != nil {
		ident := x.String()
		if ident != "" && ident != "@" {
			if _, has := v.knownRules[ident]; !has {
				v.err = append(v.err, fmt.Errorf("identifier '%s' is not a defined rule", ident))
			}
		}
	} else if x := tree.OneStr(); x.Node != nil {

	} else if x := tree.OneRe(); x.Node != nil {
		if _, err := regexp.Compile(x.String()); err != nil {
			v.err = append(v.err, fmt.Errorf("regex '%s' is not valid, %s", x.String(), err))
		}
	} else if x := tree.OneRef(); x.Node != nil {

	} else if x := tree.OneTerm(); x.Node != nil {
		v.validateTerm(x)
	}
}

func (v *validator) validateQuant(tree QuantNode) {
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
}
