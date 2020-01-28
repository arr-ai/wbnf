# ωBNF &mdash; super awesome parser engine

[![GitHub Actions Go status](https://github.com/arr-ai/wbnf/workflows/Go/badge.svg)](.)

## Grammar Syntax Guide

ωBNF is self describing!

<!-- INJECT: ```text\n${examples/wbnf.wbnf}\n``` -->
```text
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
            | ` (?: ``  | [^`]   )* `
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
REF     -> "\\" IDENT;

// Special
.wrapRE -> /{\s*()\s*};
```
<!-- /INJECT -->

## The basics

A ωBNF grammar file consists of an unordered list of rules (called
*productions*) or *comments*.

### Comments

A Comment can be either C++-style `// This is a comment to the end of the line`
or C-style `/* This is a comment which may span multiple lines */`

### Rules/Productions

A rule is defined in terms of *terms*, or *terminals* in the form of `NAME ->
TERM+ ;`:

- `a -> b;` indicates a *rule* named `a` which is made up of single *term* (in
  this case the *rule* `b`)
- `a -> b (c d)*;` indicates a *rule* named `a` which is made up of several
    *terms* (in this case the *rule* `b` followed by the sequence of terms `c`
    and `d`, which may occur zero or more times in succession)
- `a -> "Token";` indicates a *rule* named `a` which is made of a single
  *terminal* (in this case the string `Token`)
- `a -> /{\d+};` indicates a *rule* named `a` which is made of a single
  *terminal* (in this case the regex `\d+`)

### Terminals

- **Strings** are quoted text which match exactly the same sequence in the input
  text. They may be quoted by `"` or `'` or ` (backquote)
- **Regular Expressions** in the form `/{RE}`, where RE is the expression to
  match. The entire match will be consumed. The parser will use the first
  capturing group to populate the output node, or the entire match if none if
  there isn't one.

### Expressions

*Terms* can be grouped in various ways to build up *rules*.

- Sequence

  `a -> left /{[+-*/]} right;`

  This rule requires a `left` followed by one of the math symbols followed by
  a `right`.

- Choice

  `a -> ("hello" | "goodbye") name;`

  This rule requires either the string `hello` or `goodbye` followed by a
  `name`.

- Simple Multiplicity

  `+`, `?`, and `*` may follow any *term* to indicate how many occurrences of
  the term will be matched.

  - `+` indicates 1 or more occurrences.
  - `?` indicates 0 or 1 occurrence.
  - `*` indicates 0 or more occurrences.

  `a -> b* ("x" | "y")+;` matches any amount of `b` followed by at least one of
  either `"x"` or `"y"`

- Delimited repetition

  One of the design goals of the grammar is to minimise the amount of repetition
  in each *term*. If we wanted to create a rule to accept a comma separated list
  of words the simple version would be:

  ```text
  word -> /{\w*};
  csv -> word ("," word)*;
  ```

  The *rule* `word` appears 3 times in that tiny snippet! This can be eliminated
  with the use of the `:` operator after a term: `csv -> /{\w*}:",";` expresses
  the same *rule*. (More on this operator below)

- Min/Max repetition

  `a -> "x"{3,9}` indicates that a string of at least 3 `x` up to 9 will be
  accepted.

  - If the first number is missing, `0` will be the assumed minimum.
  - If the 2nd number is missing, `unlimited` with be the assumed maximum

- Precedence Stacks

  Languages often require some way to define the order of operations (remember
  BODMAS from school?).

  The simple form of a math parser would include something like:

  ```text
  expr    -> braces+;
  braces  -> ("(" multdiv+ ")") | multdiv;
  multdiv -> addsum (("*" | "/") addsum)*;
  addsum  -> number (("+" | "-") number)*;
  number  -> /{\d+};
  ```

  This again has heaps of repetition both in each *rule* and between *rules* (as
  each refers to the next in the precedence order).

  The `>` operator can help with this (newlines are generally ignored by the
  parser).

  ```text
  expr -> @:/{[+-]}
        > @:/{[*/]}
        > "(" @ ")"
        > /{\d+};
  ```

  In each line of the stack, the @ *term* implicitly refers to the next line
  down. *Terms* further along have a higher precedence than earlier *terms*.

  The above grammar will parse the expression `1 + 2 * 3` as the following
  nodes:

  ```text
   "+"
  ┌─┴─┐
  1  "*"
    ┌─┴─┐
    2   3
  ```

  giving the result 7.

- Named Terms

  *Terms* in a *rule* may be named as a convenience item.

  ```text
  expr -> @:op=/{[+-]}
        > @:op=/{[*/]}
        > "(" @ ")"
        > /{\d+};
  ```

  This is the same math grammar as above, except two lines have `op=` for the
  *delimiter* term name.

### Further Details

#### Delimited Repeater

This is the definition of the delimited repeater `op=/{<:|:>?} opt_leading=","?
named opt_trailing=","?`.

- `op` describes the associativity of the separated terms. All forms of the term
  `a op b` will match sequences of the form `a b a ... b a`.

  - `:` denotes a non-associative delimiter. The term `a:b` will produce trees
    conceptually like the following diagram. The parser will emit a single node
    with all terms and delimiters in it:

    ```text
      b       b
    ┌─┴─┬ ─ ─ ┴─┐
    a   a       a
    ```

  - `:>` denotes a left-to-right associative delimiter. This will produce a
    chain of binary nodes. The term `a<:b` will produce trees looking like this:

    ```text
          b
        ┌─┴─┐
        b   a
      ┌─┴─┐
      b   a
    ┌─┴─┐
    a   a
    ```

  - `<:` denotes a right-to-left associative delimiter. The term `a<:b` will
    produce trees looking like this:

    ```text
      b
    ┌─┴─┐
    a   b
      ┌─┴─┐
      a   b
        ┌─┴─┐
        a   a
    ```

- `opt_leading` and `opt_trailing` are optional markers used to allow the
  separator to start and/or end the sequence.

  - If `opt_leading` is present, the sequence is allowed to start with the
    separater term.
  - If `opt_trailing` is present, the sequence is allowed to end with the
    separater term.
  - Both are allowed together also.

  Examples:

  - `x -> a:b,` - Allows `ab...aba` or `ab...ab` (where `...` represents any
    amount of `ab`)
  - `x -> a:,b` - Allows `ab...aba` or `bab..aba` (where `...` represents any
    amount of `ab`)

#### Magic rules

*Rules* prefixed by a `.` are special rules governing the parser's overall
behaviour. The following rules are recognised:

##### `.wrapRE -> /{some () regex}`

This rule instructs the parser to wrap every regular expression with this one.
The actual regex is insertd into the `()`.

Example:

- `.wrapRE -> /{\s*()\s*};` ignore all whitespace surrounded every token in the
  grammar.
