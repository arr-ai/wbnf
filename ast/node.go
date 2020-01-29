package ast

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/arr-ai/wbnf/errors"
	"github.com/arr-ai/wbnf/parser"
	"github.com/arr-ai/wbnf/wbnf"
)

const (
	seqTag   = "_"
	oneofTag = "|"
	delimTag = ":"
	quantTag = "?"

	RuleTag   = "@rule"
	ChoiceTag = "@choice"
	SkipTag   = "@skip"
)

var coreNode = func() Node {
	return ParserNodeToNode(wbnf.Core().Grammar(), *wbnf.Core().Node())
}()

func CoreNode() Node {
	return coreNode
}

func ParserNodeToNode(g wbnf.Grammar, v interface{}) Branch {
	rule := wbnf.NodeRule(v)
	term := g[rule]
	result := Branch{}
	result.one("@rule", Extra{rule})
	result.fromTerm(g, term, newCounters(term), v)
	return result
}

func NodeToParserNode(g wbnf.Grammar, branch Branch) interface{} {
	branch = branch.clone().(Branch)
	rule := branch.pullOne(RuleTag).(Extra).Data.(wbnf.Rule)
	term := g[rule]
	ctrs := newCounters(term)
	return relabelNode(string(rule), branch.toTerm(g, term, ctrs))
}

func relabelNode(name string, v interface{}) interface{} {
	if n, ok := v.(parser.Node); ok {
		n.Tag = name
		return n
	}
	return v
}

type Children interface {
	fmt.Stringer
	Scanner() parser.Scanner
	isChildren()
	clone() Children
	narrow() bool
}

func (One) isChildren()  {}
func (Many) isChildren() {}

type One struct {
	Node Node
}

type Many []Node

type Node interface {
	fmt.Stringer
	One(name string) Node
	MustOne(name string) Node
	Many(name string) []Node
	MustMany(name string) []Node
	Scanner() parser.Scanner
	collapse(level int) Node
	isNode()
	clone() Node
	narrow() bool
}

func (Branch) isNode() {}
func (Leaf) isNode()   {}
func (Extra) isNode()  {}

type Leaf parser.Scanner

func (Leaf) One(_ string) Node {
	return nil
}

func (Leaf) MustOne(_ string) Node {
	panic(errors.Inconceivable)
}

func (Leaf) Many(_ string) []Node {
	return nil
}

func (Leaf) MustMany(_ string) []Node {
	panic(errors.Inconceivable)
}

func (l Leaf) collapse(level int) Node {
	return l
}

type Branch map[string]Children

func (n Branch) One(name string) Node {
	if c, has := n[name]; has {
		if one, ok := c.(One); ok {
			return one.Node
		}
	}
	return nil
}

func (n Branch) MustOne(name string) Node {
	if c, has := n[name]; has {
		if one, ok := c.(One); ok {
			return one.Node
		}
		panic(fmt.Errorf("not a One"))
	}
	panic(fmt.Errorf("name %q not found", name))
}

func (n Branch) Many(name string) []Node {
	if c, has := n[name]; has {
		if many, ok := c.(Many); ok {
			return many
		}
	}
	return nil
}

func (n Branch) MustMany(name string) []Node {
	if c, has := n[name]; has {
		if many, ok := c.(Many); ok {
			return many
		}
		panic(fmt.Errorf("not a Many"))
	}
	panic(fmt.Errorf("name %q not found", name))
}

type Extra struct {
	Data interface{}
}

func (Extra) One(_ string) Node {
	return nil
}

func (Extra) MustOne(_ string) Node {
	panic(errors.Inconceivable)
}

func (Extra) Many(_ string) []Node {
	return nil
}

func (Extra) MustMany(_ string) []Node {
	panic(errors.Inconceivable)
}

func (c Extra) collapse(level int) Node {
	return c
}

var stackLevelRE = regexp.MustCompile(`^(\w+)@(\d+)$`)

func unlevel(name string, g wbnf.Grammar) (string, int) {
	if m := stackLevelRE.FindStringSubmatch(name); m != nil {
		i, err := strconv.Atoi(m[2])
		if err != nil {
			panic(errors.Inconceivable)
		}
		return m[1], i
	}
	if !strings.Contains(name, wbnf.StackDelim) {
		if _, has := g[wbnf.Rule(name+"@1")]; has {
			return name, 0
		}
	}
	return name, -1
}

