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

	curToken      lexer.Token // The current token being examined
	peekToken     lexer.Token // The next token (used for lookahead)
	peekPeekToken lexer.Token // The token after peek (used for two-token lookahead)

	errors []string // Errors encountered during parsing

	// Maps for handling operator precedence
	prefixParseFns map[lexer.TokenType]prefixParseFn // Functions for parsing prefix expressions
	infixParseFns  map[lexer.TokenType]infixParseFn  // Functions for parsing infix expressions

	// Temporary storage for parameter defaults and types during function parsing
	currentDefaults   []ast.Expression
	currentParamTypes []ast.Expression
	currentVariadic   bool
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
	LOWEST    = 1  // Lowest precedence (evaluated last)
	TERNARY   = 2  // condition ? a : b
	NIL_COAL  = 3  // ??
	LOGIC_OR  = 4  // ||
	LOGIC_AND = 5  // &&
	CONTAINS  = 6  // in
	EQUALS    = 7  // ==, !=
	COMPARE   = 8  // <, >, <=, >=
	SUM       = 9  // +, -
	PRODUCT   = 10 // *, /
	POWER     = 11 // **
	PREFIX    = 12 // -X, !X
	PIPE      = 13 // |>
	CALL      = 14 // myFunction(X)
	INDEX     = 15 // array[index]
	DOT       = 16 // obj.property (highest precedence)
)

// precedences maps token types to their precedence levels.
var precedences = map[lexer.TokenType]int{
	lexer.AND:          LOGIC_AND,
	lexer.OR:           LOGIC_OR,
	lexer.EQ:           EQUALS,
	lexer.NOT_EQ:       EQUALS,
	lexer.LT:           COMPARE,
	lexer.GT:           COMPARE,
	lexer.LTE:          COMPARE,
	lexer.GTE:          COMPARE,
	lexer.PLUS:         SUM,
	lexer.MINUS:        SUM,
	lexer.SLASH:        PRODUCT,
	lexer.ASTERISK:     PRODUCT,
	lexer.MODULO:       PRODUCT,
	lexer.LPAREN:       CALL,
	lexer.DOT:          DOT,
	lexer.LBRACKET:     INDEX,
	lexer.DOTDOT:       EQUALS, // Range operator precedence (below comparison but above LOWEST)
	lexer.DOTDOTDOT:    EQUALS, // Range operator precedence
	lexer.QUESTION:     TERNARY,
	lexer.PIPE_ARROW:   PIPE,
	lexer.IN:           CONTAINS,
	lexer.NIL_COALESCE: NIL_COAL,
	lexer.POWER_OP:     POWER,
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

	// Initialize the parser by reading three tokens
	// so curToken, peekToken, and peekPeekToken are all set
	p.peekPeekToken = p.l.NextToken()
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
	p.registerPrefix(lexer.LBRACE, p.parseHashLiteral)
	p.registerPrefix(lexer.TRY, p.parseTryExpression)
	p.registerPrefix(lexer.UNLESS, p.parseUnlessExpression)
	p.registerPrefix(lexer.CASE, p.parseCaseExpression)
	p.registerPrefix(lexer.ARROW, p.parseArrowFunction)
	p.registerPrefix(lexer.SELF, p.parseSelfExpression)
	p.registerPrefix(lexer.SUPER, p.parseSuperExpression)

	// Register infix parse functions
	p.infixParseFns = make(map[lexer.TokenType]infixParseFn)
	p.registerInfix(lexer.AND, p.parseInfixExpression)
	p.registerInfix(lexer.OR, p.parseInfixExpression)
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
	p.registerInfix(lexer.QUESTION, p.parseTernaryExpression)
	p.registerInfix(lexer.PIPE_ARROW, p.parsePipeExpression)
	p.registerInfix(lexer.NIL_COALESCE, p.parseNilCoalesceExpression)
	p.registerInfix(lexer.IN, p.parseInExpression)
	p.registerInfix(lexer.POWER_OP, p.parseInfixExpression)

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
	p.peekToken = p.peekPeekToken
	p.peekPeekToken = p.l.NextToken()
}

// ParseProgram parses a complete program.
// It parses statements until it encounters an EOF token.
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

