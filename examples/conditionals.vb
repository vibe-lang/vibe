# Conditionals in Vibe

# Basic if statement
age = 25

if age >= 18
  puts("You are an adult")
else
  puts("You are a minor")
end

# If with multiple conditions
score = 85

if score >= 90
  grade = "A"
elsif score >= 80
  grade = "B"
elsif score >= 70
  grade = "C"
elsif score >= 60
  grade = "D"
else
  grade = "F"
end

puts("Your grade is ${grade}")

# If expression used for assignment
temperature = 28
status = if temperature > 30
  "hot"
elsif temperature > 20
  "warm"
elsif temperature > 10
  "cool"
else
  "cold"
end

puts("It is ${status} today")

# Complex conditional with logical operators
username = "admin"
password = "secret123"
is_logged_in = false

if username == "admin" && password == "secret123"
  is_logged_in = true
  puts("Login successful")
else
  puts("Login failed")
end

if is_logged_in || username == "guest"
  puts("Access granted")
else
  puts("Access denied")
end

# Nested if statements
num = 15

if num > 0
  if num % 2 == 0
    puts("Positive even number")
  else
    puts("Positive odd number")
  end
else
  if num < 0
    puts("Negative number")
  else
    puts("Zero")
  end
end

# Using conditionals in a function
def check_number(n: int): string
  if n > 0
    return "positive"
  elsif n < 0
    return "negative"
  else
    return "zero"
  end
end

puts("5 is ${check_number(5)}")
puts("-3 is ${check_number(-3)}")
puts("0 is ${check_number(0)}")
