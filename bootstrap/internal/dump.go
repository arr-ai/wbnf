package internal

import (
	"fmt"
	"strings"

	"github.com/arr-ai/wbnf/parser"
)

func Join(iter parser.Iter, sep string) string {
	var out []string
	parser.ForEach(iter, func(node parser.BaseNode) {
		out = append(out, dump(node))
	})
	return strings.Join(out, sep)
}
func Join2(children []parser.BaseNode, sep string) string {
	var out []string
	for _, c := range children {
		out = append(out, dump(c))
	}
	return strings.Join(out, sep)
}

func dump(node parser.BaseNode) string {
	switch x := node.(type) {
	case *Stmt:
		return dump(x.AllChildren()[0])
	case *Prod:
		return fmt.Sprintf("%s %s %s %s",
			x.IDENT(), x.GetToken(0),
			Join(x.AllTerm(), " "), x.GetToken(1))
	case *Named:
		out := ""
		if ident := x.IDENT(); ident != nil {
			out = fmt.Sprintf("%s %s ", dump(ident), dump(x.Op()))
		}
		out += dump(x.Atom())
		return out
	case *Term:
		sep := " "
		if op := x.GetOp(0); op != nil {
			sep = fmt.Sprintf(" %s ", dump(op))
		}
		return Join2(x.AllChildren(), sep)
	case *Quant:
		switch x.choice {
		case 0:
			return dump(x.GetOp(0))
		case 1:
			return Join2(x.AllChildren(), " ")
		case 2:
			return Join2(x.AllChildren(), " ")
		}
	case *Atom:
		return Join2(x.AllChildren(), " ")
	case *IDENT, *parser.Terminal, *INT, *COMMENT:
		return fmt.Sprintf("%s", x)
	case *STR:
		return fmt.Sprintf(`"%s"`, x)
	case *RE:
		return fmt.Sprintf(`/{%s}`, x)

	}
	return ""
}
