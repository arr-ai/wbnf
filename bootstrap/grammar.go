package bootstrap

import (
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"

	"github.com/arr-ai/arrai/grammar/parse"
)

var (
	grammarR = Rule("grammar")
	stmt     = Rule("stmt")
	comment  = Rule("comment")
	prod     = Rule("prod")
	term     = Rule("term")
	atom     = Rule("atom")
	quant    = Rule("quant")
	ident    = Rule("ident")
	str      = Rule("str")
	intR     = Rule("int")
	re       = Rule("re")

	// WrapRE is a special rule to indicate a wrapper around all regexps and
	// strings. When supplied in the form "pre()post", then all regexes will be
	// wrapped in "pre(?:" and ")post" and all strings will be escaped using
	// regexp.QuoteMeta then likewise wrapped.
	WrapRE = Rule(".wrapRE")
)

const grammarGrammarSrc = `
// Non-terminals
grammar -> stmt+;
stmt    -> comment | prod;
comment -> /\/\/.*$|(?s:\/\*(?:[^*]|\*+[^*\/])\*\/)/;
prod    -> ident "->" term+ ";";
term    -> term:"^"
         ^ term:"|"
         ^ term+
         ^ ("<" ident ">")? term
         ^ atom quant?;
atom    -> ident | str | re | "(" term ")";
quant   -> /[?*+]/
         | "{" int? "," int? "}"
         | /<:|:>?/ "!"? atom "!"?;

// Terminals
ident   -> /[A-Za-z_\.]\w*/;
str     -> /"((?:[^"\\]|\\.)*)"/;
int     -> /\d+/;
re      -> /\/((?:[^\/\\]|\\.)*)\//;
.wrapRE -> /\s*()\s*/;
`

var grammarGrammar = Grammar{
	grammarR: Some(stmt),
	stmt:     Oneof{comment, prod},
	comment:  RE(`//.*$|(?s:/\*(?:[^*]|\*+[^*/])\*/)`),
	prod:     Seq{ident, S("->"), Some(term), S(";")},
	term: Tower{
		Delim{Term: term, Sep: S("^")},
		Delim{Term: term, Sep: S("|")},
		Some(term),
		Seq{Opt(Seq{S("<"), ident, S(">")}), term},
		Seq{atom, Opt(quant)},
	},
	atom: Oneof{ident, str, re, Seq{S("("), term, S(")")}},
	quant: Oneof{
		RE(`[?*+]`),
		Seq{S("{"), Opt(intR), S(","), Opt(intR), S("}")},
		Seq{RE(`<:|:>?`), Opt(S("!")), atom, Opt(S("!"))},
	},

	ident:  RE(`[A-Za-z_\.]\w*`),
	str:    RE(`"((?:[^"\\]|\\.)*)"`),
	intR:   RE(`\d+`),
	re:     RE(`/((?:[^/\\]|\\.)*)/`),
	WrapRE: RE(`\s*()\s*`),
}

func nodeRule(v interface{}) Rule {
	tag := v.(parse.Node).Tag
	backslash := strings.IndexRune(tag, '\\')
	return Rule(tag[:backslash])
}

type Grammar map[Rule]Term

// Build the grammar grammar from grammarGrammarSrc and check that it matches
// GrammarGrammar.
var core = func() Parsers {
	parsers := grammarGrammar.Compile()

	r := parse.NewScanner(grammarGrammarSrc)
	v, err := parsers.Parse(grammarR, r)
	if err != nil {
		panic(err)
	}
	if err := parsers.Grammar().ValidateParse(v); err != nil {
		panic(err)
	}
	g := v.(parse.Node)

	newGrammarGrammar := NewFromNode(g)

	if !reflect.DeepEqual(newGrammarGrammar, grammarGrammar) {
		panic(fmt.Errorf("mismatch between parsed and bootstrap grammar"))
	}

	return newGrammarGrammar.Compile()
}()

func Core() Parsers {
	return core
}

// ValidateParse performs numerous checks on a generated AST to ensure it
// conforms to the parser that generated it. It is useful for testing the
// parser engine, but also for any tools that synthesise parser output.
func (g Grammar) ValidateParse(v interface{}) error {
	rule := nodeRule(v)
	return g[rule].ValidateParse(g, rule, v)
}