// parseStatement parses a statement.
// Statements in Vibe include let statements, return statements, and expression statements.
func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case lexer.RETURN:
		return p.parseReturnStatement()
	case lexer.STRUCT:
		return p.parseStructStatement()
	case lexer.FOR:
		return p.parseForStatement()
	case lexer.WHILE:
		return p.parseWhileStatement()
	case lexer.LET:
		return p.parseLetStatement()
	case lexer.BREAK:
		return &ast.BreakStatement{Token: p.curToken}
	case lexer.CONTINUE:
		return &ast.ContinueStatement{Token: p.curToken}
	case lexer.IMPORT:
		return p.parseImportStatement()
	case lexer.THROW:
		return p.parseThrowStatement()
	case lexer.UNTIL:
		return p.parseUntilStatement()
	case lexer.ENUM:
		return p.parseEnumStatement()
	case lexer.CONST:
		return p.parseConstStatement()
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
func (p *Parser) parseReturnStatement() ast.Statement {
	stmt := &ast.ReturnStatement{Token: p.curToken}

	p.nextToken()

	if p.curTokenIs(lexer.SEMICOLON) {
		return stmt
	}

	stmt.Value = p.parseExpression(LOWEST)

	// Check for postfix if/unless on return (must be on the same line)
	if (p.peekTokenIs(lexer.IF) || p.peekTokenIs(lexer.UNLESS)) && p.peekToken.Line == p.curToken.Line {
		p.nextToken() // consume if/unless
		isUnless := p.curTokenIs(lexer.UNLESS)
		condToken := p.curToken
		p.nextToken() // move to condition
		condition := p.parseExpression(LOWEST)

		if p.peekTokenIs(lexer.SEMICOLON) {
			p.nextToken()
		}

		return &ast.PostfixCondition{
			Token:     condToken,
			Statement: stmt,
			Condition: condition,
			Unless:    isUnless,
		}
	}

	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parseLetStatement parses a let statement.
// Format: let <name> = <expression>
// or:    let <name>: <type> = <expression>
func (p *Parser) parseLetStatement() ast.Statement {
	// Skip the 'let' keyword - treat as a variable assignment
	p.nextToken()

	// Now we should be on an identifier - parse it as an expression statement
	return p.parseExpressionStatement()
}

// parseExpressionStatement parses a statement that consists of just an expression.
// Format: <expression>;
//
// Example Vibe code:
//
//	x + 5;
//	foo();
func (p *Parser) parseExpressionStatement() ast.Statement {
	stmt := &ast.ExpressionStatement{Token: p.curToken}

	// Special case for typed array assignments like "h: int[] = [1, 2, 3]"
	if p.curTokenIs(lexer.IDENT) && p.peekTokenIs(lexer.COLON) {
		ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

		p.nextToken() // consume the identifier
		p.nextToken() // consume the colon

		// Parse the type annotation
		typeAnnotation := p.parseTypeAnnotation()

		// Check for assignment
		if p.peekTokenIs(lexer.ASSIGN) {
			p.nextToken() // move to the equals sign

			// Create an assignment expression
			assignment := &ast.AssignmentExpression{
				Token: p.curToken,
				Name:  ident, TypeAnnotation: typeAnnotation,
			}

			p.nextToken() // consume the equals sign

			assignment.Value = p.parseExpression(LOWEST)

			stmt.Expression = assignment

			// Optional semicolon
			if p.peekTokenIs(lexer.SEMICOLON) {
				p.nextToken()
			}

			return stmt
		}

		// No assignment - this is just a typed identifier declaration (e.g., in struct fields)
		typeAnno, ok := typeAnnotation.(*ast.TypeAnnotation)
		if ok {
			stmt.Expression = &ast.TypedIdentifier{
				Token:      ident.Token,
				Identifier: ident,
				Type:       typeAnno,
			}
		} else {
			// Compound or array type annotation without assignment
			stmt.Expression = &ast.TypedIdentifier{
				Token:      ident.Token,
				Identifier: ident,
				Type:       &ast.TypeAnnotation{Token: ident.Token, Name: typeAnnotation.String()},
			}
		}

		// Optional semicolon
		if p.peekTokenIs(lexer.SEMICOLON) {
			p.nextToken()
		}

		return stmt
	} else if p.curTokenIs(lexer.IDENT) && p.peekTokenIs(lexer.COMMA) {
		// Check if this is a destructuring assignment: a, b, c = [1, 2, 3]
		if destructure := p.tryParseDestructure(); destructure != nil {
			stmt.Expression = destructure
			if p.peekTokenIs(lexer.SEMICOLON) {
				p.nextToken()
			}
			return stmt
		}
		// Not a destructure — fall through to regular expression
		stmt.Expression = p.parseExpression(LOWEST)
	} else {
		// Regular expression statement
		stmt.Expression = p.parseExpression(LOWEST)
	}

	// Check for postfix if/unless (must be on the same line as the expression)
	if (p.peekTokenIs(lexer.IF) || p.peekTokenIs(lexer.UNLESS)) && p.peekToken.Line == p.curToken.Line {
		p.nextToken() // consume the if/unless token
		isUnless := p.curTokenIs(lexer.UNLESS)
		condToken := p.curToken
		p.nextToken() // move to condition
		condition := p.parseExpression(LOWEST)

		wrapped := &ast.PostfixCondition{
			Token:     condToken,
			Statement: stmt,
			Condition: condition,
			Unless:    isUnless,
		}

		if p.peekTokenIs(lexer.SEMICOLON) {
			p.nextToken()
		}

		return wrapped
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

	// Check for compound assignment operators (+=, -=, *=, /=, %=)
	if p.isCompoundAssign(p.peekToken.Type) {
		if ident, ok := leftExp.(*ast.Identifier); ok {
			return p.parseCompoundAssignment(ident)
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
					Token:          p.curToken,
					Name:           ident,
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
				Token:      ident.Token,
				Identifier: ident,
				Type:       typeAnno,
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

	// Check if this is a struct instantiation: StructName(field: value, ...)
	// or a generic struct: StructName<T, U>(field: value, ...)
	// Struct constructors have the pattern: IDENT ( IDENT : ...
	// Function calls have: IDENT ( expr, ... )
	// We distinguish by checking if the first token after ( is IDENT followed by :
	// For now, check if the identifier starts with uppercase (convention for struct names)
	firstChar := p.curToken.Literal[0]
	if firstChar >= 'A' && firstChar <= 'Z' {
		// Check for generic struct/class: Name<type, type>(...)
		// We can use a non-rolling-back approach: peek=<, peekPeek=IDENT
		// Since this is uppercase, < is unambiguous (not inheritance context)
		if p.peekTokenIs(lexer.LT) && p.peekPeekTokenIs(lexer.IDENT) {
			// Save the struct/class name before consuming type args
			savedNameToken := p.curToken
			savedName := p.curToken.Literal

			// Speculatively consume < and IDENT to check what follows
			p.nextToken() // consume <
			p.nextToken() // consume first type arg IDENT
			firstName := p.curToken.Literal

			if p.peekTokenIs(lexer.GT) || p.peekTokenIs(lexer.COMMA) {
				// This is a generic instantiation: Name<type>(...)
				typeArgs := []string{firstName}
				for p.peekTokenIs(lexer.COMMA) {
					p.nextToken() // consume comma
					if !p.expectPeek(lexer.IDENT) {
						return nil
					}
					typeArgs = append(typeArgs, p.curToken.Literal)
				}
				if !p.expectPeek(lexer.GT) {
					return nil
				}
				// Create the struct literal using the saved name
				structLiteral := &ast.StructLiteral{
					Token:    savedNameToken,
					Type:     savedName,
					TypeArgs: typeArgs,
					Fields:   make(map[string]ast.Expression),
				}
				// Parse optional parenthesized fields
				if p.peekTokenIs(lexer.LPAREN) {
					p.nextToken() // consume (
					if p.peekTokenIs(lexer.RPAREN) {
						p.nextToken() // consume )
						return structLiteral
					}
					for {
						if !p.expectPeek(lexer.IDENT) {
							return nil
						}
						fieldName := p.curToken.Literal
						if !p.expectPeek(lexer.COLON) {
							return nil
						}
						p.nextToken()
						fieldValue := p.parseExpression(LOWEST)
						structLiteral.Fields[fieldName] = fieldValue
						if p.peekTokenIs(lexer.RPAREN) {
							p.nextToken() // consume )
							break
						}
						if !p.expectPeek(lexer.COMMA) {
							return nil
						}
					}
				}
				return structLiteral
			}
			// Not type args — but we've consumed tokens. This is a parsing error
			// in the context of uppercase identifiers followed by < then IDENT.
			p.addError(fmt.Sprintf("unexpected token after '%s<': expected '>' or ','", savedName))
			return nil
		}
		if p.peekTokenIs(lexer.LPAREN) && p.peekPeekTokenIs(lexer.IDENT) {
			return p.parseStructLiteral()
		}
		// Also handle empty-parens struct construction like Person()
		if p.peekTokenIs(lexer.LPAREN) && p.peekPeekTokenIs(lexer.RPAREN) {
			return p.parseStructLiteral()
		}
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
				Token:          p.curToken,
				IsCompoundType: true,
				CompoundTypes:  []string{},
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
		Token:       str.Token,
		Value:       str.Value,
		Expressions: []ast.Expression{},
		Parts:       []string{},
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

		// Extract the expression text and parse it
		exprText := value[start+2 : end]

		// Lex and parse the expression properly so we can handle
		// arbitrary expressions like function calls, dot access, etc.
		exprLexer := lexer.New(exprText)
		exprParser := New(exprLexer)
		expr := exprParser.parseExpression(LOWEST)
		if expr == nil {
			// Fallback: treat as simple identifier
			expr = &ast.Identifier{
				Token: lexer.Token{
					Type:    lexer.IDENT,
					Literal: exprText,
				},
				Value: exprText,
			}
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
	expression := &ast.PrefixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
	}

	p.nextToken()

	expression.Right = p.parseExpression(PREFIX)

	return expression
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
// Supports generic type parameters with <T, U> syntax.
//
// Example Vibe code:
//
//	fn(x, y) { x + y }
//	def greet(name: string): string
//	  "Hello, ${name}!"
//	end
//	def identity<T>(x: T): T
//	  x
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

		// Check for generic type parameters: def name<T, U>(...)
		if p.peekTokenIs(lexer.LT) {
			typeParams := p.tryParseTypeParams()
			if typeParams != nil {
				lit.TypeParams = typeParams
			}
		}
	}

	// Parse parameters
	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}

	p.currentDefaults = []ast.Expression{}
	p.currentParamTypes = []ast.Expression{}
	p.currentVariadic = false
	lit.Parameters = p.parseFunctionParameters()
	lit.ParamDefaults = p.currentDefaults
	lit.ParamTypes = p.currentParamTypes
	lit.Variadic = p.currentVariadic
	p.currentDefaults = nil
	p.currentParamTypes = nil
	p.currentVariadic = false

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
			lit.Body.Statements = append(lit.Body.Statements, stmt)
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
	expression := &ast.IfExpression{Token: p.curToken}

	// Parse the condition (no parentheses required, Ruby-style)
	p.nextToken()
	expression.Condition = p.parseExpression(LOWEST)

	// Parse the consequence block (statements until else/elsif/end)
	expression.Consequence = &ast.BlockStatement{
		Token:      p.curToken,
		Statements: []ast.Statement{},
	}

	p.nextToken() // move past the condition

	for !p.curTokenIs(lexer.ELSE) && !p.curTokenIs(lexer.ELSIF) && !p.curTokenIs(lexer.END) && !p.curTokenIs(lexer.EOF) {
		stmt := p.parseStatement()
		expression.Consequence.Statements = append(expression.Consequence.Statements, stmt)
		p.nextToken()
	}

	// Handle elsif chains
	if p.curTokenIs(lexer.ELSIF) {
		// Parse elsif as a nested if expression in the alternative block
		nestedIf := p.parseIfExpression()
		if nestedIf != nil {
			expression.Alternative = &ast.BlockStatement{
				Token: p.curToken,
				Statements: []ast.Statement{
					&ast.ExpressionStatement{
						Token:      p.curToken,
						Expression: nestedIf,
					},
				},
			}
		}
		return expression
	}

	// Handle else
	if p.curTokenIs(lexer.ELSE) {
		expression.Alternative = &ast.BlockStatement{
			Token:      p.curToken,
			Statements: []ast.Statement{},
		}

		p.nextToken() // move past 'else'

		for !p.curTokenIs(lexer.END) && !p.curTokenIs(lexer.EOF) {
			stmt := p.parseStatement()
			expression.Alternative.Statements = append(expression.Alternative.Statements, stmt)
			p.nextToken()
		}
	}

	if !p.curTokenIs(lexer.END) {
		p.addError("expected 'end' to close if expression")
		return nil
	}

	return expression
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
	lit := &ast.ClassLiteral{Token: p.curToken}

	// Expect the class name
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}

	lit.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// Check for generic type parameters or inheritance with <
	// Disambiguation: class Box<T> (generics) vs class Dog < Animal (inheritance)
	// Key insight: generics use NO space before < (Box<T>), inheritance uses space (Dog < Animal)
	// But we can't rely on whitespace. Instead, use structural disambiguation:
	// After <IDENT>, if next is , or >, it's type params. Otherwise it's inheritance.
	if p.peekTokenIs(lexer.LT) {
		// Look at peekPeek: if it's IDENT, we need further disambiguation
		if p.peekPeekTokenIs(lexer.IDENT) {
			// Consume < and the IDENT to check what follows
			p.nextToken() // consume < (now curToken)
			p.nextToken() // consume IDENT (now curToken = the name after <)
			identName := p.curToken.Literal

			if p.peekTokenIs(lexer.GT) || p.peekTokenIs(lexer.COMMA) {
				// This is type params: <T> or <T, U>
				typeParams := []string{identName}
				for p.peekTokenIs(lexer.COMMA) {
					p.nextToken() // consume comma
					if !p.expectPeek(lexer.IDENT) {
						return nil
					}
					typeParams = append(typeParams, p.curToken.Literal)
				}
				if !p.expectPeek(lexer.GT) {
					return nil
				}
				lit.TypeParams = typeParams

				// After type params, there could still be inheritance: class Box<T> < Parent
				if p.peekTokenIs(lexer.LT) {
					p.nextToken() // consume <
					if !p.expectPeek(lexer.IDENT) {
						return nil
					}
					lit.Parent = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
				}
			} else {
				// This is inheritance: class Dog < Animal (identName = Animal)
				lit.Parent = &ast.Identifier{Token: p.curToken, Value: identName}
			}
		}
	}

	// Parse the class body until 'end'
	lit.Body = &ast.BlockStatement{
		Token:      p.curToken,
		Statements: []ast.Statement{},
	}

	p.nextToken() // move past the class name

	for !p.curTokenIs(lexer.END) && !p.curTokenIs(lexer.EOF) {
		stmt := p.parseStatement()
		lit.Body.Statements = append(lit.Body.Statements, stmt)
		p.nextToken()
	}

	if !p.curTokenIs(lexer.END) {
		p.addError("expected 'end' to close class definition")
		return nil
	}

	return lit
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

	baseAnnotation := &ast.TypeAnnotation{
		Token: ident.Token,
		Name:  baseTypeName,
	}

	// Check for optional type: type?
	if p.peekTokenIs(lexer.QUESTION) {
		p.nextToken() // consume ?
		return &ast.UnionTypeAnnotation{
			Token: baseAnnotation.Token,
			Types: []ast.Expression{baseAnnotation, &ast.TypeAnnotation{Token: p.curToken, Name: "nil"}},
		}
	}

	// Check for union type: type | type
	if p.peekTokenIs(lexer.PIPE) {
		types := []ast.Expression{baseAnnotation}
		for p.peekTokenIs(lexer.PIPE) {
			p.nextToken() // consume |
			p.nextToken() // move to next type
			if p.curTokenIs(lexer.NIL) {
				types = append(types, &ast.TypeAnnotation{Token: p.curToken, Name: "nil"})
			} else {
				types = append(types, &ast.TypeAnnotation{Token: p.curToken, Name: p.curToken.Literal})
			}
		}
		return &ast.UnionTypeAnnotation{
			Token: baseAnnotation.Token,
			Types: types,
		}
	}

	return baseAnnotation
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

	// After the loop, curToken should be ']' (the closing bracket of the compound type)
	// Don't consume it yet - check for array dimensions first

	// Check for array type annotation like [string, int][] or [string, int][][]
	dimensions := 0
	for p.peekTokenIs(lexer.LBRACKET) && p.peekPeekTokenIs(lexer.RBRACKET) {
		p.nextToken() // consume the '['
		p.nextToken() // consume the ']'
		dimensions++
	}

	if dimensions > 0 {
		// Build compound type name
		types := []string{}
		for _, t := range annotation.Types {
			types = append(types, t.String())
		}
		typeName := "[" + strings.Join(types, ", ") + "]"
		for j := 0; j < dimensions; j++ {
			typeName += "[]"
		}

		return &ast.TypeAnnotation{
			Token:          annotation.Token,
			Name:           typeName,
			IsCompoundType: true,
			CompoundTypes:  types,
		}
	}

	return annotation
}

// peekPeekTokenIs returns true if the token after the peek token is of the given type.
// This uses the pre-fetched peekPeekToken and does not consume any tokens.
func (p *Parser) peekPeekTokenIs(t lexer.TokenType) bool {
	return p.peekPeekToken.Type == t
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
//	struct SomeStruct
//	  n = 4
//	  l: int = 9
//	  u: string = "vibe"
//	  arr: int[] = [1, 2, 5]
//	end
func (p *Parser) parseStructStatement() *ast.StructStatement {
	stmt := &ast.StructStatement{Token: p.curToken}

	// Expect struct name
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}

	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// Check for generic type parameters: struct Pair<A, B>
	if p.peekTokenIs(lexer.LT) {
		typeParams := p.tryParseTypeParams()
		if typeParams != nil {
			stmt.TypeParams = typeParams
		}
	}

	// Parse fields until we reach 'end'
	stmt.Fields = []ast.Statement{}

	// Move past the struct name
	p.nextToken()

	// Keep parsing fields until we reach 'end'
	for !p.curTokenIs(lexer.END) && !p.curTokenIs(lexer.EOF) {
		field := p.parseStatement()
		stmt.Fields = append(stmt.Fields, field)
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
		Token:  p.curToken,
		Type:   p.curToken.Literal,
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
// It handles simple parameters, parameters with type annotations, and default values.
// Returns identifiers and defaults slices.
func (p *Parser) parseFunctionParameters() []*ast.Identifier {
	identifiers := []*ast.Identifier{}

	// Empty parameter list
	if p.peekTokenIs(lexer.RPAREN) {
		p.nextToken()
		return identifiers
	}

	p.nextToken()

	// Check for variadic parameter: *args
	if p.curTokenIs(lexer.ASTERISK) {
		p.nextToken() // move past *
		p.currentVariadic = true
	}

	// First parameter
	ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	identifiers = append(identifiers, ident)

	// Check for type annotation
	if p.peekTokenIs(lexer.COLON) {
		p.nextToken() // consume the colon
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		// Store the type annotation
		p.currentParamTypes = append(p.currentParamTypes, &ast.TypeAnnotation{
			Token: p.curToken,
			Name:  p.curToken.Literal,
		})
	} else {
		p.currentParamTypes = append(p.currentParamTypes, nil)
	}

	// Check for default value
	if p.peekTokenIs(lexer.ASSIGN) {
		p.nextToken() // consume =
		p.nextToken() // move to default value expression
		defaultVal := p.parseExpression(LOWEST)
		p.currentDefaults = append(p.currentDefaults, defaultVal)
	} else {
		p.currentDefaults = append(p.currentDefaults, nil)
	}

	// Additional parameters
	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken() // consume the comma
		p.nextToken() // move to the parameter name

		// Check for variadic parameter: *args
		if p.curTokenIs(lexer.ASTERISK) {
			p.nextToken() // move past *
			p.currentVariadic = true
		}

		ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		identifiers = append(identifiers, ident)

		// Check for type annotation
		if p.peekTokenIs(lexer.COLON) {
			p.nextToken() // consume the colon
			if !p.expectPeek(lexer.IDENT) {
				return nil
			}
			// Store the type annotation
			p.currentParamTypes = append(p.currentParamTypes, &ast.TypeAnnotation{
				Token: p.curToken,
				Name:  p.curToken.Literal,
			})
		} else {
			p.currentParamTypes = append(p.currentParamTypes, nil)
		}

		// Check for default value
		if p.peekTokenIs(lexer.ASSIGN) {
			p.nextToken() // consume =
			p.nextToken() // move to default value expression
			defaultVal := p.parseExpression(LOWEST)
			p.currentDefaults = append(p.currentDefaults, defaultVal)
		} else {
			p.currentDefaults = append(p.currentDefaults, nil)
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
		block.Statements = append(block.Statements, stmt)
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

	// Check for second iterator: for k, v in hash
	if p.peekTokenIs(lexer.COMMA) {
		p.nextToken() // consume comma
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		forLoop.ValueIterator = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	}

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
		forLoop.Body.Statements = append(forLoop.Body.Statements, stmt)
		p.nextToken()
	}

	if !p.curTokenIs(lexer.END) {
		p.addError("expected 'end' to close for loop")
		return nil
	}

	return forLoop
}

// parseDotExpression parses a dot expression for struct field access.
// Also handles dot assignment: obj.field = value
func (p *Parser) parseDotExpression(left ast.Expression) ast.Expression {
	dotToken := p.curToken

	// Move past the dot
	p.nextToken()

	if !p.curTokenIs(lexer.IDENT) {
		p.addError(fmt.Sprintf("expected identifier after dot, got %s", p.curToken.Type))
		return nil
	}

	fieldName := p.curToken.Literal
	fieldIdent := &ast.Identifier{Token: p.curToken, Value: fieldName}

	// Check for assignment: obj.field = value
	if p.peekTokenIs(lexer.ASSIGN) {
		p.nextToken() // consume '='
		p.nextToken() // move to value
		value := p.parseExpression(LOWEST)
		return &ast.DotAssignment{
			Token: dotToken,
			Left:  left,
			Field: fieldName,
			Value: value,
		}
	}

	return &ast.DotExpression{
		Token: dotToken,
		Left:  left,
		Field: fieldIdent,
	}
}

// parseIndexExpression parses an index expression for array/hash access.
// Also handles index assignment: arr[i] = value
func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	indexToken := p.curToken

	// Move past the [
	p.nextToken()

	// Parse the index expression
	index := p.parseExpression(LOWEST)

	// Expect closing ]
	if !p.expectPeek(lexer.RBRACKET) {
		return nil
	}

	// Check for assignment: arr[i] = value
	if p.peekTokenIs(lexer.ASSIGN) {
		p.nextToken() // consume '='
		p.nextToken() // move to value
		value := p.parseExpression(LOWEST)
		return &ast.IndexAssignment{
			Token: indexToken,
			Left:  left,
			Index: index,
			Value: value,
		}
	}

	return &ast.IndexExpression{
		Token: indexToken,
		Left:  left,
		Index: index,
	}
}

// parseAssignError is a special function to provide better error messages when = is found as a prefix
func (p *Parser) parseAssignError() ast.Expression {
	msg := "unexpected assignment operator. Did you mean to use a variable name before '='?"
	p.errors = append(p.errors, fmt.Sprintf("[%d:%d] %s",
		p.curToken.Line, p.curToken.Column, msg))
	return nil
}

// parseWhileStatement parses a while loop: while <condition> ... end
func (p *Parser) parseWhileStatement() *ast.WhileLoop {
	loop := &ast.WhileLoop{Token: p.curToken}

	p.nextToken()
	loop.Condition = p.parseExpression(LOWEST)

	loop.Body = &ast.BlockStatement{Token: p.curToken, Statements: []ast.Statement{}}

	p.nextToken()

	for !p.curTokenIs(lexer.END) && !p.curTokenIs(lexer.EOF) {
		stmt := p.parseStatement()
		loop.Body.Statements = append(loop.Body.Statements, stmt)
		p.nextToken()
	}

	if !p.curTokenIs(lexer.END) {
		p.addError("expected 'end' to close while loop")
		return nil
	}

	return loop
}

// parseHashLiteral parses a hash map literal: {key: value, ...}
// Supports both bare identifier keys ({host: "localhost"})
// and expression keys ({"host": "localhost"}).
func (p *Parser) parseHashLiteral() ast.Expression {
	hash := &ast.HashLiteral{Token: p.curToken}
	hash.Pairs = make(map[ast.Expression]ast.Expression)

	for !p.peekTokenIs(lexer.RBRACE) {
		p.nextToken()

		var key ast.Expression
		// Handle bare identifier keys: {host: value}
		// When we see IDENT followed by COLON, treat it as a string key
		// to avoid the type annotation parser intercepting it.
		if p.curTokenIs(lexer.IDENT) && p.peekTokenIs(lexer.COLON) {
			key = &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
		} else {
			key = p.parseExpression(LOWEST)
		}

		if !p.expectPeek(lexer.COLON) {
			return nil
		}

		p.nextToken()
		value := p.parseExpression(LOWEST)

		hash.Pairs[key] = value

		if !p.peekTokenIs(lexer.RBRACE) && !p.expectPeek(lexer.COMMA) {
			return nil
		}
	}

	if !p.expectPeek(lexer.RBRACE) {
		return nil
	}

	return hash
}

// parseImportStatement parses: import "path/to/file.vb"
func (p *Parser) parseImportStatement() *ast.ImportStatement {
	stmt := &ast.ImportStatement{Token: p.curToken}

	if !p.expectPeek(lexer.STRING) {
		return nil
	}

	stmt.Path = p.curToken.Literal
	return stmt
}

// parseTryExpression parses: try ... catch e ... finally ... end
func (p *Parser) parseTryExpression() ast.Expression {
	expr := &ast.TryExpression{Token: p.curToken}

	// Parse try body
	expr.Body = &ast.BlockStatement{Token: p.curToken, Statements: []ast.Statement{}}

	p.nextToken()

	for !p.curTokenIs(lexer.CATCH) && !p.curTokenIs(lexer.EOF) {
		stmt := p.parseStatement()
		expr.Body.Statements = append(expr.Body.Statements, stmt)
		p.nextToken()
	}

	if !p.curTokenIs(lexer.CATCH) {
		p.addError("expected 'catch' in try expression")
		return nil
	}

	// Parse optional catch variable
	if p.peekTokenIs(lexer.IDENT) {
		p.nextToken()
		expr.CatchVar = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	}

	// Parse catch body
	expr.CatchBody = &ast.BlockStatement{Token: p.curToken, Statements: []ast.Statement{}}

	p.nextToken()

	for !p.curTokenIs(lexer.END) && !p.curTokenIs(lexer.FINALLY) && !p.curTokenIs(lexer.EOF) {
		stmt := p.parseStatement()
		expr.CatchBody.Statements = append(expr.CatchBody.Statements, stmt)
		p.nextToken()
	}

	// Parse optional finally block
	if p.curTokenIs(lexer.FINALLY) {
		expr.FinallyBody = &ast.BlockStatement{Token: p.curToken, Statements: []ast.Statement{}}

		p.nextToken()

		for !p.curTokenIs(lexer.END) && !p.curTokenIs(lexer.EOF) {
			stmt := p.parseStatement()
			expr.FinallyBody.Statements = append(expr.FinallyBody.Statements, stmt)
			p.nextToken()
		}
	}

	if !p.curTokenIs(lexer.END) {
		p.addError("expected 'end' to close try expression")
		return nil
	}

	return expr
}

// parseThrowStatement parses: throw <expression>
func (p *Parser) parseThrowStatement() *ast.ThrowStatement {
	stmt := &ast.ThrowStatement{Token: p.curToken}

	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)

	return stmt
}

// parseEnumStatement parses: enum Name value1 value2 ... end
func (p *Parser) parseEnumStatement() *ast.EnumStatement {
	stmt := &ast.EnumStatement{Token: p.curToken}

	if !p.expectPeek(lexer.IDENT) {
		return nil
	}

	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	p.nextToken() // move past enum name

	for !p.curTokenIs(lexer.END) && !p.curTokenIs(lexer.EOF) {
		if p.curTokenIs(lexer.IDENT) {
			stmt.Values = append(stmt.Values, p.curToken.Literal)
		}
		p.nextToken()
	}

	if !p.curTokenIs(lexer.END) {
		p.addError("expected 'end' to close enum definition")
		return nil
	}

	return stmt
}

// parseSelfExpression parses the 'self' keyword as an identifier.
func (p *Parser) parseSelfExpression() ast.Expression {
	return &ast.Identifier{Token: p.curToken, Value: "self"}
}

// parseSuperExpression parses the 'super' keyword as an identifier.
// super can be used as:
//   - super.method(args) — call a specific parent method
//   - super(args)        — call the parent's version of the current method
func (p *Parser) parseSuperExpression() ast.Expression {
	return &ast.Identifier{Token: p.curToken, Value: "super"}
}

// tryParseDestructure tries to parse a destructuring assignment: a, b = expr
// It checks if the current pattern is IDENT (COMMA IDENT)+ ASSIGN.
// Returns nil if not a destructure pattern. Since the parser doesn't support
// full backtracking, we use a scanning approach.
func (p *Parser) tryParseDestructure() ast.Expression {
	// We're sitting on the first IDENT and know peek is COMMA.
	// Scan forward to collect identifiers until we hit ASSIGN.
	names := []*ast.Identifier{
		{Token: p.curToken, Value: p.curToken.Literal},
	}

	// Save current state for rollback
	savedCur := p.curToken
	savedPeek := p.peekToken
	savedPeekPeek := p.peekPeekToken
	savedErrors := make([]string, len(p.errors))
	copy(savedErrors, p.errors)

	// Scan: COMMA IDENT COMMA IDENT ... ASSIGN
	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken() // consume comma
		if !p.peekTokenIs(lexer.IDENT) {
			// Not a destructure — rollback
			p.curToken = savedCur
			p.peekToken = savedPeek
			p.peekPeekToken = savedPeekPeek
			p.errors = savedErrors
			return nil
		}
		p.nextToken() // consume ident
		names = append(names, &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})
	}

	// After collecting names, we expect ASSIGN
	if !p.peekTokenIs(lexer.ASSIGN) {
		// Not a destructure — rollback
		p.curToken = savedCur
		p.peekToken = savedPeek
		p.peekPeekToken = savedPeekPeek
		p.errors = savedErrors
		return nil
	}

	p.nextToken() // consume ASSIGN
	assignToken := p.curToken
	p.nextToken() // move to value

	value := p.parseExpression(LOWEST)

	return &ast.DestructureAssignment{
		Token: assignToken,
		Names: names,
		Value: value,
	}
}

