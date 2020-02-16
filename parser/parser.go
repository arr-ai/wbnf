package parser

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/arr-ai/frozen"
	"github.com/arr-ai/wbnf/errors"
)

const (
	StackDelim = "@"
	At         = Rule(StackDelim)

	seqTag    = "_"
	oneofTag  = "|"
	delimTag  = ":"
	quantTag  = "?"
	externTag = "%"
	WrapRE    = Rule(".wrapRE")
)

type cache struct {
	parsers    map[Rule]Parser
	grammar    Grammar
	rulePtrses map[Rule][]*Parser
}

func (c cache) registerRule(parser *Parser) {
	if rule, ok := (*parser).(ruleParser); ok {
		c.rulePtrses[rule.t] = append(c.rulePtrses[rule.t], parser)
	}
}

func (c cache) registerRules(parsers []Parser) {
	for i := range parsers {
		c.registerRule(&parsers[i])
	}
}

func (c cache) makeParsers(terms []Term) []Parser {
	parsers := make([]Parser, 0, len(terms))
	for _, t := range terms {
		parsers = append(parsers, t.Parser("", c))
	}
	c.registerRules(parsers)
	return parsers
}

func ruleOrAlt(rule Rule, alt Rule) Rule {
	if rule == "" {
		return alt
	}
	return rule
}

type putter func(output *TreeElement, extra Extra, children ...TreeElement) error

type scopeVal struct {
	p   Parser
	val TreeElement
}

type Scope struct {
	m frozen.Map
}

func (s Scope) String() string {
	return s.m.String()
}

func (s Scope) Keys() frozen.Set {
	return s.m.Keys()
}

func (s Scope) With(ident string, v interface{}) Scope {
	return Scope{m: s.m.With(ident, v)}
}

func (s Scope) Has(ident string) bool {
	return s.m.Has(ident)
}

func (s Scope) GetExternal(ident externalRef) (External, bool) {
	if v, has := s.m.Get(ident); has {
		return v.(External), true
	}
	return nil, false
}

func (s Scope) WithVal(ident string, p Parser, val TreeElement) Scope {
	if ident == "" {
		return s
	}
	return Scope{m: s.m.With(ident, &scopeVal{p: p, val: val})}
}

func (s Scope) GetVal(ident string) (Parser, TreeElement, bool) {
	if val, ok := s.m.Get(ident); ok {
		sv := val.(*scopeVal)
		return sv.p, sv.val, ok
	}
	return nil, nil, false
}

func tag(rule Rule, alt Rule) putter {
	rule = ruleOrAlt(rule, alt)
	return func(output *TreeElement, extra Extra, children ...TreeElement) error {
		*output = Node{
			Tag:      string(rule),
			Extra:    extra,
			Children: children,
		}
		return nil
	}
}

func (g Grammar) clone() Grammar {
	clone := make(Grammar, len(g))
	for rule, term := range g {
		clone[rule] = term
	}
	return clone
}

func (g Grammar) ResolveStacks() {
	for rule, term := range g {
		if stack, ok := term.(Stack); ok {
			oldRule := rule
			for i, layer := range stack {
				newRule := rule
				if j := (i + 1) % len(stack); j > 0 {
					newRule = Rule(fmt.Sprintf("%s%s%d", rule, StackDelim, j))
				}
				g[oldRule] = layer.Resolve(StackDelim, newRule)
				oldRule = newRule
			}
		}
	}
}

// Compile prepares a grammar for parsing. The parser holds a copy of the
// grammar modified to support parser execution.
func (g Grammar) Compile(node *Node) Parsers {
	for _, term := range g {
		if _, ok := term.(Stack); ok {
			g = g.clone()
			g.ResolveStacks()
			break
		}
	}

	c := cache{
		parsers:    map[Rule]Parser{},
		grammar:    g,
		rulePtrses: map[Rule][]*Parser{},
	}
	for rule, term := range g {
		for {
			switch r := term.(type) {
			case Rule:
				term = g[r]
				continue
			}
			break
		}
		c.parsers[rule] = term.Parser(rule, c)
	}

	for rule, rulePtrs := range c.rulePtrses {
		p := c.parsers[rule]
		for _, rulePtr := range rulePtrs {
			*rulePtr = p
		}
	}

	return Parsers{
		parsers: c.parsers,
		grammar: g,
		node:    node,
	}
}

//-----------------------------------------------------------------------------

type ruleParser struct {
	rule Rule
	t    Rule
}

func (p ruleParser) Parse(scope Scope, input *Scanner, output *TreeElement) error {
	panic(errors.Inconceivable)
}

