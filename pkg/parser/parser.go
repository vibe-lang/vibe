package parser

import (
	"fmt"
	"strconv"
	"strings"

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
const (
	LOWEST  = 1 // Lowest precedence (evaluated last)
	EQUALS  = 2 // ==, !=
	COMPARE = 3 // <, >, <=, >=
	SUM     = 4 // +, -
	PRODUCT = 5 // *, /
	PREFIX  = 6 // -X, !X
	CALL    = 7 // myFunction(X)
	INDEX   = 8 // array[index]
	DOT     = 9 // obj.property (highest precedence)
)

// precedences maps token types to their precedence levels.
var precedences = map[lexer.TokenType]int{
	lexer.EQ:       EQUALS,
	lexer.NOT_EQ:   EQUALS,
	lexer.LT:       COMPARE,
	lexer.GT:       COMPARE,
	lexer.LTE:      COMPARE,
	lexer.GTE:      COMPARE,
	lexer.PLUS:     SUM,
	lexer.MINUS:    SUM,
	lexer.SLASH:    PRODUCT,
	lexer.ASTERISK: PRODUCT,
	lexer.LPAREN:   CALL,
	lexer.DOT:      DOT,
	lexer.LBRACKET: INDEX,
	lexer.DOTDOT:   LOWEST, // Range operator precedence
	lexer.DOTDOTDOT: LOWEST, // Range operator precedence
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
//	l := lexer.New("x = 5;")
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
	p.registerPrefix(lexer.STRUCT, p.parseCompoundLiteral)
	p.registerPrefix(lexer.DEF, p.parseFunctionLiteral)
	p.registerPrefix(lexer.ASSIGN, p.parseAssignError)

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
	p.registerInfix(lexer.DOT, p.parseDotExpression)
	p.registerInfix(lexer.DOTDOT, p.parseRangeExpression)
	p.registerInfix(lexer.DOTDOTDOT, p.parseRangeExpression)
	p.registerInfix(lexer.LBRACKET, p.parseIndexExpression)
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

// ParseProgram parses a complete program.
// It parses statements until it encounters an EOF token.
func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}
	program.Statements = []ast.Statement{}

	for !p.curTokenIs(lexer.EOF) {
		// Debug: print position
		fmt.Printf("Token at %d:%d: %s (%s), next: %s (%s)\n",
			p.curToken.Line, p.curToken.Column,
			p.curToken.Type, p.curToken.Literal,
			p.peekToken.Type, p.peekToken.Literal)

		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}

	return program
}

