package parser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/vibe-lang/vibe/pkg/ast"
	"github.com/vibe-lang/vibe/pkg/lexer"
)

func TestVariableStatements(t *testing.T) {
	input := `
x = 5
y = 10
foobar = 838383
`
	l := lexer.New(input)
	p := New(l)

	program := p.ParseProgram()
	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	if len(program.Statements) != 3 {
		t.Fatalf("program.Statements does not contain 3 statements. got=%d",
			len(program.Statements))
	}

	tests := []struct {
		expectedIdentifier string
		expectedValue      int64
	}{
		{"x", 5},
		{"y", 10},
		{"foobar", 838383},
	}

	for i, tt := range tests {
		stmt, ok := program.Statements[i].(*ast.ExpressionStatement)
		if !ok {
			t.Fatalf("program.Statements[%d] is not ast.ExpressionStatement. got=%T",
				i, program.Statements[i])
			continue
		}

		assignment, ok := stmt.Expression.(*ast.AssignmentExpression)
		if !ok {
			t.Fatalf("stmt.Expression is not ast.AssignmentExpression. got=%T",
				stmt.Expression)
		}

		if assignment.Name.Value != tt.expectedIdentifier {
			t.Errorf("assignment.Name.Value not '%s'. got=%s", tt.expectedIdentifier, assignment.Name.Value)
		}

		if !testIntegerLiteral(t, assignment.Value, tt.expectedValue) {
			return
		}
	}
}

func TestArrayLiterals(t *testing.T) {
	tests := []struct {
		input         string
		expectedArray []interface{}
	}{
		{
			`g = []`,
			[]interface{}{},
		},
		{
			`h = [1, 2, 3]`,
			[]interface{}{1, 2, 3},
		},
		{
			`i = ["hello", "world"]`,
			[]interface{}{"hello", "world"},
		},
	}

	for idx, tt := range tests {
		t.Logf("Test #%d: %s", idx, tt.input)
		l := lexer.New(tt.input)
		p := New(l)

		program := p.ParseProgram()
		checkParserErrors(t, p)

		if len(program.Statements) != 1 {
			t.Fatalf("program.Statements does not contain 1 statement. got=%d",
				len(program.Statements))
		}

		stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
		if !ok {
			t.Fatalf("program.Statements[0] is not ast.ExpressionStatement. got=%T",
				program.Statements[0])
		}

		assignment, ok := stmt.Expression.(*ast.AssignmentExpression)
		if !ok {
			t.Fatalf("stmt.Expression is not ast.AssignmentExpression. got=%T",
				stmt.Expression)
		}

		array, ok := assignment.Value.(*ast.ArrayLiteral)
		if !ok {
			t.Fatalf("assignment.Value is not ast.ArrayLiteral. got=%T", assignment.Value)
		}

		if len(array.Elements) != len(tt.expectedArray) {
			t.Fatalf("array.Elements has wrong length. got=%d, want=%d",
				len(array.Elements), len(tt.expectedArray))
		}

		for i, expectedElem := range tt.expectedArray {
			switch expected := expectedElem.(type) {
			case int:
				testIntegerLiteral(t, array.Elements[i], int64(expected))
			case string:
				testStringLiteral(t, array.Elements[i], expected)
			default:
				t.Fatalf("Unsupported expected element type: %T", expected)
			}
		}
	}
}

func TestTypedArrayAssignments(t *testing.T) {
	tests := []struct {
		input            string
		expectedName     string
		expectedType     string
		expectedElements []interface{}
	}{
		{
			`h = [1, 2, 3]`,
			"h",
			"int[]",
			[]interface{}{1, 2, 3},
		},
		{
			`i = ["hello", "world"]`,
			"i",
			"string[]",
			[]interface{}{"hello", "world"},
		},
		{
			`j = [1.5, 3.8, 1.0]`,
			"j",
			"float[]",
			[]interface{}{1.5, 3.8, 1.0},
		},
		{
			`k = []`,
			"k",
			"int[]",
			[]interface{}{},
		},
	}

	for i, tt := range tests {
		t.Logf("Test case %d: %q", i, tt.input)
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		if len(program.Statements) != 1 {
			t.Fatalf("program.Statements does not contain 1 statement. got=%d",
				len(program.Statements))
		}

		stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
		if !ok {
			t.Fatalf("program.Statements[0] is not ast.ExpressionStatement. got=%T",
				program.Statements[0])
		}

		assignment, ok := stmt.Expression.(*ast.AssignmentExpression)
		if !ok {
			t.Fatalf("stmt.Expression is not ast.AssignmentExpression. got=%T",
				stmt.Expression)
		}

		if assignment.Name.Value != tt.expectedName {
			t.Errorf("assignment.Name.Value not '%s'. got=%s", tt.expectedName, assignment.Name.Value)
		}

		array, ok := assignment.Value.(*ast.ArrayLiteral)
		if !ok {
			t.Fatalf("assignment.Value is not ast.ArrayLiteral. got=%T", assignment.Value)
		}

		if len(array.Elements) != len(tt.expectedElements) {
			t.Fatalf("array.Elements has wrong length. got=%d, want=%d",
				len(array.Elements), len(tt.expectedElements))
		}

		for i, expectedElem := range tt.expectedElements {
			switch expected := expectedElem.(type) {
			case int:
				testIntegerLiteral(t, array.Elements[i], int64(expected))
			case float64:
				intLit, isInt := array.Elements[i].(*ast.IntegerLiteral)
				if isInt {
					floatVal := float64(intLit.Value)
					if floatVal != expected {
						t.Errorf("integer value %f does not match expected float %f", floatVal, expected)
					}
				} else {
					testFloatLiteral(t, array.Elements[i], expected)
				}
			case string:
				testStringLiteral(t, array.Elements[i], expected)
			default:
				t.Fatalf("Unsupported expected element type: %T", expected)
			}
		}
	}
}

