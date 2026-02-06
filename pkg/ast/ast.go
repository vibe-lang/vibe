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

// TypeAnnotation represents a type annotation like `: String` in Vibe.
// Type annotations provide static type information to variables, parameters,
// and return values. They are a key feature that distinguishes Vibe from Ruby.
type TypeAnnotation struct {
	Token          lexer.Token // the COLON token
	Name           string      // The name of the type (e.g., "String", "Int")
	IsCompoundType bool        // Whether this is a compound type like [int, string]
	CompoundTypes  []string    // The list of types in a compound type
}

func (ta *TypeAnnotation) expressionNode()      {}
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

func (rs *ReturnStatement) statementNode()       {}
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
	Expression Expression  // The expression
}

func (es *ExpressionStatement) statementNode()       {}
func (es *ExpressionStatement) TokenLiteral() string { return es.Token.Literal }

// String returns a string representation of the ExpressionStatement.
// Format: "<expression>"
func (es *ExpressionStatement) String() string {
	if es.Expression != nil {
		return es.Expression.String()
	}
	return ""
}

// BlockStatement represents a block of statements.
// A block is a sequence of statements enclosed in a structure like if/else, function body, etc.
// Example in Vibe code for a function body:
//
//	def example()
//	    x = 5
//	    return x
//	end
type BlockStatement struct {
	Token      lexer.Token // The token that starts the block (e.g., '{', 'do', etc.)
	Statements []Statement // The statements in the block
}

func (bs *BlockStatement) statementNode()       {}
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

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }
func (i *Identifier) String() string       { return i.Value }

// IntegerLiteral represents an integer literal.
// Example in Vibe code: `5` or `42`
type IntegerLiteral struct {
	Token lexer.Token // the INT token
	Value int64       // The numeric value
}

func (il *IntegerLiteral) expressionNode()      {}
func (il *IntegerLiteral) TokenLiteral() string { return il.Token.Literal }
func (il *IntegerLiteral) String() string       { return il.Token.Literal }

// FloatLiteral represents a floating point literal.
// Example in Vibe code: `3.14` or `2.5`
type FloatLiteral struct {
	Token lexer.Token // the FLOAT token
	Value float64     // The numeric value
}

func (fl *FloatLiteral) expressionNode()      {}
func (fl *FloatLiteral) TokenLiteral() string { return fl.Token.Literal }
func (fl *FloatLiteral) String() string       { return fl.Token.Literal }

// StringLiteral represents a string literal.
// Example in Vibe code: `"hello"` or `'world'`
type StringLiteral struct {
	Token lexer.Token // the STRING token
	Value string      // The string value
}

func (sl *StringLiteral) expressionNode()      {}
func (sl *StringLiteral) TokenLiteral() string { return sl.Token.Literal }
func (sl *StringLiteral) String() string       { return "\"" + sl.Value + "\"" }

// BooleanLiteral represents a boolean literal.
// Example in Vibe code: `true` or `false`
type BooleanLiteral struct {
	Token lexer.Token // the TRUE or FALSE token
	Value bool        // The boolean value
}

func (bl *BooleanLiteral) expressionNode()      {}
func (bl *BooleanLiteral) TokenLiteral() string { return bl.Token.Literal }
func (bl *BooleanLiteral) String() string       { return bl.Token.Literal }

// NilLiteral represents a nil value.
// Example in Vibe code: `nil`
//
// Nil is similar to null in other languages and represents the absence of a value.
type NilLiteral struct {
	Token lexer.Token // the NIL token
}

func (nl *NilLiteral) expressionNode()      {}
func (nl *NilLiteral) TokenLiteral() string { return nl.Token.Literal }
func (nl *NilLiteral) String() string       { return "nil" }

