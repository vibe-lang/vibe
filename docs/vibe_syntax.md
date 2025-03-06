# Vibe Language Syntax Guide

This guide documents the syntax and features of the Vibe programming language, a Ruby-like language with static type annotations.

## Table of Contents

1. [Variables](#variables)
2. [Types](#types)
3. [Operators](#operators)
4. [Control Structures](#control-structures)
5. [Functions](#functions)
6. [Classes and Objects](#classes-and-objects)
7. [Comments](#comments)
8. [String Interpolation](#string-interpolation)

## Variables

Variables in Vibe are declared using the `let` keyword. Variable names should start with a letter or underscore and can contain letters, numbers, and underscores.

### Variable Declaration with Type Inference

```vibe
let name = "Alice"
let age = 30
let is_student = false
```

### Variable Declaration with Type Annotation

```vibe
let name: String = "Alice"
let age: Int = 30
let is_student: Boolean = false
let salary: Float = 75000.50
```

## Types

Vibe includes the following built-in types:

- `Int`: Integer numbers (e.g., `42`)
- `Float`: Floating-point numbers (e.g., `3.14`)
- `String`: Text strings (e.g., `"hello"` or `'world'`)
- `Boolean`: Boolean values (`true` or `false`)
- `Nil`: The absence of a value (`nil`)

## Operators

### Arithmetic Operators

```vibe
let a = 10
let b = 3

let sum = a + b        # Addition: 13
let difference = a - b # Subtraction: 7
let product = a * b    # Multiplication: 30
let quotient = a / b   # Division: 3
let remainder = a % b  # Modulo: 1
```

### Comparison Operators

```vibe
let a = 10
let b = 3

let is_equal = a == b          # Equal to: false
let is_not_equal = a != b      # Not equal to: true
let is_greater = a > b         # Greater than: true
let is_less = a < b            # Less than: false
let is_greater_or_equal = a >= b # Greater than or equal to: true
let is_less_or_equal = a <= b  # Less than or equal to: false
```

### Logical Operators

```vibe
let is_true = true
let is_false = false

let and_result = is_true && is_false  # Logical AND: false
let or_result = is_true || is_false   # Logical OR: true
let not_result = !is_true             # Logical NOT: false
```

## Control Structures

### If Statements

```vibe
if age >= 18 {
  puts("You are an adult")
} elsif age >= 13 {
  puts("You are a teenager")
} else {
  puts("You are a child")
}
```

### If Expressions

If statements can also be used as expressions that return values:

```vibe
let status = if age >= 18 {
  "adult"
} else {
  "minor"
}
```

## Functions

### Function Definition

Functions in Vibe are defined using the `func` keyword.

```vibe
func greet(name: String): String {
  return "Hello, #{name}!"
}
```

### Function Calls

```vibe
let message = greet("World")
puts(message)  # Output: Hello, World!
```

### Functions without Return Values

If a function doesn't specify a return type, it implicitly returns `nil`.

```vibe
func display_info(name: String, age: Int) {
  puts("Name: #{name}, Age: #{age}")
}
```

### Anonymous Functions

```vibe
let add = func(a: Int, b: Int): Int {
  return a + b
}

let result = add(3, 4)  # Result: 7
```

## Classes and Objects

### Class Definition

```vibe
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

  func have_birthday() {
    @age = @age + 1
  }
}
```

### Creating Objects

```vibe
let alice = Person.new("Alice", 30)
puts(alice.describe())  # Output: Alice is 30 years old

alice.have_birthday()
puts(alice.describe())  # Output: Alice is 31 years old
```

### Instance Variables

Instance variables are prefixed with `@` and are accessible within instance methods.

```vibe
func describe(): String {
  return "My name is @{name} and I am @{age} years old"
}
```

## Comments

Vibe supports single-line comments using the `#` character:

```vibe
# This is a comment
let x = 42  # This is an end-of-line comment
```

## String Interpolation

String interpolation allows you to embed expressions inside string literals:

```vibe
let name = "Alice"
let age = 30
let message = "Hello, my name is #{name} and I am #{age} years old"
```

In the above example, `#{name}` will be replaced with the value of the `name` variable, and `#{age}` will be replaced with the value of the `age` variable.