package ast

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/arr-ai/wbnf/errors"
	"github.com/arr-ai/wbnf/parser"
)

func (c One) String() string {
	if c.Node == nil {
		panic(errors.Inconceivable)
	}
	return c.Node.String()
}

func (c Many) String() string {
	var sb strings.Builder
	sb.WriteString("[")
	pre := ""
	complex := len(c) > 1
	if complex {
		wide := false
		for _, child := range c {
			if !child.narrow() {
				wide = true
				break
			}
		}
		if !wide {
			complex = false
		}
	}
	if complex {
		pre = "  "
		sb.WriteString("\n" + pre)
	}
	for i, child := range c {
		if i > 0 {
			if complex {
				sb.WriteString(",\n" + pre)
			} else {
				sb.WriteString(", ")
			}
		}
		fmt.Fprintf(&sb, "%s", strings.ReplaceAll(child.String(), "\n", "\n"+pre))
	}
	if complex {
		sb.WriteString(",\n")
	}
	sb.WriteString("]")
	return sb.String()
}

var specialCharRE = regexp.MustCompile("[[:cntrl:]:,'`(){}[\\]]")

func (l Leaf) String() string {
	var sb strings.Builder
	scanner := parser.Scanner(l)
	s := scanner.String()
	fmt.Fprintf(&sb, "%d‣", scanner.Offset())
	switch {
	case specialCharRE.MatchString(s):
		fmt.Fprintf(&sb, "%q", s)
	case strings.Contains(s, `"`):
		fmt.Fprintf(&sb, "`%s`", strings.ReplaceAll(s, "`", "``"))
	default:
		fmt.Fprintf(&sb, "%s", s)
	}
	return sb.String()
}

func (n Branch) String() string {
	var sb strings.Builder
	sb.WriteString("(")
	pre := ""
	if len(n) > 1 {
		sb.WriteString("\n  ")
		pre = "  "
	}
	i := 0
	names := make([]string, 0, len(n))
	for name := range n {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		group := n[name]
		if i > 0 {
			sb.WriteString(",\n  ")
		}
		i++
		child := strings.ReplaceAll(group.String(), "\n", "\n"+pre)
		if name == "" {
			name = "''"
		}
		fmt.Fprintf(&sb, "%s: %s", name, child)
	}
	if len(n) > 1 {
		sb.WriteString(",\n")
	}
	sb.WriteString(")")
	return sb.String()
}

func (c Extra) String() string {
	return fmt.Sprintf("%v", c.Data)
}

func (c One) Scanner() parser.Scanner {
	return c.Node.Scanner()
}

func (c Many) Scanner() parser.Scanner {
	childrenScanners := make([]parser.Scanner, 0, len(c))
	for _, n := range c {
		childrenScanners = append(childrenScanners, n.Scanner())
	}

	manyScanner, err := parser.MergeScanners(childrenScanners...)
	if err != nil {
		panic(err)
	}

	return manyScanner
}

func (c Extra) Scanner() parser.Scanner {
	panic("Scanner() not valid for Extra")
}

func (l Leaf) Scanner() parser.Scanner {
	return parser.Scanner(l)
}

func (n Branch) Scanner() parser.Scanner {
	if len(n) == 1 && n.oneChild() != nil {
		return n.oneChild().Scanner()
	}

	scanners := make([]parser.Scanner, 0)
	for childrenName, ch := range n {
		if !strings.HasPrefix(childrenName, "@") {
			switch c := ch.(type) {
			case One:
				scanners = append(scanners, c.Node.Scanner())
			case Many:
				scanners = append(scanners, c.Scanner())
			}
		}
	}

	branchScanner, err := parser.MergeScanners(scanners...)
	if err != nil {
		panic(err)
	}

	return branchScanner
}
