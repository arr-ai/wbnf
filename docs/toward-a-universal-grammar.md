# Toward a Universal Grammar: Design Explorations for wBNF

## Abstract

This document explores a set of interconnected ideas for evolving wBNF into a
grammar formalism of maximal generality. Starting from wBNF's existing
scannerless design with scoped whitespace control, we develop several proposed
extensions: unification of regex and grammar notation, integrated tokenization
via labeled alternatives, generalized positional constraints for
indentation-sensitive and layout-sensitive parsing, and algebraic grammar
composition for safe sublanguage embedding. We then show that many classically
"hard" parsing problems (C's typedef ambiguity, C++'s template bracket problem)
dissolve when the grammar is restricted to describing surface syntax and
semantic interpretation is deferred to a separate phase. The combined effect of
these ideas brings wBNF close to what we argue is the theoretical ceiling of
grammar formalism expressiveness: a system capable of describing the syntax of
any language whose syntax can be defined independently of its semantics.

## 1. Background and Motivation

### 1.1 The Sysl Experience

wBNF's design was motivated by the experience of building parsers for
[Sysl](https://github.com/anz-bank/sysl), a system modelling language with
heavily context-dependent syntax. Sysl's original parser (circa 2016) was a
hand-written Python recursive-descent parser built on a `SimpleParser` base
class whose `eat(regex)` method consumed tokens directly from the input string
at the current position. This scannerless approach made context-dependent
tokenization trivial: REST path segments (`/foo/{bar}/baz`), expressions
(`a + b * c`), type specifications, and indentation-sensitive blocks each used
different regex patterns from within their respective parser methods. The parser
author had full control over what constituted a token at every point in the
grammar.

When Sysl was ported to Go (2018) using ANTLR, the fundamental mismatch between
ANTLR's globally-defined lexer token types and Sysl's context-dependent syntax
made development increasingly unwieldy. ANTLR's lexer-mode system could
approximate context-dependent tokenization, but the complexity of managing mode
transitions grew with the language.

wBNF was designed to recover the fluidity of the Python parser's scannerless
approach while providing a declarative grammar specification language and a
compiled parsing engine.

### 1.2 Design Principles

The following principles guided wBNF's design:

1. **Scannerless parsing.** No separate lexer. Terminals (string literals and
   regexes) are first-class grammar elements matched directly against the input.

2. **Context-dependent whitespace.** The `.wrapRE` pragma controls what
   whitespace is injected between terminals. It is scoped: different regions of
   the grammar can have different whitespace rules. This replaces the binary
   lexical/syntactic distinction of systems like SDF and ANTLR with a
   continuously variable parameter.

3. **Self-hosting.** The wBNF grammar syntax is itself defined in wBNF
   (`examples/wbnf.wbnf`) and parsed by an auto-generated parser.

4. **Declarative composition.** Features like precedence (`Stack`), delimited
   repetition (`x:","`) and scoped grammars (`ScopedGrammar`) allow complex
   parsing patterns to be expressed as grammar constructs rather than imperative
   code.

## 2. Existing Mechanisms

Before describing proposed extensions, we summarize the key mechanisms already
present in wBNF that the extensions build upon.

### 2.1 Scoped Grammars

A `ScopedGrammar` introduces a nested grammar with its own rule namespace and
inherited properties. The critical inherited property is `.wrapRE`, which
controls whitespace handling within the scope. For example, a scoped grammar
for parsing REST path segments can set `.wrapRE -> /{()}/` to suppress all
whitespace matching, while the enclosing grammar uses `.wrapRE -> /{\s}/` to
allow whitespace between other tokens.

### 2.2 Back-References (REF)

The `REF` mechanism matches a string previously captured by a named term. This
provides context-sensitive matching power: the parser can require that a later
part of the input equals an earlier part. Applications include matching XML
closing tags to opening tags and heredoc delimiters.

### 2.3 Precedence Stacks

The `Stack` type with `@` self-reference provides a declarative encoding of
operator precedence. A stack like:

```
expr -> @:op1 > @:op2 > @:op3 > atom;
```

desugars into a chain of rules where each level references the next tighter
level, encoding the standard precedence-climbing pattern as a grammar construct.

### 2.4 Delimited Repetition

The notation `x:","` expresses "one or more `x` separated by commas" as a
single grammar construct. This compiles to `x ("," x)*` with
associativity-based tree restructuring.

### 2.5 CutPoints

The CutPoint optimization identifies globally unique string literals in the
grammar and wraps them in committed-choice points. When a CutPoint is matched,
backtracking past it is prevented. This provides a form of error recovery and
improves parse performance for unambiguous tokens.

### 2.6 Lookahead

Positive lookahead (`LookAhead`) asserts that a pattern matches at the current
position without consuming input.

## 3. Proposed Extensions

### 3.1 Regex/Grammar Unification

#### 3.1.1 The Isomorphism

Regular expression syntax and BNF syntax are not merely similar; they are
overlapping notations for the same underlying concepts:

| Concept          | Regex    | BNF      |
|------------------|----------|----------|
| Concatenation    | `ab`     | `a b`    |
| Alternation      | `a\|b`   | `a \| b` |
| Kleene star      | `a*`     | `a*`     |
| Kleene plus      | `a+`     | `a+`     |
| Optional         | `a?`     | `a?`     |
| Grouping         | `(a b)`  | `(a b)`  |
| Character class  | `[a-z]`  | --       |
| Back-reference   | `\1`     | `REF`    |

This convergence is not coincidental. Regular languages are a strict subset of
context-free languages. The notations evolved independently in different
communities (Kleene/Thompson for automata theory, Backus/Naur for programming
language specification) but converged on nearly identical syntax for the shared
constructs.

The divergence occurs only at the boundary where context-free grammars exceed
regular language power:

- **Recursion**: BNF has named rules that can reference themselves; regex has no
  equivalent (PCRE's `(?R)` notwithstanding, as it is effectively a disguised
  CFG).
- **Character classes**: regex provides `[a-z]`, `\d`, `\w` as compact
  character-set notation; traditional BNF does not, but nothing prevents it.
- **Anchors/lookaround**: regex has `^`, `$`, `(?=...)`, `(?!...)`; wBNF
  already has `LookAhead`.

#### 3.1.2 The Proposal

Adopt character-class syntax (`[a-z]`, `\d`, `\w`, etc.) as native wBNF
primitives. These are simply shorthand for large alternations over single
characters. With this addition, wBNF notation *is* regex notation for the
regular fragment of any grammar, and smoothly extends to recursive structures
when needed. A pattern like `[a-z]+ ("," [a-z]+)*` would be simultaneously
valid as a regex-like pattern and as a wBNF grammar expression.

The current `/pattern/` syntax for embedding regexes would no longer serve as a
notational escape hatch (since all regex constructs would be native). However,
it retains a distinct role as a **leaf collapse** directive: the content matched
by `/pattern/` is emitted as a single flat string in the parse tree rather than
as a structured subtree of character matches. This is a projection directive on
the output tree, not a language boundary.

#### 3.1.3 Leaf Collapse as Orthogonal Concern

The distinction between "parse structurally, emit as tree" and "parse
structurally, emit as flat string" is orthogonal to the grammar syntax. Any
subtree could potentially be collapsed to a leaf. The `/pattern/` notation (or
an equivalent annotation) serves this purpose:

- **No collapse** (default): match structurally, emit full AST.
- **Leaf collapse** (`/pattern/` or annotation): match structurally, emit as a
  single opaque string. Internal structure is used for parsing but discarded in
  the output.

This is analogous to ANTLR's distinction between parser rules (which produce
tree nodes) and fragment rules (which participate in matching but don't produce
tokens).

