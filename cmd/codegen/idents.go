package codegen

import (
	"fmt"
	"strings"

	"github.com/arr-ai/frozen"
	"github.com/arr-ai/wbnf/wbnf"
)

type IdentsWriter struct {
	wbnf.GrammarNode
}

func IdentName(name string) string {
	return "Ident" + strings.NewReplacer(".", "", "%", "").Replace(name)
}

func (i IdentsWriter) String() string {
	names := frozen.NewSet()

	wbnf.WalkerOps{
		EnterNamedNode: func(node wbnf.NamedNode) wbnf.Stopper {
			n := node.OneIdent().String()
			if n != "@" {
				names = names.With(n)
			}
			return nil
		},
		EnterProdNode: func(node wbnf.ProdNode) wbnf.Stopper {
			names = names.With(node.OneIdent().String())
			return nil
		},
	}.Walk(i.GrammarNode)

	sorted := names.OrderedElements(func(a, b interface{}) bool {
		return strings.Compare(IdentName(a.(string)), IdentName(b.(string))) < 0
	})
	out := "const (\n"
	for _, name := range sorted {
		if name != "" {
			out += fmt.Sprintf("%s = \"%s\"\n", IdentName(name.(string)), name)
		}
	}
	return out + ")\n"
}
