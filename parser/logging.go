package parser

import (
	"reflect"
	"strings"
	"sync/atomic"

	"github.com/sirupsen/logrus"
)

var depth int64

func indentf(format string, args ...interface{}) {
	logrus.Tracef(strings.Repeat("    ", int(atomic.LoadInt64(&depth)))+format, args...)
}

type enterexit struct{}

func enterf(format string, args ...interface{}) enterexit { //nolint:unparam
	indentf("--> "+format, args...)
	atomic.AddInt64(&depth, 1)
	return enterexit{}
}

func (enterexit) exitf(format string, ptrs ...interface{}) { //nolint:unparam
	atomic.AddInt64(&depth, -1)
	args := make([]interface{}, 0, len(ptrs))
	for _, ptr := range ptrs {
		args = append(args, reflect.ValueOf(ptr).Elem().Interface())
	}
	indentf("<-- "+format, args...)
}