// parseConstStatement parses: const NAME = value or const NAME: type = value
func (p *Parser) parseConstStatement() *ast.ConstStatement {
	stmt := &ast.ConstStatement{Token: p.curToken}

	if !p.expectPeek(lexer.IDENT) {
		return nil
	}

	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// Check for optional type annotation
	if p.peekTokenIs(lexer.COLON) {
		p.nextToken() // consume colon
		p.nextToken() // move to type
		stmt.TypeAnnotation = p.parseTypeAnnotation()
	}

	if !p.expectPeek(lexer.ASSIGN) {
		return nil
	}

	p.nextToken() // move past =
	stmt.Value = p.parseExpression(LOWEST)

	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parseUnlessExpression parses: unless condition ... end or unless condition ... else ... end
func (p *Parser) parseUnlessExpression() ast.Expression {
	expression := &ast.UnlessExpression{Token: p.curToken}

	p.nextToken()
	expression.Condition = p.parseExpression(LOWEST)

	expression.Consequence = &ast.BlockStatement{
		Token:      p.curToken,
		Statements: []ast.Statement{},
	}

	p.nextToken()

	for !p.curTokenIs(lexer.ELSE) && !p.curTokenIs(lexer.END) && !p.curTokenIs(lexer.EOF) {
		stmt := p.parseStatement()
		expression.Consequence.Statements = append(expression.Consequence.Statements, stmt)
		p.nextToken()
	}

	if p.curTokenIs(lexer.ELSE) {
		expression.Alternative = &ast.BlockStatement{
			Token:      p.curToken,
			Statements: []ast.Statement{},
		}
		p.nextToken()
		for !p.curTokenIs(lexer.END) && !p.curTokenIs(lexer.EOF) {
			stmt := p.parseStatement()
			expression.Alternative.Statements = append(expression.Alternative.Statements, stmt)
			p.nextToken()
		}
	}

	if !p.curTokenIs(lexer.END) {
		p.addError("expected 'end' to close unless expression")
		return nil
	}

	return expression
}

// parseUntilStatement parses: until condition ... end
func (p *Parser) parseUntilStatement() *ast.UntilLoop {
	loop := &ast.UntilLoop{Token: p.curToken}

	p.nextToken()
	loop.Condition = p.parseExpression(LOWEST)

	loop.Body = &ast.BlockStatement{Token: p.curToken, Statements: []ast.Statement{}}

	p.nextToken()

	for !p.curTokenIs(lexer.END) && !p.curTokenIs(lexer.EOF) {
		stmt := p.parseStatement()
		loop.Body.Statements = append(loop.Body.Statements, stmt)
		p.nextToken()
	}

	if !p.curTokenIs(lexer.END) {
		p.addError("expected 'end' to close until loop")
		return nil
	}

	return loop
}

// parseCaseExpression parses: case subject when v1 ... when v2 ... else ... end
func (p *Parser) parseCaseExpression() ast.Expression {
	expr := &ast.CaseExpression{Token: p.curToken}

	p.nextToken()
	expr.Subject = p.parseExpression(LOWEST)

	p.nextToken()

	// Parse when clauses
	for p.curTokenIs(lexer.WHEN) {
		when := &ast.WhenClause{Token: p.curToken}

		// Parse one or more match values separated by commas
		p.nextToken()
		when.Values = append(when.Values, p.parseExpression(LOWEST))
		for p.peekTokenIs(lexer.COMMA) {
			p.nextToken() // consume comma
			p.nextToken() // move to next value
			when.Values = append(when.Values, p.parseExpression(LOWEST))
		}

		// Parse body until next when/else/end
		when.Body = &ast.BlockStatement{Token: p.curToken, Statements: []ast.Statement{}}
		p.nextToken()
		for !p.curTokenIs(lexer.WHEN) && !p.curTokenIs(lexer.ELSE) && !p.curTokenIs(lexer.END) && !p.curTokenIs(lexer.EOF) {
			stmt := p.parseStatement()
			when.Body.Statements = append(when.Body.Statements, stmt)
			p.nextToken()
		}

		expr.Whens = append(expr.Whens, when)
	}

	// Parse optional else
	if p.curTokenIs(lexer.ELSE) {
		expr.Default = &ast.BlockStatement{Token: p.curToken, Statements: []ast.Statement{}}
		p.nextToken()
		for !p.curTokenIs(lexer.END) && !p.curTokenIs(lexer.EOF) {
			stmt := p.parseStatement()
			expr.Default.Statements = append(expr.Default.Statements, stmt)
			p.nextToken()
		}
	}

	if !p.curTokenIs(lexer.END) {
		p.addError("expected 'end' to close case expression")
		return nil
	}

	return expr
}

// parseTernaryExpression parses: condition ? consequence : alternative
func (p *Parser) parseTernaryExpression(condition ast.Expression) ast.Expression {
	expr := &ast.TernaryExpression{
		Token:     p.curToken,
		Condition: condition,
	}

	p.nextToken() // move past ?
	expr.Consequence = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.COLON) {
		p.addError("expected ':' in ternary expression")
		return nil
	}

	p.nextToken() // move past :
	expr.Alternative = p.parseExpression(LOWEST)

	return expr
}

