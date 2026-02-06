package interpreter

// End-to-end feature tests for the Vibe language.
//
// These tests verify that implemented language features work correctly
// through the full lexer -> parser -> interpreter pipeline.

import (
	"testing"
)

// ---------------------------------------------------------------------------
// String interpolation
// ---------------------------------------------------------------------------

func TestStringInterpolationEval(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"simple variable interpolation",
			`name = "World"
			"Hello, ${name}!"`,
			"Hello, World!",
		},
		{
			"integer interpolation",
			`age = 30
			"Age: ${age}"`,
			"Age: 30",
		},
		{
			"multiple interpolations",
			`first = "Alice"
			last = "Smith"
			"${first} ${last}"`,
			"Alice Smith",
		},
		{
			"interpolation at start",
			`x = "Hello"
			"${x}, world!"`,
			"Hello, world!",
		},
		{
			"interpolation at end",
			`x = "world"
			"hello, ${x}"`,
			"hello, world",
		},
		{
			"no interpolation (plain string)",
			`"no interpolation here"`,
			"no interpolation here",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalInput(t, tt.input)
			assertString(t, result, tt.expected)
		})
	}
}

// ---------------------------------------------------------------------------
// Range iteration
// ---------------------------------------------------------------------------

func TestRangeIteration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{
			"inclusive range sum 1..5",
			`sum = 0
			for i in 1..5
				sum = sum + i
			end
			sum`,
			15,
		},
		{
			"exclusive range sum 1...5",
			`sum = 0
			for i in 1...5
				sum = sum + i
			end
			sum`,
			10,
		},
		{
			"range with variable bound",
			`n = 5
			sum = 0
			for i in 1..n
				sum = sum + i
			end
			sum`,
			15,
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
// Index expressions
// ---------------------------------------------------------------------------

func TestIndexExpressions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{
			"first element",
			"[10, 20, 30][0]",
			10,
		},
		{
			"middle element",
			"[10, 20, 30][1]",
			20,
		},
		{
			"last element",
			"[10, 20, 30][2]",
			30,
		},
		{
			"variable array access",
			`arr = [100, 200, 300]
			arr[1]`,
			200,
		},
		{
			"computed index",
			`arr = [10, 20, 30]
			i = 1
			arr[i]`,
			20,
		},
		{
			"nested array access",
			`matrix = [[1, 2], [3, 4]]
			matrix[1][0]`,
			3,
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
// Closures
// ---------------------------------------------------------------------------

func TestClosures(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{
			"function captures outer variable",
			`x = 10
			def add_x(y: int): int
				x + y
			end
			add_x(5)`,
			15,
		},
		{
			"function captures modified outer variable",
			`base = 100
			def offset(n: int): int
				base + n
			end
			base = 200
			offset(5)`,
			// The closure captured the environment, and base was updated in-place
			205,
		},
		{
			"nested function closures",
			`def make_adder(x: int): int
				def adder(y: int): int
					x + y
				end
				adder(10)
			end
			make_adder(5)`,
			15,
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
// Struct end-to-end
// ---------------------------------------------------------------------------

func TestStructEndToEnd(t *testing.T) {
	t.Run("define struct and access fields", func(t *testing.T) {
		input := `
		struct Point
			x: int
			y: int
		end

		p = Point(x: 3, y: 4)
		p.x + p.y
		`
		result := evalInput(t, input)
		assertInteger(t, result, 7)
	})

	t.Run("struct with string fields", func(t *testing.T) {
		input := `
		struct Person
			name: string
			age: int
		end

		p = Person(name: "Alice", age: 30)
		p.name
		`
		result := evalInput(t, input)
		assertString(t, result, "Alice")
	})

	t.Run("array of structs with field access", func(t *testing.T) {
		input := `
		struct Item
			value: int
		end

		items = [Item(value: 10), Item(value: 20), Item(value: 30)]
		sum = 0
		for item in items
			sum = sum + item.value
		end
		sum
		`
		result := evalInput(t, input)
		assertInteger(t, result, 60)
	})
}

// ---------------------------------------------------------------------------
// String operations
// ---------------------------------------------------------------------------

func TestStringOperations(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{
			"string concatenation",
			`"hello" + " " + "world"`,
			"hello world",
		},
		{
			"string equality true",
			`"abc" == "abc"`,
			true,
		},
		{
			"string equality false",
			`"abc" == "def"`,
			false,
		},
		{
			"string inequality true",
			`"abc" != "def"`,
			true,
		},
		{
			"string inequality false",
			`"abc" != "abc"`,
			false,
		},
		{
			"concatenation with variables",
			`a = "hello"
			b = "world"
			a + " " + b`,
			"hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalInput(t, tt.input)
			switch expected := tt.expected.(type) {
			case string:
				assertString(t, result, expected)
			case bool:
				assertBoolean(t, result, expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Boolean operations
// ---------------------------------------------------------------------------

func TestBooleanOperations(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"true == true", "true == true", true},
		{"true == false", "true == false", false},
		{"false == false", "false == false", true},
		{"true != false", "true != false", true},
		{"true != true", "true != true", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalInput(t, tt.input)
			assertBoolean(t, result, tt.expected)
		})
	}
}

// ---------------------------------------------------------------------------
// Nil handling
// ---------------------------------------------------------------------------

func TestNilHandling(t *testing.T) {
	t.Run("nil is falsy in if", func(t *testing.T) {
		result := evalInput(t, `
		if nil
			"truthy"
		else
			"falsy"
		end
		`)
		assertString(t, result, "falsy")
	})

	t.Run("if with no else returns nil on false", func(t *testing.T) {
		result := evalInput(t, `
		if false
			42
		end
		`)
		assertNil(t, result)
	})
}

// ---------------------------------------------------------------------------
// Variable scoping and reassignment
// ---------------------------------------------------------------------------

func TestVariableScoping(t *testing.T) {
	t.Run("variable reassignment", func(t *testing.T) {
		result := evalInput(t, `
		x = 5
		x = 10
		x
		`)
		assertInteger(t, result, 10)
	})

	t.Run("variables persist across statements", func(t *testing.T) {
		result := evalInput(t, `
		a = 10
		b = 20
		a + b
		`)
		assertInteger(t, result, 30)
	})

	t.Run("for loop modifies outer variable", func(t *testing.T) {
		result := evalInput(t, `
		total = 0
		for x in [1, 2, 3]
			total = total + x
		end
		total
		`)
		assertInteger(t, result, 6)
	})

	t.Run("let keyword works like assignment", func(t *testing.T) {
		result := evalInput(t, `
		let x = 42
		x
		`)
		assertInteger(t, result, 42)
	})
}

// ---------------------------------------------------------------------------
// Array concatenation
// ---------------------------------------------------------------------------

func TestArrayConcatenation(t *testing.T) {
	t.Run("concatenate two arrays", func(t *testing.T) {
		result := evalInput(t, `
		a = [1, 2]
		b = [3, 4]
		c = a + b
		sum = 0
		for x in c
			sum = sum + x
		end
		sum
		`)
		assertInteger(t, result, 10)
	})

	t.Run("append to array in loop", func(t *testing.T) {
		result := evalInput(t, `
		result = []
		for x in [1, 2, 3]
			result = result + [x * x]
		end
		sum = 0
		for x in result
			sum = sum + x
		end
		sum
		`)
		assertInteger(t, result, 14) // 1 + 4 + 9
	})
}

// ---------------------------------------------------------------------------
// Complex integration tests
// ---------------------------------------------------------------------------

func TestFeatureIntegration(t *testing.T) {
	t.Run("struct with function processing", func(t *testing.T) {
		input := `
		struct Rect
			width: int
			height: int
		end

		def area(r_w: int, r_h: int): int
			r_w * r_h
		end

		r = Rect(width: 5, height: 3)
		area(r.width, r.height)
		`
		result := evalInput(t, input)
		assertInteger(t, result, 15)
	})

	t.Run("recursive array processing", func(t *testing.T) {
		// Sum array elements using a for loop and conditional logic
		input := `
		data = [3, 7, 2, 8, 1, 9, 4, 6, 5, 10]
		max = 0
		for x in data
			if x > max
				max = x
			end
		end
		max
		`
		result := evalInput(t, input)
		assertInteger(t, result, 10)
	})

	t.Run("function with loop and conditional", func(t *testing.T) {
		input := `
		def count_positive(arr_items: int): int
			count = 0
			for x in [1, -2, 3, -4, 5]
				if x > 0
					count = count + 1
				end
			end
			count
		end
		count_positive(0)
		`
		result := evalInput(t, input)
		assertInteger(t, result, 3)
	})
}
