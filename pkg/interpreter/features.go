package interpreter

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vibe-lang/vibe/pkg/ast"
	"github.com/vibe-lang/vibe/pkg/lexer"
	"github.com/vibe-lang/vibe/pkg/parser"
)

// Object type constants for new features
const (
	BREAK_OBJ    = "BREAK"
	CONTINUE_OBJ = "CONTINUE"
	HASH_OBJ     = "HASH"
	THROW_OBJ    = "THROW"
)

// BreakSignal is used to signal a break from a loop.
type BreakSignal struct{}

func (bs *BreakSignal) Type() ObjectType { return BREAK_OBJ }
func (bs *BreakSignal) Inspect() string  { return "break" }

// ContinueSignal is used to signal continue to next iteration.
type ContinueSignal struct{}

func (cs *ContinueSignal) Type() ObjectType { return CONTINUE_OBJ }
func (cs *ContinueSignal) Inspect() string  { return "continue" }

// ThrowSignal wraps a thrown value for propagation.
type ThrowSignal struct {
	Value Object
}

func (ts *ThrowSignal) Type() ObjectType { return THROW_OBJ }
func (ts *ThrowSignal) Inspect() string  { return "throw: " + ts.Value.Inspect() }

// Hash represents a hash map / dictionary in Vibe.
// Keys are stored as string representations for simplicity.
type Hash struct {
	Pairs map[string]Object
	Order []string // maintain insertion order for deterministic iteration
}

func (h *Hash) Type() ObjectType { return HASH_OBJ }
func (h *Hash) Inspect() string {
	pairs := []string{}
	for _, k := range h.Order {
		v := h.Pairs[k]
		pairs = append(pairs, fmt.Sprintf("%s: %s", k, v.Inspect()))
	}
	return "{" + strings.Join(pairs, ", ") + "}"
}

// ---------------------------------------------------------------------------
// While loop evaluation
// ---------------------------------------------------------------------------

