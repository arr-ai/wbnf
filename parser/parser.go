package parser

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/arr-ai/wbnf/errors"
)

const (
	StackDelim = "@"
	At         = Rule(StackDelim)

	seqTag   = "_"
	oneofTag = "|"
	delimTag = ":"
	quantTag = "?"
	WrapRE   = Rule(".wrapRE")
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

func (g Grammar) resolveStacks() {
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
func (g Grammar) Compile(node interface{}) Parsers {
	for _, term := range g {
		if _, ok := term.(Stack); ok {
			g = g.clone()
			g.resolveStacks()
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
func parseEscape(p Parser, scope Scope, ident string, t Term, input *Scanner, output *TreeElement) (bool, error) {
	if esc := scope.getParserEscape(); esc != nil {
		var match Scanner
		if _, ok := input.EatRegexp(esc.openDelim, &match, nil); ok {
			if ident != "" {
				scope = scope.With(ident, t)
			}
			te, err := esc.external(scope.With("(term)", p.AsTerm()), input)
			if err != nil {
				unconsumed, ok := err.(UnconsumedInputError)
				if !ok {
					return false, err
				}
				te = unconsumed.tree
				*input = unconsumed.residue
			}
			// FIXME: We cant verify that the te returned matches what the grammar requires
			// this means we cant safely convert this to an ast later.
			*output = te
			if _, ok := input.EatRegexp(esc.closeDelim, &match, nil); !ok {
				return false, fmt.Errorf("missing escape terminator")
			}
			return true, nil
		}
	}
	return false, nil
}

//-----------------------------------------------------------------------------
type ruleParser struct {
	rule Rule
	t    Rule
}

func (p ruleParser) Parse(scope Scope, input *Scanner, output *TreeElement, stk *call) error {
	panic(errors.Inconceivable)
}
func (p ruleParser) AsTerm() Term { return p.t }

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

	return NewScanner(text).Context(DefaultLimit)
}

func eatRegexp(input *Scanner, re *regexp.Regexp, output *TreeElement) bool {
	var eaten [2]Scanner
	if n, ok := input.EatRegexp(re, nil, eaten[:]); ok {
		*output = eaten[n-1]
		return true
	}
	return false
}

func applyWrapRE(re string, prepare func(string) string, c cache) string {
	pre := prepare(re)
	if wrap, has := c.grammar[WrapRE]; has {
		if oneof, ok := wrap.(Oneof); ok {
			for _, t := range oneof[:len(oneof)-1] {
				switch t := t.(type) {
				case S:
					if string(t) == re {
						return pre
					}
				case RE:
					if string(t) == re {
						return pre
					}
				}
			}
			wrap = oneof[len(oneof)-1]
		}
		return strings.Replace(string(wrap.(RE)), "()", "(?:"+pre+")", 1)
	}
	return pre
}

type sParser struct {
	rule Rule
	t    S
	re   *regexp.Regexp
}

func (p *sParser) Parse(scope Scope, input *Scanner, output *TreeElement, stk *call) error { //nolint:dupl
	if escaped, err := parseEscape(p, scope, string(p.rule), p.t, input, output); escaped || err != nil {
		return err
	}
	if ok := eatRegexp(input, p.re, output); !ok {
		return newParseError(p.rule, "")(scope.GetCutPoint(),
			func() error { return fmt.Errorf("expect: %s", NewScanner(p.t.String()).Context(DefaultLimit)) },
			func() error { return fmt.Errorf("actual: %s", getErrorStrings(input)) },
			func() error { return stk },
		)
	}
	return nil
}
func (p *sParser) AsTerm() Term { return p.t }

func (t S) Parser(rule Rule, c cache) Parser {
	re := applyWrapRE(string(t), func(re string) string { return "(" + regexp.QuoteMeta(re) + ")" }, c)
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

func (p *reParser) Parse(scope Scope, input *Scanner, output *TreeElement, stk *call) error { //nolint:dupl
	if escaped, err := parseEscape(p, scope, string(p.rule), p.t, input, output); escaped || err != nil {
		return err
	}
	if ok := eatRegexp(input, p.re, output); !ok {
		return newParseError(p.rule, "")(scope.GetCutPoint(),
			func() error { return fmt.Errorf("expect: %s", NewScanner(p.re.String()).Context(DefaultLimit)) },
			func() error { return fmt.Errorf("actual: %s", getErrorStrings(input)) },
			func() error { return stk },
		)
	}
	return nil
}
func (p *reParser) AsTerm() Term { return p.t }

func (t RE) Parser(rule Rule, c cache) Parser {
	re := applyWrapRE(string(t), func(re string) string { return "(" + re + ")" }, c)
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

func (p *seqParser) Parse(scope Scope, input *Scanner, output *TreeElement, stk *call) (out error) {
	// defer enterf("%s: %T %[2]v", p.rule, p.t).exitf("%v %v", &out, output)
	if escaped, err := parseEscape(p, scope, "", nil, input, output); escaped || err != nil {
		return err
	}
	result := make([]TreeElement, 0, len(p.parsers))
	furthest := *input

	for i, item := range p.parsers {
		var v TreeElement
		ident := identFromTerm(p.t[i])
		if err := item.Parse(scope, input, &v, stk.push(ident, item.AsTerm())); err != nil {
			if isFatal(err) {
				return err
			}
			*input = furthest
			return newParseError(p.rule, "could not complete sequence")(scope.GetCutPoint(),
				func() error { return err },
				func() error { return stk },
			)
		}
		if _, ok := item.(*cutPointParser); ok {
			scope, _, _ = scope.ReplaceCutPoint(true)
		}
		scope = scope.WithVal(ident, p.parsers[i], v)
		furthest = *input
		result = append(result, v)
	}
	return p.put(output, nil, result...)
}
func (p *seqParser) AsTerm() Term { return p.t }

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
	rule  Rule
	t     Delim
	child Parser
	put   putter
}

type Empty struct{}

func (Empty) IsTreeElement() {}

func (p *delimParser) Parse(scope Scope, input *Scanner, output *TreeElement, stk *call) (out error) {
	// defer enterf("%s: %T %[2]v", p.rule, p.t).exitf("%v %v", &out, output)
	if escaped, err := parseEscape(p, scope, "", nil, input, output); escaped || err != nil {
		return err
	}
	result := []TreeElement{}

	stk = stk.push(string(p.rule), p.AsTerm())

	if out := p.child.Parse(scope, input, output, stk); out != nil {
		return out
	}

	var seq Node
	var final TreeElement
	if p.t.CanStartWithSep {
		if x := (*output).(Node).GetNode(0).Children; len(x) != 0 {
			result = append(result, []TreeElement{Empty{}, x[0]}...)
		}
		result = append(result, (*output).(Node).Get(1, 0)) // term
		seq = (*output).(Node).GetNode(1, 1)
		if p.t.CanEndWithSep {
			final = (*output).(Node).Get(2)
		}
	} else {
		result = append(result, (*output).(Node).Get(0, 0)) // term
		seq = (*output).(Node).GetNode(0, 1)
		if p.t.CanEndWithSep {
			final = (*output).(Node).Get(1)
		}
	}

	for _, child := range seq.Children {
		child := child.(Node)
		result = append(result, child.Get(0)) // sep
		result = append(result, child.Get(1)) // term
	}

	if final != nil && final.(Node).Count() > 0 {
		result = append(result, []TreeElement{final.(Node).Get(0), Empty{}}...)
	}

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
func (p *delimParser) AsTerm() Term { return p.t }

func (t Delim) Parser(rule Rule, c cache) Parser {
	// Convert the delim to the equivalent sequence.
	// a -> x:y    ===   a -> x (y x)*
	// a -> x:,y   ===   a -> y? x (y x)*
	// a -> x:y,   ===   a -> x (y x)* y?
	seq := Seq{}
	if t.CanStartWithSep {
		seq = append(seq, Opt(t.Sep))
	}
	seq = append(seq, Seq{t.Term, Any(Seq{t.Sep, t.Term})})
	if t.CanEndWithSep {
		seq = append(seq, Opt(t.Sep))
	}

	p := &delimParser{
		rule:  rule,
		t:     t,
		child: seq.Parser(rule, c),
		put:   tag(rule, delimTag),
	}
	c.registerRule(&p.child)

	return p
}

type LRTGen struct {
	sides   [2]Term
	sep     Term
	side    int
	sepnext bool
}

func (l *LRTGen) Next() Term {
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
func (t Delim) LRTerms(node Node) LRTGen {
	associativity := node.Extra.(Associativity)
	switch {
	case associativity < 0:
		return LRTGen{sides: [2]Term{t.Term, t}, sep: t.Sep}
	case associativity > 0:
		return LRTGen{sides: [2]Term{t, t.Term}, sep: t.Sep}
	}
	return LRTGen{sides: [2]Term{t.Term, t.Term}, sep: t.Sep}
}

//-----------------------------------------------------------------------------

type quantParser struct {
	rule Rule
	t    Quant
	term Parser
	put  putter
}

func (p *quantParser) Parse(scope Scope, input *Scanner, output *TreeElement, stk *call) (out error) {
	// defer enterf("%s: %T %[2]v", p.rule, p.t).exitf("%v %v", &out, output)
	if escaped, err := parseEscape(p, scope, "", nil, input, output); escaped || err != nil {
		return err
	}
	result := make([]TreeElement, 0, p.t.Min)
	var v TreeElement
	start := *input

	stk = stk.push(string(p.rule), p.AsTerm())

	scope, prevcp, mycp := scope.ReplaceCutPoint(false)
	for p.t.Max == 0 || len(result) < p.t.Max {
		if out = p.term.Parse(scope, &start, &v, stk); out != nil {
			if isNotMyFatalError(out, mycp) {
				return out
			}
			break
		}
		result = append(result, v)
		*input = start
	}

	if len(result) >= p.t.Min {
		return p.put(output, nil, result...)
	}

	return newParseError(p.rule,
		"quant failed, expected: (%d, %d), have %d value(s)",
		p.t.Min, p.t.Max, len(result),
	)(prevcp, func() error { return out }, func() error { return stk })
}
func (p *quantParser) AsTerm() Term { return p.t }

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

func (p *oneofParser) Parse(scope Scope, input *Scanner, output *TreeElement, stk *call) (out error) {
	// defer enterf("%s: %T %[2]v", p.rule, p.t).exitf("%v %v", &out, output)
	if escaped, err := parseEscape(p, scope, "", nil, input, output); escaped || err != nil {
		return err
	}
	furthest := *input

	stk = stk.push(string(p.rule), p.AsTerm())
	scope, prevcp, mycp := scope.ReplaceCutPoint(false)
	var errors []func() error
	for i, par := range p.parsers {
		var v TreeElement
		start := *input
		if err := par.Parse(scope, &start, &v, stk); err != nil {
			if isNotMyFatalError(err, mycp) {
				return err
			}
			errors = append(errors, func() error { return err })

			if furthest.Offset() < start.Offset() {
				furthest = start
			}
		} else {
			*input = start
			return p.put(output, Choice(i), v)
		}
	}
	errors = append(errors, func() error { return stk })
	*input = furthest
	return newParseError(p.rule, "None of the available options could be satisfied")(prevcp, errors...)
}
func (p *oneofParser) AsTerm() Term { return p.t }

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
func (t Stack) Term() Term { return t }

//-----------------------------------------------------------------------------

func (t Named) Parser(rule Rule, c cache) Parser {
	return t.Term.Parser(Rule(t.Name), c)
}
func (t Named) AsTerm() Term { return t }

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

func (t *REF) Parse(scope Scope, input *Scanner, output *TreeElement, stk *call) (out error) {
	if escaped, err := parseEscape(t, scope, "", nil, input, output); escaped || err != nil {
		return err
	}
	stk = stk.push(t.Ident, t.AsTerm())
	var v TreeElement
	if _, expected, ok := scope.GetVal(t.Ident); ok {
		term := termFromRefVal(expected)
		parser := term.Parser(Rule(t.Ident), cache{})
		if err := parser.Parse(scope, input, &v, stk); err != nil {
			return err
		}
		if !nodesEqual(v, expected) {
			return newParseError(Rule(t.Ident), "Backref not matched")(invalidCutpoint,
				func() error { return fmt.Errorf("expected: %s", expected) },
				func() error { return fmt.Errorf("actual: %s", v) },
				func() error { return stk },
			)
		}
	} else if t.Default != nil {
		if err := t.Default.Parser(Rule(t.Ident), cache{}).Parse(scope, input, &v, stk); err != nil {
			return err
		}
	} else {
		return newParseError(Rule(t.Ident), "Backref not found")(invalidCutpoint,
			func() error { return stk },
		)
	}
	*output = v
	return nil
}

func (t REF) Parser(rule Rule, c cache) Parser {
	return &t
}
func (t REF) AsTerm() Term { return t }

func (t ExtRef) Parse(scope Scope, input *Scanner, output *TreeElement, stk *call) (out error) {
	if escaped, err := parseEscape(t, scope, "", nil, input, output); escaped || err != nil {
		return err
	}
	fn := scope.GetExternal(string(t))
	if fn == nil {
		return newParseError(Rule(string(t)), "External handler not found")(Cutpointdata(1),
			func() error { return stk.push(string(t), t.AsTerm()) },
		)
	}
	*output, out = fn(scope, input)
	return out
}

func (t ExtRef) Parser(name Rule, c cache) Parser {
	return t
}
func (t ExtRef) AsTerm() Term { return t }

//-----------------------------------------------------------------------------

func (t ScopedGrammar) Parser(name Rule, c cache) Parser {
	for _, term := range t.Grammar {
		if _, ok := term.(Stack); ok {
			t.Grammar = t.Grammar.clone()
			t.Grammar.resolveStacks()
			break
		}
	}
	if wrap, has := c.grammar[WrapRE]; has {
		if _, has := t.Grammar[WrapRE]; !has {
			t.Grammar[WrapRE] = wrap
		}
	}

	cc := cache{
		parsers:    map[Rule]Parser{},
		grammar:    t.Grammar,
		rulePtrses: map[Rule][]*Parser{},
	}
	for rule, term := range t.Grammar {
		for {
			switch r := term.(type) {
			case Rule:
				term = t.Grammar[r]
				continue
			}
			break
		}
		cc.parsers[rule] = term.Parser(rule, cc)
	}

	// At this point we have the nested grammar cache populated with the grammar rules
	// We now need to hook up the correct terms from the original grammar

	result := t.Term.Parser(name, cc)
	cc.registerRule(&result)
	for rule, rulePtrs := range cc.rulePtrses {
		if _, has := t.Grammar[rule]; has {
			// local rule, simply hook up the pointers
			p := cc.parsers[rule]
			for _, rulePtr := range rulePtrs {
				*rulePtr = p
			}
		} else {
			// must be from the previous scope, add it to the previous scopes cache
			for _, rulePtr := range rulePtrs {
				c.registerRule(rulePtr)
			}
		}
	}
	return result
}

type cutPointParser struct {
	p Parser
	t CutPoint
}

func (t *cutPointParser) Parse(scope Scope, input *Scanner, output *TreeElement, stk *call) (out error) {
	return t.p.Parse(scope, input, output, stk)
}
func (t *cutPointParser) AsTerm() Term { return t.t }
func (t CutPoint) Parser(rule Rule, c cache) Parser {
	return &cutPointParser{t.Term.Parser(rule, c), t}
}
