// Code generated by "ωBNF gen" DO NOT EDIT.
// $ wbnf gen --grammar ../examples/wbnf.wbnf --start grammar --pkg wbnf
package wbnf

import (
	"github.com/arr-ai/wbnf/ast"
	"github.com/arr-ai/wbnf/parser"
)

func Grammar() parser.Parsers {
	return parser.Grammar{"grammar": parser.Some(parser.Rule(`stmt`)),
		"stmt": parser.Oneof{parser.Rule(`COMMENT`),
			parser.Rule(`prod`),
			parser.Rule(`pragma`)},
		"prod": parser.Seq{parser.Rule(`IDENT`),
			parser.S("->"),
			parser.Some(parser.Rule(`term`)),
			parser.S(";")},
		"term": parser.Stack{parser.Delim{Term: parser.Rule(`@`),
			Sep: parser.Eq("op",
				parser.S(">")),
			Assoc: parser.NonAssociative},
			parser.Delim{Term: parser.Rule(`@`),
				Sep: parser.Eq("op",
					parser.S("|")),
				Assoc: parser.NonAssociative},
			parser.Some(parser.Rule(`@`)),
			parser.Seq{parser.Rule(`named`),
				parser.Any(parser.Rule(`quant`))}},
		"named": parser.Seq{parser.Opt(parser.Seq{parser.Rule(`IDENT`),
			parser.Eq("op",
				parser.S("="))}),
			parser.Rule(`atom`)},
		"quant": parser.Oneof{parser.Eq("op",
			parser.RE(`[?*+]`)),
			parser.Seq{parser.S("{"),
				parser.Opt(parser.Eq("min",
					parser.Rule(`INT`))),
				parser.S(","),
				parser.Opt(parser.Eq("max",
					parser.Rule(`INT`))),
				parser.S("}")},
			parser.Seq{parser.Eq("op",
				parser.RE(`<:|:>?`)),
				parser.Opt(parser.Eq("opt_leading",
					parser.S(","))),
				parser.Rule(`named`),
				parser.Opt(parser.Eq("opt_trailing",
					parser.S(",")))}},
		"atom": parser.Oneof{parser.Rule(`IDENT`),
			parser.Rule(`STR`),
			parser.Rule(`RE`),
			parser.Rule(`REF`),
			parser.Seq{parser.S("("),
				parser.Rule(`term`),
				parser.S(")")},
			parser.Seq{parser.S("("),
				parser.S(")")}},
		"COMMENT": parser.RE(`//.*$|(?s:/\*(?:[^*]|\*+[^*/])\*/)`),
		"IDENT":   parser.RE(`@|[A-Za-z_\.]\w*`),
		"INT":     parser.RE(`\d+`),
		"STR":     parser.RE(`"(?:\\.|[^\\"])*"|'(?:\\.|[^\\'])*'|` + "`" + `(?:` + "`" + `` + "`" + `|[^` + "`" + `])*` + "`" + ``),
		"RE":      parser.RE(`/{(?:\\.|{(?:(?:\d+(?:,\d*)?|,\d+)\})?|\[(?:\\.|\[:^?[a-z]+:\]|[^\]])+]|[^\\{\}])*\}|(?:(?:\[(?:\\.|\[:^?[a-z]+:\]|[^\]])+]|\\[pP](?:[a-z]|\{[a-zA-Z_]+\})|\\[a-zA-Z]|[.^$])(?:(?:[+*?]|\{\d+,?\d?\})\??)?)+`),
		"REF": parser.Seq{parser.S("%"),
			parser.Rule(`IDENT`),
			parser.Opt(parser.Seq{parser.S("="),
				parser.Eq("default",
					parser.Rule(`STR`))})},
		"pragma": parser.Eq("import",
			parser.Seq{parser.S(".import"),
				parser.Eq("path",
					parser.Delim{Term: parser.Oneof{parser.S(".."),
						parser.S("."),
						parser.RE(`[a-zA-Z0-9.:]+`)},
						Sep:             parser.S("/"),
						Assoc:           parser.NonAssociative,
						CanStartWithSep: true})}),
		".wrapRE": parser.RE(`\s*()\s*`)}.Compile(nil)
}

type Stopper interface {
	ExitNode() bool
	Abort() bool
}
type nodeExiter struct{}

func (n *nodeExiter) ExitNode() bool { return true }
func (n *nodeExiter) Abort() bool    { return false }

type aborter struct{}

