package interpreter

import (
	"bytes"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/vibe-lang/vibe/pkg/ast"
)

// Object represents a value in the Vibe language.
// All values in Vibe are objects, each with a type and methods to
// inspect or manipulate the value. This interface defines the common
// behavior of all objects in the language.
type Object interface {
	Type() ObjectType // Returns the type of the object
	Inspect() string  // Returns a string representation of the object
}

// ObjectType is a string identifier for the type of an object.
// It allows for type checking and differentiation between different
// kinds of objects in the Vibe runtime.
type ObjectType string

// Object type constants define the possible types of objects in Vibe.
// These constants are used for type checking and type representation
// throughout the interpreter.
const (
	INTEGER_OBJ         = "INTEGER"         // Integer values (e.g., 42)
	FLOAT_OBJ           = "FLOAT"           // Floating-point values (e.g., 3.14)
	BOOLEAN_OBJ         = "BOOLEAN"         // Boolean values (true, false)
	STRING_OBJ          = "STRING"          // String values (e.g., "hello")
	NIL_OBJ             = "NIL"             // Nil value (absence of a value)
	RETURN_VALUE_OBJ    = "RETURN_VALUE"    // Wrapper for return values
	ERROR_OBJ           = "ERROR"           // Error values
	FUNCTION_OBJ        = "FUNCTION"        // Function values
	CLASS_OBJ           = "CLASS"           // Class definitions
	INSTANCE_OBJ        = "INSTANCE"        // Class instances (objects)
	ARRAY_OBJ           = "ARRAY"           // Array values
	STRUCT_OBJ          = "STRUCT"          // Struct type definitions
	STRUCT_INSTANCE_OBJ = "STRUCT_INSTANCE" // Struct instances
	COMPOUND_OBJ        = "COMPOUND"        // Compound values (tuples)
	BUILTIN_OBJ         = "BUILTIN"         // Builtin functions
)

// Integer represents an integer value in Vibe.
// It wraps a Go int64 value and implements the Object interface.
type Integer struct {
	Value int64 // The actual integer value
}

// Type returns the type of the Integer object.
func (i *Integer) Type() ObjectType { return INTEGER_OBJ }

// Inspect returns a string representation of the Integer.
func (i *Integer) Inspect() string { return fmt.Sprintf("%d", i.Value) }

// Float represents a floating-point value in Vibe.
// It wraps a Go float64 value and implements the Object interface.
type Float struct {
	Value float64 // The actual float value
}

// Type returns the type of the Float object.
func (f *Float) Type() ObjectType { return FLOAT_OBJ }

// Inspect returns a string representation of the Float.
func (f *Float) Inspect() string { return fmt.Sprintf("%g", f.Value) }

// Boolean represents a boolean value in Vibe.
// It wraps a Go bool value and implements the Object interface.
type Boolean struct {
	Value bool // The actual boolean value
}

// Type returns the type of the Boolean object.
func (b *Boolean) Type() ObjectType { return BOOLEAN_OBJ }

// Inspect returns a string representation of the Boolean.
func (b *Boolean) Inspect() string { return fmt.Sprintf("%t", b.Value) }

// String represents a string value in Vibe.
// It wraps a Go string value and implements the Object interface.
type String struct {
	Value string // The actual string value
}

// Type returns the type of the String object.
func (s *String) Type() ObjectType { return STRING_OBJ }

// Inspect returns the string value itself.
func (s *String) Inspect() string { return s.Value }

// Nil represents a nil value in Vibe.
// It is similar to null or nil in other languages and represents
// the absence of a value or an uninitialized value.
type Nil struct{}

// Type returns the type of the Nil object.
func (n *Nil) Type() ObjectType { return NIL_OBJ }

// Inspect returns the string "nil".
func (n *Nil) Inspect() string { return "nil" }

// ReturnValue is a wrapper object for return values in Vibe.
// It is used to propagate return values up the call stack until
// they reach the appropriate function call boundary.
type ReturnValue struct {
	Value Object // The actual value being returned
}

// Type returns the type of the ReturnValue object.
func (rv *ReturnValue) Type() ObjectType { return RETURN_VALUE_OBJ }

// Inspect delegates to the wrapped value's Inspect method.
func (rv *ReturnValue) Inspect() string { return rv.Value.Inspect() }

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

// Builtin represents a builtin function in Vibe.
type Builtin struct {
	Fn func(args ...Object) Object
}

// Type returns the type of the Builtin object.
func (b *Builtin) Type() ObjectType { return BUILTIN_OBJ }

// Inspect returns a string representation of the Builtin.
func (b *Builtin) Inspect() string { return "builtin function" }

// Environment represents a scope for variables and bindings in Vibe.
// Environments form a chain, where each environment can have an outer
// (parent) environment. This chain implements lexical scoping.
type Environment struct {
	store  map[string]Object // Map of variable names to their values
	consts map[string]bool   // Set of constant variable names
	outer  *Environment      // Outer (enclosing) environment, or nil if this is the top level
}

