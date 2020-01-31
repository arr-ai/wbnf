package wbnf

import (
	"io"

	"github.com/arr-ai/wbnf/errors"
	"github.com/arr-ai/wbnf/wbnf/parser"
)

// The following methods assume a valid parse. Call (Term).ValidateParse first if
// unsure.

func (t S) Unparse(g Grammar, v interface{}, w io.Writer) (n int, err error) {
	return w.Write([]byte(v.(parser.Scanner).String()))
}

func (t RE) Unparse(g Grammar, v interface{}, w io.Writer) (n int, err error) {
	return w.Write([]byte(v.(parser.Scanner).String()))
}
func (t REF) Unparse(g Grammar, v interface{}, w io.Writer) (n int, err error) {
	return w.Write([]byte("\\" + v.(parser.Scanner).String()))
}

func unparse(g Grammar, term Term, v interface{}, w io.Writer, N *int) error {
	n, err := term.Unparse(g, v, w)
	if err == nil {
		*N += n
	}
	return err
}

func (t Seq) Unparse(g Grammar, v interface{}, w io.Writer) (n int, err error) {
	node := v.(parser.Node)
	for i, term := range t {
		if err = unparse(g, term, node.Children[i], w, &n); err != nil {
			return
		}
	}
	return n, nil
}

func (t Oneof) Unparse(g Grammar, v interface{}, w io.Writer) (n int, err error) {
	node := v.(parser.Node)
	return t[node.Extra.(int)].Unparse(g, node.Children[0], w)
}

func (t Delim) Unparse(g Grammar, v interface{}, w io.Writer) (n int, err error) {
	node := v.(parser.Node)
	left, right := t.LRTerms(node)

	if err = unparse(g, left, node.Children[0], w, &n); err != nil {
		return
	}
	for i := 1; i < node.Count(); i += 2 {
		if err = unparse(g, t.Sep, node.Children[i], w, &n); err != nil {
			return
		}
		if err = unparse(g, right, node.Children[i+1], w, &n); err != nil {
			return
		}
	}
	return
}

func (t Quant) Unparse(g Grammar, v interface{}, w io.Writer) (n int, err error) {
	for _, child := range v.(parser.Node).Children {
		if err = unparse(g, t.Term, child, w, &n); err != nil {
			return
		}
	}
	return
}

//-----------------------------------------------------------------------------

func (t Rule) Unparse(g Grammar, v interface{}, w io.Writer) (n int, err error) {
	return g[t].Unparse(g, v, w)
}

//-----------------------------------------------------------------------------

func (t Stack) Unparse(g Grammar, v interface{}, w io.Writer) (n int, err error) {
	panic(errors.Inconceivable)
}

//-----------------------------------------------------------------------------

func (t Named) Unparse(g Grammar, v interface{}, w io.Writer) (n int, err error) {
	err = unparse(g, t.Term, v, w, &n)
	return
}
