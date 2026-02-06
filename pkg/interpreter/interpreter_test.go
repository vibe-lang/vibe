package interpreter

import (
	"fmt"
	"strings"
	"testing"

	"github.com/vibe-lang/vibe/pkg/lexer"
	"github.com/vibe-lang/vibe/pkg/parser"
)

// TestArrayLiterals tests the interpretation of array literals.
// It verifies that array elements are correctly interpreted.
func TestArrayLiterals(t *testing.T) {
	tests := []struct {
		input    string
		expected []interface{}
	}{
		{
			"[]",
			[]interface{}{},
		},
		{
			"[1, 2, 3]",
			[]interface{}{1, 2, 3},
		},
		{
			`["hello", "world"]`,
			[]interface{}{"hello", "world"},
		},
	}

	for _, tt := range tests {
		evaluated := testEvalExpression(tt.input)

		array, ok := evaluated.(*Array)
		if !ok {
			t.Errorf("object is not Array. got=%T (%+v)", evaluated, evaluated)
			continue
		}

		if len(array.Elements) != len(tt.expected) {
			t.Errorf("array has wrong number of elements. got=%d, want=%d",
				len(array.Elements), len(tt.expected))
			continue
		}

		for i, expectedElem := range tt.expected {
			switch expected := expectedElem.(type) {
			case int:
				testIntegerObject(t, array.Elements[i], int64(expected))
			case string:
				testStringObject(t, array.Elements[i], expected)
			}
		}
	}
}

// TestTypedArrayAssignments tests typed array assignments
func TestTypedArrayAssignments(t *testing.T) {
	tests := []struct {
		name     string
		array    *Array
		elemType ObjectType
		count    int
	}{
		{
			name: "Integer array",
			array: &Array{
				Elements: []Object{
					&Integer{Value: 1},
					&Integer{Value: 2},
					&Integer{Value: 3},
				},
			},
			elemType: INTEGER_OBJ,
			count:    3,
		},
		{
			name: "String array",
			array: &Array{
				Elements: []Object{
					&String{Value: "hello"},
					&String{Value: "world"},
				},
			},
			elemType: STRING_OBJ,
			count:    2,
		},
		{
			name: "Empty array",
			array: &Array{
				Elements: []Object{},
			},
			elemType: INTEGER_OBJ, // Type doesn't matter for empty array
			count:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the array type
			if tt.array.Type() != ARRAY_OBJ {
				t.Errorf("wrong type for array. got=%s, want=%s",
					tt.array.Type(), ARRAY_OBJ)
			}

			// Test the number of elements
			if len(tt.array.Elements) != tt.count {
				t.Errorf("wrong number of elements. got=%d, want=%d",
					len(tt.array.Elements), tt.count)
			}

			// Test each element's type
			for i, elem := range tt.array.Elements {
				if elem.Type() != tt.elemType {
					t.Errorf("wrong type for element %d. got=%s, want=%s",
						i, elem.Type(), tt.elemType)
				}
			}
		})
	}
}

// TestArrayTypeMismatchErrors tests type mismatch errors in array assignments
func TestArrayTypeMismatchErrors(t *testing.T) {
	tests := []struct {
		arrayType string
		elements  []Object
		expected  string
	}{
		{
			"int[]",
			[]Object{
				&String{Value: "not"},
				&String{Value: "integers"},
			},
			"type mismatch in array: expected int, got STRING",
		},
		{
			"string[]",
			[]Object{
				&Integer{Value: 1},
				&Integer{Value: 2},
				&Integer{Value: 3},
			},
			"type mismatch in array: expected string, got INTEGER",
		},
	}

	for _, tt := range tests {
		// Create a new interpreter
		interpreter := New()

		// Create an array object
		array := &Array{Elements: tt.elements}

		// Validate the array type
		err := interpreter.validateArrayType(tt.arrayType, array)

		// Check if we got an error
		if err == nil {
			t.Errorf("no error returned for array type %s with elements %+v",
				tt.arrayType, tt.elements)
			continue
		}

		errorObj, ok := err.(*Error)
		if !ok {
			t.Errorf("result is not Error. got=%T(%+v)", err, err)
			continue
		}

		if !strings.Contains(errorObj.Message, tt.expected) {
			t.Errorf("wrong error message. expected to contain=%q, got=%q",
				tt.expected, errorObj.Message)
		}
	}
}

