package ast

import (
	"sort"
	"strconv"
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
		tree.Add(n.String())
		return tree
	case Leaf:
		return gotree.New(n.String())
	}

	// Remove redundant stack levels
	if len(tree.Items()) == 1 && tree.Text() == tree.Items()[0].Text() {
		return tree.Items()[0]
	}

	sort.Slice(tree.Items(), func(i, j int) bool {
		toInt := func(s string) int {
			v, err := strconv.Atoi(s)
			if err != nil {
				return -1
			}
			return v
		}
		return toInt(tree.Items()[i].Text()) < toInt(tree.Items()[j].Text())
	})

	return tree
}
