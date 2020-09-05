package parser

import (
	"fmt"
)

type TreeElement interface {
	IsTreeElement()
}

func (Node) IsTreeElement()    {}
func (Scanner) IsTreeElement() {}

type Extra interface {
	IsExtra()
}

type Node struct {
	Tag      string        `json:"tag"`
	Extra    Extra         `json:"extra"`
	Children []TreeElement `json:"nodes"`
}

func NewNode(tag string, extra Extra, children ...TreeElement) *Node {
	return &Node{Tag: tag, Extra: extra, Children: children}
}

func (n Node) Count() int {
	return len(n.Children)
}

func (n Node) Normalize() Node {
	children := make([]TreeElement, 0, len(n.Children))
	for _, v := range n.Children {
		if node, ok := v.(Node); ok {
			v = node.Normalize()
		}
		children = append(children, v)
	}
	return Node{
		Tag:      n.Tag,
		Extra:    n.Extra,
		Children: children,
	}
}

func (n Node) Get(path ...int) TreeElement {
	var v TreeElement = n
	for _, i := range path {
		v = v.(Node).Children[i]
	}
	return v
}

func (n Node) GetNode(path ...int) Node {
	return n.Get(path...).(Node)
}

func (n Node) GetScanner(path ...int) Scanner {
	return n.Get(path...).(Scanner)
}

func (n Node) GetString(path ...int) string {
	return n.GetScanner(path...).String()
}

func (n Node) String() string {
	return fmt.Sprintf("%s", n) //nolint:gosimple
}

func (n Node) Format(state fmt.State, c rune) {
	fmt.Fprintf(state, "%s", n.Tag)
	format := "%" + string(c)
	if n.Extra != nil {
		fmt.Fprintf(state, "║"+format, n.Extra)
	}
	fmt.Fprint(state, "[")
	for i, child := range n.Children {
		if i > 0 {
			fmt.Fprint(state, ", ")
		}
		fmt.Fprintf(state, format, child)
	}
	fmt.Fprint(state, "]")
}

type Parser interface {
	Parse(scope Scope, input *Scanner, output *TreeElement, stk *call) error
	AsTerm() Term
}
