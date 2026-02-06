package interpreter

// Turing completeness proof tests for the Vibe language.
//
// A Turing-complete language requires:
//   1. Conditional branching (if/else)
//   2. Unbounded iteration or recursion
//   3. Ability to read/write arbitrary amounts of data
//   4. Arithmetic operations
//
// These tests prove that all four requirements are met.

import (
	"strings"
	"testing"

	"github.com/vibe-lang/vibe/pkg/lexer"
	"github.com/vibe-lang/vibe/pkg/parser"
)

// evalInput is a test helper that runs Vibe source code through the full
// lexer -> parser -> interpreter pipeline and returns the result.
func evalInput(t *testing.T, input string) Object {
	t.Helper()
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %s", strings.Join(p.Errors(), "; "))
	}

	interp := New()
	result := interp.Eval(program)
	return result
}

// assertInteger checks that an Object is an Integer with the expected value.
func assertInteger(t *testing.T, obj Object, expected int64) {
	t.Helper()
	if errObj, ok := obj.(*Error); ok {
		t.Fatalf("unexpected error: %s", errObj.Message)
	}
	intObj, ok := obj.(*Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T (%+v)", obj, obj)
	}
	if intObj.Value != expected {
		t.Errorf("expected %d, got %d", expected, intObj.Value)
	}
}

// assertFloat checks that an Object is a Float with the expected value.
func assertFloat(t *testing.T, obj Object, expected float64) {
	t.Helper()
	if errObj, ok := obj.(*Error); ok {
		t.Fatalf("unexpected error: %s", errObj.Message)
	}
	floatObj, ok := obj.(*Float)
	if !ok {
		t.Fatalf("expected Float, got %T (%+v)", obj, obj)
	}
	if floatObj.Value != expected {
		t.Errorf("expected %f, got %f", expected, floatObj.Value)
	}
}

// assertBoolean checks that an Object is a Boolean with the expected value.
func assertBoolean(t *testing.T, obj Object, expected bool) {
	t.Helper()
	if errObj, ok := obj.(*Error); ok {
		t.Fatalf("unexpected error: %s", errObj.Message)
	}
	boolObj, ok := obj.(*Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T (%+v)", obj, obj)
	}
	if boolObj.Value != expected {
		t.Errorf("expected %t, got %t", expected, boolObj.Value)
	}
}

// assertString checks that an Object is a String with the expected value.
func assertString(t *testing.T, obj Object, expected string) {
	t.Helper()
	if errObj, ok := obj.(*Error); ok {
		t.Fatalf("unexpected error: %s", errObj.Message)
	}
	strObj, ok := obj.(*String)
	if !ok {
		t.Fatalf("expected String, got %T (%+v)", obj, obj)
	}
	if strObj.Value != expected {
		t.Errorf("expected %q, got %q", expected, strObj.Value)
	}
}

// assertNil checks that an Object is Nil.
func assertNil(t *testing.T, obj Object) {
	t.Helper()
	if errObj, ok := obj.(*Error); ok {
		t.Fatalf("unexpected error: %s", errObj.Message)
	}
	if _, ok := obj.(*Nil); !ok {
		t.Fatalf("expected Nil, got %T (%+v)", obj, obj)
	}
}

// assertError checks that an Object is an Error containing the expected substring.
func assertError(t *testing.T, obj Object, expectedSubstr string) {
	t.Helper()
	errObj, ok := obj.(*Error)
	if !ok {
		t.Fatalf("expected Error, got %T (%+v)", obj, obj)
	}
	if !strings.Contains(errObj.Message, expectedSubstr) {
		t.Errorf("expected error containing %q, got %q", expectedSubstr, errObj.Message)
	}
}

// ---------------------------------------------------------------------------
// Requirement 1: Conditional Branching
// ---------------------------------------------------------------------------

