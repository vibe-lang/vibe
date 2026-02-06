package interpreter

// Tests for gap analysis features: compound assignment, string methods,
// ternary, unless/until, case/when, default params, negative indexing,
// arrow functions, pipe operator, in operator.

import "testing"

// ---------------------------------------------------------------------------
// 1. Compound assignment operators (+=, -=, *=, /=, %=)
// ---------------------------------------------------------------------------

func TestCompoundAssignment(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{"plus assign", "x = 10\nx += 5\nx", 15},
		{"minus assign", "x = 10\nx -= 3\nx", 7},
		{"multiply assign", "x = 4\nx *= 3\nx", 12},
		{"divide assign", "x = 20\nx /= 4\nx", 5},
		{"modulo assign", "x = 17\nx %= 5\nx", 2},
		{"chain compound", "x = 1\nx += 2\nx *= 3\nx", 9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalInput(t, tt.input)
			assertInteger(t, result, tt.expected)
		})
	}
}

func TestCompoundAssignmentString(t *testing.T) {
	result := evalInput(t, `
s = "hello"
s += " world"
s`)
	assertString(t, result, "hello world")
}

// ---------------------------------------------------------------------------
// 2. String methods
// ---------------------------------------------------------------------------

func TestStringUpcase(t *testing.T) {
	result := evalInput(t, `"hello".upcase`)
	assertString(t, result, "HELLO")
}

func TestStringDowncase(t *testing.T) {
	result := evalInput(t, `"HELLO".downcase`)
	assertString(t, result, "hello")
}

func TestStringCapitalize(t *testing.T) {
	result := evalInput(t, `"hello WORLD".capitalize`)
	assertString(t, result, "Hello world")
}

func TestStringStartsWith(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{`"hello world".starts_with("hello")`, true},
		{`"hello world".starts_with("world")`, false},
	}
	for _, tt := range tests {
		result := evalInput(t, tt.input)
		assertBoolean(t, result, tt.expected)
	}
}

func TestStringEndsWith(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{`"hello world".ends_with("world")`, true},
		{`"hello world".ends_with("hello")`, false},
	}
	for _, tt := range tests {
		result := evalInput(t, tt.input)
		assertBoolean(t, result, tt.expected)
	}
}

func TestStringRepeat(t *testing.T) {
	result := evalInput(t, `"ha".repeat(3)`)
	assertString(t, result, "hahaha")
}

func TestStringChars(t *testing.T) {
	result := evalInput(t, `"abc".chars.len`)
	assertInteger(t, result, 3)
}

func TestStringReverse(t *testing.T) {
	result := evalInput(t, `"hello".reverse`)
	assertString(t, result, "olleh")
}

func TestStringPadStart(t *testing.T) {
	result := evalInput(t, `"5".pad_start(3, "0")`)
	assertString(t, result, "005")
}

func TestStringPadEnd(t *testing.T) {
	result := evalInput(t, `"hi".pad_end(5, ".")`)
	assertString(t, result, "hi...")
}

// ---------------------------------------------------------------------------
// 3. Ternary operator
// ---------------------------------------------------------------------------

func TestTernaryOperator(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{"true condition", "true ? 1 : 2", int64(1)},
		{"false condition", "false ? 1 : 2", int64(2)},
		{"expression condition", "5 > 3 ? \"yes\" : \"no\"", "yes"},
		{"nested ternary", "true ? (false ? 1 : 2) : 3", int64(2)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalInput(t, tt.input)
			switch expected := tt.expected.(type) {
			case int64:
				assertInteger(t, result, expected)
			case string:
				assertString(t, result, expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 4. unless + until
// ---------------------------------------------------------------------------

func TestUnless(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{
			"unless false executes body",
			`x = 0
unless false
  x = 42
end
x`,
			int64(42),
		},
		{
			"unless true skips body",
			`x = 0
unless true
  x = 42
end
x`,
			int64(0),
		},
		{
			"unless with else",
			`x = 0
unless true
  x = 1
else
  x = 2
end
x`,
			int64(2),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalInput(t, tt.input)
			assertInteger(t, result, tt.expected.(int64))
		})
	}
}

func TestUntil(t *testing.T) {
	result := evalInput(t, `
x = 0
until x >= 5
  x += 1
end
x`)
	assertInteger(t, result, 5)
}

func TestUntilBreak(t *testing.T) {
	result := evalInput(t, `
x = 0
until false
  x += 1
  if x == 3
    break
  end
end
x`)
	assertInteger(t, result, 3)
}

// ---------------------------------------------------------------------------
// 5. case/when
// ---------------------------------------------------------------------------

func TestCaseWhen(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"match first",
			`case 1
when 1
  "one"
when 2
  "two"
end`,
			"one",
		},
		{
			"match second",
			`case 2
when 1
  "one"
when 2
  "two"
end`,
			"two",
		},
		{
			"match else",
			`case 99
when 1
  "one"
when 2
  "two"
else
  "other"
end`,
			"other",
		},
		{
			"multi-value when",
			`case 3
when 1, 2
  "low"
when 3, 4
  "mid"
else
  "high"
end`,
			"mid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalInput(t, tt.input)
			assertString(t, result, tt.expected)
		})
	}
}