func TestNestedArrayLiterals(t *testing.T) {
	tests := []struct {
		input         string
		expectedArray [][]interface{}
	}{
		{
			`nested = [[1, 2], [3, 4]]`,
			[][]interface{}{
				{1, 2},
				{3, 4},
			},
		},
		{
			`strings = [["hello", "world"], ["foo", "bar"]]`,
			[][]interface{}{
				{"hello", "world"},
				{"foo", "bar"},
			},
		},
		{
			`mixed = [[], [1, 2]]`,
			[][]interface{}{
				{},
				{1, 2},
			},
		},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		if len(program.Statements) != 1 {
			t.Fatalf("program.Statements does not contain 1 statement. got=%d",
				len(program.Statements))
		}

		stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
		if !ok {
			t.Fatalf("program.Statements[0] is not ast.ExpressionStatement. got=%T",
				program.Statements[0])
		}

		assignment, ok := stmt.Expression.(*ast.AssignmentExpression)
		if !ok {
			t.Fatalf("stmt.Expression is not ast.AssignmentExpression. got=%T",
				stmt.Expression)
		}

		outerArray, ok := assignment.Value.(*ast.ArrayLiteral)
		if !ok {
			t.Fatalf("assignment.Value is not ast.ArrayLiteral. got=%T", assignment.Value)
		}

		if len(outerArray.Elements) != len(tt.expectedArray) {
			t.Fatalf("outerArray.Elements has wrong length. got=%d, want=%d",
				len(outerArray.Elements), len(tt.expectedArray))
		}

		for i, expectedNestedArr := range tt.expectedArray {
			innerArray, ok := outerArray.Elements[i].(*ast.ArrayLiteral)
			if !ok {
				t.Fatalf("outerArray.Elements[%d] is not ast.ArrayLiteral. got=%T",
					i, outerArray.Elements[i])
			}

			if len(innerArray.Elements) != len(expectedNestedArr) {
				t.Fatalf("innerArray.Elements has wrong length. got=%d, want=%d",
					len(innerArray.Elements), len(expectedNestedArr))
			}

			for j, expectedElem := range expectedNestedArr {
				switch expected := expectedElem.(type) {
				case int:
					testIntegerLiteral(t, innerArray.Elements[j], int64(expected))
				case string:
					testStringLiteral(t, innerArray.Elements[j], expected)
				case []interface{}:
					deepArray, ok := innerArray.Elements[j].(*ast.ArrayLiteral)
					if !ok {
						t.Fatalf("innerArray.Elements[%d] is not ast.ArrayLiteral. got=%T",
							j, innerArray.Elements[j])
					}
					if len(deepArray.Elements) != len(expected) {
						t.Fatalf("deepArray.Elements has wrong length. got=%d, want=%d",
							len(deepArray.Elements), len(expected))
					}
				default:
					t.Fatalf("Unsupported expected element type: %T", expected)
				}
			}
		}
	}
}

func TestTypedNestedArrayAssignments(t *testing.T) {
	tests := []struct {
		input            string
		expectedName     string
		expectedType     string
	}{
		{
			`matrix: int[][] = [[1, 2], [3, 4]]`,
			"matrix",
			"int[][]",
		},
		{
			`names: string[][] = [["John", "Doe"], ["Jane", "Smith"]]`,
			"names",
			"string[][]",
		},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		if len(program.Statements) != 1 {
			t.Fatalf("program.Statements does not contain 1 statement. got=%d",
				len(program.Statements))
		}

		stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
		if !ok {
			t.Fatalf("program.Statements[0] is not ast.ExpressionStatement. got=%T",
				program.Statements[0])
		}

		assignment, ok := stmt.Expression.(*ast.AssignmentExpression)
		if !ok {
			t.Fatalf("stmt.Expression is not ast.AssignmentExpression. got=%T",
				stmt.Expression)
		}

		if assignment.Name.Value != tt.expectedName {
			t.Errorf("assignment.Name.Value not '%s'. got=%s", tt.expectedName, assignment.Name.Value)
		}

		if assignment.TypeAnnotation == nil {
			t.Fatalf("assignment.TypeAnnotation is nil. Expected a type annotation.")
		}

		typeAnnotation, ok := assignment.TypeAnnotation.(*ast.TypeAnnotation)
		if !ok {
			t.Fatalf("assignment.TypeAnnotation is not ast.TypeAnnotation. got=%T", assignment.TypeAnnotation)
		}

		if typeAnnotation.Name != tt.expectedType {
			t.Errorf("typeAnnotation.Name not '%s'. got=%s", tt.expectedType, typeAnnotation.Name)
		}

		_, ok = assignment.Value.(*ast.ArrayLiteral)
		if !ok {
			t.Fatalf("assignment.Value is not ast.ArrayLiteral. got=%T", assignment.Value)
		}
	}
}