func (n *aborter) ExitNode() bool { return true }
func (n *aborter) Abort() bool    { return true }

const (
	IdentCOMMENT     = "COMMENT"
	IdentIDENT       = "IDENT"
	IdentRE          = "RE"
	IdentREF         = "REF"
	IdentSTR         = "STR"
	IdentAtom        = "atom"
	IdentDefault     = "default"
	IdentImport      = "import"
	IdentMax         = "max"
	IdentMin         = "min"
	IdentNamed       = "named"
	IdentOp          = "op"
	IdentOptLeading  = "opt_leading"
	IdentOptTrailing = "opt_trailing"
	IdentPath        = "path"
	IdentPragma      = "pragma"
	IdentProd        = "prod"
	IdentQuant       = "quant"
	IdentStmt        = "stmt"
	IdentTerm        = "term"
)

type WrapreNode struct{ ast.Node }

func (c WrapreNode) String() string {
	if c.Node == nil {
		return ""
	}
	return c.Node.Scanner().String()
}
func WalkWrapreNode(node WrapreNode, ops WalkerOps) Stopper {
	if fn := ops.EnterWrapreNode; fn != nil {
		s := fn(node)
		switch {
		case s == nil:
		case s.ExitNode():
			return nil
		case s.Abort():
			return s
		}
	}

	if fn := ops.ExitWrapreNode; fn != nil {
		if s := fn(node); s != nil && s.Abort() {
			return s
		}
	}
	return nil
}

type CommentNode struct{ ast.Node }

func (c CommentNode) String() string {
	if c.Node == nil {
		return ""
	}
	return c.Node.Scanner().String()
}
func WalkCommentNode(node CommentNode, ops WalkerOps) Stopper {
	if fn := ops.EnterCommentNode; fn != nil {
		s := fn(node)
		switch {
		case s == nil:
		case s.ExitNode():
			return nil
		case s.Abort():
			return s
		}
	}

	if fn := ops.ExitCommentNode; fn != nil {
		if s := fn(node); s != nil && s.Abort() {
			return s
		}
	}
	return nil
}

type IdentNode struct{ ast.Node }

func (c IdentNode) String() string {
	if c.Node == nil {
		return ""
	}
	return c.Node.Scanner().String()
}
func WalkIdentNode(node IdentNode, ops WalkerOps) Stopper {
	if fn := ops.EnterIdentNode; fn != nil {
		s := fn(node)
		switch {
		case s == nil:
		case s.ExitNode():
			return nil
		case s.Abort():
			return s
		}
	}

	if fn := ops.ExitIdentNode; fn != nil {
		if s := fn(node); s != nil && s.Abort() {
			return s
		}
	}
	return nil
}

type IntNode struct{ ast.Node }

func (c IntNode) String() string {
	if c.Node == nil {
		return ""
	}
	return c.Node.Scanner().String()
}
func WalkIntNode(node IntNode, ops WalkerOps) Stopper {
	if fn := ops.EnterIntNode; fn != nil {
		s := fn(node)
		switch {
		case s == nil:
		case s.ExitNode():
			return nil
		case s.Abort():
			return s
		}
	}

	if fn := ops.ExitIntNode; fn != nil {
		if s := fn(node); s != nil && s.Abort() {
			return s
		}
	}
	return nil
}

type ReNode struct{ ast.Node }

func (c ReNode) String() string {
	if c.Node == nil {
		return ""
	}
	return c.Node.Scanner().String()
}
func WalkReNode(node ReNode, ops WalkerOps) Stopper {
	if fn := ops.EnterReNode; fn != nil {
		s := fn(node)
		switch {
		case s == nil:
		case s.ExitNode():
			return nil
		case s.Abort():
			return s
		}
	}

	if fn := ops.ExitReNode; fn != nil {
		if s := fn(node); s != nil && s.Abort() {
			return s
		}
	}
	return nil
}

type RefNode struct{ ast.Node }

func (c RefNode) AllIdent() []IdentNode {
	var out []IdentNode
	for _, child := range ast.All(c.Node, "IDENT") {
		out = append(out, IdentNode{child})
	}
	return out
}

func (c RefNode) OneIdent() IdentNode {
	return IdentNode{ast.First(c.Node, "IDENT")}
}

func (c RefNode) AllDefault() []StrNode {
	var out []StrNode
	for _, child := range ast.All(c.Node, "default") {
		out = append(out, StrNode{child})
	}
	return out
}

