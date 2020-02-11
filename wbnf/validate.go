package wbnf

import (
	"fmt"

	"github.com/arr-ai/wbnf/errors"
	"github.com/arr-ai/wbnf/parser"
)

func validationErrorf(format string, args ...interface{}) error {
	return fmt.Errorf(format, args...)
}

func validateNode(e parser.TreeElement, expectedTag Rule, validate func(parser.Node) error) error {
	if node, ok := e.(parser.Node); ok {
		if node.Tag != string(expectedTag) {
			return validationErrorf("expecting tag `%s`, got `%s`", expectedTag, node.Tag)
		}
		return validate(node)
	}
	return validationErrorf("not a node: %v", e)
}

func validateScanner(e parser.TreeElement, validate func(parser.Scanner) error) error {
	if scanner, ok := e.(parser.Scanner); ok {
		return validate(scanner)
	}
	return validationErrorf("not a scanner: %v", e)
}

func (t S) ValidateParse(g Grammar, rule Rule, e parser.TreeElement) error {
	return validateScanner(e, func(scanner parser.Scanner) error { return nil })
}

func (t RE) ValidateParse(g Grammar, rule Rule, e parser.TreeElement) error {
	return validateScanner(e, func(scanner parser.Scanner) error {
		return nil
		// if _, err := regexp.Parse()
	})
}

func (t REF) ValidateParse(g Grammar, rule Rule, e parser.TreeElement) error {
	panic(errors.Inconceivable)
}

func (t Seq) ValidateParse(g Grammar, rule Rule, e parser.TreeElement) error {
	return validateNode(e, ruleOrAlt(rule, seqTag), func(node parser.Node) error {
		if node.Count() != len(t) {
			return validationErrorf("seq(%d): wrong number of children: %d", len(t), node.Count())
		}
		for i, term := range t {
			if err := term.ValidateParse(g, "", node.Children[i]); err != nil {
				return err
			}
		}
		return nil
	})
}

func (t Oneof) ValidateParse(g Grammar, rule Rule, e parser.TreeElement) error {
	return validateNode(e, ruleOrAlt(rule, oneofTag), func(node parser.Node) error {
		if n := node.Count(); n != 1 {
			return validationErrorf("oneof: expecting one child, got %d", n)
		}
		if i, ok := node.Extra.(Choice); ok {
			return t[i].ValidateParse(g, "", node.Children[0])
		}
		return validationErrorf("oneof: missing selected child")
	})
}

func (t Delim) ValidateParse(g Grammar, rule Rule, e parser.TreeElement) error {
	return validateNode(e, ruleOrAlt(rule, delimTag), func(node parser.Node) error {
		n := node.Count()
		if n == 0 {
			return validationErrorf("delim: no children")
		}
		if n%2 != 1 {
			return validationErrorf("delim: expecting odd number of children, not %d", n)
		}
		_, ok := node.Extra.(Associativity)
		if !ok {
			return validationErrorf("delim: missing depth")
		}

		tgen := t.LRTerms(node)
		for _, child := range node.Children {
			term, _ := tgen.Next()
			if err := term.ValidateParse(g, "", child); err != nil {
				return err
			}
		}
		return nil
	})
}

func (t Quant) ValidateParse(g Grammar, rule Rule, e parser.TreeElement) error {
	return validateNode(e, ruleOrAlt(rule, quantTag), func(node parser.Node) error {
		n := node.Count()
		if !t.Contains(n) {
			return validationErrorf("quant(%d..%d): wrong number of children: %d", t.Min, t.Max, n)
		}
		for _, child := range node.Children {
			if err := t.Term.ValidateParse(g, "", child); err != nil {
				return err
			}
		}
		return nil
	})
}

func (t Rule) ValidateParse(g Grammar, rule Rule, e parser.TreeElement) error {
	return g[t].ValidateParse(g, t, e)
}

//-----------------------------------------------------------------------------

func (t Stack) ValidateParse(_ Grammar, _ Rule, _ parser.TreeElement) error {
	panic(errors.Inconceivable)
}

//-----------------------------------------------------------------------------

func (t Named) ValidateParse(g Grammar, rule Rule, e parser.TreeElement) error {
	// TODO: Be a little more thorough.
	return nil
}
