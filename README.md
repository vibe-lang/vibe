# Vibe Programming Language

A Ruby-inspired interpreted language with TypeScript-style type annotations, built in Go.

```vibe
# Hello World in Vibe
def greet(name: string): string
  "Hello, ${name}!"
end

puts(greet("World"))
```

## Installation

### One-line install (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/vibe-lang/vibe/main/install.sh | sh
```

This detects your platform (Linux/macOS, x86_64/arm64), downloads the latest release, verifies the SHA-256 checksum, and installs to `~/.vibe/bin`.

### Options

Pin a specific version:

```bash
VIBE_VERSION=0.2.0 curl -fsSL https://raw.githubusercontent.com/vibe-lang/vibe/main/install.sh | sh
```

Custom install directory:

```bash
VIBE_INSTALL=/opt/vibe curl -fsSL https://raw.githubusercontent.com/vibe-lang/vibe/main/install.sh | sh
```

### Build from source

Requires Go 1.24+.

```bash
git clone https://github.com/vibe-lang/vibe.git
cd vibe
./build.sh
```

### Verify

```bash
vibe --version
```

## Features

- **Everything is an object** -- all primitives support method-style calls (`"hello".len`, `[1,2,3].sort`, `42.type`)
- **Ruby-like syntax** -- `def`/`end` blocks, implicit returns, no semicolons
- **Type annotations** -- optional `name: type` on variables, parameters, and return types
- **Union and optional types** -- `string | nil`, `int?`
- **Generics** -- `def identity<T>(x: T): T`, `class Box<T>`, `struct Pair<A, B>`
- **Classes with inheritance** -- `class Dog < Animal` with `initialize`, `self`, `super`
- **Structs** -- lightweight data types with named fields and mutation
- **Enums** -- `enum Color Red Green Blue end`
- **Hash maps** -- `{key: value}` literals with bracket and dot access
- **First-class functions** -- closures, anonymous functions, arrow functions (`-> (x) { x * 2 }`)
- **Higher-order functions** -- `map`, `filter`, `reduce`, `each`, `sort_by`, and 30+ more
- **Control flow** -- `if`/`elsif`/`else`, `unless`, `case`/`when`, `for`/`in`, `while`, `until`, `break`/`continue`
- **Postfix conditionals** -- `return "no" if x < 0`, `y = "ok" unless invalid`
- **Ternary operator** -- `x > 0 ? "positive" : "non-positive"`
- **Nil coalescing** -- `value ?? default`
- **Pipe operator** -- `data |> transform |> format`
- **Destructuring** -- `a, b, c = [1, 2, 3]`
- **Compound assignment** -- `+=`, `-=`, `*=`, `/=`, `%=`
- **Power operator** -- `2 ** 10` (1024)
- **String repetition** -- `"ha" * 3` ("hahaha")
- **String indexing** -- `"hello"[0]` ("h"), `"hello"[-1]` ("o")
- **String comparison** -- `"apple" < "banana"` (lexicographic)
- **Multi-line strings** -- `"""..."""` with preserved newlines
- **`in` operator** -- `"x" in array`, `key in hash`, `char in string`, `5 in 1..10`
- **Variadic functions** -- `def log(level, *messages)`
- **Constants** -- `const PI = 3.14159`
- **Default parameters** -- `def greet(name = "World")`
- **Negative array indexing** -- `arr[-1]` for last element
- **Error handling** -- `try`/`catch`/`finally`/`throw`
- **String interpolation** -- `"Hello, ${name}!"` with arbitrary expressions
- **Ranges** -- `1..5` (inclusive) and `1...5` (exclusive)
- **Auto type coercion** -- string + any type, mixed int/float arithmetic
- **Standard library** -- `import "math"`, `import "regex"`, `import "concurrent"`
- **60+ built-in functions** -- I/O, collections, math, strings, file operations, JSON
- **File imports** -- `import "path/to/file.vb"`
- **REPL** -- interactive mode with history and arrow key navigation

## Quick Start

```bash
# Run a script
vibe run examples/hello.vb

