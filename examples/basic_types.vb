# Basic Types and Operators in Vibe

# Variable assignment with type inference
name = "Alice"
age = 30
is_active = true
height = 1.75

# Variables with type annotations
count: int = 42
temperature: float = 98.6
username: string = "bob_123"
enabled: boolean = false

# Print variables with string interpolation
puts("Name: ${name}")
puts("Age: ${age}")
puts("Active status: ${is_active}")
puts("Height: ${height}m")

# Arithmetic operations
sum = 10 + 5
difference = 20 - 7
product = 6 * 8
quotient = 100 / 4
remainder = 15 % 4

puts("Sum: ${sum}")
puts("Difference: ${difference}")
puts("Product: ${product}")
puts("Quotient: ${quotient}")
puts("Remainder: ${remainder}")

# Comparison and logical operators
a = 10
b = 20
puts("a < b: ${a < b}")
puts("a == b: ${a == b}")
puts("a != b: ${a != b}")
puts("a < 5 || b > 15: ${a < 5 || b > 15}")
puts("a > 5 && b < 25: ${a > 5 && b < 25}")