func TestDebugTypedNestedArrayAssignment(t *testing.T) {
	input := `matrix: int[][] = [[1, 2], [3, 4]]`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d",
			len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("program.Statements[0] is not ast.ExpressionStatement. got=%T",
			program.Statements[0])
	}

	assignment, ok := stmt.Expression.(*ast.AssignmentExpression)
	if !ok {
		t.Fatalf("stmt.Expression is not ast.AssignmentExpression. got=%T",
			stmt.Expression)
	}

	if assignment.Name.Value != "matrix" {
		t.Errorf("assignment.Name.Value not %s. got=%s", "matrix", assignment.Name.Value)
	}

	typeAnnotation, ok := assignment.TypeAnnotation.(*ast.TypeAnnotation)
	if !ok {
		t.Fatalf("assignment.TypeAnnotation is not *ast.TypeAnnotation. got=%T",
			assignment.TypeAnnotation)
	}

	if typeAnnotation.Name != "int[][]" {
		t.Errorf("typeAnnotation.Name not %s. got=%s", "int[][]", typeAnnotation.Name)
	}
}

func TestRangeExpressions(t *testing.T) {
	input := `
x = Range(1, 5)
y = Range(10, 20, true)
`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 2 {
		t.Fatalf("program.Statements does not contain 2 statements. got=%d",
			len(program.Statements))
	}

	stmt1, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("program.Statements[0] is not ast.ExpressionStatement. got=%T",
			program.Statements[0])
	}

	assignment1, ok := stmt1.Expression.(*ast.AssignmentExpression)
	if !ok {
		t.Fatalf("stmt1.Expression is not ast.AssignmentExpression. got=%T",
			stmt1.Expression)
	}

	rangeCall1, ok := assignment1.Value.(*ast.RangeCallExpression)
	if !ok {
		t.Fatalf("assignment1.Value is not ast.RangeCallExpression. got=%T",
			assignment1.Value)
	}

	testIntegerLiteral(t, rangeCall1.Start, 1)
	testIntegerLiteral(t, rangeCall1.End, 5)

	if rangeCall1.Exclusive {
		t.Errorf("rangeCall1.Exclusive should be false")
	}

	stmt2, ok := program.Statements[1].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("program.Statements[1] is not ast.ExpressionStatement. got=%T",
			program.Statements[1])
	}

	assignment2, ok := stmt2.Expression.(*ast.AssignmentExpression)
	if !ok {
		t.Fatalf("stmt2.Expression is not ast.AssignmentExpression. got=%T",
			stmt2.Expression)
	}

	rangeCall2, ok := assignment2.Value.(*ast.RangeCallExpression)
	if !ok {
		t.Fatalf("assignment2.Value is not ast.RangeCallExpression. got=%T",
			assignment2.Value)
	}

	testIntegerLiteral(t, rangeCall2.Start, 10)
	testIntegerLiteral(t, rangeCall2.End, 20)

	if !rangeCall2.Exclusive {
		t.Errorf("rangeCall2.Exclusive should be true")
	}
}

func TestRangeConstructor(t *testing.T) {
	input := `
Range(1, 10)
Range(5, 15, true)
`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 2 {
		t.Fatalf("program.Statements does not contain 2 statements. got=%d",
			len(program.Statements))
	}

	stmt1, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("program.Statements[0] is not ast.ExpressionStatement. got=%T",
			program.Statements[0])
	}

	rangeCall1, ok := stmt1.Expression.(*ast.RangeCallExpression)
	if !ok {
		t.Fatalf("stmt1.Expression is not ast.RangeCallExpression. got=%T",
			stmt1.Expression)
	}

	testIntegerLiteral(t, rangeCall1.Start, 1)
	testIntegerLiteral(t, rangeCall1.End, 10)

	if rangeCall1.Exclusive {
		t.Errorf("rangeCall1.Exclusive should be false")
	}

	stmt2, ok := program.Statements[1].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("program.Statements[1] is not ast.ExpressionStatement. got=%T",
			program.Statements[1])
	}

	rangeCall2, ok := stmt2.Expression.(*ast.RangeCallExpression)
	if !ok {
		t.Fatalf("stmt2.Expression is not ast.RangeCallExpression. got=%T",
			stmt2.Expression)
	}

	testIntegerLiteral(t, rangeCall2.Start, 5)
	testIntegerLiteral(t, rangeCall2.End, 15)

	if !rangeCall2.Exclusive {
		t.Errorf("rangeCall2.Exclusive should be true")
	}
}