func (c RefNode) OneDefault() StrNode {
	return StrNode{ast.First(c.Node, "default")}
}
func WalkRefNode(node RefNode, ops WalkerOps) Stopper {
	if fn := ops.EnterRefNode; fn != nil {
		s := fn(node)
		switch {
		case s == nil:
		case s.ExitNode():
			return nil
		case s.Abort():
			return s
		}
	}

	for _, child := range node.AllIdent() {
		s := WalkIdentNode(child, ops)
		switch {
		case s == nil:
		case s.ExitNode():
			return nil
		case s.Abort():
			return s
		}
	}
	if fn := ops.ExitRefNode; fn != nil {
		if s := fn(node); s != nil && s.Abort() {
			return s
		}
	}
	return nil
}

type StrNode struct{ ast.Node }

func (c StrNode) String() string {
	if c.Node == nil {
		return ""
	}
	return c.Node.Scanner().String()
}
func WalkStrNode(node StrNode, ops WalkerOps) Stopper {
	if fn := ops.EnterStrNode; fn != nil {
		s := fn(node)
		switch {
		case s == nil:
		case s.ExitNode():
			return nil
		case s.Abort():
			return s
		}
	}

	if fn := ops.ExitStrNode; fn != nil {
		if s := fn(node); s != nil && s.Abort() {
			return s
		}
	}
	return nil
}

type AtomNode struct{ ast.Node }

func (c AtomNode) Choice() int {
	return ast.Choice(c.Node)
}

func (c AtomNode) AllIdent() []IdentNode {
	var out []IdentNode
	for _, child := range ast.All(c.Node, "IDENT") {
		out = append(out, IdentNode{child})
	}
	return out
}

func (c AtomNode) OneIdent() IdentNode {
	return IdentNode{ast.First(c.Node, "IDENT")}
}

func (c AtomNode) AllRe() []ReNode {
	var out []ReNode
	for _, child := range ast.All(c.Node, "RE") {
		out = append(out, ReNode{child})
	}
	return out
}

func (c AtomNode) OneRe() ReNode {
	return ReNode{ast.First(c.Node, "RE")}
}

func (c AtomNode) AllRef() []RefNode {
	var out []RefNode
	for _, child := range ast.All(c.Node, "REF") {
		out = append(out, RefNode{child})
	}
	return out
}

func (c AtomNode) OneRef() RefNode {
	return RefNode{ast.First(c.Node, "REF")}
}

func (c AtomNode) AllStr() []StrNode {
	var out []StrNode
	for _, child := range ast.All(c.Node, "STR") {
		out = append(out, StrNode{child})
	}
	return out
}

func (c AtomNode) OneStr() StrNode {
	return StrNode{ast.First(c.Node, "STR")}
}

func (c AtomNode) AllTerm() []TermNode {
	var out []TermNode
	for _, child := range ast.All(c.Node, "term") {
		out = append(out, TermNode{child})
	}
	return out
}

func (c AtomNode) OneTerm() TermNode {
	return TermNode{ast.First(c.Node, "term")}
}
func WalkAtomNode(node AtomNode, ops WalkerOps) Stopper {
	if fn := ops.EnterAtomNode; fn != nil {
		s := fn(node)
		switch {
		case s == nil:
		case s.ExitNode():
			return nil
		case s.Abort():
			return s
		}
	}

	for _, child := range node.AllIdent() {
		s := WalkIdentNode(child, ops)
		switch {
		case s == nil:
		case s.ExitNode():
			return nil
		case s.Abort():
			return s
		}
	}
	for _, child := range node.AllRe() {
		s := WalkReNode(child, ops)
		switch {
		case s == nil:
		case s.ExitNode():
			return nil
		case s.Abort():
			return s
		}
	}
	for _, child := range node.AllRef() {
		s := WalkRefNode(child, ops)
		switch {
		case s == nil:
		case s.ExitNode():
			return nil
		case s.Abort():
			return s
		}
	}
	for _, child := range node.AllStr() {
		s := WalkStrNode(child, ops)
		switch {
		case s == nil:
		case s.ExitNode():
			return nil
		case s.Abort():
			return s
		}
	}
	for _, child := range node.AllTerm() {
		s := WalkTermNode(child, ops)
		switch {
		case s == nil:
		case s.ExitNode():
			return nil
		case s.Abort():
			return s
		}
	}
	if fn := ops.ExitAtomNode; fn != nil {
		if s := fn(node); s != nil && s.Abort() {
			return s
		}
	}
	return nil
}