# Start the REPL
vibe repl
```

## Language Overview

### Variables and Constants

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

# Constants (cannot be reassigned)
const MAX_SIZE = 100
const PI: float = 3.14159
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

# Default parameters
def greet(name = "World")
  "Hello, ${name}!"
end

greet()         # "Hello, World!"
greet("Alice")  # "Hello, Alice!"

# Variadic functions
def sum_all(*nums)
  total = 0
  for n in nums
    total += n
  end
  total
end

sum_all(1, 2, 3, 4, 5)  # 15

# Variadic with required params
def log(level, *messages)
  for msg in messages
    puts("[${level}] ${msg}")
  end
end
```

### Anonymous and Arrow Functions

```vibe
# fn keyword with brace syntax
doubler = fn(x) { x * 2 }
puts(doubler(5))     # 10

# Arrow functions
square = -> (x) { x * x }
puts(square(4))      # 16

# Passing to higher-order functions
nums = [1, 2, 3, 4, 5]
result = map(nums, -> (x) { x * x })
puts(result)         # [1, 4, 9, 16, 25]
```

### Pipe Operator

```vibe
# Chain transformations with |>
result = [1, 2, 3, 4, 5] |> reverse |> first
puts(result)  # 5

# With arguments
"hello world" |> replace("world", "vibe") |> upcase
```

### Destructuring

```vibe
# Array destructuring
a, b, c = [1, 2, 3]
puts(a)  # 1
puts(b)  # 2
puts(c)  # 3

# Hash destructuring
name, age = {"name": "Alice", "age": 30}
```

### Method-Style Calls

Every value in Vibe is an object. All built-in functions can be called as methods:

```vibe
# String methods
"hello world".split(" ")       # ["hello", "world"]
"hello".len                    # 5
"hello".upcase                 # "HELLO"
"HELLO".downcase               # "hello"
"hello".capitalize             # "Hello"
"hello".starts_with("he")     # true
"hello".ends_with("lo")       # true
"hello".repeat(3)             # "hellohellohello"
"hello".chars                  # ["h", "e", "l", "l", "o"]
"hello".string_reverse         # "olleh"
"hello"[0]                     # "h"
"hello"[-1]                    # "o"

# Array methods
[3, 1, 2].sort                 # [1, 2, 3]
[1, 2, 3].reverse              # [3, 2, 1]
[1, 2, 3].first                # 1
[1, 2, 3].last                 # 3

# Method chaining
"  hello world  ".trim.split(" ").reverse   # ["world", "hello"]
```

### Classes

```vibe
class Animal
  def initialize(name)
    self.name = name
  end

  def speak()
    "..."
  end
end

class Dog < Animal
  def speak()
    "Woof! I'm ${self.name}"
  end
end

class Puppy < Dog
  def speak()
    super.speak() + " (tiny bark)"
  end
end

fido = Dog("Fido")
puts(fido.speak())   # "Woof! I'm Fido"
```

### Generics

```vibe
# Generic functions
def identity<T>(x: T): T
  x
end

identity<int>(42)      # 42
identity<string>("hi") # "hi"

# Type inference
identity(42)           # T inferred as int

# Generic classes
class Box<T>
  def initialize(value: T)
    self.value = value
  end

  def get(): T
    self.value
  end
end

box = Box<int>(value: 42)
puts(box.get())  # 42

# Generic structs
struct Pair<A, B>
  first: A
  second: B
end

p = Pair<int, string>(first: 1, second: "hello")
```

### Enums

```vibe
enum Color
  Red
  Green
  Blue
end

puts(Color.Red)    # 0
puts(Color.Green)  # 1
puts(Color.Blue)   # 2
```

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

# Unless (inverse of if)
unless logged_in
  puts("Please log in")
end

# Case / when
case status
when "active"
  puts("Active")
when "inactive", "disabled"
  puts("Not active")
else
  puts("Unknown")
end

# Ternary
result = x > 0 ? "positive" : "non-positive"

# Postfix conditionals
return "error" if x < 0
y = "ok" unless invalid

# For loops
for i in 1..10
  puts(i)
end

for name in ["Alice", "Bob"]
  puts(name)
end

# Hash iteration
for key, value in {a: 1, b: 2}
  puts("${key}: ${value}")
end

