package parser

import "reflect"

type Tag string

const NoTag = Tag("")

type BaseNode interface {
	Tag() Tag
	isBaseNode()
}

type Terminal struct {
	tag Tag
	val string
}

func (t *Terminal) String() string { return t.val }
func (t *Terminal) Tag() Tag       { return t.tag }
func (t *Terminal) isBaseNode()    {}
func (t *Terminal) NewFromPtr(value string, tag Tag) BaseNode {
	t.val = value
	t.tag = tag
	return t
}
func (t Terminal) New(value string, tag Tag) BaseNode {
	return (&t).NewFromPtr(value, tag)
}

type NonTerminal struct {
	nodes []BaseNode
	tag   Tag
}

func (t *NonTerminal) AllChildren() []BaseNode { return t.nodes }
func (t *NonTerminal) Count() int              { return len(t.nodes) }
func (t *NonTerminal) Tag() Tag                { return t.tag }
func (t *NonTerminal) isBaseNode()             {}
func (t *NonTerminal) Add(child ...BaseNode) {
	t.nodes = append(t.nodes, child...)
}
func (t *NonTerminal) AddAndSet(child BaseNode, dest *BaseNode) {
	t.nodes = append(t.nodes, child)
	*dest = child
}
func (t *NonTerminal) AddAndCount(child BaseNode, dest *int) {
	t.nodes = append(t.nodes, child)
	*dest++
}
func (t *NonTerminal) AtIndex(ofType reflect.Type, tag Tag, index int) BaseNode {
	return AtIndex(t.AllChildren(), reflect.PtrTo(ofType), tag, index)
}
func (t *NonTerminal) Iter(ofType reflect.Type, tag Tag) Iter {
	return NewIter(t.AllChildren(), reflect.PtrTo(ofType), tag)
}

var _ BaseNode = &Terminal{}
var _ BaseNode = &NonTerminal{}

type Foo struct{ Terminal }

var _ BaseNode = &Foo{}
