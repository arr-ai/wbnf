package wbnf

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/arr-ai/frozen"

	"github.com/arr-ai/wbnf/errors"
	"github.com/arr-ai/wbnf/parser"
)

const (
	StackDelim = "@"
	at         = Rule(StackDelim)

	seqTag    = "_"
	oneofTag  = "|"
	delimTag  = ":"
	quantTag  = "?"
	externTag = "%"
)

type cache struct {
	parsers    map[Rule]parser.Parser
	grammar    Grammar
	rulePtrses map[Rule][]*parser.Parser
}

func (c cache) registerRule(parser *parser.Parser) {
	if rule, ok := (*parser).(ruleParser); ok {
		c.rulePtrses[rule.t] = append(c.rulePtrses[rule.t], parser)
	}
}

func (c cache) registerRules(parsers []parser.Parser) {
	for i := range parsers {
		c.registerRule(&parsers[i])
	}
}

func (c cache) makeParsers(terms []Term) []parser.Parser {
	parsers := make([]parser.Parser, 0, len(terms))
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

type putter func(output *parser.TreeElement, extra parser.Extra, children ...parser.TreeElement) error

type scopeVal struct {
	p   parser.Parser
	val parser.TreeElement
}

func NewScopeWith(s frozen.Map, ident string, p parser.Parser, val parser.TreeElement) frozen.Map {
	if ident == "" {
		return s
	}
	return s.With(ident, &scopeVal{p, val})
}
func valFrom(s frozen.Map, ident string) (*scopeVal, bool) {
	if val, ok := s.Get(ident); ok {
		return val.(*scopeVal), ok
	}
	return nil, false
}

func tag(rule Rule, alt Rule) putter {
	rule = ruleOrAlt(rule, alt)
	return func(output *parser.TreeElement, extra parser.Extra, children ...parser.TreeElement) error {
		*output = parser.Node{
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
func (g Grammar) Compile(node *parser.Node) Parsers {
	for _, term := range g {
		if _, ok := term.(Stack); ok {
			g = g.clone()
			g.resolveStacks()
			break
		}
	}

	c := cache{
		parsers:    map[Rule]parser.Parser{},
		grammar:    g,
		rulePtrses: map[Rule][]*parser.Parser{},
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
		parsers:    c.parsers,
		grammar:    g,
		node:       node,
		singletons: g.singletons(),
	}
}

func Compile(grammar string) (Parsers, error) {
	v, err := Core().Parse(GrammarRule, parser.NewScanner(grammar))
	if err != nil {
		return Parsers{}, err
	}
	node := v.(parser.Node)
	return NewFromNode(node).Compile(&node), nil
}

func MustCompile(grammar string) Parsers {
	p, err := Compile(grammar)
	if err != nil {
		panic(err)
	}
	return p
}

//-----------------------------------------------------------------------------

type ruleParser struct {
	rule Rule
	t    Rule
}

func (p ruleParser) Parse(scope frozen.Map, input *parser.Scanner, output *parser.TreeElement) error {
	panic(errors.Inconceivable)
}

func (t Rule) Parser(rule Rule, c cache) parser.Parser {
	return ruleParser{
		rule: rule,
		t:    t,
	}
}

//-----------------------------------------------------------------------------

func getErrorStrings(input *parser.Scanner) string {
	text := input.String()
	if len(text) > 40 {
		text = text[:40] + "  ..."
	}

	return parser.NewScanner(text).Context()
}

func eatRegexp(input *parser.Scanner, re *regexp.Regexp, output *parser.TreeElement) bool {
	var eaten [2]parser.Scanner
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

func (p *sParser) Parse(scope frozen.Map, input *parser.Scanner, output *parser.TreeElement) error {
	if ok := eatRegexp(input, p.re, output); !ok {
		return newParseError(p.rule, "",
			fmt.Errorf("expect: %s", parser.NewScanner(p.t.String()).Context()),
			fmt.Errorf("actual: %s", getErrorStrings(input)))
	}
	return nil
}

func (t S) Parser(rule Rule, c cache) parser.Parser {
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

func (p *reParser) Parse(_ frozen.Map, input *parser.Scanner, output *parser.TreeElement) error {
	if ok := eatRegexp(input, p.re, output); !ok {
		return newParseError(p.rule, "",
			fmt.Errorf("expect: %s", parser.NewScanner(p.re.String()).Context()),
			fmt.Errorf("actual: %s", getErrorStrings(input)))
	}
	return nil
}

func (t RE) Parser(rule Rule, c cache) parser.Parser {
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
	parsers []parser.Parser
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
		case parser.Node:
			b := b.(parser.Node)
			if a.Count() == b.Count() {
				for i := range a.Children {
					if !nodesEqual(a.Children[i], b.Children[i]) {
						return false
					}
				}
			}
			return true
		case parser.Scanner:
			b := b.(parser.Scanner)
			if a.String() == b.String() {
				return true
			}
		}
	}
	return false
}

func (p *seqParser) Parse(scope frozen.Map, input *parser.Scanner, output *parser.TreeElement) (out error) {
	defer enterf("%s: %T %[2]v", p.rule, p.t).exitf("%v %v", &out, output)
	result := make([]parser.TreeElement, 0, len(p.parsers))
	furthest := *input

	for i, item := range p.parsers {
		var v parser.TreeElement
		ident := identFromTerm(p.t[i])
		if err := item.Parse(scope, input, &v); err != nil {
			*input = furthest
			return err
		}
		scope = NewScopeWith(scope, ident, p.parsers[i], v)
		furthest = *input
		result = append(result, v)
	}
	return p.put(output, nil, result...)
}

func (t Seq) Parser(rule Rule, c cache) parser.Parser {
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
	term parser.Parser
	sep  parser.Parser
	put  putter
}

func parseAppend(p parser.Parser, scope frozen.Map,
	input *parser.Scanner, slice *[]parser.TreeElement, errOut *error) bool {
	var v parser.TreeElement
	if err := p.Parse(scope, input, &v); err != nil {
		*errOut = err
		return false
	}
	*slice = append(*slice, v)
	return true
}

type Empty struct{}

func (Empty) IsTreeElement() {}

func (p *delimParser) Parse(scope frozen.Map, input *parser.Scanner, output *parser.TreeElement) (out error) {
	defer enterf("%s: %T %[2]v", p.rule, p.t).exitf("%v %v", &out, output)
	var result []parser.TreeElement

	var parseErr error
	switch {
	case parseAppend(p.term, scope, input, &result, &parseErr):
	case p.t.CanStartWithSep:
		result = append(result, Empty{})
		if !parseAppend(p.sep, scope, input, &result, &parseErr) {
			return parseErr
		}
		scope = NewScopeWith(scope, identFromTerm(p.t.Sep), p.sep, result[len(result)-1])
		if !parseAppend(p.term, scope, input, &result, &parseErr) {
			return parseErr
		}
	default:
		return parseErr
	}

	s := *input
	scope = NewScopeWith(scope, identFromTerm(p.t.Term), p.term, result[len(result)-1])
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
		scope = NewScopeWith(scope, identFromTerm(p.t.Term), p.term, result[len(result)-1])
		s = *input
	}
	*input = s

	if n := len(result); n > 1 {
		switch p.t.Assoc {
		case LeftToRight:
			v := result[0]
			for i := 1; i < n; i += 2 {
				p.put(&v, Associativity(i/2), v, result[i], result[i+1]) //nolint:errcheck
			}
			*output = v
		case RightToLeft:
			v := result[n-1]
			for i := 1; i < n; i += 2 {
				j := n - 1 - i
				p.put(&v, Associativity(-j/2), result[j-1], result[j], v) //nolint:errcheck
			}
			*output = v
		}
	}

	return p.put(output, Associativity(0), result...)
}

func (t Delim) Parser(rule Rule, c cache) parser.Parser {
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

func (t Delim) LRTerms(node parser.Node) (left, right Term) {
	associativity := node.Extra.(Associativity)
	switch {
	case associativity < 0:
		return t.Term, t
	case associativity > 0:
		return t, t.Term
	}
	return t.Term, t.Term
}

//-----------------------------------------------------------------------------

type quantParser struct {
	rule Rule
	t    Quant
	term parser.Parser
	put  putter
}

func (p *quantParser) Parse(scope frozen.Map, input *parser.Scanner, output *parser.TreeElement) (out error) {
	defer enterf("%s: %T %[2]v", p.rule, p.t).exitf("%v %v", &out, output)
	result := make([]parser.TreeElement, 0, p.t.Min)
	var v parser.TreeElement
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
	return out
}

func (t Quant) Parser(rule Rule, c cache) parser.Parser {
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
	parsers []parser.Parser
	put     putter
}

func (p *oneofParser) Parse(scope frozen.Map, input *parser.Scanner, output *parser.TreeElement) (out error) {
	defer enterf("%s: %T %[2]v", p.rule, p.t).exitf("%v %v", &out, output)
	furthest := *input

	var errors []error
	for i, par := range p.parsers {
		var v parser.TreeElement
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

func (t Oneof) Parser(rule Rule, c cache) parser.Parser {
	return &oneofParser{
		rule:    rule,
		t:       t,
		parsers: c.makeParsers(t),
		put:     tag(rule, oneofTag),
	}
}

//-----------------------------------------------------------------------------

func (t Stack) Parser(_ Rule, _ cache) parser.Parser {
	panic(errors.Inconceivable)
}

//-----------------------------------------------------------------------------

func (t Named) Parser(rule Rule, c cache) parser.Parser {
	return t.Term.Parser(Rule(t.Name), c)
}

//-----------------------------------------------------------------------------

func termFromRefVal(from parser.TreeElement) Term {
	var term Term
	switch n := from.(type) {
	case parser.Node:
		s := Seq{}
		for _, v := range n.Children {
			s = append(s, termFromRefVal(v))
		}
		term = s
	case parser.Scanner:
		term = S(n.String())
	}
	return term
}

type SubGrammar struct {
	parser.Node
}

func (SubGrammar) IsExtra() {}

func (t *REF) Parse(scope frozen.Map, input *parser.Scanner, output *parser.TreeElement) (out error) {
	var v parser.TreeElement
	if expected, ok := valFrom(scope, t.Ident); ok {
		term := termFromRefVal(expected.val)
		if t.External {
			if external, has := scope.Get(externalRef(t.Ident)); has {
				subgrammar, rule, err := external.(External)(term, expected.val, input)
				if err != nil {
					return newParseError(Rule(t.Ident), "External parse failed", err)
				}
				parsers := NewFromNode(subgrammar).Compile(&subgrammar)
				foreign, err := parsers.ParsePartial(rule, input, nil)
				if err != nil {
					return newParseError(Rule(t.Ident), "External parse failed", err)
				}
				v = parser.NewNode(externTag, SubGrammar{subgrammar}, foreign)
			} else {
				return newParseError(Rule(t.Ident), "External ref not found")
			}
		} else {
			if err := term.Parser(Rule(t.Ident), cache{}).Parse(scope, input, &v); err != nil {
				return err
			}
			if !nodesEqual(v, expected.val) {
				return newParseError(Rule(t.Ident), "Backref not matched",
					fmt.Errorf("expected: %s", expected),
					fmt.Errorf("actual: %s", v))
			}
		}
	} else if t.Default != nil {
		if err := t.Default.Parser(Rule(t.Ident), cache{}).Parse(scope, input, &v); err != nil {
			return err
		}
	} else {
		return newParseError(Rule(t.Ident), "Backref not found")
	}
	*output = v
	return nil
}

func (t REF) Parser(rule Rule, c cache) parser.Parser {
	return &t
}