func TestCaseWhenWithStrings(t *testing.T) {
	result := evalInput(t, `
name = "alice"
case name
when "alice"
  "found alice"
when "bob"
  "found bob"
else
  "unknown"
end`)
	assertString(t, result, "found alice")
}

// ---------------------------------------------------------------------------
// 6. Default parameter values
// ---------------------------------------------------------------------------

func TestDefaultParameters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"use default",
			`def greet(name = "world")
  "Hello, ${name}!"
end
greet()`,
			"Hello, world!",
		},
		{
			"override default",
			`def greet(name = "world")
  "Hello, ${name}!"
end
greet("Alice")`,
			"Hello, Alice!",
		},
		{
			"mixed required and default",
			`def greet(greeting, name = "world")
  "${greeting}, ${name}!"
end
greet("Hi")`,
			"Hi, world!",
		},
		{
			"multiple defaults",
			`def point(x = 0, y = 0)
  x + y
end
point()`,
			"0", // 0 + 0 as int
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalInput(t, tt.input)
			if tt.name == "multiple defaults" {
				assertInteger(t, result, 0)
			} else {
				assertString(t, result, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 7. Negative array indexing
// ---------------------------------------------------------------------------

func TestNegativeArrayIndex(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{"last element", "[10, 20, 30][-1]", 30},
		{"second to last", "[10, 20, 30][-2]", 20},
		{"first via negative", "[10, 20, 30][-3]", 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalInput(t, tt.input)
			assertInteger(t, result, tt.expected)
		})
	}
}

func TestNegativeArrayIndexAssignment(t *testing.T) {
	result := evalInput(t, `
arr = [1, 2, 3]
arr[-1] = 99
arr[-1]`)
	assertInteger(t, result, 99)
}

// ---------------------------------------------------------------------------
// 8. Arrow functions
// ---------------------------------------------------------------------------

func TestArrowFunction(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{
			"basic arrow",
			`double = -> (x) { x * 2 }
double(5)`,
			10,
		},
		{
			"arrow with multiple params",
			`add = -> (x, y) { x + y }
add(3, 4)`,
			7,
		},
		{
			"arrow as callback",
			`[1, 2, 3].map(-> (x) { x * 10 }).last`,
			30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalInput(t, tt.input)
			assertInteger(t, result, tt.expected)
		})
	}
}

// ---------------------------------------------------------------------------
// 9. Pipe operator
// ---------------------------------------------------------------------------

func TestPipeOperator(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{
			"pipe to function",
			`def double(x)
  x * 2
end
5 |> double`,
			10,
		},
		{
			"pipe chain",
			`def double(x)
  x * 2
end
def add_one(x)
  x + 1
end
5 |> double |> add_one`,
			11,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalInput(t, tt.input)
			assertInteger(t, result, tt.expected)
		})
	}
}

func TestPipeWithArgs(t *testing.T) {
	result := evalInput(t, `
def add(a, b)
  a + b
end
5 |> add(3)`)
	assertInteger(t, result, 8)
}

// ---------------------------------------------------------------------------
// 18. in operator
// ---------------------------------------------------------------------------

func TestInOperatorArray(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{`3 in [1, 2, 3, 4, 5]`, true},
		{`6 in [1, 2, 3, 4, 5]`, false},
		{`"hello" in ["hello", "world"]`, true},
		{`"foo" in ["hello", "world"]`, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := evalInput(t, tt.input)
			assertBoolean(t, result, tt.expected)
		})
	}
}

func TestInOperatorString(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{`"ell" in "hello"`, true},
		{`"xyz" in "hello"`, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := evalInput(t, tt.input)
			assertBoolean(t, result, tt.expected)
		})
	}
}