// parseArrowFunction parses: -> (params) { body } or -> { body }
func (p *Parser) parseArrowFunction() ast.Expression {
	af := &ast.ArrowFunction{Token: p.curToken}

	if p.peekTokenIs(lexer.LPAREN) {
		p.nextToken() // consume ->
		p.currentDefaults = []ast.Expression{}
		p.currentParamTypes = []ast.Expression{}
		af.Parameters = p.parseFunctionParameters()
		p.currentDefaults = nil
		p.currentParamTypes = nil
	} else {
		af.Parameters = []*ast.Identifier{}
	}

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	af.Body = p.parseBlockStatement()

	return af
}

// parsePipeExpression parses: expr |> func or expr |> func(args)
func (p *Parser) parsePipeExpression(left ast.Expression) ast.Expression {
	expr := &ast.PipeExpression{
		Token: p.curToken,
		Left:  left,
	}

	p.nextToken() // move past |>
	expr.Right = p.parseExpression(PIPE)

	return expr
}

// parseInExpression parses: value in collection (when not in for loop context)
func (p *Parser) parseInExpression(left ast.Expression) ast.Expression {
	expr := &ast.InExpression{
		Token: p.curToken,
		Left:  left,
	}

	p.nextToken()
	expr.Right = p.parseExpression(CONTAINS)

	return expr
}

