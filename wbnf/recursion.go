package wbnf

import (
	"fmt"
	"sort"
	"strings"

	"github.com/arr-ai/frozen"
)

/*
Need to test for rule chains which could recurse infinitely.

Basic definition: Any Seq{} which could make a graph from itself back to itself without first passing through
				  some other non-optional term is recursive.
Obvious ones:
	a -> a;
	a -> "("? a;
*/
func checkForRecursion(tree GrammarNode) error {
	dangers := map[string]frozen.Set{}

	// First get a map with every rules directly connected rules
	WalkerOps{EnterProdNode: func(node ProdNode) Stopper {
		td := frozen.NewSet()
		for _, t := range node.AllTerm() {
			td = td.Union(getSequenceDangerTerms(t))
		}
		dangers[node.OneIdent().String()] = td
		return NodeExiter
	}}.Walk(tree)

	// walk every path
	gn := &gnode{map[string]*gnode{}}
	for k := range dangers {
		gn.walkRule(k, &dangers)
	}

	// now determine the cycles

	var badRoutes []string
	paths := findPaths("", gn, frozen.NewSet(), nil)
	for _, p := range paths {
		if len(p) != frozen.NewSetFromStrings(p...).Count() {
			badRoutes = append(badRoutes, strings.Join(p, " > "))
		}
	}

	if len(badRoutes) > 0 {
		return validationError{
			msg:  fmt.Sprintf("Possible cycle(s) detected: \n\t%s", strings.Join(badRoutes, "\n\t")),
			kind: PossibleCycleDetected,
		}
	}
	return nil
}

func findPaths(name string, node *gnode, seen frozen.Set, current []string) [][]string {
	if name != "" {
		current = append(current, name)
	}
	var possibles [][]string
	for k, next := range node.next {
		if seen.Has(k) {
			possibles = append(possibles, append(current[:], k))
		} else {
			possibles = append(possibles, findPaths(k, next, seen.With(k), current)...)
		}
	}

	return possibles
}

func sortedSet(s frozen.Set) []string {
	out := make([]string, 0, s.Count())
	for _, x := range s.Elements() {
		out = append(out, x.(string))
	}
	sort.Strings(out)
	return out
}

type gnode struct {
	next map[string]*gnode
}

func (g *gnode) walkRule(start string, dangers *map[string]frozen.Set) *gnode {
	if f, has := g.next[start]; has {
		return f
	}
	node := &gnode{next: map[string]*gnode{}}
	g.next[start] = node
	for _, target := range sortedSet((*dangers)[start]) {
		if f, has := g.next[target]; has {
			node.next[target] = f
		} else {
			node.next[target] = g.walkRule(target, dangers)
		}
	}
	return node
}

func getSequenceDangerTerms(tree TermNode) frozen.Set {
	result := frozen.NewSet()

	if tree.OneOp() == "|" {
		// Any of the options could be dangerous, need to | them all
		for _, term := range tree.AllTerm() {
			result = result.Union(getSequenceDangerTerms(term))
		}
	} else if childTerms := tree.AllTerm(); len(childTerms) > 0 {
		// As soon as any term is required we can consider the whole tree safe
		for _, term := range childTerms {
			dangers := getSequenceDangerTerms(term)
			if dangers.IsEmpty() {
				return result
			}
			result = result.Union(dangers.Without(""))
		}
	} else {
		for _, q := range tree.AllQuant() {
			if q.OneOp() == "+" || (q.OneMin() != nil && q.OneMin().String() != "0") || q.OneNamed() != nil {
				return result // Any of these mean at least one of this term is required
			}
		}
		atom := tree.OneNamed().OneAtom()
		if term := atom.OneTerm(); term != nil {
			result = result.Union(getSequenceDangerTerms(*term))
		} else if ident := atom.OneIdent(); ident != nil {
			result = result.With(ident.String())
		} else if len(tree.AllQuant()) == 0 { // no quant means its some form of required string
			return result
		} else {
			result = result.With("")
		}
	}
	return result
}
