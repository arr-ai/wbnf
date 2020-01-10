package internal

import (
	"fmt"
	"strings"
)

func Join(iter Iter, sep string) string {
	var out []string
	ForEach(iter, func(node isGenNode) {
		out = append(out, dump(node))
	})
	return strings.Join(out, sep)
}
func Join2(children []isGenNode, sep string) string {
	var out []string
	for _, c := range children {
		out = append(out, dump(c))
	}
	return strings.Join(out, sep)
}

func dump(node isGenNode) string {
	switch x := node.(type) {
	case *Stmt:
		return dump(x.children[0])
	case *Prod:
		return fmt.Sprintf("%s %s %s %s",
			x.IDENT(), x.Token(0),
			Join(x.AllTerm(), " "), x.Token(1))
	case *Named:
		out := ""
		if ident := x.IDENT(); ident != nil {
			out = fmt.Sprintf("%s %s ", dump(ident), dump(x.Op()))
		}
		out += dump(x.Atom())
		return out
	case *Term:
		sep := " "
		if op := x.Op(); op != nil {
			sep = fmt.Sprintf(" %s ", dump(op))
		}
		return Join2(x.AllChildren(), sep)
	case *Quant:
		switch x.choice {
		case 0:
			return dump(x.op)
		case 1:
			return Join2(x.children, " ")
		case 2:
			return Join2(x.children, " ")
		}
	case *Atom:
		return Join2(x.AllChildren(), " ")
	case *IDENT, *Token, *INT, *COMMENT:
		return fmt.Sprintf("%s", x)
	case *STR:
		return fmt.Sprintf(`"%s"`, x)
	case *RE:
		return fmt.Sprintf(`/{%s}`, x)

	}
	return ""
}