// FunctionLiteral represents a function definition.
// Example in Vibe code:
//
//	def add(x: int, y: int): int
//	    x + y
//	end
//
//	def identity<T>(x: T): T
//	    x
//	end
//
// Functions in Vibe can be named or anonymous, can have
// typed or untyped parameters, and can specify a return type.
// They can also have type parameters for generics.
type FunctionLiteral struct {
	Token         lexer.Token     // The 'fn' or 'def' token
	Name          *Identifier     // Optional function name (can be nil for anonymous functions)
	TypeParams    []string        // Generic type parameters (e.g., ["T", "U"])
	Parameters    []*Identifier   // Parameters (can be empty)
	ParamTypes    []Expression    // Parameter type annotations (can be nil for untyped parameters)
	ParamDefaults []Expression    // Default values for parameters (can be nil for required parameters)
	ReturnType    Expression      // Return type annotation (can be nil)
	Variadic      bool            // Whether the last parameter is variadic (*args)
	Body          *BlockStatement // Function body
}

func (fl *FunctionLiteral) expressionNode()      {}
func (fl *FunctionLiteral) TokenLiteral() string { return fl.Token.Literal }

// String returns a string representation of the function.
// Format: "def [name][<T, U>](<params>) [: <return_type>]
//
//	    <body>
//	end"
func (fl *FunctionLiteral) String() string {
	var out bytes.Buffer

	if fl.Token.Literal == "def" {
		out.WriteString("def ")
	} else {
		out.WriteString("fn ")
	}

	if fl.Name != nil {
		out.WriteString(fl.Name.Value)
	}

	if len(fl.TypeParams) > 0 {
		out.WriteString("<")
		out.WriteString(strings.Join(fl.TypeParams, ", "))
		out.WriteString(">")
	}

	params := []string{}
	for i, p := range fl.Parameters {
		if i < len(fl.ParamTypes) && fl.ParamTypes[i] != nil {
			params = append(params, p.String()+": "+fl.ParamTypes[i].String())
		} else {
			params = append(params, p.String())
		}
	}

	out.WriteString("(")
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(")")

	if fl.ReturnType != nil {
		out.WriteString(": ")
		out.WriteString(fl.ReturnType.String())
	}

	out.WriteString("\n")
	out.WriteString(fl.Body.String())
	out.WriteString("\nend")

	return out.String()
}

// ClassLiteral represents a class definition.
// Example in Vibe code:
//
//	class Person { prop name: String; func greet() { ... } }
//	class Box<T>
//	  def initialize(value: T)
//	    self.value = value
//	  end
//	end
//
// Classes in Vibe are similar to Ruby classes but include type annotations
// for properties and methods, providing better type safety.
// They can also have type parameters for generics.
type ClassLiteral struct {
	Token      lexer.Token     // the 'class' token
	Name       *Identifier     // The class name
	TypeParams []string        // Generic type parameters (e.g., ["T", "U"])
	Parent     *Identifier     // Optional parent class (for inheritance with <)
	Body       *BlockStatement // Class body containing properties and methods
}

func (cl *ClassLiteral) expressionNode()      {}
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

func (ps *PropertyStatement) statementNode()       {}
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
// Example in Vibe code: `add(1, 2)` or `identity<int>(42)`
//
// Function calls consist of:
// - The function being called (an expression that evaluates to a function)
// - Optional type arguments for generic functions (e.g., <int, string>)
// - The arguments passed to the function (a list of expressions)
type CallExpression struct {
	Token     lexer.Token  // The '(' token
	Function  Expression   // Identifier or FunctionLiteral
	TypeArgs  []string     // Explicit type arguments for generic calls (e.g., ["int", "string"])
	Arguments []Expression // The arguments passed to the function
}

func (ce *CallExpression) expressionNode()      {}
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
// Example in Vibe code: `if x > 5
//
//	return true
//
// else
//
//	return false
//
// end`
//
// If expressions in Vibe are similar to Ruby and can be used
// as expressions that produce values, not just as control flow statements.
type IfExpression struct {
	Token       lexer.Token     // The 'if' token
	Condition   Expression      // The condition to evaluate
	Consequence *BlockStatement // The block to execute if condition is true
	Alternative *BlockStatement // The block to execute if condition is false (optional)
}

func (ie *IfExpression) expressionNode()      {}
func (ie *IfExpression) TokenLiteral() string { return ie.Token.Literal }

