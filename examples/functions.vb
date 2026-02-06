# Functions in Vibe

# Basic function definition with explicit return
def add(a: int, b: int): int
  return a + b
end

result = add(5, 3)
puts("5 + 3 = ${result}")

# Function with implicit return (last expression)
def multiply(x: int, y: int): int
  x * y
end

product = multiply(4, 7)
puts("4 * 7 = ${product}")

# Recursive function
def factorial(n: int): int
  if n <= 1
    return 1
  end
  n * factorial(n - 1)
end

puts("10! = ${factorial(10)}")

# Function with string return
def greet(name: string): string
  "Hello, ${name}!"
end

message = greet("World")
puts(message)

# Higher-order functions
nums = [1, 2, 3, 4, 5]

def double(x: int): int
  x * 2
end

def is_even(x: int): boolean
  x % 2 == 0
end

doubled = map(nums, double)
puts("Doubled: ${doubled}")

evens = filter(nums, is_even)
puts("Evens: ${evens}")

# Closures
def make_adder(n: int)
  def adder(x: int): int
    x + n
  end
  adder
end

add5 = make_adder(5)
puts("add5(10) = ${add5(10)}")