func (t Rule) Parser(rule Rule, c cache) Parser {
	return ruleParser{
		rule: rule,
		t:    t,
	}
}

//-----------------------------------------------------------------------------

func getErrorStrings(input *Scanner) string {
	text := strings.TrimSpace(input.String())
	if len(text) > 40 {
		text = text[:40] + "  ..."
	}

	return NewScanner(text).Context()
}

func eatRegexp(input *Scanner, re *regexp.Regexp, output *TreeElement) bool {
	var eaten [2]Scanner
	if n, ok := input.EatRegexp(re, nil, eaten[:]); ok {
		*output = eaten[n-1]
		return true
	}
	return false
}

type sParser struct {
	rule Rule
	t    S
	re   *regexp.Regexp
}

func (p *sParser) Parse(scope Scope, input *Scanner, output *TreeElement) error {
	if ok := eatRegexp(input, p.re, output); !ok {
		return newParseError(p.rule, "",
			fmt.Errorf("expect: %s", NewScanner(p.t.String()).Context()),
			fmt.Errorf("actual: %s", getErrorStrings(input)))
	}
	return nil
}

func (t S) Parser(rule Rule, c cache) Parser {
	re := "(" + regexp.QuoteMeta(string(t)) + ")"
	if wrap, has := c.grammar[WrapRE]; has {
		re = strings.Replace(string(wrap.(RE)), "()", "(?:"+re+")", 1)
	}
	return &sParser{
		rule: rule,
		t:    t,
		re:   regexp.MustCompile(`(?m)\A` + re),
	}
}

type reParser struct {
	rule Rule
	t    RE
	re   *regexp.Regexp
}

func (p *reParser) Parse(_ Scope, input *Scanner, output *TreeElement) error {
	if ok := eatRegexp(input, p.re, output); !ok {
		return newParseError(p.rule, "",
			fmt.Errorf("expect: %s", NewScanner(p.re.String()).Context()),
			fmt.Errorf("actual: %s", getErrorStrings(input)))
	}
	return nil
}

func (t RE) Parser(rule Rule, c cache) Parser {
	re := "(" + string(t) + ")"
	if wrap, has := c.grammar[WrapRE]; has {
		re = strings.Replace(string(wrap.(RE)), "()", "(?:"+re+")", 1)
	}
	return &reParser{
		rule: rule,
		t:    t,
		re:   regexp.MustCompile(`(?m)\A` + re),
	}
}

//-----------------------------------------------------------------------------

type seqParser struct {
	rule    Rule
	t       Seq
	parsers []Parser
	put     putter
}

func identFromTerm(term Term) string {
	switch x := term.(type) {
	case Named:
		if x.Name != "" {
			return x.Name
		}
		return identFromTerm(x.Term)
	case Rule:
		return string(x)
	case Quant:
		return identFromTerm(x.Term)
	}
	return ""
}

func nodesEqual(a, b interface{}) bool {
	aType := reflect.TypeOf(a)
	bType := reflect.TypeOf(b)
	if aType == bType {
		switch a := a.(type) {
		case Node:
			b := b.(Node)
			if a.Count() == b.Count() {
				for i := range a.Children {
					if !nodesEqual(a.Children[i], b.Children[i]) {
						return false
					}
				}
			}
			return true
		case Scanner:
			b := b.(Scanner)
			if a.String() == b.String() {
				return true
			}
		}
	}
	return false
}

const dontAttemptSeqFix = "dont-attempt-seq-fix"

// Special helper function to try and determine if the sequence could be completed with simple missing strings
// The main purpose of this is to catch `missing ;` style errors at end-of-line
func (p *seqParser) attemptRuleCompletion(scope Scope, first, input *Scanner, failedIndex int) error {
	if scope.Has(dontAttemptSeqFix) {
		return nil
	}
	switch p.parsers[failedIndex].(type) {
	case *sParser:
	default:
		return nil
	}
	oldSlice := append([]Parser{}, p.parsers...)
	defer func() { p.parsers = oldSlice }()
	p.parsers = p.parsers[failedIndex+1:]
	var v TreeElement
	if p.Parse(scope, NewScanner(input.String()), &v) == nil {
		switch child := oldSlice[failedIndex].(type) {
		case *sParser:
			ctx := first.Slice(0, strings.Index(first.String(), input.String()))

			return possibleFixup(fmt.Sprintf("Missing '%s' @ %s", child.t, getErrorStrings(ctx)))
		}
	}
	return nil
}