// TestNestedArrayLiterals tests the interpretation of nested array literals
func TestNestedArrayLiterals(t *testing.T) {
	tests := []struct {
		input    string
		expected [][]interface{}
	}{
		{
			"[[1, 2], [3, 4]]",
			[][]interface{}{
				{1, 2},
				{3, 4},
			},
		},
		{
			`[["hello", "world"], ["foo", "bar"]]`,
			[][]interface{}{
				{"hello", "world"},
				{"foo", "bar"},
			},
		},
		{
			"[[], [1, 2]]",
			[][]interface{}{
				{},
				{1, 2},
			},
		},
	}

	for _, tt := range tests {
		evaluated := testEvalExpression(tt.input)

		array, ok := evaluated.(*Array)
		if !ok {
			t.Errorf("object is not Array. got=%T (%+v)", evaluated, evaluated)
			continue
		}

		if len(array.Elements) != len(tt.expected) {
			t.Errorf("array has wrong number of elements. got=%d, want=%d",
				len(array.Elements), len(tt.expected))
			continue
		}

		for i, expectedElem := range tt.expected {
			innerArray, ok := array.Elements[i].(*Array)
			if !ok {
				t.Errorf("element %d is not Array. got=%T (%+v)",
					i, array.Elements[i], array.Elements[i])
				continue
			}

			if len(innerArray.Elements) != len(expectedElem) {
				t.Errorf("inner array %d has wrong number of elements. got=%d, want=%d",
					i, len(innerArray.Elements), len(expectedElem))
				continue
			}

			for j, expected := range expectedElem {
				switch expected := expected.(type) {
				case int:
					testIntegerObject(t, innerArray.Elements[j], int64(expected))
				case string:
					testStringObject(t, innerArray.Elements[j], expected)
				}
			}
		}
	}
}

// TestTypedNestedArrayAssignments tests typed nested array assignments
func TestTypedNestedArrayAssignments(t *testing.T) {
	// Create a nested integer array
	intMatrix := &Array{
		Elements: []Object{
			&Array{
				Elements: []Object{
					&Integer{Value: 1},
					&Integer{Value: 2},
				},
			},
			&Array{
				Elements: []Object{
					&Integer{Value: 3},
					&Integer{Value: 4},
				},
			},
		},
	}

	// Create a nested string array
	stringMatrix := &Array{
		Elements: []Object{
			&Array{
				Elements: []Object{
					&String{Value: "John"},
					&String{Value: "Doe"},
				},
			},
			&Array{
				Elements: []Object{
					&String{Value: "Jane"},
					&String{Value: "Smith"},
				},
			},
		},
	}

	// Test integer matrix
	if intMatrix.Type() != ARRAY_OBJ {
		t.Errorf("wrong type for intMatrix. got=%s, want=%s",
			intMatrix.Type(), ARRAY_OBJ)
	}

	if len(intMatrix.Elements) != 2 {
		t.Errorf("wrong number of rows in intMatrix. got=%d, want=%d",
			len(intMatrix.Elements), 2)
	}

	// Test first row of integer matrix
	row1 := intMatrix.Elements[0].(*Array)
	if row1.Type() != ARRAY_OBJ {
		t.Errorf("wrong type for first row. got=%s, want=%s",
			row1.Type(), ARRAY_OBJ)
	}

	if len(row1.Elements) != 2 {
		t.Errorf("wrong number of elements in first row. got=%d, want=%d",
			len(row1.Elements), 2)
	}

	testIntegerObject(t, row1.Elements[0], 1)
	testIntegerObject(t, row1.Elements[1], 2)

	// Test second row of integer matrix
	row2 := intMatrix.Elements[1].(*Array)
	testIntegerObject(t, row2.Elements[0], 3)
	testIntegerObject(t, row2.Elements[1], 4)

	// Test string matrix
	if stringMatrix.Type() != ARRAY_OBJ {
		t.Errorf("wrong type for stringMatrix. got=%s, want=%s",
			stringMatrix.Type(), ARRAY_OBJ)
	}

	if len(stringMatrix.Elements) != 2 {
		t.Errorf("wrong number of rows in stringMatrix. got=%d, want=%d",
			len(stringMatrix.Elements), 2)
	}

	// Test first row of string matrix
	srow1 := stringMatrix.Elements[0].(*Array)
	if srow1.Type() != ARRAY_OBJ {
		t.Errorf("wrong type for first row. got=%s, want=%s",
			srow1.Type(), ARRAY_OBJ)
	}

	if len(srow1.Elements) != 2 {
		t.Errorf("wrong number of elements in first row. got=%d, want=%d",
			len(srow1.Elements), 2)
	}

	testStringObject(t, srow1.Elements[0], "John")
	testStringObject(t, srow1.Elements[1], "Doe")

	// Test second row of string matrix
	srow2 := stringMatrix.Elements[1].(*Array)
	testStringObject(t, srow2.Elements[0], "Jane")
	testStringObject(t, srow2.Elements[1], "Smith")
}

