package lexer

import (
	"unicode"
	"unicode/utf8"
)

// Lexer represents a lexical analyzer for the Vibe programming language.
// It transforms source code (as a string) into a stream of tokens that
// the parser can then process. The lexer is the first phase of the
// interpretation process.
//
// The lexer handles:
// - Breaking up the input into meaningful tokens
// - Categorizing these tokens (identifiers, keywords, literals, etc.)
// - Tracking line and column information for error reporting
// - Ignoring whitespace and comments where appropriate
type Lexer struct {
	input        string // the source code being analyzed
	position     int    // current position in input (points to current char)
	readPosition int    // current reading position in input (after current char)
	ch           rune   // current character under examination
	line         int    // current line number for error reporting
	column       int    // current column number for error reporting
}

// New creates a new Lexer for the provided input string.
// It initializes the lexer to start reading from the beginning of the input
// and sets up line/column tracking for error reporting.
//
// Example:
//
//	l := lexer.New("let x = 5;")
//	// l is now ready to tokenize the input string
func New(input string) *Lexer {
	l := &Lexer{input: input, line: 1, column: 0}
	l.readChar() // Initialize by reading the first character
	return l
}

// readChar advances the lexer to the next character in the input.
// This is a fundamental operation that moves the lexer forward in the input stream.
// It handles UTF-8 encoded input by properly decoding runes.
//
// This method:
// - Updates position and readPosition
// - Decodes the next UTF-8 character
// - Handles the end of input by setting ch to 0 (EOF)
// - Tracks column position for error reporting
func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0 // EOF
	} else {
		r, size := utf8.DecodeRuneInString(l.input[l.readPosition:])
		l.ch = r
		l.position = l.readPosition
		l.readPosition += size
	}
	l.column++
}

// peekChar returns the next character in the input without advancing the lexer.
// This allows the lexer to look ahead one character to make decisions about
// multi-character tokens like "==" or "!=".
//
// Returns 0 (EOF) if at the end of input.
func (l *Lexer) peekChar() rune {
	if l.readPosition >= len(l.input) {
		return 0 // EOF
	}
	r, _ := utf8.DecodeRuneInString(l.input[l.readPosition:])
	return r
}

// NextToken returns the next token from the input.
// This is the main method called by the parser to get tokens one at a time.
// It determines what type of token is next in the input stream based on
// the current character and context.
//
// The method processes:
// - Operators (both single-character and multi-character)
// - Delimiters
// - Identifiers and keywords
// - Numbers (integers and floats)
// - Strings
// - Comments
// - Whitespace (which is skipped)
//
// Example:
//
//	l := lexer.New("let x = 5;")
//	token := l.NextToken() // token will be {Type: LET, Literal: "let", ...}
func (l *Lexer) NextToken() Token {
	var tok Token

	l.skipWhitespace()

	// Set the starting position for the token
	tok.Line = l.line
	tok.Column = l.column

	switch l.ch {
	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: EQ, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column - 1}
		} else {
			tok = Token{Type: ASSIGN, Literal: string(l.ch), Line: l.line, Column: l.column}
		}
	case '+':
		tok = Token{Type: PLUS, Literal: string(l.ch), Line: l.line, Column: l.column}
	case '-':
		tok = Token{Type: MINUS, Literal: string(l.ch), Line: l.line, Column: l.column}
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: NOT_EQ, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column - 1}
		} else {
			tok = Token{Type: BANG, Literal: string(l.ch), Line: l.line, Column: l.column}
		}
	case '*':
		tok = Token{Type: ASTERISK, Literal: string(l.ch), Line: l.line, Column: l.column}
	case '/':
		if l.peekChar() == '/' {
			// Comment, skip until end of line
			l.skipComment()
			return l.NextToken()
		} else {
			tok = Token{Type: SLASH, Literal: string(l.ch), Line: l.line, Column: l.column}
		}
	case '%':
		tok = Token{Type: MODULO, Literal: string(l.ch), Line: l.line, Column: l.column}
	case '<':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: LTE, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column - 1}
		} else {
			tok = Token{Type: LT, Literal: string(l.ch), Line: l.line, Column: l.column}
		}
	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: GTE, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column - 1}
		} else {
			tok = Token{Type: GT, Literal: string(l.ch), Line: l.line, Column: l.column}
		}
	case ';':
		tok = Token{Type: SEMICOLON, Literal: string(l.ch), Line: l.line, Column: l.column}
	case ':':
		tok = Token{Type: COLON, Literal: string(l.ch), Line: l.line, Column: l.column}
	case ',':
		tok = Token{Type: COMMA, Literal: string(l.ch), Line: l.line, Column: l.column}
	case '.':
		if l.peekChar() == '.' {
			ch := l.ch
			l.readChar() // consume the first '.'
			if l.peekChar() == '.' {
				// This is a '...' (exclusive range)
				ch2 := l.ch
				l.readChar() // consume the second '.'
				tok = Token{Type: DOTDOTDOT, Literal: string(ch) + string(ch2) + string(l.ch), Line: l.line, Column: l.column - 2}
			} else {
				// This is a '..' (inclusive range)
				tok = Token{Type: DOTDOT, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column - 1}
			}
			l.readChar() // Move to the next character after the range operator
			return tok
		} else {
			tok = Token{Type: DOT, Literal: string(l.ch), Line: l.line, Column: l.column}
		}
	case '(':
		tok = Token{Type: LPAREN, Literal: string(l.ch), Line: l.line, Column: l.column}
	case ')':
		tok = Token{Type: RPAREN, Literal: string(l.ch), Line: l.line, Column: l.column}
	case '{':
		tok = Token{Type: LBRACE, Literal: string(l.ch), Line: l.line, Column: l.column}
	case '}':
		tok = Token{Type: RBRACE, Literal: string(l.ch), Line: l.line, Column: l.column}
	case '[':
		tok = Token{Type: LBRACKET, Literal: string(l.ch), Line: l.line, Column: l.column}
	case ']':
		tok = Token{Type: RBRACKET, Literal: string(l.ch), Line: l.line, Column: l.column}
	case '#':
		tok = Token{Type: HASH, Literal: string(l.ch), Line: l.line, Column: l.column}
		// Skip the rest of the line after a comment character
		l.skipComment()
	case '"':
		tok.Type = STRING
		tok.Literal = l.readString()
		tok.Line = l.line
		tok.Column = l.column
	case '\'':
		tok.Type = STRING
		tok.Literal = l.readString()
		tok.Line = l.line
		tok.Column = l.column
	case 0:
		tok.Type = EOF
		tok.Literal = ""
		tok.Line = l.line
		tok.Column = l.column
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = LookupIdent(tok.Literal)
			tok.Line = l.line
			tok.Column = l.column
			return tok
		} else if isDigit(l.ch) {
			return l.readNumber()
		} else {
			tok = Token{Type: ILLEGAL, Literal: string(l.ch), Line: l.line, Column: l.column}
		}
	}

	l.readChar()
	return tok
}

