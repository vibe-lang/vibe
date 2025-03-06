package ast

import (
	"bytes"
	"strings"

	"github.com/vibe-lang/vibe/pkg/lexer"
)

// Node is the basic interface all AST nodes implement.
// The Abstract Syntax Tree (AST) is a tree representation of the syntactic
// structure of the Vibe source code. It serves as the intermediate representation
// between the parser and the interpreter.
//
// Every node in the AST must implement this interface, providing methods to:
// - Get the literal value of the token associated with the node
// - Generate a string representation of the node for debugging
type Node interface {
	TokenLiteral() string // Returns the literal value of the token
	String() string       // Returns a string representation of the node
}

// Statement is a node that represents a statement in Vibe.
// Statements are language constructs that perform actions but don't
// necessarily produce values (e.g., variable declarations, return statements).
// They are typically executed for their side effects.
//
// All statement nodes implement both the Node interface and the statementNode marker method.
type Statement interface {
	Node
	statementNode() // Marker method to identify statement nodes
}

// Expression is a node that represents an expression in Vibe.
// Expressions are language constructs that produce values (e.g., literals,
// function calls, arithmetic operations).
//
// All expression nodes implement both the Node interface and the expressionNode marker method.
type Expression interface {
	Node
	expressionNode() // Marker method to identify expression nodes
}

// Program represents the entire program as a collection of statements.
// It is the root node of the AST and contains all the statements that
// make up a Vibe program.
type Program struct {
	Statements []Statement // Slice of all statements in the program
}

// TokenLiteral returns the literal value of the first token in the program.
// If the program is empty, returns an empty string.
func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}
	return ""
}

// String returns a string representation of the Program.
// This method concatenates the string representations of all statements
// in the program, which is useful for debugging.
func (p *Program) String() string {
	var out bytes.Buffer

	for _, s := range p.Statements {
		out.WriteString(s.String())
	}

	return out.String()
}

// LetStatement represents a variable declaration statement in Vibe.
// Example in Vibe code: `let x = 5;` or `let name: String = "John";`
//
// A LetStatement consists of:
// - The 'let' token
// - The identifier being declared
// - An optional type annotation
// - The value being assigned to the identifier
type LetStatement struct {
	Token lexer.Token     // the LET token
	Name  *Identifier     // The variable name
	Value Expression      // The expression that produces the value
	Type  *TypeAnnotation // Optional type annotation (e.g., `: String`)
}

func (ls *LetStatement) statementNode() {}
func (ls *LetStatement) TokenLiteral() string { return ls.Token.Literal }

// String returns a string representation of the LetStatement.
// Format: "let <name>: <type> = <value>;" or "let <name> = <value>;"
func (ls *LetStatement) String() string {
	var out bytes.Buffer

	out.WriteString(ls.TokenLiteral() + " ")
	out.WriteString(ls.Name.String())

	if ls.Type != nil {
		out.WriteString(": " + ls.Type.String())
	}

	out.WriteString(" = ")

	if ls.Value != nil {
		out.WriteString(ls.Value.String())
	}

	out.WriteString(";")

	return out.String()
}

// TypeAnnotation represents a type annotation like `: String` in Vibe.
// Type annotations provide static type information to variables, parameters,
// and return values. They are a key feature that distinguishes Vibe from Ruby.
type TypeAnnotation struct {
	Token            lexer.Token     // the COLON token
	Name             string          // The name of the type (e.g., "String", "Int")
	IsCompoundType   bool            // Whether this is a compound type like [int, string]
	CompoundTypes    []string        // The list of types in a compound type
}

func (ta *TypeAnnotation) expressionNode() {}
func (ta *TypeAnnotation) TokenLiteral() string { return ta.Token.Literal }
func (ta *TypeAnnotation) String() string {
	if ta.IsCompoundType {
		return "[" + strings.Join(ta.CompoundTypes, ", ") + "]" + strings.TrimPrefix(ta.Name, "[...]")
	}
	return ta.Name
}