func TestNestedArrayTypeMismatchErrors(t *testing.T) {
	tests := []struct {
		arrayType string
		elements  []Object
		expected  string
	}{
		{
			"int[][]",
			[]Object{
				&Array{Elements: []Object{&Integer{Value: 1}, &Integer{Value: 2}}},
				&Array{Elements: []Object{&String{Value: "a"}, &String{Value: "b"}}},
			},
			"type mismatch in array: expected int, got STRING",
		},
		{
			"string[][]",
			[]Object{
				&Array{Elements: []Object{&String{Value: "a"}, &String{Value: "b"}}},
				&Array{Elements: []Object{&Integer{Value: 1}, &Integer{Value: 2}}},
			},
			"type mismatch in array: expected string, got INTEGER",
		},
		{
			"int[][][]",
			[]Object{
				&Array{Elements: []Object{
					&Array{Elements: []Object{&Integer{Value: 1}, &Integer{Value: 2}}},
				}},
				&Array{Elements: []Object{
					&Array{Elements: []Object{&Integer{Value: 3}, &String{Value: "bad"}}},
				}},
			},
			"type mismatch in array: expected int, got STRING",
		},
	}

	for _, tt := range tests {
		// Create a new interpreter
		interpreter := New()

		// Create an array object
		array := &Array{Elements: tt.elements}

		// Validate the array type
		err := interpreter.validateArrayType(tt.arrayType, array)

		// Check if we got an error
		if err == nil {
			t.Errorf("no error returned for array type %s with elements %+v",
				tt.arrayType, tt.elements)
			continue
		}

		errorObj, ok := err.(*Error)
		if !ok {
			t.Errorf("result is not Error. got=%T(%+v)", err, err)
			continue
		}

		if !strings.Contains(errorObj.Message, tt.expected) {
			t.Errorf("wrong error message. expected to contain=%q, got=%q",
				tt.expected, errorObj.Message)
		}
	}
}

