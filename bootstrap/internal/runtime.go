package internal

import "reflect"

type isGenNode interface {
	isGenNode()
}

type isParentNode interface {
	AllChildren() []isGenNode
	CountChildren() int
}

type Iter interface {
	Next() isGenNode
}

type iter struct {
	nodes   []isGenNode
	current int
	of      reflect.Type
}

func (i *iter) Next() isGenNode {
	for i.current < len(i.nodes) {
		out := i.nodes[i.current]
		i.current++
		if reflect.TypeOf(out) == i.of {
			return out
		}
	}
	return nil
}

func NewIter(nodes []isGenNode, ofType reflect.Type) Iter {
	return &iter{
		nodes:   nodes,
		current: 0,
		of:      ofType,
	}
}

func AtIndex(nodes []isGenNode, ofType reflect.Type, index int) isGenNode {
	iter := NewIter(nodes, ofType)
	var out isGenNode
	for i := 0; i <= index; i++ {
		out = iter.Next()
	}
	return out
}

func ForEach(iter Iter, fn func(node isGenNode)) {
	for {
		val := iter.Next()
		if val == nil {
			return
		}
		fn(val)
	}
}

type Token struct{ v string }

func (t *Token) isGenNode()     {}
func (t *Token) String() string { return t.v }