func (n Branch) collapse(level int) Node {
	if level >= 0 {
		if oneBranch := n.oneBranch(); oneBranch != nil {
			oneBranch.inc(SkipTag, 1)
			if choice, has := n[ChoiceTag]; has {
				if oChoice, has := oneBranch[ChoiceTag]; has {
					oneBranch[ChoiceTag] = append(choice.(Many), oChoice.(Many)...)
				} else {
					oneBranch[ChoiceTag] = choice
				}
			}
			if rule, has := n[RuleTag]; has {
				oneBranch[RuleTag] = rule
			}
			return oneBranch
		}
	}
	return n
}

func (n Branch) add(name string, node Node, ctr counter) {
	switch ctr {
	case counter{}:
		panic(errors.Inconceivable)
	case zeroOrOne, oneOne:
		n.one(name, node)
	default:
		n.many(name, node)
	}
}

func (n Branch) one(name string, node Node) {
	if _, has := n[name]; has {
		panic(errors.Inconceivable)
	}
	n[name] = One{Node: node}
}

func (n Branch) many(name string, node Node) {
	if many, has := n[name]; has {
		n[name] = append(many.(Many), node)
	} else {
		n[name] = Many([]Node{node})
	}
}

func (n Branch) oneBranch() Branch {
	var oneChildren Children
	for childrenName, children := range n {
		if !strings.HasPrefix(childrenName, "@") {
			if oneChildren != nil {
				return nil
			}
			oneChildren = children
		}
	}
	var b Branch
	if oneChildren != nil {
		switch c := oneChildren.(type) {
		case One:
			b, _ = c.Node.(Branch)
		case Many:
			if len(c) == 1 {
				b, _ = c[0].(Branch)
			}
		}
	}
	return b
}

func (n Branch) fromTerm(g wbnf.Grammar, term wbnf.Term, ctrs counters, v interface{}) {
	var tag string
	defer enterf("term=%v, v=%v", term, v).exitf("tag=%q, n=%v", &tag, &n)
	switch t := term.(type) {
	case wbnf.S, wbnf.RE:
		n.add("", Leaf(v.(parser.Scanner)), ctrs[""])
	case wbnf.Rule:
		term := g[t]
		childCtrs := newCounters(term)
		b := Branch{}
		unleveled, level := unlevel(string(t), g)
		b.fromTerm(g, term, childCtrs, v)
		var node Node = b
		if name := childCtrs.singular(); name != nil {
			node = b[*name].(One).Node
			// TODO: zeroOrOne
		}
		node = node.collapse(level)
		n.add(unleveled, node, ctrs[string(t)])
	case wbnf.Seq:
		node := v.(parser.Node)
		tag = node.Tag
		for i, child := range node.Children {
			n.fromTerm(g, t[i], ctrs, child)
		}
	case wbnf.Oneof:
		node := v.(parser.Node)
		tag = node.Tag
		n.many(ChoiceTag, Extra{Data: node.Extra.(int)})
		n.fromTerm(g, t[node.Extra.(int)], ctrs, node.Children[0])
	case wbnf.Delim:
		node := v.(parser.Node)
		tag = node.Tag
		if node.Extra.(wbnf.Associativity) != wbnf.NonAssociative {
			panic(errors.Unfinished)
		}
		L, R := t.LRTerms(node)
		terms := [2]wbnf.Term{L, t.Sep}
		for i, child := range node.Children {
			n.fromTerm(g, terms[i%2], ctrs, child)
			terms[0] = R
		}
	case wbnf.Quant:
		node := v.(parser.Node)
		tag = node.Tag
		for _, child := range node.Children {
			n.fromTerm(g, t.Term, ctrs, child)
		}
	case wbnf.Named:
		b := Branch{}
		childCtrs := newCounters(t.Term)
		b.fromTerm(g, t.Term, childCtrs, v)
		var node Node = b
		if name := childCtrs.singular(); name != nil {
			for _, child := range b {
				node = child.(One).Node
				break
			}
			// TODO: zeroOrOne
		}
		n.add(t.Name, node, ctrs[t.Name])
	case wbnf.REF:
		n.add(t.Ident, n.fromRefNode(v), ctrs[t.Ident])
	default:
		panic(fmt.Errorf("unexpected term type: %v %[1]T", t))
	}
}

