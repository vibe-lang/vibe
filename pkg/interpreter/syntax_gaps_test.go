package interpreter

// Tests for syntax gap features:
// 1. String comparison operators (<, >, <=, >=)
// 2. String indexing (str[0], str[-1])
// 3. Nil coalescing ?? and optional chaining ?.
// 4. Hash iteration (for k, v in hash)
// 5. Variadic functions / splat (*args)
// 6. Const declarations
// 7. String * repetition and ** power operator
// 8. Multi-line strings / heredocs
// 9. delete/remove_at for hashes and arrays
// 10. Postfix if/unless

import (
	"strings"
	"testing"
)

// ===========================================================================
// 1. String comparison operators (<, >, <=, >=)
// ===========================================================================

func TestStringComparison(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"less than true", `"apple" < "banana"`, true},
		{"less than false", `"banana" < "apple"`, false},
		{"greater than true", `"banana" > "apple"`, true},
		{"greater than false", `"apple" > "banana"`, false},
		{"less than equal true", `"apple" <= "banana"`, true},
		{"less than equal same", `"apple" <= "apple"`, true},
		{"less than equal false", `"banana" <= "apple"`, false},
		{"greater than equal true", `"banana" >= "apple"`, true},
		{"greater than equal same", `"banana" >= "banana"`, true},
		{"greater than equal false", `"apple" >= "banana"`, false},
		{"case sensitivity", `"Z" < "a"`, true},
		{"empty string less", `"" < "a"`, true},
		{"single char", `"a" < "b"`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalInput(t, tt.input)
			assertBoolean(t, result, tt.expected)
		})
	}
}

func TestStringComparisonInIf(t *testing.T) {
	result := evalInput(t, `
name = "Charlie"
if name > "Bob"
  "after bob"
else
  "before bob"
end`)
	assertString(t, result, "after bob")
}

// ===========================================================================
// 2. String indexing (str[0], str[-1])
// ===========================================================================

func TestStringIndexing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"first char", `"hello"[0]`, "h"},
		{"last char via index", `"hello"[4]`, "o"},
		{"middle char", `"hello"[2]`, "l"},
		{"negative index last", `"hello"[-1]`, "o"},
		{"negative index first", `"hello"[-5]`, "h"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalInput(t, tt.input)
			assertString(t, result, tt.expected)
		})
	}
}

func TestStringIndexingVariable(t *testing.T) {
	result := evalInput(t, `
s = "world"
s[0]`)
	assertString(t, result, "w")
}

func TestStringIndexingNegative(t *testing.T) {
	result := evalInput(t, `
s = "vibe"
s[-1]`)
	assertString(t, result, "e")
}

func TestStringIndexOutOfBounds(t *testing.T) {
	result := evalInput(t, `"hi"[5]`)
	assertError(t, result, "index out of bounds")
}

// ===========================================================================
// 3. Nil coalescing ?? and optional chaining ?.
// ===========================================================================

func TestNilCoalescing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"nil uses default", `x = nil
x ?? "default"`, "default"},
		{"value ignores default", `x = "hello"
x ?? "default"`, "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalInput(t, tt.input)
			assertString(t, result, tt.expected)
		})
	}
}

func TestNilCoalescingInteger(t *testing.T) {
	result := evalInput(t, `
x = nil
x ?? 42`)
	assertInteger(t, result, 42)
}

func TestNilCoalescingNonNil(t *testing.T) {
	result := evalInput(t, `
x = 10
x ?? 42`)
	assertInteger(t, result, 10)
}

func TestNilCoalescingChained(t *testing.T) {
	result := evalInput(t, `
a = nil
b = nil
c = "found"
a ?? b ?? c`)
	assertString(t, result, "found")
}

func TestNilCoalescingFalseNotNil(t *testing.T) {
	// false is not nil, should not trigger default
	result := evalInput(t, `
x = false
x ?? "default"`)
	assertBoolean(t, result, false)
}

// ===========================================================================
// 4. Hash iteration (for k, v in hash)
// ===========================================================================

func TestHashIterationKeys(t *testing.T) {
	result := evalInput(t, `
h = {"a": 1, "b": 2, "c": 3}
result = []
for k, v in h
  result = push(result, k)
end
len(result)`)
	assertInteger(t, result, 3)
}

func TestHashIterationValues(t *testing.T) {
	result := evalInput(t, `
h = {"x": 10, "y": 20}
total = 0
for k, v in h
  total += v
end
total`)
	assertInteger(t, result, 30)
}

func TestHashIterationKeyValue(t *testing.T) {
	result := evalInput(t, `
h = {"name": "Alice"}
result = ""
for k, v in h
  result = k + "=" + v
end
result`)
	assertString(t, result, "name=Alice")
}

// ===========================================================================
// 5. Variadic functions / splat (*args)
// ===========================================================================