func TestFunctionDefinition(t *testing.T) {
	input := `
def greet(name: string): string
  "Hello, ${name}!"
end

def add_numbers(x: int, y: int): int
  x + y
end

def process(x, y: string, z: int): boolean
  true
end

def one_line(): int 42 end
`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 4 {
		t.Fatalf("program.Statements does not contain 4 statements. got=%d",
			len(program.Statements))
	}

	tests := []struct {
		expectedName   string
		expectedParams int
		hasReturnType  bool
	}{
		{"greet", 1, true},
		{"add_numbers", 2, true},
		{"process", 3, true},
		{"one_line", 0, true},
	}

	for i, tt := range tests {
		stmt := program.Statements[i]
		if !testFunctionStatement(t, stmt, tt.expectedName, tt.expectedParams, tt.hasReturnType) {
			return
		}
	}
}

func testFunctionStatement(t *testing.T, stmt ast.Statement, name string, paramCount int, hasReturnType bool) bool {
	exprStmt, ok := stmt.(*ast.ExpressionStatement)
	if !ok {
		t.Errorf("stmt not *ast.ExpressionStatement. got=%T", stmt)
		return false
	}

	function, ok := exprStmt.Expression.(*ast.FunctionLiteral)
	if !ok {
		t.Errorf("stmt.Expression not *ast.FunctionLiteral. got=%T", exprStmt.Expression)
		return false
	}

	if function.Name.Value != name {
		t.Errorf("function name not '%s'. got=%s", name, function.Name.Value)
		return false
	}

	if len(function.Parameters) != paramCount {
		t.Errorf("function %s has wrong number of parameters. want %d, got=%d",
			name, paramCount, len(function.Parameters))
		return false
	}

	if hasReturnType && function.ReturnType == nil {
		t.Errorf("function %s should have return type but got nil", name)
		return false
	}

	return true
}

func checkParserErrors(t *testing.T, p *Parser) {
	errors := p.Errors()
	if len(errors) == 0 {
		return
	}

	t.Errorf("parser has %d errors", len(errors))
	for _, msg := range errors {
		t.Errorf("parser error: %q", msg)
	}
	t.FailNow()
}

func testIntegerLiteral(t *testing.T, il ast.Expression, value int64) bool {
	integ, ok := il.(*ast.IntegerLiteral)
	if !ok {
		t.Errorf("il not *ast.IntegerLiteral. got=%T", il)
		return false
	}

	if integ.Value != value {
		t.Errorf("integ.Value not %d. got=%d", value, integ.Value)
		return false
	}

	if integ.TokenLiteral() != fmt.Sprintf("%d", value) {
		t.Errorf("integ.TokenLiteral not %d. got=%s", value, integ.TokenLiteral())
		return false
	}

	return true
}

func testFloatLiteral(t *testing.T, fl ast.Expression, value float64) bool {
	float, ok := fl.(*ast.FloatLiteral)
	if !ok {
		t.Errorf("fl not *ast.FloatLiteral. got=%T", fl)
		return false
	}

	if float.Value != value {
		t.Errorf("float.Value not %f. got=%f", value, float.Value)
		return false
	}

	return true
}

func testStringLiteral(t *testing.T, sl ast.Expression, value string) bool {
	str, ok := sl.(*ast.StringLiteral)
	if !ok {
		t.Errorf("sl not *ast.StringLiteral. got=%T", sl)
		return false
	}

	if str.Value != value {
		t.Errorf("str.Value not %s. got=%s", value, str.Value)
		return false
	}

	return true
}

func TestStringInterpolation(t *testing.T) {
	input := `
name = "Alice"
age = 30
message = "Hello, my name is ${name} and I am ${age} years old"
greeting = "Welcome, ${name}!"
complex = "The result is ${age * 2 + 5}"
`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 5 {
		t.Fatalf("program.Statements does not contain 5 statements. got=%d",
			len(program.Statements))
	}

	tests := []struct {
		expectedIdentifier string
	}{
		{"name"},
		{"age"},
		{"message"},
		{"greeting"},
		{"complex"},
	}

	for i, tt := range tests {
		stmt := program.Statements[i]
		if !testAssignmentStatement(t, stmt, tt.expectedIdentifier) {
			return
		}
	}
}

