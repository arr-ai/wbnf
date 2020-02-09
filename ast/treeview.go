package ast

import (
	"sort"
	"strings"

	"github.com/arr-ai/wbnf/gotree"
)

func BuildTreeView(rootname string, root Node, skipAtNodes bool) string {
	return fromAst(rootname, root, skipAtNodes).Print()
}

func fromAst(name string, node Node, skipAtNodes bool) gotree.Tree {
	tree := gotree.New(name)

	switch n := node.(type) {
	case Branch:
		for name, val := range n {
			if skipAtNodes && strings.HasPrefix(name, "@") {
				continue
			}
			switch c := val.(type) {
			case One:
				tree.AddTree(fromAst(name, c.Node, skipAtNodes))
			case Many:
				for _, c := range c {
					tree.AddTree(fromAst(name, c, skipAtNodes))
				}
			}
		}
	case Extra:
		t := gotree.New(name)
		t.Add(n.String())
		return t
	case Leaf:
		return gotree.New(n.Scanner().String())
	}

	// Remove redundant stack levels
	if len(tree.Items()) == 1 && tree.Text() == tree.Items()[0].Text() {
		return tree.Items()[0]
	}

	sort.Slice(tree.Items(), func(i, j int) bool {
		return strings.Compare(tree.Items()[i].Text(), tree.Items()[j].Text()) < 0
	})

	return tree
}