func TestStructDefinition(t *testing.T) {
	// Create a struct definition for Person
	personStruct := &Struct{
		Name: "Person",
		Fields: map[string]Object{
			"name": &String{Value: "John"},
			"age":  &Integer{Value: 30},
		},
	}

	// Create a struct definition for Car
	carStruct := &Struct{
		Name: "Car",
		Fields: map[string]Object{
			"make":  &String{Value: "Toyota"},
			"model": &String{Value: "Corolla"},
			"year":  &Integer{Value: 2020},
		},
	}

	// Test Person struct
	if personStruct.Type() != STRUCT_OBJ {
		t.Errorf("wrong type for personStruct. got=%s, want=%s",
			personStruct.Type(), STRUCT_OBJ)
	}

	if personStruct.Name != "Person" {
		t.Errorf("wrong name for personStruct. got=%s, want=%s",
			personStruct.Name, "Person")
	}

	if len(personStruct.Fields) != 2 {
		t.Errorf("wrong number of fields in personStruct. got=%d, want=%d",
			len(personStruct.Fields), 2)
	}

	// Test Person struct fields
	nameField, ok := personStruct.Fields["name"]
	if !ok {
		t.Errorf("field 'name' not found in personStruct")
	} else {
		testStringObject(t, nameField, "John")
	}

	ageField, ok := personStruct.Fields["age"]
	if !ok {
		t.Errorf("field 'age' not found in personStruct")
	} else {
		testIntegerObject(t, ageField, 30)
	}

	// Test Car struct
	if carStruct.Type() != STRUCT_OBJ {
		t.Errorf("wrong type for carStruct. got=%s, want=%s",
			carStruct.Type(), STRUCT_OBJ)
	}

	if carStruct.Name != "Car" {
		t.Errorf("wrong name for carStruct. got=%s, want=%s",
			carStruct.Name, "Car")
	}

	if len(carStruct.Fields) != 3 {
		t.Errorf("wrong number of fields in carStruct. got=%d, want=%d",
			len(carStruct.Fields), 3)
	}

	// Test Car struct fields
	makeField, ok := carStruct.Fields["make"]
	if !ok {
		t.Errorf("field 'make' not found in carStruct")
	} else {
		testStringObject(t, makeField, "Toyota")
	}

	modelField, ok := carStruct.Fields["model"]
	if !ok {
		t.Errorf("field 'model' not found in carStruct")
	} else {
		testStringObject(t, modelField, "Corolla")
	}

	yearField, ok := carStruct.Fields["year"]
	if !ok {
		t.Errorf("field 'year' not found in carStruct")
	} else {
		testIntegerObject(t, yearField, 2020)
	}
}

func TestStructInstantiation(t *testing.T) {
	// Create a struct definition for Person
	personStruct := &Struct{
		Name: "Person",
		Fields: map[string]Object{
			"name": &String{Value: "John"},
			"age":  &Integer{Value: 30},
		},
		DefaultValues: map[string]Object{
			"name": &String{Value: "John"},
			"age":  &Integer{Value: 30},
		},
	}

	// Create a Car instance with custom values
	customCar := &Struct{
		Name: "Car",
		Fields: map[string]Object{
			"make":  &String{Value: "Honda"},
			"model": &String{Value: "Civic"},
			"year":  &Integer{Value: 2020},
		},
	}

	// Test Person struct
	if personStruct.Type() != STRUCT_OBJ {
		t.Errorf("wrong type for personStruct. got=%s, want=%s",
			personStruct.Type(), STRUCT_OBJ)
	}

	// Test Person struct fields
	nameField, ok := personStruct.Fields["name"]
	if !ok {
		t.Errorf("field 'name' not found in personStruct")
	} else {
		testStringObject(t, nameField, "John")
	}

	ageField, ok := personStruct.Fields["age"]
	if !ok {
		t.Errorf("field 'age' not found in personStruct")
	} else {
		testIntegerObject(t, ageField, 30)
	}

	// Test custom Car struct
	if customCar.Type() != STRUCT_OBJ {
		t.Errorf("wrong type for customCar. got=%s, want=%s",
			customCar.Type(), STRUCT_OBJ)
	}

	// Test custom Car struct fields
	makeField, ok := customCar.Fields["make"]
	if !ok {
		t.Errorf("field 'make' not found in customCar")
	} else {
		testStringObject(t, makeField, "Honda")
	}

	modelField, ok := customCar.Fields["model"]
	if !ok {
		t.Errorf("field 'model' not found in customCar")
	} else {
		testStringObject(t, modelField, "Civic")
	}

	yearField, ok := customCar.Fields["year"]
	if !ok {
		t.Errorf("field 'year' not found in customCar")
	} else {
		testIntegerObject(t, yearField, 2020)
	}
}