// parseStatement parses a statement.
// Statements in Vibe include let statements, return statements, and expression statements.
func (p *Parser) parseStatement() ast.Statement {
	fmt.Printf("parseStatement: curToken=%s(%s), peekToken=%s(%s)\n",
		p.curToken.Type, p.curToken.Literal, p.peekToken.Type, p.peekToken.Literal)

	switch p.curToken.Type {
	case lexer.RETURN:
		return p.parseReturnStatement()
	case lexer.STRUCT:
		return p.parseStructStatement()
	case lexer.FOR:
		return p.parseForStatement()
	default:
		return p.parseExpressionStatement()
	}
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

// parseExpressionStatement parses a statement that consists of just an expression.
// Format: <expression>;
//
// Example Vibe code:
//
//	x + 5;
//	foo();
func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	fmt.Printf("parseExpressionStatement: curToken=%s(%s), peekToken=%s(%s)\n",
		p.curToken.Type, p.curToken.Literal, p.peekToken.Type, p.peekToken.Literal)

	stmt := &ast.ExpressionStatement{Token: p.curToken}

	// Special case for typed array assignments like "h: int[] = [1, 2, 3]"
	if p.curTokenIs(lexer.IDENT) && p.peekTokenIs(lexer.COLON) {
		fmt.Printf("  Found typed assignment pattern\n")
		ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

		p.nextToken() // consume the identifier
		p.nextToken() // consume the colon

		// Parse the type annotation
		typeAnnotation := p.parseTypeAnnotation()
		fmt.Printf("  After type annotation: curToken=%s(%s), peekToken=%s(%s)\n",
			p.curToken.Type, p.curToken.Literal, p.peekToken.Type, p.peekToken.Literal)

		// Check for assignment
		if p.peekTokenIs(lexer.ASSIGN) {
			fmt.Printf("  Found assignment after type annotation\n")
			p.nextToken() // move to the equals sign

			// Create a typed identifier
			typedIdent := &ast.TypedIdentifier{
				Token: ident.Token,
				Identifier: ident,
				Type: typeAnnotation,
			}

			// Create an assignment expression
			assignment := &ast.AssignmentExpression{
				Token: p.curToken,
				Left: typedIdent,
			}

			p.nextToken() // consume the equals sign
			fmt.Printf("  After consuming equals: curToken=%s(%s), peekToken=%s(%s)\n",
				p.curToken.Type, p.curToken.Literal, p.peekToken.Type, p.peekToken.Literal)

			assignment.Value = p.parseExpression(LOWEST)

			stmt.Expression = assignment

			// Optional semicolon
			if p.peekTokenIs(lexer.SEMICOLON) {
				p.nextToken()
			}

			return stmt
		}
	} else {
		// Regular expression statement
		stmt.Expression = p.parseExpression(LOWEST)
	}

	// Optional semicolon
	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parseExpression parses an expression with the given precedence.
func (p *Parser) parseExpression(precedence int) ast.Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.addError(fmt.Sprintf("no prefix parse function for %s found", p.curToken.Type))
		return nil
	}
	leftExp := prefix()

	// Check for assignment expression
	if p.peekTokenIs(lexer.ASSIGN) {
		if _, ok := leftExp.(*ast.Identifier); ok {
			return p.parseAssignmentExpression(leftExp)
		}
		if _, ok := leftExp.(*ast.TypedIdentifier); ok {
			return p.parseAssignmentExpression(leftExp)
		}
	}

	// Check for type annotation
	if p.peekTokenIs(lexer.COLON) {
		if ident, ok := leftExp.(*ast.Identifier); ok {
			p.nextToken() // consume the identifier
			p.nextToken() // consume the colon

			// Parse the type annotation
			typeAnnotation := p.parseTypeAnnotation()

			// Check for assignment
			if p.peekTokenIs(lexer.ASSIGN) {
				// Create an assignment expression with type annotation
				p.nextToken() // consume the type

				// Create an assignment expression
				stmt := &ast.AssignmentExpression{
					Token: p.curToken,
					Name: ident,
					TypeAnnotation: typeAnnotation,
				}

				p.nextToken() // consume the equals sign
				stmt.Value = p.parseExpression(LOWEST)

				return stmt
			}

			// Just a typed identifier without assignment
			typeAnno, ok := typeAnnotation.(*ast.TypeAnnotation)
			if !ok {
				p.addError(fmt.Sprintf("expected type annotation, got %T", typeAnnotation))
				return nil
			}

			return &ast.TypedIdentifier{
				Token: ident.Token,
				Identifier: ident,
				Type: typeAnno,
			}
		}
	}

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
	// Special case for Range constructor
	if p.curToken.Literal == "Range" && p.peekTokenIs(lexer.LPAREN) {
		return p.parseRangeCall(&ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})
	}

	// First, check if this is a struct instantiation
	if p.peekTokenIs(lexer.LPAREN) {
		// Look for a matching struct definition
		// For now, we'll assume any identifier followed by parentheses
		// could be a struct constructor
		return p.parseStructLiteral()
	}

	// Check if this might be an identifier with a type annotation (for assignments)
	if p.peekTokenIs(lexer.COLON) {
		ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

		// Consume the colon
		p.nextToken()

		// Parse the type name
		if !p.peekTokenIs(lexer.IDENT) && !p.peekTokenIs(lexer.LBRACKET) {
			p.addError(fmt.Sprintf("expected type name after ':', got %s", p.peekToken.Type))
			return ident
		}

		p.nextToken()

		// Handle compound type annotation like [int, string]
		if p.curTokenIs(lexer.LBRACKET) {
			typeAnnotation := &ast.TypeAnnotation{
				Token: p.curToken,
				IsCompoundType: true,
				CompoundTypes: []string{},
			}

			// Parse the types inside the brackets
			for {
				if !p.peekTokenIs(lexer.IDENT) {
					if p.peekTokenIs(lexer.RBRACKET) {
						// Empty brackets or end of list
						p.nextToken()
						break
					}
					p.addError(fmt.Sprintf("expected type name in compound type, got %s", p.peekToken.Type))
					return ident
				}

				p.nextToken() // move to the type name
				typeAnnotation.CompoundTypes = append(typeAnnotation.CompoundTypes, p.curToken.Literal)

				p.nextToken() // move past the type name

				if p.curTokenIs(lexer.RBRACKET) {
					break // End of compound type
				}

				if !p.curTokenIs(lexer.COMMA) {
					p.addError(fmt.Sprintf("expected ',' or ']' in compound type, got %s", p.curToken.Type))
					return ident
				}
			}

			// Set the full type name
			typeName := "[" + strings.Join(typeAnnotation.CompoundTypes, ", ") + "]"

			// Check for array dimensions
			for p.peekTokenIs(lexer.LBRACKET) {
				p.nextToken() // consume [

				// Expect ] to complete the array type notation
				if !p.peekTokenIs(lexer.RBRACKET) {
					p.addError(fmt.Sprintf("expected ']' to complete array type, got %s", p.peekToken.Type))
					return ident
				}

				p.nextToken() // consume ]
				typeName += "[]"
			}

			typeAnnotation.Name = typeName

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
		} else {
			// Regular type annotation
			typeName := p.curToken.Literal

			// Check for array type notation (e.g., "int[]" or "int[][]")
			// Continue checking for array dimensions until there are no more []
			for p.peekTokenIs(lexer.LBRACKET) {
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

// parseStringLiteral parses a string literal and handles interpolation.
// In Vibe, string literals are enclosed in double quotes.
// String interpolation is supported using ${expression} syntax.
func (p *Parser) parseStringLiteral() ast.Expression {
	str := &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}

	// Check if the string contains any interpolation (marked by ${)
	if strings.Contains(str.Value, "${") {
		return p.parseStringInterpolation(str)
	}

	return str
}

// parseStringInterpolation handles string literals with interpolation.
// It parses the interpolated expressions and creates a StringInterpolationLiteral.
func (p *Parser) parseStringInterpolation(str *ast.StringLiteral) ast.Expression {
	interpolation := &ast.StringInterpolationLiteral{
		Token: str.Token,
		Value: str.Value,
		Expressions: []ast.Expression{},
		Parts: []string{},
	}

	// Example: "Hello, ${name}!"
	// We need to extract "Hello, " and "!" as parts, and "name" as an expression

	// Simple regex-like approach for basic cases
	value := str.Value
	for {
		// Find the next ${
		start := strings.Index(value, "${")
		if start == -1 {
			// No more interpolations
			if len(value) > 0 {
				interpolation.Parts = append(interpolation.Parts, value)
			}
			break
		}

		// Add the text before the interpolation
		if start > 0 {
			interpolation.Parts = append(interpolation.Parts, value[:start])
		} else {
			// If the string starts with an interpolation, add an empty part
			interpolation.Parts = append(interpolation.Parts, "")
		}

		// Find the closing }
		end := strings.Index(value[start:], "}") + start
		if end <= start {
			// Missing closing brace
			p.addError("unterminated string interpolation")
			break
		}

		// Extract the variable name/expression
		exprText := value[start+2:end]

		// For now, just handle simple identifiers
		expr := &ast.Identifier{
			Token: lexer.Token{
				Type:    lexer.IDENT,
				Literal: exprText,
			},
			Value: exprText,
		}
		interpolation.Expressions = append(interpolation.Expressions, expr)

		// Move to the rest of the string
		if end+1 < len(value) {
			value = value[end+1:]
		} else {
			// End of string
			interpolation.Parts = append(interpolation.Parts, "")
			break
		}
	}

	return interpolation
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
func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.peekToken.Type]; ok {
		return p
	}
	return LOWEST
}

// curPrecedence returns the precedence of the current token.
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
	expr := &ast.InfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	expr.Right = p.parseExpression(precedence)

	return expr
}

