package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPushType_AddUnnamedTokensToType(t *testing.T) {
	tm := TypeMap{}

	tm.pushType("", "Foo", unnamedToken{parent: "Foo", count: wantOneGetter})
	tm.pushType("", "Bar", unnamedToken{parent: "Bar", count: wantAllGetter})

	// These two should combine to a rule() with a many unnamed token
	tm.pushType("", "Too", unnamedToken{parent: "Too", count: wantOneGetter})
	tm.pushType("", "Too", unnamedToken{parent: "Too", count: wantOneGetter})

	tm.pushType("", "Three", nil)

	assert.IsType(t, rule{}, tm["BarNode"])
	assert.IsType(t, basicRule(""), tm["FooNode"])
	assert.IsType(t, rule{}, tm["TooNode"])
	assert.IsType(t, rule{}, tm["ThreeNode"])
	testChildren(t, tm["BarNode"].Children(), childrenTestData{
		"Token": {t: unnamedToken{}, quant: wantAllGetter},
	})
	testChildren(t, tm["TooNode"].Children(), childrenTestData{
		"Token": {t: unnamedToken{}, quant: wantAllGetter},
	})
}

func TestPushType_AddUnnamedTokensToExistingType(t *testing.T) {
	tm := TypeMap{}

	tm.pushType("", "Foo", unnamedToken{parent: "Foo", count: wantOneGetter})
	tm.pushType("bar", "Foo", namedToken{name: "bar", parent: "Foo", count: wantOneGetter})

	assert.Nil(t, nil, tm["BarNode"])
	assert.IsType(t, rule{}, tm["FooNode"])

	testChildren(t, tm["FooNode"].Children(), childrenTestData{
		"Token": {t: unnamedToken{}, quant: wantOneGetter},
		"bar":   {t: namedToken{}, quant: wantOneGetter},
	})
}
