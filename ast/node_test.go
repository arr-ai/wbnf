package ast

import (
	"testing"

	"github.com/arr-ai/wbnf/parser"
	"github.com/stretchr/testify/assert"
)

func TestBranchScanner(t *testing.T) {
	str := "one\ntwo\nthree\nfour"

	assertBranchScanner(t, parser.NewScannerAt(str, 0, 4), Branch{
		"foo": One{
			Node: Leaf(*parser.NewScannerAt(str, 0, 4)),
		},
	})
	assertBranchScanner(t, parser.NewScannerAt(str, 0, 6), Branch{
		"foo": One{
			Node: Leaf(*parser.NewScannerAt(str, 0, 4)),
		},
		"bar": One{
			Node: Leaf(*parser.NewScannerAt(str, 5, 1)),
		},
	})
	assertBranchScanner(t, parser.NewScannerAt(str, 0, 11), Branch{
		"foo": One{
			Node: Leaf(*parser.NewScannerAt(str, 0, 4)),
		},
		"bar": One{
			Node: Leaf(*parser.NewScannerAt(str, 10, 1)),
		},
	})
	assertBranchScanner(t, parser.NewScannerAt(str, 0, 17), Branch{
		"foo": Many{
			Leaf(*parser.NewScannerAt(str, 0, 4)),
			Leaf(*parser.NewScannerAt(str, 7, 10)),
		},
	})
	assertBranchScanner(t, parser.NewScannerAt(str, 0, 17), Branch{
		"foo": Many{
			Leaf(*parser.NewScannerAt(str, 0, 4)),
			Leaf(*parser.NewScannerAt(str, 7, 10)),
		},
		"bar": Many{
			Leaf(*parser.NewScannerAt(str, 0, 4)),
			Leaf(*parser.NewScannerAt(str, 7, 1)),
		},
	})
	assertBranchScanner(t, parser.NewScannerAt(str, 0, 17), Branch{
		"foo": Many{
			Leaf(*parser.NewScannerAt(str, 0, 4)),
			Leaf(*parser.NewScannerAt(str, 7, 10)),
		},
		"bar": One{
			Node: Leaf(*parser.NewScannerAt(str, 5, 1)),
		},
	})

}

func assertBranchScanner(t *testing.T, s *parser.Scanner, b Branch) {
	assert.Equal(t, *s, b.Scanner())
}
