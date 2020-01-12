package internal

import (
	"fmt"
	"sort"
	"strings"

	"github.com/arr-ai/wbnf/parser"
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

	hasChoice := false
	parser.ForEach(prod.AllTerm(), func(node parser.BaseNode) {
		cw.idents.walk(node, false)
		hasChoice = hasChoice || node.(*Term).op != nil
	})

	if hasChoice {
		cw.idents.Add("choice", "int", false, "")
	}
	if cw.idents.only != "" {
		cw.isTerminal = !cw.idents.names[cw.idents.only].multiple
	}

	return cw
}

type identifier struct {
	multiple bool
	typename string
	tag      string
}
type identFinder struct {
	names map[string]*identifier
	only  string
}

func (i *identFinder) Add(name, typename string, multi bool, tag string) {
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
		i.names[name] = &identifier{typename: typename, multiple: multi, tag: tag}
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

func (i *identFinder) walk(node parser.BaseNode, needsMulti bool) {
	switch x := node.(type) {
	case *Named:
		atomIf := &identFinder{}
		atomIf.walk(x.Atom(), false)
		atomName := atomIf.only
		if x.IDENT() != nil {
			i.Add(x.IDENT().String(), atomName, needsMulti, x.IDENT().String())
		}
		for name, val := range atomIf.names {
			if name != "Token" || name == atomName {
				i.Add(name, val.typename, needsMulti, "")
			}
		}
	case *Term:
		if x.Named() != nil {
			multi := false
			parser.ForEach(x.AllQuant(), func(node parser.BaseNode) {
				i.walk(node, false)
				multi = multi || quantNeedsMulti(node.(*Quant))
			})

			i.walk(x.Named(), multi)
		} else {
			parser.ForEach(x.AllTerm(), func(node parser.BaseNode) {
				i.walk(node, false)
			})
		}
	case *Atom:
		switch x.Choice() {
		case 0, 1, 2:
			switch x := x.AllChildren()[0].(type) {
			case *STR, *RE, *INT:
				i.Add("Token", "parser.Terminal", needsMulti, "")
			case *IDENT:
				i.Add(x.String(), x.String(), needsMulti, "")
			default:
			}
		case 3:
			i.walk(x.AllChildren()[1], needsMulti)
		}
	case *Quant:
		for _, child := range x.AllChildren() {
			i.walk(child, true)
		}
	default:
	}
}

func goName(str string, public bool) string {
	switch str[0] {
	case '.':
		str = str[1:]
	}

	if str == "int" {
		return str
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
	parser.NonTerminal
	{{fields}}
}

`
	if c.isTerminal {
		tmpl := `type {{name}} struct{ parser.Terminal }
func (x {{name}}) New(value string, tag parser.Tag) parser.BaseNode {
	(&x).NewFromPtr(value, tag)
	return &x
}
`
		return strings.ReplaceAll(tmpl, "{{name}}", c.name)
	}
	var fields []string
	var funcs []string
	for fname, ftype := range c.idents.names {
		pub := goName(fname, true)
		priv := goName(fname, false)
		tname := goName(ftype.typename, true)
		switch ftype.typename {
		case "parser.Terminal":
			tname = ftype.typename
		case "Token":
			tname = "parser.Terminal"
		}
		tag := ftype.tag
		if tag == "" {
			tag = "parser.NoTag"
		} else {
			tag = `"` + tag + `"`
		}

		var ff []string
		if ftype.multiple {
			fields = append(fields, fmt.Sprintf("%sCount int", priv))
			ff = []string{
				fmt.Sprintf(`func (x *{{name}}) All%s() parser.Iter { return x.Iter(reflect.TypeOf(%s{}), %s) }`, pub, tname, tag),
				fmt.Sprintf(`func (x *{{name}}) Get%s(index int) *%s {
										if res := x.AtIndex(reflect.TypeOf(%s{}), %s, index); res != nil {
											return res.(*%s)
										}; return nil }`, pub, tname, tname, tag, tname),
				fmt.Sprintf(`func (x *{{name}}) Count%s() int { return x.%sCount }`, pub, priv),
			}
		} else {
			if tname != "int" {
				tname = "*" + tname
			}

			if fname == "choice" {
				fields = append(fields, fmt.Sprintf("%s int", priv))
				ff = []string{fmt.Sprintf(`func (x *{{name}}) %s() %s { return x.%s }`, pub, tname, priv)}
			} else {
				fields = append(fields, fmt.Sprintf("%s parser.BaseNode", priv))
				ff = []string{
					fmt.Sprintf(`func (x *{{name}}) %s() %s { if x.%s == nil {return nil};return x.%s.(%s) }`, pub, tname, priv, priv, tname),
				}
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

func Codegen(node parser.BaseNode) string {
	switch x := node.(type) {
	case *Grammar:
		var out []string
		parser.ForEach(x.AllStmt(), func(node parser.BaseNode) {
			out = append(out, Codegen(node))
		})
		return strings.Join(out, "\n")
	case *Stmt:
		return Codegen(x.AllChildren()[0])
	case *Prod:
		return newClassWriter(x).Write()
	case *COMMENT:
		return fmt.Sprintf("%s", x)
	}
	return ""
}
