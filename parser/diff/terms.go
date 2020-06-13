package diff

import (
	"fmt"
	"reflect"

	"github.com/arr-ai/wbnf/parser"
)

type Report interface {
	Equal() bool
}

type InterfaceDiff struct {
	A, B interface{}
}

func (d InterfaceDiff) Equal() bool {
	return d.A == d.B
}

func diffInterfaces(a, b interface{}) InterfaceDiff {
	if diff := (InterfaceDiff{A: a, B: b}); !diff.Equal() {
		return diff
	}
	return InterfaceDiff{}
}

//-----------------------------------------------------------------------------

type GrammarDiff struct {
	OnlyInA []parser.Rule
	OnlyInB []parser.Rule
	Prods   map[parser.Rule]TermDiff
}

func (d GrammarDiff) Equal() bool {
	return len(d.OnlyInA) == 0 && len(d.OnlyInB) == 0 && len(d.Prods) == 0
}

func Grammars(a, b parser.Grammar) GrammarDiff {
	diff := GrammarDiff{
		Prods: map[parser.Rule]TermDiff{},
	}
	for rule, aTerm := range a {
		if bTerm, ok := b[rule]; ok {
			if td := Terms(aTerm, bTerm); !td.Equal() {
				diff.Prods[rule] = td
			}
		} else {
			diff.OnlyInA = append(diff.OnlyInA, rule)
		}
	}
	for rule := range b {
		if _, ok := a[rule]; !ok {
			diff.OnlyInB = append(diff.OnlyInB, rule)
		}
	}
	if diff.Equal() != reflect.DeepEqual(a, b) {
		panic(fmt.Sprintf(
			"diff.Equal() == %v != %v == reflect.DeepEqual(a, b): %#v\n%#v\n%#v",
			diff.Equal(), reflect.DeepEqual(a, b), diff, a, b))
	}
	return diff
}

//-----------------------------------------------------------------------------

type TermDiff interface {
	Report
}

type TypesDiffer struct {
	InterfaceDiff
}

func (d TypesDiffer) Equal() bool {
	return false
}

func Terms(a, b parser.Term) TermDiff {
	if reflect.TypeOf(a) != reflect.TypeOf(b) {
		return TypesDiffer{
			InterfaceDiff: diffInterfaces(
				reflect.TypeOf(a).String(),
				reflect.TypeOf(b).String(),
			),
		}
	}
	switch a := a.(type) {
	case parser.Rule:
		return diffRules(a, b.(parser.Rule))
	case parser.S:
		return diffSes(a, b.(parser.S))
	case parser.RE:
		return diffREs(a, b.(parser.RE))
	case parser.Seq:
		return diffSeqs(a, b.(parser.Seq))
	case parser.Oneof:
		return diffOneofs(a, b.(parser.Oneof))
	case parser.Stack:
		return diffTowers(a, b.(parser.Stack))
	case parser.Delim:
		return diffDelims(a, b.(parser.Delim))
	case parser.Quant:
		return diffQuants(a, b.(parser.Quant))
	case parser.Named:
		return diffNameds(a, b.(parser.Named))
	case parser.ScopedGrammar:
		return diffScopedGrammars(a, b.(parser.ScopedGrammar))
	case parser.CutPoint:
		return Terms(a.Term, b.(parser.CutPoint).Term)
	case parser.ExtRef:
		return diffSes(parser.S(string(a)), parser.S(string(a)))
	case parser.REF:
		return diffRefs(a, b.(parser.REF))
	default:
		panic(fmt.Errorf("unknown term type: %v %[1]T", a))
	}
}

//-----------------------------------------------------------------------------

type RuleDiff struct {
	A, B parser.Rule
}

func (d RuleDiff) Equal() bool {
	return d.A == d.B
}

func diffRules(a, b parser.Rule) RuleDiff {
	return RuleDiff{A: a, B: b}
}

//-----------------------------------------------------------------------------

type SDiff struct {
	A, B parser.S
}

func (d SDiff) Equal() bool {
	return d.A == d.B
}

func diffSes(a, b parser.S) SDiff {
	return SDiff{A: a, B: b}
}

//-----------------------------------------------------------------------------

type REDiff struct {
	A, B parser.RE
}

func (d REDiff) Equal() bool {
	return d.A == d.B
}

func diffREs(a, b parser.RE) REDiff {
	return REDiff{A: a, B: b}
}

//-----------------------------------------------------------------------------