### 3.2 Integrated Tokenization via Labeled Alternatives

#### 3.2.1 Motivation

While wBNF's scannerless design eliminates the need for a separate lexer, many
practical languages benefit from token-level competition: the behavior where
all possible token forms compete at each position and the longest match wins.
This prevents, for example, the keyword `if` from being recognized as the
prefix of the identifier `iff`.

Traditional systems achieve this through a separate lexer pass. wBNF needs a
mechanism that provides the same behavior without decoupling tokenization from
parsing.

#### 3.2.2 The `tok::label` Mechanism

Define a rule whose alternatives are labeled:

```
tok -> (?kw_if) "if"
     | (?kw_while) "while"
     | (?ident) /[a-zA-Z_]\w*/
     | (?number) /\d+(\.\d+)?/
     | (?lparen) "("
     | (?rparen) ")"
     | ...
     ;
```

Other parts of the grammar reference this rule with a label filter:

```
if_stmt -> tok::kw_if tok::lparen expr tok::rparen block;
```

The semantics of `tok::kw_if` are:

1. Evaluate all alternatives of `tok` at the current position.
2. Determine which alternative wins (longest match, with priority-based
   tie-breaking).
3. Succeed only if the winning alternative carries the `kw_if` label; fail
   otherwise.

This provides longest-match token competition — the defining behavior of
traditional lexers — without a separate lexing pass. The `tok` rule is invoked
lazily from within the parser at the point where a token is needed. Different
parts of the grammar can either use `tok::label` (participating in shared token
competition) or bypass `tok` entirely with inline terminals when they need
context-specific tokenization.