func testAssignmentStatement(t *testing.T, s ast.Statement, name string) bool {
	expStmt, ok := s.(*ast.ExpressionStatement)
	if !ok {
		t.Errorf("s not *ast.ExpressionStatement. got=%T", s)
		return false
	}

	assignment, ok := expStmt.Expression.(*ast.AssignmentExpression)
	if !ok {
		t.Errorf("expStmt.Expression not *ast.AssignmentExpression. got=%T", expStmt.Expression)
		return false
	}

	if assignment.Name.Value != name {
		t.Errorf("assignment.Name.Value not '%s'. got=%s", name, assignment.Name.Value)
		return false
	}

	if assignment.Name.TokenLiteral() != name {
		t.Errorf("assignment.Name.TokenLiteral() not '%s'. got=%s", name, assignment.Name.TokenLiteral())
		return false
	}

	return true
}

func TestStringInterpolationParsing(t *testing.T) {
	tests := []struct {
		input           string
		expectedParts   []string
		expectedExprNum int
	}{
		{`"Hello, ${name}!"`, []string{"Hello, ", "!"}, 1},
		{`"The sum is ${a + b}"`, []string{"The sum is ", ""}, 1},
		{`"${greeting}, ${name}!"`, []string{"", ", ", "!"}, 2},
		{`"Result: ${x * (y + z)}"`, []string{"Result: ", ""}, 1},
		{`"Nested ${outer}"`, []string{"Nested ", ""}, 1},
	}

	for i, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		if len(program.Statements) != 1 {
			t.Fatalf("test[%d] - program.Statements does not contain 1 statement. got=%d",
				i, len(program.Statements))
		}

		stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
		if !ok {
			t.Fatalf("test[%d] - program.Statements[0] is not *ast.ExpressionStatement. got=%T",
				i, program.Statements[0])
		}

		interpolation, ok := stmt.Expression.(*ast.StringInterpolationLiteral)
		if !ok {
			t.Fatalf("test[%d] - exp not *ast.StringInterpolationLiteral. got=%T",
				i, stmt.Expression)
		}

		if len(interpolation.Parts) != len(tt.expectedParts) {
			t.Fatalf("test[%d] - interpolation has wrong number of parts. expected=%d, got=%d",
				i, len(tt.expectedParts), len(interpolation.Parts))
		}

		for j, expectedPart := range tt.expectedParts {
			if interpolation.Parts[j] != expectedPart {
				t.Errorf("test[%d] - interpolation.Parts[%d] wrong. expected=%q, got=%q",
					i, j, expectedPart, interpolation.Parts[j])
			}
		}

		if len(interpolation.Expressions) != tt.expectedExprNum {
			t.Fatalf("test[%d] - interpolation has wrong number of expressions. expected=%d, got=%d",
				i, tt.expectedExprNum, len(interpolation.Expressions))
		}
	}
}

func TestBasicStringInterpolation(t *testing.T) {
	input := `"Hello, ${name}!"`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d",
			len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("program.Statements[0] is not *ast.ExpressionStatement. got=%T",
			program.Statements[0])
	}

	interpolation, ok := stmt.Expression.(*ast.StringInterpolationLiteral)
	if !ok {
		t.Fatalf("exp not *ast.StringInterpolationLiteral. got=%T", stmt.Expression)
	}

	if len(interpolation.Parts) != 2 {
		t.Fatalf("interpolation has wrong number of parts. expected=2, got=%d",
			len(interpolation.Parts))
	}

	expectedParts := []string{"Hello, ", "!"}
	for i, expectedPart := range expectedParts {
		if interpolation.Parts[i] != expectedPart {
			t.Errorf("interpolation.Parts[%d] wrong. expected=%q, got=%q",
				i, expectedPart, interpolation.Parts[i])
		}
	}

	if len(interpolation.Expressions) != 1 {
		t.Fatalf("interpolation has wrong number of expressions. expected=1, got=%d",
			len(interpolation.Expressions))
	}

	ident, ok := interpolation.Expressions[0].(*ast.Identifier)
	if !ok {
		t.Fatalf("expression is not *ast.Identifier. got=%T",
			interpolation.Expressions[0])
	}

	if ident.Value != "name" {
		t.Errorf("ident.Value not %s. got=%s", "name", ident.Value)
	}
}

func TestMultiDimensionalTypedArrays(t *testing.T) {
	input := `
matrix: int[][] = [[1, 2], [3, 4]]
cube: int[][][] = [[[1, 2], [3, 4]], [[5, 6], [7, 8]]]
jagged: int[][] = [[1, 2, 3], [4, 5]]
empty_matrix: float[][] = []
`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 4 {
		t.Fatalf("program.Statements does not contain 4 statements. got=%d",
			len(program.Statements))
	}

	tests := []struct {
		expectedIdentifier string
		expectedDimensions int
		expectedBaseType   string
	}{
		{"matrix", 2, "int"},
		{"cube", 3, "int"},
		{"jagged", 2, "int"},
		{"empty_matrix", 2, "float"},
	}

	for i, tt := range tests {
		stmt := program.Statements[i]
		if !testMultiDimensionalArrayAssignment(t, stmt, tt.expectedIdentifier, tt.expectedDimensions, tt.expectedBaseType) {
			return
		}
	}
}

