package interpreter

// Error condition tests for the Vibe language.
//
// These tests verify that the interpreter produces correct error messages
// for invalid operations, type mismatches, and runtime errors.

import (
	"strings"
	"testing"

	"github.com/vibe-lang/vibe/pkg/lexer"
	"github.com/vibe-lang/vibe/pkg/parser"
)

// evalInputAllowErrors runs Vibe source through the full pipeline,
// returning parser errors as an Error object if present.
func evalInputAllowErrors(input string) Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		return &Error{Message: "parser error: " + strings.Join(p.Errors(), "; ")}
	}

	interp := New()
	return interp.Eval(program)
}

// ---------------------------------------------------------------------------
// Division by zero
// ---------------------------------------------------------------------------

func TestDivisionByZero(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			"integer division by zero",
			"10 / 0",
			"division by zero",
		},
		{
			"integer modulo by zero",
			"10 % 0",
			"division by zero",
		},
		{
			"float division by zero",
			"10.0 / 0.0",
			"division by zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalInputAllowErrors(tt.input)
			assertError(t, result, tt.expectedError)
		})
	}
}

// ---------------------------------------------------------------------------
// Undefined variables
// ---------------------------------------------------------------------------

func TestUndefinedVariable(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			"simple undefined variable",
			"x",
			"identifier not found: x",
		},
		{
			"undefined in expression",
			"x + 5",
			"identifier not found: x",
		},
		{
			"undefined function call",
			"foo()",
			"identifier not found: foo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalInputAllowErrors(tt.input)
			assertError(t, result, tt.expectedError)
		})
	}
}

// ---------------------------------------------------------------------------
// Array index out of bounds
// ---------------------------------------------------------------------------

func TestArrayIndexOutOfBounds(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			"index too large",
			"[1, 2, 3][5]",
			"array index out of bounds",
		},
		{
			"index negative too far",
			"[1, 2, 3][-4]",
			"array index out of bounds",
		},
		{
			"index on empty array",
			"[][0]",
			"array index out of bounds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalInputAllowErrors(tt.input)
			assertError(t, result, tt.expectedError)
		})
	}
}

// ---------------------------------------------------------------------------
// Type mismatch / unknown operator errors
// ---------------------------------------------------------------------------

func TestTypeMismatchErrors(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			"string subtraction",
			`"hello" - "world"`,
			"unknown operator",
		},
		{
			"boolean addition",
			"true + false",
			"unknown operator",
		},
		{
			"negate string",
			`-"hello"`,
			"unknown operator",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalInputAllowErrors(tt.input)
			assertError(t, result, tt.expectedError)
		})
	}
}

// ---------------------------------------------------------------------------
// Index on non-array
// ---------------------------------------------------------------------------

func TestIndexOnNonArray(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			"index on integer",
			"5[0]",
			"index operator not supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalInputAllowErrors(tt.input)
			assertError(t, result, tt.expectedError)
		})
	}
}

// ---------------------------------------------------------------------------
// For loop errors
// ---------------------------------------------------------------------------

func TestForLoopErrors(t *testing.T) {
	t.Run("for loop over non-iterable", func(t *testing.T) {
		result := evalInputAllowErrors(`
		for i in 5
			i
		end
		`)
		assertError(t, result, "must be an array, range, or hash")
	})
}

// ---------------------------------------------------------------------------
// Struct errors
// ---------------------------------------------------------------------------

func TestStructErrors(t *testing.T) {
	t.Run("undefined struct type", func(t *testing.T) {
		result := evalInputAllowErrors(`Foo(x: 1)`)
		assertError(t, result, "undefined type")
	})

	t.Run("undefined struct field", func(t *testing.T) {
		result := evalInputAllowErrors(`
		struct Point
			x: int
			y: int
		end

		p = Point(x: 1, y: 2)
		p.z
		`)
		assertError(t, result, "undefined field")
	})
}

// ---------------------------------------------------------------------------
// Not-a-function errors
// ---------------------------------------------------------------------------

func TestNotAFunctionError(t *testing.T) {
	t.Run("call integer as function", func(t *testing.T) {
		result := evalInputAllowErrors(`
		x = 5
		x(1)
		`)
		assertError(t, result, "not a function")
	})
}

// ---------------------------------------------------------------------------
// Parser error propagation
// ---------------------------------------------------------------------------

func TestParserErrors(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			"unterminated function def",
			`def foo()
				5`,
			"parser error",
		},
		{
			"unexpected assignment",
			`= 5`,
			"parser error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalInputAllowErrors(tt.input)
			assertError(t, result, tt.expectedError)
		})
	}
}