func TestVariadicFunction(t *testing.T) {
	result := evalInput(t, `
def sum_all(*nums)
  total = 0
  for n in nums
    total += n
  end
  total
end
sum_all(1, 2, 3, 4, 5)`)
	assertInteger(t, result, 15)
}

func TestVariadicFunctionWithRequired(t *testing.T) {
	result := evalInput(t, `
def log(level, *messages)
  len(messages)
end
log("INFO", "server started", "port 8080", "ready")`)
	assertInteger(t, result, 3)
}

func TestVariadicFunctionEmpty(t *testing.T) {
	result := evalInput(t, `
def my_func(*args)
  len(args)
end
my_func()`)
	assertInteger(t, result, 0)
}

func TestVariadicFunctionSingleArg(t *testing.T) {
	result := evalInput(t, `
def echo(*args)
  args[0]
end
echo("hello")`)
	assertString(t, result, "hello")
}

// ===========================================================================
// 6. Const declarations
// ===========================================================================

func TestConstDeclaration(t *testing.T) {
	result := evalInput(t, `
const PI = 3.14
PI`)
	assertFloat(t, result, 3.14)
}

func TestConstDeclarationString(t *testing.T) {
	result := evalInput(t, `
const GREETING = "hello"
GREETING`)
	assertString(t, result, "hello")
}

func TestConstReassignError(t *testing.T) {
	result := evalInput(t, `
const X = 10
X = 20`)
	assertError(t, result, "cannot reassign constant")
}

func TestConstWithType(t *testing.T) {
	result := evalInput(t, `
const MAX: int = 100
MAX`)
	assertInteger(t, result, 100)
}

// ===========================================================================
// 7. String * repetition and ** power operator
// ===========================================================================

func TestStringRepetition(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"basic repeat", `"ha" * 3`, "hahaha"},
		{"single char repeat", `"x" * 5`, "xxxxx"},
		{"repeat once", `"abc" * 1`, "abc"},
		{"repeat zero", `"abc" * 0`, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalInput(t, tt.input)
			assertString(t, result, tt.expected)
		})
	}
}

func TestPowerOperatorInt(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{"2^10", `2 ** 10`, 1024},
		{"3^3", `3 ** 3`, 27},
		{"5^0", `5 ** 0`, 1},
		{"10^1", `10 ** 1`, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalInput(t, tt.input)
			assertInteger(t, result, tt.expected)
		})
	}
}

func TestPowerOperatorFloat(t *testing.T) {
	result := evalInput(t, `2.0 ** 0.5`)
	floatObj, ok := result.(*Float)
	if !ok {
		t.Fatalf("expected Float, got %T (%+v)", result, result)
	}
	// sqrt(2) ~ 1.4142
	if floatObj.Value < 1.41 || floatObj.Value > 1.42 {
		t.Errorf("expected ~1.4142, got %f", floatObj.Value)
	}
}

func TestPowerOperatorMixed(t *testing.T) {
	result := evalInput(t, `2 ** 0.5`)
	floatObj, ok := result.(*Float)
	if !ok {
		t.Fatalf("expected Float, got %T (%+v)", result, result)
	}
	if floatObj.Value < 1.41 || floatObj.Value > 1.42 {
		t.Errorf("expected ~1.4142, got %f", floatObj.Value)
	}
}

func TestPowerOperatorPrecedence(t *testing.T) {
	// ** should bind tighter than *
	result := evalInput(t, `2 * 3 ** 2`)
	assertInteger(t, result, 18)
}

// ===========================================================================
// 8. Multi-line strings / heredocs
// ===========================================================================

func TestMultiLineString(t *testing.T) {
	result := evalInput(t, "x = \"\"\"hello\nworld\"\"\"\nx")
	assertString(t, result, "hello\nworld")
}

func TestMultiLineStringPreservesNewlines(t *testing.T) {
	result := evalInput(t, "x = \"\"\"line1\nline2\nline3\"\"\"\nlen(split(x, \"\\n\"))")
	assertInteger(t, result, 3)
}

func TestMultiLineStringEmpty(t *testing.T) {
	result := evalInput(t, "x = \"\"\"\"\"\"\nx")
	assertString(t, result, "")
}

// ===========================================================================
// 9. delete/remove_at for hashes and arrays
// ===========================================================================

func TestDeleteHashKey(t *testing.T) {
	result := evalInput(t, `
h = {"a": 1, "b": 2, "c": 3}
delete(h, "b")
len(keys(h))`)
	assertInteger(t, result, 2)
}

func TestDeleteHashKeyReturnsValue(t *testing.T) {
	result := evalInput(t, `
h = {"a": 1, "b": 2}
delete(h, "a")`)
	assertInteger(t, result, 1)
}

func TestDeleteHashKeyMissing(t *testing.T) {
	result := evalInput(t, `
h = {"a": 1}
delete(h, "z")`)
	assertNil(t, result)
}