// ReturnStatement represents a return statement in a function.
// Example in Vibe code: `return x;` or `return x + y;`
//
// A ReturnStatement consists of:
// - The 'return' token
// - The expression whose value is to be returned
type ReturnStatement struct {
	Token lexer.Token // the RETURN token
	Value Expression  // The expression to return
}

func (rs *ReturnStatement) statementNode() {}
func (rs *ReturnStatement) TokenLiteral() string { return rs.Token.Literal }

// String returns a string representation of the ReturnStatement.
// Format: "return <value>;"
func (rs *ReturnStatement) String() string {
	var out bytes.Buffer

	out.WriteString(rs.TokenLiteral() + " ")

	if rs.Value != nil {
		out.WriteString(rs.Value.String())
	}

	out.WriteString(";")

	return out.String()
}

// ExpressionStatement represents a statement that consists of just an expression.
// Example in Vibe code: `x + 5;` or `foo();`
//
// This node type allows expressions to be used as statements, which is
// common in languages like Vibe/Ruby where function calls can stand alone.
type ExpressionStatement struct {
	Token      lexer.Token // the first token of the expression
	Expression Expression   // The expression
}

func (es *ExpressionStatement) statementNode() {}
func (es *ExpressionStatement) TokenLiteral() string { return es.Token.Literal }

// String returns a string representation of the ExpressionStatement.
// Format: "<expression>"
func (es *ExpressionStatement) String() string {
	if es.Expression != nil {
		return es.Expression.String()
	}
	return ""
}

// BlockStatement represents a block of statements enclosed in curly braces.
// Example in Vibe code: `{ let x = 5; return x; }`
//
// BlockStatements are used in function bodies, if statements, and
// other constructs that group multiple statements together.
type BlockStatement struct {
	Token      lexer.Token // the { token
	Statements []Statement // The statements within the block
}

func (bs *BlockStatement) statementNode() {}
func (bs *BlockStatement) TokenLiteral() string { return bs.Token.Literal }

// String returns a string representation of the BlockStatement.
// Format: "<statement1><statement2>..."
func (bs *BlockStatement) String() string {
	var out bytes.Buffer

	for _, s := range bs.Statements {
		out.WriteString(s.String())
	}

	return out.String()
}

// Identifier represents an identifier (variable name).
// Example in Vibe code: `x` or `myFunction`
//
// Identifiers can be used as expressions (when their value is needed)
// or as part of other constructs (like variable declarations).
type Identifier struct {
	Token lexer.Token // the IDENT token
	Value string      // The name of the identifier
}

func (i *Identifier) expressionNode() {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }
func (i *Identifier) String() string { return i.Value }

// IntegerLiteral represents an integer literal.
// Example in Vibe code: `5` or `42`
type IntegerLiteral struct {
	Token lexer.Token // the INT token
	Value int64       // The numeric value
}

func (il *IntegerLiteral) expressionNode() {}
func (il *IntegerLiteral) TokenLiteral() string { return il.Token.Literal }
func (il *IntegerLiteral) String() string { return il.Token.Literal }

// FloatLiteral represents a floating point literal.
// Example in Vibe code: `3.14` or `2.5`
type FloatLiteral struct {
	Token lexer.Token // the FLOAT token
	Value float64     // The numeric value
}

func (fl *FloatLiteral) expressionNode() {}
func (fl *FloatLiteral) TokenLiteral() string { return fl.Token.Literal }
func (fl *FloatLiteral) String() string { return fl.Token.Literal }

// StringLiteral represents a string literal.
// Example in Vibe code: `"hello"` or `'world'`
type StringLiteral struct {
	Token lexer.Token // the STRING token
	Value string      // The string value
}

func (sl *StringLiteral) expressionNode() {}
func (sl *StringLiteral) TokenLiteral() string { return sl.Token.Literal }
func (sl *StringLiteral) String() string { return "\"" + sl.Value + "\"" }

// BooleanLiteral represents a boolean literal.
// Example in Vibe code: `true` or `false`
type BooleanLiteral struct {
	Token lexer.Token // the TRUE or FALSE token
	Value bool        // The boolean value
}