#### 3.2.3 Caching

If multiple `tok::X` references are tried at the same position (via
backtracking or prediction), the result of "which alternative wins at position
N" should be computed once and cached. This is analogous to what a DFA-based
lexer does, but here it falls out of standard memoization.

#### 3.2.4 Whitespace Interaction

Whitespace handling (`.wrapRE`) should apply around `tok::X` references
(between tokens) but not within the `tok` rule itself (inside tokens). The
`tok` rule would typically be defined in a scoped grammar with
`.wrapRE -> /{()}/` (no whitespace), while the outer grammar's `.wrapRE`
applies between `tok::X` references.

### 3.3 Generalized Property Constraints

#### 3.3.1 Motivation: The Indentation Problem

Expressing indentation-sensitive syntax (Python, Haskell, YAML) is a
long-standing challenge for parser generators. The traditional solution — a
lexer preprocessor that injects INDENT/DEDENT tokens — is a two-pass
architecture that contradicts wBNF's scannerless philosophy.

The fundamental difficulty is that indentation is *relational*: an INDENT is
not a pattern matchable at a single position, but a relationship between the
whitespace on the current line and the whitespace on a previous line.

#### 3.3.2 REF-Based Approach (Existing Mechanisms)

Indentation blocks can be expressed using existing capture and REF mechanisms:

```
block -> /\n/ indent=/ +/ stmt (/\n/ REF(indent) stmt)*;
```

This captures the whitespace before the first statement, then requires the
exact same whitespace string before each subsequent statement. The block
terminates when `REF(indent)` fails to match. Nested blocks work through
recursion: an inner `block` captures a longer indent string, consumes the more
deeply indented lines, and when it terminates, the outer block's `REF` resumes
matching at the original indentation level.

This approach works with no new primitives. Its limitation is that it requires
exact string equality (the same whitespace characters), which handles
space-only or tab-only indentation but not column-based indentation where
mixed whitespace is allowed.

#### 3.3.3 Generalized Approach: Positional Predicates

Regex already has zero-width position assertions: `^` (start of line), `$`
(end of line), `\b` (word boundary). These are predicates over positional
properties. The natural generalization for wBNF is to extend these to richer,
relational predicates over a vocabulary of parse-position properties.

**Property vocabulary:**

| Property   | Meaning                                         |
|------------|-------------------------------------------------|
| `@col`     | Column position (distance from last newline)    |
| `@line`    | Line number                                     |
| `@offset`  | Absolute byte position in input                 |
| `@len`     | Length of the matched text                       |

**Constraint syntax:**

```
ref=@col term        // capture column at this match site
@col(=ref) term      // assert current column equals captured column
@col(>ref) term      // assert current column is greater
```

**Indentation blocks:**

```
block -> /\n/ level=@col stmt (/\n/ @col(=level) stmt)*;
nested -> @col(>parent_level) block;
```