// String returns a string representation of the IfExpression.
// Format: "if <condition> <consequence> [else <alternative>] end"
func (ie *IfExpression) String() string {
	var out bytes.Buffer

	out.WriteString("if ")
	out.WriteString(ie.Condition.String())
	out.WriteString("\n")
	out.WriteString(ie.Consequence.String())

	if ie.Alternative != nil {
		out.WriteString(" else\n")
		out.WriteString(ie.Alternative.String())
	}

	out.WriteString(" end")

	return out.String()
}

// AssignmentExpression represents an assignment expression.
// Example: x = 5
type AssignmentExpression struct {
	Token          lexer.Token // the = token
	Name           *Identifier
	TypeAnnotation Expression // Optional type annotation (can be nil)
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
	Token    lexer.Token // the base type token
	BaseType Expression  // the base type (can be an identifier or another type annotation)
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

func (al *ArrayLiteral) expressionNode()      {}
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
// They can also have type parameters for generics.
//
// Example:
//
//	struct Pair<A, B>
//	  first: A
//	  second: B
//	end
type StructStatement struct {
	Token      lexer.Token // The 'struct' token
	Name       *Identifier // Name of the struct
	TypeParams []string    // Generic type parameters (e.g., ["A", "B"])
	Fields     []Statement // Fields defined in the struct
}

func (ss *StructStatement) statementNode()       {}
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
// Example: Pair<int, string>(first: 1, second: "hello")
type StructLiteral struct {
	Token    lexer.Token           // The struct type name token
	Type     string                // Name of the struct type
	TypeArgs []string              // Explicit type arguments (e.g., ["int", "string"])
	Fields   map[string]Expression // Named field values
}

func (sl *StructLiteral) expressionNode()      {}
func (sl *StructLiteral) TokenLiteral() string { return sl.Token.Literal }
func (sl *StructLiteral) String() string {
	var out bytes.Buffer

	out.WriteString(sl.Type)
	out.WriteString("(")

	pairs := []string{}
	for key, value := range sl.Fields {
		pairs = append(pairs, key+": "+value.String())
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

// RangeExpression represents a range of values (either inclusive or exclusive).
// Example: 1..5 (inclusive) or 1...5 (exclusive)
type RangeExpression struct {
	Token     lexer.Token // The token for the range operator (.., ...)
	Start     Expression  // The start value of the range
	End       Expression  // The end value of the range
	Exclusive bool        // Whether the range excludes the end value
}

func (re *RangeExpression) expressionNode()      {}
func (re *RangeExpression) TokenLiteral() string { return re.Token.Literal }
func (re *RangeExpression) String() string {
	var out bytes.Buffer
	out.WriteString(re.Start.String())
	if re.Exclusive {
		out.WriteString("...")
	} else {
		out.WriteString("..")
	}
	out.WriteString(re.End.String())
	return out.String()
}

// RangeCallExpression represents a range created using the Range function.
// Example: Range(1, 10)
type RangeCallExpression struct {
	Token     lexer.Token // The 'Range' token
	Start     Expression  // The start value
	End       Expression  // The end value
	Exclusive bool        // Whether the range excludes the end value (optional 3rd argument)
}

func (rc *RangeCallExpression) expressionNode()      {}
func (rc *RangeCallExpression) TokenLiteral() string { return rc.Token.Literal }
func (rc *RangeCallExpression) String() string {
	var out bytes.Buffer
	out.WriteString("Range(")
	out.WriteString(rc.Start.String())
	out.WriteString(", ")
	out.WriteString(rc.End.String())
	if rc.Exclusive {
		out.WriteString(", true")
	}
	out.WriteString(")")
	return out.String()
}

// StringInterpolationLiteral represents a string that contains interpolated expressions.
// Example: "Hello, ${name}!"
//
// String interpolation in Vibe allows embedding expressions within strings.
// The expressions are evaluated at runtime and their values are inserted into the string.
type StringInterpolationLiteral struct {
	Token       lexer.Token  // The string token
	Value       string       // The raw string value including ${...} placeholders
	Parts       []string     // The string parts (before, between, and after expressions)
	Expressions []Expression // The parsed expressions within ${...} placeholders
}

func (sil *StringInterpolationLiteral) expressionNode()      {}
func (sil *StringInterpolationLiteral) TokenLiteral() string { return sil.Token.Literal }
func (sil *StringInterpolationLiteral) String() string {
	return "\"" + sil.Value + "\""
}

// ForLoop represents a for loop statement in Vibe.
// It iterates over a collection (array, range, etc.) and executes the body for each element.
//
// Example Vibe code:
//
//	for i in [1, 2, 3]
//	  puts(i)
//	end
type ForLoop struct {
	Token         lexer.Token     // The 'for' token
	Iterator      *Identifier     // The variable that holds each element (or key for hashes)
	ValueIterator *Identifier     // Optional second variable for hash value iteration (for k, v in hash)
	Collection    Expression      // The collection to iterate over (array, range, hash, etc.)
	Body          *BlockStatement // The body of the loop
}

func (fl *ForLoop) statementNode()       {}
func (fl *ForLoop) TokenLiteral() string { return fl.Token.Literal }
func (fl *ForLoop) String() string {
	var out bytes.Buffer

	out.WriteString("for ")
	out.WriteString(fl.Iterator.String())
	out.WriteString(" in ")
	out.WriteString(fl.Collection.String())
	out.WriteString(" ")
	out.WriteString(fl.Body.String())

	return out.String()
}

// DotExpression represents a dot expression for accessing struct fields.
// Example: person.name
type DotExpression struct {
	Token lexer.Token // The '.' token
	Left  Expression  // The expression before the dot (e.g., person)
	Field *Identifier // The field name after the dot (e.g., name)
}

func (de *DotExpression) expressionNode()      {}
func (de *DotExpression) TokenLiteral() string { return de.Token.Literal }
func (de *DotExpression) String() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(de.Left.String())
	out.WriteString(".")
	out.WriteString(de.Field.String())
	out.WriteString(")")

	return out.String()
}

// IndexExpression represents an index expression for accessing array elements.
// Example: arr[0]
type IndexExpression struct {
	Token lexer.Token // The '[' token
	Left  Expression  // The expression before the brackets (e.g., arr)
	Index Expression  // The index expression inside the brackets (e.g., 0)
}

func (ie *IndexExpression) expressionNode()      {}
func (ie *IndexExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IndexExpression) String() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(ie.Left.String())
	out.WriteString("[")
	out.WriteString(ie.Index.String())
	out.WriteString("])")

	return out.String()
}

// PrefixExpression represents a prefix expression: <operator> <right>
// Examples: -5, !true
type PrefixExpression struct {
	Token    lexer.Token // The prefix token, e.g. !
	Operator string      // The operator string, e.g. - or !
	Right    Expression  // The operand expression
}

func (pe *PrefixExpression) expressionNode()      {}
func (pe *PrefixExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *PrefixExpression) String() string {
	var out bytes.Buffer
	out.WriteString("(")
	out.WriteString(pe.Operator)
	out.WriteString(pe.Right.String())
	out.WriteString(")")
	return out.String()
}

// InfixExpression represents an infix expression: <left> <operator> <right>
// Examples: a + b, a == b, a < b
type InfixExpression struct {
	Token    lexer.Token // The operator token, e.g. +
	Left     Expression  // The left-hand side expression
	Operator string      // The operator string, e.g. +
	Right    Expression  // The right-hand side expression
}

func (ie *InfixExpression) expressionNode()      {}
func (ie *InfixExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *InfixExpression) String() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(ie.Left.String())
	out.WriteString(" " + ie.Operator + " ")
	out.WriteString(ie.Right.String())
	out.WriteString(")")

	return out.String()
}

