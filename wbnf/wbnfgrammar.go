// Code generated by "ωBNF gen" DO NOT EDIT.
// $ wbnf gen --grammar ../examples/wbnf.wbnf --rootrule grammar --pkg wbnf
package wbnf

import (
	"github.com/arr-ai/wbnf/ast"
	"github.com/arr-ai/wbnf/parser"
)

func Grammar() parser.Grammar {
	return parser.Grammar{"grammar": parser.Some(parser.Rule(`stmt`)),
		"stmt": parser.Oneof{parser.Rule(`COMMENT`),
			parser.Rule(`prod`)},
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
		"RE":      parser.RE(`/{((?:\\.|{(?:(?:\d+(?:,\d*)?|,\d+)\})?|\[(?:\\]|[^\]])+]|[^\\{\}])*)\}`),
		"REF": parser.Seq{parser.S("%"),
			parser.Rule(`IDENT`),
			parser.Opt(parser.Seq{parser.S("="),
				parser.Eq("default",
					parser.Rule(`STR`))})},
		".wrapRE": parser.RE(`\s*()\s*`)}
}

type QuantContext struct{ ast.Node }

func (c QuantContext) Choice() int {
	return ast.Choice(c.Node)
}

func (c QuantContext) AllMax() []IDENTContext {
	var out []IDENTContext
	for _, child := range ast.All(c.Node, "max") {
		out = append(out, IDENTContext{child})
	}
	return out
}

func (c QuantContext) OneMax() IDENTContext {
	return IDENTContext{ast.First(c.Node, "max")}
}

func (c QuantContext) AllMin() []IDENTContext {
	var out []IDENTContext
	for _, child := range ast.All(c.Node, "min") {
		out = append(out, IDENTContext{child})
	}
	return out
}

func (c QuantContext) OneMin() IDENTContext {
	return IDENTContext{ast.First(c.Node, "min")}
}

func (c QuantContext) AllNamed() []NamedContext {
	var out []NamedContext
	for _, child := range ast.All(c.Node, "named") {
		out = append(out, NamedContext{child})
	}
	return out
}

func (c QuantContext) OneNamed() NamedContext {
	return NamedContext{ast.First(c.Node, "named")}
}

func (c QuantContext) AllOp() []REContext {
	var out []REContext
	for _, child := range ast.All(c.Node, "op") {
		out = append(out, REContext{child})
	}
	return out
}

func (c QuantContext) OneOp() REContext {
	return REContext{ast.First(c.Node, "op")}
}

func (c QuantContext) AllOpt_leading() []STRContext {
	var out []STRContext
	for _, child := range ast.All(c.Node, "opt_leading") {
		out = append(out, STRContext{child})
	}
	return out
}

func (c QuantContext) OneOpt_leading() STRContext {
	return STRContext{ast.First(c.Node, "opt_leading")}
}

func (c QuantContext) AllOpt_trailing() []STRContext {
	var out []STRContext
	for _, child := range ast.All(c.Node, "opt_trailing") {
		out = append(out, STRContext{child})
	}
	return out
}

func (c QuantContext) OneOpt_trailing() STRContext {
	return STRContext{ast.First(c.Node, "opt_trailing")}
}

type COMMENTContext struct{ ast.Node }

func (c COMMENTContext) String() string {
	if c.Node == nil {
		return ""
	}
	return c.Node.Scanner().String()
}

type IDENTContext struct{ ast.Node }

func (c IDENTContext) String() string {
	if c.Node == nil {
		return ""
	}
	return c.Node.Scanner().String()
}

type StmtContext struct{ ast.Node }

func (c StmtContext) Choice() int {
	return ast.Choice(c.Node)
}

func (c StmtContext) AllCOMMENT() []COMMENTContext {
	var out []COMMENTContext
	for _, child := range ast.All(c.Node, "COMMENT") {
		out = append(out, COMMENTContext{child})
	}
	return out
}

func (c StmtContext) OneCOMMENT() COMMENTContext {
	return COMMENTContext{ast.First(c.Node, "COMMENT")}
}

func (c StmtContext) AllProd() []ProdContext {
	var out []ProdContext
	for _, child := range ast.All(c.Node, "prod") {
		out = append(out, ProdContext{child})
	}
	return out
}

