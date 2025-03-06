# Vibe Programming Language

Vibe is an interpreted programming language similar to Ruby but with type support. It aims to provide a pleasant and ergonomic developer experience while maintaining strong type safety.

## Features

- Ruby-like syntax with type support
- Interpreted execution
- Clear error messages
- Easy to extend

## Getting Started

To build the Vibe interpreter:

```bash
go build -o vibe cmd/vibe/main.go
```

To run a Vibe script:

```bash
./vibe run path/to/script.vb
```

## Syntax Example

```vibe
# Define a function with type annotations
func greet(name: String): String {
  return "Hello, #{name}!"
}

# Variables with type inference
let message = greet("World")
puts(message)

# Variables with explicit types
let count: Int = 5
puts("Count: #{count}")

# Class definition with typed properties
class Person {
  prop name: String
  prop age: Int

  func initialize(name: String, age: Int) {
    @name = name
    @age = age
  }

  func describe(): String {
    return "#{@name} is #{@age} years old"
  }
}

let person = Person.new("Alice", 30)
puts(person.describe())
```

## Project Structure

- `cmd/vibe`: Command-line interface
- `pkg/lexer`: Tokenization of source code
- `pkg/parser`: Parsing tokens into AST
- `pkg/ast`: Abstract Syntax Tree definitions
- `pkg/interpreter`: Runtime interpretation of AST

## Documentation

Detailed documentation is available in the `docs` directory:

- [Vibe Syntax Guide](docs/vibe_syntax.md): Complete language syntax reference
- [Extending Vibe](docs/extending_vibe.md): Guide for developers who want to extend the language

## Development

To contribute to Vibe, you'll need:

1. Go 1.16 or higher
2. Basic understanding of interpreters

After cloning the repository, you can build Vibe using the provided build script:

```bash
./build.sh
```

### Testing

Vibe uses Go's built-in testing framework. To run the tests once:

```bash
make test
```

#### Automated Testing

Vibe includes a file watcher that automatically runs tests when source files change. This provides immediate feedback during development.

To set up the automated testing environment:

1. Install the required dependencies:

```bash
# On macOS
brew install fswatch

# Or use the make target
make install-deps
```

2. Start the file watcher:

```bash
make watch
```

The watcher will run all tests when you save any file in the `pkg/` or `cmd/` directories.

### Development Workflow

1. Make changes to the code
2. Tests run automatically when you save files
3. Fix any failing tests
4. Run `make build` to build the interpreter
5. Test your changes manually with the built interpreter