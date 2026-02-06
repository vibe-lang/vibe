package parser

// Phase 4b: Parser AST structure tests.
//
// These tests verify that the parser produces the correct AST node types
// and structure for various language constructs.

import (
	"testing"

	"github.com/vibe-lang/vibe/pkg/ast"
	"github.com/vibe-lang/vibe/pkg/lexer"
)

// parseInput is a test helper that parses Vibe source code and returns the program.
func parseInput(t *testing.T, input string) *ast.Program {
	t.Helper()
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			t.Errorf("parser error: %s", err)
		}
		t.FailNow()
	}

	return program
}

// ---------------------------------------------------------------------------
// Infix expression parsing
// ---------------------------------------------------------------------------

func TestInfixExpressionParsing(t *testing.T) {
	tests := []struct {
		input    string
		left     int64
		operator string
		right    int64
	}{
		{"5 + 5", 5, "+", 5},
		{"5 - 5", 5, "-", 5},
		{"5 * 5", 5, "*", 5},
		{"5 / 5", 5, "/", 5},
		{"5 % 3", 5, "%", 3},
		{"5 > 5", 5, ">", 5},
		{"5 < 5", 5, "<", 5},
		{"5 == 5", 5, "==", 5},
		{"5 != 5", 5, "!=", 5},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			program := parseInput(t, tt.input)

			if len(program.Statements) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(program.Statements))
			}

			stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
			if !ok {
				t.Fatalf("expected ExpressionStatement, got %T", program.Statements[0])
			}

			exp, ok := stmt.Expression.(*ast.InfixExpression)
			if !ok {
				t.Fatalf("expected InfixExpression, got %T", stmt.Expression)
			}

			if exp.Operator != tt.operator {
				t.Errorf("expected operator %q, got %q", tt.operator, exp.Operator)
			}

			testIntegerLiteral(t, exp.Left, tt.left)
			testIntegerLiteral(t, exp.Right, tt.right)
		})
	}
}

// ---------------------------------------------------------------------------
// Prefix expression parsing
// ---------------------------------------------------------------------------

func TestPrefixExpressionParsing(t *testing.T) {
	tests := []struct {
		input    string
		operator string
		value    interface{}
	}{
		{"-5", "-", int64(5)},
		{"!true", "!", true},
		{"!false", "!", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			program := parseInput(t, tt.input)

			if len(program.Statements) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(program.Statements))
			}

			stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
			if !ok {
				t.Fatalf("expected ExpressionStatement, got %T", program.Statements[0])
			}

			exp, ok := stmt.Expression.(*ast.PrefixExpression)
			if !ok {
				t.Fatalf("expected PrefixExpression, got %T", stmt.Expression)
			}

			if exp.Operator != tt.operator {
				t.Errorf("expected operator %q, got %q", tt.operator, exp.Operator)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Operator precedence
// ---------------------------------------------------------------------------

func TestOperatorPrecedence(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"-a * b", "((-a) * b)"},
		{"a + b + c", "((a + b) + c)"},
		{"a + b * c", "(a + (b * c))"},
		{"a * b + c", "((a * b) + c)"},
		{"a + b * c + d", "((a + (b * c)) + d)"},
		{"5 > 4 == 3 < 4", "((5 > 4) == (3 < 4))"},
		{"3 + 4 * 5 == 3 * 1 + 4 * 5", "((3 + (4 * 5)) == ((3 * 1) + (4 * 5)))"},
		{"1 + (2 + 3) + 4", "((1 + (2 + 3)) + 4)"},
		{"(5 + 5) * 2", "((5 + 5) * 2)"},
		{"-(5 + 5)", "(-(5 + 5))"},
		{"!(true == true)", "(!(true == true))"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			program := parseInput(t, tt.input)

			if len(program.Statements) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(program.Statements))
			}

			actual := program.Statements[0].String()
			if actual != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, actual)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// If expression parsing
// ---------------------------------------------------------------------------