**Other applications:**

*Aligned tables/records:*
```
record -> first=@col entry (/\n/ @col(=first) entry)*;
```

*Same-line constraint:*
```
return_stmt -> ln=@line "return" @line(=ln) expr;
```

*Fence matching (Markdown code blocks):*
```
fence_open  -> ticks=/`{3,}/;
fence_close -> /`{3,}/ &@len(>=ticks);
```

*Heredoc with start-of-line constraint:*
```
heredoc -> "<<" tag=/\w+/ /\n/ content /\n/ @col(=0) REF(tag);
```

#### 3.3.4 Relationship to Existing Formalisms

Positional predicates extend wBNF's existing assertion mechanisms (REF,
LookAhead) with awareness of the two-dimensional structure of source text.
They are the grammar-level generalization of regex's position assertions, just
as REF is the grammar-level generalization of regex's back-references.

The mechanism does not mention indentation anywhere. Indentation blocks are an
application of column constraints combined with recursive rules. The formalism
is general; the application is specific.

### 3.4 Algebraic Grammar Composition

#### 3.4.1 The Delimiting Problem

When embedding one language within another (SQL in a host language, regex in a
string literal, a DSL in a general-purpose language), the host grammar must
delimit the embedded language's extent. This presents a challenge: curly
braces may not be universally safe as delimiters because the embedded language
may contain unbalanced braces.

Two strategies exist:

1. **Unique sentinels**: the user chooses a delimiter not present in the
   content, as in heredocs (`<<EOF...EOF`) or extensible delimiters
   (`$foo{EOF{...}EOF}`). These are expressible in wBNF via REF.

2. **Subgrammar-defined exit**: the embedded grammar declares its own
   termination condition as a grammar property, analogous to `.wrapRE`.

Both strategies are viable, but both entangle the embedded grammar with its
embedding context: the pure SQL grammar gets polluted with exit conditions and
host-language escape hatches (`${expr}`) that have nothing to do with SQL.

#### 3.4.2 Grammar Composition via Union

The cleaner approach is to treat grammars as composable values. A pure SQL
grammar is written as a standalone, reusable artifact. The embedding context
composes it with adaptation rules:

```
// Pure SQL grammar -- reusable, self-contained
sql {
    stmt -> select_stmt | insert_stmt | ...;
    expr -> column | literal | func_call | "(" expr ")";
    ...
}

// Embedding adapter -- specific to the host language
sql_in_host = sql + {
    .exit   -> "}";
    .wrapRE -> /{\s}/;
    expr    |= "${" host_expr "}";    // extend with escape hatch
};

// Usage in host grammar
embedded_sql -> "$sql{" sql_in_host "}";
```

The `+` operator produces a new grammar by combining rule sets. The `|=`
operator extends an existing rule with additional alternatives. The original
SQL grammar is not modified.

#### 3.4.3 Composition Operations

The minimal algebra of grammar composition:

| Operation                          | Meaning                                      |
|------------------------------------|----------------------------------------------|
| **Union** (`A + B`)                | Combine rule sets from two grammars          |
| **Extension** (`rule \|= alt`)     | Add alternatives to an existing rule         |
| **Override** (`rule = ...`)        | Replace a rule entirely in the derived grammar |
| **Property overlay** (`.wrapRE`, `.exit`) | Set or change inherited properties    |

This is sufficient to express the common embedding patterns:

- **String interpolation**: extend the string grammar's content rule with an
  escape-to-host alternative.
- **Embedded DSLs**: compose the DSL grammar with an adapter adding exit
  conditions and host escapes.
- **Language layering**: extend a base language grammar with domain-specific
  constructs.

#### 3.4.4 Separation of Concerns

Grammar composition ensures that:

- The embedded grammar is **reusable** across different host languages.
- The embedded grammar is **not polluted** with host-specific constructs.
- The **delimiting strategy** is chosen by the host grammar at the composition
  site, not by the embedded grammar.
- Grammars can be **tested and validated independently**.