type GrammarNode struct{ ast.Node }

func (c GrammarNode) AllStmt() []StmtNode {
	var out []StmtNode
	for _, child := range ast.All(c.Node, "stmt") {
		out = append(out, StmtNode{child})
	}
	return out
}

func (c GrammarNode) OneStmt() StmtNode {
	return StmtNode{ast.First(c.Node, "stmt")}
}
func WalkGrammarNode(node GrammarNode, ops WalkerOps) Stopper {
	if fn := ops.EnterGrammarNode; fn != nil {
		s := fn(node)
		switch {
		case s == nil:
		case s.ExitNode():
			return nil
		case s.Abort():
			return s
		}
	}

	for _, child := range node.AllStmt() {
		s := WalkStmtNode(child, ops)
		switch {
		case s == nil:
		case s.ExitNode():
			return nil
		case s.Abort():
			return s
		}
	}
	if fn := ops.ExitGrammarNode; fn != nil {
		if s := fn(node); s != nil && s.Abort() {
			return s
		}
	}
	return nil
}

type NamedNode struct{ ast.Node }

func (c NamedNode) AllIdent() []IdentNode {
	var out []IdentNode
	for _, child := range ast.All(c.Node, "IDENT") {
		out = append(out, IdentNode{child})
	}
	return out
}

func (c NamedNode) OneIdent() IdentNode {
	return IdentNode{ast.First(c.Node, "IDENT")}
}

func (c NamedNode) AllAtom() []AtomNode {
	var out []AtomNode
	for _, child := range ast.All(c.Node, "atom") {
		out = append(out, AtomNode{child})
	}
	return out
}

func (c NamedNode) OneAtom() AtomNode {
	return AtomNode{ast.First(c.Node, "atom")}
}

func (c NamedNode) AllOp() []string {
	var out []string
	for _, child := range ast.All(c.Node, "op") {
		out = append(out, ast.First(child, "").Scanner().String())
	}
	return out
}

func (c NamedNode) OneOp() string {
	if child := ast.First(c.Node, "op"); child != nil {
		return ast.First(child, "").Scanner().String()
	}
	return ""
}
func WalkNamedNode(node NamedNode, ops WalkerOps) Stopper {
	if fn := ops.EnterNamedNode; fn != nil {
		s := fn(node)
		switch {
		case s == nil:
		case s.ExitNode():
			return nil
		case s.Abort():
			return s
		}
	}

	for _, child := range node.AllIdent() {
		s := WalkIdentNode(child, ops)
		switch {
		case s == nil:
		case s.ExitNode():
			return nil
		case s.Abort():
			return s
		}
	}
	for _, child := range node.AllAtom() {
		s := WalkAtomNode(child, ops)
		switch {
		case s == nil:
		case s.ExitNode():
			return nil
		case s.Abort():
			return s
		}
	}
	if fn := ops.ExitNamedNode; fn != nil {
		if s := fn(node); s != nil && s.Abort() {
			return s
		}
	}
	return nil
}

type PragmaNode struct{ ast.Node }

func (c PragmaNode) Choice() int {
	return ast.Choice(c.Node)
}

func (c PragmaNode) AllImport() []TermNode {
	var out []TermNode
	for _, child := range ast.All(c.Node, "import") {
		out = append(out, TermNode{child})
	}
	return out
}

func (c PragmaNode) OneImport() TermNode {
	return TermNode{ast.First(c.Node, "import")}
}

func (c PragmaNode) AllPath() []TermNode {
	var out []TermNode
	for _, child := range ast.All(c.Node, "path") {
		out = append(out, TermNode{child})
	}
	return out
}

func (c PragmaNode) OnePath() TermNode {
	return TermNode{ast.First(c.Node, "path")}
}
func WalkPragmaNode(node PragmaNode, ops WalkerOps) Stopper {
	if fn := ops.EnterPragmaNode; fn != nil {
		s := fn(node)
		switch {
		case s == nil:
		case s.ExitNode():
			return nil
		case s.Abort():
			return s
		}
	}

	if fn := ops.ExitPragmaNode; fn != nil {
		if s := fn(node); s != nil && s.Abort() {
			return s
		}
	}
	return nil
}

type ProdNode struct{ ast.Node }