// WhileLoop represents a while loop: while <condition> ... end
type WhileLoop struct {
	Token     lexer.Token
	Condition Expression
	Body      *BlockStatement
}

func (wl *WhileLoop) statementNode()       {}
func (wl *WhileLoop) TokenLiteral() string { return wl.Token.Literal }
func (wl *WhileLoop) String() string {
	var out bytes.Buffer
	out.WriteString("while ")
	out.WriteString(wl.Condition.String())
	out.WriteString(" ")
	out.WriteString(wl.Body.String())
	return out.String()
}

// BreakStatement represents a break statement inside a loop.
type BreakStatement struct {
	Token lexer.Token
}

func (bs *BreakStatement) statementNode()       {}
func (bs *BreakStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BreakStatement) String() string       { return "break" }

// ContinueStatement represents a continue statement inside a loop.
type ContinueStatement struct {
	Token lexer.Token
}

func (cs *ContinueStatement) statementNode()       {}
func (cs *ContinueStatement) TokenLiteral() string { return cs.Token.Literal }
func (cs *ContinueStatement) String() string       { return "continue" }

// HashLiteral represents a hash map literal: {key: value, ...}
type HashLiteral struct {
	Token lexer.Token
	Pairs map[Expression]Expression
}

func (hl *HashLiteral) expressionNode()      {}
func (hl *HashLiteral) TokenLiteral() string { return hl.Token.Literal }
func (hl *HashLiteral) String() string {
	var out bytes.Buffer
	pairs := []string{}
	for key, value := range hl.Pairs {
		pairs = append(pairs, key.String()+": "+value.String())
	}
	out.WriteString("{")
	out.WriteString(strings.Join(pairs, ", "))
	out.WriteString("}")
	return out.String()
}

