# Vibe Language Syntax Guide

This guide documents the syntax and features of the Vibe programming language, a Ruby-like language with static type annotations.

## Table of Contents

1. [Variables](#variables)
2. [Types](#types)
3. [Operators](#operators)
4. [Control Structures](#control-structures)
5. [Functions](#functions)
6. [Arrays](#arrays)
7. [Ranges](#ranges)
8. [Classes and Objects](#classes-and-objects)
9. [Comments](#comments)
10. [String Interpolation](#string-interpolation)

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

is_equal = a == b          # Equal to: false
is_not_equal = a != b      # Not equal to: true
is_greater = a > b         # Greater than: true
is_less = a < b            # Less than: false
is_greater_or_equal = a >= b # Greater than or equal to: true
is_less_or_equal = a <= b  # Less than or equal to: false
```

### Logical Operators

```vibe
is_true = true
is_false = false

and_result = is_true && is_false  # Logical AND: false
or_result = is_true || is_false   # Logical OR: true
not_result = !is_true             # Logical NOT: false
```

## Control Structures

### If Statements

```vibe
if age >= 18
  puts("You are an adult")
elsif age >= 13
  puts "You are a teenager"
else
  puts "You are a child"
end
```

### If Expressions

If statements can also be used as expressions that return values:

```vibe
status = if age >= 18
  "adult"
else
  "minor"
end
```

## Functions

### Function Definition

Functions in Vibe are defined using the `def` keyword and end with the `end` keyword.

```vibe
def greet(name: string): string
  "Hello, #{name}!"
end
```

### Function with Multiple Parameters and Type Annotations

```vibe
def add_numbers(x: int, y: int): int
  x + y
end
```

### Function with Multiple Parameters Sharing a Type Annotation

```vibe
def process(x, y: string, z: int): boolean
  # Parameters x and y share the same type annotation
  true
end
```

### Function Calls

```vibe
message = greet("World")
puts message  # Output: Hello, World!
```

### Functions with Explicit Return

```vibe
def multiply(x: int, y: int): int
  return x * y
end
```

### Functions with Implicit Return

The last expression in a function body is implicitly returned.

```vibe
def subtract(a: int, b: int): int
  a - b  # This value is returned
end
```

### One-line Function Definition

```vibe
def square(n: int): int n * n end
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

## Ranges

Ranges represent a sequence of values between a start and end point.

### Inclusive Ranges

Inclusive ranges include both the start and end values.

```vibe
# Range from 1 to 5 (includes 1, 2, 3, 4, 5)
r1 = 1..5

# Using variables
start = 10
end_val = 20
r2 = start..end_val

# Descending range (includes 50, 49, ..., 10)
r3 = 50..10
```

### Exclusive Ranges

Exclusive ranges include the start value but exclude the end value.

```vibe
# Range from 1 to 5 (includes 1, 2, 3, 4, not 5)
r1 = 1...5

# Using variables
start = 10
end_val = 20
r2 = start...end_val

# Descending range (includes 50, 49, ..., 11, not 10)
r3 = 50...10
```

### Range Constructor

You can also create ranges using the `Range` constructor.

```vibe
# Inclusive range from 1 to 10
r1 = Range(1, 10)

# Using variables
start = 5
end_val = 15
r2 = Range(start, end_val)
```

### Using Ranges

Ranges can be used in various contexts:

```vibe
# Checking if a value is in a range
in_range = 3.in?(1..5)  # true

# Iterating over a range
for i in 1..5
  puts(i)
end

# Getting an array from a range
numbers = (1..5).to_array()  # [1, 2, 3, 4, 5]
```

## Classes and Objects

### Class Definition

```vibe
class Person
  prop name: string
  prop age: int

  def initialize(name: string, age: int)
    @name = name
    @age = age
  end

  def describe(): string
    "#{@name} is #{@age} years old"
  end

  def have_birthday()
    @age = @age + 1
  end
end
```

### Creating Objects

```vibe
alice = Person.new("Alice", 30)
puts(alice.describe())  # Output: Alice is 30 years old

alice.have_birthday()
puts(alice.describe())  # Output: Alice is 31 years old
```

### Instance Variables

Instance variables are prefixed with `@` and are accessible within instance methods.

```vibe
def describe(): string
  "My name is #{@name} and I am #{@age} years old"
end
```

## Comments

Vibe supports single-line comments using the `#` character:

```vibe
# This is a comment
x = 42  # This is an end-of-line comment
```

## String Interpolation

String interpolation allows you to embed expressions inside string literals:

```vibe
name = "Alice"
age = 30
message = "Hello, my name is #{name} and I am #{age} years old"
```

In the above example, `#{name}` will be replaced with the value of the `name` variable, and `#{age}` will be replaced with the value of the `age` variable.

## Structs

Vibe supports struct definitions for creating user-defined compound types:

```vibe
struct Point
  x: int
  y: int
end

p = Point(x: 10, y: 20)
```