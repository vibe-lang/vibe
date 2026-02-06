# Showcase: Vibe Language Features
# This example demonstrates hash maps, while loops, break/continue,
# try/catch, mutation, higher-order functions, and logical operators.

# --- Hash Maps ---
config = {host: "localhost", port: 8080, debug: true}
puts("Server config:")
puts("  Host: ${config["host"]}")
puts("  Port: ${config["port"]}")
puts("  Debug: ${config["debug"]}")

config["port"] = 9090
puts("  Updated port: ${config["port"]}")

# --- While loop with break/continue ---
puts("\nOdd numbers from 1 to 10:")
i = 0
while i < 10
  i = i + 1
  if i % 2 == 0
    continue
  end
  puts(i)
end

# --- FizzBuzz with logical operators ---
puts("\nFizzBuzz 1-20:")
for n in 1..20
  if n % 3 == 0 && n % 5 == 0
    puts("${n}: FizzBuzz")
  elsif n % 3 == 0
    puts("${n}: Fizz")
  elsif n % 5 == 0
    puts("${n}: Buzz")
  else
    puts("${n}: ${n}")
  end
end

# --- Try/catch error handling ---
puts("\nError handling:")
try
  throw "something went wrong"
catch e
  puts("Caught: ${e}")
end

# --- Builtins: map, filter, sort, reverse ---
nums = [5, 3, 1, 4, 2]
sorted_nums = sort(nums)
puts("\nOriginal: ${nums}")
puts("Sorted: ${sorted_nums}")
puts("Reversed: ${reverse(sorted_nums)}")

def square(x: int): int
  x * x
end

def gt2(x: int): boolean
  x > 2
end

puts("Squared: ${map(nums, square)}")
puts("Filtered (>2): ${filter(nums, gt2)}")

# --- String builtins ---
sentence = "  Hello Vibe World  "
puts("\nTrimmed: '${trim(sentence)}'")
words = split(trim(sentence), " ")
puts("Words: ${words}")
puts("Joined: ${join(words, " - ")}")
puts("Replaced: ${replace(trim(sentence), "Vibe", "Amazing")}")

# --- Hash with keys/values ---
scores = {alice: 95, bob: 87, charlie: 92}
puts("\nScores: ${scores}")
puts("Names: ${keys(scores)}")
puts("Values: ${values(scores)}")

puts("\nDone!")
