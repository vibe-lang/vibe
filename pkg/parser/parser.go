package parser

import (
	"fmt"
	"strconv"

	"github.com/vibe-lang/vibe/pkg/ast"
	"github.com/vibe-lang/vibe/pkg/lexer"
)

// Parser represents a parser for the Vibe language.
// The parser is responsible for transforming tokens from the lexer into
// an Abstract Syntax Tree (AST) that represents the structure and meaning
// of the Vibe source code.
//
// This parser uses the Pratt parsing technique (also known as Top Down
// Operator Precedence parsing), which is particularly well-suited for
// expression parsing with proper operator precedence.
type Parser struct {
	l *lexer.Lexer // The lexer that provides tokens

	curToken  lexer.Token // The current token being examined
	peekToken lexer.Token // The next token (used for lookahead)

	errors []string // Errors encountered during parsing

	// Maps for handling operator precedence
	prefixParseFns map[lexer.TokenType]prefixParseFn // Functions for parsing prefix expressions
	infixParseFns  map[lexer.TokenType]infixParseFn  // Functions for parsing infix expressions
}

// Function types for parsing expressions
type (
	// prefixParseFn parses expressions where the operator comes before the operand
	// (e.g., -5, !true, func() {...})
	prefixParseFn func() ast.Expression

	// infixParseFn parses expressions where the operator comes between two operands
	// (e.g., 1 + 2, a == b, foo())
	// The argument is the left-hand expression already parsed
	infixParseFn func(ast.Expression) ast.Expression
)

// Precedence levels for operators, used to determine the order of operations.
// Higher values indicate higher precedence (tighter binding).
const (
	_ int = iota
	LOWEST       // Default lowest precedence
	ASSIGN       // =
	EQUALS       // ==, !=
	LESSGREATER  // >, <, >=, <=
	SUM          // +, -
	PRODUCT      // *, /, %
	PREFIX       // -X, !X
	CALL         // myFunction(X)
)

// Precedence map associates token types with their precedence levels.
// This is used to resolve operator precedence when parsing expressions.
var precedences = map[lexer.TokenType]int{
	lexer.ASSIGN:   ASSIGN,
	lexer.EQ:       EQUALS,
	lexer.NOT_EQ:   EQUALS,
	lexer.LT:       LESSGREATER,
	lexer.GT:       LESSGREATER,
	lexer.LTE:      LESSGREATER,
	lexer.GTE:      LESSGREATER,
	lexer.PLUS:     SUM,
	lexer.MINUS:    SUM,
	lexer.SLASH:    PRODUCT,
	lexer.ASTERISK: PRODUCT,
	lexer.MODULO:   PRODUCT,
	lexer.LPAREN:   CALL,
}

