package diff

import (
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/arr-ai/wbnf/parser"
)

type NodeDiff struct {
	A, B     *parser.Node
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

func NewNodeDiff(a, b *parser.Node) NodeDiff {
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
		} else if childNodeA, ok := x.(parser.Node); ok {
			childNodeB := b.Children[i].(parser.Node)
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