func (bl *BooleanLiteral) expressionNode() {}
func (bl *BooleanLiteral) TokenLiteral() string { return bl.Token.Literal }
func (bl *BooleanLiteral) String() string { return bl.Token.Literal }

// NilLiteral represents a nil value.
// Example in Vibe code: `nil`
//
// Nil is similar to null in other languages and represents the absence of a value.
type NilLiteral struct {
	Token lexer.Token // the NIL token
}

func (nl *NilLiteral) expressionNode() {}
func (nl *NilLiteral) TokenLiteral() string { return nl.Token.Literal }
func (nl *NilLiteral) String() string { return "nil" }

// FunctionLiteral represents a function definition.
// Example in Vibe code: `func add(x: Int, y: Int): Int { return x + y; }`
//
// Functions in Vibe are first-class values and can be assigned to variables,
// passed as arguments, and returned from other functions.
type FunctionLiteral struct {
	Token      lexer.Token       // The 'func' token
	Name       *Identifier       // Function name (optional for anonymous functions)
	Parameters []*Identifier     // Parameter list
	ParamTypes []*TypeAnnotation // Parameter types (optional)
	ReturnType *TypeAnnotation   // Return type (optional)
	Body       *BlockStatement   // Function body
}

func (fl *FunctionLiteral) expressionNode() {}
func (fl *FunctionLiteral) TokenLiteral() string { return fl.Token.Literal }

// String returns a string representation of the FunctionLiteral.
// Format: "func [name](<params>) [: <return_type>] { <body> }"
func (fl *FunctionLiteral) String() string {
	var out bytes.Buffer

	params := []string{}
	for i, p := range fl.Parameters {
		paramStr := p.String()
		// Add type annotation if available
		if i < len(fl.ParamTypes) && fl.ParamTypes[i] != nil {
			paramStr += ": " + fl.ParamTypes[i].String()
		}
		params = append(params, paramStr)
	}

	out.WriteString(fl.TokenLiteral())
	if fl.Name != nil {
		out.WriteString(" " + fl.Name.String())
	}
	out.WriteString("(")
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(")")

	if fl.ReturnType != nil {
		out.WriteString(": " + fl.ReturnType.String())
	}

	out.WriteString(" { ")
	out.WriteString(fl.Body.String())
	out.WriteString(" }")

	return out.String()
}

// ClassLiteral represents a class definition.
// Example in Vibe code: `class Person { prop name: String; func greet() { ... } }`
//
// Classes in Vibe are similar to Ruby classes but include type annotations
// for properties and methods, providing better type safety.
type ClassLiteral struct {
	Token    lexer.Token     // the 'class' token
	Name     *Identifier     // The class name
	Body     *BlockStatement // Class body containing properties and methods
}

func (cl *ClassLiteral) expressionNode() {}
func (cl *ClassLiteral) TokenLiteral() string { return cl.Token.Literal }

// String returns a string representation of the ClassLiteral.
// Format: "class <name> { <body> }"
func (cl *ClassLiteral) String() string {
	var out bytes.Buffer

	out.WriteString(cl.TokenLiteral() + " ")
	out.WriteString(cl.Name.String())
	out.WriteString(" { ")
	out.WriteString(cl.Body.String())
	out.WriteString(" }")

	return out.String()
}

// PropertyStatement represents a property declaration in a class.
// Example in Vibe code: `prop name: String` inside a class definition
//
// Properties define the data that a class instance (object) can hold,
// and type annotations provide compile-time type checking.
type PropertyStatement struct {
	Token lexer.Token     // the 'prop' token
	Name  *Identifier     // The property name
	Type  *TypeAnnotation // The property type (optional but recommended)
}

func (ps *PropertyStatement) statementNode() {}
func (ps *PropertyStatement) TokenLiteral() string { return ps.Token.Literal }

// String returns a string representation of the PropertyStatement.
// Format: "prop <name>: <type>"
func (ps *PropertyStatement) String() string {
	var out bytes.Buffer

	out.WriteString(ps.TokenLiteral() + " ")
	out.WriteString(ps.Name.String())

	if ps.Type != nil {
		out.WriteString(": " + ps.Type.String())
	}

	return out.String()
}

