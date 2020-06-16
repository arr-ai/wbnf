package ast

import (
	"fmt"

	"github.com/arr-ai/wbnf/parser"
)

type counter struct {
	lo, hi int
}

func newCounter(lo, hi int) counter {
	return counter{lo: lo, hi: hi}
}

func newCounterFromQuant(q parser.Quant) counter {
	max := q.Max
	if max == 0 {
		max = 2
	}
	return newCounter(q.Min, max)
}

var (
	zeroOrOne  = newCounter(0, 1)
	zeroOrMore = newCounter(0, 2)
	oneOne     = newCounter(1, 1)
	oneOrMore  = newCounter(1, 2)
)

func (c counter) add(d counter) counter {
	return counter{lo: c.lo + d.lo, hi: c.hi + d.hi}
}

func (c counter) mul(d counter) counter {
	return counter{lo: c.lo * d.lo, hi: c.hi * d.hi}
}

func (c counter) union(d counter) counter {
	if c.lo > d.lo {
		c.lo = d.lo
	}
	if c.hi < d.hi {
		c.hi = d.hi
	}
	return c
}

func (c counter) String() string {
	switch c.lo {
	case 0:
		switch c.hi {
		case 0:
			return "0"
		case 1:
			return "?"
		default:
			return "*"
		}
	case 1:
		switch c.hi {
		case 1:
			return "1"
		default:
			return "+"
		}
	default:
		return "⧺"
	}
}

type counters map[string]counter

func newCounters(t parser.Term) counters {
	result := counters{}
	result.termCountChildren(t, oneOne)
	return result
}

func (ctrs counters) count(name string, c counter) {
	ctrs[name] = ctrs[name].add(c)
}

func (ctrs counters) mul(ds counters, parent counter) {
	for name, c := range ds {
		ctrs.count(name, parent.mul(c))
	}
}

func (ctrs counters) union(o counters) {
	for name, c := range o {
		ctrs[name] = ctrs[name].union(c)
	}
}

func (ctrs counters) termCountChildren(term parser.Term, parent counter) {
	switch t := term.(type) {
	case parser.S, parser.RE:
		ctrs.count("", parent)
	case parser.Rule:
		ctrs.count(string(t), parent)
	case parser.Seq:
		for _, child := range t {
			ctrs.termCountChildren(child, parent)
		}
	case parser.Oneof:
		ds := counters{}
		for _, child := range t {
			ds.union(newCounters(child))
		}
		ctrs.mul(ds, parent)
	case parser.Delim:
		ctrs.termCountChildren(t.Term, parent.mul(oneOrMore))
		ctrs.termCountChildren(t.Sep, parent.mul(zeroOrMore))
	case parser.Quant:
		ctrs.termCountChildren(t.Term, parent.mul(newCounterFromQuant(t)))
	case parser.Named:
		ctrs.count(t.Name, parent)
	case parser.REF:
		ctrs.count(t.Ident, parent.mul(oneOrMore))
	case parser.ScopedGrammar:
		ctrs.termCountChildren(t.Term, parent.mul(oneOrMore))
	case parser.CutPoint:
		ctrs.termCountChildren(t.Term, parent)
	case parser.ExtRef:
		ctrs.count(string(t), parent)
	default:
		panic(fmt.Errorf("counters.termCountChildren: unexpected term type: %v %[1]T", t))
	}
}