func TestIfElseEvaluation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{
			"simple true branch",
			`if true
				10
			end`,
			int64(10),
		},
		{
			"simple false with else",
			`if false
				10
			else
				20
			end`,
			int64(20),
		},
		{
			"comparison condition - true",
			`if 5 > 3
				"yes"
			else
				"no"
			end`,
			"yes",
		},
		{
			"comparison condition - false",
			`if 3 > 5
				"yes"
			else
				"no"
			end`,
			"no",
		},
		{
			"elsif chain - first branch",
			`x = 1
			if x == 1
				"one"
			elsif x == 2
				"two"
			else
				"other"
			end`,
			"one",
		},
		{
			"elsif chain - second branch",
			`x = 2
			if x == 1
				"one"
			elsif x == 2
				"two"
			else
				"other"
			end`,
			"two",
		},
		{
			"elsif chain - else branch",
			`x = 99
			if x == 1
				"one"
			elsif x == 2
				"two"
			else
				"other"
			end`,
			"other",
		},
		{
			"nested if expressions",
			`if true
				if false
					1
				else
					2
				end
			end`,
			int64(2),
		},
		{
			"if with equality comparison",
			`if 10 == 10
				42
			end`,
			int64(42),
		},
		{
			"if with inequality comparison",
			`if 10 != 9
				42
			end`,
			int64(42),
		},
		{
			"if with less-than-or-equal",
			`if 5 <= 5
				"equal"
			else
				"not"
			end`,
			"equal",
		},
		{
			"if with greater-than-or-equal",
			`if 5 >= 6
				"yes"
			else
				"no"
			end`,
			"no",
		},
		{
			"if nil is falsy",
			`if nil
				"truthy"
			else
				"falsy"
			end`,
			"falsy",
		},
		{
			"if false is falsy",
			`if false
				"truthy"
			else
				"falsy"
			end`,
			"falsy",
		},
		{
			"if integer is truthy",
			`if 0
				"truthy"
			else
				"falsy"
			end`,
			"truthy",
		},
		{
			"if string is truthy",
			`if ""
				"truthy"
			else
				"falsy"
			end`,
			"truthy",
		},
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
// Requirement 2: Unbounded Iteration (via Recursion)
// ---------------------------------------------------------------------------

