package wbnf

import (
	"fmt"
	"strings"

	"github.com/arr-ai/wbnf/parser/diff"

	"github.com/arr-ai/wbnf/parser"
)

var (
	GrammarRule = parser.Rule("grammar")
	stmt        = parser.Rule("stmt")
	prod        = parser.Rule("prod")
	term        = parser.Rule("term")
	named       = parser.Rule("named")
	atom        = parser.Rule("atom")
	quant       = parser.Rule("quant")
	ref         = parser.Rule("REF")
	ident       = parser.Rule("IDENT")
	str         = parser.Rule("STR")
	intR        = parser.Rule("INT")
	re          = parser.Rule("RE")
	comment     = parser.Rule("COMMENT")

	// WrapRE is a special rule to indicate a wrapper around all regexps and
	// strings. When supplied in the form "pre()post", then all regexes will be
	// wrapped in "pre(?:" and ")post" and all strings will be escaped using
	// regexp.QuoteMeta then likewise wrapped.
	WrapRE = parser.Rule(".wrapRE")
)

// unfakeBackquote replaces reversed prime with grave accent (backquote) in
// order to make the grammar below more readable.
func unfakeBackquote(s string) string {
	return strings.ReplaceAll(s, "‵", "`")
}

var grammarGrammar = parser.Grammar{
	// Non-terminals
	GrammarRule: parser.Some(stmt),
	stmt:        parser.Oneof{comment, prod},
	prod:        parser.Seq{ident, parser.S("->"), parser.Some(term), parser.S(";")},
	term: parser.Stack{
		parser.Delim{Term: parser.At, Sep: parser.Eq("op", parser.S(">"))},
		parser.Delim{Term: parser.At, Sep: parser.Eq("op", parser.S("|"))},
		parser.Some(parser.At),
		parser.Seq{named, parser.Any(quant)},
	},
	quant: parser.Oneof{
		parser.Eq("op", parser.RE(`[?*+]`)),
		parser.Seq{parser.S("{"), parser.Opt(parser.Eq("min", intR)), parser.S(","), parser.Opt(parser.Eq("max", intR)), parser.S("}")},
		parser.Seq{
			parser.Eq("op", parser.RE(`<:|:>?`)),
			parser.Opt(parser.Eq("opt_leading", parser.S(","))),
			named,
			parser.Opt(parser.Eq("opt_trailing", parser.S(","))),
		},
	},
	named: parser.Seq{parser.Opt(parser.Seq{ident, parser.Eq("op", parser.S("="))}), atom},
	atom:  parser.Oneof{ident, str, re, ref, parser.Seq{parser.S("("), term, parser.S(")")}, parser.Seq{parser.S("("), parser.S(")")}},

	// Terminals
	ident:   parser.RE(`@|[A-Za-z_\.]\w*`),
	str:     parser.RE(unfakeBackquote(`"(?:\\.|[^\\"])*"|'(?:\\.|[^\\'])*'|‵(?:‵‵|[^‵])*‵`)),
	intR:    parser.RE(`\d+`),
	re:      parser.RE(`/{((?:\\.|{(?:(?:\d+(?:,\d*)?|,\d+)\})?|\[(?:\\]|[^\]])+]|[^\\{\}])*)\}`),
	ref:     parser.Seq{parser.S("%"), ident, parser.Opt(parser.Seq{parser.S("="), parser.Eq("default", str)})},
	comment: parser.RE(`//.*$|(?s:/\*(?:[^*]|\*+[^*/])\*/)`),

	// Special
	WrapRE: parser.RE(`\s*()\s*`),
}

func GrammarGrammar() string {
	return grammarGrammarSrc
}

// Build the grammar grammar from grammarGrammarSrc and check that it matches
// grammarGrammar.
var core = func() parser.Parsers {
	parsers := grammarGrammar.Compile(nil)

	r := parser.NewScanner(grammarGrammarSrc)
	v, err := parsers.Parse(GrammarRule, r)
	if err != nil {
		panic(err)
	}
	coreNode := v.(parser.Node)

	newGrammarGrammar := NewFromNode(coreNode)

	if diff := diff.DiffGrammars(grammarGrammar, newGrammarGrammar); !diff.Equal() {
		panic(fmt.Errorf(
			"mismatch between parsed and hand-crafted core grammar"+
				"\nold: %v"+
				"\nnew: %v"+
				"\ndiff: %#v",
			grammarGrammar, newGrammarGrammar, diff,
		))
	}

	return newGrammarGrammar.Compile(&coreNode)
}()

func Core() parser.Parsers {
	return core
}
