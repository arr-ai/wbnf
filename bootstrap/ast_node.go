package bootstrap

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/arr-ai/wbnf/parser"
)

func ParserNodeToASTNode(g Grammar, v interface{}) ASTBranch {
	rule := NodeRule(v)
	term := g[rule]
	result := ASTBranch{}
	result.one("@rule", ASTExtra{extra: rule})
	result.fromTerm(g, term, newCounters(term), v)
	return result
}

func ASTNodeToParserNode(g Grammar, branch ASTBranch) interface{} {
	branch = branch.clone().(ASTBranch)
	rule := branch.pullOne("@rule").(ASTExtra).extra.(Rule)
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

type ASTChildren interface {
	fmt.Stringer
	isASTChildren()
	clone() ASTChildren
	narrow() bool
}

func (ASTOne) isASTChildren()  {}
func (ASTMany) isASTChildren() {}

type ASTOne struct {
	One ASTNode
}

type ASTMany []ASTNode

type ASTNode interface {
	fmt.Stringer
	isASTNode()
	clone() ASTNode
	narrow() bool
}

func (ASTBranch) isASTNode() {}
func (ASTLeaf) isASTNode()   {}
func (ASTExtra) isASTNode()  {}

type ASTLeaf parser.Scanner

type ASTBranch map[string]ASTChildren

type ASTExtra struct {
	extra interface{}
}

var stackLevelRE = regexp.MustCompile(`^(\w+)@(\d+)$`)

func unlevel(name string) (string, int) {
	if m := stackLevelRE.FindStringSubmatch(name); m != nil {
		i, err := strconv.Atoi(m[2])
		if err != nil {
			panic(Inconceivable)
		}
		return m[1], i
	}
	return name, 0
}

const levelTag = "@level"

func (n ASTBranch) add(name string, level int, node ASTNode, ctr counter, childCtrs counters) {
	if name := childCtrs.singular(); name != nil {
		node = node.(ASTBranch)[*name].(ASTOne).One
		// TODO: zeroOrOne
	}

	if level > 0 {
		if b, ok := node.(ASTBranch); ok {
			if children := b.singular(); children != nil {
				if many, ok := children.(ASTMany); ok {
					if len(many) == 1 {
						if b, ok := many[0].(ASTBranch); ok {
							b.inc(levelTag, 1)
							node = b
						}
					}
				}
			}
		}
	}

	switch ctr {
	case counter{}:
		panic(Inconceivable)
	case zeroOrOne, oneOne:
		n.one(name, node)
	default:
		n.many(name, node)
	}
}

func (n ASTBranch) one(name string, node ASTNode) {
	if _, has := n[name]; has {
		panic(Inconceivable)
	}
	n[name] = ASTOne{One: node}
}

func (n ASTBranch) put(name string, v interface{}) {
	n[name] = ASTOne{One: ASTExtra{extra: v}}
}

func (n ASTBranch) many(name string, node ASTNode) {
	if many, has := n[name]; has {
		n[name] = append(many.(ASTMany), node)
	} else {
		n[name] = ASTMany([]ASTNode{node})
	}
}

func (n ASTBranch) singular() ASTChildren {
	switch len(n) {
	case 1:
		for _, children := range n {
			return children
		}
	case 2:
		if _, has := n[levelTag]; has {
			for name, children := range n {
				if name != levelTag {
					return children
				}
			}
		}
	}
	return nil
}

func (n ASTBranch) fromTerm(g Grammar, term Term, ctrs counters, v interface{}) {
	switch t := term.(type) {
	case S, RE:
		n.add("", 0, ASTLeaf(v.(parser.Scanner)), ctrs[""], nil)
	case Rule:
		term := g[t]
		ctrs2 := newCounters(term)
		b := ASTBranch{}
		b.fromTerm(g, term, ctrs2, v)
		unleveled, level := unlevel(string(t))
		n.add(unleveled, level, b, ctrs[string(t)], ctrs2)
	case Seq:
		node := v.(parser.Node)
		for i, child := range node.Children {
			n.fromTerm(g, t[i], ctrs, child)
		}
	case Oneof:
		node := v.(parser.Node)
		n.many("@choice", ASTExtra{extra: node.Extra.(int)})
		n.fromTerm(g, t[node.Extra.(int)], ctrs, node.Children[0])
	case Delim:
		node := v.(parser.Node)
		if node.Extra.(Associativity) != NonAssociative {
			panic(Unfinished)
		}
		L, R := t.LRTerms(node)
		terms := [2]Term{L, t.Sep}
		for i, child := range node.Children {
			n.fromTerm(g, terms[i%2], ctrs, child)
			terms[0] = R
		}
	case Quant:
		node := v.(parser.Node)
		for _, child := range node.Children {
			n.fromTerm(g, t.Term, ctrs, child)
		}
	case Named:
		b := ASTBranch{}
		ctrs2 := newCounters(t.Term)
		b.fromTerm(g, t.Term, ctrs2, v)
		n.add(t.Name, 0, b, ctrs[t.Name], ctrs2)
	default:
		panic(fmt.Errorf("unexpected term type: %v %[1]T", t))
	}
}

func (n ASTBranch) pull(name string, level int, ctr counter, childCtrs counters) ASTNode {
	var node ASTNode
	switch ctr {
	case counter{}:
		panic(Inconceivable)
	case zeroOrOne, oneOne:
		node = n.pullOne(name)
	default:
		node = n.pullMany(name)
	}

	if level > 0 {
		if b, ok := node.(ASTBranch); ok {
			if b.inc(levelTag, -1) > 0 {
				node = ASTBranch{name: ASTMany{b}}
			}
		}
	}

	if name := childCtrs.singular(); name != nil {
		return ASTBranch{*name: ASTOne{One: node}}
	}
	return node
}

func (n ASTBranch) pullOne(name string) ASTNode {
	if child, has := n[name]; has {
		delete(n, name)
		return child.(ASTOne).One
	}
	return nil
}

func (n ASTBranch) inc(name string, delta int) int {
	i := 0
	if child, has := n[name]; has {
		i = child.(ASTOne).One.(ASTExtra).extra.(int)
	}
	j := i + delta
	if j > 0 {
		n.put(name, j)
	} else {
		delete(n, name)
	}
	return i
}

func (n ASTBranch) pullMany(name string) ASTNode {
	if node, has := n[name]; has {
		many := node.(ASTMany)
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

func (n ASTBranch) toTerm(g Grammar, term Term, ctrs counters) (out interface{}) {
	// defer enterf("%T %[1]v %v", term, ctrs).exitf("%v", &out)
	switch t := term.(type) {
	case S, RE:
		if node := n.pull("", 0, ctrs[""], nil); node != nil {
			return parser.Scanner(node.(ASTLeaf))
		}
		return nil
	case Rule:
		term := g[t]
		ctrs2 := newCounters(term)
		unleveled, level := unlevel(string(t))
		if b := n.pull(unleveled, level, ctrs[string(t)], ctrs2); b != nil {
			return relabelNode(string(t), b.(ASTBranch).toTerm(g, term, ctrs2))
		}
		return nil
	case Seq:
		result := parser.Node{Tag: seqTag}
		for _, child := range t {
			if node := n.toTerm(g, child, ctrs); node != nil {
				result.Children = append(result.Children, node)
			} else {
				return nil
			}
		}
		return result
	case Oneof:
		extra := n.pullMany("@choice").(ASTExtra).extra.(int)
		return parser.Node{
			Tag:      oneofTag,
			Extra:    extra,
			Children: []interface{}{n.toTerm(g, t[extra], ctrs)},
		}
	case Delim:
		v := parser.Node{
			Tag:   delimTag,
			Extra: NonAssociative,
		}
		terms := [2]Term{t.Term, t.Sep}
		i := 0
		for ; ; i++ {
			if child := n.toTerm(g, terms[i%2], ctrs); child != nil {
				v.Children = append(v.Children, child)
			} else {
				break
			}
		}
		if i%2 == 0 {
			panic(Inconceivable)
		}
		return v
	case Quant:
		result := parser.Node{Tag: quantTag}
		for {
			if v := n.toTerm(g, t.Term, ctrs); v != nil {
				result.Children = append(result.Children, v)
			} else {
				break
			}
		}
		if !t.Contains(len(result.Children)) {
			panic(Inconceivable)
		}
		return result
	case Named:
		ctrs2 := newCounters(t.Term)
		if b := n.pull(t.Name, 0, ctrs[t.Name], ctrs2); b != nil {
			return relabelNode(t.Name, b.(ASTBranch).toTerm(g, t.Term, ctrs2))
		}
		return nil
	default:
		panic(fmt.Errorf("unexpected term type: %v %[1]T", t))
	}
}

func (c ASTOne) clone() ASTChildren {
	return ASTOne{One: c.One.clone()}
}

func (c ASTMany) clone() ASTChildren {
	result := make(ASTMany, 0, len(c))
	for _, child := range c {
		result = append(result, child.clone())
	}
	return result
}

func (l ASTLeaf) clone() ASTNode {
	return l
}

func (n ASTBranch) clone() ASTNode {
	result := ASTBranch{}
	for name, node := range n {
		result[name] = node.clone()
	}
	return result
}

func (c ASTExtra) clone() ASTNode {
	return c
}

func (c ASTOne) narrow() bool {
	return c.One.narrow()
}

func (c ASTMany) narrow() bool {
	return len(c) == 0 || len(c) == 1 && c[0].narrow()
}

func (l ASTLeaf) narrow() bool {
	return true
}

func (n ASTBranch) narrow() bool {
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

func (c ASTExtra) narrow() bool {
	return true
}

func (c ASTOne) String() string {
	if c.One == nil {
		panic(Inconceivable)
	}
	return c.One.String()
}

func (c ASTMany) String() string {
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

func (l ASTLeaf) String() string {
	s := parser.Scanner(l).String()
	if strings.Contains(s, "`") && !strings.Contains(s, `"`) {
		return fmt.Sprintf("%q", s)
	}
	return fmt.Sprintf("`%s`", strings.ReplaceAll(s, "`", "``"))
}

func (n ASTBranch) String() string {
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

func (c ASTExtra) String() string {
	return fmt.Sprintf("%v", c.extra)
}
