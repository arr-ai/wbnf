package parser

import (
	"fmt"
	"regexp"
	"strings"
)

type Scanner struct {
	src source // the source the scanner is drawing from
}

type source interface {
	filename() *string            // the name of the file from which the source was retrieved (if any)
	slice() string                // the sliced string
	sliceInfo() sliceInfo         // the sliced range information (relative to the entire source)
	sliceIn(i, length int) source // a new source sliced within the current slice
	prefix() string               // the prefix (before the slice - for debugging)
	suffix() string               // the suffix (after the slice - for debugging)
}

type sliceInfo struct {
	start  int // the (inclusive) start of the slice range within the source
	length int // the length of slice range
	line   int // the 1-indexed line number of the start of the slice within the source
	column int // the 1-indexed column number of the start of the slice within the source line
}

type stringSource struct {
	f      *string   // the location of the origin file in the file system
	origin *string   // the entire origin string
	info   sliceInfo // the slice range information (relative to the origin string)
}

func NewScanner(str string) *Scanner {
	return &Scanner{stringSource{origin: &str, info: newSliceInfo(str, 0, len(str))}}
}

func NewScannerWithFilename(str, filename string) *Scanner {
	return &Scanner{stringSource{f: &filename, origin: &str, info: newSliceInfo(str, 0, len(str))}}
}

func NewScannerAt(str string, offset, size int) *Scanner {
	return &Scanner{stringSource{origin: &str, info: newSliceInfo(str, offset, size)}}
}

// - Scanner

func (s Scanner) Filename() *string {
	return s.src.filename()
}

func (s Scanner) String() string {
	if s.src == nil {
		return ""
	}
	return s.src.slice()
}

func (s Scanner) Format(state fmt.State, c rune) {
	if c == 'q' {
		_, _ = fmt.Fprintf(state, "%q", s.src.slice())
	} else {
		_, _ = state.Write([]byte(s.src.slice()))
	}
}

func (s Scanner) Context() string {
	return fmt.Sprintf("%s\033[1;31m%s\033[0m%s",
		s.src.prefix(),
		s.src.slice(),
		s.src.suffix(),
	)
}

// The position of the start of the scanner within the original source.
func (s Scanner) Offset() int {
	return s.src.sliceInfo().start
}

// The 1-indexed line number of the start of the scanner within the original source.
func (s Scanner) Line() int {
	return s.src.sliceInfo().line
}

// The 1-indexed column number of the start of the scanner within the original source.
func (s Scanner) Column() int {
	return s.src.sliceInfo().column
}

func (s Scanner) Slice(a, b int) *Scanner {
	return &Scanner{src: s.src.sliceIn(a, b-a)}
}

func (s Scanner) Skip(i int) *Scanner {
	return &Scanner{src: s.src.sliceIn(i, s.src.sliceInfo().length-i)}
}

func (s *Scanner) Eat(i int, eaten *Scanner) *Scanner {
	eaten.src = s.src.sliceIn(0, i)
	*s = *s.Skip(i)
	return s
}

func (s *Scanner) EatString(str string, eaten *Scanner) bool {
	if strings.HasPrefix(s.src.slice(), str) {
		s.Eat(len(str), eaten)
		return true
	}
	return false
}

// EatRegexp eats the text matching a regexp, populating match (if != nil) with
// the whole match and captures (if != nil) with any captured groups. Returns
// n as the number of captures set and ok iff a match was found.
func (s *Scanner) EatRegexp(re *regexp.Regexp, match *Scanner, captures []Scanner) (n int, ok bool) {
	if loc := re.FindStringSubmatchIndex(s.src.slice()); loc != nil {
		if loc[0] != 0 {
			panic(`re not \A-anchored`)
		}
		if match != nil {
			*match = *s.Slice(loc[0], loc[1])
		}
		skip := loc[1]
		loc = loc[2:]
		n = len(loc) / 2
		if len(captures) > n {
			captures = captures[:n]
		}
		for i := range captures {
			captures[i] = *s.Slice(loc[2*i], loc[2*i+1])
		}
		*s = *s.Skip(skip)
		return n, true
	}
	return 0, false
}

// - stringSource

func (s stringSource) filename() *string {
	return s.f
}

func (s stringSource) slice() string {
	return (*s.origin)[s.info.start:s.info.start + s.info.length]
}

func (s stringSource) sliceInfo() sliceInfo {
	return s.info
}

func (s stringSource) sliceIn(i, length int) source {
	skippedLine, skippedCol := lineColumn(s.slice()[:i], i)
	var line, col int
	if skippedLine == 1 {
		line = s.info.line
		col = s.info.column + skippedCol - 1
	} else {
		line = s.info.line + skippedLine - 1
		col = skippedCol
	}
	info := sliceInfo{s.info.start + i, length, line, col}
	return stringSource{f: s.f, origin: s.origin, info: info}

}

func (s stringSource) prefix() string {
	return (*s.origin)[0:s.info.start]
}

func (s stringSource) suffix() string {
	return (*s.origin)[s.info.start + s.info.length:]
}

// The slice info for the given string and range.
func newSliceInfo(str string, start, length int) sliceInfo {
	line, col := lineColumn(str, start)
	return sliceInfo{start: start, length: length, line: line, column: col}
}

// The 1-indexed line and column number of the given position within the given string.
func lineColumn(str string, pos int) (line, col int) {
	prefix := str[:pos]
	line = strings.Count(prefix, "\n") + 1
	col = pos - strings.LastIndex(prefix, "\n")
	return
}
