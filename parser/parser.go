package parser

import (
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/arr-ai/frozen"
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
	Parse(scope frozen.Map, input *Scanner, output *TreeElement) error
}

type Func func(scope frozen.Map, input *Scanner, output *TreeElement) error

func (f Func) Parse(scope frozen.Map, input *Scanner, output *TreeElement) error {
	return f(scope, input, output)
}

func Transform(parser Parser, transform func(Node) Node) Parser {
	return Func(func(scope frozen.Map, input *Scanner, output *TreeElement) error {
		var v TreeElement
		if err := parser.Parse(scope, input, &v); err != nil {
			return err
		}
		*output = transform(v.(Node))
		return nil
	})
}

type NodeDiff struct {
	A, B     *Node
	Children map[int]NodeDiff
	Types    map[int][2]reflect.Type
}

func (d NodeDiff) String() string {
	var sb strings.Builder
	d.report(nil, &sb)
	return sb.String()
}

func (d NodeDiff) report(path []string, w io.Writer) {
	if d.Equal() {
		return
	}
	prefix := ""
	if len(path) > 0 {
		prefix = fmt.Sprintf("[%s] ", strings.Join(path, "."))
	}
	if d.A.Tag != d.B.Tag {
		fmt.Fprintf(w, "%sTag: %v != %v", prefix, d.A.Tag, d.B.Tag)
	}
	if d.A.Extra != d.B.Extra {
		fmt.Fprintf(w, "%sExtra: %v != %v", prefix, d.A.Extra, d.B.Extra)
	}
	if len(d.A.Children) != len(d.B.Children) {
		fmt.Fprintf(w, "%slen(Children): %v != %v", prefix, len(d.A.Children), len(d.B.Children))
	}
	for i, t := range d.Types {
		fmt.Fprintf(w, "%sChildren[%d].Type: %v != %v", prefix, i, t[0], t[1])
	}
	for i, d := range d.Children {
		d.report(append(append([]string{}, path...), fmt.Sprintf("%s[%d]", d.A.Tag, i)), w)
	}
}

func (d NodeDiff) Equal() bool {
	return len(d.A.Children) == len(d.B.Children) &&
		d.A.Tag == d.B.Tag &&
		d.A.Extra == d.B.Extra &&
		len(d.Children) == 0 &&
		len(d.Types) == 0
}

func NewNodeDiff(a, b *Node) NodeDiff {
	children := map[int]NodeDiff{}
	types := map[int][2]reflect.Type{}
	n := len(a.Children)
	if n > len(b.Children) {
		n = len(b.Children)
	}
	for i, x := range a.Children[:n] {
		aType := reflect.TypeOf(a)
		bType := reflect.TypeOf(a)
		if aType != bType {
			types[i] = [2]reflect.Type{aType, bType}
		} else if childNodeA, ok := x.(Node); ok {
			childNodeB := b.Children[i].(Node)
			if d := NewNodeDiff(&childNodeA, &childNodeB); !d.Equal() {
				children[i] = d
			}
		}
	}
	return NodeDiff{
		A:        a,
		B:        b,
		Children: children,
		Types:    types,
	}
}
