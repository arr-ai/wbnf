package codegen

import (
	"fmt"
	"sort"
	"strings"
)

type VisitorWriter struct {
	types     map[string]grammarType
	startRule string
}

const suffix = "\n}\nfunc (w WalkerOps) Walk(tree {{startRuleType}}) { w.Walk{{startRuleType}}(tree) }\n"
const funcs = `	Enter{{typeName}} func ({{typeName}}) Stopper
	Exit{{typeName}} func ({{typeName}}) Stopper`

func (w VisitorWriter) String() string {
	var parts []string
	for _, t := range w.types {
		parts = append(parts, strings.ReplaceAll(funcs, "{{typeName}}", t.TypeName()))
	}
	sort.Strings(parts)
	startRule := GoTypeName(w.startRule)
	out := "\ntype WalkerOps struct {\n" + strings.Join(parts, "\n") + strings.ReplaceAll(suffix, "{{startRuleType}}", startRule)

	parts = []string{}
	for _, t := range w.types {
		if len(t.Children()) > 0 {
			parts = append(parts, w.getTypeWalker(t))
		}
	}
	sort.Strings(parts)
	return out + strings.Join(parts, "\n")
}

func (w *VisitorWriter) getTypeWalker(t grammarType) string {
	repl := strings.NewReplacer("{{.CtxName}}", t.TypeName(), `\n`, "\n")
	walker := repl.Replace(`func (w WalkerOps) Walk{{.CtxName}}(node {{.CtxName}}) Stopper {
	if fn := w.Enter{{.CtxName}}; fn != nil { 
		if s := fn(node); s != nil { if s.ExitNode() { return nil } else if s.Abort() { return s} }\n}\n`)

	for _, child := range t.Children() {
		funcs := child.CallbackData()
		switch child := child.(type) {
		case namedRule:
			if r, ok := w.types[child.returnType]; ok {
				if _, ok := r.(basicRule); ok {
					walker += getWalkerFuncs(funcs, false)
					continue
				}
			}
		case stackBackRef:
			walker += getWalkerFuncs(funcs, true)
			continue
		}
		if funcs != nil {
			walker += getWalkerFuncs(funcs, true)
		}
	}

	walker += repl.Replace(`
	if fn := w.Exit{{.CtxName}}; fn != nil { if s := fn(node); s != nil && s.Abort() { return s } }
	return nil\n}\n`)

	return walker
}

func GetVisitorWriter(types map[string]grammarType, startRule string) VisitorWriter {
	return VisitorWriter{types, startRule}
}

func getWalkerFuncs(funcs *callbackData, isWalker bool) string {
	var walker string
	if funcs.isMany {
		walker += fmt.Sprintf("for _, child := range node.All%s() {\n", funcs.getter)
	} else {
		walker += fmt.Sprintf("if child := node.One%s(); child != nil { child := *child\n", funcs.getter)
	}
	if isWalker {
		walker += fmt.Sprintf(`if s := w.Walk%s(child); s != nil { 
			if s.ExitNode() { return nil } else if s.Abort() { return s} } }`+"\n", funcs.walker)
	} else {
		walker += fmt.Sprintf(`if fn := w.Enter%s; fn != nil { 
			if s := fn(child); s != nil { 
				if s.ExitNode() { return nil } else if s.Abort() { return s} } 
			}
		}`+"\n", funcs.walker)
	}
	return walker
}