func (c StmtContext) OneProd() ProdContext {
	return ProdContext{ast.First(c.Node, "prod")}
}

type TermContext struct{ ast.Node }

func (c TermContext) AllTerm() []TermContext {
	var out []TermContext
	for _, child := range ast.All(c.Node, "term") {
		out = append(out, TermContext{child})
	}
	return out
}

func (c TermContext) OneTerm() TermContext {
	return TermContext{ast.First(c.Node, "term")}
}

func (c TermContext) AllNamed() []NamedContext {
	var out []NamedContext
	for _, child := range ast.All(c.Node, "named") {
		out = append(out, NamedContext{child})
	}
	return out
}

func (c TermContext) OneNamed() NamedContext {
	return NamedContext{ast.First(c.Node, "named")}
}

func (c TermContext) AllOp() []STRContext {
	var out []STRContext
	for _, child := range ast.All(c.Node, "op") {
		out = append(out, STRContext{child})
	}
	return out
}

func (c TermContext) OneOp() STRContext {
	return STRContext{ast.First(c.Node, "op")}
}

func (c TermContext) AllQuant() []QuantContext {
	var out []QuantContext
	for _, child := range ast.All(c.Node, "quant") {
		out = append(out, QuantContext{child})
	}
	return out
}

func (c TermContext) OneQuant() QuantContext {
	return QuantContext{ast.First(c.Node, "quant")}
}

type AtomContext struct{ ast.Node }

func (c AtomContext) Choice() int {
	return ast.Choice(c.Node)
}

func (c AtomContext) AllIDENT() []IDENTContext {
	var out []IDENTContext
	for _, child := range ast.All(c.Node, "IDENT") {
		out = append(out, IDENTContext{child})
	}
	return out
}

func (c AtomContext) OneIDENT() IDENTContext {
	return IDENTContext{ast.First(c.Node, "IDENT")}
}

func (c AtomContext) AllRE() []REContext {
	var out []REContext
	for _, child := range ast.All(c.Node, "RE") {
		out = append(out, REContext{child})
	}
	return out
}

func (c AtomContext) OneRE() REContext {
	return REContext{ast.First(c.Node, "RE")}
}

func (c AtomContext) AllREF() []REFContext {
	var out []REFContext
	for _, child := range ast.All(c.Node, "REF") {
		out = append(out, REFContext{child})
	}
	return out
}

func (c AtomContext) OneREF() REFContext {
	return REFContext{ast.First(c.Node, "REF")}
}

func (c AtomContext) AllSTR() []STRContext {
	var out []STRContext
	for _, child := range ast.All(c.Node, "STR") {
		out = append(out, STRContext{child})
	}
	return out
}

func (c AtomContext) OneSTR() STRContext {
	return STRContext{ast.First(c.Node, "STR")}
}

func (c AtomContext) AllTerm() []TermContext {
	var out []TermContext
	for _, child := range ast.All(c.Node, "term") {
		out = append(out, TermContext{child})
	}
	return out
}

func (c AtomContext) OneTerm() TermContext {
	return TermContext{ast.First(c.Node, "term")}
}

type REContext struct{ ast.Node }

func (c REContext) String() string {
	if c.Node == nil {
		return ""
	}
	return c.Node.Scanner().String()
}

type NamedContext struct{ ast.Node }

func (c NamedContext) AllIDENT() []IDENTContext {
	var out []IDENTContext
	for _, child := range ast.All(c.Node, "IDENT") {
		out = append(out, IDENTContext{child})
	}
	return out
}

func (c NamedContext) OneIDENT() IDENTContext {
	return IDENTContext{ast.First(c.Node, "IDENT")}
}

func (c NamedContext) AllAtom() []AtomContext {
	var out []AtomContext
	for _, child := range ast.All(c.Node, "atom") {
		out = append(out, AtomContext{child})
	}
	return out
}

func (c NamedContext) OneAtom() AtomContext {
	return AtomContext{ast.First(c.Node, "atom")}
}

func (c NamedContext) AllOp() []STRContext {
	var out []STRContext
	for _, child := range ast.All(c.Node, "op") {
		out = append(out, STRContext{child})
	}
	return out
}

func (c NamedContext) OneOp() STRContext {
	return STRContext{ast.First(c.Node, "op")}
}

type STRContext struct{ ast.Node }