// parseGroupedExpression parses expressions in parentheses.
// Grouped expressions have higher precedence than their surroundings.
//
// Example Vibe code:
//
//	(1 + 2) * 3
func (p *Parser) parseGroupedExpression() ast.Expression {
	p.nextToken() // Skip the opening parenthesis

	// Parse the expression with lowest precedence (resets precedence chain)
	exp := p.parseExpression(LOWEST)

	// Expect closing parenthesis
	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	return exp
}

// parseFunctionLiteral parses a function literal or definition.
// Supports both anonymous functions (fn) and named functions (def).
//
// Example Vibe code:
//
//	fn(x, y) { x + y }
//	def greet(name: string): string
//	  "Hello, ${name}!"
//	end
func (p *Parser) parseFunctionLiteral() ast.Expression {
	lit := &ast.FunctionLiteral{Token: p.curToken}
	isDef := p.curTokenIs(lexer.DEF)

	// For functions defined with 'def', we expect a name
	if isDef {
		// Expect the next token to be the function name
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}

		lit.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	}

	// Parse parameters
	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}

	lit.Parameters = p.parseFunctionParameters()

	// Check for return type annotation
	if p.peekTokenIs(lexer.COLON) {
		p.nextToken() // consume the colon

		// Parse the return type annotation
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}

		returnType := &ast.TypeAnnotation{
			Token: p.curToken,
			Name:  p.curToken.Literal,
		}
		lit.ReturnType = returnType
	}

	// Handle the function body
	if isDef {
		// For def functions, we expect a block of statements until 'end'
		lit.Body = &ast.BlockStatement{
			Token:      p.curToken,
			Statements: []ast.Statement{},
		}

		p.nextToken() // move past the return type or parameters

		// Parse statements until we reach 'end'
		for !p.curTokenIs(lexer.END) && !p.curTokenIs(lexer.EOF) {
			stmt := p.parseStatement()
			if stmt != nil {
				lit.Body.Statements = append(lit.Body.Statements, stmt)
			}
			p.nextToken()
		}

		if !p.curTokenIs(lexer.END) {
			p.addError("expected 'end' to close function definition")
			return nil
		}
	} else {
		// For fn functions, we expect a block delimited by braces
		if !p.expectPeek(lexer.LBRACE) {
			return nil
		}

		lit.Body = p.parseBlockStatement()
	}

	return lit
}

