package ast

import (
	"fmt"

	"github.com/arr-ai/wbnf/wbnf"
)

type counter struct {
	lo, hi int
}

func newCounter(lo, hi int) counter {
	return counter{lo: lo, hi: hi}
}

func newCounterFromQuant(q wbnf.Quant) counter {
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

func newCounters(t wbnf.Term) counters {
	result := counters{}
	result.termCountChildren(t, oneOne)
	return result
}

func (ctrs counters) singular() *string {
	if ctrs != nil && len(ctrs) == 1 {
		for name, c := range ctrs {
			if c == oneOne {
				return &name
			}
			// TODO: zeroOrOne
		}
	}
	return nil
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

func (ctrs counters) termCountChildren(term wbnf.Term, parent counter) {
	switch t := term.(type) {
	case wbnf.S, wbnf.RE:
		ctrs.count("", parent)
	case wbnf.Rule:
		ctrs.count(string(t), parent)
	case wbnf.Seq:
		for _, child := range t {
			ctrs.termCountChildren(child, parent)
		}
	case wbnf.Oneof:
		ds := counters{}
		for _, child := range t {
			ds.union(newCounters(child))
		}
		ctrs.mul(ds, parent)
	case wbnf.Delim:
		ctrs.termCountChildren(t.Term, parent.mul(oneOrMore))
		ctrs.termCountChildren(t.Sep, parent.mul(zeroOrMore))
	case wbnf.Quant:
		ctrs.termCountChildren(t.Term, parent.mul(newCounterFromQuant(t)))
	case wbnf.Named:
		ctrs.count(t.Name, parent)
	case wbnf.REF:
		ctrs.count(t.Ident, parent)
	default:
		panic(fmt.Errorf("unexpected term type: %v %[1]T", t))
	}
}
