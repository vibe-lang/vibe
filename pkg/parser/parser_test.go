package parser

import (
	"fmt"
	"testing"

	"github.com/vibe-lang/vibe/pkg/ast"
	"github.com/vibe-lang/vibe/pkg/lexer"
)

func TestVariableStatements(t *testing.T) {
	input := `
x = 5;
y = 10;
foobar = 838383;
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
			`g = [];`,
			[]interface{}{},
		},
		{
			`h = [1, 2, 3];`,
			[]interface{}{1, 2, 3},
		},
		{
			`i = ["hello", "world"];`,
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
			`h: int[] = [1, 2, 3];`,
			"h",
			"int[]",
			[]interface{}{1, 2, 3},
		},
		{
			`i: string[] = ["hello", "world"];`,
			"i",
			"string[]",
			[]interface{}{"hello", "world"},
		},
		{
			`j: float[] = [1.5, 3.8, 1.0];`,
			"j",
			"float[]",
			[]interface{}{1.5, 3.8, 1.0},
		},
		{
			`k: int[] = [];`,
			"k",
			"int[]",
			[]interface{}{},
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
				// Handle the case where an integer literal might be used in a float context
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
			`nested = [[1, 2], [3, 4]];`,
			[][]interface{}{
				{1, 2},
				{3, 4},
			},
		},
		{
			`strings = [["hello", "world"], ["foo", "bar"]];`,
			[][]interface{}{
				{"hello", "world"},
				{"foo", "bar"},
			},
		},
		{
			`mixed = [[], [1, 2]];`,
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

		// Check each nested array
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

			// Check each element in the nested array
			for j, expectedElem := range expectedNestedArr {
				switch expected := expectedElem.(type) {
				case int:
					testIntegerLiteral(t, innerArray.Elements[j], int64(expected))
				case string:
					testStringLiteral(t, innerArray.Elements[j], expected)
				case []interface{}:
					// Handle deeply nested arrays (3+ levels)
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
			`matrix: int[][] = [[1, 2], [3, 4]];`,
			"matrix",
			"int[][]",
		},
		{
			`names: string[][] = [["John", "Doe"], ["Jane", "Smith"]];`,
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

		// Check that the value is an array literal
		_, ok = assignment.Value.(*ast.ArrayLiteral)
		if !ok {
			t.Fatalf("assignment.Value is not ast.ArrayLiteral. got=%T", assignment.Value)
		}
	}
}

// TestDebugTypedNestedArrayAssignment is a debug test for a specific case
func TestDebugTypedNestedArrayAssignment(t *testing.T) {
	input := `matrix: int[][] = [[1, 2], [3, 4]];`
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

	// Verify the assignment has the correct structure
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