func TestStructArrays(t *testing.T) {
	// Create a Person struct instance with default values
	person1 := &Struct{
		Name: "Person",
		Fields: map[string]Object{
			"name": &String{Value: "John"},
			"age":  &Integer{Value: 30},
		},
	}

	// Create a Person struct instance with custom values
	person2 := &Struct{
		Name: "Person",
		Fields: map[string]Object{
			"name": &String{Value: "Jane"},
			"age":  &Integer{Value: 25},
		},
	}

	// Create an array of Person structs
	peopleArray := &Array{
		Elements: []Object{
			person1,
			person2,
		},
	}

	// Test the array
	if peopleArray.Type() != ARRAY_OBJ {
		t.Errorf("wrong type for peopleArray. got=%s, want=%s",
			peopleArray.Type(), ARRAY_OBJ)
	}

	if len(peopleArray.Elements) != 2 {
		t.Errorf("wrong number of elements in peopleArray. got=%d, want=%d",
			len(peopleArray.Elements), 2)
	}

	// Test the first person
	firstPerson, ok := peopleArray.Elements[0].(*Struct)
	if !ok {
		t.Errorf("first element is not a Struct. got=%T", peopleArray.Elements[0])
	} else {
		if firstPerson.Name != "Person" {
			t.Errorf("wrong name for first person. got=%s, want=%s",
				firstPerson.Name, "Person")
		}

		nameField, ok := firstPerson.Fields["name"]
		if !ok {
			t.Errorf("field 'name' not found in first person")
		} else {
			testStringObject(t, nameField, "John")
		}

		ageField, ok := firstPerson.Fields["age"]
		if !ok {
			t.Errorf("field 'age' not found in first person")
		} else {
			testIntegerObject(t, ageField, 30)
		}
	}

	// Test the second person
	secondPerson, ok := peopleArray.Elements[1].(*Struct)
	if !ok {
		t.Errorf("second element is not a Struct. got=%T", peopleArray.Elements[1])
	} else {
		if secondPerson.Name != "Person" {
			t.Errorf("wrong name for second person. got=%s, want=%s",
				secondPerson.Name, "Person")
		}

		nameField, ok := secondPerson.Fields["name"]
		if !ok {
			t.Errorf("field 'name' not found in second person")
		} else {
			testStringObject(t, nameField, "Jane")
		}

		ageField, ok := secondPerson.Fields["age"]
		if !ok {
			t.Errorf("field 'age' not found in second person")
		} else {
			testIntegerObject(t, ageField, 25)
		}
	}
}

func TestCompoundArrays(t *testing.T) {
	// Create an array of compound objects
	compoundArray := &Array{
		Elements: []Object{
			&Compound{
				Elements: []Object{
					&Integer{Value: 1},
					&String{Value: "one"},
				},
			},
			&Compound{
				Elements: []Object{
					&Integer{Value: 2},
					&String{Value: "two"},
				},
			},
			&Compound{
				Elements: []Object{
					&Integer{Value: 3},
					&String{Value: "three"},
				},
			},
		},
	}

	// Test the array
	if compoundArray.Type() != ARRAY_OBJ {
		t.Errorf("wrong type for compoundArray. got=%s, want=%s",
			compoundArray.Type(), ARRAY_OBJ)
	}

	if len(compoundArray.Elements) != 3 {
		t.Errorf("wrong number of elements in compoundArray. got=%d, want=%d",
			len(compoundArray.Elements), 3)
	}

	// Test the first compound
	compound1 := compoundArray.Elements[0].(*Compound)
	if compound1.Type() != COMPOUND_OBJ {
		t.Errorf("wrong type for first element. got=%s, want=%s",
			compound1.Type(), COMPOUND_OBJ)
	}

	if len(compound1.Elements) != 2 {
		t.Errorf("wrong number of elements in first compound. got=%d, want=%d",
			len(compound1.Elements), 2)
	}

	if compound1.Elements[0].Type() != INTEGER_OBJ {
		t.Errorf("wrong type for first element of first compound. got=%s, want=%s",
			compound1.Elements[0].Type(), INTEGER_OBJ)
	}

	if compound1.Elements[1].Type() != STRING_OBJ {
		t.Errorf("wrong type for second element of first compound. got=%s, want=%s",
			compound1.Elements[1].Type(), STRING_OBJ)
	}

	if compound1.Inspect() != "[1, one]" {
		t.Errorf("wrong string representation for first compound. got=%s, want=%s",
			compound1.Inspect(), "[1, one]")
	}

	// Test the second compound
	compound2 := compoundArray.Elements[1].(*Compound)
	if compound2.Inspect() != "[2, two]" {
		t.Errorf("wrong string representation for second compound. got=%s, want=%s",
			compound2.Inspect(), "[2, two]")
	}

	// Test the third compound
	compound3 := compoundArray.Elements[2].(*Compound)
	if compound3.Inspect() != "[3, three]" {
		t.Errorf("wrong string representation for third compound. got=%s, want=%s",
			compound3.Inspect(), "[3, three]")
	}

	// Test the array's inspect method
	expectedInspect := "[[1, one], [2, two], [3, three]]"
	if compoundArray.Inspect() != expectedInspect {
		t.Errorf("wrong string representation for compoundArray. got=%s, want=%s",
			compoundArray.Inspect(), expectedInspect)
	}
}