This reframes `ScopedGrammar` from "a grammar can contain inline sub-grammars"
to "grammars are first-class values that can be composed." The inline form is a
special case: an anonymous grammar composed at the point of use.

## 4. Dissolving "Hard" Parsing Problems

Several problems traditionally considered to require semantic feedback during
parsing dissolve when a principled separation between syntax and semantics is
maintained.

### 4.1 Principle: The Grammar Describes Syntax, Not Semantics

A grammar should produce a parse tree that faithfully represents the surface
syntax of the input: what tokens appeared, in what order, with what nesting as
determined by the grammar's productions. Semantic interpretation of this tree
-- determining what the syntax *means* -- is a separate phase.

Many "hard" parsing problems arise from grammars that conflate syntactic and
semantic categories, forcing the parser to make semantic decisions it lacks the
information to make.

### 4.2 The C Typedef Problem

The statement `T * x;` in C is traditionally considered ambiguous: it is a
pointer variable declaration if `T` is a typedef name, or a multiplication
expression if `T` is a variable. This is cited as evidence that C requires
semantic feedback (a symbol table) during parsing.

However, the ambiguity exists only because the grammar encodes the semantic
distinction between declarations and expressions as separate syntactic
categories. At the surface level, `T * x ;` is an unambiguous sequence:
`identifier STAR identifier SEMI`. A grammar that captures this surface
structure without attempting to classify it produces a single, unambiguous
parse tree. The semantic phase, armed with the symbol table, subsequently
determines whether the tree represents a declaration or an expression.

The more complex case `T * x, y;` demonstrates that the two interpretations
yield different tree structures (the comma groups differently in declarations
vs. expressions). But even here, a syntactically unified production can capture
the flat token sequence, and the semantic phase can impose the correct
grouping. This is the standard approach when producing a concrete syntax tree
(CST) and deriving the abstract syntax tree (AST) in a subsequent phase.

The typedef problem is not a parsing problem. It is a semantic analysis problem
that was misclassified as a parsing problem because traditional compiler
architectures produce ASTs directly from the parser.

### 4.3 The C++ Template Bracket Problem

The string `>>` in C++ can be the right-shift operator or two closing template
brackets. C++11 resolved this by having the parser split `>>` tokens inside
template contexts -- a hack that leaks semantic knowledge into the
lexer/parser boundary.

An alternative treatment: the grammar always matches `>` as a single character.
Template closing brackets are individual `>` tokens. The shift operator is
defined as two `>` tokens within a context that suppresses whitespace between
them:

```
shift_op -> { .wrapRE -> /{()}/; ">" ">"; };
```

The `.wrapRE` suppression ensures that `> >` with whitespace does not match as
a shift operator, while `>>` does. Template closing brackets match in the
normal grammar where `.wrapRE` allows whitespace. No conflict arises.

This treatment requires no special cases. The grammar describes the surface
syntax (individual `>` characters), and the scoped whitespace mechanism
(already present in wBNF) distinguishes the operator case from the delimiter
case. The semantic question "is this a shift or two template closings?" is
answered by context, not by the lexer.

