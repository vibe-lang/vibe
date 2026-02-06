# Ranges in Vibe

# Inclusive ranges (includes both start and end)
r1 = 1..5  # [1, 2, 3, 4, 5]
puts("Range 1..5:")

for i in r1
  puts(i)
end

# Exclusive ranges (includes start, excludes end)
r2 = 1...5  # [1, 2, 3, 4]
puts("Range 1...5:")

for i in r2
  puts(i)
end

# Using variables in ranges
start = 10
end_val = 15

r3 = start..end_val
puts("Range ${start}..${end_val}:")

for i in r3
  puts(i)
end

# Summing numbers in a range
sum = 0
for i in 1..10
  sum = sum + i
end

puts("Sum of numbers 1 to 10: ${sum}")

# Finding max in a range
max_val = 0
for i in 5..15
  if i > max_val
    max_val = i
  end
end

puts("Max value in range 5..15: ${max_val}")

# Using ranges to build an array
squares = []
for i in 1..5
  squares = squares + [i * i]
end

puts("Squares of 1..5:")
for s in squares
  puts(s)
end
