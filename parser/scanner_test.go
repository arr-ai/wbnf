package parser

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestScannerMerge(t *testing.T) {
	str := "one\ntwo\nthree\nfour"
	src := stringSource{origin: str}

	assertMergedScanner(t, src, 0, 5, []Scanner{*NewScannerAt(str, 0, 5)})
	assertMergedScanner(t, src, 0, len(str), []Scanner{*NewScanner(str), *NewScanner(str)})
	assertMergedScanner(t, src, 0, len(str), []Scanner{*NewScanner(str), *NewScannerAt(str, 0, 1)})
	assertMergedScanner(t, src, 0, 11, []Scanner{*NewScannerAt(str, 0, 1), *NewScannerAt(str, 5, 6)})
	assertMergedScanner(t, src, 0, 11,
		[]Scanner{*NewScannerAt(str, 0, 1), *NewScannerAt(str, 3, 1), *NewScannerAt(str, 5, 6)})
	assertMergedScanner(t, src, 0, 6,
		[]Scanner{*NewScannerAt(str, 0, 1), *NewScannerAt(str, 0, 4), *NewScannerAt(str, 0, 6)})

	assertMergedScannerErr(t, errors.New("needs at least one scanner"), []Scanner{})
	assertMergedScannerErr(t,
		errors.New("scanners' sources are not the same: {one\ntwo\nthree\nfour } vs {another src }"),
		[]Scanner{*NewScanner(str), *NewScanner("another src")})
}

func TestContains(t *testing.T) {
	t.Parallel()

	// in the middle
	s1 := NewScannerAt("this is a random sentence", 5, 3)
	s2 := NewScannerAt("this is a random sentence", 6, 1)
	assert.True(t, s1.Contains(*s2))

	// same range
	s2.sliceStart = 5
	s2.sliceLength = 3
	assert.True(t, s1.Contains(*s2))

	// start same, smaller length
	s2.sliceLength = 2
	assert.True(t, s1.Contains(*s2))

	// start in the middle, end at the same part
	s2.sliceStart = 6
	assert.True(t, s1.Contains(*s2))

	// start earlier
	s2.sliceStart = 4
	assert.False(t, s1.Contains(*s2))

	// start in the middle, end outside
	s2.sliceStart = 6
	s2.sliceLength = 5
	assert.False(t, s1.Contains(*s2))

	// start and end outside range
	s2.sliceStart = 4
	s2.sliceLength = 5
	assert.False(t, s1.Contains(*s2))

	// different filename
	s1 = NewScannerWithFilename("file1", "file1.wbnf")
	s2 = NewScannerWithFilename("file2", "file2.wbnf")
	assert.False(t, s1.Contains(*s2))

	// different source
	s1 = NewScanner("source1")
	s2 = NewScanner("source2")
	assert.False(t, s1.Contains(*s2))
}

func assertMergedScanner(t *testing.T, src source, offset, length int, items []Scanner) { //nolint:unparam
	s, err := MergeScanners(items...)
	assert.NoError(t, err)
	assert.Equal(t, src, s.src)
	assert.Equal(t, offset, s.sliceStart)
	assert.Equal(t, length, s.sliceLength)
}

func assertMergedScannerErr(t *testing.T, err error, items []Scanner) {
	_, e := MergeScanners(items...)
	assert.Equal(t, err, e)
}
