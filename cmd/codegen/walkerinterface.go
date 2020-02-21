package codegen

import (
	"sort"
	"strings"
)

type walker struct {
	types     map[string]grammarType
	startRule string
}

const suffix = "\n}\nfunc (w WalkerOps) Walk(tree {{startRuleType}}) { Walk{{startRuleType}}(tree, w) }\n"
const funcs = `	Enter{{typeName}} func ({{typeName}}) Stopper
	Exit{{typeName}} func ({{typeName}}) Stopper`

func (w walker) String() string {
	var parts []string
	for _, t := range w.types {
		if typeWantsCallbacks(t) {
			parts = append(parts, strings.ReplaceAll(funcs, "{{typeName}}", t.TypeName()))
		}
	}

	sort.Strings(parts)
	return "\ntype WalkerOps struct {\n" + strings.Join(parts, "\n") + strings.ReplaceAll(suffix, "{{startRuleType}}", w.startRule)
}

func typeWantsCallbacks(t grammarType) bool {
	return t.TypeName() != ""
}