func TestIfExpressionParsing(t *testing.T) {
	t.Run("simple if", func(t *testing.T) {
		program := parseInput(t, `if x < y
			x
		end`)

		if len(program.Statements) != 1 {
			t.Fatalf("expected 1 statement, got %d", len(program.Statements))
		}

		stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
		if !ok {
			t.Fatalf("expected ExpressionStatement, got %T", program.Statements[0])
		}

		ifExp, ok := stmt.Expression.(*ast.IfExpression)
		if !ok {
			t.Fatalf("expected IfExpression, got %T", stmt.Expression)
		}

		if ifExp.Consequence == nil {
			t.Fatal("consequence is nil")
		}

		if len(ifExp.Consequence.Statements) != 1 {
			t.Fatalf("expected 1 consequence statement, got %d", len(ifExp.Consequence.Statements))
		}

		if ifExp.Alternative != nil {
			t.Fatalf("expected no alternative, got %+v", ifExp.Alternative)
		}
	})

	t.Run("if with else", func(t *testing.T) {
		program := parseInput(t, `if x < y
			x
		else
			y
		end`)

		stmt := program.Statements[0].(*ast.ExpressionStatement)
		ifExp := stmt.Expression.(*ast.IfExpression)

		if ifExp.Consequence == nil {
			t.Fatal("consequence is nil")
		}

		if ifExp.Alternative == nil {
			t.Fatal("alternative is nil")
		}

		if len(ifExp.Alternative.Statements) != 1 {
			t.Fatalf("expected 1 alternative statement, got %d", len(ifExp.Alternative.Statements))
		}
	})
}

// ---------------------------------------------------------------------------
// Return statement parsing
// ---------------------------------------------------------------------------

func TestReturnStatementParsing(t *testing.T) {
	program := parseInput(t, `return 5`)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ast.ReturnStatement)
	if !ok {
		t.Fatalf("expected ReturnStatement, got %T", program.Statements[0])
	}

	if stmt.TokenLiteral() != "return" {
		t.Errorf("expected token literal 'return', got %q", stmt.TokenLiteral())
	}
}

// ---------------------------------------------------------------------------
// For loop parsing
// ---------------------------------------------------------------------------

func TestForLoopParsing(t *testing.T) {
	program := parseInput(t, `for i in arr
		x = x + i
	end`)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	forLoop, ok := program.Statements[0].(*ast.ForLoop)
	if !ok {
		t.Fatalf("expected ForLoop, got %T", program.Statements[0])
	}

	if forLoop.Iterator.Value != "i" {
		t.Errorf("expected iterator 'i', got %q", forLoop.Iterator.Value)
	}

	if forLoop.Body == nil {
		t.Fatal("for loop body is nil")
	}

	if len(forLoop.Body.Statements) != 1 {
		t.Fatalf("expected 1 body statement, got %d", len(forLoop.Body.Statements))
	}
}

// ---------------------------------------------------------------------------
// Function call expression parsing
// ---------------------------------------------------------------------------

func TestCallExpressionParsing(t *testing.T) {
	program := parseInput(t, `add(1, 2)`)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt := program.Statements[0].(*ast.ExpressionStatement)

	callExp, ok := stmt.Expression.(*ast.CallExpression)
	if !ok {
		t.Fatalf("expected CallExpression, got %T", stmt.Expression)
	}

	if callExp.Function.String() != "add" {
		t.Errorf("expected function name 'add', got %q", callExp.Function.String())
	}

	if len(callExp.Arguments) != 2 {
		t.Fatalf("expected 2 arguments, got %d", len(callExp.Arguments))
	}

	testIntegerLiteral(t, callExp.Arguments[0], 1)
	testIntegerLiteral(t, callExp.Arguments[1], 2)
}

// ---------------------------------------------------------------------------
// Dot expression parsing
// ---------------------------------------------------------------------------

func TestDotExpressionParsing(t *testing.T) {
	program := parseInput(t, `person.name`)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt := program.Statements[0].(*ast.ExpressionStatement)

	dotExp, ok := stmt.Expression.(*ast.DotExpression)
	if !ok {
		t.Fatalf("expected DotExpression, got %T", stmt.Expression)
	}

	if dotExp.Left.String() != "person" {
		t.Errorf("expected left 'person', got %q", dotExp.Left.String())
	}

	if dotExp.Field.Value != "name" {
		t.Errorf("expected field 'name', got %q", dotExp.Field.Value)
	}
}

// ---------------------------------------------------------------------------
// Index expression parsing
// ---------------------------------------------------------------------------