// NewEnvironment creates a new environment with no outer environment.
// This is typically used for the global scope.
func NewEnvironment() *Environment {
	s := make(map[string]Object)
	c := make(map[string]bool)
	return &Environment{store: s, consts: c, outer: nil}
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
// Returns the value that was set, or an error if the name is a constant.
func (e *Environment) Set(name string, val Object) Object {
	if e.IsConst(name) {
		return newError("cannot reassign constant '%s'", name)
	}
	e.store[name] = val
	return val
}

// SetConst sets a constant value in the environment.
// Constants cannot be reassigned after being set.
func (e *Environment) SetConst(name string, val Object) Object {
	e.store[name] = val
	e.consts[name] = true
	return val
}

// IsConst checks if a name is a constant in the current or any outer environment.
func (e *Environment) IsConst(name string) bool {
	if e.consts[name] {
		return true
	}
	if e.outer != nil {
		return e.outer.IsConst(name)
	}
	return false
}

// Keys returns all variable names in the current environment scope.
func (e *Environment) Keys() []string {
	keys := make([]string, 0, len(e.store))
	for k := range e.store {
		keys = append(keys, k)
	}
	return keys
}

// Interpreter evaluates AST nodes to produce values during program execution.
type Interpreter struct {
	env       *Environment
	lastError Object // Store the last error encountered
}

// New creates a new Interpreter with a fresh environment.
func New() *Interpreter {
	i := &Interpreter{
		env: NewEnvironment(),
	}

	registerBuiltins(i.env)

	return i
}

// applyFunction applies a function to the given arguments.
func (i *Interpreter) applyFunction(fn Object, args []Object) Object {
	switch fn := fn.(type) {
	case *Builtin:
		return fn.Fn(args...)
	case *Function:
		return i.applyUserFunction(fn, args)
	case *ClassObject:
		return i.callClassConstructor(fn, args)
	case *SuperProxy:
		// super(args) — call the parent's initialize method
		definingClass, initMethod := findMethodInChain(fn.Parent, "initialize")
		if initMethod != nil {
			return i.callMethodOnInstanceFromClass(fn.Instance, definingClass, initMethod, args)
		}
		return newError("parent class '%s' has no initialize method", fn.Parent.Name)
	default:
		return newError("not a function: %s", fn.Type())
	}
}

// applyUserFunction applies a user-defined function to the given arguments.
func (i *Interpreter) applyUserFunction(fn *Function, args []Object) Object {
	return i.applyUserFunctionWithTypeArgs(fn, args, nil)
}

// applyUserFunctionWithTypeArgs applies a user-defined function with optional explicit type args.
func (i *Interpreter) applyUserFunctionWithTypeArgs(fn *Function, args []Object, typeArgs []string) Object {
	// Create a new enclosed environment for the function
	extendedEnv := NewEnclosedEnvironment(fn.Env)

	// If the function has type parameters, resolve them
	if len(fn.TypeParams) > 0 {
		resolvedTypes := make(map[string]string) // TypeParam -> concrete type

		// If explicit type args provided, use them
		if len(typeArgs) > 0 {
			if len(typeArgs) != len(fn.TypeParams) {
				return newError("wrong number of type arguments for %s: expected %d, got %d",
					fn.Name, len(fn.TypeParams), len(typeArgs))
			}
			for idx, tp := range fn.TypeParams {
				resolvedTypes[tp] = typeArgs[idx]
			}
		} else {
			// Infer type parameters from argument types
			for paramIdx, paramType := range fn.ParamTypes {
				if paramIdx >= len(args) {
					break
				}
				if paramType != nil {
					typeAnno := extractTypeName(paramType)
					if typeAnno != "" {
						// Check if this type name is one of the type params
						for _, tp := range fn.TypeParams {
							if typeAnno == tp {
								// Infer this type param from the argument
								argTypeName := objectTypeName(args[paramIdx])
								if existing, ok := resolvedTypes[tp]; ok {
									if existing != argTypeName {
										return newError("type parameter %s inconsistently inferred: %s vs %s",
											tp, existing, argTypeName)
									}
								} else {
									resolvedTypes[tp] = argTypeName
								}
							}
						}
					}
				}
			}
		}

		// Validate argument types against resolved type params
		for paramIdx, paramType := range fn.ParamTypes {
			if paramIdx >= len(args) {
				break
			}
			if paramType != nil {
				typeAnno := extractTypeName(paramType)
				if resolvedType, ok := resolvedTypes[typeAnno]; ok {
					argTypeName := objectTypeName(args[paramIdx])
					if argTypeName != resolvedType {
						return newError("type mismatch for parameter '%s': expected %s (from %s = %s), got %s",
							fn.Parameters[paramIdx].Value, resolvedType, typeAnno, resolvedType, argTypeName)
					}
				}
			}
		}
	}

	// Bind parameters to arguments, using defaults for missing args
	if fn.Variadic && len(fn.Parameters) > 0 {
		// The last parameter is variadic: collect extra args into an array
		lastParamIdx := len(fn.Parameters) - 1
		// Bind normal (non-variadic) parameters
		for paramIdx := 0; paramIdx < lastParamIdx; paramIdx++ {
			if paramIdx < len(args) {
				extendedEnv.Set(fn.Parameters[paramIdx].Value, args[paramIdx])
			} else if paramIdx < len(fn.Defaults) && fn.Defaults[paramIdx] != nil {
				oldEnv := i.env
				i.env = extendedEnv
				defaultVal := i.eval(fn.Defaults[paramIdx])
				i.env = oldEnv
				if isError(defaultVal) {
					return defaultVal
				}
				extendedEnv.Set(fn.Parameters[paramIdx].Value, defaultVal)
			}
		}
		// Collect remaining args into an array for the variadic parameter
		varArgs := []Object{}
		if lastParamIdx < len(args) {
			varArgs = args[lastParamIdx:]
		}
		extendedEnv.Set(fn.Parameters[lastParamIdx].Value, &Array{Elements: varArgs})
	} else {
		for paramIdx, param := range fn.Parameters {
			if paramIdx < len(args) {
				extendedEnv.Set(param.Value, args[paramIdx])
			} else if paramIdx < len(fn.Defaults) && fn.Defaults[paramIdx] != nil {
				// Use default value
				oldEnv := i.env
				i.env = extendedEnv
				defaultVal := i.eval(fn.Defaults[paramIdx])
				i.env = oldEnv
				if isError(defaultVal) {
					return defaultVal
				}
				extendedEnv.Set(param.Value, defaultVal)
			}
		}
	}

	// Save the current environment and switch to the function environment
	oldEnv := i.env
	i.env = extendedEnv

	// Evaluate the function body
	result := i.eval(fn.Body)

	// Restore the environment
	i.env = oldEnv

	// Unwrap return values
	if returnValue, ok := result.(*ReturnValue); ok {
		return returnValue.Value
	}

	return result
}

// extractTypeName extracts a type name string from a type annotation AST expression.
func extractTypeName(expr ast.Expression) string {
	switch t := expr.(type) {
	case *ast.TypeAnnotation:
		return t.Name
	case *ast.Identifier:
		return t.Value
	default:
		return ""
	}
}

// objectTypeName returns the Vibe type name for an object (e.g., "int", "string").
func objectTypeName(obj Object) string {
	switch obj.(type) {
	case *Integer:
		return "int"
	case *Float:
		return "float"
	case *String:
		return "string"
	case *Boolean:
		return "boolean"
	case *Nil:
		return "nil"
	case *Array:
		return "array"
	case *Hash:
		return "hash"
	case *Function:
		return "function"
	case *StructInstance:
		return obj.(*StructInstance).Struct.Name
	case *ClassInstance:
		return obj.(*ClassInstance).Class.Name
	default:
		return string(obj.Type())
	}
}

// Eval evaluates an AST node and returns the resulting object.
func (i *Interpreter) Eval(node ast.Node) Object {
	builtinInterpreter = i
	result := i.eval(node)

	// Store error objects for later retrieval
	if isError(result) {
		i.lastError = result
	}

	return result
}

// eval is the internal evaluation function
func (i *Interpreter) eval(node ast.Node) Object {
	if node == nil {
		return newError("nil node passed to eval")
	}

	switch node := node.(type) {
	// Statements
	case *ast.Program:
		return i.evalProgram(node)
	case *ast.ExpressionStatement:
		return i.eval(node.Expression)
	case *ast.ReturnStatement:
		val := i.eval(node.Value)
		if isError(val) {
			return val
		}
		return &ReturnValue{Value: val}
	case *ast.BlockStatement:
		return i.evalBlockStatement(node)
	case *ast.ForLoop:
		return i.evalForLoop(node)
	case *ast.Identifier:
		// Handle 'self' as a special identifier
		if node.Value == "self" {
			if val, ok := i.env.Get("self"); ok {
				return val
			}
			return newError("'self' used outside of a class method")
		}
		// Handle 'super' as a special identifier
		if node.Value == "super" {
			selfObj, ok := i.env.Get("self")
			if !ok {
				return newError("'super' used outside of a class method")
			}
			instance, ok := selfObj.(*ClassInstance)
			if !ok {
				return newError("'super' used outside of a class context")
			}
			// Use the current defining class to find the parent, not instance.Class
			// This is critical for correct super resolution in multi-level inheritance
			currentClassObj, ok := i.env.Get("__current_class__")
			if !ok {
				// Fallback to instance's class if __current_class__ not set
				if instance.Class.Parent == nil {
					return newError("'super' used in a class with no parent")
				}
				return &SuperProxy{Instance: instance, Parent: instance.Class.Parent}
			}
			currentClass, ok := currentClassObj.(*ClassObject)
			if !ok {
				return newError("'super' used outside of a class context")
			}
			if currentClass.Parent == nil {
				return newError("'super' used in a class with no parent")
			}
			return &SuperProxy{Instance: instance, Parent: currentClass.Parent}
		}
		return i.evalIdentifier(node)
	case *ast.TypedIdentifier:
		// For TypedIdentifier, we just evaluate the underlying identifier
		// The type information is used during assignment, not during evaluation
		return i.eval(node.Identifier)
	case *ast.AssignmentExpression:
		return i.evalAssignmentExpression(node)
	case *ast.InfixExpression:
		return i.evalInfixExpression(node)
	case *ast.CallExpression:
		// Handle method-style calls: obj.method(args) -> method(obj, args)
		// When the function is a DotExpression, evaluate the left side first.
		if dotExpr, ok := node.Function.(*ast.DotExpression); ok {
			left := i.eval(dotExpr.Left)
			if isError(left) {
				return left
			}
			methodName := dotExpr.Field.Value

			// Handle super.method(args) — call parent class method
			if superProxy, ok := left.(*SuperProxy); ok {
				// Find the method and which class defines it
				definingClass, method := findMethodInChain(superProxy.Parent, methodName)
				if method != nil {
					args := i.evalExpressions(node.Arguments)
					if len(args) == 1 && isError(args[0]) {
						return args[0]
					}
					return i.callMethodOnInstanceFromClass(superProxy.Instance, definingClass, method, args)
				}
				return newError("undefined method '%s' in parent class '%s'", methodName, superProxy.Parent.Name)
			}

			// If left is a class instance, look up the method on the class
			if classInst, ok := left.(*ClassInstance); ok {
				// Check instance fields first (may be a function)
				if field, ok := classInst.Fields[methodName]; ok {
					if fn, ok := field.(*Function); ok {
						args := i.evalExpressions(node.Arguments)
						if len(args) == 1 && isError(args[0]) {
							return args[0]
						}
						return i.callMethodOnInstance(classInst, fn, args)
					}
					// Field exists but not a function — error
					return newError("'%s' is not a method", methodName)
				}
				// Look up method on the class
				if method, ok := classInst.Class.GetMethod(methodName); ok {
					args := i.evalExpressions(node.Arguments)
					if len(args) == 1 && isError(args[0]) {
						return args[0]
					}
					return i.callMethodOnInstance(classInst, method, args)
				}
				// Fall through to builtin methods
			}

			// If left is a struct instance, try field access (may return a function)
			if structInst, ok := left.(*StructInstance); ok {
				if field, ok := structInst.Fields[methodName]; ok {
					args := i.evalExpressions(node.Arguments)
					if len(args) == 1 && isError(args[0]) {
						return args[0]
					}
					return i.applyFunction(field, args)
				}
			}
			// Not a struct or field not found — treat as method call
			fn, ok := i.env.Get(methodName)
			if !ok {
				return newError("undefined method: %s", methodName)
			}
			args := i.evalExpressions(node.Arguments)
			if len(args) == 1 && isError(args[0]) {
				return args[0]
			}
			// Prepend the receiver (left) as the first argument
			allArgs := make([]Object, 0, len(args)+1)
			allArgs = append(allArgs, left)
			allArgs = append(allArgs, args...)
			return i.applyFunction(fn, allArgs)
		}
		function := i.eval(node.Function)
		if isError(function) {
			return function
		}
		args := i.evalExpressions(node.Arguments)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}
		// If explicit type args are provided, pass them through
		if len(node.TypeArgs) > 0 {
			if fn, ok := function.(*Function); ok {
				return i.applyUserFunctionWithTypeArgs(fn, args, node.TypeArgs)
			}
		}
		return i.applyFunction(function, args)
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
	case *ast.IndexExpression:
		return i.evalIndexExpression(node)
	case *ast.DotExpression:
		return i.evalDotExpression(node)
	case *ast.CompoundLiteral:
		elements := i.evalExpressions(node.Elements)
		if len(elements) == 1 && isError(elements[0]) {
			return elements[0]
		}
		return &Compound{Elements: elements}
	case *ast.PrefixExpression:
		return i.evalPrefixExpression(node)
	case *ast.IfExpression:
		return i.evalIfExpression(node)
	case *ast.StringInterpolationLiteral:
		return i.evalStringInterpolation(node)
	case *ast.RangeExpression:
		return i.evalRangeExpression(node)
	case *ast.RangeCallExpression:
		return i.evalRangeCallExpression(node)
	case *ast.FunctionLiteral:
		return i.evalFunctionLiteral(node)
	case *ast.ClassLiteral:
		return i.evalClassLiteral(node)
	case *ast.EnumStatement:
		return i.evalEnumStatement(node)
	case *ast.StructStatement:
		return i.evalStructStatement(node)
	case *ast.StructLiteral:
		return i.evalStructLiteral(node)
	case *ast.WhileLoop:
		return i.evalWhileLoop(node)
	case *ast.BreakStatement:
		return &BreakSignal{}
	case *ast.ContinueStatement:
		return &ContinueSignal{}
	case *ast.HashLiteral:
		return i.evalHashLiteral(node)
	case *ast.IndexAssignment:
		return i.evalIndexAssignment(node)
	case *ast.DotAssignment:
		return i.evalDotAssignment(node)
	case *ast.ImportStatement:
		return i.evalImportStatement(node)
	case *ast.TryExpression:
		return i.evalTryExpression(node)
	case *ast.ThrowStatement:
		return i.evalThrowStatement(node)
	case *ast.TernaryExpression:
		return i.evalTernaryExpression(node)
	case *ast.UnlessExpression:
		return i.evalUnlessExpression(node)
	case *ast.UntilLoop:
		return i.evalUntilLoop(node)
	case *ast.CaseExpression:
		return i.evalCaseExpression(node)
	case *ast.ArrowFunction:
		return i.evalArrowFunction(node)
	case *ast.PipeExpression:
		return i.evalPipeExpression(node)
	case *ast.InExpression:
		return i.evalInExpression(node)
	case *ast.NilCoalesceExpression:
		return i.evalNilCoalesceExpression(node)
	case *ast.DestructureAssignment:
		return i.evalDestructureAssignment(node)
	case *ast.ConstStatement:
		return i.evalConstStatement(node)
	case *ast.PostfixCondition:
		return i.evalPostfixCondition(node)
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
			if rt == RETURN_VALUE_OBJ || rt == ERROR_OBJ || rt == BREAK_OBJ || rt == CONTINUE_OBJ || rt == THROW_OBJ {
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
		// Handle union types
		if unionType, ok := node.TypeAnnotation.(*ast.UnionTypeAnnotation); ok {
			if !i.validateUnionType(unionType, val) {
				types := []string{}
				for _, t := range unionType.Types {
					types = append(types, t.String())
				}
				return newError("type mismatch: expected %s, got %s", strings.Join(types, " | "), val.Type())
			}
		} else {
			// Get the type name from the annotation
			var typeName string

			switch typeExpr := node.TypeAnnotation.(type) {
			case *ast.Identifier:
				typeName = typeExpr.Value
			case *ast.ArrayTypeAnnotation:
				baseType := ""
				if ident, ok := typeExpr.BaseType.(*ast.Identifier); ok {
					baseType = ident.Value
				} else if arrayType, ok := typeExpr.BaseType.(*ast.ArrayTypeAnnotation); ok {
					if baseIdent, ok := arrayType.BaseType.(*ast.Identifier); ok {
						baseType = baseIdent.Value + "[]"
					} else if compoundType, ok := arrayType.BaseType.(*ast.CompoundTypeAnnotation); ok {
						baseType = compoundType.String()
					}
				} else if compoundType, ok := typeExpr.BaseType.(*ast.CompoundTypeAnnotation); ok {
					baseType = compoundType.String()
				}
				typeName = baseType + "[]"
			case *ast.CompoundTypeAnnotation:
				typeName = typeExpr.String()
			}

			switch typeName {
			case "int":
				if _, ok := val.(*Integer); !ok {
					return newError("type mismatch: expected int, got %s", val.Type())
				}
			case "float":
				if _, ok := val.(*Float); !ok {
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
				if strings.HasSuffix(typeName, "[]") {
					if err := i.validateArrayType(typeName, val); err != nil {
						return err
					}
				}
			}
		}
	}

	result := i.env.Set(node.Name.Value, val)
	if isError(result) {
		return result
	}
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
	inner := compoundType[1 : len(compoundType)-1]

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
	Name          string
	TypeParams    []string // Generic type parameters (e.g., ["A", "B"])
	Fields        map[string]Object
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
		fields = append(fields, name+": "+s.DefaultValues[name].Inspect())
	}
	out.WriteString(strings.Join(fields, ", "))

	out.WriteString(" }")

	return out.String()
}

// StructInstance represents an instance of a struct.
// It contains the actual field values for a specific struct instance.
type StructInstance struct {
	Struct   *Struct
	TypeArgs []string // Resolved type arguments (e.g., ["int", "string"])
	Fields   map[string]Object
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
		fields = append(fields, name+": "+si.Fields[name].Inspect())
	}
	out.WriteString(strings.Join(fields, ", "))

	out.WriteString(")")

	return out.String()
}