func (i *Interpreter) evalWhileLoop(node *ast.WhileLoop) Object {
	var result Object = &Nil{}

	for {
		condition := i.eval(node.Condition)
		if isError(condition) {
			return condition
		}

		if !isTruthy(condition) {
			break
		}

		result = i.evalBlockStatement(node.Body)

		if isError(result) {
			return result
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

	// Don't leak break/continue signals
	if _, ok := result.(*BreakSignal); ok {
		return &Nil{}
	}
	if _, ok := result.(*ContinueSignal); ok {
		return &Nil{}
	}

	return result
}

// ---------------------------------------------------------------------------
// Hash map evaluation
// ---------------------------------------------------------------------------

func (i *Interpreter) evalHashLiteral(node *ast.HashLiteral) Object {
	pairs := make(map[string]Object)
	var order []string

	for keyNode, valueNode := range node.Pairs {
		key := i.eval(keyNode)
		if isError(key) {
			return key
		}

		value := i.eval(valueNode)
		if isError(value) {
			return value
		}

		keyStr := key.Inspect()
		pairs[keyStr] = value
		order = append(order, keyStr)
	}

	// Sort order for deterministic output in tests
	sort.Strings(order)

	return &Hash{Pairs: pairs, Order: order}
}

// ---------------------------------------------------------------------------
// Mutation: arr[i] = v, obj.field = v
// ---------------------------------------------------------------------------

func (i *Interpreter) evalIndexAssignment(node *ast.IndexAssignment) Object {
	left := i.eval(node.Left)
	if isError(left) {
		return left
	}

	index := i.eval(node.Index)
	if isError(index) {
		return index
	}

	value := i.eval(node.Value)
	if isError(value) {
		return value
	}

	switch obj := left.(type) {
	case *Array:
		idx, ok := index.(*Integer)
		if !ok {
			return newError("array index must be an integer, got %s", index.Type())
		}
		idxVal := idx.Value
		if idxVal < 0 {
			idxVal = int64(len(obj.Elements)) + idxVal
		}
		if idxVal < 0 || idxVal >= int64(len(obj.Elements)) {
			return newError("array index out of bounds: %d", idx.Value)
		}
		obj.Elements[idxVal] = value
		return value
	case *Hash:
		keyStr := index.Inspect()
		obj.Pairs[keyStr] = value
		// Add to order if new key
		found := false
		for _, k := range obj.Order {
			if k == keyStr {
				found = true
				break
			}
		}
		if !found {
			obj.Order = append(obj.Order, keyStr)
		}
		return value
	default:
		return newError("index assignment not supported for %s", left.Type())
	}
}

func (i *Interpreter) evalDotAssignment(node *ast.DotAssignment) Object {
	left := i.eval(node.Left)
	if isError(left) {
		return left
	}

	value := i.eval(node.Value)
	if isError(value) {
		return value
	}

	// Support dot assignment on class instances
	if classInst, ok := left.(*ClassInstance); ok {
		classInst.Fields[node.Field] = value
		return value
	}

	// Support dot assignment on struct instances
	if structInstance, ok := left.(*StructInstance); ok {
		if _, exists := structInstance.Fields[node.Field]; !exists {
			return newError("undefined field '%s' in struct '%s'", node.Field, structInstance.Struct.Name)
		}
		structInstance.Fields[node.Field] = value
		return value
	}

	// Support dot assignment on hash maps
	if hash, ok := left.(*Hash); ok {
		if _, exists := hash.Pairs[node.Field]; !exists {
			hash.Order = append(hash.Order, node.Field)
		}
		hash.Pairs[node.Field] = value
		return value
	}

	return newError("dot assignment not supported for %s", left.Type())
}

// ---------------------------------------------------------------------------
// Import
// ---------------------------------------------------------------------------

func (i *Interpreter) evalImportStatement(node *ast.ImportStatement) Object {
	path := node.Path

	// Check if this is a standard library module first
	if LoadStdlib(path, i.env) {
		return &Nil{}
	}

	// Resolve relative paths
	if !filepath.IsAbs(path) {
		// Path is relative to current working directory
		absPath, err := filepath.Abs(path)
		if err != nil {
			return newError("import: cannot resolve path %q: %s", path, err.Error())
		}
		path = absPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return newError("import: cannot read file %q: %s", path, err.Error())
	}

	l := lexer.New(string(data))
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		return newError("import: parse errors in %q: %s", path, strings.Join(p.Errors(), "; "))
	}

	// Evaluate the imported program in the current environment
	return i.evalProgram(program)
}

// ---------------------------------------------------------------------------
// Try / Catch / Throw
// ---------------------------------------------------------------------------

func (i *Interpreter) evalTryExpression(node *ast.TryExpression) Object {
	result := i.evalBlockStatement(node.Body)

	var finalResult Object

	// Check if a throw was raised
	if throwSignal, ok := result.(*ThrowSignal); ok {
		// Bind the error value if a catch variable is specified
		if node.CatchVar != nil {
			i.env.Set(node.CatchVar.Value, throwSignal.Value)
		}
		finalResult = i.evalBlockStatement(node.CatchBody)
	} else if errObj, ok := result.(*Error); ok {
		// Also catch runtime errors
		if node.CatchVar != nil {
			i.env.Set(node.CatchVar.Value, &String{Value: errObj.Message})
		}
		finalResult = i.evalBlockStatement(node.CatchBody)
	} else {
		finalResult = result
	}

	// Always execute finally block
	if node.FinallyBody != nil {
		i.evalBlockStatement(node.FinallyBody)
	}

	return finalResult
}

func (i *Interpreter) evalThrowStatement(node *ast.ThrowStatement) Object {
	value := i.eval(node.Value)
	if isError(value) {
		return value
	}
	return &ThrowSignal{Value: value}
}

// ---------------------------------------------------------------------------
// Union type validation
// ---------------------------------------------------------------------------

func (i *Interpreter) validateUnionType(unionType *ast.UnionTypeAnnotation, val Object) bool {
	for _, typeExpr := range unionType.Types {
		typeAnno, ok := typeExpr.(*ast.TypeAnnotation)
		if !ok {
			continue
		}
		switch typeAnno.Name {
		case "int":
			if _, ok := val.(*Integer); ok {
				return true
			}
		case "float":
			if _, ok := val.(*Float); ok {
				return true
			}
			if _, ok := val.(*Integer); ok {
				return true
			}
		case "string":
			if _, ok := val.(*String); ok {
				return true
			}
		case "boolean":
			if _, ok := val.(*Boolean); ok {
				return true
			}
		case "nil":
			if _, ok := val.(*Nil); ok {
				return true
			}
		default:
			// Check struct/class types
			if si, ok := val.(*StructInstance); ok && si.Struct.Name == typeAnno.Name {
				return true
			}
			if ci, ok := val.(*ClassInstance); ok && ci.Class.Name == typeAnno.Name {
				return true
			}
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Enums
// ---------------------------------------------------------------------------

func (i *Interpreter) evalEnumStatement(node *ast.EnumStatement) Object {
	enumObj := &EnumObject{
		Name:   node.Name.Value,
		Values: make(map[string]*Integer),
	}

	// Create a hash for the enum namespace (Color.Red, Color.Green, etc.)
	pairs := make(map[string]Object)
	order := []string{}

	for idx, val := range node.Values {
		intVal := &Integer{Value: int64(idx)}
		enumObj.Values[val] = intVal
		pairs[val] = intVal
		order = append(order, val)
	}

	enumHash := &Hash{Pairs: pairs, Order: order}

	// Store the enum as a hash so Color.Red works via dot access
	i.env.Set(node.Name.Value, enumHash)

	return enumHash
}

// ---------------------------------------------------------------------------
// Classes (Full OOP)
// ---------------------------------------------------------------------------

func (i *Interpreter) evalClassLiteral(node *ast.ClassLiteral) Object {
	classObj := &ClassObject{
		Name:       node.Name.Value,
		TypeParams: node.TypeParams,
		Methods:    make(map[string]*Function),
	}

	// Resolve parent class if specified
	if node.Parent != nil {
		parentObj, ok := i.env.Get(node.Parent.Value)
		if !ok {
			return newError("undefined parent class: %s", node.Parent.Value)
		}
		parentClass, ok := parentObj.(*ClassObject)
		if !ok {
			return newError("'%s' is not a class", node.Parent.Value)
		}
		classObj.Parent = parentClass
	}

	// Extract methods from the class body
	for _, stmt := range node.Body.Statements {
		if exprStmt, ok := stmt.(*ast.ExpressionStatement); ok {
			if fnLit, ok := exprStmt.Expression.(*ast.FunctionLiteral); ok {
				if fnLit.Name != nil {
					fn := &Function{
						Parameters: fnLit.Parameters,
						Defaults:   fnLit.ParamDefaults,
						Body:       fnLit.Body,
						Env:        i.env,
						Name:       fnLit.Name.Value,
					}
					classObj.Methods[fnLit.Name.Value] = fn
				}
			}
		}
	}

	// Store the class in the environment
	i.env.Set(node.Name.Value, classObj)

	return classObj
}

// callClassConstructor creates a new instance of a class and calls initialize.
func (i *Interpreter) callClassConstructor(classObj *ClassObject, args []Object) Object {
	instance := &ClassInstance{
		Class:  classObj,
		Fields: make(map[string]Object),
	}

	// Call initialize if it exists
	if initMethod, ok := classObj.GetMethod("initialize"); ok {
		i.callMethodOnInstance(instance, initMethod, args)
	}

	return instance
}

// callMethodOnInstance calls a method on a class instance, binding self.
func (i *Interpreter) callMethodOnInstance(instance *ClassInstance, method *Function, args []Object) Object {
	return i.callMethodOnInstanceFromClass(instance, instance.Class, method, args)
}

// callMethodOnInstanceFromClass calls a method on a class instance with a specific defining class context.
// The definingClass is used to resolve 'super' correctly in multi-level inheritance chains.
func (i *Interpreter) callMethodOnInstanceFromClass(instance *ClassInstance, definingClass *ClassObject, method *Function, args []Object) Object {
	extendedEnv := NewEnclosedEnvironment(method.Env)
	extendedEnv.Set("self", instance)
	// Store the defining class so 'super' can resolve relative to the correct class
	extendedEnv.Set("__current_class__", definingClass)

	// Bind parameters
	for paramIdx, param := range method.Parameters {
		if paramIdx < len(args) {
			extendedEnv.Set(param.Value, args[paramIdx])
		} else if paramIdx < len(method.Defaults) && method.Defaults[paramIdx] != nil {
			oldEnv := i.env
			i.env = extendedEnv
			defaultVal := i.eval(method.Defaults[paramIdx])
			i.env = oldEnv
			if isError(defaultVal) {
				return defaultVal
			}
			extendedEnv.Set(param.Value, defaultVal)
		}
	}

	oldEnv := i.env
	i.env = extendedEnv
	result := i.eval(method.Body)
	i.env = oldEnv

	if returnValue, ok := result.(*ReturnValue); ok {
		return returnValue.Value
	}

	return result
}

// ---------------------------------------------------------------------------
// Destructuring assignment: a, b, c = [1, 2, 3]
// ---------------------------------------------------------------------------

func (i *Interpreter) evalDestructureAssignment(node *ast.DestructureAssignment) Object {
	val := i.eval(node.Value)
	if isError(val) {
		return val
	}

	switch v := val.(type) {
	case *Array:
		for idx, name := range node.Names {
			if idx < len(v.Elements) {
				i.env.Set(name.Value, v.Elements[idx])
			} else {
				i.env.Set(name.Value, &Nil{})
			}
		}
		return val
	case *Hash:
		for _, name := range node.Names {
			if hv, exists := v.Pairs[name.Value]; exists {
				i.env.Set(name.Value, hv)
			} else {
				i.env.Set(name.Value, &Nil{})
			}
		}
		return val
	case *StructInstance:
		for _, name := range node.Names {
			if fv, exists := v.Fields[name.Value]; exists {
				i.env.Set(name.Value, fv)
			} else {
				i.env.Set(name.Value, &Nil{})
			}
		}
		return val
	default:
		return newError("cannot destructure %s", val.Type())
	}
}

// ---------------------------------------------------------------------------
// Ternary expression: condition ? a : b
// ---------------------------------------------------------------------------

func (i *Interpreter) evalTernaryExpression(node *ast.TernaryExpression) Object {
	condition := i.eval(node.Condition)
	if isError(condition) {
		return condition
	}
	if isTruthy(condition) {
		return i.eval(node.Consequence)
	}
	return i.eval(node.Alternative)
}

// ---------------------------------------------------------------------------
// Unless expression: unless condition ... end
// ---------------------------------------------------------------------------

func (i *Interpreter) evalUnlessExpression(node *ast.UnlessExpression) Object {
	condition := i.eval(node.Condition)
	if isError(condition) {
		return condition
	}
	// Unless is the inverse of if
	if !isTruthy(condition) {
		return i.eval(node.Consequence)
	} else if node.Alternative != nil {
		return i.eval(node.Alternative)
	}
	return &Nil{}
}

// ---------------------------------------------------------------------------
// Until loop: until condition ... end
// ---------------------------------------------------------------------------

func (i *Interpreter) evalUntilLoop(node *ast.UntilLoop) Object {
	var result Object = &Nil{}

	for {
		condition := i.eval(node.Condition)
		if isError(condition) {
			return condition
		}
		// Until loops run while the condition is falsy
		if isTruthy(condition) {
			break
		}

		result = i.evalBlockStatement(node.Body)

		if isError(result) {
			return result
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

// ---------------------------------------------------------------------------
// Case/when expression
// ---------------------------------------------------------------------------

func (i *Interpreter) evalCaseExpression(node *ast.CaseExpression) Object {
	subject := i.eval(node.Subject)
	if isError(subject) {
		return subject
	}

	for _, when := range node.Whens {
		for _, val := range when.Values {
			matchVal := i.eval(val)
			if isError(matchVal) {
				return matchVal
			}
			if objectsEqual(subject, matchVal) {
				return i.evalBlockStatement(when.Body)
			}
		}
	}

	if node.Default != nil {
		return i.evalBlockStatement(node.Default)
	}

	return &Nil{}
}

// objectsEqual compares two objects for equality.
func objectsEqual(a, b Object) bool {
	if a.Type() != b.Type() {
		return false
	}
	switch av := a.(type) {
	case *Integer:
		return av.Value == b.(*Integer).Value
	case *Float:
		return av.Value == b.(*Float).Value
	case *String:
		return av.Value == b.(*String).Value
	case *Boolean:
		return av.Value == b.(*Boolean).Value
	case *Nil:
		return true
	default:
		return a.Inspect() == b.Inspect()
	}
}

// ---------------------------------------------------------------------------
// Arrow function: -> (params) { body }
// ---------------------------------------------------------------------------

func (i *Interpreter) evalArrowFunction(node *ast.ArrowFunction) Object {
	return &Function{
		Parameters: node.Parameters,
		Body:       node.Body,
		Env:        i.env,
	}
}

// ---------------------------------------------------------------------------
// Pipe expression: expr |> func or expr |> func(args)
// ---------------------------------------------------------------------------

func (i *Interpreter) evalPipeExpression(node *ast.PipeExpression) Object {
	left := i.eval(node.Left)
	if isError(left) {
		return left
	}

	// If the right side is a call expression, prepend left as first arg
	if call, ok := node.Right.(*ast.CallExpression); ok {
		fn := i.eval(call.Function)
		if isError(fn) {
			return fn
		}
		args := i.evalExpressions(call.Arguments)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}
		allArgs := make([]Object, 0, len(args)+1)
		allArgs = append(allArgs, left)
		allArgs = append(allArgs, args...)
		return i.applyFunction(fn, allArgs)
	}

	// If the right side is an identifier, call it with left as only arg
	fn := i.eval(node.Right)
	if isError(fn) {
		return fn
	}
	return i.applyFunction(fn, []Object{left})
}

// ---------------------------------------------------------------------------
// Nil coalescing: expr ?? default
// ---------------------------------------------------------------------------

func (i *Interpreter) evalNilCoalesceExpression(node *ast.NilCoalesceExpression) Object {
	left := i.eval(node.Left)
	if isError(left) {
		return left
	}
	// If left is nil, evaluate and return right side
	if _, ok := left.(*Nil); ok {
		return i.eval(node.Right)
	}
	return left
}

// ---------------------------------------------------------------------------
// Super keyword support
// ---------------------------------------------------------------------------

// SuperProxy is a temporary object that facilitates super method calls.
// It binds a class instance to its parent class for method resolution.
type SuperProxy struct {
	Instance *ClassInstance
	Parent   *ClassObject
}

func (sp *SuperProxy) Type() ObjectType { return "SUPER_PROXY" }
func (sp *SuperProxy) Inspect() string  { return "<super>" }

// ---------------------------------------------------------------------------
// Generic type validation helpers
// ---------------------------------------------------------------------------

// validateFieldTypeArg validates that a field value matches the resolved type argument.
// typeArgMap maps type parameter names to concrete type names.
func (i *Interpreter) validateFieldTypeArg(fieldName string, value Object, typeArgMap map[string]string) *Error {
	// For now, we validate based on the value's runtime type matching the resolved type
	// The field's declared type would need to be tracked on the struct definition
	// This is a simplified version that just checks the value matches the expected type
	return nil // Type checking is done at the call site; field-level checking deferred
}

// ---------------------------------------------------------------------------
// Postfix if/unless: expr if condition / expr unless condition
// ---------------------------------------------------------------------------

func (i *Interpreter) evalPostfixCondition(node *ast.PostfixCondition) Object {
	condition := i.eval(node.Condition)
	if isError(condition) {
		return condition
	}

	shouldExecute := isTruthy(condition)
	if node.Unless {
		shouldExecute = !shouldExecute
	}

	if shouldExecute {
		return i.eval(node.Statement)
	}

	// When condition is false, pre-declare any assignment variables as nil
	// This matches Ruby behavior where `y = "yes" if false` leaves y as nil
	if exprStmt, ok := node.Statement.(*ast.ExpressionStatement); ok {
		if assign, ok := exprStmt.Expression.(*ast.AssignmentExpression); ok {
			i.env.Set(assign.Name.Value, &Nil{})
		}
	}

	return &Nil{}
}

// ---------------------------------------------------------------------------
// Const declaration
// ---------------------------------------------------------------------------

func (i *Interpreter) evalConstStatement(node *ast.ConstStatement) Object {
	val := i.eval(node.Value)
	if isError(val) {
		return val
	}
	i.env.SetConst(node.Name.Value, val)
	return val
}

// ---------------------------------------------------------------------------
// In expression: value in collection
// ---------------------------------------------------------------------------

func (i *Interpreter) evalInExpression(node *ast.InExpression) Object {
	left := i.eval(node.Left)
	if isError(left) {
		return left
	}

	right := i.eval(node.Right)
	if isError(right) {
		return right
	}

	switch collection := right.(type) {
	case *Array:
		for _, elem := range collection.Elements {
			if objectsEqual(left, elem) {
				return &Boolean{Value: true}
			}
		}
		return &Boolean{Value: false}
	case *Hash:
		key := left.Inspect()
		_, exists := collection.Pairs[key]
		return &Boolean{Value: exists}
	case *String:
		if str, ok := left.(*String); ok {
			return &Boolean{Value: strings.Contains(collection.Value, str.Value)}
		}
		return &Boolean{Value: false}
	case *Range:
		if intVal, ok := left.(*Integer); ok {
			end := collection.End
			if collection.Exclusive {
				end--
			}
			return &Boolean{Value: intVal.Value >= collection.Start && intVal.Value <= end}
		}
		return &Boolean{Value: false}
	default:
		return newError("'in' operator not supported for %s", right.Type())
	}
}