// parseNilCoalesceExpression parses: expr ?? default
func (p *Parser) parseNilCoalesceExpression(left ast.Expression) ast.Expression {
	expr := &ast.NilCoalesceExpression{
		Token: p.curToken,
		Left:  left,
	}

	p.nextToken()
	expr.Right = p.parseExpression(NIL_COAL - 1) // right-associative

	return expr
}

// tryParseTypeParams attempts to parse generic type parameters: <T, U, V>
// Uses two-token lookahead to disambiguate without needing rollback.
// Peek must be LT, peekPeek must be IDENT for this to trigger.
// After parsing <IDENT, it checks if the next is , or > to confirm it's type params.
// Returns the list of type parameter names, or nil if not a type param list.
//
// IMPORTANT: This method is only safe to call when <T> is unambiguously type params
// (i.e., after function names or struct names, NOT after class names where < could mean inheritance).
// For class names, use inline disambiguation in parseClassLiteral.
func (p *Parser) tryParseTypeParams() []string {
	// We expect peek to be LT and peekPeek to be IDENT
	if !p.peekTokenIs(lexer.LT) || !p.peekPeekTokenIs(lexer.IDENT) {
		return nil
	}

	p.nextToken() // consume <
	p.nextToken() // move to first type param IDENT

	firstIdent := p.curToken.Literal

	// Check what follows the first IDENT
	if !p.peekTokenIs(lexer.GT) && !p.peekTokenIs(lexer.COMMA) {
		// Not type params — but we've consumed tokens!
		// This should only be called in contexts where <IDENT> is unambiguous
		// (functions and structs). If we get here, it's an error.
		p.addError(fmt.Sprintf("expected '>' or ',' in type parameters, got %s", p.peekToken.Type))
		return nil
	}

	typeParams := []string{firstIdent}

	// Parse additional type params: , IDENT
	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken() // consume comma
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		typeParams = append(typeParams, p.curToken.Literal)
	}

	// Must end with >
	if !p.expectPeek(lexer.GT) {
		return nil
	}

	return typeParams
}