// ImportStatement represents: import "path/to/file.vb"
type ImportStatement struct {
	Token lexer.Token
	Path  string
}

func (is *ImportStatement) statementNode()       {}
func (is *ImportStatement) TokenLiteral() string { return is.Token.Literal }
func (is *ImportStatement) String() string       { return "import \"" + is.Path + "\"" }

// TryExpression represents: try ... catch e ... finally ... end
type TryExpression struct {
	Token       lexer.Token
	Body        *BlockStatement
	CatchVar    *Identifier
	CatchBody   *BlockStatement
	FinallyBody *BlockStatement // optional finally block
}

func (te *TryExpression) expressionNode()      {}
func (te *TryExpression) TokenLiteral() string { return te.Token.Literal }
func (te *TryExpression) String() string {
	var out bytes.Buffer
	out.WriteString("try ")
	out.WriteString(te.Body.String())
	out.WriteString(" catch ")
	if te.CatchVar != nil {
		out.WriteString(te.CatchVar.String())
		out.WriteString(" ")
	}
	out.WriteString(te.CatchBody.String())
	return out.String()
}

// ThrowStatement represents: throw <expression>
type ThrowStatement struct {
	Token lexer.Token
	Value Expression
}

func (ts *ThrowStatement) statementNode()       {}
func (ts *ThrowStatement) TokenLiteral() string { return ts.Token.Literal }
func (ts *ThrowStatement) String() string       { return "throw " + ts.Value.String() }

// IndexAssignment represents: arr[i] = value
type IndexAssignment struct {
	Token lexer.Token
	Left  Expression
	Index Expression
	Value Expression
}

func (ia *IndexAssignment) expressionNode()      {}
func (ia *IndexAssignment) TokenLiteral() string { return ia.Token.Literal }
func (ia *IndexAssignment) String() string {
	return ia.Left.String() + "[" + ia.Index.String() + "] = " + ia.Value.String()
}

// DotAssignment represents: obj.field = value
type DotAssignment struct {
	Token lexer.Token
	Left  Expression
	Field string
	Value Expression
}

// TernaryExpression represents: condition ? consequence : alternative
type TernaryExpression struct {
	Token       lexer.Token // The '?' token
	Condition   Expression
	Consequence Expression
	Alternative Expression
}

func (te *TernaryExpression) expressionNode()      {}
func (te *TernaryExpression) TokenLiteral() string { return te.Token.Literal }
func (te *TernaryExpression) String() string {
	return "(" + te.Condition.String() + " ? " + te.Consequence.String() + " : " + te.Alternative.String() + ")"
}