func (c ProdNode) AllIdent() []IdentNode {
	var out []IdentNode
	for _, child := range ast.All(c.Node, "IDENT") {
		out = append(out, IdentNode{child})
	}
	return out
}

func (c ProdNode) OneIdent() IdentNode {
	return IdentNode{ast.First(c.Node, "IDENT")}
}

func (c ProdNode) AllTerm() []TermNode {
	var out []TermNode
	for _, child := range ast.All(c.Node, "term") {
		out = append(out, TermNode{child})
	}
	return out
}

func (c ProdNode) OneTerm() TermNode {
	return TermNode{ast.First(c.Node, "term")}
}
func WalkProdNode(node ProdNode, ops WalkerOps) Stopper {
	if fn := ops.EnterProdNode; fn != nil {
		s := fn(node)
		switch {
		case s == nil:
		case s.ExitNode():
			return nil
		case s.Abort():
			return s
		}
	}

	for _, child := range node.AllIdent() {
		s := WalkIdentNode(child, ops)
		switch {
		case s == nil:
		case s.ExitNode():
			return nil
		case s.Abort():
			return s
		}
	}
	for _, child := range node.AllTerm() {
		s := WalkTermNode(child, ops)
		switch {
		case s == nil:
		case s.ExitNode():
			return nil
		case s.Abort():
			return s
		}
	}
	if fn := ops.ExitProdNode; fn != nil {
		if s := fn(node); s != nil && s.Abort() {
			return s
		}
	}
	return nil
}

type QuantNode struct{ ast.Node }

func (c QuantNode) Choice() int {
	return ast.Choice(c.Node)
}

func (c QuantNode) AllMax() []IntNode {
	var out []IntNode
	for _, child := range ast.All(c.Node, "max") {
		out = append(out, IntNode{child})
	}
	return out
}

func (c QuantNode) OneMax() IntNode {
	return IntNode{ast.First(c.Node, "max")}
}

func (c QuantNode) AllMin() []IntNode {
	var out []IntNode
	for _, child := range ast.All(c.Node, "min") {
		out = append(out, IntNode{child})
	}
	return out
}

func (c QuantNode) OneMin() IntNode {
	return IntNode{ast.First(c.Node, "min")}
}

func (c QuantNode) AllNamed() []NamedNode {
	var out []NamedNode
	for _, child := range ast.All(c.Node, "named") {
		out = append(out, NamedNode{child})
	}
	return out
}

func (c QuantNode) OneNamed() NamedNode {
	return NamedNode{ast.First(c.Node, "named")}
}

func (c QuantNode) AllOp() []string {
	var out []string
	for _, child := range ast.All(c.Node, "op") {
		out = append(out, ast.First(child, "").Scanner().String())
	}
	return out
}

func (c QuantNode) OneOp() string {
	if child := ast.First(c.Node, "op"); child != nil {
		return ast.First(child, "").Scanner().String()
	}
	return ""
}

func (c QuantNode) AllOptLeading() []string {
	var out []string
	for _, child := range ast.All(c.Node, "opt_leading") {
		out = append(out, ast.First(child, "").Scanner().String())
	}
	return out
}

func (c QuantNode) OneOptLeading() string {
	if child := ast.First(c.Node, "opt_leading"); child != nil {
		return ast.First(child, "").Scanner().String()
	}
	return ""
}

func (c QuantNode) AllOptTrailing() []string {
	var out []string
	for _, child := range ast.All(c.Node, "opt_trailing") {
		out = append(out, ast.First(child, "").Scanner().String())
	}
	return out
}

func (c QuantNode) OneOptTrailing() string {
	if child := ast.First(c.Node, "opt_trailing"); child != nil {
		return ast.First(child, "").Scanner().String()
	}
	return ""
}
func WalkQuantNode(node QuantNode, ops WalkerOps) Stopper {
	if fn := ops.EnterQuantNode; fn != nil {
		s := fn(node)
		switch {
		case s == nil:
		case s.ExitNode():
			return nil
		case s.Abort():
			return s
		}
	}

	for _, child := range node.AllNamed() {
		s := WalkNamedNode(child, ops)
		switch {
		case s == nil:
		case s.ExitNode():
			return nil
		case s.Abort():
			return s
		}
	}
	if fn := ops.ExitQuantNode; fn != nil {
		if s := fn(node); s != nil && s.Abort() {
			return s
		}
	}
	return nil
}

type StmtNode struct{ ast.Node }

func (c StmtNode) Choice() int {
	return ast.Choice(c.Node)
}