// Unparse inverts the action of a parser, taking a generated AST and producing
// the source it came from. Currently, it doesn't quite do that, and is only
// being used for quick eyeballing to validate output.
func (g Grammar) Unparse(v interface{}, w io.Writer) (n int, err error) {
	rule := nodeRule(v)
	return g[rule].Unparse(g, v, w)
}

// Parsers holds Parsers generated by Grammar.Compile.
type Parsers struct {
	parsers map[Rule]parse.Parser
	grammar Grammar
}

func (p Parsers) Grammar() Grammar {
	return p.grammar
}

func (p Parsers) ValidateParse(v interface{}) error {
	return p.grammar.ValidateParse(v)
}

func (p Parsers) Unparse(v interface{}, w io.Writer) (n int, err error) {
	return p.grammar.Unparse(v, w)
}

// Parse parses some source per a given rule.
func (p Parsers) Parse(rule Rule, input *parse.Scanner) (interface{}, error) {
	var v interface{}
	if p.parsers[rule].Parse(input, &v) {
		if input.String() == "" {
			return v, nil
		}
		return nil, fmt.Errorf("unconsumed input: %q", input.String())
	}
	return nil, fmt.Errorf("failed to parse %s", rule)
}

// Term represents the terms of a grammar specification.
type Term interface {
	fmt.Stringer
	Parser(name Rule, c cache) parse.Parser
	ValidateParse(g Grammar, rule Rule, v interface{}) error
	Unparse(g Grammar, v interface{}, w io.Writer) (n int, err error)
	Resolve(oldRule, newRule Rule) Term
}

type Associativity int

func NewAssociativity(s string) Associativity {
	switch s {
	case ":":
		return NonAssociative
	case ":>":
		return LeftToRight
	case "<:":
		return RightToLeft
	}
	panic(BadInput)
}

func (a Associativity) String() string {
	switch {
	case a < 0:
		return "<:"
	case a > 0:
		return ":>"
	}
	return ":"
}

const (
	RightToLeft = iota - 1
	NonAssociative
	LeftToRight
)

type (
	Rule  string
	S     string
	RE    string
	Seq   []Term
	Oneof []Term
	Tower []Term
	Delim struct {
		Term            Term
		Sep             Term
		Assoc           Associativity
		CanStartWithSep bool
		CanEndWithSep   bool
	}
	Quant struct {
		Term Term
		Min  int
		Max  int // 0 = infinity
	}
	Named struct {
		Name string
		Term Term
	}
)

func NonAssoc(term, sep Term) Delim { return Delim{Term: term, Sep: sep, Assoc: NonAssociative} }
func L2R(term, sep Term) Delim      { return Delim{Term: term, Sep: sep, Assoc: LeftToRight} }
func R2L(term, sep Term) Delim      { return Delim{Term: term, Sep: sep, Assoc: RightToLeft} }

func Opt(term Term) *Quant  { return &Quant{Term: term, Max: 1} }
func Any(term Term) *Quant  { return &Quant{Term: term} }
func Some(term Term) *Quant { return &Quant{Term: term, Min: 1} }

func Name(name string, term Term) Named {
	return Named{Name: name, Term: term}
}

func join(terms []Term, sep string) string {
	s := []string{}
	for _, t := range terms {
		s = append(s, t.String())
	}
	return strings.Join(s, sep)
}

func (g Grammar) String() string {
	keys := make([]string, 0, len(g))
	for key := range g {
		keys = append(keys, string(key))
	}
	sort.Strings(keys)

	var sb strings.Builder
	count := 0
	for _, key := range keys {
		if count > 0 {
			sb.WriteString("; ")
		}
		fmt.Fprintf(&sb, "%s -> %v", key, g[Rule(key)])
		count++
	}
	return sb.String()
}

func (t Rule) String() string  { return string(t) }
func (t S) String() string     { return fmt.Sprintf("%q", string(t)) }
func (t RE) String() string    { return fmt.Sprintf("/%v/", string(t)) }
func (t Seq) String() string   { return join(t, " ") }
func (t Oneof) String() string { return join(t, " | ") }
func (t Tower) String() string { return join(t, " >> ") }
func (t Delim) String() string { return fmt.Sprintf("%v%s%v", t.Term, t.Assoc, t.Sep) }
func (t Quant) String() string { return fmt.Sprintf("%v{%d,%d}", t.Term, t.Min, t.Max) }
func (t Named) String() string { return fmt.Sprintf("<%s>%v", t.Name, t.Term) }