func TestInOperatorHash(t *testing.T) {
	result := evalInput(t, `h = {name: "Alice", age: 30}
"name" in h`)
	assertBoolean(t, result, true)
}

func TestInOperatorRange(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{`3 in 1..5`, true},
		{`5 in 1..5`, true},
		{`6 in 1..5`, false},
		{`5 in 1...5`, false}, // exclusive
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := evalInput(t, tt.input)
			assertBoolean(t, result, tt.expected)
		})
	}
}

// ---------------------------------------------------------------------------
// 10. Destructuring assignment
// ---------------------------------------------------------------------------

func TestDestructureArray(t *testing.T) {
	result := evalInput(t, `
a, b, c = [1, 2, 3]
a + b + c`)
	assertInteger(t, result, 6)
}

func TestDestructureArrayExtraNames(t *testing.T) {
	// Extra names get nil
	result := evalInput(t, `
a, b, c = [1, 2]
c`)
	if _, ok := result.(*Nil); !ok {
		t.Fatalf("expected Nil for extra destructure name, got %T", result)
	}
}

func TestDestructureHash(t *testing.T) {
	result := evalInput(t, `
name, age = {name: "Alice", age: 30}
name`)
	// Hash destructure extracts by key name
	assertString(t, result, "Alice")
}

// ---------------------------------------------------------------------------
// 12. finally block
// ---------------------------------------------------------------------------

func TestFinallyBlock(t *testing.T) {
	result := evalInput(t, `
x = 0
try
  x = 1
  throw "error"
catch e
  x = 2
finally
  x = x + 10
end
x`)
	assertInteger(t, result, 12)
}

func TestFinallyWithoutError(t *testing.T) {
	result := evalInput(t, `
x = 0
try
  x = 5
catch e
  x = -1
finally
  x = x + 100
end
x`)
	assertInteger(t, result, 105)
}

// ---------------------------------------------------------------------------
// 15. Math stdlib
// ---------------------------------------------------------------------------

func TestMathSqrt(t *testing.T) {
	result := evalInput(t, `import "math"
sqrt(16)`)
	assertFloat(t, result, 4.0)
}

func TestMathFloor(t *testing.T) {
	result := evalInput(t, `import "math"
floor(3.7)`)
	assertInteger(t, result, 3)
}

func TestMathCeil(t *testing.T) {
	result := evalInput(t, `import "math"
ceil(3.2)`)
	assertInteger(t, result, 4)
}

func TestMathRound(t *testing.T) {
	result := evalInput(t, `import "math"
round(3.5)`)
	assertInteger(t, result, 4)
}

func TestMathPow(t *testing.T) {
	result := evalInput(t, `import "math"
pow(2, 10)`)
	assertInteger(t, result, 1024)
}

func TestMathPI(t *testing.T) {
	result := evalInput(t, `import "math"
Math["PI"]`)
	f, ok := result.(*Float)
	if !ok {
		t.Fatalf("expected Float, got %T", result)
	}
	if f.Value < 3.14 || f.Value > 3.15 {
		t.Errorf("expected PI ~3.14159, got %f", f.Value)
	}
}

func TestMathRandom(t *testing.T) {
	result := evalInput(t, `import "math"
random()`)
	f, ok := result.(*Float)
	if !ok {
		t.Fatalf("expected Float, got %T", result)
	}
	if f.Value < 0 || f.Value >= 1 {
		t.Errorf("expected random in [0, 1), got %f", f.Value)
	}
}

// ---------------------------------------------------------------------------
// 11. Classes (Full OOP)
// ---------------------------------------------------------------------------

func TestClassBasic(t *testing.T) {
	result := evalInput(t, `
class Animal
  def initialize(name)
    self.name = name
  end

  def speak()
    "..."
  end
end

a = Animal("Rex")
a.name`)
	assertString(t, result, "Rex")
}

func TestClassMethod(t *testing.T) {
	result := evalInput(t, `
class Animal
  def initialize(name)
    self.name = name
  end

  def speak()
    "I am ${self.name}"
  end
end

a = Animal("Rex")
a.speak()`)
	assertString(t, result, "I am Rex")
}

func TestClassInheritance(t *testing.T) {
	result := evalInput(t, `
class Animal
  def initialize(name)
    self.name = name
  end

  def speak()
    "..."
  end
end

class Dog < Animal
  def speak()
    "Woof! I'm ${self.name}"
  end
end

d = Dog("Buddy")
d.speak()`)
	assertString(t, result, "Woof! I'm Buddy")
}