func (n Branch) fromRefNode(v interface{}) Node {
	if s, ok := v.(parser.Scanner); ok {
		return Leaf(s)
	}
	b := Branch{}
	for _, child := range v.(parser.Node).Children {
		b.many("", n.fromRefNode(child))
	}
	return b
}

func (n Branch) pull(name string, level int, ctr counter, childCtrs counters) Node {
	var node Node
	switch ctr {
	case counter{}:
		panic(errors.Inconceivable)
	case zeroOrOne, oneOne:
		node = n.pullOne(name)
	default:
		node = n.pullMany(name)
	}

	if level >= 0 {
		if b, ok := node.(Branch); ok {
			if b.inc(SkipTag, -1) > 0 {
				node = Branch{name: Many{b}}
			}
		}
	}

	if name := childCtrs.singular(); name != nil {
		return Branch{*name: One{Node: node}}
	}
	return node
}

func (n Branch) pullOne(name string) Node {
	if child, has := n[name]; has {
		delete(n, name)
		return child.(One).Node
	}
	return nil
}

func (n Branch) inc(name string, delta int) int {
	i := 0
	if child, has := n[name]; has {
		i = child.(One).Node.(Extra).Data.(int)
	}
	j := i + delta
	if j > 0 {
		n[name] = One{Node: Extra{Data: j}}
	} else {
		delete(n, name)
	}
	return i
}

func (n Branch) pullMany(name string) Node {
	if node, has := n[name]; has {
		many := node.(Many)
		if len(many) > 0 {
			result := many[0]
			if len(many) > 1 {
				n[name] = many[1:]
			} else {
				delete(n, name)
			}
			return result
		}
	}
	return nil
}

func (n Branch) toTerm(g wbnf.Grammar, term wbnf.Term, ctrs counters) (out interface{}) {
	// defer enterf("%T %[1]v %v", term, ctrs).exitf("%v", &out)
	switch t := term.(type) {
	case wbnf.S, wbnf.RE:
		if node := n.pull("", 0, ctrs[""], nil); node != nil {
			return parser.Scanner(node.(Leaf))
		}
		return nil
	case wbnf.Rule:
		term := g[t]
		ctrs2 := newCounters(term)
		unleveled, level := unlevel(string(t), g)
		if b := n.pull(unleveled, level, ctrs[string(t)], ctrs2); b != nil {
			return relabelNode(string(t), b.(Branch).toTerm(g, term, ctrs2))
		}
		return nil
	case wbnf.Seq:
		result := parser.Node{Tag: seqTag}
		for _, child := range t {
			if node := n.toTerm(g, child, ctrs); node != nil {
				result.Children = append(result.Children, node)
			} else {
				return nil
			}
		}
		return result
	case wbnf.Oneof:
		extra := n.pullMany(ChoiceTag).(Extra).Data.(int)
		return parser.Node{
			Tag:      oneofTag,
			Extra:    extra,
			Children: []interface{}{n.toTerm(g, t[extra], ctrs)},
		}
	case wbnf.Delim:
		v := parser.Node{
			Tag:   delimTag,
			Extra: wbnf.NonAssociative,
		}
		terms := [2]wbnf.Term{t.Term, t.Sep}
		i := 0
		for ; ; i++ {
			if child := n.toTerm(g, terms[i%2], ctrs); child != nil {
				v.Children = append(v.Children, child)
			} else {
				break
			}
		}
		if i%2 == 0 {
			panic(errors.Inconceivable)
		}
		return v
	case wbnf.Quant:
		result := parser.Node{Tag: quantTag}
		for {
			if v := n.toTerm(g, t.Term, ctrs); v != nil {
				result.Children = append(result.Children, v)
			} else {
				break
			}
		}
		if !t.Contains(len(result.Children)) {
			panic(errors.Inconceivable)
		}
		return result
	case wbnf.Named:
		ctrs2 := newCounters(t.Term)
		if b := n.pull(t.Name, 0, ctrs[t.Name], ctrs2); b != nil {
			return relabelNode(t.Name, b.(Branch).toTerm(g, t.Term, ctrs2))
		}
		return nil
	default:
		panic(fmt.Errorf("unexpected term type: %v %[1]T", t))
	}
}

