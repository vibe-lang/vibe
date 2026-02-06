# Vibe Language Syntax Guide

This guide documents the syntax and features of the Vibe programming language, a Ruby-like language with optional type annotations.

## Table of Contents

1. [Variables](#variables)
2. [Types](#types)
3. [Operators](#operators)
4. [Control Structures](#control-structures)
5. [Functions](#functions)
6. [Arrays](#arrays)
7. [Hash Maps](#hash-maps)
8. [Ranges](#ranges)
9. [Structs](#structs)
10. [Comments](#comments)
11. [String Interpolation](#string-interpolation)
12. [Error Handling](#error-handling)
13. [Imports](#imports)
14. [Built-in Functions](#built-in-functions)

## Variables

Variables in Vibe are declared using direct assignment. Variable names should start with a letter or underscore and can contain letters, numbers, and underscores.

### Variable Declaration with Type Inference

```vibe
name = "Alice"
age = 30
is_student = false
```

### Variable Declaration with Type Annotation

```vibe
name: string = "Alice"
age: int = 30
is_student: boolean = false
salary: float = 75000.50
```

### Using `let` Keyword

The `let` keyword is optional and works the same as direct assignment:

```vibe
let x = 42
let name = "Bob"
```

## Types

Vibe includes the following built-in types:

- `int`: Integer numbers (e.g., `42`)
- `float`: Floating-point numbers (e.g., `3.14`)
- `string`: Text strings (e.g., `"hello"` or `'world'`)
- `boolean`: Boolean values (`true` or `false`)
- `nil`: The absence of a value (`nil`)

## Operators

### Arithmetic Operators

```vibe
a = 10
b = 3

sum = a + b        # Addition: 13
difference = a - b # Subtraction: 7
product = a * b    # Multiplication: 30
quotient = a / b   # Division: 3
remainder = a % b  # Modulo: 1
```

### Comparison Operators

```vibe
a = 10
b = 3

a == b    # Equal to: false
a != b    # Not equal to: true
a > b     # Greater than: true
a < b     # Less than: false
a >= b    # Greater than or equal to: true
a <= b    # Less than or equal to: false
```

### Logical Operators

```vibe
# AND — returns false if left side is falsy, otherwise returns right side
true && true     # true
true && false    # false
false && true    # false (short-circuits, right side not evaluated)

# OR — returns left side if truthy, otherwise returns right side
true || false    # true (short-circuits, right side not evaluated)
false || true    # true
false || false   # false

# NOT
!true            # false
!false           # true
!nil             # true (nil is falsy)
```

`&&` binds tighter than `||`, so `a || b && c` is parsed as `a || (b && c)`.

### Prefix Operators

```vibe
-5       # Negation
!true    # Logical NOT
```

## Control Structures

### If / Elsif / Else

```vibe
if age >= 18
  puts("You are an adult")
elsif age >= 13
  puts("You are a teenager")
else
  puts("You are a child")
end
```

### If Expressions

If statements can be used as expressions that return values:

```vibe
status = if age >= 18
  "adult"
else
  "minor"
end
```

### For Loops

Iterate over arrays or ranges:

```vibe
# Over an array
for name in ["Alice", "Bob", "Charlie"]
  puts(name)
end

# Over a range
for i in 1..10
  puts(i)
end

# Over a variable
numbers = [1, 2, 3, 4, 5]
for num in numbers
  puts(num)
end
```

### While Loops

```vibe
i = 0
while i < 10
  puts(i)
  i = i + 1
end
```

### Break and Continue

`break` exits the current loop. `continue` skips to the next iteration.

```vibe
# Break out of a loop early
for i in 1..100
  if i > 5
    break
  end
  puts(i)
end

# Skip even numbers
for i in 1..10
  if i % 2 == 0
    continue
  end
  puts(i)
end

# Works in while loops too
count = 0
while true
  count = count + 1
  if count > 10
    break
  end
end
```

## Functions

### Function Definition

Functions are defined using `def` and end with `end`. The last expression is implicitly returned.

```vibe
def greet(name: string): string
  "Hello, ${name}!"
end
```

### Explicit Return

```vibe
def factorial(n: int): int
  if n <= 1
    return 1
  end
  n * factorial(n - 1)
end
```

### Multiple Parameters

```vibe
def add(a: int, b: int): int
  a + b
end
```

### Function Calls

```vibe
result = add(5, 3)
puts(result)       # 8
puts(greet("World"))  # Hello, World!
```

### Closures

Functions capture variables from their enclosing scope:

```vibe
def make_counter()
  count = 0
  def increment(): int
    count = count + 1
    count
  end
  increment
end

counter = make_counter()
puts(counter())  # 1
puts(counter())  # 2
```

### Higher-Order Functions

Functions can be passed as arguments:

```vibe
def double(x: int): int
  x * 2
end

nums = [1, 2, 3, 4, 5]
result = map(nums, double)   # [2, 4, 6, 8, 10]
```

## Arrays

### Array Declaration

```vibe
empty = []
numbers = [1, 2, 3]
names = ["Alice", "Bob", "Charlie"]
```

### Typed Arrays

```vibe
integers: int[] = [1, 2, 3]
strings: string[] = ["hello", "world"]
```

### Nested Arrays

```vibe
matrix = [[1, 2], [3, 4]]
nested: int[][] = [[1, 2], [3, 4]]
```

### Array Access and Mutation

```vibe
arr = [10, 20, 30]
puts(arr[0])     # 10
puts(arr[2])     # 30

arr[1] = 99      # Mutation
puts(arr)        # [10, 99, 30]
```

### Array Concatenation

```vibe
a = [1, 2]
b = [3, 4]
c = a + b        # [1, 2, 3, 4]
```

## Hash Maps

Hash maps store key-value pairs.

### Creating a Hash

```vibe
# Bare identifier keys (converted to strings)
config = {host: "localhost", port: 8080, debug: true}

# String keys
settings = {"theme": "dark", "font_size": 14}

# Empty hash
empty = {}
```

### Access and Mutation

```vibe
config = {host: "localhost", port: 8080}

puts(config["host"])    # localhost
config["port"] = 9090   # Mutation
config["new_key"] = 42  # Add new key
```

### Hash Built-ins

```vibe
h = {a: 1, b: 2, c: 3}

keys(h)       # ["a", "b", "c"]
values(h)     # [1, 2, 3]
len(h)        # 3
contains(h, "a")  # true
```

## Ranges

Ranges represent a sequence of values between a start and end point.

### Inclusive Ranges

```vibe
# Range from 1 to 5 (includes 1, 2, 3, 4, 5)
r = 1..5
```

### Exclusive Ranges

```vibe
# Range from 1 to 5 (includes 1, 2, 3, 4, NOT 5)
r = 1...5
```

### Using Ranges

```vibe
# Iterating over a range
for i in 1..5
  puts(i)
end

# Ranges with variables
start = 10
end_val = 20
for i in start..end_val
  puts(i)
end
```

## Structs

Structs define lightweight data types with named fields.

### Struct Definition

```vibe
struct Point
  x: int
  y: int
end

struct Person
  name: string
  age: int
  active: boolean
end
```

### Creating Instances

```vibe
p = Point(x: 10, y: 20)
alice = Person(name: "Alice", age: 30, active: true)
```

### Field Access and Mutation

```vibe
puts(alice.name)    # Alice
puts(alice.age)     # 30

alice.age = 31      # Field mutation
puts(alice.age)     # 31
```

### Arrays of Structs

```vibe
people = [
  Person(name: "Alice", age: 30, active: true),
  Person(name: "Bob", age: 25, active: false)
]

for person in people
  puts("${person.name}: ${person.age}")
end
```

## Comments

Vibe supports single-line comments using `#`:

```vibe
# This is a comment
x = 42  # This is an end-of-line comment
```

## String Interpolation

Embed arbitrary expressions inside double-quoted strings using `${expression}`:

```vibe
name = "Alice"
age = 30
puts("Hello, ${name}!")                    # Hello, Alice!
puts("${name} is ${age} years old")        # Alice is 30 years old
puts("2 + 2 = ${2 + 2}")                  # 2 + 2 = 4
puts("Is adult: ${age >= 18}")             # Is adult: true
```

Interpolation supports any expression: variables, function calls, arithmetic, comparisons, dot access, and more.

Single-quoted strings do not support interpolation:

```vibe
puts('Hello, ${name}')  # Prints literally: Hello, ${name}
```

## Error Handling

### Try / Catch

```vibe
try
  result = risky_operation()
catch e
  puts("Error: ${e}")
end
```

### Throw

Raise errors with `throw`:

```vibe
def divide(a: int, b: int): int
  if b == 0
    throw "division by zero"
  end
  a / b
end

try
  divide(10, 0)
catch e
  puts("Caught: ${e}")
end
```

`catch` handles both `throw` values and runtime errors (like index out of bounds).

## Imports

Import and execute another Vibe file in the current environment:

```vibe
import "lib/helpers.vb"

# All functions and variables from helpers.vb are now available
result = helper_function()
```

Imports evaluate the file in the current scope, so all definitions become available.

## Built-in Functions

### I/O

| Function | Description | Example |
|----------|-------------|---------|
| `puts(value)` | Print value with newline | `puts("hello")` |
| `print(value)` | Print value without newline | `print("hello ")` |
| `input(prompt?)` | Read a line from stdin | `name = input("Name: ")` |

### Type Conversion

| Function | Description | Example |
|----------|-------------|---------|
| `type(value)` | Get type as string | `type(42)` → `"INTEGER"` |
| `to_s(value)` | Convert to string | `to_s(42)` → `"42"` |
| `to_i(value)` | Convert to integer | `to_i("42")` → `42` |
| `to_f(value)` | Convert to float | `to_f("3.14")` → `3.14` |

### Collections

| Function | Description | Example |
|----------|-------------|---------|
| `len(collection)` | Get length | `len([1,2,3])` → `3` |
| `push(arr, elem)` | Append element (returns new array) | `push([1,2], 3)` → `[1,2,3]` |
| `pop(arr)` | Get last element | `pop([1,2,3])` → `3` |
| `first(arr)` | Get first element | `first([1,2,3])` → `1` |
| `last(arr)` | Get last element | `last([1,2,3])` → `3` |
| `rest(arr)` | All elements except first | `rest([1,2,3])` → `[2,3]` |
| `append(arr, elem)` | Alias for `push` | `append([1,2], 3)` → `[1,2,3]` |

### Higher-Order

| Function | Description | Example |
|----------|-------------|---------|
| `map(arr, fn)` | Transform each element | `map([1,2,3], double)` |
| `filter(arr, fn)` | Keep elements where fn returns true | `filter([1,2,3], is_even)` |
| `each(arr, fn)` | Execute fn for each element | `each(names, puts)` |

### Array Operations

| Function | Description | Example |
|----------|-------------|---------|
| `sort(arr)` | Sort array (returns new) | `sort([3,1,2])` → `[1,2,3]` |
| `reverse(arr)` | Reverse array (returns new) | `reverse([1,2,3])` → `[3,2,1]` |
| `contains(arr, val)` | Check if array contains value | `contains([1,2,3], 2)` → `true` |

### Hash Operations

| Function | Description | Example |
|----------|-------------|---------|
| `keys(hash)` | Get all keys as array | `keys({a: 1})` → `["a"]` |
| `values(hash)` | Get all values as array | `values({a: 1})` → `[1]` |

### String Operations

| Function | Description | Example |
|----------|-------------|---------|
| `split(str, delim)` | Split string into array | `split("a,b,c", ",")` → `["a","b","c"]` |
| `join(arr, sep)` | Join array into string | `join(["a","b"], ",")` → `"a,b"` |
| `replace(str, old, new)` | Replace all occurrences | `replace("hello", "l", "r")` → `"herro"` |
| `trim(str)` | Remove leading/trailing whitespace | `trim("  hi  ")` → `"hi"` |
| `string_length(str)` | Get string character count | `string_length("hello")` → `5` |
| `contains(str, sub)` | Check if string contains substring | `contains("hello", "ell")` → `true` |

### Math

| Function | Description | Example |
|----------|-------------|---------|
| `abs(num)` | Absolute value | `abs(-5)` → `5` |
| `min(a, b)` | Smaller of two integers | `min(3, 7)` → `3` |
| `max(a, b)` | Larger of two integers | `max(3, 7)` → `7` |

### File I/O

| Function | Description | Example |
|----------|-------------|---------|
| `read_file(path)` | Read file contents as string | `read_file("data.txt")` |
| `write_file(path, content)` | Write string to file | `write_file("out.txt", "hello")` |
| `file_exists(path)` | Check if file exists | `file_exists("data.txt")` → `true` |

### System

| Function | Description | Example |
|----------|-------------|---------|
| `exit(code?)` | Exit the program | `exit(1)` |
