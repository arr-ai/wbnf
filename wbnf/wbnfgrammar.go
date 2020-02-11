package wbnf

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
                 | \[ (?: \\] | [^\]] )+ ]
                 | [^\\{\}]
               )*
             \}
           | (?: \[ (?: \\. | \[:^?[a-z]+:\] | [^\]] )+ \]
             | \\[pP](?:[a-z]|\{[a-zA-Z_]+\})
             | \\[a-zA-Z]
             | \.
             )(?: (?:[+*?]|\{\d+,?\d?\}) \?? )?
           };
REF     -> "%" IDENT ("=" default=STR)?;

// Special
.wrapRE -> /{\s*()\s*};
`)