// parseIfExpression parses an if expression.
// If expressions can have optional else clauses.
//
// Example Vibe code:
//
//	if x > 5
//	    return true
//	else
//	    return false
//	end
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
	// Special case for Range constructor
	if ident, ok := function.(*ast.Identifier); ok && ident.Value == "Range" {
		return p.parseRangeCall(ident)
	}

	// Regular function call
	expr := &ast.CallExpression{Token: p.curToken, Function: function}
	expr.Arguments = p.parseExpressionList(lexer.RPAREN)
	return expr
}

// parseRangeCall parses a Range constructor call.
// Example: Range(1, 10) or Range(1, 10, true)
func (p *Parser) parseRangeCall(ident ast.Expression) ast.Expression {
	expr := &ast.RangeCallExpression{
		Token:     p.curToken,
		Exclusive: false,
	}

	// Expect opening parenthesis
	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}

	// Parse start value
	p.nextToken()
	expr.Start = p.parseExpression(LOWEST)
	if expr.Start == nil {
		return nil
	}

	// Expect comma
	if !p.expectPeek(lexer.COMMA) {
		return nil
	}

	// Parse end value
	p.nextToken()
	expr.End = p.parseExpression(LOWEST)
	if expr.End == nil {
		return nil
	}

	// Check for optional third argument (exclusive flag)
	if p.peekTokenIs(lexer.COMMA) {
		p.nextToken() // consume comma
		p.nextToken() // move to the exclusive flag

		// The third argument should be a boolean
		if boolLit, ok := p.parseExpression(LOWEST).(*ast.BooleanLiteral); ok {
			expr.Exclusive = boolLit.Value
		} else {
			p.addError("third argument to Range must be a boolean")
			return nil
		}
	}

	// Expect closing parenthesis
	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	return expr
}

