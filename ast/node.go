package ast

import (
	"fmt"
	"regexp"
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
	Many(name string) []Node
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

func (Leaf) Many(_ string) []Node {
	return nil
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

func (n Branch) Many(name string) []Node {
	if c, has := n[name]; has {
		if many, ok := c.(Many); ok {
			return many
		}
	}
	return nil
}

type Extra struct {
	Data interface{}
}

func (Extra) One(_ string) Node {
	return nil
}

func (Extra) Many(_ string) []Node {
	return nil
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