func (p *seqParser) Parse(scope Scope, input *Scanner, output *TreeElement) (out error) {
	defer enterf("%s: %T %[2]v", p.rule, p.t).exitf("%v %v", &out, output)
	result := make([]TreeElement, 0, len(p.parsers))
	furthest := *input
	// first := NewScanner(input.String())

	for i, item := range p.parsers {
		var v TreeElement
		ident := identFromTerm(p.t[i])
		if err := item.Parse(scope, input, &v); err != nil {
			elist := []error{err}
			// if fixup := p.attemptRuleCompletion(scope, first, input, i); fixup != nil {
			// 	elist = append(elist, fixup)
			// }
			*input = furthest
			return newParseError(p.rule, "could not complete sequence", elist...)
		}
		scope = scope.WithVal(ident, p.parsers[i], v)
		furthest = *input
		result = append(result, v)
	}
	return p.put(output, nil, result...)
}

func (t Seq) Parser(rule Rule, c cache) Parser {
	return &seqParser{
		rule:    rule,
		t:       t,
		parsers: c.makeParsers(t),
		put:     tag(rule, seqTag),
	}
}

//-----------------------------------------------------------------------------

type delimParser struct {
	rule Rule
	t    Delim
	term Parser
	sep  Parser
	put  putter
}

func parseAppend(p Parser, scope Scope,
	input *Scanner, slice *[]TreeElement, errOut *error) bool {
	var v TreeElement
	if err := p.Parse(scope, input, &v); err != nil {
		*errOut = err
		return false
	}
	*slice = append(*slice, v)
	return true
}

type Empty struct{}

func (Empty) IsTreeElement() {}

func (p *delimParser) Parse(scope Scope, input *Scanner, output *TreeElement) (out error) {
	defer enterf("%s: %T %[2]v", p.rule, p.t).exitf("%v %v", &out, output)
	var result []TreeElement

	defer func(err *error) {
		if *err != nil {
			var errs []error
			if result != nil {
				var res TreeElement
				p.put(&res, Associativity(0), result...)
				errs = []error{errorNode{res}}
			}
			*err = newParseError(p.rule, "delim didnt complete", append(errs, *err)...)
		}
	}(&out)

	scope = scope.With(dontAttemptSeqFix, struct{}{})
	var parseErr error
	switch {
	case parseAppend(p.term, scope, input, &result, &parseErr):
	case p.t.CanStartWithSep:
		result = append(result, Empty{})
		if !parseAppend(p.sep, scope, input, &result, &parseErr) {
			return parseErr
		}
		scope = scope.WithVal(identFromTerm(p.t.Sep), p.sep, result[len(result)-1])
		if !parseAppend(p.term, scope, input, &result, &parseErr) {
			return parseErr
		}
	default:
		return parseErr
	}

	s := *input
	scope = scope.WithVal(identFromTerm(p.t.Term), p.term, result[len(result)-1])
	for parseAppend(p.sep, scope, input, &result, &parseErr) {
		if !parseAppend(p.term, scope, input, &result, &parseErr) {
			if p.t.CanEndWithSep {
				s = *input
				result = append(result, Empty{})
			} else {
				result = result[:len(result)-1]
			}
			break
		}
		scope = scope.WithVal(identFromTerm(p.t.Term), p.term, result[len(result)-1])
		s = *input
	}
	*input = s

	if n := len(result); n > 1 {
		switch p.t.Assoc {
		case LeftToRight:
			v := result[0]
			for i := 1; i < n; i += 2 {
				p.put(&v, p.t.Assoc, v, result[i], result[i+1]) //nolint:errcheck
			}
			*output = v
			return nil
		case RightToLeft:
			v := result[n-1]
			for i := 1; i < n; i += 2 {
				j := n - 1 - i
				p.put(&v, p.t.Assoc, result[j-1], result[j], v) //nolint:errcheck
			}
			*output = v
			return nil
		}
	}

	return p.put(output, Associativity(0), result...)
}

func (t Delim) Parser(rule Rule, c cache) Parser {
	p := &delimParser{
		rule: rule,
		t:    t,
		term: t.Term.Parser("", c),
		sep:  t.Sep.Parser("", c),
		put:  tag(rule, delimTag),
	}
	c.registerRule(&p.term)
	c.registerRule(&p.sep)
	return p
}

type lrtgen struct {
	sides   [2]Term
	sep     Term
	side    int
	sepnext bool
}

