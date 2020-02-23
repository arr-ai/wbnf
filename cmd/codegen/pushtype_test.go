package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPushType_AddUnnamedTokensToType(t *testing.T) {
	tm := TypeMap{}

	tm.pushType("", "Foo", unnamedToken{parent: "Foo", count: setWantOneGetter()})
	tm.pushType("", "Bar", unnamedToken{parent: "Bar", count: setWantAllGetter()})

	// These two should combine to a rule() with a many unnamed token
	tm.pushType("", "Too", unnamedToken{parent: "Too", count: setWantOneGetter()})
	tm.pushType("", "Too", unnamedToken{parent: "Too", count: setWantOneGetter()})

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

	tm.pushType("", "Foo", unnamedToken{parent: "Foo", count: setWantOneGetter()})
	tm.pushType("bar", "Foo", namedToken{name: "bar", parent: "Foo", count: setWantOneGetter()})

	assert.Nil(t, nil, tm["BarNode"])
	assert.IsType(t, rule{}, tm["FooNode"])

	testChildren(t, tm["FooNode"].Children(), childrenTestData{
		"Token": {t: unnamedToken{}, quant: wantOneGetter},
		"bar":   {t: namedToken{}, quant: wantOneGetter},
	})
}

func TestPushType_MultipleWantOnesShouldNotCombine(t *testing.T) {
	tm := TypeMap{}

	c := setWantOneGetter()
	tm.pushType("", "Foo", namedToken{name: "op", parent: "Foo", count: c.pushSingleNode(123)})
	tm.pushType("", "Foo", namedToken{name: "op", parent: "Foo", count: c.pushSingleNode(456)})
	tm.pushType("", "Foo", namedToken{name: "op", parent: "Foo", count: c.pushSingleNode(789)})

	assert.IsType(t, rule{}, tm["FooNode"])
	testChildren(t, tm["FooNode"].Children(), childrenTestData{
		"op": {t: namedToken{}, quant: wantOneGetter},
	})
}

func TestPushType_MultipleWantOnesShouldCombine(t *testing.T) {
	tm := TypeMap{}

	c := setWantOneGetter()
	tm.pushType("", "Foo", namedToken{name: "op", parent: "Foo", count: c.pushSingleNode(123)})
	tm.pushType("", "Foo", namedToken{name: "op", parent: "Foo", count: c.pushSingleNode(123)})
	tm.pushType("", "Foo", namedToken{name: "op", parent: "Foo", count: c.pushSingleNode(789)})

	assert.IsType(t, rule{}, tm["FooNode"])
	testChildren(t, tm["FooNode"].Children(), childrenTestData{
		"op": {t: namedToken{}, quant: wantAllGetter | wantOneGetter},
	})
}

func TestPushType_MultipleWantOnesShouldCombine2(t *testing.T) {
	tm := TypeMap{}

	c := setWantOneGetter()
	tm.pushType("", "Foo", unnamedToken{parent: "Foo", count: c.pushSingleNode(123)})
	tm.pushType("", "Foo", unnamedToken{parent: "Foo", count: c.pushSingleNode(123)})
	tm.pushType("", "Foo", unnamedToken{parent: "Foo", count: c.pushSingleNode(789)})

	assert.IsType(t, rule{}, tm["FooNode"])
	testChildren(t, tm["FooNode"].Children(), childrenTestData{
		"Token": {t: unnamedToken{}, quant: wantAllGetter | wantOneGetter},
	})
}