// parseAssignmentExpression parses an assignment expression.
// Assignment expressions are used to assign values to variables.
//
// Example Vibe code:
//
//	x = 5;
//	name = "John";
func (p *Parser) parseAssignmentExpression(name ast.Expression) ast.Expression {
	stmt := &ast.AssignmentExpression{Token: p.curToken}

	// Check if the left side is an identifier or a typed identifier
	if ident, ok := name.(*ast.Identifier); ok {
		stmt.Name = ident
	} else if typedIdent, ok := name.(*ast.TypedIdentifier); ok {
		stmt.Name = typedIdent.Identifier
		stmt.TypeAnnotation = typedIdent.Type
	} else {
		p.errors = append(p.errors, fmt.Sprintf("expected identifier on left side of assignment, got %s", name.TokenLiteral()))
		return nil
	}

	// Check for type annotation if not already set from a TypedIdentifier
	if stmt.TypeAnnotation == nil && p.peekTokenIs(lexer.COLON) {
		p.nextToken() // consume the colon
		p.nextToken() // move to the type

		// Parse the type annotation
		stmt.TypeAnnotation = p.parseTypeAnnotation()
		if stmt.TypeAnnotation == nil {
			return nil
		}

		// Check for compound type annotation like [int, string]
		if _, ok := stmt.TypeAnnotation.(*ast.CompoundTypeAnnotation); ok {
			// If the value is an array literal, convert it to a compound literal
			p.nextToken() // move to the equals sign
			p.nextToken() // move to the value

			if p.curTokenIs(lexer.LBRACKET) {
				// Parse the value as a compound literal
				stmt.Value = p.parseCompoundLiteral()
				return stmt
			}
		}

		// Expect equals sign after type annotation
		if !p.expectPeek(lexer.ASSIGN) {
			return nil
		}
	} else if stmt.TypeAnnotation == nil {
		// No type annotation, expect equals sign
		if !p.expectPeek(lexer.ASSIGN) {
			return nil
		}
	}

	p.nextToken() // move to the value
	stmt.Value = p.parseExpression(LOWEST)

	return stmt
}

// parseTypeAnnotation parses a type annotation.
// This function handles various type annotations including:
// - Simple types (e.g., 'int', 'string')
// - Array types (e.g., 'int[]', 'string[]')
// - Multi-dimensional array types (e.g., 'int[][]', 'string[][]')
// - Compound types (e.g., '[int, string]')
func (p *Parser) parseTypeAnnotation() ast.Expression {
	// Check for compound type annotation like [int, string]
	if p.curTokenIs(lexer.LBRACKET) {
		return p.parseCompoundTypeAnnotation()
	}

	// Simple type annotation
	ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	baseTypeName := ident.Value
	dimensions := 0

	// Check for array type annotation (possibly multi-dimensional)
	for p.peekTokenIs(lexer.LBRACKET) && p.peekPeekTokenIs(lexer.RBRACKET) {
		p.nextToken() // consume the '['
		p.nextToken() // consume the ']'
		dimensions++
	}

	// If we parsed array dimensions, create the appropriate type annotation
	if dimensions > 0 {
		typeName := baseTypeName
		for i := 0; i < dimensions; i++ {
			typeName += "[]"
		}

		return &ast.TypeAnnotation{
			Token: ident.Token,
			Name:  typeName,
		}
	}

	// For simple types, return a TypeAnnotation
	return &ast.TypeAnnotation{
		Token: ident.Token,
		Name:  baseTypeName,
	}
}

// parseCompoundTypeAnnotation parses a compound type annotation like [int, string].
func (p *Parser) parseCompoundTypeAnnotation() ast.Expression {
	annotation := &ast.CompoundTypeAnnotation{
		Token: p.curToken,
		Types: []ast.Expression{},
	}

	p.nextToken() // consume the '['

	// Parse the type list
	for !p.curTokenIs(lexer.RBRACKET) {
		// Parse the type
		typeExpr := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		annotation.Types = append(annotation.Types, typeExpr)

		// Check for comma or end of list
		if p.peekTokenIs(lexer.COMMA) {
			p.nextToken() // consume the type
			p.nextToken() // consume the comma
		} else if p.peekTokenIs(lexer.RBRACKET) {
			p.nextToken() // consume the type
			break
		} else {
			p.errors = append(p.errors, fmt.Sprintf("expected comma or closing bracket, got %s", p.peekToken.Literal))
			return nil
		}
	}

	p.nextToken() // consume the ']'

	// Check for array type annotation
	if p.peekTokenIs(lexer.LBRACKET) && p.peekPeekTokenIs(lexer.RBRACKET) {
		p.nextToken() // consume the '['
		p.nextToken() // consume the ']'

		// Create an array type annotation
		return &ast.ArrayTypeAnnotation{
			Token:    annotation.Token,
			BaseType: annotation,
		}
	}

	return annotation
}