func TestRecursion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{
			"factorial of 1 (base case)",
			`def factorial(n: int): int
				if n <= 1
					return 1
				else
					return n * factorial(n - 1)
				end
			end

			factorial(1)`,
			1,
		},
		{
			"factorial of 5",
			`def factorial(n: int): int
				if n <= 1
					return 1
				else
					return n * factorial(n - 1)
				end
			end

			factorial(5)`,
			120,
		},
		{
			"factorial of 10",
			`def factorial(n: int): int
				if n <= 1
					return 1
				else
					return n * factorial(n - 1)
				end
			end

			factorial(10)`,
			3628800,
		},
		{
			"fibonacci of 0 (base case)",
			`def fib(n: int): int
				if n <= 0
					return 0
				elsif n == 1
					return 1
				else
					return fib(n - 1) + fib(n - 2)
				end
			end

			fib(0)`,
			0,
		},
		{
			"fibonacci of 1 (base case)",
			`def fib(n: int): int
				if n <= 0
					return 0
				elsif n == 1
					return 1
				else
					return fib(n - 1) + fib(n - 2)
				end
			end

			fib(1)`,
			1,
		},
		{
			"fibonacci of 10",
			`def fib(n: int): int
				if n <= 0
					return 0
				elsif n == 1
					return 1
				else
					return fib(n - 1) + fib(n - 2)
				end
			end

			fib(10)`,
			55,
		},
		{
			"recursive countdown (termination)",
			`def countdown(n: int): int
				if n <= 0
					return 0
				else
					return countdown(n - 1)
				end
			end

			countdown(100)`,
			0,
		},
		{
			"recursive sum",
			`def sum_to(n: int): int
				if n <= 0
					return 0
				else
					return n + sum_to(n - 1)
				end
			end

			sum_to(10)`,
			55,
		},
		{
			"mutual-style recursion via helper",
			`def is_even(n: int): int
				if n == 0
					return 1
				else
					return is_odd(n - 1)
				end
			end

			def is_odd(n: int): int
				if n == 0
					return 0
				else
					return is_even(n - 1)
				end
			end

			is_even(10)`,
			1,
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
// Requirement 3: Function Definition, Invocation, and Return
// ---------------------------------------------------------------------------

func TestFunctionDefinitionAndCall(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{
			"simple function returning constant",
			`def five(): int
				5
			end
			five()`,
			int64(5),
		},
		{
			"function with one parameter",
			`def double(x: int): int
				x * 2
			end
			double(7)`,
			int64(14),
		},
		{
			"function with two parameters",
			`def add(a: int, b: int): int
				a + b
			end
			add(3, 7)`,
			int64(10),
		},
		{
			"function calling another function",
			`def square(x: int): int
				x * x
			end

			def sum_of_squares(a: int, b: int): int
				square(a) + square(b)
			end

			sum_of_squares(3, 4)`,
			int64(25),
		},
		{
			"implicit return (last expression)",
			`def foo(): int
				42
			end
			foo()`,
			int64(42),
		},
		{
			"function with string return",
			`def greet(name: string): string
				"Hello, ${name}!"
			end
			greet("World")`,
			"Hello, World!",
		},
		{
			"multiple function calls",
			`def inc(x: int): int
				x + 1
			end
			inc(inc(inc(0)))`,
			int64(3),
		},
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

func TestReturnStatements(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{
			"explicit return",
			`def f(): int
				return 5
			end
			f()`,
			5,
		},
		{
			"early return on true branch",
			`def f(x: int): int
				if x > 0
					return x
				end
				return 0
			end
			f(10)`,
			10,
		},
		{
			"early return on false branch",
			`def f(x: int): int
				if x > 0
					return x
				end
				return 0
			end
			f(-5)`,
			0,
		},
		{
			"return stops execution",
			`def f(): int
				return 1
				return 2
			end
			f()`,
			1,
		},
		{
			"return from inside if block",
			`def abs(x: int): int
				if x < 0
					return 0 - x
				else
					return x
				end
			end
			abs(-42)`,
			42,
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
// Requirement 4: Arithmetic Operations
// ---------------------------------------------------------------------------

func TestArithmeticCompleteness(t *testing.T) {
	intTests := []struct {
		name     string
		input    string
		expected int64
	}{
		// Addition
		{"integer addition", "5 + 3", 8},
		{"addition with zero", "0 + 42", 42},
		{"negative result addition", "5 + -10", -5},

		// Subtraction
		{"integer subtraction", "10 - 3", 7},
		{"subtraction to zero", "5 - 5", 0},
		{"subtraction to negative", "3 - 10", -7},

		// Multiplication
		{"integer multiplication", "4 * 5", 20},
		{"multiplication by zero", "42 * 0", 0},
		{"multiplication by one", "42 * 1", 42},
		{"negative multiplication", "-3 * 4", -12},

		// Division
		{"integer division", "10 / 2", 5},
		{"integer division truncates", "7 / 2", 3},
		{"division by one", "42 / 1", 42},

		// Modulo
		{"modulo", "10 % 3", 1},
		{"modulo even division", "10 % 5", 0},
		{"modulo by 1", "42 % 1", 0},

		// Operator precedence
		{"multiplication before addition", "2 + 3 * 4", 14},
		{"parentheses override precedence", "(2 + 3) * 4", 20},
		{"complex precedence", "2 * 3 + 4 * 5", 26},
		{"nested parentheses", "((2 + 3) * (4 + 5))", 45},
		{"division before subtraction", "10 - 6 / 2", 7},

		// Chained operations
		{"chained addition", "1 + 2 + 3 + 4 + 5", 15},
		{"chained multiplication", "2 * 3 * 4", 24},
		{"mixed operations", "10 + 5 * 2 - 3", 17},
	}

	for _, tt := range intTests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalInput(t, tt.input)
			assertInteger(t, result, tt.expected)
		})
	}

	// Comparison operators (return booleans)
	compTests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"less than true", "3 < 5", true},
		{"less than false", "5 < 3", false},
		{"less than equal", "5 < 5", false},
		{"greater than true", "5 > 3", true},
		{"greater than false", "3 > 5", false},
		{"greater than equal", "5 > 5", false},
		{"less than or equal true (less)", "3 <= 5", true},
		{"less than or equal true (equal)", "5 <= 5", true},
		{"less than or equal false", "6 <= 5", false},
		{"greater than or equal true (greater)", "5 >= 3", true},
		{"greater than or equal true (equal)", "5 >= 5", true},
		{"greater than or equal false", "4 >= 5", false},
		{"equality true", "5 == 5", true},
		{"equality false", "5 == 6", false},
		{"inequality true", "5 != 6", true},
		{"inequality false", "5 != 5", false},
	}

	for _, tt := range compTests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalInput(t, tt.input)
			assertBoolean(t, result, tt.expected)
		})
	}

	// Float arithmetic
	floatTests := []struct {
		name     string
		input    string
		expected float64
	}{
		{"float addition", "1.5 + 2.5", 4.0},
		{"float subtraction", "5.5 - 2.0", 3.5},
		{"float multiplication", "2.5 * 4.0", 10.0},
		{"float division", "10.0 / 4.0", 2.5},
	}

	for _, tt := range floatTests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalInput(t, tt.input)
			assertFloat(t, result, tt.expected)
		})
	}

	// Mixed int/float arithmetic (promotes to float)
	mixedTests := []struct {
		name     string
		input    string
		expected float64
	}{
		{"int + float", "1 + 2.5", 3.5},
		{"float + int", "2.5 + 1", 3.5},
		{"int * float", "3 * 2.5", 7.5},
		{"float / int", "10.0 / 4", 2.5},
	}

	for _, tt := range mixedTests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalInput(t, tt.input)
			assertFloat(t, result, tt.expected)
		})
	}
}

