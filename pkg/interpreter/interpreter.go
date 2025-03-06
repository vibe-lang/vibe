package interpreter

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/vibe-lang/vibe/pkg/ast"
)

// Object represents a value in the Vibe language.
// All values in Vibe are objects, each with a type and methods to
// inspect or manipulate the value. This interface defines the common
// behavior of all objects in the language.
type Object interface {
	Type() ObjectType    // Returns the type of the object
	Inspect() string     // Returns a string representation of the object
}

// ObjectType is a string identifier for the type of an object.
// It allows for type checking and differentiation between different
// kinds of objects in the Vibe runtime.
type ObjectType string

// Object type constants define the possible types of objects in Vibe.
// These constants are used for type checking and type representation
// throughout the interpreter.
const (
	INTEGER_OBJ      = "INTEGER"      // Integer values (e.g., 42)
	FLOAT_OBJ        = "FLOAT"        // Floating-point values (e.g., 3.14)
	BOOLEAN_OBJ      = "BOOLEAN"      // Boolean values (true, false)
	STRING_OBJ       = "STRING"       // String values (e.g., "hello")
	NIL_OBJ          = "NIL"          // Nil value (absence of a value)
	RETURN_VALUE_OBJ = "RETURN_VALUE" // Wrapper for return values
	ERROR_OBJ        = "ERROR"        // Error values
	FUNCTION_OBJ     = "FUNCTION"     // Function values
	CLASS_OBJ        = "CLASS"        // Class definitions
	INSTANCE_OBJ     = "INSTANCE"     // Class instances (objects)
	ARRAY_OBJ        = "ARRAY"        // Array values
	STRUCT_OBJ       = "STRUCT"       // Struct type definitions
	STRUCT_INSTANCE_OBJ = "STRUCT_INSTANCE" // Struct instances
	COMPOUND_OBJ     = "COMPOUND"     // Compound values (tuples)
)

// Integer represents an integer value in Vibe.
// It wraps a Go int64 value and implements the Object interface.
type Integer struct {
	Value int64 // The actual integer value
}

// Type returns the type of the Integer object.
func (i *Integer) Type() ObjectType { return INTEGER_OBJ }

// Inspect returns a string representation of the Integer.
func (i *Integer) Inspect() string  { return fmt.Sprintf("%d", i.Value) }

// Float represents a floating-point value in Vibe.
// It wraps a Go float64 value and implements the Object interface.
type Float struct {
	Value float64 // The actual float value
}

// Type returns the type of the Float object.
func (f *Float) Type() ObjectType { return FLOAT_OBJ }

// Inspect returns a string representation of the Float.
func (f *Float) Inspect() string  { return fmt.Sprintf("%g", f.Value) }

// Boolean represents a boolean value in Vibe.
// It wraps a Go bool value and implements the Object interface.
type Boolean struct {
	Value bool // The actual boolean value
}

// Type returns the type of the Boolean object.
func (b *Boolean) Type() ObjectType { return BOOLEAN_OBJ }

// Inspect returns a string representation of the Boolean.
func (b *Boolean) Inspect() string  { return fmt.Sprintf("%t", b.Value) }

// String represents a string value in Vibe.
// It wraps a Go string value and implements the Object interface.
type String struct {
	Value string // The actual string value
}

// Type returns the type of the String object.
func (s *String) Type() ObjectType { return STRING_OBJ }

// Inspect returns the string value itself.
func (s *String) Inspect() string  { return s.Value }

// Nil represents a nil value in Vibe.
// It is similar to null or nil in other languages and represents
// the absence of a value or an uninitialized value.
type Nil struct{}

// Type returns the type of the Nil object.
func (n *Nil) Type() ObjectType { return NIL_OBJ }

// Inspect returns the string "nil".
func (n *Nil) Inspect() string  { return "nil" }

// ReturnValue is a wrapper object for return values in Vibe.
// It is used to propagate return values up the call stack until
// they reach the appropriate function call boundary.
type ReturnValue struct {
	Value Object // The actual value being returned
}

// Type returns the type of the ReturnValue object.
func (rv *ReturnValue) Type() ObjectType { return RETURN_VALUE_OBJ }

// Inspect delegates to the wrapped value's Inspect method.
func (rv *ReturnValue) Inspect() string  { return rv.Value.Inspect() }

