package internal

import (
	"reflect"

	"github.com/arr-ai/wbnf/parser"
)

// Non-terminals

type Grammar struct {
	parser.NonTerminal
	stmtCount int
}

func (x *Grammar) AllStmt() parser.Iter { return x.Iter(reflect.TypeOf(Stmt{}), parser.NoTag) }
func (x *Grammar) CountStmt() int       { return x.stmtCount }
func (x *Grammar) GetStmt(index int) *Stmt {
	if res := x.AtIndex(reflect.TypeOf(Stmt{}), parser.NoTag, index); res != nil {
		return res.(*Stmt)
	}
	return nil
}

type Stmt struct {
	parser.NonTerminal
	comment parser.BaseNode
	prod    parser.BaseNode
	choice  int
}

func (x *Stmt) COMMENT() *COMMENT { return x.comment.(*COMMENT) }
func (x *Stmt) Choice() int       { return x.choice }
func (x *Stmt) Prod() *Prod       { return x.prod.(*Prod) }

type Prod struct {
	parser.NonTerminal
	ident      parser.BaseNode
	tokenCount int
	termCount  int
}

func (x *Prod) AllTerm() parser.Iter  { return x.Iter(reflect.TypeOf(Term{}), parser.NoTag) }
func (x *Prod) AllToken() parser.Iter { return x.Iter(reflect.TypeOf(parser.Terminal{}), parser.NoTag) }
func (x *Prod) CountTerm() int        { return x.termCount }
func (x *Prod) CountToken() int       { return x.tokenCount }
func (x *Prod) GetTerm(index int) *Term {
	if res := x.AtIndex(reflect.TypeOf(Term{}), parser.NoTag, index); res != nil {
		return res.(*Term)
	}
	return nil
}
func (x *Prod) GetToken(index int) *parser.Terminal {
	if res := x.AtIndex(reflect.TypeOf(parser.Terminal{}), parser.NoTag, index); res != nil {
		return res.(*parser.Terminal)
	}
	return nil
}
func (x *Prod) IDENT() *IDENT { return x.ident.(*IDENT) }

type Term struct {
	parser.NonTerminal
	opCount    int
	tokenCount int
	termCount  int
	named      parser.BaseNode
	quantCount int
	choice     int
}

func (x *Term) AllOp() parser.Iter    { return x.Iter(reflect.TypeOf(parser.Terminal{}), "op") }
func (x *Term) AllQuant() parser.Iter { return x.Iter(reflect.TypeOf(Quant{}), parser.NoTag) }
func (x *Term) AllTerm() parser.Iter  { return x.Iter(reflect.TypeOf(Term{}), parser.NoTag) }
func (x *Term) AllToken() parser.Iter { return x.Iter(reflect.TypeOf(parser.Terminal{}), parser.NoTag) }
func (x *Term) Choice() int           { return x.choice }
func (x *Term) CountOp() int          { return x.opCount }
func (x *Term) CountQuant() int       { return x.quantCount }
func (x *Term) CountTerm() int        { return x.termCount }
func (x *Term) CountToken() int       { return x.tokenCount }
func (x *Term) GetOp(index int) *parser.Terminal {
	if res := x.AtIndex(reflect.TypeOf(parser.Terminal{}), "op", index); res != nil {
		return res.(*parser.Terminal)
	}
	return nil
}
func (x *Term) GetQuant(index int) *Quant {
	if res := x.AtIndex(reflect.TypeOf(Quant{}), parser.NoTag, index); res != nil {
		return res.(*Quant)
	}
	return nil
}
func (x *Term) GetTerm(index int) *Term {
	if res := x.AtIndex(reflect.TypeOf(Term{}), parser.NoTag, index); res != nil {
		return res.(*Term)
	}
	return nil
}
func (x *Term) GetToken(index int) *parser.Terminal {
	if res := x.AtIndex(reflect.TypeOf(parser.Terminal{}), parser.NoTag, index); res != nil {
		return res.(*parser.Terminal)
	}
	return nil
}
func (x *Term) Named() *Named {
	if x.named == nil {
		return nil
	}
	return x.named.(*Named)
}

type Quant struct {
	parser.NonTerminal
	named      parser.BaseNode
	rbang      parser.BaseNode
	choice     int
	min        parser.BaseNode
	max        parser.BaseNode
	lbang      parser.BaseNode
	opCount    int
	tokenCount int
	intCount   int
}

