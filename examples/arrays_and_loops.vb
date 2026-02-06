# Arrays and Loops in Vibe

# Array declaration with type inference
numbers = [1, 2, 3, 4, 5]
names = ["Alice", "Bob", "Charlie"]

# Typed arrays
integers: int[] = [10, 20, 30, 40, 50]
strings: string[] = ["apple", "banana", "cherry"]

# Nested arrays
matrix = [[1, 2, 3], [4, 5, 6], [7, 8, 9]]
nested: int[][] = [[10, 20], [30, 40], [50, 60]]

# Array operations
sum = 0
for num in numbers
  sum = sum + num
end

puts("Sum of numbers: ${sum}")

# Nested array iteration
matrix_sum = 0
for row in matrix
  for value in row
    matrix_sum = matrix_sum + value
  end
end

puts("Sum of matrix values: ${matrix_sum}")

# Array modification during iteration
squares = []
for num in numbers
  squares = squares + [num * num]
end

puts("Squares: ${squares}")

# Index tracking in loops
i = 0
element_sum = 0
for name in names
  puts("${i}: ${name}")
  element_sum = element_sum + i
  i = i + 1
end

puts("Sum of indices: ${element_sum}")

# Range-based iteration
range_sum = 0
for i in 1..5
  range_sum = range_sum + i
end

puts("Sum of range 1..5: ${range_sum}")

# Array mutation
arr = [1, 2, 3, 4, 5]
arr[2] = 99
puts("After mutation: ${arr}")

# While loop with break
count = 0
while true
  count = count + 1
  if count > 5
    break
  end
end
puts("Counted to: ${count}")
