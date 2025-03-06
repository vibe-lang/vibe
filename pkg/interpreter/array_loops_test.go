package interpreter

import (
	"testing"

	"github.com/vibe-lang/vibe/pkg/lexer"
	"github.com/vibe-lang/vibe/pkg/parser"
)

// TestArrayForLoops tests the functionality of iterating through arrays with for loops
func TestArrayForLoops(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{
			"Basic array iteration",
			`
			arr = [1, 2, 3, 4, 5]
			sum = 0

			for i in arr
				sum = sum + i
			end

			sum
			`,
			15,
		},
		{
			"Multidimensional array iteration",
			`
			matrix = [[1, 2], [3, 4], [5, 6]]
			sum = 0

			for row in matrix
				for num in row
					sum = sum + num
				end
			end

			sum
			`,
			21,
		},
		{
			"Typed array with structs",
			`
			struct Person
				name: string
				age: int
			end

			people = [
				Person(name: "Alice", age: 25),
				Person(name: "Bob", age: 30),
				Person(name: "Charlie", age: 35)
			]

			total_age = 0

			for person in people
				total_age = total_age + person.age
			end

			total_age
			`,
			90,
		},
		{
			"Array modification during iteration",
			`
			numbers = [1, 2, 3, 4, 5]
			squared = []

			for num in numbers
				squared = squared + [num * num]
			end

			sum = 0
			for sq in squared
				sum = sum + sq
			end

			sum
			`,
			55, // 1 + 4 + 9 + 16 + 25 = 55
		},
		{
			"Nested iteration with index tracking",
			`
			cube = [[[1, 2], [3, 4]], [[5, 6], [7, 8]]]
			sum = 0
			i = 0

			for matrix in cube
				j = 0
				for row in matrix
					k = 0
					for val in row
						mult = i*16 + j*4 + k
						sum = sum + val * mult
						k = k + 1
					end
					j = j + 1
				end
				i = i + 1
			end

			sum
			`,
			524, // Updated expected value based on the actual calculation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				t.Fatalf("Parser errors: %v", p.Errors())
			}

			interpreter := New()
			result := interpreter.Eval(program)

			// Check if result is an error
			if errObj, ok := result.(*Error); ok {
				t.Fatalf("Evaluation error: %s", errObj.Message)
			}

			intObj, ok := result.(*Integer)
			if !ok {
				t.Fatalf("result is not Integer. got=%T (%+v)", result, result)
			}

			if intObj.Value != int64(tt.expected.(int)) {
				t.Errorf("result has wrong value. got=%d, want=%d", intObj.Value, tt.expected)
			}
		})
	}
}

// TestBasicArrayIteration tests a simple for loop iterating over an array
func TestBasicArrayIteration(t *testing.T) {
	input := `
	arr = [1, 2, 3, 4, 5]
	sum = 0

	for i in arr
		sum = sum + i
	end

	sum
	`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	// Print the AST for debugging
	t.Logf("AST: %s", program.String())

	interpreter := New()
	result := interpreter.Eval(program)

	// Check if result is an error
	if errObj, ok := result.(*Error); ok {
		t.Fatalf("Evaluation error: %s", errObj.Message)
	}

	intObj, ok := result.(*Integer)
	if !ok {
		t.Fatalf("result is not Integer. got=%T (%+v)", result, result)
	}

	if intObj.Value != 15 {
		t.Errorf("result has wrong value. got=%d, want=%d", intObj.Value, 15)
	}
}

// TestSimpleExpression tests a very simple expression to make sure the interpreter is working
func TestSimpleExpression(t *testing.T) {
	input := `
	5 + 10
	`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	// Print the AST for debugging
	t.Logf("AST: %s", program.String())

	interpreter := New()
	result := interpreter.Eval(program)

	// Check if result is an error
	if errObj, ok := result.(*Error); ok {
		t.Fatalf("Evaluation error: %s", errObj.Message)
	}

	intObj, ok := result.(*Integer)
	if !ok {
		t.Fatalf("result is not Integer. got=%T (%+v)", result, result)
	}

	if intObj.Value != 15 {
		t.Errorf("result has wrong value. got=%d, want=%d", intObj.Value, 15)
	}
}