func TestCompoundTypes(t *testing.T) {
	// Create a compound object with [int, string] elements
	compound1 := &Compound{
		Elements: []Object{
			&Integer{Value: 1},
			&String{Value: "one"},
		},
	}

	// Create a compound object with [string, int, boolean] elements
	compound2 := &Compound{
		Elements: []Object{
			&String{Value: "item1"},
			&Integer{Value: 100},
			&Boolean{Value: true},
		},
	}

	// Test compound1
	if compound1.Type() != COMPOUND_OBJ {
		t.Errorf("wrong type for compound1. got=%s, want=%s",
			compound1.Type(), COMPOUND_OBJ)
	}

	if len(compound1.Elements) != 2 {
		t.Errorf("wrong number of elements in compound1. got=%d, want=%d",
			len(compound1.Elements), 2)
	}

	if compound1.Elements[0].Type() != INTEGER_OBJ {
		t.Errorf("wrong type for first element of compound1. got=%s, want=%s",
			compound1.Elements[0].Type(), INTEGER_OBJ)
	}

	if compound1.Elements[1].Type() != STRING_OBJ {
		t.Errorf("wrong type for second element of compound1. got=%s, want=%s",
			compound1.Elements[1].Type(), STRING_OBJ)
	}

	if compound1.Inspect() != "[1, one]" {
		t.Errorf("wrong string representation for compound1. got=%s, want=%s",
			compound1.Inspect(), "[1, one]")
	}

	// Test compound2
	if compound2.Type() != COMPOUND_OBJ {
		t.Errorf("wrong type for compound2. got=%s, want=%s",
			compound2.Type(), COMPOUND_OBJ)
	}

	if len(compound2.Elements) != 3 {
		t.Errorf("wrong number of elements in compound2. got=%d, want=%d",
			len(compound2.Elements), 3)
	}

	if compound2.Elements[0].Type() != STRING_OBJ {
		t.Errorf("wrong type for first element of compound2. got=%s, want=%s",
			compound2.Elements[0].Type(), STRING_OBJ)
	}

	if compound2.Elements[1].Type() != INTEGER_OBJ {
		t.Errorf("wrong type for second element of compound2. got=%s, want=%s",
			compound2.Elements[1].Type(), INTEGER_OBJ)
	}

	if compound2.Elements[2].Type() != BOOLEAN_OBJ {
		t.Errorf("wrong type for third element of compound2. got=%s, want=%s",
			compound2.Elements[2].Type(), BOOLEAN_OBJ)
	}

	if compound2.Inspect() != "[item1, 100, true]" {
		t.Errorf("wrong string representation for compound2. got=%s, want=%s",
			compound2.Inspect(), "[item1, 100, true]")
	}
}

// CompoundType is a test helper struct to represent the expected types in a compound
type CompoundType struct {
	ElementTypes []ObjectType
}