func TestClassInheritedMethod(t *testing.T) {
	result := evalInput(t, `
class Animal
  def initialize(name)
    self.name = name
  end

  def get_name()
    self.name
  end
end

class Dog < Animal
  def speak()
    "Woof!"
  end
end

d = Dog("Buddy")
d.get_name()`)
	assertString(t, result, "Buddy")
}

func TestClassMultipleInstances(t *testing.T) {
	result := evalInput(t, `
class Counter
  def initialize()
    self.count = 0
  end

  def increment()
    self.count = self.count + 1
  end

  def get_count()
    self.count
  end
end

c1 = Counter()
c2 = Counter()
c1.increment()
c1.increment()
c2.increment()
c1.get_count() + c2.get_count()`)
	assertInteger(t, result, 3)
}

func TestInOperatorInIfCondition(t *testing.T) {
	result := evalInput(t, `
names = ["alice", "bob", "charlie"]
if "alice" in names
  "found"
else
  "not found"
end`)
	assertString(t, result, "found")
}

// ---------------------------------------------------------------------------
// 13. Union types and optional types
// ---------------------------------------------------------------------------

func TestUnionTypeStringOrNil(t *testing.T) {
	// string | nil should accept nil
	result := evalInput(t, `
name: string | nil = nil
name`)
	if _, ok := result.(*Nil); !ok {
		t.Fatalf("expected Nil, got %T", result)
	}
}

func TestUnionTypeStringOrNilWithValue(t *testing.T) {
	result := evalInput(t, `
name: string | nil = "Alice"
name`)
	assertString(t, result, "Alice")
}

func TestOptionalTypeSyntax(t *testing.T) {
	// string? is shorthand for string | nil
	result := evalInput(t, `
name: string? = nil
name`)
	if _, ok := result.(*Nil); !ok {
		t.Fatalf("expected Nil, got %T", result)
	}
}

func TestOptionalTypeWithValue(t *testing.T) {
	result := evalInput(t, `
count: int? = 42
count`)
	assertInteger(t, result, 42)
}

func TestUnionTypeIntOrString(t *testing.T) {
	result := evalInput(t, `
value: int | string = "hello"
value`)
	assertString(t, result, "hello")
}

// ---------------------------------------------------------------------------
// 17. Regex support
// ---------------------------------------------------------------------------

func TestRegexMatch(t *testing.T) {
	result := evalInput(t, `
import "regex"
pattern = Regex("\\d+")
match(pattern, "hello 123 world")`)
	assertString(t, result, "123")
}

func TestRegexMatchAll(t *testing.T) {
	result := evalInput(t, `
import "regex"
pattern = Regex("\\d+")
match_all(pattern, "a1 b2 c3").len`)
	assertInteger(t, result, 3)
}

func TestRegexNoMatch(t *testing.T) {
	result := evalInput(t, `
import "regex"
match(Regex("\\d+"), "no numbers here")`)
	if _, ok := result.(*Nil); !ok {
		t.Fatalf("expected Nil for no match, got %T", result)
	}
}

func TestRegexReplace(t *testing.T) {
	result := evalInput(t, `
import "regex"
replace_regex(Regex("\\d+"), "a1 b2 c3", "X")`)
	assertString(t, result, "aX bX cX")
}

// ---------------------------------------------------------------------------
// 19. Enums
// ---------------------------------------------------------------------------

func TestEnumBasic(t *testing.T) {
	result := evalInput(t, `
enum Color
  Red
  Green
  Blue
end
Color.Red`)
	assertInteger(t, result, 0)
}

func TestEnumValues(t *testing.T) {
	result := evalInput(t, `
enum Color
  Red
  Green
  Blue
end
Color.Blue`)
	assertInteger(t, result, 2)
}

func TestEnumComparison(t *testing.T) {
	result := evalInput(t, `
enum Direction
  North
  South
  East
  West
end
d = Direction.North
d == Direction.North`)
	assertBoolean(t, result, true)
}

// ---------------------------------------------------------------------------
// 16. Concurrency (basic test)
// ---------------------------------------------------------------------------

func TestConcurrentSpawnAwait(t *testing.T) {
	result := evalInput(t, `
import "concurrent"
task = spawn(fn() { 42 })
await(task)`)
	assertInteger(t, result, 42)
}

func TestConcurrentChannel(t *testing.T) {
	result := evalInput(t, `
import "concurrent"
ch = Channel()
spawn(fn() { send(ch, 99) })
receive(ch)`)
	assertInteger(t, result, 99)
}
