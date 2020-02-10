// Package gotree create and print tree.
package gotree

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	newLine      = "\n"
	emptySpace   = "    "
	middleItem   = "├── "
	continueItem = "│   "
	lastItem     = "└── "
)

type (
	tree struct {
		text  string
		items []Tree
	}

	// Tree is tree interface
	Tree interface {
		Add(text string) Tree
		AddTree(tree Tree)
		Items() []Tree
		SortItems()
		Rank() int
		Text() string
		Print() string
	}

	printer struct {
	}

	// Printer is printer interface
	Printer interface {
		Print(Tree) string
	}
)

//New returns a new GoTree.Tree
func New(text string) Tree {
	return &tree{
		text:  text,
		items: []Tree{},
	}
}

//Add adds a node to the tree
func (t *tree) Add(text string) Tree {
	n := New(text)
	t.items = append(t.items, n)
	return n
}

//AddTree adds a tree as an item
func (t *tree) AddTree(tree Tree) {
	t.items = append(t.items, tree)
}

//Text returns the node's value
func (t *tree) Text() string {
	return t.text
}

//Items returns all items in the tree
func (t *tree) Items() []Tree {
	return t.items
}

//Print returns an visual representation of the tree
func (t *tree) Print() string {
	return newPrinter().Print(t)
}

// SortItems sorts the items of tree by the determined rank
func (t *tree) SortItems() {
	sort.Slice(t.items, func(i, j int) bool {
		return t.items[i].Rank() < t.items[j].Rank()
	})
}

// Rank returns the rank of a tree
func (t *tree) Rank() int {
	num := regexp.MustCompile("^[1-9][0-9]*").Find([]byte(t.Text()))
	if num != nil {
		r, err := strconv.Atoi(string(num))
		if err != nil {
			panic(err)
		}
		return r
	}
	if len(t.Items()) == 0 {
		return -1
	}
	// max int
	r := int(^uint(0) >> 1)
	for _, i := range t.Items() {
		iRank := i.Rank()
		if r > iRank {
			r = iRank
		}
	}
	return r
}

func newPrinter() Printer {
	return &printer{}
}

//Print prints a tree to a string
func (p *printer) Print(t Tree) string {
	return t.Text() + newLine + p.printItems(t.Items(), []bool{})
}

func (p *printer) printText(text string, spaces []bool, last bool) string {
	var result string
	for _, space := range spaces {
		if space {
			result += emptySpace
		} else {
			result += continueItem
		}
	}

	indicator := middleItem
	if last {
		indicator = lastItem
	}

	var out string
	lines := strings.Split(text, "\n")
	for i := range lines {
		text := lines[i]
		if i == 0 {
			out += result + indicator + text + newLine
			continue
		}
		if last {
			indicator = emptySpace
		} else {
			indicator = continueItem
		}
		out += result + indicator + text + newLine
	}

	return out
}

func (p *printer) printItems(t []Tree, spaces []bool) string {
	var result string
	for i, f := range t {
		last := i == len(t)-1
		result += p.printText(f.Text(), spaces, last)
		if len(f.Items()) > 0 {
			spacesChild := append(spaces, last)
			result += p.printItems(f.Items(), spacesChild)
		}
	}
	return result
}