// peekPeekTokenIs returns true if the token after the peek token is of the given type.
func (p *Parser) peekPeekTokenIs(t lexer.TokenType) bool {
	// Save the current tokens
	savedCur := p.curToken
	savedPeek := p.peekToken

	// Advance to the next token
	p.nextToken()

	// Check if the new peek token is of the given type
	result := p.peekTokenIs(t)

	// Restore the tokens
	p.curToken = savedCur
	p.peekToken = savedPeek

	return result
}

// parseArrayLiteral parses an array literal expression.
// Example: [1, 2, 3]
func (p *Parser) parseArrayLiteral() ast.Expression {
	array := &ast.ArrayLiteral{Token: p.curToken}

	array.Elements = p.parseExpressionList(lexer.RBRACKET)

	return array
}

// parseCompoundLiteral parses a compound literal expression.
// This is similar to an array literal but represents a heterogeneous tuple.
// Example: [1, "hello", true] with type [int, string, boolean]
func (p *Parser) parseCompoundLiteral() ast.Expression {
	compound := &ast.CompoundLiteral{Token: p.curToken}

	compound.Elements = p.parseExpressionList(lexer.RBRACKET)

	return compound
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

// parseStructStatement parses a struct definition statement.
// Struct definitions create a new user-defined type with fields.
//
// Example Vibe code:
//
//  struct SomeStruct
//    n = 4
//    l: int = 9
//    u: string = "vibe"
//    arr: int[] = [1, 2, 5]
//  end
func (p *Parser) parseStructStatement() *ast.StructStatement {
	stmt := &ast.StructStatement{Token: p.curToken}

	// Expect struct name
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}

	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// Parse fields until we reach 'end'
	stmt.Fields = []ast.Statement{}

	// Move past the struct name
	p.nextToken()

	// Keep parsing fields until we reach 'end'
	for !p.curTokenIs(lexer.END) && !p.curTokenIs(lexer.EOF) {
		field := p.parseStatement()
		if field != nil {
			stmt.Fields = append(stmt.Fields, field)
		}
		p.nextToken()
	}

	if !p.curTokenIs(lexer.END) {
		p.addError(fmt.Sprintf("expected 'end' to close struct definition, got %s", p.curToken.Type))
		return nil
	}

	return stmt
}

// parseStructLiteral parses a struct instantiation expression
// Example: SomeStruct() or SomeStruct(n: 10, u: "lang")
func (p *Parser) parseStructLiteral() ast.Expression {
	structLiteral := &ast.StructLiteral{
		Token: p.curToken,
		Type: p.curToken.Literal,
		Fields: make(map[string]ast.Expression),
	}

	// Optional parentheses for fields
	if p.peekTokenIs(lexer.LPAREN) {
		p.nextToken() // consume (

		// Handle empty field list
		if p.peekTokenIs(lexer.RPAREN) {
			p.nextToken() // consume )
			return structLiteral
		}

		// Parse field: value pairs
		for {
			// Expect field name
			if !p.expectPeek(lexer.IDENT) {
				return nil
			}

			fieldName := p.curToken.Literal

			// Expect colon after field name
			if !p.expectPeek(lexer.COLON) {
				return nil
			}

			// Parse the field value
			p.nextToken() // move past the colon
			fieldValue := p.parseExpression(LOWEST)

			structLiteral.Fields[fieldName] = fieldValue

			// Check if we've reached the end of the field list
			if p.peekTokenIs(lexer.RPAREN) {
				p.nextToken() // consume )
				break
			}

			// Expect comma between fields
			if !p.expectPeek(lexer.COMMA) {
				return nil
			}
		}
	}

	return structLiteral
}

// parseRangeExpression parses a range expression.
// Range expressions define a sequence of values between a start and end point.
//
// Example Vibe code:
//
//	1..5  // Inclusive range from 1 to 5
//	1...5 // Exclusive range from 1 to 5 (does not include 5)
func (p *Parser) parseRangeExpression(left ast.Expression) ast.Expression {
	expr := &ast.RangeExpression{
		Token: p.curToken,
		Start: left,
	}

	// Determine if this is an inclusive or exclusive range
	expr.Exclusive = p.curToken.Type == lexer.DOTDOTDOT

	precedence := p.curPrecedence()
	p.nextToken()

	expr.End = p.parseExpression(precedence)

	return expr
}

