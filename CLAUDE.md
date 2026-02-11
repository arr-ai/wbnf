# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**wbnf** (ωBNF) is a self-hosting parser generator framework for Go. It defines an extended BNF grammar syntax, parses grammar files into an internal representation, validates them, and compiles them into parsers. It can also generate Go code (types and visitor interfaces) from grammars.

The ωBNF grammar syntax is itself defined in ωBNF (`examples/wbnf.wbnf`) and parsed by an auto-generated parser (`wbnf/wbnfgrammar.go`).

## Build & Development Commands

```bash
make build        # Build binary to ./dist/wbnf
make test         # Run all tests (uses gotestsum if available)
make lint         # Run golangci-lint
make coverage     # Run tests + verify coverage ≥ 40%
make all          # test + lint + build + coverage

# Run a single test
go test ./parser/ -run TestParserName
go test ./wbnf/ -run TestGrammar

# Run tests for a single package
go test ./parser/...
go test ./wbnf/...
```

## Architecture

### Package Relationships

```
wbnf/          ← Grammar compilation, validation, recursion detection, cutpoint optimization
  ↓ uses
parser/        ← Core parsing engine: Term types, Parser interface, Scanner, error types
  ↓ produces
ast/           ← High-level AST (Branch/Leaf/Extra) converted from parser output
  ↑ used by
cmd/codegen/   ← Go code generation from parsed grammars (types, walkers, visitors)
```

### Key Abstractions

- **`parser.Term`** — interface implemented by all grammar components (`Rule`, `S`, `RE`, `Seq`, `Oneof`, `Delim`, `Quant`, `Stack`, `Named`, `LookAhead`, `CutPoint`, `ScopedGrammar`, `REF`, `ExtRef`). Each Term produces a `Parser` via its `Parser()` method.
- **`parser.Grammar`** — `map[Rule]Term`, the internal grammar representation. Compiled into `Parsers` via `Grammar.Compile()`.
- **`parser.Parsers`** — the compiled, executable parser. Entry point: `Parsers.Parse(rule, scanner)` or `Parsers.ParseWithExternals()`.
- **`parser.TreeElement`** — variant output type from parsing (either a `Node` with tag/children or a `Scanner` leaf).
- **`ast.Node`** — high-level AST interface (`Branch`, `Leaf`, `Extra`) converted from `TreeElement` via `ast.FromParserNode()`.

### wbnf Package (Grammar Processing Pipeline)

1. `wbnf.MustCompile(grammar)` / `wbnf.Compile()` — main entry point
2. **Validation** (`validation.go`) — checks for duplicate rules, missing rule references
3. **Recursion detection** (`recursion.go`) — detects infinite recursion patterns
4. **Cutpoint insertion** (`cutpoints.go`) — optimization that inserts backtracking cutoff points
5. **Self-hosting** — `wbnf.Grammar()` returns the parser for ωBNF syntax itself; `wbnfgrammar.go` is auto-generated

### Error Hierarchy

- `ParseError` — standard parse failure with position and context
- `FatalError` — unrecoverable error (extends ParseError, includes cutpoint data)
- `StopError` — interface for external errors that bypass ParseError wrapping
- `UnconsumedInputError` — input not fully consumed after a successful parse

## Conventions

- Tests use `testify/assert` and `testify/require` with table-driven patterns and `t.Parallel()`
- Line length limit: 120 characters (enforced by golangci-lint)
- Logging uses `logrus`
- Go version: 1.19 (per go.mod)
- Key dependency: `github.com/arr-ai/frozen` for immutable data structures (sets, maps)
