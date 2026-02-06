# Vibe Programming Language

A Ruby-inspired interpreted language with type annotations, built in Go.

```vibe
# Hello World in Vibe
def greet(name: string): string
  "Hello, ${name}!"
end

puts(greet("World"))
```

## Features

- **Everything is an object** — all primitives support method-style calls (`"hello".len`, `[1,2,3].sort`, `42.type`)
- Ruby-like syntax — `def`/`end` blocks, implicit returns, no semicolons
- Type annotations — optional `name: type` on variables, parameters, and return types
- Structs — lightweight data types with named fields and mutation
- Hash maps — `{key: value}` literals with bracket access
- First-class functions — closures, anonymous functions, higher-order functions (`map`, `filter`, `each`)
- Control flow — `if`/`elsif`/`else`, `for`/`in`, `while`, `break`/`continue`
- Logical operators — `&&` and `||` with short-circuit evaluation
- Error handling — `try`/`catch`/`throw`
- String interpolation — `"Hello, ${name}!"` with arbitrary expressions
- Ranges — `1..5` (inclusive) and `1...5` (exclusive)
- Auto type coercion — string + any type, mixed int/float arithmetic
- 35 built-in functions — I/O, collections, math, strings, file operations
- File imports — `import "path/to/file.vb"`
- REPL — interactive mode with history and arrow key navigation

## Quick Start

### Build

```bash
./build.sh
# or
make build
```

### Run a script

```bash
./vibe run examples/hello.vb
```

### Start the REPL

```bash
./vibe repl
```

### Check version

```bash
./vibe --version
```

## Language Overview

### Variables

```vibe
# Type inference
name = "Alice"
age = 30
pi = 3.14159

# With let keyword
let count = 42

# Explicit type annotations
greeting: string = "hello"
active: boolean = true
numbers: int[] = [1, 2, 3]
```

### Functions

```vibe
# Named functions with def/end
def add(a: int, b: int): int
  a + b
end

# Implicit return (last expression)
def multiply(x: int, y: int): int
  x * y
end

# Explicit return
def factorial(n: int): int
  if n <= 1
    return 1
  end
  n * factorial(n - 1)
end

puts(add(5, 3))        # 8
puts(factorial(10))     # 3628800
```

### Anonymous Functions

```vibe
# fn keyword with brace syntax
doubler = fn(x) { x * 2 }
puts(doubler(5))     # 10

# Passing anonymous functions
nums = [1, 2, 3, 4, 5]
result = map(nums, fn(x) { x * x })
puts(result)         # [1, 4, 9, 16, 25]
```

### Method-Style Calls

Every value in Vibe is an object. All built-in functions can be called as methods, and parentheses are optional for zero-argument methods:

```vibe
# String methods
"hello world".split(" ")       # ["hello", "world"]
"hello".len                    # 5
"hello".replace("l", "r")     # "herro"
"  padded  ".trim              # "padded"
"hello".contains("ell")       # true
"hello".type                   # STRING

# Array methods
[3, 1, 2].sort                 # [1, 2, 3]
[1, 2, 3].reverse              # [3, 2, 1]
[1, 2, 3].first                # 1
[1, 2, 3].last                 # 3
[1, 2, 3].len                  # 3
[1, 2, 3].contains(2)          # true

# Works on any type
42.to_s                        # "42"
3.14.to_i                      # 3
true.type                      # BOOLEAN

# Method chaining
"  hello world  ".trim.split(" ").reverse   # ["world", "hello"]

# Higher-order methods
def double(x: int): int
  x * 2
end
[1, 2, 3].map(double)         # [2, 4, 6]

# Hash methods
h = {a: 1, b: 2}
h.keys                         # ["a", "b"]
h.values                       # [1, 2]
```

Parentheses are optional for zero-argument calls: `arr.len` and `arr.len()` are equivalent.

### Structs

```vibe
struct Person
  name: string
  age: int
end

alice = Person(name: "Alice", age: 30)
puts(alice.name)    # Alice
alice.age = 31      # Mutation supported
```

### Control Flow