// Error represents a runtime error in Vibe.
// It includes information about the error message and position
// in the source code where the error occurred.
type Error struct {
	Message string // The error message
	Line    int    // Line number where the error occurred
	Column  int    // Column position where the error occurred
}

// Type returns the type of the Error object.
func (e *Error) Type() ObjectType { return ERROR_OBJ }

// Inspect returns a formatted error message with position information.
func (e *Error) Inspect() string {
	return fmt.Sprintf("Error at [%d:%d]: %s", e.Line, e.Column, e.Message)
}

// Environment represents a scope for variables and bindings in Vibe.
// Environments form a chain, where each environment can have an outer
// (parent) environment. This chain implements lexical scoping.
type Environment struct {
	store map[string]Object // Map of variable names to their values
	outer *Environment      // Outer (enclosing) environment, or nil if this is the top level
}

// NewEnvironment creates a new environment with no outer environment.
// This is typically used for the global scope.
func NewEnvironment() *Environment {
	s := make(map[string]Object)
	return &Environment{store: s, outer: nil}
}

// NewEnclosedEnvironment creates a new environment with the given outer environment.
// This is used for creating local scopes (e.g., in function calls) that can
// access variables from their enclosing scopes.
func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := NewEnvironment()
	env.outer = outer
	return env
}

// Get retrieves a value from the environment by name.
// It searches the current environment first, then checks outer environments
// if the variable is not found locally. Returns the value and a boolean
// indicating whether the variable was found.
func (e *Environment) Get(name string) (Object, bool) {
	obj, ok := e.store[name]
	if !ok && e.outer != nil {
		obj, ok = e.outer.Get(name)
	}
	return obj, ok
}

// Set sets a value in the environment by name.
// This creates or updates a variable binding in the current environment.
// Returns the value that was set.
func (e *Environment) Set(name string, val Object) Object {
	e.store[name] = val
	return val
}

// Interpreter evaluates AST nodes to produce values during program execution.
type Interpreter struct {
	env *Environment
	lastError Object // Store the last error encountered
}

// New creates a new Interpreter with a fresh environment.
func New() *Interpreter {
	return &Interpreter{
		env: NewEnvironment(),
	}
}

// Eval evaluates an AST node and returns the resulting object.
func (i *Interpreter) Eval(node ast.Node) Object {
	result := i.eval(node)

	// Store error objects for later retrieval
	if isError(result) {
		i.lastError = result
	}

	return result
}

// eval is the internal evaluation function
func (i *Interpreter) eval(node ast.Node) Object {
	switch node := node.(type) {
	// Statements
	case *ast.Program:
		return i.evalProgram(node)
	case *ast.ExpressionStatement:
		return i.eval(node.Expression)
	case *ast.LetStatement:
		val := i.eval(node.Value)
		if isError(val) {
			return val
		}
		i.env.Set(node.Name.Value, val)
		return val
	case *ast.ReturnStatement:
		val := i.eval(node.Value)
		if isError(val) {
			return val
		}
		return &ReturnValue{Value: val}
	case *ast.BlockStatement:
		return i.evalBlockStatement(node)

	// Expressions
	case *ast.AssignmentExpression:
		return i.evalAssignmentExpression(node)
	case *ast.IntegerLiteral:
		return &Integer{Value: node.Value}
	case *ast.FloatLiteral:
		return &Float{Value: node.Value}
	case *ast.StringLiteral:
		return &String{Value: node.Value}
	case *ast.BooleanLiteral:
		return &Boolean{Value: node.Value}
	case *ast.NilLiteral:
		return &Nil{}
	case *ast.ArrayLiteral:
		elements := i.evalExpressions(node.Elements)
		if len(elements) == 1 && isError(elements[0]) {
			return elements[0]
		}
		return &Array{Elements: elements}
	case *ast.CompoundLiteral:
		elements := i.evalExpressions(node.Elements)
		if len(elements) == 1 && isError(elements[0]) {
			return elements[0]
		}
		return &Compound{Elements: elements}
	case *ast.Identifier:
		return i.evalIdentifier(node)
	case *ast.TypedIdentifier:
		// For TypedIdentifier, we just evaluate the underlying identifier
		// The type information is used during assignment, not during evaluation
		return i.eval(node.Identifier)
	case *ast.StructStatement:
		return i.evalStructStatement(node)
	case *ast.StructLiteral:
		return i.evalStructLiteral(node)
	default:
		return newError("unknown node type: %T", node)
	}
}