// parseFunctionParameters parses the parameter list for a function.
// It handles both simple parameters and parameters with type annotations.
func (p *Parser) parseFunctionParameters() []*ast.Identifier {
	identifiers := []*ast.Identifier{}

	// Empty parameter list
	if p.peekTokenIs(lexer.RPAREN) {
		p.nextToken()
		return identifiers
	}

	p.nextToken()

	// First parameter
	ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	identifiers = append(identifiers, ident)

	// Check for type annotation
	if p.peekTokenIs(lexer.COLON) {
		p.nextToken() // consume the colon
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		// Skip past the type name for now, we're just collecting parameter identifiers
		// In a more complete implementation, we would capture the type annotation here
	}

	// Additional parameters
	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken() // consume the comma
		p.nextToken() // move to the parameter name

		ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		identifiers = append(identifiers, ident)

		// Check for type annotation
		if p.peekTokenIs(lexer.COLON) {
			p.nextToken() // consume the colon
			if !p.expectPeek(lexer.IDENT) {
				return nil
			}
			// Skip past the type name
		}
	}

	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	return identifiers
}

// parseBlockStatement parses a block of statements enclosed in braces.
// In Vibe, block statements are used in functions, if statements, etc.
func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	block := &ast.BlockStatement{Token: p.curToken, Statements: []ast.Statement{}}

	p.nextToken()

	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}

	return block
}

// parseForStatement parses a for loop statement.
// For loops in Vibe iterate over a collection (array, range, etc.).
//
// Example Vibe code:
//
//	for i in [1, 2, 3]
//	  puts(i)
//	end
func (p *Parser) parseForStatement() *ast.ForLoop {
	forLoop := &ast.ForLoop{Token: p.curToken}

	// Expect iterator variable
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}

	forLoop.Iterator = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// Expect 'in' keyword
	if !p.expectPeek(lexer.IN) {
		return nil
	}

	// Move to the collection expression
	p.nextToken()

	// Parse the collection to iterate over
	forLoop.Collection = p.parseExpression(LOWEST)

	// Parse the loop body
	forLoop.Body = &ast.BlockStatement{Token: p.curToken}
	forLoop.Body.Statements = []ast.Statement{}

	// Skip any semicolons after the collection expression
	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken()
	}

	p.nextToken() // Move to the first token of the body

	// Parse statements until we reach 'end'
	for !p.curTokenIs(lexer.END) && !p.curTokenIs(lexer.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			forLoop.Body.Statements = append(forLoop.Body.Statements, stmt)
		}
		p.nextToken()
	}

	if !p.curTokenIs(lexer.END) {
		p.addError("expected 'end' to close for loop")
		return nil
	}

	return forLoop
}

// parseDotExpression parses a dot expression for struct field access.
// Example: person.name
func (p *Parser) parseDotExpression(left ast.Expression) ast.Expression {
	expr := &ast.DotExpression{
		Token: p.curToken,
		Left:  left,
	}

	// Move past the dot
	p.nextToken()

	// The right side should be an identifier (field name)
	if !p.curTokenIs(lexer.IDENT) {
		p.addError(fmt.Sprintf("expected identifier after dot, got %s", p.curToken.Type))
		return nil
	}

	expr.Field = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	return expr
}

// parseIndexExpression parses an index expression for array access.
// Example: arr[0]
func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	expr := &ast.IndexExpression{
		Token: p.curToken,
		Left:  left,
	}

	// Move past the [
	p.nextToken()

	// Parse the index expression
	expr.Index = p.parseExpression(LOWEST)

	// Expect closing ]
	if !p.expectPeek(lexer.RBRACKET) {
		return nil
	}

	return expr
}

// parseAssignError is a special function to provide better error messages when = is found as a prefix
func (p *Parser) parseAssignError() ast.Expression {
	msg := fmt.Sprintf("unexpected assignment operator. Did you mean to use a variable name before '='?")
	p.errors = append(p.errors, fmt.Sprintf("[%d:%d] %s",
		p.curToken.Line, p.curToken.Column, msg))
	return nil
}