func TestRemoveAtArray(t *testing.T) {
	result := evalInput(t, `
arr = [10, 20, 30, 40]
remove_at(arr, 1)
len(arr)`)
	assertInteger(t, result, 3)
}

func TestRemoveAtArrayReturnsValue(t *testing.T) {
	result := evalInput(t, `
arr = [10, 20, 30]
remove_at(arr, 0)`)
	assertInteger(t, result, 10)
}

func TestRemoveAtArrayNegativeIndex(t *testing.T) {
	result := evalInput(t, `
arr = [10, 20, 30]
remove_at(arr, -1)`)
	assertInteger(t, result, 30)
}

func TestRemoveAtArrayCheckContents(t *testing.T) {
	result := evalInput(t, `
arr = ["a", "b", "c", "d"]
remove_at(arr, 1)
arr[1]`)
	assertString(t, result, "c")
}

// ===========================================================================
// 10. Postfix if/unless
// ===========================================================================

func TestPostfixIf(t *testing.T) {
	result := evalInput(t, `
x = 10
y = "yes" if x > 5
y`)
	assertString(t, result, "yes")
}

func TestPostfixIfFalse(t *testing.T) {
	result := evalInput(t, `
x = 2
y = "yes" if x > 5
y`)
	assertNil(t, result)
}

func TestPostfixUnless(t *testing.T) {
	result := evalInput(t, `
x = 10
y = "ok" unless x < 0
y`)
	assertString(t, result, "ok")
}

func TestPostfixUnlessFalse(t *testing.T) {
	result := evalInput(t, `
x = -5
y = "ok" unless x < 0
y`)
	assertNil(t, result)
}

func TestPostfixIfWithReturn(t *testing.T) {
	result := evalInput(t, `
def check(n)
  return "negative" if n < 0
  return "zero" if n == 0
  "positive"
end
check(-5)`)
	assertString(t, result, "negative")
}

func TestPostfixIfWithReturnZero(t *testing.T) {
	result := evalInput(t, `
def check(n)
  return "negative" if n < 0
  return "zero" if n == 0
  "positive"
end
check(0)`)
	assertString(t, result, "zero")
}

func TestPostfixIfWithReturnPositive(t *testing.T) {
	result := evalInput(t, `
def check(n)
  return "negative" if n < 0
  return "zero" if n == 0
  "positive"
end
check(42)`)
	assertString(t, result, "positive")
}

// ===========================================================================
// Integration tests combining multiple new features
// ===========================================================================

func TestIntegrationStringComparisonSort(t *testing.T) {
	// Use string comparison in a manual sort check
	result := evalInput(t, `
a = "banana"
b = "apple"
if a > b
  a
else
  b
end`)
	assertString(t, result, "banana")
}

func TestIntegrationNilCoalescingWithHash(t *testing.T) {
	result := evalInput(t, `
config = {"host": "localhost"}
port = config["port"] ?? 8080
port`)
	assertInteger(t, result, 8080)
}

func TestIntegrationConstWithPower(t *testing.T) {
	result := evalInput(t, `
const BASE = 2
const BITS = 8
BASE ** BITS`)
	assertInteger(t, result, 256)
}

func TestIntegrationVariadicWithStringRepeat(t *testing.T) {
	result := evalInput(t, `
def separator(*chars)
  if len(chars) == 0
    "-" * 20
  else
    chars[0] * 20
  end
end
len(separator())`)
	assertInteger(t, result, 20)
}

func TestIntegrationHashIterationBuild(t *testing.T) {
	result := evalInput(t, `
data = {"name": "Alice", "age": "30"}
parts = []
for k, v in data
  parts = push(parts, k + ": " + v)
end
len(parts)`)
	assertInteger(t, result, 2)
}

func TestIntegrationDeleteAndIterate(t *testing.T) {
	result := evalInput(t, `
h = {"a": 1, "b": 2, "c": 3}
delete(h, "b")
total = 0
for k, v in h
  total += v
end
total`)
	assertInteger(t, result, 4)
}

func TestNilCoalescingStringIndex(t *testing.T) {
	// Combine string indexing with nil coalescing
	result := evalInput(t, `
s = "hello"
ch = s[0]
ch ?? "none"`)
	assertString(t, result, "h")
}

// Verify error message quality
func TestConstReassignErrorMessage(t *testing.T) {
	result := evalInput(t, `
const NAME = "Vibe"
NAME = "Other"`)
	errObj, ok := result.(*Error)
	if !ok {
		t.Fatalf("expected Error, got %T (%s)", result, result.Inspect())
	}
	if !strings.Contains(errObj.Message, "NAME") || !strings.Contains(errObj.Message, "constant") {
		t.Errorf("error should mention variable name and 'constant', got: %s", errObj.Message)
	}
}