```vibe
# If / elsif / else
if score >= 90
  grade = "A"
elsif score >= 80
  grade = "B"
else
  grade = "F"
end

# If expressions (return values)
status = if age >= 18
  "adult"
else
  "minor"
end

# For loops with ranges
for i in 1..10
  puts(i)
end

# For loops with arrays
for name in ["Alice", "Bob", "Charlie"]
  puts("Hello, ${name}!")
end

# While loops with break/continue
i = 0
while i < 100
  i = i + 1
  if i % 2 == 0
    continue
  end
  if i > 10
    break
  end
  puts(i)
end
```

### Arrays and Hash Maps

```vibe
# Arrays
numbers = [1, 2, 3, 4, 5]
numbers[0] = 10              # Mutation
squares = []
for n in numbers
  squares = squares + [n * n]
end

# Typed arrays
names: string[] = ["Alice", "Bob"]

# Array concatenation
a = [1, 2] + [3, 4]         # [1, 2, 3, 4]

# Hash maps (bare identifier keys or string keys)
config = {host: "localhost", port: 8080}
settings = {"theme": "dark", "font_size": 14}
puts(config["host"])         # localhost
config["port"] = 9090        # Mutation
```

### Operators

```vibe
# Arithmetic
10 + 5    # 15
10 - 3    # 7
6 * 8     # 48
10 / 3    # 3
15 % 4    # 3

# Comparison
a == b    a != b
a < b     a > b
a <= b    a >= b

# Logical (short-circuit)
true && false    # false
true || false    # true
!true            # false

# Prefix
-5        # negation
!true     # logical NOT

# Mixed int/float (auto-promotes to float)
5 + 3.14         # 8.14
10 / 3.0         # 3.333...

# String + any type (auto-converts to string)
"count: " + 42   # "count: 42"
"pi: " + 3.14    # "pi: 3.14"
```

### Ranges

```vibe
# Inclusive (1 through 5)
for i in 1..5
  puts(i)     # 1, 2, 3, 4, 5
end

# Exclusive (1 through 4)
for i in 1...5
  puts(i)     # 1, 2, 3, 4
end

# Range constructor
r = Range(1, 10)
r = Range(1, 10, true)   # exclusive
```

### Error Handling

```vibe
try
  result = risky_operation()
catch e
  puts("Error: ${e}")
end

throw "something went wrong"
```

### String Interpolation

Supports arbitrary expressions inside `${...}`:

```vibe
name = "World"
puts("Hello, ${name}!")          # Hello, World!
puts("2 + 2 = ${2 + 2}")        # 2 + 2 = 4
puts("Type: ${42.type}")         # Type: INTEGER
puts("Big? ${age > 100}")        # Big? false
```

### Imports

```vibe
import "lib/helpers.vb"
# Functions from helpers.vb are now available
```

### Built-in Functions

| Category | Functions |
|----------|-----------|
| **I/O** | `puts`, `print`, `input` |
| **Type** | `type`, `to_s`, `to_i`, `to_f` |
| **Collections** | `len`, `push`, `pop`, `first`, `last`, `rest`, `append` |
| **Higher-order** | `map`, `filter`, `each` |
| **Ordering** | `sort`, `reverse` |
| **Search** | `contains` (works on arrays, strings, and hashes) |
| **Hash** | `keys`, `values` |
| **String** | `split`, `join`, `replace`, `trim`, `string_length` |
| **Math** | `abs`, `min`, `max` |
| **File** | `read_file`, `write_file`, `file_exists` |
| **System** | `exit` |

All built-in functions can also be called as methods: `arr.sort()` is equivalent to `sort(arr)`.

## Project Structure

```
cmd/vibe/          CLI entry point
pkg/lexer/         Tokenization
pkg/parser/        Parsing tokens into AST
pkg/ast/           Abstract Syntax Tree definitions
pkg/interpreter/   Runtime evaluation
examples/          Example Vibe programs
docs/              Language documentation
editors/vscode/    VS Code syntax highlighting extension
```

## Development

### Prerequisites

- Go 1.16+

### Testing

```bash
# Run all tests
make test

# Run a specific test
go test -v -run=TestMethodStyleCalls ./pkg/interpreter/...

# Watch for changes (requires fswatch)
make install-deps
make watch
```

### CI

GitHub Actions runs tests on every push across Ubuntu and macOS. See `.github/workflows/ci.yml`.

### REPL

The REPL supports full line editing with arrow keys, persistent history across sessions (stored in `~/.vibe_history`), and reverse search with Ctrl+R.

```bash
./vibe repl
```

## License

MIT — see [LICENSE](LICENSE).
