package parse

import (
	"fmt"
	"regexp"
	"strings"
)

type Scanner struct {
	src         source // the source the scanner is drawing from
	sliceStart  int    // the start of the slice visible to the scanner
	sliceLength int    // the length of the slice visible to the scanner
}

type source interface {
	length() int                // the length of the entire source string
	slice(i, length int) string // the string of the given slice
	filename() string           // the name of the file from which the source is derived (or empty if none)
}

type stringSource struct {
	origin *string // the entire source string
	f      string  // the source filename
}

func NewScanner(str string) *Scanner {
	return &Scanner{stringSource{origin: &str}, 0, len(str)}
}

func NewScannerWithFilename(str, filename string) *Scanner {
	return &Scanner{stringSource{&str, filename}, 0, len(str)}
}

func NewScannerAt(str string, offset, size int) *Scanner {
	return &Scanner{stringSource{origin: &str}, offset, size}
}

// - Scanner

// The name of the file from which the source is derived (or empty if none).
func (s Scanner) Filename() string {
	return s.src.filename()
}

func (s Scanner) String() string {
	if s.src == nil {
		return ""
	}
	return s.slice()
}

func (s Scanner) Format(state fmt.State, c rune) {
	if c == 'q' {
		_, _ = fmt.Fprintf(state, "%q", s.slice())
	} else {
		_, _ = state.Write([]byte(s.slice()))
	}
}

func (s Scanner) Context() string {
	end := s.sliceStart + s.sliceLength
	return fmt.Sprintf("%s\033[1;31m%s\033[0m%s",
		s.src.slice(0, s.sliceStart),
		s.slice(),
		s.src.slice(end, s.src.length()-end),
	)
}

// The position of the start of the scanner within the original source.
func (s Scanner) Offset() int {
	return s.sliceStart
}

// The 1-indexed line and column number of the start of the scanner within the original source.
func (s Scanner) Position() (int, int) {
	return lineColumn(s.src.slice(0, s.sliceStart), s.sliceStart)
}

// The slice that is visible to the scanner
func (s Scanner) slice() string {
	return s.src.slice(s.sliceStart, s.sliceLength)
}

func (s Scanner) Slice(a, b int) *Scanner {
	return &Scanner{s.src, s.sliceStart + a, b - a}
}

func (s Scanner) Skip(i int) *Scanner {
	return &Scanner{s.src, s.sliceStart + i, s.sliceLength - i}
}

func (s *Scanner) Eat(i int, eaten *Scanner) *Scanner {
	eaten.src = s.src
	eaten.sliceStart = s.sliceStart
	eaten.sliceLength = i
	*s = *s.Skip(i)
	return s
}

func (s *Scanner) EatString(str string, eaten *Scanner) bool {
	if strings.HasPrefix(s.slice(), str) {
		s.Eat(len(str), eaten)
		return true
	}
	return false
}

// EatRegexp eats the text matching a regexp, populating match (if != nil) with
// the whole match and captures (if != nil) with any captured groups. Returns
// n as the number of captures set and ok iff a match was found.
func (s *Scanner) EatRegexp(re *regexp.Regexp, match *Scanner, captures []Scanner) (n int, ok bool) {
	if loc := re.FindStringSubmatchIndex(s.slice()); loc != nil {
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

func (s stringSource) length() int {
	return len(*s.origin)
}

func (s stringSource) slice(i, length int) string {
	return (*s.origin)[i : i+length]
}

func (s stringSource) filename() string {
	return s.f
}

// The 1-indexed line and column number of the given position within the given string.
func lineColumn(str string, pos int) (line, col int) {
	prefix := str[:pos]
	line = strings.Count(prefix, "\n") + 1
	col = pos - strings.LastIndex(prefix, "\n")
	return
}