func (c StmtNode) AllComment() []CommentNode {
	var out []CommentNode
	for _, child := range ast.All(c.Node, "COMMENT") {
		out = append(out, CommentNode{child})
	}
	return out
}

func (c StmtNode) OneComment() CommentNode {
	return CommentNode{ast.First(c.Node, "COMMENT")}
}

func (c StmtNode) AllPragma() []PragmaNode {
	var out []PragmaNode
	for _, child := range ast.All(c.Node, "pragma") {
		out = append(out, PragmaNode{child})
	}
	return out
}

func (c StmtNode) OnePragma() PragmaNode {
	return PragmaNode{ast.First(c.Node, "pragma")}
}

func (c StmtNode) AllProd() []ProdNode {
	var out []ProdNode
	for _, child := range ast.All(c.Node, "prod") {
		out = append(out, ProdNode{child})
	}
	return out
}

func (c StmtNode) OneProd() ProdNode {
	return ProdNode{ast.First(c.Node, "prod")}
}
func WalkStmtNode(node StmtNode, ops WalkerOps) Stopper {
	if fn := ops.EnterStmtNode; fn != nil {
		s := fn(node)
		switch {
		case s == nil:
		case s.ExitNode():
			return nil
		case s.Abort():
			return s
		}
	}

	for _, child := range node.AllComment() {
		s := WalkCommentNode(child, ops)
		switch {
		case s == nil:
		case s.ExitNode():
			return nil
		case s.Abort():
			return s
		}
	}
	for _, child := range node.AllPragma() {
		s := WalkPragmaNode(child, ops)
		switch {
		case s == nil:
		case s.ExitNode():
			return nil
		case s.Abort():
			return s
		}
	}
	for _, child := range node.AllProd() {
		s := WalkProdNode(child, ops)
		switch {
		case s == nil:
		case s.ExitNode():
			return nil
		case s.Abort():
			return s
		}
	}
	if fn := ops.ExitStmtNode; fn != nil {
		if s := fn(node); s != nil && s.Abort() {
			return s
		}
	}
	return nil
}

type TermNode struct{ ast.Node }

func (c TermNode) AllTerm() []TermNode {
	var out []TermNode
	for _, child := range ast.All(c.Node, "term") {
		out = append(out, TermNode{child})
	}
	return out
}

func (c TermNode) OneTerm() TermNode {
	return TermNode{ast.First(c.Node, "term")}
}

func (c TermNode) AllNamed() []NamedNode {
	var out []NamedNode
	for _, child := range ast.All(c.Node, "named") {
		out = append(out, NamedNode{child})
	}
	return out
}

func (c TermNode) OneNamed() NamedNode {
	return NamedNode{ast.First(c.Node, "named")}
}

func (c TermNode) AllOp() []string {
	var out []string
	for _, child := range ast.All(c.Node, "op") {
		out = append(out, ast.First(child, "").Scanner().String())
	}
	return out
}

func (c TermNode) OneOp() string {
	if child := ast.First(c.Node, "op"); child != nil {
		return ast.First(child, "").Scanner().String()
	}
	return ""
}

func (c TermNode) AllQuant() []QuantNode {
	var out []QuantNode
	for _, child := range ast.All(c.Node, "quant") {
		out = append(out, QuantNode{child})
	}
	return out
}

func (c TermNode) OneQuant() QuantNode {
	return QuantNode{ast.First(c.Node, "quant")}
}
func WalkTermNode(node TermNode, ops WalkerOps) Stopper {
	if fn := ops.EnterTermNode; fn != nil {
		s := fn(node)
		switch {
		case s == nil:
		case s.ExitNode():
			return nil
		case s.Abort():
			return s
		}
	}

	for _, child := range node.AllTerm() {
		s := WalkTermNode(child, ops)
		switch {
		case s == nil:
		case s.ExitNode():
			return nil
		case s.Abort():
			return s
		}
	}
	for _, child := range node.AllNamed() {
		s := WalkNamedNode(child, ops)
		switch {
		case s == nil:
		case s.ExitNode():
			return nil
		case s.Abort():
			return s
		}
	}
	for _, child := range node.AllQuant() {
		s := WalkQuantNode(child, ops)
		switch {
		case s == nil:
		case s.ExitNode():
			return nil
		case s.Abort():
			return s
		}
	}
	if fn := ops.ExitTermNode; fn != nil {
		if s := fn(node); s != nil && s.Abort() {
			return s
		}
	}
	return nil
}

