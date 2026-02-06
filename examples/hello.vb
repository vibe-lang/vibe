# Hello World in Vibe
def greet(name: string): string
  return "Hello, ${name}!"
end

# Variable declaration with type inference
message = greet("World")
puts(message)

# Variable with explicit type
count: int = 5
puts("Count: ${count}")

# Struct definition
struct Person
  name: string
  age: int
end

# Create and use a Person instance
person = Person(name: "Alice", age: 30)
puts(person.name)
puts(person.age)