// evalProgram evaluates a program node.
// It evaluates each statement in the program and returns the result of the last statement.
// If a return statement is encountered, it unwraps the return value and returns it.
func (i *Interpreter) evalProgram(program *ast.Program) Object {
	var result Object

	for _, statement := range program.Statements {
		result = i.eval(statement)

		switch result := result.(type) {
		case *ReturnValue:
			return result.Value
		case *Error:
			return result
		}
	}

	return result
}

// evalBlockStatement evaluates a block statement.
// It evaluates each statement in the block and returns the result of the last statement.
// If a return statement is encountered, it returns the return value (still wrapped).
func (i *Interpreter) evalBlockStatement(block *ast.BlockStatement) Object {
	var result Object

	for _, statement := range block.Statements {
		result = i.eval(statement)

		if result != nil {
			rt := result.Type()
			if rt == RETURN_VALUE_OBJ || rt == ERROR_OBJ {
				return result
			}
		}
	}

	return result
}

// evalIdentifier evaluates an identifier.
// It looks up the identifier in the environment and returns its value.
// If the identifier is not found, it returns an error.
func (i *Interpreter) evalIdentifier(node *ast.Identifier) Object {
	if val, ok := i.env.Get(node.Value); ok {
		return val
	}

	return newError("identifier not found: %s", node.Value)
}

// Array represents an array value in Vibe.
// It contains a list of other objects.
type Array struct {
	Elements []Object // The array elements
}

// Type returns the type of the Array object.
func (a *Array) Type() ObjectType { return ARRAY_OBJ }

// Inspect returns a string representation of the Array.
func (a *Array) Inspect() string {
	var out bytes.Buffer

	elements := []string{}
	for _, e := range a.Elements {
		elements = append(elements, e.Inspect())
	}

	out.WriteString("[")
	out.WriteString(strings.Join(elements, ", "))
	out.WriteString("]")

	return out.String()
}

// evalAssignmentExpression evaluates an assignment expression.
// This handles both simple assignments (x = 5) and typed assignments (x: int = 5).
// It also performs type checking for typed assignments.
//
// Example Vibe code:
//
//	x = 5
//	name: string = "John"
//	numbers: int[] = [1, 2, 3]
func (i *Interpreter) evalAssignmentExpression(node *ast.AssignmentExpression) Object {
	val := i.eval(node.Value)
	if isError(val) {
		return val
	}

	// If there's a type annotation, validate the type
	if node.TypeAnnotation != nil {
		// Get the type name from the annotation
		var typeName string

		switch typeExpr := node.TypeAnnotation.(type) {
		case *ast.Identifier:
			// Simple type like 'int' or 'string'
			typeName = typeExpr.Value
		case *ast.ArrayTypeAnnotation:
			// Array type like 'int[]'
			baseType := ""

			// Extract the base type name
			if ident, ok := typeExpr.BaseType.(*ast.Identifier); ok {
				baseType = ident.Value
			} else if arrayType, ok := typeExpr.BaseType.(*ast.ArrayTypeAnnotation); ok {
				// Nested array type like 'int[][]'
				if baseIdent, ok := arrayType.BaseType.(*ast.Identifier); ok {
					baseType = baseIdent.Value + "[]"
				} else if compoundType, ok := arrayType.BaseType.(*ast.CompoundTypeAnnotation); ok {
					// Compound array type like '[int, string][]'
					baseType = compoundType.String()
				}
			} else if compoundType, ok := typeExpr.BaseType.(*ast.CompoundTypeAnnotation); ok {
				// Compound array type like '[int, string][]'
				baseType = compoundType.String()
			}

			typeName = baseType + "[]"
		case *ast.CompoundTypeAnnotation:
			// Compound type like '[int, string]'
			typeName = typeExpr.String()
		}

		// Validate the type
		switch typeName {
		case "int":
			if _, ok := val.(*Integer); !ok {
				return newError("type mismatch: expected int, got %s", val.Type())
			}
		case "float":
			if _, ok := val.(*Float); !ok {
				// Allow integers in float context
				if _, ok := val.(*Integer); !ok {
					return newError("type mismatch: expected float, got %s", val.Type())
				}
			}
		case "string":
			if _, ok := val.(*String); !ok {
				return newError("type mismatch: expected string, got %s", val.Type())
			}
		case "boolean":
			if _, ok := val.(*Boolean); !ok {
				return newError("type mismatch: expected boolean, got %s", val.Type())
			}
		default:
			// Check if it's an array type
			if strings.HasSuffix(typeName, "[]") {
				if err := i.validateArrayType(typeName, val); err != nil {
					return err
				}
			}
		}
	}

	i.env.Set(node.Name.Value, val)
	return val
}