func testMultiDimensionalArrayAssignment(t *testing.T, s ast.Statement, name string, dimensions int, baseType string) bool {
	expStmt, ok := s.(*ast.ExpressionStatement)
	if !ok {
		t.Errorf("s not *ast.ExpressionStatement. got=%T", s)
		return false
	}

	assignment, ok := expStmt.Expression.(*ast.AssignmentExpression)
	if !ok {
		t.Errorf("expStmt.Expression not *ast.AssignmentExpression. got=%T", expStmt.Expression)
		return false
	}

	if assignment.Name.Value != name {
		t.Errorf("assignment.Name.Value not '%s'. got=%s", name, assignment.Name.Value)
		return false
	}

	var typeAnnotation *ast.TypeAnnotation

	switch annotation := assignment.TypeAnnotation.(type) {
	case *ast.TypedIdentifier:
		typeAnnotation = annotation.Type
	case *ast.TypeAnnotation:
		typeAnnotation = annotation
	default:
		t.Errorf("assignment.TypeAnnotation not a type annotation. got=%T", assignment.TypeAnnotation)
		return false
	}

	if !strings.HasPrefix(typeAnnotation.Name, baseType) {
		t.Errorf("Type annotation base type wrong. expected=%s, got=%s",
			baseType, typeAnnotation.Name)
		return false
	}

	dimCount := strings.Count(typeAnnotation.Name, "[]")
	if dimCount != dimensions {
		t.Errorf("Type annotation has wrong number of dimensions. expected=%d, got=%d",
			dimensions, dimCount)
		return false
	}

	arrayLit, ok := assignment.Value.(*ast.ArrayLiteral)
	if !ok && name != "empty_matrix" {
		t.Errorf("assignment.Value not *ast.ArrayLiteral. got=%T", assignment.Value)
		return false
	}

	if name == "empty_matrix" {
		if arrayLit != nil && len(arrayLit.Elements) > 0 {
			t.Errorf("Expected empty array, got array with %d elements", len(arrayLit.Elements))
			return false
		}
	}

	return true
}

func TestStructTypedArrays(t *testing.T) {
	input := `
struct Person
  name: string
  age: int
end

struct Team
  name: string
  members: Person[]
end

students: Person[] = [
  Person(name: "Alice", age: 20),
  Person(name: "Bob", age: 22)
]

empty_staff: Person[] = []

teams: Team[] = [
  Team(name: "Red", members: [Person(name: "Charlie", age: 25), Person(name: "Diana", age: 23)]),
  Team(name: "Blue", members: [Person(name: "Eve", age: 24)])
]

nested: Person[][] = [
  [Person(name: "Frank", age: 30), Person(name: "Grace", age: 28)],
  [Person(name: "Heidi", age: 32)]
]
`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 6 {
		t.Fatalf("program.Statements does not contain 6 statements. got=%d",
			len(program.Statements))
	}

	personStruct, ok := program.Statements[0].(*ast.StructStatement)
	if !ok {
		t.Fatalf("program.Statements[0] is not *ast.StructStatement. got=%T",
			program.Statements[0])
	}
	if personStruct.Name.Value != "Person" {
		t.Fatalf("personStruct.Name.Value not 'Person'. got=%s", personStruct.Name.Value)
	}
	if len(personStruct.Fields) != 2 {
		t.Fatalf("personStruct.Fields does not contain 2 fields. got=%d",
			len(personStruct.Fields))
	}

	teamStruct, ok := program.Statements[1].(*ast.StructStatement)
	if !ok {
		t.Fatalf("program.Statements[1] is not *ast.StructStatement. got=%T",
			program.Statements[1])
	}
	if teamStruct.Name.Value != "Team" {
		t.Fatalf("teamStruct.Name.Value not 'Team'. got=%s", teamStruct.Name.Value)
	}
	if len(teamStruct.Fields) != 2 {
		t.Fatalf("teamStruct.Fields does not contain 2 fields. got=%d",
			len(teamStruct.Fields))
	}

	tests := []struct {
		expectedIdentifier string
		expectedType       string
		expectedArraySize  int
		isMultiDimensional bool
	}{
		{"students", "Person[]", 2, false},
		{"empty_staff", "Person[]", 0, false},
		{"teams", "Team[]", 2, false},
		{"nested", "Person[][]", 2, true},
	}

	for i, tt := range tests {
		stmt := program.Statements[i+2]
		testStructArrayAssignment(t, stmt, tt.expectedIdentifier, tt.expectedType, tt.expectedArraySize, tt.isMultiDimensional)
	}
}