type RefDiff struct {
	A, B parser.REF
}

func (d RefDiff) Equal() bool {
	if d.A.Ident == d.B.Ident {
		return (d.A.Default == nil && d.B.Default == nil) || Terms(d.A.Default, d.B.Default).Equal()
	}
	return false
}

func diffRefs(a, b parser.REF) RefDiff {
	return RefDiff{A: a, B: b}
}

//-----------------------------------------------------------------------------

type termsesDiff struct {
	Len   InterfaceDiff
	Terms []TermDiff
}

func (d termsesDiff) Equal() bool {
	return d.Len.Equal() && d.Terms == nil
}

func diffTermses(a, b []parser.Term) termsesDiff {
	var tsd termsesDiff
	tsd.Len = diffInterfaces(len(a), len(b))
	lenDiff := len(a) - len(b)
	switch {
	case lenDiff < 0:
		b = b[:len(a)]
	case lenDiff > 0:
		a = a[:len(b)]
	}

	for i, term := range a {
		if td := Terms(term, b[i]); !td.Equal() {
			tsd.Terms = append(tsd.Terms, td)
		}
	}
	return tsd
}

type SeqDiff termsesDiff

func (d SeqDiff) Equal() bool {
	return (termsesDiff(d)).Equal()
}

func diffSeqs(a, b parser.Seq) SeqDiff {
	return SeqDiff(diffTermses(a, b))
}

type OneofDiff termsesDiff

func (d OneofDiff) Equal() bool {
	return (termsesDiff(d)).Equal()
}

func diffOneofs(a, b parser.Oneof) OneofDiff {
	return OneofDiff(diffTermses(a, b))
}

type TowerDiff termsesDiff

func (d TowerDiff) Equal() bool {
	return (termsesDiff(d)).Equal()
}

func diffTowers(a, b parser.Stack) TowerDiff {
	return TowerDiff(diffTermses(a, b))
}

//-----------------------------------------------------------------------------

type DelimDiff struct {
	Term            TermDiff
	Sep             TermDiff
	Assoc           InterfaceDiff
	CanStartWithSep InterfaceDiff
	CanEndWithSep   InterfaceDiff
}

func (d DelimDiff) Equal() bool {
	return d.Term.Equal() &&
		d.Sep.Equal() &&
		d.Assoc.Equal() &&
		d.CanStartWithSep.Equal() &&
		d.CanEndWithSep.Equal()
}

func diffDelims(a, b parser.Delim) DelimDiff {
	return DelimDiff{
		Term:            Terms(a.Term, b.Term),
		Sep:             Terms(a.Sep, b.Sep),
		Assoc:           diffInterfaces(a.Assoc, b.Assoc),
		CanStartWithSep: diffInterfaces(a.CanStartWithSep, b.CanStartWithSep),
		CanEndWithSep:   diffInterfaces(a.CanEndWithSep, b.CanEndWithSep),
	}
}

//-----------------------------------------------------------------------------

type QuantDiff struct {
	Term TermDiff
	Min  InterfaceDiff
	Max  InterfaceDiff
}

func (d QuantDiff) Equal() bool {
	return d.Term.Equal() && d.Min.Equal() && d.Max.Equal()
}

func diffQuants(a, b parser.Quant) QuantDiff {
	return QuantDiff{
		Term: Terms(a.Term, b.Term),
		Min:  diffInterfaces(a.Min, b.Min),
		Max:  diffInterfaces(a.Max, b.Max),
	}
}

//-----------------------------------------------------------------------------

type NamedDiff struct {
	Name InterfaceDiff
	Term TermDiff
}

func (d NamedDiff) Equal() bool {
	return d.Name.Equal() && d.Term.Equal()
}

func diffNameds(a, b parser.Named) NamedDiff {
	return NamedDiff{
		Name: diffInterfaces(a.Name, b.Name),
		Term: Terms(a.Term, b.Term),
	}
}

//-----------------------------------------------------------------------------

type ScopedGrammarDiff struct {
	Term    TermDiff
	Grammar GrammarDiff
}

func (d ScopedGrammarDiff) Equal() bool {
	return d.Term.Equal() && d.Grammar.Equal()
}

func diffScopedGrammars(a, b parser.ScopedGrammar) ScopedGrammarDiff {
	return ScopedGrammarDiff{
		Term:    Terms(a.Term, b.Term),
		Grammar: Grammars(a.Grammar, b.Grammar),
	}
}