// New creates a new parser for the given lexer.
// It initializes the parser state and registers parsing functions
// for various types of expressions.
//
// This function sets up the parsing functions that will be used to parse
// different parts of the Vibe language, including literals, expressions,
// and language constructs like functions and classes.
//
// Example:
//
//	l := lexer.New("let x = 5;")
//	p := parser.New(l)
//	program := p.ParseProgram()
func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
	}

	// Initialize the parser by reading two tokens
	// so curToken and peekToken are both set
	p.nextToken()
	p.nextToken()

	// Register prefix parse functions
	p.prefixParseFns = make(map[lexer.TokenType]prefixParseFn)
	p.registerPrefix(lexer.IDENT, p.parseIdentifier)
	p.registerPrefix(lexer.INT, p.parseIntegerLiteral)
	p.registerPrefix(lexer.FLOAT, p.parseFloatLiteral)
	p.registerPrefix(lexer.STRING, p.parseStringLiteral)
	p.registerPrefix(lexer.TRUE, p.parseBooleanLiteral)
	p.registerPrefix(lexer.FALSE, p.parseBooleanLiteral)
	p.registerPrefix(lexer.NIL, p.parseNilLiteral)
	p.registerPrefix(lexer.BANG, p.parsePrefixExpression)
	p.registerPrefix(lexer.MINUS, p.parsePrefixExpression)
	p.registerPrefix(lexer.LPAREN, p.parseGroupedExpression)
	p.registerPrefix(lexer.FUNCTION, p.parseFunctionLiteral)
	p.registerPrefix(lexer.IF, p.parseIfExpression)
	p.registerPrefix(lexer.CLASS, p.parseClassLiteral)
	p.registerPrefix(lexer.LBRACKET, p.parseArrayLiteral)

	// Register infix parse functions
	p.infixParseFns = make(map[lexer.TokenType]infixParseFn)
	p.registerInfix(lexer.PLUS, p.parseInfixExpression)
	p.registerInfix(lexer.MINUS, p.parseInfixExpression)
	p.registerInfix(lexer.SLASH, p.parseInfixExpression)
	p.registerInfix(lexer.ASTERISK, p.parseInfixExpression)
	p.registerInfix(lexer.MODULO, p.parseInfixExpression)
	p.registerInfix(lexer.EQ, p.parseInfixExpression)
	p.registerInfix(lexer.NOT_EQ, p.parseInfixExpression)
	p.registerInfix(lexer.LT, p.parseInfixExpression)
	p.registerInfix(lexer.GT, p.parseInfixExpression)
	p.registerInfix(lexer.LTE, p.parseInfixExpression)
	p.registerInfix(lexer.GTE, p.parseInfixExpression)
	p.registerInfix(lexer.LPAREN, p.parseCallExpression)
	p.registerInfix(lexer.ASSIGN, p.parseAssignmentExpression)

	return p
}

// Errors returns all errors encountered during parsing.
// These errors are collected rather than immediately throwing exceptions,
// allowing the parser to continue and potentially find more errors.
func (p *Parser) Errors() []string {
	return p.errors
}

// addError adds an error with the provided message to the errors slice.
// It includes line and column information for better error localization.
func (p *Parser) addError(msg string) {
	line := p.curToken.Line
	column := p.curToken.Column
	p.errors = append(p.errors, fmt.Sprintf("[%d:%d] %s", line, column, msg))
}

// peekError adds an error when the next token is not the expected type.
// This is a common error case when parsing syntax with specific patterns.
func (p *Parser) peekError(t lexer.TokenType) {
	msg := fmt.Sprintf("expected next token to be %s, got %s instead",
		t, p.peekToken.Type)
	p.addError(msg)
}

// nextToken advances to the next token from the lexer.
// This moves the parser forward in the token stream.
func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

// ParseProgram parses a complete Vibe program.
// This is the entry point for parsing and returns the root of the AST.
//
// The program is the root node of the AST and contains all the statements
// that make up the Vibe program.
//
// Example:
//
//	program := parser.ParseProgram()
//	// program now contains the AST representation of the source code
func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}
	program.Statements = []ast.Statement{}

	for !p.curTokenIs(lexer.EOF) {
		stmt := p.parseStatement()
		program.Statements = append(program.Statements, stmt)
		p.nextToken()
	}

	return program
}

// parseStatement parses a single statement.
// It decides what type of statement to parse based on the current token.
func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case lexer.LET:
		return p.parseLetStatement()
	case lexer.RETURN:
		return p.parseReturnStatement()
	case lexer.PROP:
		return p.parsePropertyStatement()
	default:
		return p.parseExpressionStatement()
	}
}

