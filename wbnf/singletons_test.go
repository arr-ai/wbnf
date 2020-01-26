package wbnf

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertUnitaryPaths(t *testing.T, grammar string, paths ...string) bool {
	expected := append([]string{}, paths...)
	sort.Strings(expected)

	p, err := Compile(grammar)
	require.NoError(t, err)
	actual := p.Singletons().keys()
	sort.Strings(actual)

	return assert.Equal(t, expected, actual)
}

func TestUnitaryPathsSimple(t *testing.T) {
	t.Parallel()

	assertUnitaryPaths(t, `a -> 'x';`, "a.")
	assertUnitaryPaths(t, `a -> oper='x';`, "a.oper", "a.oper.")

	assertUnitaryPaths(t, `a -> 'x'+;`)
	assertUnitaryPaths(t, `a -> 'x' 'x';`)
	assertUnitaryPaths(t, `a -> 'x' 'y';`)
	assertUnitaryPaths(t, `a -> m='x' n='y';`, "a.m", "a.m.", "a.n", "a.n.")
	assertUnitaryPaths(t, `a -> m='x' m='y';`)
	assertUnitaryPaths(t, `a -> m='x' 'y';`, "a.", "a.m", "a.m.")
	assertUnitaryPaths(t, `a -> 'x' m='y';`, "a.", "a.m", "a.m.")

	assertUnitaryPaths(t, `a -> m=('x' n='y');`, "a.m", "a.m.", "a.m.n", "a.m.n.")
	assertUnitaryPaths(t, `a -> m=('x'? n='y');`, "a.m", "a.m.n", "a.m.n.")
	assertUnitaryPaths(t, `a -> m=('x'? n='y' 'z');`, "a.m", "a.m.n", "a.m.n.")
}

func TestUnitaryPathsStacks(t *testing.T) {
	assertUnitaryPaths(t, `a -> @:op='x' > @:op=('y'?) > z='z';`, "a.op.", "a@2.z", "a@2.z.")
}
