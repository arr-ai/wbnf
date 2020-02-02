package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func AssertEqualNodes(t *testing.T, v, u Node) bool {
	if !assertEqualNodes(t, v, u, []interface{}{}) {
		t.Logf("\nexpected: %v\nactual:   %v", v, u)
		return false
	}
	return true
}

func assertEqualNodes(t *testing.T, v, u Node, path []interface{}) bool {
	result := true
	ok := func(ok bool) bool {
		result = result && ok
		return ok
	}
	ok(assert.Equal(t, v.Tag, u.Tag, "%v", path))
	ok(assert.Equal(t, v.Extra, u.Extra, "%v", path))
	ok(assert.Equal(t, len(v.Children), len(u.Children)))
	n := len(v.Children)
	if n > len(u.Children) {
		n = len(u.Children)
	}
	for i := 0; i < n; i++ {
		subpath := append(path, i)
		vc := v.Children[i]
		uc := u.Children[i]
		if ok(assert.IsType(t, vc, uc, "%v: %v != %v", subpath, vc, uc)) {
			switch vc := vc.(type) {
			case Node:
				ok(assertEqualNodes(t, vc, uc.(Node), subpath))
			case Scanner:
				ok(assert.Equal(t, vc, uc, "%v: %v != %v", subpath, vc, uc))
			default:
				ok(false)
				t.Errorf("%v unexpected type %T: %[1]v %v", vc, uc)
			}
		}
	}
	for i, c := range v.Children[n:] {
		t.Errorf("%v expected node not found: %v", append(path, n+i), c)
	}
	for i, c := range u.Children[n:] {
		t.Errorf("%v unexpected node found: %v", append(path, n+i), c)
	}
	return result
}
