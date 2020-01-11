package parser

import "reflect"

type Iter interface {
	Next() BaseNode
}

type iter struct {
	nodes   []BaseNode
	current int
	of      reflect.Type
	tag     Tag
}

func (i *iter) Next() BaseNode {
	for i.current < len(i.nodes) {
		out := i.nodes[i.current]
		i.current++
		if reflect.TypeOf(out) == i.of && out.Tag() == i.tag {
			return out
		}
	}
	return nil
}

func NewIter(nodes []BaseNode, ofType reflect.Type, tag Tag) Iter {
	return &iter{
		nodes:   nodes,
		current: 0,
		of:      ofType,
		tag:     tag,
	}
}

func AtIndex(nodes []BaseNode, ofType reflect.Type, tag Tag, index int) BaseNode {
	iter := NewIter(nodes, ofType, tag)
	var out BaseNode
	for i := 0; i <= index; i++ {
		out = iter.Next()
	}
	return out
}

func ForEach(iter Iter, fn func(node BaseNode)) {
	for {
		val := iter.Next()
		if val == nil {
			return
		}
		fn(val)
	}
}