// Compound represents a compound value (tuple) in Vibe.
// It contains heterogeneous values in a fixed structure.
type Compound struct {
	Elements     []Object
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

// evalStructStatement evaluates a struct definition statement.
// It creates a new struct type and stores it in the environment.
func (i *Interpreter) evalStructStatement(node *ast.StructStatement) Object {
	// Create a new struct object
	structObj := &Struct{
		Name:          node.Name.Value,
		TypeParams:    node.TypeParams,
		Fields:        make(map[string]Object),
		DefaultValues: make(map[string]Object),
	}

	// Store the struct definition in the environment
	i.env.Set(node.Name.Value, structObj)

	// Process field declarations
	for _, stmt := range node.Fields {
		if exprStmt, ok := stmt.(*ast.ExpressionStatement); ok {
			// Check if it's an assignment with a value
			if assignment, ok := exprStmt.Expression.(*ast.AssignmentExpression); ok {
				fieldName := assignment.Name.Value

				// If it has a value, evaluate it
				if assignment.Value != nil {
					fieldValue := i.eval(assignment.Value)
					if isError(fieldValue) {
						return fieldValue
					}
					structObj.Fields[fieldName] = fieldValue
					structObj.DefaultValues[fieldName] = fieldValue
				} else if assignment.TypeAnnotation != nil {
					// If it only has a type annotation, use nil as default value
					structObj.Fields[fieldName] = &Nil{}
					structObj.DefaultValues[fieldName] = &Nil{}
				}
			} else if typedIdent, ok := exprStmt.Expression.(*ast.TypedIdentifier); ok {
				// Handle field declarations like: name: string
				fieldName := typedIdent.Identifier.Value
				structObj.Fields[fieldName] = &Nil{} // Default value for fields with just type annotations
				structObj.DefaultValues[fieldName] = &Nil{}
			} else if ident, ok := exprStmt.Expression.(*ast.Identifier); ok {
				// Handle field declarations like: name
				fieldName := ident.Value
				structObj.Fields[fieldName] = &Nil{} // Default value for fields without type annotations
				structObj.DefaultValues[fieldName] = &Nil{}
			}
		}
	}

	return structObj
}

// evalStructLiteral evaluates a struct literal (instantiation),
// creating a new instance of a struct or class.
func (i *Interpreter) evalStructLiteral(node *ast.StructLiteral) Object {
	// Look up the struct/class definition
	structObj, ok := i.env.Get(node.Type)
	if !ok {
		return newError("undefined type: %s", node.Type)
	}

	// If it's a builtin function (e.g., Channel()), call it
	if builtin, ok := structObj.(*Builtin); ok {
		args := []Object{}
		for _, expr := range node.Fields {
			val := i.eval(expr)
			if isError(val) {
				return val
			}
			args = append(args, val)
		}
		return builtin.Fn(args...)
	}

	// If it's a class, use class constructor
	if classObj, ok := structObj.(*ClassObject); ok {
		// Validate type args count if provided
		if len(node.TypeArgs) > 0 && len(classObj.TypeParams) > 0 {
			if len(node.TypeArgs) != len(classObj.TypeParams) {
				return newError("wrong number of type arguments for %s: expected %d, got %d",
					node.Type, len(classObj.TypeParams), len(node.TypeArgs))
			}
		}
		// Evaluate field expressions as positional args
		args := []Object{}
		for _, expr := range node.Fields {
			val := i.eval(expr)
			if isError(val) {
				return val
			}
			args = append(args, val)
		}
		instance := i.callClassConstructor(classObj, args)
		// Store type args on the instance
		if classInst, ok := instance.(*ClassInstance); ok && len(node.TypeArgs) > 0 {
			classInst.TypeArgs = node.TypeArgs
		}
		return instance
	}

	structType, ok := structObj.(*Struct)
	if !ok {
		return newError("not a struct type: %s", node.Type)
	}

	// Validate type args count if provided
	if len(node.TypeArgs) > 0 && len(structType.TypeParams) > 0 {
		if len(node.TypeArgs) != len(structType.TypeParams) {
			return newError("wrong number of type arguments for %s: expected %d, got %d",
				node.Type, len(structType.TypeParams), len(node.TypeArgs))
		}
	}

	// Create a new struct instance with default values
	instance := &StructInstance{
		Struct:   structType,
		TypeArgs: node.TypeArgs,
		Fields:   make(map[string]Object),
	}

	// Copy default values
	for field, value := range structType.DefaultValues {
		instance.Fields[field] = value
	}

	// Build type param -> concrete type mapping if type args are provided
	var typeArgMap map[string]string
	if len(node.TypeArgs) > 0 && len(structType.TypeParams) > 0 {
		typeArgMap = make(map[string]string)
		for idx, tp := range structType.TypeParams {
			if idx < len(node.TypeArgs) {
				typeArgMap[tp] = node.TypeArgs[idx]
			}
		}
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

		// If we have type arg mappings, validate field type
		if typeArgMap != nil {
			// Check if the field's declared type is a type parameter
			if err := i.validateFieldTypeArg(field, value, typeArgMap); err != nil {
				return err
			}
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

// evalForLoop evaluates a for loop statement.
// It iterates over a collection (array, range, hash, etc.) and executes the body for each element.
// Supports: for x in array, for k, v in hash
func (i *Interpreter) evalForLoop(node *ast.ForLoop) Object {
	// Evaluate the collection to iterate over
	collection := i.eval(node.Collection)
	if isError(collection) {
		return collection
	}

	// Handle hash iteration: for k, v in hash
	if hash, ok := collection.(*Hash); ok && node.ValueIterator != nil {
		return i.evalForLoopHash(node, hash)
	}

	// Convert range to array if needed
	if rangeObj, ok := collection.(*Range); ok {
		collection = rangeObj.ToArray()
	}

	// Check if the collection is an array
	array, ok := collection.(*Array)
	if !ok {
		return newError("for loop collection must be an array, range, or hash, got %s", collection.Type())
	}

	var result Object = &Nil{}
	for idx, element := range array.Elements {
		i.env.Set(node.Iterator.Value, element)
		// If there's a value iterator on an array, set it to the index
		if node.ValueIterator != nil {
			i.env.Set(node.ValueIterator.Value, &Integer{Value: int64(idx)})
		}

		result = i.evalBlockStatement(node.Body)

		if isError(result) {
			break
		}
		if _, ok := result.(*BreakSignal); ok {
			break
		}
		if _, ok := result.(*ContinueSignal); ok {
			continue
		}
		if _, ok := result.(*ReturnValue); ok {
			return result
		}
	}

	// Don't leak break/continue signals out of the loop
	if _, ok := result.(*BreakSignal); ok {
		return &Nil{}
	}
	if _, ok := result.(*ContinueSignal); ok {
		return &Nil{}
	}

	return result
}

// evalForLoopHash iterates over a hash, binding key and value to the iterators.
func (i *Interpreter) evalForLoopHash(node *ast.ForLoop, hash *Hash) Object {
	var result Object = &Nil{}

	for _, key := range hash.Order {
		value := hash.Pairs[key]
		i.env.Set(node.Iterator.Value, &String{Value: key})
		i.env.Set(node.ValueIterator.Value, value)

		result = i.evalBlockStatement(node.Body)

		if isError(result) {
			break
		}
		if _, ok := result.(*BreakSignal); ok {
			break
		}
		if _, ok := result.(*ContinueSignal); ok {
			continue
		}
		if _, ok := result.(*ReturnValue); ok {
			return result
		}
	}

	if _, ok := result.(*BreakSignal); ok {
		return &Nil{}
	}
	if _, ok := result.(*ContinueSignal); ok {
		return &Nil{}
	}

	return result
}

// evalIndexExpression evaluates an index expression for array access.
// Example: arr[0]
func (i *Interpreter) evalIndexExpression(node *ast.IndexExpression) Object {
	// Evaluate the left side (the array)
	left := i.eval(node.Left)
	if isError(left) {
		return left
	}

	// Evaluate the index
	index := i.eval(node.Index)
	if isError(index) {
		return index
	}

	switch obj := left.(type) {
	case *Array:
		idx, ok := index.(*Integer)
		if !ok {
			return newError("array index must be an integer, got %s", index.Type())
		}
		idxVal := idx.Value
		// Support negative indexing: arr[-1] is the last element
		if idxVal < 0 {
			idxVal = int64(len(obj.Elements)) + idxVal
		}
		if idxVal < 0 || idxVal >= int64(len(obj.Elements)) {
			return newError("array index out of bounds: %d", idx.Value)
		}
		return obj.Elements[idxVal]
	case *String:
		idx, ok := index.(*Integer)
		if !ok {
			return newError("string index must be an integer, got %s", index.Type())
		}
		runes := []rune(obj.Value)
		idxVal := idx.Value
		if idxVal < 0 {
			idxVal = int64(len(runes)) + idxVal
		}
		if idxVal < 0 || idxVal >= int64(len(runes)) {
			return newError("string index out of bounds: %d", idx.Value)
		}
		return &String{Value: string(runes[idxVal])}
	case *Hash:
		keyStr := index.Inspect()
		val, exists := obj.Pairs[keyStr]
		if !exists {
			return &Nil{}
		}
		return val
	default:
		return newError("index operator not supported for %s", left.Type())
	}
}

// evalDotExpression evaluates a dot expression for struct field access
// or as a zero-argument method call on any value (e.g., "hello".len).
// Example: person.name, [1,2,3].len, "hello".type
func (i *Interpreter) evalDotExpression(node *ast.DotExpression) Object {
	// Evaluate the left side
	left := i.eval(node.Left)
	if isError(left) {
		return left
	}

	fieldName := node.Field.Value

	// Check if the left side is a SuperProxy (for super.method())
	if superProxy, ok := left.(*SuperProxy); ok {
		// Look up the method on the parent class, tracking which class defines it
		definingClass, method := findMethodInChain(superProxy.Parent, fieldName)
		if method != nil {
			return i.callMethodOnInstanceFromClass(superProxy.Instance, definingClass, method, []Object{})
		}
		return newError("undefined method '%s' in parent class '%s'", fieldName, superProxy.Parent.Name)
	}

	// Check if the left side is a class instance
	if classInst, ok := left.(*ClassInstance); ok {
		// Check instance fields
		if field, ok := classInst.Fields[fieldName]; ok {
			return field
		}
		// Check methods — return as zero-arg call
		if method, found := classInst.Class.GetMethod(fieldName); found {
			return i.callMethodOnInstance(classInst, method, []Object{})
		}
		// Fall through to builtin method lookup
	}

	// Check if the left side is a struct instance
	if structInstance, ok := left.(*StructInstance); ok {
		// First check if the field exists in the instance
		if field, ok := structInstance.Fields[fieldName]; ok {
			return field
		}

		// If not found in the instance, check if it exists in the struct type definition
		if structInstance.Struct != nil {
			if _, ok := structInstance.Struct.Fields[fieldName]; ok {
				return newError("field '%s' exists in struct definition but not in instance", fieldName)
			}
		}

		// Field not found on struct — fall through to try as method call
	}

	// Check if the left side is a hash — support dot access for keys
	if hash, ok := left.(*Hash); ok {
		if val, exists := hash.Pairs[fieldName]; exists {
			return val
		}
		// Key not found — fall through to try as method call
	}

	// Not a struct or field not found — try as a zero-argument method call.
	// This allows "hello".len, [1,2,3].first, 42.type, etc.
	if fn, ok := i.env.Get(fieldName); ok {
		return i.applyFunction(fn, []Object{left})
	}

	// Nothing matched
	if structInstance, ok := left.(*StructInstance); ok {
		return newError("undefined field '%s' in struct '%s'", fieldName, structInstance.Struct.Name)
	}
	return newError("undefined method '%s' for %s", fieldName, left.Type())
}

// evalInfixExpression evaluates an infix expression like 1 + 2, a == b, etc.
func (i *Interpreter) evalInfixExpression(node *ast.InfixExpression) Object {
	// Short-circuit evaluation for logical operators
	if node.Operator == "&&" {
		left := i.eval(node.Left)
		if isError(left) {
			return left
		}
		if !isTruthy(left) {
			return &Boolean{Value: false}
		}
		right := i.eval(node.Right)
		if isError(right) {
			return right
		}
		return &Boolean{Value: isTruthy(right)}
	}
	if node.Operator == "||" {
		left := i.eval(node.Left)
		if isError(left) {
			return left
		}
		if isTruthy(left) {
			return &Boolean{Value: true}
		}
		right := i.eval(node.Right)
		if isError(right) {
			return right
		}
		return &Boolean{Value: isTruthy(right)}
	}

	left := i.eval(node.Left)
	if isError(left) {
		return left
	}

	right := i.eval(node.Right)
	if isError(right) {
		return right
	}

	switch {
	// Integer operations
	case left.Type() == INTEGER_OBJ && right.Type() == INTEGER_OBJ:
		return i.evalIntegerInfixExpression(node.Operator, left, right)

	// Float operations
	case left.Type() == FLOAT_OBJ && right.Type() == FLOAT_OBJ:
		return i.evalFloatInfixExpression(node.Operator, left, right)

	// Mixed integer and float operations
	case (left.Type() == INTEGER_OBJ && right.Type() == FLOAT_OBJ) ||
		(left.Type() == FLOAT_OBJ && right.Type() == INTEGER_OBJ):
		return i.evalMixedNumberInfixExpression(node.Operator, left, right)

	// String operations
	case left.Type() == STRING_OBJ && right.Type() == STRING_OBJ:
		return i.evalStringInfixExpression(node.Operator, left, right)

	// Boolean operations
	case left.Type() == BOOLEAN_OBJ && right.Type() == BOOLEAN_OBJ:
		return i.evalBooleanInfixExpression(node.Operator, left, right)

	// Array operations
	case left.Type() == ARRAY_OBJ && right.Type() == ARRAY_OBJ:
		return i.evalArrayInfixExpression(node.Operator, left, right)

	// String * integer repetition
	case node.Operator == "*" && left.Type() == STRING_OBJ && right.Type() == INTEGER_OBJ:
		str := left.(*String).Value
		count := right.(*Integer).Value
		if count < 0 {
			return newError("string repetition count must be non-negative")
		}
		return &String{Value: strings.Repeat(str, int(count))}

	// String + any other type: auto-convert to string and concatenate
	case node.Operator == "+" && (left.Type() == STRING_OBJ || right.Type() == STRING_OBJ):
		return &String{Value: left.Inspect() + right.Inspect()}

	default:
		return newError("unsupported operator %s for types %s and %s",
			node.Operator, left.Type(), right.Type())
	}
}

// evalArrayInfixExpression evaluates infix expressions with array operands
func (i *Interpreter) evalArrayInfixExpression(operator string, left, right Object) Object {
	leftArray := left.(*Array)
	rightArray := right.(*Array)

	switch operator {
	case "+":
		// Array concatenation
		newElements := make([]Object, len(leftArray.Elements)+len(rightArray.Elements))
		copy(newElements, leftArray.Elements)
		copy(newElements[len(leftArray.Elements):], rightArray.Elements)
		return &Array{Elements: newElements}
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

// evalIntegerInfixExpression evaluates an infix expression with integer operands.
func (i *Interpreter) evalIntegerInfixExpression(operator string, left, right Object) Object {
	leftVal := left.(*Integer).Value
	rightVal := right.(*Integer).Value

	switch operator {
	case "+":
		return &Integer{Value: leftVal + rightVal}
	case "-":
		return &Integer{Value: leftVal - rightVal}
	case "*":
		return &Integer{Value: leftVal * rightVal}
	case "/":
		if rightVal == 0 {
			return newError("division by zero")
		}
		return &Integer{Value: leftVal / rightVal}
	case "%":
		if rightVal == 0 {
			return newError("division by zero")
		}
		return &Integer{Value: leftVal % rightVal}
	case "**":
		return &Integer{Value: int64(math.Pow(float64(leftVal), float64(rightVal)))}
	case "<":
		return &Boolean{Value: leftVal < rightVal}
	case ">":
		return &Boolean{Value: leftVal > rightVal}
	case "<=":
		return &Boolean{Value: leftVal <= rightVal}
	case ">=":
		return &Boolean{Value: leftVal >= rightVal}
	case "==":
		return &Boolean{Value: leftVal == rightVal}
	case "!=":
		return &Boolean{Value: leftVal != rightVal}
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

// evalFloatInfixExpression evaluates an infix expression with float operands.
func (i *Interpreter) evalFloatInfixExpression(operator string, left, right Object) Object {
	leftVal := left.(*Float).Value
	rightVal := right.(*Float).Value

	switch operator {
	case "+":
		return &Float{Value: leftVal + rightVal}
	case "-":
		return &Float{Value: leftVal - rightVal}
	case "*":
		return &Float{Value: leftVal * rightVal}
	case "/":
		if rightVal == 0.0 {
			return newError("division by zero")
		}
		return &Float{Value: leftVal / rightVal}
	case "**":
		return &Float{Value: math.Pow(leftVal, rightVal)}
	case "<":
		return &Boolean{Value: leftVal < rightVal}
	case ">":
		return &Boolean{Value: leftVal > rightVal}
	case "<=":
		return &Boolean{Value: leftVal <= rightVal}
	case ">=":
		return &Boolean{Value: leftVal >= rightVal}
	case "==":
		return &Boolean{Value: leftVal == rightVal}
	case "!=":
		return &Boolean{Value: leftVal != rightVal}
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

// evalMixedNumberInfixExpression evaluates an infix expression with mixed number operands (int and float).
func (i *Interpreter) evalMixedNumberInfixExpression(operator string, left, right Object) Object {
	var leftVal, rightVal float64

	if left.Type() == INTEGER_OBJ {
		leftVal = float64(left.(*Integer).Value)
	} else {
		leftVal = left.(*Float).Value
	}

	if right.Type() == INTEGER_OBJ {
		rightVal = float64(right.(*Integer).Value)
	} else {
		rightVal = right.(*Float).Value
	}

	switch operator {
	case "+":
		return &Float{Value: leftVal + rightVal}
	case "-":
		return &Float{Value: leftVal - rightVal}
	case "*":
		return &Float{Value: leftVal * rightVal}
	case "/":
		if rightVal == 0.0 {
			return newError("division by zero")
		}
		return &Float{Value: leftVal / rightVal}
	case "**":
		return &Float{Value: math.Pow(leftVal, rightVal)}
	case "<":
		return &Boolean{Value: leftVal < rightVal}
	case ">":
		return &Boolean{Value: leftVal > rightVal}
	case "<=":
		return &Boolean{Value: leftVal <= rightVal}
	case ">=":
		return &Boolean{Value: leftVal >= rightVal}
	case "==":
		return &Boolean{Value: leftVal == rightVal}
	case "!=":
		return &Boolean{Value: leftVal != rightVal}
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

// evalStringInfixExpression evaluates an infix expression with string operands.
func (i *Interpreter) evalStringInfixExpression(operator string, left, right Object) Object {
	leftVal := left.(*String).Value
	rightVal := right.(*String).Value

	switch operator {
	case "+":
		return &String{Value: leftVal + rightVal}
	case "==":
		return &Boolean{Value: leftVal == rightVal}
	case "!=":
		return &Boolean{Value: leftVal != rightVal}
	case "<":
		return &Boolean{Value: leftVal < rightVal}
	case ">":
		return &Boolean{Value: leftVal > rightVal}
	case "<=":
		return &Boolean{Value: leftVal <= rightVal}
	case ">=":
		return &Boolean{Value: leftVal >= rightVal}
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

// evalBooleanInfixExpression evaluates an infix expression with boolean operands.
func (i *Interpreter) evalBooleanInfixExpression(operator string, left, right Object) Object {
	leftVal := left.(*Boolean).Value
	rightVal := right.(*Boolean).Value

	switch operator {
	case "==":
		return &Boolean{Value: leftVal == rightVal}
	case "!=":
		return &Boolean{Value: leftVal != rightVal}
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

// evalPrefixExpression evaluates a prefix expression like -5 or !true.
func (i *Interpreter) evalPrefixExpression(node *ast.PrefixExpression) Object {
	right := i.eval(node.Right)
	if isError(right) {
		return right
	}

	switch node.Operator {
	case "!":
		return i.evalBangOperator(right)
	case "-":
		return i.evalMinusPrefixOperator(right)
	default:
		return newError("unknown operator: %s%s", node.Operator, right.Type())
	}
}

func (i *Interpreter) evalBangOperator(right Object) Object {
	switch obj := right.(type) {
	case *Boolean:
		return &Boolean{Value: !obj.Value}
	case *Nil:
		return &Boolean{Value: true}
	default:
		return &Boolean{Value: false}
	}
}

func (i *Interpreter) evalMinusPrefixOperator(right Object) Object {
	switch obj := right.(type) {
	case *Integer:
		return &Integer{Value: -obj.Value}
	case *Float:
		return &Float{Value: -obj.Value}
	default:
		return newError("unknown operator: -%s", right.Type())
	}
}

// evalIfExpression evaluates an if expression.
func (i *Interpreter) evalIfExpression(node *ast.IfExpression) Object {
	condition := i.eval(node.Condition)
	if isError(condition) {
		return condition
	}

	if isTruthy(condition) {
		return i.eval(node.Consequence)
	} else if node.Alternative != nil {
		return i.eval(node.Alternative)
	} else {
		return &Nil{}
	}
}

// isTruthy determines if an object is truthy.
// nil and false are falsy, everything else is truthy.
func isTruthy(obj Object) bool {
	switch obj := obj.(type) {
	case *Nil:
		return false
	case *Boolean:
		return obj.Value
	default:
		return true
	}
}

// evalStringInterpolation evaluates a string with interpolated expressions.
func (i *Interpreter) evalStringInterpolation(node *ast.StringInterpolationLiteral) Object {
	var out bytes.Buffer

	for idx, part := range node.Parts {
		out.WriteString(part)

		if idx < len(node.Expressions) {
			val := i.eval(node.Expressions[idx])
			if isError(val) {
				return val
			}
			out.WriteString(val.Inspect())
		}
	}

	return &String{Value: out.String()}
}

// Range represents a range of integer values.
type Range struct {
	Start     int64
	End       int64
	Exclusive bool
}

func (r *Range) Type() ObjectType { return "RANGE" }
func (r *Range) Inspect() string {
	if r.Exclusive {
		return fmt.Sprintf("%d...%d", r.Start, r.End)
	}
	return fmt.Sprintf("%d..%d", r.Start, r.End)
}

// ToArray converts a Range to an Array of Integers.
func (r *Range) ToArray() *Array {
	var elements []Object
	if r.Start <= r.End {
		end := r.End
		if r.Exclusive {
			end--
		}
		for val := r.Start; val <= end; val++ {
			elements = append(elements, &Integer{Value: val})
		}
	} else {
		end := r.End
		if r.Exclusive {
			end++
		}
		for val := r.Start; val >= end; val-- {
			elements = append(elements, &Integer{Value: val})
		}
	}
	return &Array{Elements: elements}
}

// evalRangeExpression evaluates a range expression like 1..5 or 1...5.
func (i *Interpreter) evalRangeExpression(node *ast.RangeExpression) Object {
	start := i.eval(node.Start)
	if isError(start) {
		return start
	}

	end := i.eval(node.End)
	if isError(end) {
		return end
	}

	startInt, ok := start.(*Integer)
	if !ok {
		return newError("range start must be an integer, got %s", start.Type())
	}

	endInt, ok := end.(*Integer)
	if !ok {
		return newError("range end must be an integer, got %s", end.Type())
	}

	return &Range{
		Start:     startInt.Value,
		End:       endInt.Value,
		Exclusive: node.Exclusive,
	}
}

// evalRangeCallExpression evaluates a Range() constructor call.
func (i *Interpreter) evalRangeCallExpression(node *ast.RangeCallExpression) Object {
	start := i.eval(node.Start)
	if isError(start) {
		return start
	}

	end := i.eval(node.End)
	if isError(end) {
		return end
	}

	startInt, ok := start.(*Integer)
	if !ok {
		return newError("Range start must be an integer, got %s", start.Type())
	}

	endInt, ok := end.(*Integer)
	if !ok {
		return newError("Range end must be an integer, got %s", end.Type())
	}

	return &Range{
		Start:     startInt.Value,
		End:       endInt.Value,
		Exclusive: node.Exclusive,
	}
}

// ClassObject represents a class definition in Vibe.
type ClassObject struct {
	Name       string
	TypeParams []string // Generic type parameters (e.g., ["T", "U"])
	Parent     *ClassObject
	Methods    map[string]*Function
}

func (c *ClassObject) Type() ObjectType { return CLASS_OBJ }
func (c *ClassObject) Inspect() string  { return "<class " + c.Name + ">" }

// GetMethod looks up a method on the class or any parent class.
func (c *ClassObject) GetMethod(name string) (*Function, bool) {
	if fn, ok := c.Methods[name]; ok {
		return fn, true
	}
	if c.Parent != nil {
		return c.Parent.GetMethod(name)
	}
	return nil, false
}

// findMethodInChain searches for a method starting from the given class,
// returning both the class that defines the method and the method itself.
// This is critical for correct super resolution in multi-level inheritance.
func findMethodInChain(class *ClassObject, name string) (*ClassObject, *Function) {
	if fn, ok := class.Methods[name]; ok {
		return class, fn
	}
	if class.Parent != nil {
		return findMethodInChain(class.Parent, name)
	}
	return nil, nil
}

// ClassInstance represents an instance of a class.
type ClassInstance struct {
	Class    *ClassObject
	TypeArgs []string // Resolved type arguments (e.g., ["int", "string"])
	Fields   map[string]Object
}

func (ci *ClassInstance) Type() ObjectType { return INSTANCE_OBJ }
func (ci *ClassInstance) Inspect() string {
	var out bytes.Buffer
	out.WriteString("<" + ci.Class.Name + " instance")
	if len(ci.Fields) > 0 {
		fields := []string{}
		for k, v := range ci.Fields {
			fields = append(fields, k+": "+v.Inspect())
		}
		sort.Strings(fields)
		out.WriteString(" ")
		out.WriteString(strings.Join(fields, ", "))
	}
	out.WriteString(">")
	return out.String()
}

// Function represents a user-defined function.
type Function struct {
	Parameters []*ast.Identifier
	Defaults   []ast.Expression // Default values for parameters (nil = required)
	Body       *ast.BlockStatement
	Env        *Environment
	Name       string
	TypeParams []string         // Generic type parameters (e.g., ["T", "U"])
	ParamTypes []ast.Expression // Parameter type annotations for type checking
	ReturnType ast.Expression   // Return type annotation
	Variadic   bool             // Whether the last parameter is variadic (*args)
}

func (f *Function) Type() ObjectType { return FUNCTION_OBJ }
func (f *Function) Inspect() string {
	var out bytes.Buffer
	params := []string{}
	for _, p := range f.Parameters {
		params = append(params, p.String())
	}
	out.WriteString("fn(")
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(") { ... }")
	return out.String()
}

// evalFunctionLiteral evaluates a function literal, creating a closure.
func (i *Interpreter) evalFunctionLiteral(node *ast.FunctionLiteral) Object {
	fn := &Function{
		Parameters: node.Parameters,
		Defaults:   node.ParamDefaults,
		Body:       node.Body,
		Env:        i.env,
		TypeParams: node.TypeParams,
		ParamTypes: node.ParamTypes,
		ReturnType: node.ReturnType,
		Variadic:   node.Variadic,
	}

	// If it's a named function (def), bind it in the environment
	if node.Name != nil {
		fn.Name = node.Name.Value
		i.env.Set(node.Name.Value, fn)
	}

	return fn
}
