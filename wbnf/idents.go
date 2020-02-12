package wbnf

import (
	"strings"

	"github.com/arr-ai/frozen"
	"github.com/arr-ai/wbnf/ast"
)

// Return a map where the keys are the rule names of the AST
// and the values are the set of possible identifiers found in that rule
func IdentMap(b ast.Branch) map[string][]string {
	result := map[string][]string{}
	for _, stmt := range b.Many("stmt") {
		if rule := stmt.One("prod"); rule != nil {
			name := rule.One("IDENT").Scanner().String()
			x := frozen.NewSet()
			for _, term := range rule.Many("term") {
				x = x.Union(idents(term.(ast.Branch)))
			}
			result[name] = toStringSlice(x)
		}
	}
	return result
}

func toStringSlice(s frozen.Set) []string {
	out := make([]string, 0, s.Count())
	less := func(a, b interface{}) bool { return strings.Compare(a.(string), b.(string)) < 0 }
	for _, v := range s.OrderedElements(less) {
		out = append(out, v.(string))
	}
	return out
}

func Idents(b ast.Branch) []string {
	return toStringSlice(idents(b))
}

func idents(b ast.Branch) frozen.Set {
	founds := frozen.NewSet()
	if len(b.Many("term")) != 0 && len(b.Many("op")) != 0 {
		if b.Many("op")[0].Scanner().String() == "|" {
			founds = founds.With(ast.ChoiceTag)
		}
	}
	for _, children := range b {
		switch c := children.(type) {
		case ast.Many:
			for _, n := range c {
				if child, ok := n.(ast.Branch); ok {
					founds = founds.Union(idents(child))
				}
			}
		case ast.One:
			if child, ok := c.Node.(ast.Branch); ok {
				founds = founds.Union(idents(child))
			}
		}
		x, child := ast.Which(b, "named")
		switch x {
		case "named":
			atom := ast.First(child.(ast.One).Node, "atom")
			if id := child.(ast.One).Node.One("IDENT"); id != nil {
				realType, _ := ast.Which(atom.(ast.Branch), "STR", "RE", "REF", "term", "IDENT")
				founds = founds.With(id.Scanner().String() + "@" + realType)
			} else {
				if id := ast.First(atom, "IDENT"); id != nil {
					founds = founds.With(id.Scanner().String())
				}
			}
		}
	}
	return founds
}