// validateArrayType checks if the value matches the array type specification
// Handles both simple arrays (int[]) and nested arrays (int[][], int[][][], etc.)
func (i *Interpreter) validateArrayType(typeName string, val Object) Object {
	// Handle compound types like [int, string][]
	if strings.HasPrefix(typeName, "[") && strings.Contains(typeName, ",") {
		// Extract compound type definition and array dimensions
		compoundEndIdx := strings.Index(typeName, "]")
		if compoundEndIdx == -1 {
			return newError("invalid compound type: %s", typeName)
		}

		// Get the compound type definition
		compoundTypeDef := typeName[:compoundEndIdx+1]

		// Count the nesting level by counting the number of "[]" occurrences
		remainingType := typeName[compoundEndIdx+1:]
		nestingLevel := strings.Count(remainingType, "[]")

		// Check if this is an array of compound types
		if nestingLevel > 0 {
			// Validate the array and its elements recursively
			array, ok := val.(*Array)
			if !ok {
				return newError("type mismatch: expected array of %s, got %s",
					compoundTypeDef, val.Type())
			}

			// Parse compound type elements
			compoundTypes := parseCompoundType(compoundTypeDef)

			// Check each array element
			for _, elem := range array.Elements {
				// Each element should be a compound type
				compound, ok := elem.(*Compound)
				if !ok {
					return newError("type mismatch in array: expected compound %s, got %s",
						compoundTypeDef, elem.Type())
				}

				// Check if the compound has the right number of elements
				if len(compound.Elements) != len(compoundTypes) {
					return newError("type mismatch in array: expected compound with %d elements, got %d",
						len(compoundTypes), len(compound.Elements))
				}

				// Check each element of the compound
				for j, expectedType := range compoundTypes {
					elemValue := compound.Elements[j]

					if !matchesType(elemValue, expectedType) {
						return newError("type mismatch in compound: expected %s, got %s",
							expectedType, elemValue.Type())
					}
				}
			}

			return nil // No error
		}

		// Simple compound type (not an array)
		compound, ok := val.(*Compound)
		if !ok {
			return newError("type mismatch: expected compound %s, got %s",
				compoundTypeDef, val.Type())
		}

		// Parse compound type elements
		compoundTypes := parseCompoundType(compoundTypeDef)

		// Check if the compound has the right number of elements
		if len(compound.Elements) != len(compoundTypes) {
			return newError("type mismatch: expected compound with %d elements, got %d",
				len(compoundTypes), len(compound.Elements))
		}

		// Check each element of the compound
		for j, expectedType := range compoundTypes {
			elemValue := compound.Elements[j]

			if !matchesType(elemValue, expectedType) {
				return newError("type mismatch in compound: expected %s, got %s",
					expectedType, elemValue.Type())
			}
		}

		return nil // No error
	}

	// Handle struct arrays
	if !strings.HasPrefix(typeName, "[") && strings.HasSuffix(typeName, "[]") {
		baseType := typeName[:len(typeName)-2]

		// Check if it's a struct type
		structObj, ok := i.env.Get(baseType)
		if ok {
			_, isStruct := structObj.(*Struct)
			if isStruct {
				// Validate as struct array
				array, ok := val.(*Array)
				if !ok {
					return newError("type mismatch: expected array of %s, got %s",
						baseType, val.Type())
				}

				// Check each element of the array
				for _, elem := range array.Elements {
					instance, ok := elem.(*StructInstance)
					if !ok {
						return newError("type mismatch in array: expected %s instance, got %s",
							baseType, elem.Type())
					}

					if instance.Struct.Name != baseType {
						return newError("type mismatch in array: expected %s instance, got %s instance",
							baseType, instance.Struct.Name)
					}
				}

				return nil // No error
			}
		}
	}

	// Count the nesting level by counting the number of "[]" occurrences
	nestingLevel := strings.Count(typeName, "[]")

	// Extract the base type (e.g., "int" from "int[][]")
	baseType := typeName[:len(typeName)-(nestingLevel*2)]

	// Validate the array and its elements recursively
	return i.validateArrayWithDepth(baseType, val, nestingLevel)
}

