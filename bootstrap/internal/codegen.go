package internal

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

type classWriter struct {
	name    string
	multis  []string
	singles []string
}

func newClassWriter(prod *Prod) classWriter {
	cw := classWriter{name: goName(prod.IDENT().String(), true)}

	fields := map[string]int{}
	ForEach(prod.AllTerm(), func(node isGenNode) {
		termName(node.(*Term), &fields)
	})

	for key, count := range fields {
		if count == 1 {
			cw.singles = append(cw.singles, key)
		} else {
			cw.multis = append(cw.multis, key)
		}
	}

	ForEach(prod.AllTerm(), func(node isGenNode) {
		if node.(*Term).op != nil {
			cw.singles = append(cw.singles, "choice:int")
		}
	})

	return cw
}

type identifier struct {
	multiple bool
	typename string
}
type identFinder struct {
	names map[string]*identifier
}

/*
func (i *identFinder) Add(name, typename string, multi bool) {
	if v, has := i.names[name]; has {
		if v.typename != typename {
			panic("oops")
		}
		v.multiple = true
	} else {
		i.names[name] = &identifier{typename: typename, multiple: multi}
	}
}
func (i *identFinder) walk(node isGenNode) {
	switch x := node.(type) {
	case *Named:
		if x.IDENT() != nil {
			name := x.IDENT().String()
			atom := nameFrom(x.Atom())
		}
	case *Term:
		if x.Named() != nil {
			name := nameFrom(x.Named())
		}

	}
}
*/
func nameFrom(node isGenNode) string {
	switch x := node.(type) {
	case *Named:
		if x.IDENT() != nil {
			name := x.IDENT().String()
			if t := AtIndex(x.children, reflect.TypeOf(&Term{}), 0); t != nil {
				panic("eek")
			}

			atom := nameFrom(x.Atom())
			return name + ":" + atom
		}
		return nameFrom(x.Atom())
	case *Term:
		if x.Named() != nil {
			return nameFrom(x.Named())
		}
		return "poo"
	case *Atom:
		switch x.Choice() {
		case 0, 1, 2:
			return nameFrom(x.AllChildren()[0])
		default:
		}
	case *IDENT:
		return string(*x)
	case *STR, *RE, *INT:
		return "Token"
	}
	return "<unknown>"
}

func maxCount(q *Quant) (string, int) {
	switch q.choice {
	case 0:
		switch fmt.Sprintf("%s", q.op) {
		case "?":
			return "", 0
		}
	case 2:
		return nameFrom(q.children[1]), 0
	}
	return "", 1
}

func termName(node *Term, dest *map[string]int) {
	if node.Named() != nil {
		name := nameFrom(node)
		val, _ := (*dest)[name]
		ForEach(node.AllQuant(), func(node isGenNode) {
			tname, inc := maxCount(node.(*Quant))
			if tname != "" {
				v, _ := (*dest)[tname]
				(*dest)[tname] = v + 1
			}
			val += inc
		})
		(*dest)[name] = val + 1
		return
	}
	ForEach(node.AllTerm(), func(node isGenNode) {
		termName(node.(*Term), dest)
	})
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
	if len(c.multis) == 0 && len(c.singles) == 1 {
		tmpl := `type {{name}} string
func (x *{{name}}) isGenNode()     {}
func (x *{{name}}) String() string { return string(*x) }
`
		return strings.ReplaceAll(tmpl, "{{name}}", c.name)
	}
	var fields []string
	var funcs []string
	for _, f := range c.multis {
		parts := strings.Split(f, ":")
		pub := goName(parts[0], true)
		priv := goName(parts[0], false)
		tname := pub
		if len(parts) == 2 {
			tname = goName(parts[1], true)
		}
		fields = append(fields, fmt.Sprintf("%sCount int", priv))

		ff := []string{
			fmt.Sprintf(`func (x *{{name}}) All%s() Iter { return NewIter(x.children, reflect.TypeOf(&%s{})) }`, pub, tname),
			fmt.Sprintf(`func (x *{{name}}) Count%s() int { return x.%sCount }`, pub, priv),
		}
		funcs = append(funcs, ff...)
	}

	for _, f := range c.singles {
		parts := strings.Split(f, ":")
		pub := goName(parts[0], true)
		priv := goName(parts[0], false)
		tname := "*" + pub
		if len(parts) == 2 {
			tname = "*" + goName(parts[1], true)
		}
		if parts[0] == "choice" {
			tname = "int"
		}
		fields = append(fields, fmt.Sprintf("%s %s", priv, tname))

		ff := []string{
			fmt.Sprintf(`func (x *{{name}}) %s() %s { return x.%s }`, pub, tname, priv),
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
