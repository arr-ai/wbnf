package bootstrap

import (
	"fmt"
	"sort"
	"strings"
)

type cardinality struct {
	lo, hi int
}

func newCardinality(lo, hi int) cardinality {
	return cardinality{lo: lo, hi: hi}
}

func (r cardinality) Plus(s cardinality) cardinality {
	return newCardinality(r.lo+s.lo, r.hi+s.hi)
}

func (r cardinality) Times(s cardinality) cardinality {
	return newCardinality(r.lo*s.lo, r.hi*s.hi)
}

type cardinalityKey string
type cardinalities map[cardinalityKey]cardinality

func (rs cardinalities) add(name string, r cardinality) cardinality {
	s := rs[cardinalityKey(name)].Plus(r)
	rs[cardinalityKey(name)] = s
	return s
}

type PathSet map[string]struct{}

func (ps PathSet) Has(name string) bool {
	_, has := ps[name]
	return has
}

func (ps PathSet) sorted() []string {
	paths := make([]string, 0, len(ps))
	for p := range ps {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths
}

func (ps PathSet) String() string {
	var sb strings.Builder
	sb.WriteString("{")
	for i, p := range ps.sorted() {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(p)
	}
	sb.WriteString("}")
	return sb.String()
}

func (ps PathSet) keys() []string {
	paths := make([]string, 0, len(ps))
	for path := range ps {
		paths = append(paths, path)
	}
	return paths
}

func (g Grammar) singletons() PathSet {
	ranges := cardinalities{}
	for rule, term := range g {
		termNodeCardinality(g, term, string(rule)+".", cardinality{lo: 1, hi: 1}, ranges)
	}
	ps := PathSet{}
	for name, cr := range ranges {
		if cr.lo == 1 && cr.hi == 1 {
			ps[string(name)] = struct{}{}
		}
	}
	return ps
}

func termNodeCardinality(g Grammar, t Term, prefix string, parent cardinality, crs cardinalities) {
	switch t := t.(type) {
	case S, RE, REF:
		crs.add(prefix, parent)
	case Rule:
		crs.add(prefix+string(t), parent)
	case Named:
		crs.add(prefix+t.Name, parent)
		termNodeCardinality(g, t.Term, prefix+t.Name+".", cardinality{lo: 1, hi: 1}, crs)
	case Seq:
		for _, term := range t {
			termNodeCardinality(g, term, prefix, parent, crs)
		}
	case Oneof:
		for _, term := range t {
			termNodeCardinality(g, term, prefix, parent, crs)
		}
	case Delim:
		// TODO: Deal with side-to-side associativity.
		termNodeCardinality(g, t.Term, prefix, parent.Times(newCardinality(1, 2)), crs)
		termNodeCardinality(g, t.Sep, prefix, parent.Times(newCardinality(0, 2)), crs)
	case Quant:
		max := t.Max
		if max == 0 {
			max = 2
		}
		termNodeCardinality(g, t.Term, prefix, parent.Times(newCardinality(t.Min, max)), crs)
	default:
		panic(fmt.Errorf("unexpected term type: %v %[1]T", t))
	}
}