// Helper function to parse a compound type string like "[int, string]"
// Returns a slice of type names
func parseCompoundType(compoundType string) []string {
	// Remove brackets
	inner := compoundType[1:len(compoundType)-1]

	// Split by comma
	parts := strings.Split(inner, ",")

	// Trim whitespace
	types := make([]string, len(parts))
	for i, p := range parts {
		types[i] = strings.TrimSpace(p)
	}

	return types
}

// Helper function to check if a value matches a type name
func matchesType(val Object, typeName string) bool {
	switch typeName {
	case "int":
		_, ok := val.(*Integer)
		return ok
	case "float":
		_, ok := val.(*Float)
		if ok {
			return true
		}
		// Allow integers in float context
		_, ok = val.(*Integer)
		return ok
	case "string":
		_, ok := val.(*String)
		return ok
	case "boolean":
		_, ok := val.(*Boolean)
		return ok
	default:
		// Could be a struct type
		if strings.HasSuffix(typeName, "[]") {
			// Array type
			array, ok := val.(*Array)
			if !ok {
				return false
			}

			// Check array elements if there are any
			if len(array.Elements) > 0 {
				elemTypeName := typeName[:len(typeName)-2]
				for _, elem := range array.Elements {
					if !matchesType(elem, elemTypeName) {
						return false
					}
				}
			}

			return true
		}

		// Try struct instance
		instance, ok := val.(*StructInstance)
		if ok {
			return instance.Struct.Name == typeName
		}
	}

	return false
}

// evalExpressions evaluates a list of expressions and returns a list of objects.
// Used for evaluating array elements and function arguments.
func (i *Interpreter) evalExpressions(exps []ast.Expression) []Object {
	var result []Object

	for _, e := range exps {
		evaluated := i.eval(e)
		if isError(evaluated) {
			return []Object{evaluated}
		}
		result = append(result, evaluated)
	}

	return result
}

// Helper functions

// newError creates a new error object with the given format and arguments.
// This is a convenience function for creating error objects with formatted messages.
// In a more complete implementation, this would include source position information.
func newError(format string, a ...interface{}) *Error {
	return &Error{Message: fmt.Sprintf(format, a...), Line: 0, Column: 0}
}

// isError checks if an object is an error object.
// This is a convenience function for error checking throughout the interpreter.
func isError(obj Object) bool {
	if obj != nil {
		return obj.Type() == ERROR_OBJ
	}
	return false
}

// GetEnvironment returns the current environment of the interpreter.
// This is useful for testing to check the state of variables after evaluation.
func (i *Interpreter) GetEnvironment() *Environment {
	return i.env
}

// Struct represents a struct definition in Vibe.
// It stores the structure of a user-defined type.
type Struct struct {
	Name        string
	Fields      map[string]Object
	DefaultValues map[string]Object
}

func (s *Struct) Type() ObjectType { return STRUCT_OBJ }
func (s *Struct) Inspect() string {
	var out bytes.Buffer

	out.WriteString("struct ")
	out.WriteString(s.Name)
	out.WriteString(" { ")

	// Sort the field names for consistent output
	fieldNames := make([]string, 0, len(s.DefaultValues))
	for name := range s.DefaultValues {
		fieldNames = append(fieldNames, name)
	}
	sort.Strings(fieldNames)

	fields := []string{}
	for _, name := range fieldNames {
		fields = append(fields, name + ": " + s.DefaultValues[name].Inspect())
	}
	out.WriteString(strings.Join(fields, ", "))

	out.WriteString(" }")

	return out.String()
}

// StructInstance represents an instance of a struct.
// It contains the actual field values for a specific struct instance.
type StructInstance struct {
	Struct      *Struct
	Fields      map[string]Object
}