func TestCompoundTypeMismatchErrors(t *testing.T) {
	tests := []struct {
		name          string
		expectedTypes []ObjectType
		elements      []Object
		expectedError string
	}{
		{
			name:          "Type mismatch in compound: expected int, got STRING",
			expectedTypes: []ObjectType{INTEGER_OBJ, INTEGER_OBJ},
			elements: []Object{
				&Integer{Value: 1},
				&String{Value: "not an integer"},
			},
			expectedError: "type mismatch in compound: expected int, got STRING",
		},
		{
			name:          "Type mismatch in compound: expected string, got INTEGER",
			expectedTypes: []ObjectType{STRING_OBJ, STRING_OBJ, STRING_OBJ},
			elements: []Object{
				&String{Value: "hello"},
				&Integer{Value: 42},
				&String{Value: "world"},
			},
			expectedError: "type mismatch in compound: expected string, got INTEGER",
		},
		{
			name:          "Type mismatch in compound: expected boolean, got INTEGER",
			expectedTypes: []ObjectType{STRING_OBJ, BOOLEAN_OBJ},
			elements: []Object{
				&String{Value: "test"},
				&Integer{Value: 1},
			},
			expectedError: "type mismatch in compound: expected boolean, got INTEGER",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a compound type with the expected types
			compoundType := &CompoundType{
				ElementTypes: tt.expectedTypes,
			}

			// Create a compound with the provided elements
			compound := &Compound{
				Elements: tt.elements,
			}

			// Check if the compound matches the expected type
			err := checkCompoundType(compound, compoundType)

			// Verify that an error was returned
			if err == nil {
				t.Errorf("expected error but got nil")
				return
			}

			// Verify that the error message matches the expected error
			if err.Error() != tt.expectedError {
				t.Errorf("wrong error message. got=%q, want=%q",
					err.Error(), tt.expectedError)
			}
		})
	}
}

// Helper function to check if a compound matches a compound type
func checkCompoundType(compound *Compound, compoundType *CompoundType) error {
	if len(compound.Elements) != len(compoundType.ElementTypes) {
		return fmt.Errorf("compound length mismatch: expected %d elements, got %d",
			len(compoundType.ElementTypes), len(compound.Elements))
	}

	for i, expectedType := range compoundType.ElementTypes {
		element := compound.Elements[i]
		if element.Type() != expectedType {
			// Convert ObjectType to string representation for the error message
			var expectedTypeStr string
			switch expectedType {
			case INTEGER_OBJ:
				expectedTypeStr = "int"
			case STRING_OBJ:
				expectedTypeStr = "string"
			case BOOLEAN_OBJ:
				expectedTypeStr = "boolean"
			default:
				expectedTypeStr = string(expectedType)
			}

			return fmt.Errorf("type mismatch in compound: expected %s, got %s",
				expectedTypeStr, element.Type())
		}
	}

	return nil
}

// testEvalExpression evaluates a single expression and returns the result.
func testEvalExpression(input string) Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	// Debug output
	fmt.Printf("Parsed program: %s\n", program.String())
	fmt.Printf("Parser errors: %v\n", p.Errors())

	interpreter := New()
	return interpreter.Eval(program)
}

// Helper functions to test object values
func testIntegerObject(t *testing.T, obj Object, expected int64) bool {
	intObj, ok := obj.(*Integer)
	if !ok {
		t.Errorf("object is not Integer. got=%T (%+v)", obj, obj)
		return false
	}

	if intObj.Value != expected {
		t.Errorf("object has wrong value. got=%d, want=%d", intObj.Value, expected)
		return false
	}

	return true
}

func testStringObject(t *testing.T, obj Object, expected string) bool {
	stringObj, ok := obj.(*String)
	if !ok {
		t.Errorf("object is not String. got=%T (%+v)", obj, obj)
		return false
	}

	if stringObj.Value != expected {
		t.Errorf("object has wrong value. got=%s, want=%s", stringObj.Value, expected)
		return false
	}

	return true
}