// parseLetStatement parses a variable declaration statement.
// Format: let <identifier> [: <type>] = <expression>;
//
// Example Vibe code:
//
//	let x = 5;
//	let name: String = "John";
func (p *Parser) parseLetStatement() *ast.LetStatement {
	stmt := &ast.LetStatement{Token: p.curToken}

	if !p.expectPeek(lexer.IDENT) {
		return nil
	}

	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// Check for type annotation
	if p.peekTokenIs(lexer.COLON) {
		p.nextToken() // consume the colon
		p.nextToken() // move to the type name
		if !p.curTokenIs(lexer.IDENT) {
			p.addError("expected type name after colon")
			return nil
		}
		stmt.Type = &ast.TypeAnnotation{
			Token: p.curToken,
			Name:  p.curToken.Literal,
		}
	}

	if !p.expectPeek(lexer.ASSIGN) {
		return nil
	}

	p.nextToken()

	stmt.Value = p.parseExpression(LOWEST)

	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parseReturnStatement parses a return statement.
// Format: return <expression>;
//
// Example Vibe code:
//
//	return 42;
//	return x + y;
func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	stmt := &ast.ReturnStatement{Token: p.curToken}

	p.nextToken()

	if p.curTokenIs(lexer.SEMICOLON) {
		return stmt
	}

	stmt.Value = p.parseExpression(LOWEST)

	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parsePropertyStatement parses a property declaration in a class.
// Format: prop <identifier> [: <type>];
//
// Example Vibe code:
//
//	prop name: String;
//	prop age: Int;
func (p *Parser) parsePropertyStatement() *ast.PropertyStatement {
	stmt := &ast.PropertyStatement{Token: p.curToken}

	if !p.expectPeek(lexer.IDENT) {
		return nil
	}

	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// Check for type annotation
	if p.peekTokenIs(lexer.COLON) {
		p.nextToken() // consume the colon
		p.nextToken() // move to the type name
		if !p.curTokenIs(lexer.IDENT) {
			p.addError("expected type name after colon")
			return nil
		}
		stmt.Type = &ast.TypeAnnotation{
			Token: p.curToken,
			Name:  p.curToken.Literal,
		}
	}

	return stmt
}

// parseExpressionStatement parses a statement that consists of just an expression.
// Format: <expression>;
//
// Example Vibe code:
//
//	x + 5;
//	foo();
func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	stmt := &ast.ExpressionStatement{Token: p.curToken}

	stmt.Expression = p.parseExpression(LOWEST)

	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parseExpression parses an expression with the specified precedence.
// This is the core of the Pratt parsing algorithm, which uses precedence
// to correctly handle operator associativity and precedence.
//
// The precedence parameter determines how "greedy" the parser should be
// in consuming tokens for the current expression.
func (p *Parser) parseExpression(precedence int) ast.Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.addError(fmt.Sprintf("no prefix parse function for %s found", p.curToken.Type))
		return nil
	}
	leftExp := prefix()

	for !p.peekTokenIs(lexer.SEMICOLON) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken()

		leftExp = infix(leftExp)
	}

	return leftExp
}

// parseIdentifier parses an identifier.
// Identifiers in Vibe can be variable names, function names, etc.
//
// Example Vibe code:
//
//	x
//	myFunction
func (p *Parser) parseIdentifier() ast.Expression {
	// Check if this might be an identifier with a type annotation (for assignments)
	if p.peekTokenIs(lexer.COLON) {
		ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

		// Consume the colon
		p.nextToken()

		// Parse the type name
		if !p.peekTokenIs(lexer.IDENT) {
			p.addError(fmt.Sprintf("expected type name after ':', got %s", p.peekToken.Type))
			return ident
		}

		p.nextToken()
		typeName := p.curToken.Literal

		// Check for array type notation (e.g., "int[]")
		if p.peekTokenIs(lexer.LBRACKET) {
			p.nextToken() // consume [

			// Expect ] to complete the array type notation
			if !p.peekTokenIs(lexer.RBRACKET) {
				p.addError(fmt.Sprintf("expected ']' to complete array type, got %s", p.peekToken.Type))
				return ident
			}

			p.nextToken() // consume ]
			typeName += "[]"
		}

		typeAnnotation := &ast.TypeAnnotation{
			Token: p.curToken,
			Name:  typeName,
		}

		// If the next token is ASSIGN, we're in a typed assignment
		if p.peekTokenIs(lexer.ASSIGN) {
			// Create a temporary Identifier/TypeAnnotation pair
			// This will be converted to an AssignmentExpression later
			return &ast.TypedIdentifier{
				Identifier: ident,
				Type:       typeAnnotation,
			}
		}

		// Otherwise, this is a typed identifier (e.g., in a parameter list)
		return &ast.TypedIdentifier{
			Identifier: ident,
			Type:       typeAnnotation,
		}
	}

	// Simple identifier without type annotation
	return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

// parseIntegerLiteral parses an integer literal.
// Integer literals are sequences of digits that represent whole numbers.
//
// Example Vibe code:
//
//	42
//	100
func (p *Parser) parseIntegerLiteral() ast.Expression {
	lit := &ast.IntegerLiteral{Token: p.curToken}

	value, err := strconv.ParseInt(p.curToken.Literal, 0, 64)
	if err != nil {
		msg := fmt.Sprintf("could not parse %q as integer", p.curToken.Literal)
		p.addError(msg)
		return nil
	}

	lit.Value = value
	return lit
}

// parseFloatLiteral parses a floating-point literal.
// Float literals are numbers with decimal points.
//
// Example Vibe code:
//
//	3.14
//	2.5
func (p *Parser) parseFloatLiteral() ast.Expression {
	lit := &ast.FloatLiteral{Token: p.curToken}

	value, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		msg := fmt.Sprintf("could not parse %q as float", p.curToken.Literal)
		p.addError(msg)
		return nil
	}

	lit.Value = value
	return lit
}

