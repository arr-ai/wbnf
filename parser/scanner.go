package parser

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

type Scanner struct {
	src         source // the source the scanner is drawing from
	sliceStart  int    // the start of the slice visible to the scanner, based on the original src
	sliceLength int    // the length of the slice visible to the scanner, based on the original src
}

type source interface {
	length() int                // the length of the entire source string
	slice(i, length int) string // the string of the given slice
	filename() string           // the name of the file from which the source is derived (or empty if none)
	stripSource(i, length int) source
}

type stringSource struct {
	origin string // the entire source string
	f      string // the source filename
}

func NewScanner(str string) *Scanner {
	return &Scanner{stringSource{origin: str}, 0, len(str)}
}

func NewScannerWithFilename(str, filename string) *Scanner {
	return &Scanner{stringSource{str, filename}, 0, len(str)}
}

func NewScannerAt(str string, offset, size int) *Scanner {
	return &Scanner{stringSource{origin: str}, offset, size}
}

// - Scanner

func (s Scanner) StripSource() Scanner {
	s.src = s.src.stripSource(s.sliceStart, s.sliceLength)
	return s
}

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

func (s Scanner) IsNil() bool {
	return s.src == nil
}

func (s Scanner) Format(state fmt.State, c rune) {
	if c == 'q' {
		_, _ = fmt.Fprintf(state, "%q", s.slice())
	} else {
		_, _ = state.Write([]byte(s.slice()))
	}
}

var (
	NoLimit      = -1
	DefaultLimit = 1
)

func (s Scanner) Contains(sn Scanner) bool {
	if s.Filename() != sn.Filename() || s.src != sn.src {
		return false
	}

	return s.sliceStart <= sn.sliceStart &&
		s.sliceStart+s.sliceLength >= sn.sliceStart+sn.sliceLength
}

func (s Scanner) Context(limitLines int) string {
	end := s.sliceStart + s.sliceLength
	lineno, colno := s.Position()

	aboveCxt := s.src.slice(0, s.sliceStart)
	belowCxt := s.src.slice(end, s.src.length()-end)
	if limitLines != NoLimit {
		a := strings.Split(aboveCxt, "\n")
		if len(a) > limitLines {
			aboveCxt = strings.Join(a[len(a)-limitLines-1:], "\n")
		}
		b := strings.Split(belowCxt, "\n")
		if len(b) > limitLines {
			belowCxt = strings.Join(b[:limitLines], "\n")
		}
	}

	return fmt.Sprintf("\n\033[1;37m%s:%d:%d:\033[0m\n%s\033[1;31m%s\033[0m%s",
		s.Filename(),
		lineno,
		colno,
		aboveCxt,
		s.slice(),
		belowCxt,
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

func MergeScanners(items ...Scanner) (Scanner, error) {
	if len(items) == 0 {
		return Scanner{}, errors.New("needs at least one scanner")
	}
	if len(items) == 1 {
		return items[0], nil
	}

	l, r := items[0].sliceStart, items[0].sliceStart+items[0].sliceLength
	src := items[0].src

	for _, v := range items[1:] {
		if v.src != src {
			return Scanner{}, fmt.Errorf("scanners' sources are not the same: %s vs %s", src, v.src)
		}
		if v.sliceStart < l {
			l = v.sliceStart
		}
		if v.sliceStart+v.sliceLength > r {
			r = v.sliceStart + v.sliceLength
		}
	}

	return Scanner{
		src:         src,
		sliceStart:  l,
		sliceLength: r - l,
	}, nil
}

// Eat returns a scanner containing the next i bytes and advances s past them.
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

func (s stringSource) stripSource(offset, size int) source {
	s.origin = s.slice(offset, size)
	return s
}

func (s stringSource) length() int {
	return len(s.origin)
}

func (s stringSource) slice(i, length int) string {
	// Since offset and length based on the original origin string, so they might be out of range
	if i < 0 || i+length < 0 || i > len(s.origin) || i+length > len(s.origin) {
		return s.origin
	}
	return (s.origin)[i : i+length]
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