func testStructArrayAssignment(t *testing.T, s ast.Statement, name string, expectedType string, expectedSize int, isMultidimensional bool) bool {
	expStmt, ok := s.(*ast.ExpressionStatement)
	if !ok {
		t.Errorf("s not *ast.ExpressionStatement. got=%T", s)
		return false
	}

	assignment, ok := expStmt.Expression.(*ast.AssignmentExpression)
	if !ok {
		t.Errorf("expStmt.Expression not *ast.AssignmentExpression. got=%T", expStmt.Expression)
		return false
	}

	if assignment.Name.Value != name {
		t.Errorf("assignment.Name.Value not '%s'. got=%s", name, assignment.Name.Value)
		return false
	}

	var typeAnnotation *ast.TypeAnnotation

	switch annotation := assignment.TypeAnnotation.(type) {
	case *ast.TypedIdentifier:
		typeAnnotation = annotation.Type
	case *ast.TypeAnnotation:
		typeAnnotation = annotation
	default:
		t.Errorf("assignment.TypeAnnotation not a type annotation. got=%T", assignment.TypeAnnotation)
		return false
	}

	if typeAnnotation.Name != expectedType {
		t.Errorf("Type annotation wrong. expected=%s, got=%s",
			expectedType, typeAnnotation.Name)
		return false
	}

	arrayLit, ok := assignment.Value.(*ast.ArrayLiteral)
	if !ok {
		t.Errorf("assignment.Value not *ast.ArrayLiteral. got=%T", assignment.Value)
		return false
	}

	if len(arrayLit.Elements) != expectedSize {
		t.Errorf("array size wrong. expected=%d, got=%d",
			expectedSize, len(arrayLit.Elements))
		return false
	}

	if isMultidimensional && expectedSize > 0 {
		_, ok := arrayLit.Elements[0].(*ast.ArrayLiteral)
		if !ok {
			t.Errorf("Expected nested array literal, got=%T", arrayLit.Elements[0])
			return false
		}
	}

	return true
}

func TestCompoundTypedArrays(t *testing.T) {
	input := `
pairs: [string, int][] = [
  ["one", 1],
  ["two", 2],
  ["three", 3]
]

coordinates: [float, float][] = [
  [10.5, 20.3],
  [30.2, 40.1],
  [50.7, 60.9]
]

empty_records: [string, int, boolean][] = []

matrix: [int, int][][] = [
  [[1, 2], [3, 4]],
  [[5, 6], [7, 8]]
]
`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 4 {
		t.Fatalf("program.Statements does not contain 4 statements. got=%d",
			len(program.Statements))
	}

	tests := []struct {
		expectedIdentifier string
		expectedElementCount int
		expectedCompoundTypes int
		isMultiDimensional bool
	}{
		{"pairs", 3, 2, false},
		{"coordinates", 3, 2, false},
		{"empty_records", 0, 3, false},
		{"matrix", 2, 2, true},
	}

	for i, tt := range tests {
		stmt := program.Statements[i]
		testCompoundArrayAssignment(t, stmt, tt.expectedIdentifier, tt.expectedElementCount, tt.expectedCompoundTypes, tt.isMultiDimensional)
	}
}

func testCompoundArrayAssignment(t *testing.T, s ast.Statement, name string, expectedSize int, expectedCompoundTypes int, isMultidimensional bool) bool {
	expStmt, ok := s.(*ast.ExpressionStatement)
	if !ok {
		t.Errorf("s not *ast.ExpressionStatement. got=%T", s)
		return false
	}

	assignment, ok := expStmt.Expression.(*ast.AssignmentExpression)
	if !ok {
		t.Errorf("expStmt.Expression not *ast.AssignmentExpression. got=%T", expStmt.Expression)
		return false
	}

	if assignment.Name.Value != name {
		t.Errorf("assignment.Name.Value not '%s'. got=%s", name, assignment.Name.Value)
		return false
	}

	var typeAnnotation *ast.TypeAnnotation

	switch annotation := assignment.TypeAnnotation.(type) {
	case *ast.TypedIdentifier:
		typeAnnotation = annotation.Type
	case *ast.TypeAnnotation:
		typeAnnotation = annotation
	default:
		t.Errorf("assignment.TypeAnnotation not a type annotation. got=%T", assignment.TypeAnnotation)
		return false
	}

	if !typeAnnotation.IsCompoundType && expectedCompoundTypes > 0 {
		t.Errorf("Expected compound type annotation, got simple type: %s", typeAnnotation.Name)
		return false
	}

	if expectedCompoundTypes > 0 && len(typeAnnotation.CompoundTypes) != expectedCompoundTypes {
		t.Errorf("Wrong number of compound types. expected=%d, got=%d",
			expectedCompoundTypes, len(typeAnnotation.CompoundTypes))
		return false
	}

	arrayLit, ok := assignment.Value.(*ast.ArrayLiteral)
	if !ok {
		t.Errorf("assignment.Value not *ast.ArrayLiteral. got=%T", assignment.Value)
		return false
	}

	if len(arrayLit.Elements) != expectedSize {
		t.Errorf("array size wrong. expected=%d, got=%d",
			expectedSize, len(arrayLit.Elements))
		return false
	}

	if isMultidimensional && expectedSize > 0 {
		_, ok := arrayLit.Elements[0].(*ast.ArrayLiteral)
		if !ok {
			t.Errorf("Expected nested array literal, got=%T", arrayLit.Elements[0])
			return false
		}
	}

	return true
}