// CallExpression represents a function call.
// Example in Vibe code: `add(1, 2)` or `person.greet()`
//
// Function calls consist of:
// - The function being called (an expression that evaluates to a function)
// - The arguments passed to the function (a list of expressions)
type CallExpression struct {
	Token     lexer.Token  // The '(' token
	Function  Expression   // Identifier or FunctionLiteral
	Arguments []Expression // The arguments passed to the function
}

func (ce *CallExpression) expressionNode() {}
func (ce *CallExpression) TokenLiteral() string { return ce.Token.Literal }

// String returns a string representation of the CallExpression.
// Format: "<function>(<arg1>, <arg2>, ...)"
func (ce *CallExpression) String() string {
	var out bytes.Buffer

	args := []string{}
	for _, a := range ce.Arguments {
		args = append(args, a.String())
	}

	out.WriteString(ce.Function.String())
	out.WriteString("(")
	out.WriteString(strings.Join(args, ", "))
	out.WriteString(")")

	return out.String()
}

// IfExpression represents an if expression.
// Example in Vibe code: `if x > 5 { return true; } else { return false; }`
//
// If expressions in Vibe are similar to other languages but can be used
// as expressions that produce values, not just as control flow statements.
type IfExpression struct {
	Token       lexer.Token     // The 'if' token
	Condition   Expression      // The condition to evaluate
	Consequence *BlockStatement // The block to execute if condition is true
	Alternative *BlockStatement // The block to execute if condition is false (optional)
}

func (ie *IfExpression) expressionNode() {}
func (ie *IfExpression) TokenLiteral() string { return ie.Token.Literal }

// String returns a string representation of the IfExpression.
// Format: "if <condition> { <consequence> } [else { <alternative> }]"
func (ie *IfExpression) String() string {
	var out bytes.Buffer

	out.WriteString("if ")
	out.WriteString(ie.Condition.String())
	out.WriteString(" { ")
	out.WriteString(ie.Consequence.String())
	out.WriteString(" }")

	if ie.Alternative != nil {
		out.WriteString(" else { ")
		out.WriteString(ie.Alternative.String())
		out.WriteString(" }")
	}

	return out.String()
}

// AssignmentExpression represents an assignment expression.
// Example: x = 5
type AssignmentExpression struct {
	Token          lexer.Token // the = token
	Name           *Identifier
	TypeAnnotation Expression  // Optional type annotation (can be nil)
	Value          Expression
}

func (ae *AssignmentExpression) expressionNode()      {}
func (ae *AssignmentExpression) TokenLiteral() string { return ae.Token.Literal }
func (ae *AssignmentExpression) String() string {
	var out bytes.Buffer

	out.WriteString(ae.Name.String())

	if ae.TypeAnnotation != nil {
		out.WriteString(": ")
		out.WriteString(ae.TypeAnnotation.String())
	}

	out.WriteString(" = ")

	if ae.Value != nil {
		out.WriteString(ae.Value.String())
	}

	return out.String()
}

// TypedIdentifier represents an identifier with a type annotation.
// Example: x: int
type TypedIdentifier struct {
	Token      lexer.Token // the identifier token
	Identifier *Identifier
	Type       *TypeAnnotation
}

func (ti *TypedIdentifier) expressionNode()      {}
func (ti *TypedIdentifier) TokenLiteral() string { return ti.Token.Literal }
func (ti *TypedIdentifier) String() string {
	var out bytes.Buffer
	out.WriteString(ti.Identifier.String())
	out.WriteString(": ")
	out.WriteString(ti.Type.String())
	return out.String()
}

// ArrayTypeAnnotation represents an array type annotation.
// Example: int[] or string[][]
type ArrayTypeAnnotation struct {
	Token    lexer.Token  // the base type token
	BaseType Expression   // the base type (can be an identifier or another type annotation)
}