func (si *StructInstance) Type() ObjectType { return STRUCT_INSTANCE_OBJ }
func (si *StructInstance) Inspect() string {
	var out bytes.Buffer

	out.WriteString(si.Struct.Name)
	out.WriteString("(")

	// Sort the field names for consistent output
	fieldNames := make([]string, 0, len(si.Fields))
	for name := range si.Fields {
		fieldNames = append(fieldNames, name)
	}
	sort.Strings(fieldNames)

	fields := []string{}
	for _, name := range fieldNames {
		fields = append(fields, name + ": " + si.Fields[name].Inspect())
	}
	out.WriteString(strings.Join(fields, ", "))

	out.WriteString(")")

	return out.String()
}

// Compound represents a compound value (tuple) in Vibe.
// It contains heterogeneous values in a fixed structure.
type Compound struct {
	Elements    []Object
	ElementTypes []string
}

func (c *Compound) Type() ObjectType { return COMPOUND_OBJ }
func (c *Compound) Inspect() string {
	var out bytes.Buffer

	out.WriteString("[")

	elements := []string{}
	for _, element := range c.Elements {
		elements = append(elements, element.Inspect())
	}
	out.WriteString(strings.Join(elements, ", "))

	out.WriteString("]")

	return out.String()
}

// evalStructStatement evaluates a struct statement,
// creating a new struct type definition.
func (i *Interpreter) evalStructStatement(node *ast.StructStatement) Object {
	// Create a new struct
	structObj := &Struct{
		Name: node.Name.Value,
		Fields: make(map[string]Object),
		DefaultValues: make(map[string]Object),
	}

	// Store the struct definition in the environment
	i.env.Set(node.Name.Value, structObj)

	// Evaluate default values for fields
	for _, stmt := range node.Fields {
		if exprStmt, ok := stmt.(*ast.ExpressionStatement); ok {
			if assignment, ok := exprStmt.Expression.(*ast.AssignmentExpression); ok {
				fieldName := assignment.Name.Value
				fieldValue := i.eval(assignment.Value)

				if isError(fieldValue) {
					return fieldValue
				}

				structObj.Fields[fieldName] = fieldValue
				structObj.DefaultValues[fieldName] = fieldValue
			}
		}
	}

	return structObj
}

// evalStructLiteral evaluates a struct literal (instantiation),
// creating a new instance of a struct.
func (i *Interpreter) evalStructLiteral(node *ast.StructLiteral) Object {
	// Look up the struct definition
	structObj, ok := i.env.Get(node.Type)
	if !ok {
		return newError("undefined struct type: %s", node.Type)
	}

	structType, ok := structObj.(*Struct)
	if !ok {
		return newError("not a struct type: %s", node.Type)
	}

	// Create a new struct instance with default values
	instance := &StructInstance{
		Struct: structType,
		Fields: make(map[string]Object),
	}

	// Copy default values
	for field, value := range structType.DefaultValues {
		instance.Fields[field] = value
	}

	// Apply provided field values
	for field, valueExpr := range node.Fields {
		// Check if the field exists
		if _, exists := structType.Fields[field]; !exists {
			return newError("undefined field '%s' in struct '%s'", field, node.Type)
		}

		// Evaluate the field value
		value := i.eval(valueExpr)
		if isError(value) {
			return value
		}

		instance.Fields[field] = value
	}

	return instance
}

// validateArrayWithDepth recursively validates arrays at the specified nesting depth
func (i *Interpreter) validateArrayWithDepth(baseType string, val Object, depth int) Object {
	// If depth is 0, we should check the base type
	if depth == 0 {
		typeMatches := false

		switch baseType {
		case "int":
			_, typeMatches = val.(*Integer)
		case "float":
			_, typeMatches = val.(*Float)
			// Also allow int values in float arrays
			if !typeMatches {
				_, typeMatches = val.(*Integer)
			}
		case "string":
			_, typeMatches = val.(*String)
		case "boolean":
			_, typeMatches = val.(*Boolean)
		}

		if !typeMatches {
			return newError("type mismatch in array: expected %s, got %s", baseType, val.Type())
		}

		return nil // No error
	}

	// Check if the value is an array
	array, ok := val.(*Array)
	if !ok {
		return newError("type mismatch: expected array, got %s", val.Type())
	}

	// For each element in the array, validate it at the next depth level
	for _, elem := range array.Elements {
		if err := i.validateArrayWithDepth(baseType, elem, depth-1); err != nil {
			return err
		}
	}

	return nil // No error
}

// GetLastError returns the last error encountered during evaluation
func (i *Interpreter) GetLastError() Object {
	return i.lastError
}