func (c STRContext) String() string {
	if c.Node == nil {
		return ""
	}
	return c.Node.Scanner().String()
}

type WrapREContext struct{ ast.Node }

func (c WrapREContext) String() string {
	if c.Node == nil {
		return ""
	}
	return c.Node.Scanner().String()
}

type GrammarContext struct{ ast.Node }

func (c GrammarContext) AllStmt() []StmtContext {
	var out []StmtContext
	for _, child := range ast.All(c.Node, "stmt") {
		out = append(out, StmtContext{child})
	}
	return out
}

func (c GrammarContext) OneStmt() StmtContext {
	return StmtContext{ast.First(c.Node, "stmt")}
}

type ProdContext struct{ ast.Node }

func (c ProdContext) AllIDENT() []IDENTContext {
	var out []IDENTContext
	for _, child := range ast.All(c.Node, "IDENT") {
		out = append(out, IDENTContext{child})
	}
	return out
}

func (c ProdContext) OneIDENT() IDENTContext {
	return IDENTContext{ast.First(c.Node, "IDENT")}
}

func (c ProdContext) AllTerm() []TermContext {
	var out []TermContext
	for _, child := range ast.All(c.Node, "term") {
		out = append(out, TermContext{child})
	}
	return out
}

func (c ProdContext) OneTerm() TermContext {
	return TermContext{ast.First(c.Node, "term")}
}

type INTContext struct{ ast.Node }

func (c INTContext) String() string {
	if c.Node == nil {
		return ""
	}
	return c.Node.Scanner().String()
}

type REFContext struct{ ast.Node }

func (c REFContext) AllIDENT() []IDENTContext {
	var out []IDENTContext
	for _, child := range ast.All(c.Node, "IDENT") {
		out = append(out, IDENTContext{child})
	}
	return out
}

func (c REFContext) OneIDENT() IDENTContext {
	return IDENTContext{ast.First(c.Node, "IDENT")}
}

func (c REFContext) AllDefault() []IDENTContext {
	var out []IDENTContext
	for _, child := range ast.All(c.Node, "default") {
		out = append(out, IDENTContext{child})
	}
	return out
}

func (c REFContext) OneDefault() IDENTContext {
	return IDENTContext{ast.First(c.Node, "default")}
}

func (c GrammarContext) GetAstNode() *ast.Node { return &c.Node }

func NewGrammarContext(from ast.Node) GrammarContext { return GrammarContext{from} }

func Parse(input *parser.Scanner) (GrammarContext, error) {
	p := Grammar().Compile(nil)
	tree, err := p.Parse("%s", input)
	if err != nil {
		return GrammarContext{nil}, err
	}
	return GrammarContext{ast.FromParserNode(p.Grammar(), tree)}, nil
}

func ParseString(input string) (GrammarContext, error) {
	return Parse(parser.NewScanner(input))
}

var grammarGrammarSrc = unfakeBackquote(`
// Non-terminals
grammar -> stmt+;
stmt    -> COMMENT | prod;
prod    -> IDENT "->" term+ ";";
term    -> @:op=">"
         > @:op="|"
         > @+
         > named quant*;
named   -> (IDENT op="=")? atom;
quant   -> op=/{[?*+]}
         | "{" min=INT? "," max=INT? "}"
         | op=/{<:|:>?} opt_leading=","? named opt_trailing=","?;
atom    -> IDENT | STR | RE | REF | "(" term ")" | "(" ")";

// Terminals
COMMENT -> /{ //.*$
            | (?s: /\* (?: [^*] | \*+[^*/] ) \*/ )
            };
IDENT   -> /{@|[A-Za-z_\.]\w*};
INT     -> /{\d+};
STR     -> /{ " (?: \\. | [^\\"] )* "
            | ' (?: \\. | [^\\'] )* '
            | ‵ (?: ‵‵  | [^‵]   )* ‵
            };
RE      -> /{
             /{
               ((?:
                 \\.
                 | { (?: (?: \d+(?:,\d*)? | ,\d+ ) \} )?
                 | \[ (?: \\] | [^\]] )+ ]
                 | [^\\{\}]
               )*)
             \}
           };
REF     -> "%" IDENT ("=" default=STR)?;

// Special
.wrapRE -> /{\s*()\s*};
`)
