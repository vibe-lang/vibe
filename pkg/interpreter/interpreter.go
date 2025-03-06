package interpreter

import (
	"bytes"
	"fmt"
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

// Interpreter represents the Vibe language interpreter.
// It evaluates AST nodes and executes the Vibe program.
type Interpreter struct {
	env *Environment // The current environment for variable bindings
}

// New creates a new Interpreter with a fresh global environment.
// The global environment will be initialized with any built-in
// functions or values that should be available to all Vibe programs.
func New() *Interpreter {
	return &Interpreter{
		env: NewEnvironment(),
	}
}

// Eval evaluates an AST node and returns the resulting Object.
// This is the main entry point for the interpreter. It dispatches
// to appropriate evaluation functions based on the node type.
//
// The interpretation follows these principles:
// - Expressions produce values
// - Statements produce side effects and may return values
// - Control structures affect the flow of execution
//
// Example:
//
//	program := parser.ParseProgram()
//	interpreter := interpreter.New()
//	result := interpreter.Eval(program)
func (i *Interpreter) Eval(node ast.Node) Object {
	switch node := node.(type) {
	// Statements
	case *ast.Program:
		return i.evalProgram(node)
	case *ast.ExpressionStatement:
		return i.Eval(node.Expression)
	case *ast.LetStatement:
		val := i.Eval(node.Value)
		if isError(val) {
			return val
		}
		i.env.Set(node.Name.Value, val)
		return val
	case *ast.ReturnStatement:
		val := i.Eval(node.Value)
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
	case *ast.Identifier:
		return i.evalIdentifier(node)
	default:
		return newError("unknown node type: %T", node)
	}
}

// evalProgram evaluates a program node, which is the root of the AST.
// It evaluates each statement in sequence and handles return values and errors.
// If a return value is encountered, its value is returned directly rather than
// the ReturnValue wrapper, as the program boundary unwraps return values.
func (i *Interpreter) evalProgram(program *ast.Program) Object {
	var result Object

	for _, statement := range program.Statements {
		result = i.Eval(statement)

		switch result := result.(type) {
		case *ReturnValue:
			return result.Value
		case *Error:
			return result
		}
	}

	return result
}

// evalBlockStatement evaluates a block of statements.
// It creates a new scope for the block and evaluates each statement in sequence.
// If a return value or error is encountered, it is propagated up immediately.
func (i *Interpreter) evalBlockStatement(block *ast.BlockStatement) Object {
	var result Object

	for _, statement := range block.Statements {
		result = i.Eval(statement)

		if result != nil {
			rt := result.Type()
			if rt == RETURN_VALUE_OBJ || rt == ERROR_OBJ {
				return result
			}
		}
	}

	return result
}

// evalIdentifier evaluates an identifier by looking up its value in the environment.
// If the identifier is not found, an error is returned.
func (i *Interpreter) evalIdentifier(node *ast.Identifier) Object {
	val, ok := i.env.Get(node.Value)
	if !ok {
		return newError("identifier not found: %s", node.Value)
	}
	return val
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
// It evaluates the right-hand side expression, then sets a variable with the identifier
// name to that value. If a type annotation is present, it checks if the value matches
// the declared type.
//
// Example Vibe code:
//
//   x = 5                  // Simple assignment
//   name: string = "John"  // Typed assignment
//   nums: int[] = [1, 2]   // Typed array assignment
func (i *Interpreter) evalAssignmentExpression(node *ast.AssignmentExpression) Object {
	val := i.Eval(node.Value)
	if isError(val) {
		return val
	}

	// If there's a type annotation, validate the type
	if node.Type != nil {
		typeName := node.Type.Name
		isArrayType := false
		elementTypeName := ""

		// Check if it's an array type (ends with [])
		if len(typeName) > 2 && typeName[len(typeName)-2:] == "[]" {
			isArrayType = true
			elementTypeName = typeName[:len(typeName)-2]
		}

		if isArrayType {
			// Validate array type
			array, ok := val.(*Array)
			if !ok {
				return newError("type mismatch: expected array of %s, got %s",
					elementTypeName, val.Type())
			}

			// Validate array element types
			for _, elem := range array.Elements {
				elemTypeMatches := false
				switch elementTypeName {
				case "int":
					_, elemTypeMatches = elem.(*Integer)
				case "float":
					_, elemTypeMatches = elem.(*Float)
					// Also allow int values in float arrays
					if !elemTypeMatches {
						_, elemTypeMatches = elem.(*Integer)
					}
				case "string":
					_, elemTypeMatches = elem.(*String)
				case "boolean":
					_, elemTypeMatches = elem.(*Boolean)
				}

				if !elemTypeMatches {
					return newError("type mismatch in array: expected %s, got %s",
						elementTypeName, elem.Type())
				}
			}
		} else {
			// Regular non-array type checking
			typeMatches := false

			switch typeName {
			case "int":
				_, typeMatches = val.(*Integer)
			case "float":
				_, typeMatches = val.(*Float)
			case "string":
				_, typeMatches = val.(*String)
			case "boolean":
				_, typeMatches = val.(*Boolean)
			}

			if !typeMatches {
				return newError("type mismatch: expected %s, got %s",
					typeName, val.Type())
			}
		}
	}

	// Set the variable in the environment
	i.env.Set(node.Name.Value, val)
	return val
}

// evalExpressions evaluates a list of expressions and returns a list of objects.
// Used for evaluating array elements and function arguments.
func (i *Interpreter) evalExpressions(exps []ast.Expression) []Object {
	var result []Object

	for _, e := range exps {
		evaluated := i.Eval(e)
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