// isCompoundAssign checks if a token type is a compound assignment operator.
func (p *Parser) isCompoundAssign(t lexer.TokenType) bool {
	switch t {
	case lexer.PLUS_ASSIGN, lexer.MINUS_ASSIGN, lexer.ASTERISK_ASSIGN, lexer.SLASH_ASSIGN, lexer.MODULO_ASSIGN:
		return true
	}
	return false
}

// compoundAssignOp returns the operator string for a compound assignment token.
func compoundAssignOp(t lexer.TokenType) string {
	switch t {
	case lexer.PLUS_ASSIGN:
		return "+"
	case lexer.MINUS_ASSIGN:
		return "-"
	case lexer.ASTERISK_ASSIGN:
		return "*"
	case lexer.SLASH_ASSIGN:
		return "/"
	case lexer.MODULO_ASSIGN:
		return "%"
	}
	return ""
}

// parseCompoundAssignment desugars x += 5 into x = x + 5
func (p *Parser) parseCompoundAssignment(ident *ast.Identifier) ast.Expression {
	p.nextToken() // move to the compound assignment operator
	op := compoundAssignOp(p.curToken.Type)
	assignToken := lexer.Token{Type: lexer.ASSIGN, Literal: "=", Line: p.curToken.Line, Column: p.curToken.Column}

	p.nextToken() // move to the value

	value := p.parseExpression(LOWEST)

	// Desugar: x += 5 -> x = x + 5
	return &ast.AssignmentExpression{
		Token: assignToken,
		Name:  ident,
		Value: &ast.InfixExpression{
			Token:    lexer.Token{Type: lexer.TokenType(op), Literal: op},
			Left:     &ast.Identifier{Token: ident.Token, Value: ident.Value},
			Operator: op,
			Right:    value,
		},
	}
}