func (x *Quant) AllINT() parser.Iter   { return x.Iter(reflect.TypeOf(INT{}), parser.NoTag) }
func (x *Quant) AllOp() parser.Iter    { return x.Iter(reflect.TypeOf(parser.Terminal{}), "op") }
func (x *Quant) AllToken() parser.Iter { return x.Iter(reflect.TypeOf(parser.Terminal{}), parser.NoTag) }
func (x *Quant) Choice() int           { return x.choice }
func (x *Quant) CountINT() int         { return x.intCount }
func (x *Quant) CountOp() int          { return x.opCount }
func (x *Quant) CountToken() int       { return x.tokenCount }
func (x *Quant) GetINT(index int) *INT {
	if res := x.AtIndex(reflect.TypeOf(INT{}), parser.NoTag, index); res != nil {
		return res.(*INT)
	}
	return nil
}
func (x *Quant) GetOp(index int) *parser.Terminal {
	if res := x.AtIndex(reflect.TypeOf(parser.Terminal{}), "op", index); res != nil {
		return res.(*parser.Terminal)
	}
	return nil
}
func (x *Quant) GetToken(index int) *parser.Terminal {
	if res := x.AtIndex(reflect.TypeOf(parser.Terminal{}), parser.NoTag, index); res != nil {
		return res.(*parser.Terminal)
	}
	return nil
}
func (x *Quant) Lbang() *parser.Terminal { return x.lbang.(*parser.Terminal) }
func (x *Quant) Max() *INT               { return x.max.(*INT) }
func (x *Quant) Min() *INT               { return x.min.(*INT) }
func (x *Quant) Named() *Named           { return x.named.(*Named) }
func (x *Quant) Rbang() *parser.Terminal { return x.rbang.(*parser.Terminal) }

type Named struct {
	parser.NonTerminal
	ident parser.BaseNode
	op    parser.BaseNode
	atom  parser.BaseNode
}

func (x *Named) Atom() *Atom          { return x.atom.(*Atom) }
func (x *Named) IDENT() *IDENT        { return x.ident.(*IDENT) }
func (x *Named) Op() *parser.Terminal { return x.op.(*parser.Terminal) }

type Atom struct {
	parser.NonTerminal
	tokenCount int
	term       parser.BaseNode
	choice     int
	ident      parser.BaseNode
	str        parser.BaseNode
	re         parser.BaseNode
}

func (x *Atom) AllToken() parser.Iter { return x.Iter(reflect.TypeOf(parser.Terminal{}), parser.NoTag) }
func (x *Atom) Choice() int           { return x.choice }
func (x *Atom) CountToken() int       { return x.tokenCount }
func (x *Atom) GetToken(index int) *parser.Terminal {
	if res := x.AtIndex(reflect.TypeOf(parser.Terminal{}), parser.NoTag, index); res != nil {
		return res.(*parser.Terminal)
	}
	return nil
}
func (x *Atom) IDENT() *IDENT { return x.ident.(*IDENT) }
func (x *Atom) RE() *RE       { return x.re.(*RE) }
func (x *Atom) STR() *STR     { return x.str.(*STR) }
func (x *Atom) Term() *Term   { return x.term.(*Term) }

// Terminals
type IDENT struct{ parser.Terminal }

func (x IDENT) New(value string, tag parser.Tag) parser.BaseNode {
	(&x).NewFromPtr(value, tag)
	return &x
}

type STR struct{ parser.Terminal }

func (x STR) New(value string, tag parser.Tag) parser.BaseNode {
	(&x).NewFromPtr(value, tag)
	return &x
}

type INT struct{ parser.Terminal }

func (x INT) New(value string, tag parser.Tag) parser.BaseNode {
	(&x).NewFromPtr(value, tag)
	return &x
}

type RE struct{ parser.Terminal }

func (x RE) New(value string, tag parser.Tag) parser.BaseNode {
	(&x).NewFromPtr(value, tag)
	return &x
}

type COMMENT struct{ parser.Terminal }

func (x COMMENT) New(value string, tag parser.Tag) parser.BaseNode {
	(&x).NewFromPtr(value, tag)
	return &x
}

// Special
type Wrapre struct{ parser.Terminal }

func (x Wrapre) New(value string, tag parser.Tag) parser.BaseNode {
	(&x).NewFromPtr(value, tag)
	return &x
}
