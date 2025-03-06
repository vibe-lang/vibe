# Extending Vibe

This guide explains how to extend the Vibe language interpreter by adding new features, operators, or built-in functions. It's intended for developers who want to customize or enhance the language.

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Adding New Token Types](#adding-new-token-types)
3. [Extending the Lexer](#extending-the-lexer)
4. [Adding New AST Nodes](#adding-new-ast-nodes)
5. [Extending the Parser](#extending-the-parser)
6. [Adding New Object Types](#adding-new-object-types)
7. [Extending the Interpreter](#extending-the-interpreter)
8. [Adding Built-in Functions](#adding-built-in-functions)
9. [Testing Your Extensions](#testing-your-extensions)

## Architecture Overview

Vibe follows a typical interpreter design pattern:

1. **Lexer** (`pkg/lexer`): Converts source code into tokens
2. **Parser** (`pkg/parser`): Transforms tokens into an Abstract Syntax Tree (AST)
3. **AST** (`pkg/ast`): Defines the structure of the syntax tree
4. **Interpreter** (`pkg/interpreter`): Evaluates the AST and executes the program

Each component is designed to be extensible, allowing you to add new features without rewriting the entire system.

## Adding New Token Types

To add a new token type (for example, a new operator or keyword):

1. Add the token type constant in `pkg/lexer/token.go`:

```go
const (
    // ... existing tokens ...

    // New token
    FOR = "FOR" // 'for' loop keyword
)
```

2. If it's a keyword, add it to the `keywords` map:

```go
var keywords = map[string]TokenType{
    // ... existing keywords ...
    "for": FOR,
}
```

## Extending the Lexer

To make the lexer recognize your new tokens:

1. Modify the `NextToken` method in `pkg/lexer/lexer.go` to handle the new token type if it requires special scanning logic.

For example, to add support for a new operator:

```go
func (l *Lexer) NextToken() Token {
    // ... existing code ...

    switch l.ch {
    // ... existing cases ...

    case '@': // Instance variable marker
        tok = Token{Type: AT, Literal: string(l.ch)}

    // ... rest of the function ...
    }
}
```

## Adding New AST Nodes

To add a new language construct (like a 'for' loop), you need to create new AST node types:

1. Define the new AST node struct in `pkg/ast/ast.go`:

```go
// ForLoop represents a for loop statement
type ForLoop struct {
    Token      lexer.Token  // the 'for' token
    Init       Statement    // initialization statement
    Condition  Expression   // loop condition
    Update     Statement    // update statement
    Body       *BlockStatement
}

func (fl *ForLoop) statementNode() {}
func (fl *ForLoop) TokenLiteral() string { return fl.Token.Literal }
func (fl *ForLoop) String() string {
    // Implementation of String method
}
```

## Extending the Parser

To parse your new language construct:

1. Add a new parsing function in `pkg/parser/parser.go`:

```go
func (p *Parser) parseForStatement() *ast.ForLoop {
    loop := &ast.ForLoop{Token: p.curToken}

    // Parse initialization
    // Parse condition
    // Parse update statement
    // Parse loop body

    return loop
}
```

2. Register the parsing function in the appropriate place:

```go
func (p *Parser) parseStatement() ast.Statement {
    switch p.curToken.Type {
    // ... existing cases ...
    case lexer.FOR:
        return p.parseForStatement()
    default:
        return p.parseExpressionStatement()
    }
}
```

## Adding New Object Types

To add a new runtime object type (like an array or hash):

1. Define a new object type constant in `pkg/interpreter/interpreter.go`:

```go
const (
    // ... existing object types ...
    ARRAY_OBJ = "ARRAY"
)
```

2. Create a new struct implementing the `Object` interface:

```go
// Array represents an array of objects
type Array struct {
    Elements []Object
}

func (a *Array) Type() ObjectType { return ARRAY_OBJ }
func (a *Array) Inspect() string {
    // Implementation of Inspect method
}
```

## Extending the Interpreter

To make the interpreter evaluate your new language constructs:

1. Add a new case to the `Eval` method in `pkg/interpreter/interpreter.go`:

```go
func (i *Interpreter) Eval(node ast.Node) Object {
    switch node := node.(type) {
    // ... existing cases ...

    case *ast.ForLoop:
        return i.evalForLoop(node)

    // ... rest of the function ...
    }
}
```

2. Implement the evaluation function:

```go
func (i *Interpreter) evalForLoop(fl *ast.ForLoop) Object {
    // Evaluate initialization
    // Loop while condition is true
    //   Evaluate body
    //   Evaluate update
    // Return result
}
```

## Adding Built-in Functions

To add built-in functions (like `puts` or `len`):

1. Define the built-in function in `pkg/interpreter/builtins.go` (create this file if it doesn't exist):

```go
var builtins = map[string]*Builtin{
    "len": &Builtin{
        Fn: func(args ...Object) Object {
            if len(args) != 1 {
                return newError("wrong number of arguments: got=%d, want=1", len(args))
            }

            switch arg := args[0].(type) {
            case *String:
                return &Integer{Value: int64(len(arg.Value))}
            default:
                return newError("argument to `len` not supported: %s", args[0].Type())
            }
        },
    },
    "puts": &Builtin{
        Fn: func(args ...Object) Object {
            for _, arg := range args {
                fmt.Println(arg.Inspect())
            }
            return &Nil{}
        },
    },
}
```

2. Initialize the built-ins in the `New` function in `pkg/interpreter/interpreter.go`:

```go
func New() *Interpreter {
    env := NewEnvironment()

    // Register built-in functions
    for name, builtin := range builtins {
        env.Set(name, builtin)
    }

    return &Interpreter{env: env}
}
```

## Testing Your Extensions

It's important to write tests for your extensions to ensure they work correctly. Here's how to test each component:

1. **Lexer Tests**: Add test cases to ensure your new tokens are recognized correctly.
2. **Parser Tests**: Add test cases to ensure your new language constructs are parsed into the correct AST nodes.
3. **Interpreter Tests**: Add test cases to ensure your new language features are evaluated correctly.

Example test for a 'for' loop:

```go
func TestForLoop(t *testing.T) {
    input := `
    let sum = 0;
    for (let i = 0; i < 10; i = i + 1) {
        sum = sum + i;
    }
    sum;
    `

    evaluated := testEval(input)
    testIntegerObject(t, evaluated, 45) // 0+1+2+3+4+5+6+7+8+9 = 45
}
```

By following these guidelines, you can extend Vibe with new features while maintaining the language's design philosophy and integrity.