package lexer

// TokenType represents the type of a token in the Vibe language.
// The lexer categorizes code into these token types which the parser
// then uses to build the abstract syntax tree.
type TokenType string

// Token represents a lexical token in the Vibe language.
// Each token has a type, a literal value, and position information
// for error reporting. The position information (Line, Column) helps
// provide meaningful error messages to the programmer.
type Token struct {
	Type    TokenType // The category of this token
	Literal string    // The actual text from the source code
	Line    int       // Line number where the token appears (1-indexed)
	Column  int       // Column position where the token starts (1-indexed)
}

// Token types define all possible token categories in the Vibe language.
// These constants are used by the lexer to categorize input and by the parser
// to make decisions about the structure of the code.
const (
	// Special tokens
	ILLEGAL = "ILLEGAL" // Token for characters that don't belong
	EOF     = "EOF"     // End of file marker

	// Identifiers and literals
	IDENT     = "IDENT"     // variable names, function names, etc.
	INT       = "INT"       // integers like 123
	FLOAT     = "FLOAT"     // floating point numbers like 123.45
	STRING    = "STRING"    // string literals like "hello" or 'world'
	BOOLEAN   = "BOOLEAN"   // true or false

	// Operators
	ASSIGN   = "="  // Assignment operator
	PLUS     = "+"  // Addition operator
	MINUS    = "-"  // Subtraction operator
	BANG     = "!"  // Logical negation operator
	ASTERISK = "*"  // Multiplication operator
	SLASH    = "/"  // Division operator
	MODULO   = "%"  // Modulo operator

	// Comparison operators
	EQ     = "=="  // Equality operator
	NOT_EQ = "!="  // Inequality operator
	LT     = "<"   // Less than operator
	GT     = ">"   // Greater than operator
	LTE    = "<="  // Less than or equal operator
	GTE    = ">="  // Greater than or equal operator

	// Range operators
	DOTDOT   = ".."   // Inclusive range operator (a..b)
	DOTDOTDOT = "..." // Exclusive range operator (a...b)

	// Delimiters
	COMMA     = ","   // Separates items in lists
	SEMICOLON = ";"   // Terminates statements
	COLON     = ":"   // Used for type annotations and key-value pairs
	DOT       = "."   // Object property access
	LPAREN    = "("   // Start of parameter lists or expressions
	RPAREN    = ")"   // End of parameter lists or expressions
	LBRACE    = "{"   // Start of blocks
	RBRACE    = "}"   // End of blocks
	LBRACKET  = "["   // Start of array literals
	RBRACKET  = "]"   // End of array literals
	HASH      = "#"   // Used for string interpolation and comments

	// Keywords
	FUNCTION = "FUNCTION" // 'func' keyword for function definitions
	TRUE     = "TRUE"     // 'true' boolean literal
	FALSE    = "FALSE"    // 'false' boolean literal
	IF       = "IF"       // 'if' conditional statement
	ELSE     = "ELSE"     // 'else' alternative in if statement
	ELSIF    = "ELSIF"    // 'elsif' alternative with condition
	RETURN   = "RETURN"   // 'return' statement in functions
	CLASS    = "CLASS"    // 'class' keyword for class definitions
	PROP     = "PROP"     // 'prop' keyword for class property declarations
	NIL      = "NIL"      // 'nil' value (similar to null in other languages)
	STRUCT   = "STRUCT"   // 'struct' keyword for struct definitions
	END      = "END"      // 'end' keyword to close blocks like struct definitions
	LET      = "LET"      // 'let' keyword for variable declarations
	DEF      = "DEF"      // 'def' keyword for function definitions
)

// keywords maps string keywords to their token types
var keywords = map[string]TokenType{
	"fn":      FUNCTION,
	"def":     DEF,
	"let":     LET,
	"true":    TRUE,
	"false":   FALSE,
	"if":      IF,
	"else":    ELSE,
	"elsif":   ELSIF,
	"return":  RETURN,
	"class":   CLASS,
	"nil":     NIL,
	"end":     END,
	"struct":  STRUCT,
	"prop":    PROP,
	"Range":   IDENT, // Range constructor is treated as an identifier
}

// LookupIdent checks if the given identifier is a keyword.
// If it is a keyword, returns the keyword's TokenType, otherwise returns IDENT.
// This function is called by the lexer after reading an identifier to determine
// its proper categorization.
//
// Example:
//   - LookupIdent("func") returns FUNCTION
//   - LookupIdent("myVariable") returns IDENT
func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}