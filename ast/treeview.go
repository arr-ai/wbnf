package ast

import (
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
		if len(tree.Items()) == 1 && len(tree.Items()[0].Items()) == 1 {
			item := tree.Items()[0]
			tree = gotree.New(name + " > " + item.Text())
			tree.AddTree(item.Items()[0])
			return tree
		}
	case Extra:
		tree.Add(n.String())
		return tree
	case Leaf:
		return gotree.New(n.String())
	}

	tree.SortItems()
	return tree
}
