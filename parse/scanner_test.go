package parse

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestScannerLineColumn(t *testing.T) {
	scanner := NewScanner("one\ntwo\nthree\nfour")

	// test the scanner starts at position 1,1
	assertLineColumn(t, scanner, 1, 1)

	// eat within the same line:
	// test the eaten scanner is left at the existing position
	// test the scanner is advanced within the line
	eaten := Scanner{}
	scanner.Eat(1, &eaten)
	assertLineColumn(t, &eaten, 1, 1)
	assertLineColumn(t, scanner, 1, 2)

	// eat a line
	scanner.Eat(3, &eaten)
	assertLineColumn(t, &eaten, 1, 2)
	assertLineColumn(t, scanner, 2, 1)

	// eat multiple lines and into a column
	scanner.Eat(12, &eaten)
	assertLineColumn(t, &eaten, 2, 1)
	assertLineColumn(t, scanner, 4, 3)
}

func assertLineColumn(t *testing.T, scanner *Scanner, line, column int) {
	l, c := scanner.Position()
	assert.Equal(t, line, l)
	assert.Equal(t, column, c)
}
