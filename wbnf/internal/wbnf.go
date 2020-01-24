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
func (x *Grammar) ForEachStmt(fn func(node *Stmt)) {
	parser.ForEach(x.AllStmt(), func(node parser.BaseNode) {
		fn(node.(*Stmt))
	})
}
func (x *Grammar) GetStmt(index int) *Stmt {
	if res := x.AtIndex(reflect.TypeOf(Stmt{}), parser.NoTag, index); res != nil {
		return res.(*Stmt)
	}
	return nil
}

type Stmt struct {
	parser.NonTerminal
	choice  int
	comment parser.BaseNode
	prod    parser.BaseNode
}

func (x *Stmt) COMMENT() *COMMENT {
	if x.comment == nil {
		return nil
	}
	return x.comment.(*COMMENT)
}
func (x *Stmt) Choice() int { return x.choice }
func (x *Stmt) Prod() *Prod {
	if x.prod == nil {
		return nil
	}
	return x.prod.(*Prod)
}

type Prod struct {
	parser.NonTerminal
	ident      parser.BaseNode
	termCount  int
	tokenCount int
}

func (x *Prod) AllTerm() parser.Iter  { return x.Iter(reflect.TypeOf(Term{}), parser.NoTag) }
func (x *Prod) AllToken() parser.Iter { return x.Iter(reflect.TypeOf(parser.Terminal{}), parser.NoTag) }
func (x *Prod) CountTerm() int        { return x.termCount }
func (x *Prod) CountToken() int       { return x.tokenCount }
func (x *Prod) ForEachTerm(fn func(node *Term)) {
	parser.ForEach(x.AllTerm(), func(node parser.BaseNode) {
		fn(node.(*Term))
	})
}
func (x *Prod) ForEachToken(fn func(node *parser.Terminal)) {
	parser.ForEach(x.AllToken(), func(node parser.BaseNode) {
		fn(node.(*parser.Terminal))
	})
}
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
func (x *Prod) IDENT() *IDENT {
	if x.ident == nil {
		return nil
	}
	return x.ident.(*IDENT)
}

type Term struct {
	parser.NonTerminal
	choice     int
	named      parser.BaseNode
	opCount    int
	quantCount int
	termCount  int
	tokenCount int
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
func (x *Term) ForEachOp(fn func(node *parser.Terminal)) {
	parser.ForEach(x.AllOp(), func(node parser.BaseNode) {
		fn(node.(*parser.Terminal))
	})
}
func (x *Term) ForEachQuant(fn func(node *Quant)) {
	parser.ForEach(x.AllQuant(), func(node parser.BaseNode) {
		fn(node.(*Quant))
	})
}
func (x *Term) ForEachTerm(fn func(node *Term)) {
	parser.ForEach(x.AllTerm(), func(node parser.BaseNode) {
		fn(node.(*Term))
	})
}
func (x *Term) ForEachToken(fn func(node *parser.Terminal)) {
	parser.ForEach(x.AllToken(), func(node parser.BaseNode) {
		fn(node.(*parser.Terminal))
	})
}
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
	choice     int
	intCount   int
	lbang      parser.BaseNode
	max        parser.BaseNode
	min        parser.BaseNode
	named      parser.BaseNode
	opCount    int
	rbang      parser.BaseNode
	tokenCount int
}

func (x *Quant) AllINT() parser.Iter   { return x.Iter(reflect.TypeOf(INT{}), parser.NoTag) }
func (x *Quant) AllOp() parser.Iter    { return x.Iter(reflect.TypeOf(parser.Terminal{}), "op") }
func (x *Quant) AllToken() parser.Iter { return x.Iter(reflect.TypeOf(parser.Terminal{}), parser.NoTag) }
func (x *Quant) Choice() int           { return x.choice }
func (x *Quant) CountINT() int         { return x.intCount }
func (x *Quant) CountOp() int          { return x.opCount }
func (x *Quant) CountToken() int       { return x.tokenCount }
func (x *Quant) ForEachINT(fn func(node *INT)) {
	parser.ForEach(x.AllINT(), func(node parser.BaseNode) {
		fn(node.(*INT))
	})
}
func (x *Quant) ForEachOp(fn func(node *parser.Terminal)) {
	parser.ForEach(x.AllOp(), func(node parser.BaseNode) {
		fn(node.(*parser.Terminal))
	})
}
func (x *Quant) ForEachToken(fn func(node *parser.Terminal)) {
	parser.ForEach(x.AllToken(), func(node parser.BaseNode) {
		fn(node.(*parser.Terminal))
	})
}
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
func (x *Quant) Lbang() *parser.Terminal {
	if x.lbang == nil {
		return nil
	}
	return x.lbang.(*parser.Terminal)
}
func (x *Quant) Max() *INT {
	if x.max == nil {
		return nil
	}
	return x.max.(*INT)
}
func (x *Quant) Min() *INT {
	if x.min == nil {
		return nil
	}
	return x.min.(*INT)
}
func (x *Quant) Named() *Named {
	if x.named == nil {
		return nil
	}
	return x.named.(*Named)
}
func (x *Quant) Rbang() *parser.Terminal {
	if x.rbang == nil {
		return nil
	}
	return x.rbang.(*parser.Terminal)
}

type Named struct {
	parser.NonTerminal
	atom  parser.BaseNode
	ident parser.BaseNode
	op    parser.BaseNode
}

func (x *Named) Atom() *Atom {
	if x.atom == nil {
		return nil
	}
	return x.atom.(*Atom)
}
func (x *Named) IDENT() *IDENT {
	if x.ident == nil {
		return nil
	}
	return x.ident.(*IDENT)
}
func (x *Named) Op() *parser.Terminal {
	if x.op == nil {
		return nil
	}
	return x.op.(*parser.Terminal)
}

type Atom struct {
	parser.NonTerminal
	choice     int
	ident      parser.BaseNode
	re         parser.BaseNode
	str        parser.BaseNode
	term       parser.BaseNode
	tokenCount int
}

func (x *Atom) AllToken() parser.Iter { return x.Iter(reflect.TypeOf(parser.Terminal{}), parser.NoTag) }
func (x *Atom) Choice() int           { return x.choice }
func (x *Atom) CountToken() int       { return x.tokenCount }
func (x *Atom) ForEachToken(fn func(node *parser.Terminal)) {
	parser.ForEach(x.AllToken(), func(node parser.BaseNode) {
		fn(node.(*parser.Terminal))
	})
}
func (x *Atom) GetToken(index int) *parser.Terminal {
	if res := x.AtIndex(reflect.TypeOf(parser.Terminal{}), parser.NoTag, index); res != nil {
		return res.(*parser.Terminal)
	}
	return nil
}
func (x *Atom) IDENT() *IDENT {
	if x.ident == nil {
		return nil
	}
	return x.ident.(*IDENT)
}
func (x *Atom) RE() *RE {
	if x.re == nil {
		return nil
	}
	return x.re.(*RE)
}
func (x *Atom) STR() *STR {
	if x.str == nil {
		return nil
	}
	return x.str.(*STR)
}
func (x *Atom) Term() *Term {
	if x.term == nil {
		return nil
	}
	return x.term.(*Term)
}

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
