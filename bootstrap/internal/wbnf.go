package internal

import (
	"reflect"
	"strings"
)

// grammar -> stmt+;
type Grammar struct {
	children  []isGenNode
	stmtCount int
}

func (g *Grammar) isGenNode()               {}
func (g *Grammar) AllChildren() []isGenNode { return g.children }
func (g *Grammar) AllStmt() Iter {
	return NewIter(g.children, reflect.TypeOf(&Stmt{}))
}
func (g *Grammar) CountStmt() int { return g.stmtCount }
func (g *Grammar) Dump() string {
	var out []string
	ForEach(g.AllStmt(), func(node isGenNode) {
		out = append(out, dump(node))
	})

	return strings.Join(out, "\n")
}

type Stmt struct {
	children []isGenNode
	choice   int
}

func (s *Stmt) isGenNode()               {}
func (s *Stmt) AllChildren() []isGenNode { return s.children }
func (s *Stmt) Choice() int              { return s.choice }
func (s *Stmt) COMMENT() *COMMENT        { return s.children[0].(*COMMENT) }
func (s *Stmt) Prod() *Prod              { return s.children[1].(*Prod) }

type Prod struct {
	children  []isGenNode
	termCount int
}

func (p *Prod) isGenNode()               {}
func (p *Prod) AllChildren() []isGenNode { return p.children }
func (p *Prod) IDENT() *IDENT            { return p.children[0].(*IDENT) }
func (p *Prod) CountTerm() int           { return p.termCount }
func (p *Prod) AllTerm() Iter {
	return NewIter(p.children, reflect.TypeOf(&Term{}))
}
func (p *Prod) Term(index int) *Term {
	t := AtIndex(p.children, reflect.TypeOf(&Term{}), index)
	if t != nil {
		return t.(*Term)
	}
	return nil
}
func (p *Prod) Token(index int) *Token {
	t := AtIndex(p.children, reflect.TypeOf(&Token{}), index)
	if t != nil {
		return t.(*Token)
	}
	return nil
}

type Term struct {
	children   []isGenNode
	choice     int
	termCount  int
	quantCount int
	op         isGenNode
}

func (t *Term) isGenNode()               {}
func (t *Term) Choice() int              { return t.choice }
func (t *Term) AllChildren() []isGenNode { return t.children }
func (t *Term) Op() isGenNode            { return t.op }
func (t *Term) CountTerm() int           { return t.termCount }
func (t *Term) CountQuant() int          { return t.quantCount }
func (t *Term) Named() *Named {
	t2 := AtIndex(t.children, reflect.TypeOf(&Named{}), 0)
	if t2 != nil {
		return t2.(*Named)
	}
	return nil
}
func (t *Term) AllQuant() Iter {
	return NewIter(t.children, reflect.TypeOf(&Quant{}))
}
func (t *Term) AllTerm() Iter {
	return NewIter(t.children, reflect.TypeOf(&Term{}))
}
func (t *Term) Term(index int) *Term {
	t2 := AtIndex(t.children, reflect.TypeOf(&Term{}), index)
	if t2 != nil {
		return t2.(*Term)
	}
	return nil
}

type Quant struct {
	children []isGenNode
	choice   int
	op       isGenNode
	min      isGenNode
	max      isGenNode
	lbang    isGenNode
	rbang    isGenNode
}

func (t *Quant) isGenNode()               {}
func (t *Quant) AllChildren() []isGenNode { return t.children }

type Named struct {
	children []isGenNode
	op       isGenNode
}

func (t *Named) isGenNode()               {}
func (t *Named) AllChildren() []isGenNode { return t.children }
func (t *Named) Op() isGenNode            { return t.op }
func (t *Named) IDENT() *IDENT {
	var temp IDENT
	t2 := AtIndex(t.children, reflect.TypeOf(&temp), 0)
	if t2 != nil {
		return t2.(*IDENT)
	}
	return nil
}
func (t *Named) Atom() *Atom {
	var temp Atom
	t2 := AtIndex(t.children, reflect.TypeOf(&temp), 0)
	if t2 != nil {
		return t2.(*Atom)
	}
	return nil
}

type Atom struct {
	children  []isGenNode
	choice    int
	numTokens int
}

func (t *Atom) isGenNode()               {}
func (t *Atom) AllChildren() []isGenNode { return t.children }
func (t *Atom) Choice() int              { return t.choice }

type IDENT string

func (c *IDENT) isGenNode()     {}
func (c *IDENT) String() string { return string(*c) }

type STR string

func (c *STR) isGenNode()     {}
func (c *STR) String() string { return string(*c) }

type INT string

func (c *INT) isGenNode()     {}
func (c *INT) String() string { return string(*c) }

type RE string

func (c *RE) isGenNode()     {}
func (c *RE) String() string { return string(*c) }

type COMMENT string

func (c *COMMENT) isGenNode()     {}
func (c *COMMENT) String() string { return string(*c) }