func TestLexerTokenization(t *testing.T) {
	input := `h: int[] = [1, 2, 3]`

	l := lexer.New(input)

	tokens := []struct {
		expectedType    lexer.TokenType
		expectedLiteral string
	}{
		{lexer.IDENT, "h"},
		{lexer.COLON, ":"},
		{lexer.IDENT, "int"},
		{lexer.LBRACKET, "["},
		{lexer.RBRACKET, "]"},
		{lexer.ASSIGN, "="},
		{lexer.LBRACKET, "["},
		{lexer.INT, "1"},
		{lexer.COMMA, ","},
		{lexer.INT, "2"},
		{lexer.COMMA, ","},
		{lexer.INT, "3"},
		{lexer.RBRACKET, "]"},
	}

	for i, tt := range tokens {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tokens[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tokens[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestTokenizeMultiDimensionalArrayType(t *testing.T) {
	input := `matrix: int[][] = [[1, 2], [3, 4]]`

	l := lexer.New(input)
	p := New(l)

	// Manually simulate the parsing process
	t.Logf("Initial tokens: curToken=%s(%s), peekToken=%s(%s)",
		p.curToken.Type, p.curToken.Literal, p.peekToken.Type, p.peekToken.Literal)

	// Parse the variable name 'matrix'
	if !p.curTokenIs(lexer.IDENT) || p.curToken.Literal != "matrix" {
		t.Fatalf("Expected 'matrix' identifier, got %s(%s)", p.curToken.Type, p.curToken.Literal)
	}

	// Next should be colon
	p.nextToken()
	t.Logf("After identifier: curToken=%s(%s), peekToken=%s(%s)",
		p.curToken.Type, p.curToken.Literal, p.peekToken.Type, p.peekToken.Literal)

	if !p.curTokenIs(lexer.COLON) {
		t.Fatalf("Expected ':', got %s(%s)", p.curToken.Type, p.curToken.Literal)
	}

	// Next should be the type 'int'
	p.nextToken()
	t.Logf("After colon: curToken=%s(%s), peekToken=%s(%s)",
		p.curToken.Type, p.curToken.Literal, p.peekToken.Type, p.peekToken.Literal)

	if !p.curTokenIs(lexer.IDENT) || p.curToken.Literal != "int" {
		t.Fatalf("Expected 'int' identifier, got %s(%s)", p.curToken.Type, p.curToken.Literal)
	}

	// Next should be first '['
	p.nextToken()
	t.Logf("After type: curToken=%s(%s), peekToken=%s(%s)",
		p.curToken.Type, p.curToken.Literal, p.peekToken.Type, p.peekToken.Literal)

	if !p.curTokenIs(lexer.LBRACKET) {
		t.Fatalf("Expected '[', got %s(%s)", p.curToken.Type, p.curToken.Literal)
	}

	// Next should be first ']'
	p.nextToken()
	t.Logf("After first '[': curToken=%s(%s), peekToken=%s(%s)",
		p.curToken.Type, p.curToken.Literal, p.peekToken.Type, p.peekToken.Literal)

	if !p.curTokenIs(lexer.RBRACKET) {
		t.Fatalf("Expected ']', got %s(%s)", p.curToken.Type, p.curToken.Literal)
	}

	// Next should be second '['
	p.nextToken()
	t.Logf("After first ']': curToken=%s(%s), peekToken=%s(%s)",
		p.curToken.Type, p.curToken.Literal, p.peekToken.Type, p.peekToken.Literal)

	if !p.curTokenIs(lexer.LBRACKET) {
		t.Fatalf("Expected '[', got %s(%s)", p.curToken.Type, p.curToken.Literal)
	}

	// Next should be second ']'
	p.nextToken()
	t.Logf("After second '[': curToken=%s(%s), peekToken=%s(%s)",
		p.curToken.Type, p.curToken.Literal, p.peekToken.Type, p.peekToken.Literal)

	if !p.curTokenIs(lexer.RBRACKET) {
		t.Fatalf("Expected ']', got %s(%s)", p.curToken.Type, p.curToken.Literal)
	}

	// Next should be '='
	p.nextToken()
	t.Logf("After second ']': curToken=%s(%s), peekToken=%s(%s)",
		p.curToken.Type, p.curToken.Literal, p.peekToken.Type, p.peekToken.Literal)

	if !p.curTokenIs(lexer.ASSIGN) {
		t.Fatalf("Expected '=', got %s(%s)", p.curToken.Type, p.curToken.Literal)
	}

	// This is where we're having issues - after the equals sign
	p.nextToken()
	t.Logf("After '=': curToken=%s(%s), peekToken=%s(%s)",
		p.curToken.Type, p.curToken.Literal, p.peekToken.Type, p.peekToken.Literal)
}