func (ata *ArrayTypeAnnotation) expressionNode()      {}
func (ata *ArrayTypeAnnotation) TokenLiteral() string { return ata.Token.Literal }
func (ata *ArrayTypeAnnotation) String() string {
	var out bytes.Buffer
	out.WriteString(ata.BaseType.String())
	out.WriteString("[]")
	return out.String()
}

// CompoundTypeAnnotation represents a compound type annotation.
// Example: [int, string] or [int, string][]
type CompoundTypeAnnotation struct {
	Token lexer.Token  // the '[' token
	Types []Expression // the types in the compound type
}

func (cta *CompoundTypeAnnotation) expressionNode()      {}
func (cta *CompoundTypeAnnotation) TokenLiteral() string { return cta.Token.Literal }
func (cta *CompoundTypeAnnotation) String() string {
	var out bytes.Buffer

	types := []string{}
	for _, t := range cta.Types {
		types = append(types, t.String())
	}

	out.WriteString("[")
	out.WriteString(strings.Join(types, ", "))
	out.WriteString("]")

	return out.String()
}

// ArrayLiteral represents an array literal expression in Vibe.
// Example in Vibe code: `[1, 2, 3]` or `["hello", "world"]`
//
// An ArrayLiteral consists of:
// - The '[' token
// - A slice of expressions representing the elements
type ArrayLiteral struct {
	Token    lexer.Token  // the [ token
	Elements []Expression // The array elements
}

func (al *ArrayLiteral) expressionNode() {}
func (al *ArrayLiteral) TokenLiteral() string { return al.Token.Literal }
func (al *ArrayLiteral) String() string {
	var out bytes.Buffer

	elements := []string{}
	for _, el := range al.Elements {
		elements = append(elements, el.String())
	}

	out.WriteString("[")
	out.WriteString(strings.Join(elements, ", "))
	out.WriteString("]")

	return out.String()
}

// StructStatement represents a struct definition in Vibe.
// Structs are user-defined types that contain a collection of fields.
type StructStatement struct {
	Token      lexer.Token       // The 'struct' token
	Name       *Identifier       // Name of the struct
	Fields     []Statement         // Fields defined in the struct
}

func (ss *StructStatement) statementNode() {}
func (ss *StructStatement) TokenLiteral() string { return ss.Token.Literal }
func (ss *StructStatement) String() string {
	var out bytes.Buffer

	out.WriteString("struct ")
	out.WriteString(ss.Name.String())
	out.WriteString("\n")

	for _, field := range ss.Fields {
		out.WriteString(field.String())
		out.WriteString("\n")
	}

	out.WriteString("end")

	return out.String()
}

// StructLiteral represents a struct instantiation expression.
// It creates a new instance of a struct.
type StructLiteral struct {
	Token      lexer.Token                   // The struct type name token
	Type       string                          // Name of the struct type
	Fields     map[string]Expression           // Named field values
}

func (sl *StructLiteral) expressionNode() {}
func (sl *StructLiteral) TokenLiteral() string { return sl.Token.Literal }
func (sl *StructLiteral) String() string {
	var out bytes.Buffer

	out.WriteString(sl.Type)
	out.WriteString("(")

	pairs := []string{}
	for key, value := range sl.Fields {
		pairs = append(pairs, key + ": " + value.String())
	}
	out.WriteString(strings.Join(pairs, ", "))

	out.WriteString(")")

	return out.String()
}

// CompoundLiteral represents a compound literal (heterogeneous tuple) expression.
// Example: [1, "hello", true] with type [int, string, boolean]
type CompoundLiteral struct {
	Token    lexer.Token // the '[' token
	Elements []Expression
}

func (cl *CompoundLiteral) expressionNode()      {}
func (cl *CompoundLiteral) TokenLiteral() string { return cl.Token.Literal }
func (cl *CompoundLiteral) String() string {
	var out bytes.Buffer

	elements := []string{}
	for _, el := range cl.Elements {
		elements = append(elements, el.String())
	}

	out.WriteString("[")
	out.WriteString(strings.Join(elements, ", "))
	out.WriteString("]")

	return out.String()
}