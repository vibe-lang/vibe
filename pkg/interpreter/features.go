package interpreter

import (
	"fmt"
	"os"
	"path/filepath"
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
type BreakSignal struct{ Value Object }

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
	Pairs   map[string]Object
	Order   []string          // maintain insertion order for deterministic iteration
	IsKw    bool              // built from keyword arguments at a call site
	Keys    map[string]Object // original key objects when they are not strings
	Default Object
	Frozen  bool
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

// ---------------------------------------------------------------------------
// Hash map evaluation
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Mutation: arr[i] = v, obj.field = v
// ---------------------------------------------------------------------------

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
// Destructuring assignment: a, b, c = [1, 2, 3]
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Case/when expression
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Arrow function: -> (params) { body }
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Pipe expression: expr |> func or expr |> func(args)
// ---------------------------------------------------------------------------

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
	After    *ClassObject // class whose method is running; lookup resumes after it
}

func (sp *SuperProxy) Type() ObjectType { return "SUPER_PROXY" }
func (sp *SuperProxy) Inspect() string  { return "<super>" }

// ---------------------------------------------------------------------------
// Postfix if/unless: expr if condition / expr unless condition
// ---------------------------------------------------------------------------

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