// UnlessExpression represents: unless condition ... end
type UnlessExpression struct {
	Token       lexer.Token
	Condition   Expression
	Consequence *BlockStatement
	Alternative *BlockStatement // optional else
}

func (ue *UnlessExpression) expressionNode()      {}
func (ue *UnlessExpression) TokenLiteral() string { return ue.Token.Literal }
func (ue *UnlessExpression) String() string {
	var out bytes.Buffer
	out.WriteString("unless ")
	out.WriteString(ue.Condition.String())
	out.WriteString("\n")
	out.WriteString(ue.Consequence.String())
	if ue.Alternative != nil {
		out.WriteString(" else\n")
		out.WriteString(ue.Alternative.String())
	}
	out.WriteString(" end")
	return out.String()
}

// UntilLoop represents: until condition ... end
type UntilLoop struct {
	Token     lexer.Token
	Condition Expression
	Body      *BlockStatement
}

func (ul *UntilLoop) statementNode()       {}
func (ul *UntilLoop) TokenLiteral() string { return ul.Token.Literal }
func (ul *UntilLoop) String() string {
	var out bytes.Buffer
	out.WriteString("until ")
	out.WriteString(ul.Condition.String())
	out.WriteString(" ")
	out.WriteString(ul.Body.String())
	return out.String()
}

// CaseExpression represents: case value when ... else ... end
type CaseExpression struct {
	Token   lexer.Token
	Subject Expression      // The value being matched
	Whens   []*WhenClause   // List of when clauses
	Default *BlockStatement // Optional else block
}

func (ce *CaseExpression) expressionNode()      {}
func (ce *CaseExpression) TokenLiteral() string { return ce.Token.Literal }
func (ce *CaseExpression) String() string {
	var out bytes.Buffer
	out.WriteString("case ")
	out.WriteString(ce.Subject.String())
	for _, w := range ce.Whens {
		out.WriteString("\n")
		out.WriteString(w.String())
	}
	if ce.Default != nil {
		out.WriteString("\nelse\n")
		out.WriteString(ce.Default.String())
	}
	out.WriteString("\nend")
	return out.String()
}

// WhenClause represents a single when clause in a case expression
type WhenClause struct {
	Token  lexer.Token
	Values []Expression // One or more match values
	Body   *BlockStatement
}

func (wc *WhenClause) expressionNode()      {}
func (wc *WhenClause) TokenLiteral() string { return wc.Token.Literal }
func (wc *WhenClause) String() string {
	var out bytes.Buffer
	out.WriteString("when ")
	vals := []string{}
	for _, v := range wc.Values {
		vals = append(vals, v.String())
	}
	out.WriteString(strings.Join(vals, ", "))
	out.WriteString("\n")
	out.WriteString(wc.Body.String())
	return out.String()
}

// ArrowFunction represents: -> (params) { body }
type ArrowFunction struct {
	Token      lexer.Token
	Parameters []*Identifier
	Body       *BlockStatement
}

func (af *ArrowFunction) expressionNode()      {}
func (af *ArrowFunction) TokenLiteral() string { return af.Token.Literal }
func (af *ArrowFunction) String() string {
	var out bytes.Buffer
	params := []string{}
	for _, p := range af.Parameters {
		params = append(params, p.String())
	}
	out.WriteString("-> (")
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(") { ")
	out.WriteString(af.Body.String())
	out.WriteString(" }")
	return out.String()
}

// PipeExpression represents: expr |> func or expr |> func(args)
type PipeExpression struct {
	Token lexer.Token
	Left  Expression
	Right Expression // CallExpression or Identifier
}

func (pe *PipeExpression) expressionNode()      {}
func (pe *PipeExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *PipeExpression) String() string {
	return "(" + pe.Left.String() + " |> " + pe.Right.String() + ")"
}

// NilCoalesceExpression represents: expr ?? default
type NilCoalesceExpression struct {
	Token lexer.Token
	Left  Expression
	Right Expression
}