# While / until
while count < 10
  count += 1
end

until done
  process_next()
end
```

### Operators

```vibe
# Arithmetic
10 + 5    # 15
10 - 3    # 7
6 * 8     # 48
10 / 3    # 3
15 % 4    # 3
2 ** 10   # 1024 (power)

# Compound assignment
x += 5    x -= 3    x *= 2    x /= 4    x %= 3

# String repetition
"ha" * 3        # "hahaha"

# Nil coalescing
value ?? "default"      # returns value if non-nil, else "default"
a ?? b ?? c             # chained

# Comparison
a == b    a != b
a < b     a > b
a <= b    a >= b
"apple" < "banana"      # lexicographic string comparison

# Logical (short-circuit)
true && false    # false
true || false    # true
!true            # false

# Containment
5 in [1, 2, 3, 4, 5]   # true
"x" in "text"           # true
"key" in {key: 1}       # true
5 in 1..10              # true
```

### Arrays and Hash Maps

```vibe
# Arrays with negative indexing
numbers = [1, 2, 3, 4, 5]
numbers[0]       # 1
numbers[-1]      # 5 (last element)
numbers[-2]      # 4

# Array mutation
delete_result = remove_at(arr, 1)  # removes and returns element at index

# Hash maps
config = {host: "localhost", port: 8080}
config["host"]           # "localhost"
config.host              # "localhost" (dot access)
delete(config, "port")   # removes key, returns removed value

# Hash iteration
for k, v in config
  puts("${k} = ${v}")
end
```

### Strings

```vibe
# Multi-line strings
text = """
This is a
multi-line string
"""

# String indexing
"hello"[0]     # "h"
"hello"[-1]    # "o"

# String repetition
"abc" * 3      # "abcabcabc"

# String comparison
"apple" < "banana"   # true (lexicographic)

# String interpolation with any expression
"Result: ${2 + 2}"
"Type: ${value.type}"
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
finally
  cleanup()
end

throw "something went wrong"
```

### Union and Optional Types

```vibe
# Union types
x: string | int = "hello"
x = 42  # also valid

# Optional types (shorthand for type | nil)
name: string? = nil
name = "Alice"  # also valid
```

### Standard Library

```vibe
# Math module
import "math"
math_sqrt(16)     # 4
math_floor(3.7)   # 3
math_ceil(3.2)    # 4
math_round(3.5)   # 4
math_pow(2, 10)   # 1024
math_pi()         # 3.14159...
math_random()     # random float 0..1

# Regex module
import "regex"
regex_match("hello123", "[0-9]+")       # "123"
regex_match_all("a1b2c3", "[0-9]+")     # ["1", "2", "3"]
regex_replace("hello world", "world", "vibe")  # "hello vibe"

# Concurrent module
import "concurrent"
task = spawn(-> { expensive_work() })
result = await(task)
ch = Channel()
send(ch, "message")
msg = receive(ch)
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
| **Collections** | `len`, `push`, `pop`, `first`, `last`, `rest`, `append`, `unshift` |
| **Higher-order** | `map`, `filter`, `each`, `reduce`, `find`, `find_index`, `any`, `all`, `none`, `reject`, `flat_map`, `sort_by`, `map_with_index`, `each_with_index`, `partition`, `group_by` |
| **Array** | `sort`, `reverse`, `flatten`, `compact`, `uniq`, `take`, `drop`, `slice`, `zip`, `concat`, `sum`, `count`, `empty`, `index_of` |
| **Mutation** | `delete` (hash), `remove_at` (array) |
| **Search** | `contains` (arrays, strings, hashes) |
| **Hash** | `keys`, `values` |
| **String** | `split`, `join`, `replace`, `trim`, `string_length`, `upcase`, `downcase`, `capitalize`, `starts_with`, `ends_with`, `repeat`, `chars`, `pad_start`, `pad_end`, `string_reverse`, `string_slice`, `string_contains`, `index_of_string` |
| **Math** | `abs`, `min`, `max` |
| **File** | `read_file`, `write_file`, `file_exists` |
| **JSON** | `json_parse`, `json_encode` |
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

MIT -- see [LICENSE](LICENSE).
