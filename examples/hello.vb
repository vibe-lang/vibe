# Hello World in Vibe
func greet(name: String): String {
  return "Hello, #{name}!"
}

# Variable declaration with type inference
let message = greet("World")
puts(message)

# Variable with explicit type
let count: Int = 5
puts("Count: #{count}")

# Simple class with typed properties
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

# Create and use a Person instance
let person = Person.new("Alice", 30)
puts(person.describe())