// parseStringLiteral parses a string literal.
// String literals are sequences of characters enclosed in quotes.
//
// Example Vibe code:
//
//	"hello world"
//	'single quotes'
func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

// parseBooleanLiteral parses a boolean literal.
// Boolean literals are 'true' or 'false'.
//
// Example Vibe code:
//
//	true
//	false
func (p *Parser) parseBooleanLiteral() ast.Expression {
	return &ast.BooleanLiteral{
		Token: p.curToken,
		Value: p.curTokenIs(lexer.TRUE),
	}
}

// parseNilLiteral parses a nil literal.
// Nil represents the absence of a value.
//
// Example Vibe code:
//
//	nil
func (p *Parser) parseNilLiteral() ast.Expression {
	return &ast.NilLiteral{Token: p.curToken}
}

// Helper functions for checking token types

// curTokenIs checks if the current token is of the specified type.
func (p *Parser) curTokenIs(t lexer.TokenType) bool {
	return p.curToken.Type == t
}

// peekTokenIs checks if the next token is of the specified type.
func (p *Parser) peekTokenIs(t lexer.TokenType) bool {
	return p.peekToken.Type == t
}

// expectPeek checks if the next token is of the expected type,
// advances the parser if it is, and returns true.
// If not, it adds an error and returns false.
func (p *Parser) expectPeek(t lexer.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

// peekPrecedence returns the precedence of the next token.
// If the token has no defined precedence, returns LOWEST.
func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.peekToken.Type]; ok {
		return p
	}
	return LOWEST
}

// curPrecedence returns the precedence of the current token.
// If the token has no defined precedence, returns LOWEST.
func (p *Parser) curPrecedence() int {
	if p, ok := precedences[p.curToken.Type]; ok {
		return p
	}
	return LOWEST
}