func (c One) clone() Children {
	return One{Node: c.Node.clone()}
}

func (c Many) clone() Children {
	result := make(Many, 0, len(c))
	for _, child := range c {
		result = append(result, child.clone())
	}
	return result
}

func (l Leaf) clone() Node {
	return l
}

func (n Branch) clone() Node {
	result := Branch{}
	for name, node := range n {
		result[name] = node.clone()
	}
	return result
}

func (c Extra) clone() Node {
	return c
}

func (c One) narrow() bool {
	return c.Node.narrow()
}

func (c Many) narrow() bool {
	return len(c) == 0 || len(c) == 1 && c[0].narrow()
}

func (l Leaf) narrow() bool {
	return true
}

func (n Branch) narrow() bool {
	switch len(n) {
	case 0:
		return true
	case 1:
		for _, group := range n {
			return group.narrow()
		}
	}
	return false
}

func (c Extra) narrow() bool {
	return true
}

func (c One) String() string {
	if c.Node == nil {
		panic(errors.Inconceivable)
	}
	return c.Node.String()
}

func (c Many) String() string {
	var sb strings.Builder
	sb.WriteString("[")
	pre := ""
	complex := len(c) > 1
	if complex {
		wide := false
		for _, child := range c {
			if !child.narrow() {
				wide = true
				break
			}
		}
		if !wide {
			complex = false
		}
	}
	if complex {
		pre = "  "
		sb.WriteString("\n" + pre)
	}
	for i, child := range c {
		if i > 0 {
			if complex {
				sb.WriteString(",\n" + pre)
			} else {
				sb.WriteString(", ")
			}
		}
		fmt.Fprintf(&sb, "%s", strings.ReplaceAll(child.String(), "\n", "\n"+pre))
	}
	if complex {
		sb.WriteString(",\n")
	}
	sb.WriteString("]")
	return sb.String()
}

var specialCharRE = regexp.MustCompile("[[:cntrl:]:,'`(){}[\\]]")

func (l Leaf) String() string {
	var sb strings.Builder
	scanner := parser.Scanner(l)
	s := scanner.String()
	fmt.Fprintf(&sb, "%d‣", scanner.Offset())
	switch {
	case specialCharRE.MatchString(s):
		fmt.Fprintf(&sb, "%q", s)
	case strings.Contains(s, `"`):
		fmt.Fprintf(&sb, "`%s`", strings.ReplaceAll(s, "`", "``"))
	default:
		fmt.Fprintf(&sb, "%s", s)
	}
	return sb.String()
}

func (n Branch) String() string {
	var sb strings.Builder
	sb.WriteString("(")
	pre := ""
	if len(n) > 1 {
		sb.WriteString("\n  ")
		pre = "  "
	}
	i := 0
	names := make([]string, 0, len(n))
	for name := range n {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		group := n[name]
		if i > 0 {
			sb.WriteString(",\n  ")
		}
		i++
		child := strings.ReplaceAll(group.String(), "\n", "\n"+pre)
		if name == "" {
			name = "''"
		}
		fmt.Fprintf(&sb, "%s: %s", name, child)
	}
	if len(n) > 1 {
		sb.WriteString(",\n")
	}
	sb.WriteString(")")
	return sb.String()
}

func (c Extra) String() string {
	return fmt.Sprintf("%v", c.Data)
}

func (c One) Scanner() parser.Scanner {
	return c.Node.Scanner()
}

func (c Many) Scanner() parser.Scanner {
	panic("Scanner() not valid for Many")
}

func (c Extra) Scanner() parser.Scanner {
	panic("Scanner() not valid for Extra")
}

func (l Leaf) Scanner() parser.Scanner {
	return parser.Scanner(l)
}

func (n Branch) Scanner() parser.Scanner {
	panic("Scanner() not valid for Branch")
}