// TestNestedArrayIteration tests nested for loops iterating over multidimensional arrays
func TestNestedArrayIteration(t *testing.T) {
	input := `
	matrix = [[1, 2], [3, 4], [5, 6]]
	sum = 0

	for row in matrix
		for num in row
			sum = sum + num
		end
	end

	sum
	`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	// Print the AST for debugging
	t.Logf("AST: %s", program.String())

	interpreter := New()
	result := interpreter.Eval(program)

	// Check if result is an error
	if errObj, ok := result.(*Error); ok {
		t.Fatalf("Evaluation error: %s", errObj.Message)
	}

	intObj, ok := result.(*Integer)
	if !ok {
		t.Fatalf("result is not Integer. got=%T (%+v)", result, result)
	}

	if intObj.Value != 21 {
		t.Errorf("result has wrong value. got=%d, want=%d", intObj.Value, 21)
	}
}

// TestForLoopWithIndexTracking tests a for loop with manual index variables
func TestForLoopWithIndexTracking(t *testing.T) {
	input := `
	cube = [[[1, 2], [3, 4]], [[5, 6], [7, 8]]]
	sum = 0
	i = 0

	for matrix in cube
		j = 0
		for row in matrix
			k = 0
			for val in row
				mult = i*16 + j*4 + k
				sum = sum + val * mult
				k = k + 1
			end
			j = j + 1
		end
		i = i + 1
	end

	sum
	`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	// Print the AST for debugging
	t.Logf("AST: %s", program.String())

	interpreter := New()
	result := interpreter.Eval(program)

	// Check if result is an error
	if errObj, ok := result.(*Error); ok {
		t.Fatalf("Evaluation error: %s", errObj.Message)
	}

	intObj, ok := result.(*Integer)
	if !ok {
		t.Fatalf("result is not Integer. got=%T (%+v)", result, result)
	}

	expected := 524  // Calculated value based on the formula
	if intObj.Value != int64(expected) {
		t.Errorf("result has wrong value. got=%d, want=%d", intObj.Value, expected)
	}
}

// TestStructArrayIteration tests iterating over an array of struct instances
func TestStructArrayIteration(t *testing.T) {
	input := `
	struct Person
		name: string
		age: int
	end

	people = [
		Person(name: "Alice", age: 25),
		Person(name: "Bob", age: 30),
		Person(name: "Charlie", age: 35)
	]

	total_age = 0

	for person in people
		total_age = total_age + person.age
	end

	total_age
	`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	// Print the AST for debugging
	t.Logf("AST: %s", program.String())

	interpreter := New()
	result := interpreter.Eval(program)

	// Check if result is an error
	if errObj, ok := result.(*Error); ok {
		t.Fatalf("Evaluation error: %s", errObj.Message)
	}

	intObj, ok := result.(*Integer)
	if !ok {
		t.Fatalf("result is not Integer. got=%T (%+v)", result, result)
	}

	if intObj.Value != 90 {
		t.Errorf("result has wrong value. got=%d, want=%d", intObj.Value, 90)
	}
}

// TestArrayModificationDuringIteration tests modifying an array during iteration
func TestArrayModificationDuringIteration(t *testing.T) {
	input := `
	numbers = [1, 2, 3, 4, 5]
	squared = []

	for num in numbers
		squared = squared + [num * num]
	end

	sum = 0
	for sq in squared
		sum = sum + sq
	end

	sum
	`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	// Print the AST for debugging
	t.Logf("AST: %s", program.String())

	interpreter := New()
	result := interpreter.Eval(program)

	// Check if result is an error
	if errObj, ok := result.(*Error); ok {
		t.Fatalf("Evaluation error: %s", errObj.Message)
	}

	intObj, ok := result.(*Integer)
	if !ok {
		t.Fatalf("result is not Integer. got=%T (%+v)", result, result)
	}

	if intObj.Value != 55 {
		t.Errorf("result has wrong value. got=%d, want=%d", intObj.Value, 55)
	}
}