// registerPrefix registers a prefix parse function for a token type.
// This function will be called when the token appears at the start of an expression.
func (p *Parser) registerPrefix(tokenType lexer.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

// registerInfix registers an infix parse function for a token type.
// This function will be called when the token appears between two expressions.
func (p *Parser) registerInfix(tokenType lexer.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

// The following functions are stubs for parsing various expressions.
// In a complete implementation, these would be fully implemented.

// parsePrefixExpression parses a prefix expression.
// Prefix expressions are expressions where the operator comes before the operand.
//
// Example Vibe code:
//
//	-5
//	!true
func (p *Parser) parsePrefixExpression() ast.Expression {
	// Implement prefix expression parsing
	return nil
}

// parseInfixExpression parses an infix expression.
// Infix expressions are expressions where the operator comes between two operands.
//
// Example Vibe code:
//
//	1 + 2
//	a == b
func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	// Implement infix expression parsing
	return nil
}

// parseGroupedExpression parses expressions in parentheses.
// Grouped expressions have higher precedence than their surroundings.
//
// Example Vibe code:
//
//	(1 + 2) * 3
func (p *Parser) parseGroupedExpression() ast.Expression {
	// Implement grouped expression parsing
	return nil
}

// parseFunctionLiteral parses a function definition.
// Functions in Vibe can be named or anonymous and have optional type annotations.
//
// Example Vibe code:
//
//	func add(x: Int, y: Int): Int { return x + y; }
//	func() { puts("Hello!"); }
func (p *Parser) parseFunctionLiteral() ast.Expression {
	// Implement function literal parsing
	return nil
}

// parseIfExpression parses an if expression.
// If expressions can have optional else clauses.
//
// Example Vibe code:
//
//	if x > 5 { return true; } else { return false; }
func (p *Parser) parseIfExpression() ast.Expression {
	// Implement if expression parsing
	return nil
}

// parseClassLiteral parses a class definition.
// Classes in Vibe can contain properties and methods.
//
// Example Vibe code:
//
//	class Person {
//	  prop name: String;
//	  func greet() { puts("Hello, I'm " + @name); }
//	}
func (p *Parser) parseClassLiteral() ast.Expression {
	// Implement class literal parsing
	return nil
}

// parseCallExpression parses a function call.
// Function calls consist of a function expression followed by arguments in parentheses.
//
// Example Vibe code:
//
//	add(1, 2)
//	person.greet()
func (p *Parser) parseCallExpression(function ast.Expression) ast.Expression {
	// Implement call expression parsing
	return nil
}

// parseAssignmentExpression parses an assignment expression.
// Assignment expressions are used to assign values to variables.
//
// Example Vibe code:
//
//	x = 5;
//	name = "John";
func (p *Parser) parseAssignmentExpression(left ast.Expression) ast.Expression {
	var name *ast.Identifier
	var typeAnnotation *ast.TypeAnnotation

	// Check if we have a simple identifier or a typed identifier
	if typedIdent, ok := left.(*ast.TypedIdentifier); ok {
		name = typedIdent.Identifier
		typeAnnotation = typedIdent.Type
	} else if ident, ok := left.(*ast.Identifier); ok {
		name = ident
		typeAnnotation = nil
	} else {
		p.addError(fmt.Sprintf("expected identifier, got %T", left))
		return nil
	}

	// Initialize assignment expression
	expression := &ast.AssignmentExpression{
		Token: p.curToken,  // the = token
		Name:  name,
		Type:  typeAnnotation,
	}

	// Parse the value
	p.nextToken()
	expression.Value = p.parseExpression(LOWEST)

	return expression
}

// parseArrayLiteral parses an array literal.
// Array literals are used to create arrays of values.
//
// Example Vibe code:
//
//	[1, 2, 3]
//	["apple", "banana", "cherry"]
func (p *Parser) parseArrayLiteral() ast.Expression {
	array := &ast.ArrayLiteral{Token: p.curToken}

	array.Elements = p.parseExpressionList(lexer.RBRACKET)

	return array
}

// parseExpressionList parses a list of expressions until the end token is encountered.
// Used for array elements and function call arguments.
func (p *Parser) parseExpressionList(end lexer.TokenType) []ast.Expression {
	list := []ast.Expression{}

	// Handle empty lists
	if p.peekTokenIs(end) {
		p.nextToken()
		return list
	}

	// Parse the first expression
	p.nextToken()
	list = append(list, p.parseExpression(LOWEST))

	// Parse any additional expressions separated by commas
	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken() // consume the comma
		p.nextToken() // move to the next expression
		list = append(list, p.parseExpression(LOWEST))
	}

	// Expect the ending token
	if !p.expectPeek(end) {
		return nil
	}

	return list
}