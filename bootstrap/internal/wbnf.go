package internal

import (
	"reflect"
	"strings"

	"github.com/arr-ai/wbnf/parser"
)

// grammar -> stmt+;
type Grammar struct {
	parser.NonTerminal
	stmtCount int
}

func (g *Grammar) AllStmt() parser.Iter {
	return parser.NewIter(g.AllChildren(), reflect.TypeOf(&Stmt{}), "")
}
func (g *Grammar) CountStmt() int { return g.stmtCount }
func (g *Grammar) Dump() string {
	var out []string
	//	parser.ForEach(g.AllStmt(), func(node parser.BaseNode) {
	//		out = append(out, dump(node))
	//	})

	return strings.Join(out, "\n")
}

type Stmt struct {
	parser.NonTerminal
	choice int
}

func (s *Stmt) Choice() int       { return s.choice }
func (s *Stmt) COMMENT() *COMMENT { return s.AllChildren()[0].(*COMMENT) }
func (s *Stmt) Prod() *Prod       { return s.AllChildren()[1].(*Prod) }

type Prod struct {
	parser.NonTerminal
	termCount int
}

func (p *Prod) IDENT() *IDENT  { return p.AllChildren()[0].(*IDENT) }
func (p *Prod) CountTerm() int { return p.termCount }
func (p *Prod) AllTerm() parser.Iter {
	return parser.NewIter(p.AllChildren(), reflect.TypeOf(&Term{}), "")
}
func (p *Prod) Term(index int) *Term {
	t := parser.AtIndex(p.AllChildren(), reflect.TypeOf(&Term{}), "", index)
	if t != nil {
		return t.(*Term)
	}
	return nil
}
func (p *Prod) Token(index int) *parser.Terminal {
	t := parser.AtIndex(p.AllChildren(), reflect.TypeOf(&parser.Terminal{}), "", index)
	if t != nil {
		return t.(*parser.Terminal)
	}
	return nil
}

type Term struct {
	parser.NonTerminal
	choice     int
	termCount  int
	quantCount int
	op         parser.BaseNode
}

func (t *Term) Choice() int         { return t.choice }
func (t *Term) Op() parser.BaseNode { return t.op }
func (t *Term) CountTerm() int      { return t.termCount }
func (t *Term) CountQuant() int     { return t.quantCount }
func (t *Term) Named() *Named {
	var temp Named
	t2 := parser.AtIndex(t.AllChildren(), reflect.TypeOf(&temp), "", 0)
	if t2 != nil {
		return t2.(*Named)
	}
	return nil
}
func (t *Term) AllQuant() parser.Iter {
	return parser.NewIter(t.AllChildren(), reflect.TypeOf(&Quant{}), "")
}
func (t *Term) AllTerm() parser.Iter {
	return parser.NewIter(t.AllChildren(), reflect.TypeOf(&Term{}), "")
}
func (t *Term) Term(index int) *Term {
	var temp Term
	t2 := parser.AtIndex(t.AllChildren(), reflect.TypeOf(&temp), "", 0)
	if t2 != nil {
		return t2.(*Term)
	}
	return nil
}

type Quant struct {
	parser.NonTerminal
	choice int
	op     parser.BaseNode
	min    parser.BaseNode
	max    parser.BaseNode
	lbang  parser.BaseNode
	rbang  parser.BaseNode
}

type Named struct {
	parser.NonTerminal
	op parser.BaseNode
}

func (t *Named) Op() parser.BaseNode { return t.op }
func (t *Named) IDENT() *IDENT {
	var temp IDENT
	t2 := parser.AtIndex(t.AllChildren(), reflect.TypeOf(&temp), "", 0)
	if t2 != nil {
		return t2.(*IDENT)
	}
	return nil
}
func (t *Named) Atom() *Atom {
	var temp Atom
	t2 := parser.AtIndex(t.AllChildren(), reflect.TypeOf(&temp), "", 0)
	if t2 != nil {
		return t2.(*Atom)
	}
	return nil
}

type Atom struct {
	parser.NonTerminal
	choice    int
	numTokens int
}

func (t *Atom) Choice() int { return t.choice }

type IDENT struct{ parser.Terminal }

func (t IDENT) New(value string, tag parser.Tag) parser.BaseNode {
	(&t).NewFromPtr(value, tag)
	return &t
}

type STR struct{ parser.Terminal }

func (t STR) New(value string, tag parser.Tag) parser.BaseNode {
	(&t).NewFromPtr(value, tag)
	return &t
}

type INT struct{ parser.Terminal }

func (t INT) New(value string, tag parser.Tag) parser.BaseNode {
	(&t).NewFromPtr(value, tag)
	return &t
}

type RE struct{ parser.Terminal }

func (t RE) New(value string, tag parser.Tag) parser.BaseNode {
	(&t).NewFromPtr(value, tag)
	return &t
}

type COMMENT struct{ parser.Terminal }

func (t COMMENT) New(value string, tag parser.Tag) parser.BaseNode {
	(&t).NewFromPtr(value, tag)
	return &t
}