func TestIndexExpressionParsing(t *testing.T) {
	program := parseInput(t, `arr[0]`)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt := program.Statements[0].(*ast.ExpressionStatement)

	indexExp, ok := stmt.Expression.(*ast.IndexExpression)
	if !ok {
		t.Fatalf("expected IndexExpression, got %T", stmt.Expression)
	}

	if indexExp.Left.String() != "arr" {
		t.Errorf("expected left 'arr', got %q", indexExp.Left.String())
	}

	testIntegerLiteral(t, indexExp.Index, 0)
}

// ---------------------------------------------------------------------------
// Grouped expression parsing
// ---------------------------------------------------------------------------

func TestGroupedExpressionParsing(t *testing.T) {
	program := parseInput(t, `(1 + 2) * 3`)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt := program.Statements[0].(*ast.ExpressionStatement)

	infix, ok := stmt.Expression.(*ast.InfixExpression)
	if !ok {
		t.Fatalf("expected InfixExpression, got %T", stmt.Expression)
	}

	if infix.Operator != "*" {
		t.Errorf("expected operator *, got %q", infix.Operator)
	}

	// Left side should be (1 + 2) which is another InfixExpression
	leftInfix, ok := infix.Left.(*ast.InfixExpression)
	if !ok {
		t.Fatalf("expected left to be InfixExpression, got %T", infix.Left)
	}

	if leftInfix.Operator != "+" {
		t.Errorf("expected left operator +, got %q", leftInfix.Operator)
	}

	testIntegerLiteral(t, leftInfix.Left, 1)
	testIntegerLiteral(t, leftInfix.Right, 2)
	testIntegerLiteral(t, infix.Right, 3)
}

// ---------------------------------------------------------------------------
// Boolean literal parsing
// ---------------------------------------------------------------------------

func TestBooleanLiteralParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"false", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			program := parseInput(t, tt.input)

			if len(program.Statements) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(program.Statements))
			}

			stmt := program.Statements[0].(*ast.ExpressionStatement)

			boolLit, ok := stmt.Expression.(*ast.BooleanLiteral)
			if !ok {
				t.Fatalf("expected BooleanLiteral, got %T", stmt.Expression)
			}

			if boolLit.Value != tt.expected {
				t.Errorf("expected %t, got %t", tt.expected, boolLit.Value)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Nil literal parsing
// ---------------------------------------------------------------------------

func TestNilLiteralParsing(t *testing.T) {
	program := parseInput(t, `nil`)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt := program.Statements[0].(*ast.ExpressionStatement)

	_, ok := stmt.Expression.(*ast.NilLiteral)
	if !ok {
		t.Fatalf("expected NilLiteral, got %T", stmt.Expression)
	}
}

// ---------------------------------------------------------------------------
// Range expression parsing (infix .. and ...)
// ---------------------------------------------------------------------------

func TestRangeInfixExpressionParsing(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		exclusive bool
	}{
		{"inclusive range", "1..5", false},
		{"exclusive range", "1...5", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program := parseInput(t, tt.input)

			if len(program.Statements) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(program.Statements))
			}

			stmt := program.Statements[0].(*ast.ExpressionStatement)

			rangeExp, ok := stmt.Expression.(*ast.RangeExpression)
			if !ok {
				t.Fatalf("expected RangeExpression, got %T", stmt.Expression)
			}

			if rangeExp.Exclusive != tt.exclusive {
				t.Errorf("expected exclusive=%t, got %t", tt.exclusive, rangeExp.Exclusive)
			}

			testIntegerLiteral(t, rangeExp.Start, 1)
			testIntegerLiteral(t, rangeExp.End, 5)
		})
	}
}

// ---------------------------------------------------------------------------
// Parser error detection
// ---------------------------------------------------------------------------

func TestParserErrorDetection(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"leading equals", "= 5"},
		{"missing end in for", "for i in arr\n  x"},
		{"missing rparen", "add(1, 2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			p.ParseProgram()

			if len(p.Errors()) == 0 {
				t.Errorf("expected parser errors for input %q, got none", tt.input)
			}
		})
	}
}

// testIntegerLiteral is defined in parser_test.go and shared across test files.
