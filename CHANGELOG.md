# Changelog

All notable changes to the Vibe language will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/), and this project adheres to [Semantic Versioning](https://semver.org/).

## [0.2.1](https://github.com/vibe-lang/vibe/compare/v0.2.0...v0.2.1) (2026-02-06)


### Features

* add 10 syntax features, install script, and release automation ([ace941d](https://github.com/vibe-lang/vibe/commit/ace941dabfb7494be52d685098fd6a45b67cb4d3))

## [0.2.0] - 2025-02-05

### Added
- **Logical operators**: `&&` (and) and `||` (or) with short-circuit evaluation
- **While loops**: `while condition ... end` for conditional iteration
- **Break/continue**: Loop control flow with `break` and `continue`
- **Hash maps**: `{key: value}` syntax with bracket access and mutation
- **Array mutation**: `arr[i] = value` for in-place element updates
- **Struct mutation**: `obj.field = value` for in-place field updates
- **File imports**: `import "path/to/file.vb"` to include other Vibe files
- **Error handling**: `try ... catch e ... end` with `throw` for exceptions
- **Core builtins**: `len`, `type`, `push`, `pop`, `first`, `last`, `rest`, `map`, `filter`, `each`, `to_s`, `to_i`, `to_f`, `input`, `exit`, `append`, `keys`, `values`, `contains`, `split`, `join`, `replace`, `trim`, `abs`, `min`, `max`, `sort`, `reverse`, `string_length`
- **File I/O builtins**: `read_file`, `write_file`, `file_exists`
- **VS Code syntax highlighting** extension in `editors/vscode/`
- **GitHub Actions CI** pipeline with testing and linting
- **`--version` flag** on the CLI
- MIT License
- CONTRIBUTING.md guide
- This changelog

### Fixed
- Lexer `readChar()` not updating position on EOF (caused truncated tokens at end of input)
- `peekPeekTokenIs` destructively consuming tokens from the lexer
- `parsePrefixExpression`, `parseIfExpression`, `parseClassLiteral` returning nil (stubbed out)
- `evalForLoop` using hardcoded variable names instead of generic propagation
- `evalStructStatement` using hardcoded field name hack
- Parser emitting debug `fmt.Printf` statements to stdout
- REPL running hardcoded expressions instead of reading stdin
- `#` comments emitting a HASH token before skipping
- Single-quote strings not terminating correctly
- Function calls being misidentified as struct constructors
- Compound typed array annotation parsing (`[string, int][]`)
- Range operator precedence too low to parse in for loops

## [0.1.0] - 2025-01-01

### Added
- Initial language implementation
- Lexer with full token support
- Parser producing AST
- Tree-walking interpreter
- Types: integers, floats, booleans, strings, nil, arrays, structs, ranges, functions
- Operators: arithmetic, comparison, prefix (-, !)
- Control flow: if/elsif/else/end
- For loops: `for x in collection ... end`
- Functions: `def name(params) ... end` with closures and recursion
- Structs: definition, instantiation, field access
- String interpolation: `"Hello, ${name}!"`
- Ranges: `1..5` (inclusive), `1...5` (exclusive)
- Type annotations (checked at assignment)
- Comments: `#` and `//`
- REPL and file execution via CLI
- `puts` builtin function