func (l *lrtgen) Next() Term {
	var out Term
	if l.sepnext {
		out = l.sep
		l.sepnext = !l.sepnext
	} else {
		out = l.sides[l.side%2]
		l.side++
		l.sepnext = true
	}
	return out
}
func (t Delim) LRTerms(node Node) lrtgen {
	associativity := node.Extra.(Associativity)
	switch {
	case associativity < 0:
		return lrtgen{sides: [2]Term{t.Term, t}, sep: t.Sep}
	case associativity > 0:
		return lrtgen{sides: [2]Term{t, t.Term}, sep: t.Sep}
	}
	return lrtgen{sides: [2]Term{t.Term, t.Term}, sep: t.Sep}
}

//-----------------------------------------------------------------------------

type quantParser struct {
	rule Rule
	t    Quant
	term Parser
	put  putter
}

func (p *quantParser) Parse(scope Scope, input *Scanner, output *TreeElement) (out error) {
	defer enterf("%s: %T %[2]v", p.rule, p.t).exitf("%v %v", &out, output)
	result := make([]TreeElement, 0, p.t.Min)
	var v TreeElement
	start := *input
	for i := 0; p.t.Max == 0 || i < p.t.Max; i++ {
		if out = p.term.Parse(scope, &start, &v); out != nil {
			break
		}
		result = append(result, v)
		*input = start
	}

	if len(result) >= p.t.Min {
		return p.put(output, nil, result...)
	}

	return newParseError(p.rule,
		fmt.Sprintf("quant failed, expected: %v, have %d value(s)", p.t, len(result)), out)
}

func (t Quant) Parser(rule Rule, c cache) Parser {
	p := &quantParser{
		rule: rule,
		t:    t,
		term: t.Term.Parser("", c),
		put:  tag(rule, quantTag),
	}
	c.registerRule(&p.term)
	return p
}

//-----------------------------------------------------------------------------

type oneofParser struct {
	rule    Rule
	t       Oneof
	parsers []Parser
	put     putter
}

func (p *oneofParser) Parse(scope Scope, input *Scanner, output *TreeElement) (out error) {
	defer enterf("%s: %T %[2]v", p.rule, p.t).exitf("%v %v", &out, output)
	furthest := *input

	var errors []error
	for i, par := range p.parsers {
		var v TreeElement
		start := *input
		if err := par.Parse(scope, &start, &v); err != nil {
			errors = append(errors, err)

			if furthest.Offset() < start.Offset() {
				furthest = start
			}
		} else {
			*input = start
			return p.put(output, Choice(i), v)
		}
	}
	*input = furthest
	return newParseError(p.rule, "None of the available options could be satisfied", errors...)
}

func (t Oneof) Parser(rule Rule, c cache) Parser {
	return &oneofParser{
		rule:    rule,
		t:       t,
		parsers: c.makeParsers(t),
		put:     tag(rule, oneofTag),
	}
}

//-----------------------------------------------------------------------------

func (t Stack) Parser(_ Rule, _ cache) Parser {
	panic(errors.Inconceivable)
}

//-----------------------------------------------------------------------------

func (t Named) Parser(rule Rule, c cache) Parser {
	return t.Term.Parser(Rule(t.Name), c)
}

//-----------------------------------------------------------------------------

func termFromRefVal(from TreeElement) Term {
	var term Term
	switch n := from.(type) {
	case Node:
		s := Seq{}
		for _, v := range n.Children {
			s = append(s, termFromRefVal(v))
		}
		term = s
	case Scanner:
		term = S(n.String())
	}
	return term
}

func (t *REF) Parse(scope Scope, input *Scanner, output *TreeElement) error {
	if t.External {
		if external, has := scope.GetExternal(externalRef(t.Ident)); has {
			foreigner, subgrammar, err := external(scope, input, false)
			if err != nil {
				return newParseError(Rule(t.Ident), "External parse failed", err)
			}
			if foreigner != nil {
				*output = NewNode(externTag, SubGrammar{subgrammar}, foreigner)
			} else {
				*output = NewNode(externTag, nil)
			}
			return nil
		}
		return newParseError(Rule(t.Ident), "External ref handler not found")
	}
	if parser, val, has := scope.GetVal(t.Ident); has {
		term := termFromRefVal(val)
		if err := term.Parser(Rule(t.Ident), cache{}).Parse(scope, input, output); err != nil {
			return err
		}
		if !nodesEqual(*output, val) {
			return newParseError(Rule(t.Ident), "Backref not matched",
				fmt.Errorf("expected: parser=%s, val=%s", parser, val),
				fmt.Errorf("actual: %s", *output))
		}
		return nil
	}
	if t.Default != nil {
		return t.Default.Parser(Rule(t.Ident), cache{}).Parse(scope, input, output)
	}
	return newParseError(Rule(t.Ident), "Backref not found")
}

func (t REF) Parser(rule Rule, c cache) Parser {
	return &t
}
