# Contributing to Vibe

Thanks for your interest in contributing to Vibe! This document provides guidelines and information for contributors.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/vibe.git`
3. Create a feature branch: `git checkout -b my-feature`
4. Make your changes
5. Run tests: `make test`
6. Commit and push your changes
7. Open a Pull Request

## Development Setup

```bash
# Build the interpreter
make build

# Run tests
make test

# Watch for changes and auto-test
make install-deps  # installs fswatch
make watch
```

## Project Architecture

Vibe follows the classic interpreter pipeline:

```
Source Code -> Lexer -> Tokens -> Parser -> AST -> Interpreter -> Result
```

| Package | Responsibility |
|---------|---------------|
| `pkg/lexer` | Tokenizes source code into a stream of tokens |
| `pkg/parser` | Parses tokens into an Abstract Syntax Tree (AST) |
| `pkg/ast` | Defines all AST node types |
| `pkg/interpreter` | Evaluates the AST and produces results |
| `cmd/vibe` | CLI entry point (run, repl) |

## Adding a New Feature

When adding a new language feature, you typically need to touch all four packages:

1. **Lexer**: Add any new tokens to `pkg/lexer/token.go` and handle them in `pkg/lexer/lexer.go`
2. **AST**: Define new node types in `pkg/ast/ast.go`
3. **Parser**: Add parsing logic in `pkg/parser/parser.go`
4. **Interpreter**: Add evaluation logic in `pkg/interpreter/interpreter.go`

See `docs/extending_vibe.md` for a detailed walkthrough.

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- CamelCase for exported names, lowerCamelCase for unexported
- Group imports: standard library, third-party, internal
- Use pointer receivers for methods that modify state
- Return errors explicitly; never panic in production code
- Write comprehensive tests for every new feature

## Testing

- **Always** write tests before or alongside new features
- Use table-driven tests with descriptive case names
- Test both happy paths and error conditions
- Tests must verify interpreter behavior independently
- Never modify the interpreter to handle specific test values
- Run the full suite before submitting: `make test`

## Commit Messages

- Use present tense: "add feature" not "added feature"
- First line: concise summary (50 chars or less)
- Separate body with a blank line if needed
- Reference issues with `#123` syntax

## Reporting Issues

- Use GitHub Issues
- Include a minimal reproducing example in Vibe code
- Include the output you got vs. what you expected
- Include your Go version and OS