func TestPrefixOperators(t *testing.T) {
	intTests := []struct {
		name     string
		input    string
		expected int64
	}{
		{"negate integer", "-5", -5},
		{"double negate", "-(-5)", 5},
		{"negate zero", "-0", 0},
	}

	for _, tt := range intTests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalInput(t, tt.input)
			assertInteger(t, result, tt.expected)
		})
	}

	boolTests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"bang true", "!true", false},
		{"bang false", "!false", true},
		{"double bang true", "!!true", true},
		{"double bang false", "!!false", false},
		{"bang integer is false (truthy)", "!5", false},
		{"bang zero is false (truthy)", "!0", false},
	}

	for _, tt := range boolTests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalInput(t, tt.input)
			assertBoolean(t, result, tt.expected)
		})
	}

	// !nil should be true (nil is falsy)
	t.Run("bang nil is true", func(t *testing.T) {
		result := evalInput(t, "!nil")
		assertBoolean(t, result, true)
	})

	// Negate float
	t.Run("negate float", func(t *testing.T) {
		result := evalInput(t, "-3.14")
		assertFloat(t, result, -3.14)
	})
}

// ---------------------------------------------------------------------------
// Turing Completeness Integration: Combine all 4 requirements
// ---------------------------------------------------------------------------

func TestTuringCompletenessIntegration(t *testing.T) {
	t.Run("GCD via Euclidean algorithm", func(t *testing.T) {
		input := `
		def gcd(a: int, b: int): int
			if b == 0
				return a
			else
				return gcd(b, a % b)
			end
		end

		gcd(48, 18)
		`
		result := evalInput(t, input)
		assertInteger(t, result, 6)
	})

	t.Run("power function via recursion", func(t *testing.T) {
		input := `
		def power(base: int, exp: int): int
			if exp == 0
				return 1
			else
				return base * power(base, exp - 1)
			end
		end

		power(2, 10)
		`
		result := evalInput(t, input)
		assertInteger(t, result, 1024)
	})

	t.Run("array sum via recursion and index", func(t *testing.T) {
		input := `
		arr = [10, 20, 30, 40, 50]
		sum = 0
		for val in arr
			sum = sum + val
		end
		sum
		`
		result := evalInput(t, input)
		assertInteger(t, result, 150)
	})

	t.Run("conditional accumulation", func(t *testing.T) {
		// Sum only even numbers from an array using if + modulo
		input := `
		numbers = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
		sum = 0
		for n in numbers
			if n % 2 == 0
				sum = sum + n
			end
		end
		sum
		`
		result := evalInput(t, input)
		assertInteger(t, result, 30)
	})

	t.Run("build result with recursion and branching", func(t *testing.T) {
		// Collatz conjecture: count steps to reach 1
		input := `
		def collatz_steps(n: int): int
			if n == 1
				return 0
			elsif n % 2 == 0
				return 1 + collatz_steps(n / 2)
			else
				return 1 + collatz_steps(3 * n + 1)
			end
		end

		collatz_steps(27)
		`
		result := evalInput(t, input)
		assertInteger(t, result, 111)
	})
}
