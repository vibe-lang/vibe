# Structs in Vibe

# Define a basic struct
struct Point
  x: int
  y: int
end

# Create struct instances
origin = Point(x: 0, y: 0)
p1 = Point(x: 5, y: 10)

puts("Origin: (${origin.x}, ${origin.y})")
puts("Point p1: (${p1.x}, ${p1.y})")

# Calculate distance squared between points
def distance_squared(a: Point, b: Point): int
  dx = b.x - a.x
  dy = b.y - a.y
  dx * dx + dy * dy
end

dist_sq = distance_squared(origin, p1)
puts("Distance squared from origin to p1: ${dist_sq}")

# More complex struct with multiple fields
struct Person
  name: string
  age: int
  active: boolean
end

# Create and use a person struct
alice = Person(name: "Alice", age: 30, active: true)
bob = Person(name: "Bob", age: 25, active: false)

puts("Person: ${alice.name}, Age: ${alice.age}, Active: ${alice.active}")
puts("Person: ${bob.name}, Age: ${bob.age}, Active: ${bob.active}")

# Struct mutation
alice.age = 31
puts("After birthday: ${alice.name} is now ${alice.age}")

# Array of structs
people = [
  Person(name: "Charlie", age: 35, active: true),
  Person(name: "Diana", age: 28, active: true),
  Person(name: "Evan", age: 42, active: false)
]

# Process array of structs
total_age = 0
active_count = 0

for person in people
  total_age = total_age + person.age

  if person.active
    active_count = active_count + 1
  end
end

puts("Total age: ${total_age}, Average age: ${total_age / 3}")
puts("Active people: ${active_count}")