This approach extends naturally to `>>>` (Java's unsigned right-shift), `>>=`
(compound assignment), and similar multi-character operators.

### 4.4 The General Pattern

Both cases share a common structure:

1. A traditional grammar encodes semantic distinctions in its syntactic
   categories (declaration vs. expression; shift-operator vs. template-bracket).
2. The input is syntactically ambiguous only with respect to these semantic
   categories, not with respect to the surface token sequence.
3. Resolving the ambiguity requires semantic information not available during
   parsing.
4. The traditional solution is to inject semantic information into the parser
   (symbol tables, lexer modes, token-splitting hacks).
5. The principled solution is to not encode the semantic distinction in the
   grammar in the first place. Parse the surface syntax; classify later.

This pattern suggests that the class of languages traditionally considered to
"require semantic feedback for parsing" is smaller than believed. In many
cases, what is required is semantic feedback for *interpretation*, not for
parsing, and the conflation of the two reflects an architectural decision (AST
production during parsing) rather than a theoretical necessity.

## 5. Toward Universality

### 5.1 Formal Expressiveness

With the proposed extensions, wBNF's recognizable language class would include:

- All regular languages (via regex subsumption)
- All context-free languages (given a suitable parsing algorithm such as
  ALL(\*) or GLR)
- Some context-sensitive languages (via REF for copy languages, `@len`
  constraints for counting languages)

The formal ceiling is in the *mildly context-sensitive* range, overlapping with
indexed grammars and tree-adjoining grammars. This is deliberately below full
context-sensitivity (linear-bounded automata) and Turing-completeness, which
would sacrifice decidability and static analyzability.

### 5.2 Practical Coverage

The combination of existing mechanisms and proposed extensions covers the
following challenging parsing patterns:

| Pattern                         | Mechanism                                    |
|---------------------------------|----------------------------------------------|
| Indentation sensitivity         | `@col` constraints                           |
| Context-dependent tokenization  | `.wrapRE` + scoped grammars                  |
| Semicolon insertion             | `@line` constraints                          |
| Context-sensitive keywords      | Scoped grammars                              |
| Heredocs                        | REF + `@col` constraints                     |
| String interpolation            | Grammar composition + scoped grammars        |
| Fence matching                  | REF + `@len` constraints                     |
| Embedded DSLs                   | Grammar composition                          |
| Operator precedence             | Stack type                                   |
| Longest-match tokenization      | `tok::label`                                 |
| XML/HTML tag matching           | REF                                          |
| C typedef ambiguity             | Surface-syntax grammar + semantic phase      |
| C++ template brackets           | Individual `>` tokens + `.wrapRE` scoping    |

### 5.3 Remaining Boundaries

The cases that remain genuinely outside the reach of any static grammar
formalism:

1. **Truly unbounded grammar self-modification**: languages where parsed
   content alters the grammar in Turing-complete ways (e.g., unrestricted
   Lisp macros). Note that even here, the base syntax (S-expressions) is
   trivially parseable; the "grammar change" occurs at the semantic level
   (macro expansion).

2. **Unmarked grammar transitions**: languages where the shift between
   sublanguages has no syntactic delimiter, requiring execution of arbitrary
   code to determine which grammar applies.

Notably, most practical instances of "extensible syntax" (string interpolation,
embedded DSLs, macro invocations) do have syntactic delimiters and are
therefore expressible through grammar composition (Section 3.4).

### 5.4 The Universality Claim

We argue that wBNF with the proposed extensions approaches the theoretical
ceiling of what a grammar formalism can express without becoming a programming
language. The remaining unreachable cases are languages whose syntax cannot be
defined independently of their semantics or whose grammar is subject to
unbounded runtime modification. These are arguably not grammar problems but
metaprogramming problems, and their intractability reflects a fundamental
boundary between syntax and computation, not a limitation of any particular
formalism.

## 6. Related Work

### 6.1 Scannerless Parsing

The **Syntax Definition Formalism** (SDF, SDF2, SDF3) developed at CWI and TU
Delft (Visser, 1997; Klint, 2003) is the primary precedent for scannerless
parsing in a declarative grammar formalism. SDF defines lexical and
context-free syntax in the same notation, parsed by a single generalized (GLR)
engine. Its successor **Rascal** extends the formalism with character classes
and regex operators in grammar productions. Both distinguish `lexical` and
`syntax`/`context-free` declaration types that control whether layout is
inserted between symbols.

wBNF's `.wrapRE` differs from SDF/Rascal's approach by making whitespace
handling a continuously variable, scopeable property rather than a binary
classification tied to declaration type.

### 6.2 Layout-Sensitive Parsing

**Adams (2013)**, *"Principled Parsing for Indentation-Sensitive Languages"*,
extends CFGs with layout constraints (aligned, indented, offside) checked
against source positions. **Erdweg et al. (2013)**, *"Layout-Sensitive
Generalized Parsing"*, integrates layout constraints into SDF/SGLR.

The positional constraints proposed in Section 3.3 generalize Adams' approach
from indentation-specific annotations to a general vocabulary of positional
properties with relational operators.

### 6.3 Data-Dependent Grammars

**Jim, Mandelbaum, and Walker (2010)**, *"Semantics and Algorithms for
Data-Dependent Grammars"*, formalize grammars where productions can bind
values and use them as constraints in subsequent productions. Their **Yakker**
parser generator implements this formalism.

Data-dependent grammars provide the formal foundation for much of what wBNF's
REF mechanism and the proposed property constraints achieve. The key
difference: data-dependent grammars allow arbitrary predicates
(Turing-complete), while wBNF restricts constraints to structural comparisons
(`=`, `>`, `<`) over a fixed property vocabulary, trading expressiveness for
analyzability.

### 6.4 PEG-Based Unification

**OMeta** (Warth, 2007) and its successor **Ohm** unify lexical and syntactic
parsing in a PEG framework. Ohm distinguishes "syntactic rules"
(whitespace-skipping) from "lexical rules" (character-exact) by naming
convention. **LPEG** (Ierusalimschy, 2009) provides a parsing library based on
parsing expressions that operates at the character level and is used as a regex
replacement with grammar capabilities.

### 6.5 Grammar Modularity

SDF2/SDF3 provides a module system with import, renaming, and extension
operations. Rascal has an `extend` keyword for grammar extension. These
operate at the module level. The grammar composition proposed in Section 3.4
treats grammars as values combined through algebraic operators, enabling
composition at arbitrary points in the grammar.

### 6.6 Attribute Grammars

**Knuth (1968)**, *"Semantics of Context-Free Languages"*, introduced
attribute grammars with inherited and synthesized attributes. wBNF's `.wrapRE`
is an inherited attribute in all but name. The proposed `.exit` property and
positional constraints extend the inherited attribute mechanism. The grammar
composition operations (Section 3.4) can be understood as algebra over
attributed grammars.

## 7. Future Work

### 7.1 Parsing Algorithm

The current wBNF implementation uses a backtracking recursive-descent parser
with PEG-style ordered choice. The design intent is CFG semantics with
ANTLR-style ALL(\*) prediction or a similar algorithm that provides:

- Unordered alternation (true CFG semantics)
- Lookahead DFA construction for prediction
- Linear-time parsing for most practical grammars
- Ambiguity detection

The grammar language and compilation pipeline are already rich enough to
support this; the migration path is primarily in the parsing engine.

### 7.2 Formal Semantics

A formal semantics for the extended wBNF, building on the data-dependent
grammar framework of Jim et al., would clarify the expressiveness boundaries
and enable formal reasoning about grammar equivalence and composition.

### 7.3 Implementation of Proposed Extensions

The extensions proposed in this document are ordered by implementation
complexity:

1. **Regex syntax integration** (character classes): primarily notation; low
   complexity.
2. **`tok::label`**: requires filtered matching on named alternatives; moderate
   complexity.
3. **Positional constraints** (`@col`, `@line`, `@len`, `@offset`): requires
   exposing scanner position properties to the grammar; moderate complexity.
4. **Grammar composition** (`+`, `|=`, override): requires grammar-level
   operations and a module/import system; higher complexity.

## 8. Conclusion

wBNF's existing design -- scannerless parsing with scoped whitespace control,
back-references, precedence stacks, and scoped grammars -- already occupies an
unusual and underexplored region of the grammar formalism design space. The
proposed extensions (regex/grammar unification, labeled-alternative
tokenization, generalized positional constraints, and algebraic grammar
composition) are individually modest additions but collectively bring the
formalism close to what we argue is the theoretical ceiling of grammar
expressiveness: the ability to describe the syntax of any language whose syntax
is definable independently of its semantics.

The name omega-BNF aspires to finality. While no formalism can be truly final
in a domain as rich as formal languages, the gap between the aspiration and the
reality is, upon careful analysis, smaller than one might expect. The remaining
unreachable territory is not a grammar problem but a metaprogramming problem,
and its intractability reflects a fundamental boundary between syntax and
computation rather than a limitation of any particular formalism.
