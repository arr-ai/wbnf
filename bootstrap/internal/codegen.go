package internal

import (
	"fmt"
	"sort"
	"strings"
)

type classWriter struct {
	name       string
	idents     *identFinder
	isTerminal bool
}

func newClassWriter(prod *Prod) classWriter {
	cw := classWriter{
		name:   goName(prod.IDENT().String(), true),
		idents: &identFinder{},
	}

	ForEach(prod.AllTerm(), func(node isGenNode) {
		cw.idents.walk(node, false)
	})

	if prod.CountTerm() > 0 {
		cw.idents.Add("choice", "int", false)
	}
	if len(cw.idents.names) == 1 {
		for _, val := range cw.idents.names {
			cw.isTerminal = !val.multiple
			break
		}
	}

	return cw
}

type identifier struct {
	multiple bool
	typename string
}
type identFinder struct {
	names map[string]*identifier
	only  string
}

func (i *identFinder) Add(name, typename string, multi bool) {
	if len(i.names) == 0 {
		i.only = name
	} else {
		i.only = ""
	}
	if i.names == nil {
		i.names = map[string]*identifier{}
	}
	if v, has := i.names[name]; has {
		if v.typename != typename {
			panic("oops")
		}
		v.multiple = true
	} else {
		i.names[name] = &identifier{typename: typename, multiple: multi}
	}
}

func quantNeedsMulti(q *Quant) bool {
	switch q.choice {
	case 0:
		switch fmt.Sprintf("%s", q.op) {
		case "?":
			return false
		}
	}
	return true
}

func (i *identFinder) walk(node isGenNode, needsMulti bool) {
	switch x := node.(type) {
	case *Named:
		atomIf := &identFinder{}
		atomIf.walk(x.Atom(), false)
		atomName := atomIf.only
		if x.IDENT() != nil {
			i.Add(x.IDENT().String(), atomName, needsMulti)
		}
		if atomName != "" {
			i.Add(atomName, atomIf.names[atomName].typename, needsMulti)
		}
	case *Term:
		if x.Named() != nil {
			multi := false
			ForEach(x.AllQuant(), func(node isGenNode) {
				multi = multi || quantNeedsMulti(node.(*Quant))
			})

			i.walk(x.Named(), multi)
		} else {
			ForEach(x.AllTerm(), func(node isGenNode) {
				i.walk(node, false)
				n := node.(*Term)
				ForEach(n.AllQuant(), func(node isGenNode) {
					i.walk(node, false)
				})
			})
		}
	case *Atom:
		switch x.Choice() {
		case 0, 1, 2:
			switch x := x.AllChildren()[0].(type) {
			case *STR, *RE, *INT:
				i.Add("Token", "Token", needsMulti)
			case *IDENT:
				i.Add(x.String(), x.String(), needsMulti)
			default:
			}
		case 3:
			i.walk(x.children[1], needsMulti)
		}
	case *Quant:
		for _, child := range x.AllChildren() {
			switch x := child.(type) {
			case *Named:
				i.walk(x, true)
			}
		}
	default:
	}
}

func goName(str string, public bool) string {
	switch str[0] {
	case '.':
		str = str[1:]
	}

	if public && strings.ToUpper(str) == str {
		return str // keep it allcaps
	}

	str = strings.ToLower(str)
	if !public {
		return str
	}
	return strings.ToUpper(string(str[0])) + str[1:]
}

func (c classWriter) Write() string {
	tmpl := `
type {{name}} struct {
	children []isGenNode
	{{fields}}
}
func (x *{{name}}) isGenNode() {}
func (x *{{name}}) AllChildren() []isGenNode { return x.children }

`
	if c.isTerminal {
		tmpl := `type {{name}} string
func (x *{{name}}) isGenNode()     {}
func (x *{{name}}) String() string { return string(*x) }
`
		return strings.ReplaceAll(tmpl, "{{name}}", c.name)
	}
	var fields []string
	var funcs []string
	for fname, ftype := range c.idents.names {
		pub := goName(fname, true)
		priv := goName(fname, false)
		tname := goName(ftype.typename, true)

		var ff []string
		if ftype.multiple {
			fields = append(fields, fmt.Sprintf("%sCount int", priv))
			ff = []string{
				fmt.Sprintf(`func (x *{{name}}) All%s() Iter { return NewIter(x.children, reflect.TypeOf(&%s{})) }`, pub, tname),
				fmt.Sprintf(`func (x *{{name}}) Count%s() int { return x.%sCount }`, pub, priv),
			}
		} else {
			if tname != "int" {
				tname = "*" + tname
			}

			fields = append(fields, fmt.Sprintf("%s %s", priv, tname))

			ff = []string{
				fmt.Sprintf(`func (x *{{name}}) %s() %s { return x.%s }`, pub, tname, priv),
			}
		}
		funcs = append(funcs, ff...)
	}

	sort.Strings(funcs)

	tmpl += strings.Join(funcs, "\n")
	tmpl = strings.ReplaceAll(tmpl, "{{fields}}", strings.Join(fields, "\n\t"))
	out := strings.ReplaceAll(tmpl, "{{name}}", c.name)
	return out
}

func Codegen(node isGenNode) string {
	switch x := node.(type) {
	case *Grammar:
		var out []string
		ForEach(x.AllStmt(), func(node isGenNode) {
			out = append(out, Codegen(node))
		})
		return strings.Join(out, "\n")
	case *Stmt:
		return Codegen(x.children[0])
	case *Prod:
		return newClassWriter(x).Write()
	case *COMMENT:
		return fmt.Sprintf("%s", x)
	}
	return ""
}