// skipWhitespace skips any whitespace characters in the input.
// This includes spaces, tabs, newlines, and carriage returns.
// Line and column tracking is maintained properly when skipping whitespace.
func (l *Lexer) skipWhitespace() {
	for unicode.IsSpace(l.ch) {
		if l.ch == '\n' {
			l.line++
			l.column = 0
		}
		l.readChar()
	}
}

// skipComment skips a line comment (// ...).
// Comments in Vibe are single-line comments starting with // and ending at the newline.
func (l *Lexer) skipComment() {
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
}

// readIdentifier reads an identifier or keyword from the input.
// Identifiers in Vibe start with a letter or underscore and can contain
// letters, digits, and underscores.
func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

// readNumber reads a numeric literal from the input.
// It handles both integers and floating-point numbers.
// A number is recognized as a floating-point if it contains a decimal point
// followed by at least one digit.
func (l *Lexer) readNumber() Token {
	position := l.position
	isFloat := false

	for isDigit(l.ch) {
		l.readChar()
		// Check for decimal point
		if l.ch == '.' && !isFloat && isDigit(l.peekChar()) {
			isFloat = true
			l.readChar()
		}
	}

	literal := l.input[position:l.position]
	if isFloat {
		return Token{Type: FLOAT, Literal: literal, Line: l.line, Column: l.column}
	}
	return Token{Type: INT, Literal: literal, Line: l.line, Column: l.column}
}

// readString reads a string literal from the input.
// String literals are enclosed in either single quotes (') or double quotes (").
// The function handles escape sequences for the closing quote.
func (l *Lexer) readString() string {
	// Store the quote character (either " or ')
	quote := l.ch
	l.readChar() // Skip the opening quote
	position := l.position

	for l.ch != quote && l.ch != 0 {
		// Handle escape sequences
		if l.ch == '\\' && l.peekChar() == quote {
			l.readChar() // Skip the backslash
		}
		l.readChar()
	}

	if l.ch == 0 {
		// Unterminated string
		return l.input[position:l.position]
	}

	str := l.input[position:l.position]
	return str
}

// isLetter returns true if the character is a letter or underscore.
// This defines what characters can start and be part of an identifier.
func isLetter(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_'
}

// isDigit returns true if the character is a digit.
// This defines what characters can be part of a numeric literal.
func isDigit(ch rune) bool {
	return unicode.IsDigit(ch)
}