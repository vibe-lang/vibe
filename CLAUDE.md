# Vibe Language Development Guide

## Interpreter Development
- Build: `./build.sh` or `make build`
- Run tests: `make test` or `go test -v -p=1 ./pkg/... ./cmd/...`
- Run single test: `go test -v -run=TestName ./pkg/package/...`
- Watch for changes: `make watch` (requires fswatch: `make install-deps`)
- **Code Style**: 
  - Follow Go conventions: CamelCase for public, lowerCamelCase for private
  - Group imports: standard lib, third-party, internal
  - Use pointer receivers for methods modifying state
  - Return errors explicitly, never panic in production code
  - Always write comprehensive tests for new features

## Vibe Language Syntax
- Execute programs: `./vibe examples/file.vb`
- Run example: `./vibe examples/hello.vb`
- **Syntax Guidelines**:
  - No semicolons required
  - Functions defined with `fn` keyword
  - Variables declared with `let` keyword
  - Supported types: numbers, strings, arrays, functions
  - Array indices are zero-based
  - String interpolation with `${expression}`

## Testing Principles
- Tests must verify interpreter behavior independently
- Never modify interpreter to handle specific test values
- Write tests before implementing new features
- Test both valid syntax and error conditions
- Use table-driven tests with descriptive case names
- Each test should have clear assertions and expectations