func (nc *NilCoalesceExpression) expressionNode()      {}
func (nc *NilCoalesceExpression) TokenLiteral() string { return nc.Token.Literal }
func (nc *NilCoalesceExpression) String() string {
	return "(" + nc.Left.String() + " ?? " + nc.Right.String() + ")"
}

// DestructureAssignment represents: a, b, c = [1, 2, 3]
type DestructureAssignment struct {
	Token lexer.Token
	Names []*Identifier
	Value Expression
}

func (da *DestructureAssignment) expressionNode()      {}
func (da *DestructureAssignment) TokenLiteral() string { return da.Token.Literal }
func (da *DestructureAssignment) String() string {
	names := []string{}
	for _, n := range da.Names {
		names = append(names, n.Value)
	}
	return strings.Join(names, ", ") + " = " + da.Value.String()
}

// UnionTypeAnnotation represents a union type: string | int | nil
type UnionTypeAnnotation struct {
	Token lexer.Token
	Types []Expression // The types in the union
}

func (uta *UnionTypeAnnotation) expressionNode()      {}
func (uta *UnionTypeAnnotation) TokenLiteral() string { return uta.Token.Literal }
func (uta *UnionTypeAnnotation) String() string {
	types := []string{}
	for _, t := range uta.Types {
		types = append(types, t.String())
	}
	return strings.Join(types, " | ")
}

// EnumStatement represents an enum definition: enum Color Red Green Blue end
type EnumStatement struct {
	Token  lexer.Token
	Name   *Identifier
	Values []string
}

func (es *EnumStatement) statementNode()       {}
func (es *EnumStatement) TokenLiteral() string { return es.Token.Literal }
func (es *EnumStatement) String() string {
	var out bytes.Buffer
	out.WriteString("enum ")
	out.WriteString(es.Name.Value)
	out.WriteString("\n")
	for _, v := range es.Values {
		out.WriteString("  " + v + "\n")
	}
	out.WriteString("end")
	return out.String()
}

// InExpression represents: value in collection
type InExpression struct {
	Token lexer.Token
	Left  Expression // the value to check
	Right Expression // the collection
}

func (ie *InExpression) expressionNode()      {}
func (ie *InExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *InExpression) String() string {
	return "(" + ie.Left.String() + " in " + ie.Right.String() + ")"
}

func (da *DotAssignment) expressionNode()      {}
func (da *DotAssignment) TokenLiteral() string { return da.Token.Literal }
func (da *DotAssignment) String() string {
	return da.Left.String() + "." + da.Field + " = " + da.Value.String()
}

// PostfixCondition represents a postfix if/unless: expr if condition or expr unless condition
type PostfixCondition struct {
	Token     lexer.Token // the IF or UNLESS token
	Statement Statement   // the statement to conditionally execute
	Condition Expression  // the condition
	Unless    bool        // true if this is an unless (inverted condition)
}

func (pc *PostfixCondition) statementNode()       {}
func (pc *PostfixCondition) TokenLiteral() string { return pc.Token.Literal }
func (pc *PostfixCondition) String() string {
	keyword := "if"
	if pc.Unless {
		keyword = "unless"
	}
	return pc.Statement.String() + " " + keyword + " " + pc.Condition.String()
}

// ConstStatement represents a constant declaration: const X = value
type ConstStatement struct {
	Token          lexer.Token // the 'const' token
	Name           *Identifier
	TypeAnnotation Expression // Optional type annotation
	Value          Expression
}

func (cs *ConstStatement) statementNode()       {}
func (cs *ConstStatement) TokenLiteral() string { return cs.Token.Literal }
func (cs *ConstStatement) String() string {
	var out bytes.Buffer
	out.WriteString("const ")
	out.WriteString(cs.Name.String())
	if cs.TypeAnnotation != nil {
		out.WriteString(": ")
		out.WriteString(cs.TypeAnnotation.String())
	}
	out.WriteString(" = ")
	if cs.Value != nil {
		out.WriteString(cs.Value.String())
	}
	return out.String()
}