type WalkerOps struct {
	EnterWrapreNode  func(WrapreNode) Stopper
	ExitWrapreNode   func(WrapreNode) Stopper
	EnterCommentNode func(CommentNode) Stopper
	ExitCommentNode  func(CommentNode) Stopper
	EnterIdentNode   func(IdentNode) Stopper
	ExitIdentNode    func(IdentNode) Stopper
	EnterIntNode     func(IntNode) Stopper
	ExitIntNode      func(IntNode) Stopper
	EnterReNode      func(ReNode) Stopper
	ExitReNode       func(ReNode) Stopper
	EnterRefNode     func(RefNode) Stopper
	ExitRefNode      func(RefNode) Stopper
	EnterStrNode     func(StrNode) Stopper
	ExitStrNode      func(StrNode) Stopper
	EnterAtomNode    func(AtomNode) Stopper
	ExitAtomNode     func(AtomNode) Stopper
	EnterGrammarNode func(GrammarNode) Stopper
	ExitGrammarNode  func(GrammarNode) Stopper
	EnterNamedNode   func(NamedNode) Stopper
	ExitNamedNode    func(NamedNode) Stopper
	EnterPragmaNode  func(PragmaNode) Stopper
	ExitPragmaNode   func(PragmaNode) Stopper
	EnterProdNode    func(ProdNode) Stopper
	ExitProdNode     func(ProdNode) Stopper
	EnterQuantNode   func(QuantNode) Stopper
	ExitQuantNode    func(QuantNode) Stopper
	EnterStmtNode    func(StmtNode) Stopper
	ExitStmtNode     func(StmtNode) Stopper
	EnterTermNode    func(TermNode) Stopper
	ExitTermNode     func(TermNode) Stopper
}

func (w WalkerOps) Walk(tree GrammarNode)  { WalkGrammarNode(tree, w) }
func (c GrammarNode) GetAstNode() ast.Node { return c.Node }

func NewGrammarNode(from ast.Node) GrammarNode { return GrammarNode{from} }

func Parse(input *parser.Scanner) (GrammarNode, error) {
	p := Grammar()
	tree, err := p.Parse("grammar", input)
	if err != nil {
		return GrammarNode{nil}, err
	}
	return GrammarNode{ast.FromParserNode(p.Grammar(), tree)}, nil
}

func ParseString(input string) (GrammarNode, error) {
	return Parse(parser.NewScanner(input))
}

var grammarGrammarSrc = unfakeBackquote(`
// Non-terminals
grammar -> stmt+;
stmt    -> COMMENT | prod | pragma;
prod    -> IDENT "->" term+ ";";
term    -> @:op=">"
         > @:op="|"
         > @+
         > named quant*;
named   -> (IDENT op="=")? atom;
quant   -> op=[?*+]
         | "{" min=INT? "," max=INT? "}"
         | op=/{<:|:>?} opt_leading=","? named opt_trailing=","?;
atom    -> IDENT | STR | RE | REF | "(" term ")" | "(" ")";

// Terminals
COMMENT -> /{ //.*$
            | (?s: /\* (?: [^*] | \*+[^*/] ) \*/ )
            };
IDENT   -> /{@|[A-Za-z_\.]\w*};
INT     -> \d+;
STR     -> /{ " (?: \\. | [^\\"] )* "
            | ' (?: \\. | [^\\'] )* '
            | ‵ (?: ‵‵  | [^‵]   )* ‵
            };
RE      -> /{
             /{
               (?:
                 \\.
                 | { (?: (?: \d+(?:,\d*)? | ,\d+ ) \} )?
                 | \[ (?: \\. | \[:^?[a-z]+:\] | [^\]] )+ ]
                 | [^\\{\}]
               )*
             \}
           | (?:
               (?:
                 \[ (?: \\. | \[:^?[a-z]+:\] | [^\]] )+ ]
               | \\[pP](?:[a-z]|\{[a-zA-Z_]+\})
               | \\[a-zA-Z]
               | [.^$]
               )(?: (?:[+*?]|\{\d+,?\d?\}) \?? )?
             )+
           };
REF     -> "%" IDENT ("=" default=STR)?;
// Special
pragma  -> (
                import=(".import" path=((".."|"."|[a-zA-Z0-9.:]+):,"/"))
           );

.wrapRE -> /{\s*()\s*};
`)
