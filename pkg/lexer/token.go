package lexer

// TokenType represents the type of a token in the Vibe language.
type TokenType string

// Token represents a lexical token in the Vibe language.
type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
	// NLBefore reports whether a line break separates this token from the previous one.
	NLBefore bool
	// SpaceBefore reports whether whitespace, a comment or a line break precedes this token.
	SpaceBefore bool
	// Extra carries auxiliary data: regex flags, or the kind ("w" or "i") of a word list.
	Extra string
}

const (
	// Special tokens
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"

	// Identifiers and literals
	IDENT   = "IDENT"
	INT     = "INT"
	FLOAT   = "FLOAT"
	STRING  = "STRING"
	BOOLEAN = "BOOLEAN"

	// Operators
	ASSIGN   = "="
	PLUS     = "+"
	MINUS    = "-"
	BANG     = "!"
	ASTERISK = "*"
	SLASH    = "/"
	MODULO   = "%"
	AND      = "&&"
	OR       = "||"
	PIPE     = "|"

	// Compound assignment operators
	PLUS_ASSIGN     = "+="
	MINUS_ASSIGN    = "-="
	ASTERISK_ASSIGN = "*="
	SLASH_ASSIGN    = "/="
	MODULO_ASSIGN   = "%="

	// Arrow and pipe operators
	ARROW      = "->"
	PIPE_ARROW = "|>"

	// Ternary and nil coalescing operators
	QUESTION     = "?"
	NIL_COALESCE = "??"

	// Comparison operators
	EQ     = "=="
	NOT_EQ = "!="
	LT     = "<"
	GT     = ">"
	LTE    = "<="
	GTE    = ">="

	// Range operators
	DOTDOT    = ".."
	DOTDOTDOT = "..."

	// Delimiters
	COMMA     = ","
	SEMICOLON = ";"
	COLON     = ":"
	DOT       = "."
	LPAREN    = "("
	RPAREN    = ")"
	LBRACE    = "{"
	RBRACE    = "}"
	LBRACKET  = "["
	RBRACKET  = "]"
	HASH      = "#"

	// Keywords
	FUNCTION = "FUNCTION"
	TRUE     = "TRUE"
	FALSE    = "FALSE"
	IF       = "IF"
	ELSE     = "ELSE"
	ELSIF    = "ELSIF"
	RETURN   = "RETURN"
	CLASS    = "CLASS"
	PROP     = "PROP"
	NIL      = "NIL"
	STRUCT   = "STRUCT"
	END      = "END"
	LET      = "LET"
	DEF      = "DEF"
	FOR      = "FOR"
	IN       = "IN"
	WHILE    = "WHILE"
	BREAK    = "BREAK"
	CONTINUE = "CONTINUE"
	IMPORT   = "IMPORT"
	TRY      = "TRY"
	CATCH    = "CATCH"
	THROW    = "THROW"
	UNLESS   = "UNLESS"
	UNTIL    = "UNTIL"
	CASE     = "CASE"
	WHEN     = "WHEN"
	FINALLY  = "FINALLY"
	SELF     = "SELF"
	SUPER    = "SUPER"
	ENUM     = "ENUM"
	CONST    = "CONST"

	// Power operator
	POWER_OP = "**"

	// Bitwise operators
	BITAND = "&"
	BITXOR = "^"
	BITNOT = "~"
	LSHIFT = "<<"
	RSHIFT = ">>"

	// Other operators
	SPACESHIP    = "<=>"
	FAT_ARROW    = "=>"
	SAFE_NAV     = "&."
	DOUBLE_COLON = "::"
	MATCH        = "=~"
	NOT_MATCH    = "!~"

	// More compound assignment operators
	POWER_ASSIGN  = "**="
	OR_ASSIGN     = "||="
	AND_ASSIGN    = "&&="
	NIL_ASSIGN    = "??="
	BITAND_ASSIGN = "&="
	BITOR_ASSIGN  = "|="
	BITXOR_ASSIGN = "^="
	LSHIFT_ASSIGN = "<<="
	RSHIFT_ASSIGN = ">>="

	// Variable and literal tokens
	IVAR   = "IVAR"   // @name
	CVAR   = "CVAR"   // @@name
	GVAR   = "GVAR"   // $name
	SYMBOL = "SYMBOL" // :name
	REGEX  = "REGEX"  // /pattern/flags
	WORDS  = "WORDS"  // %w[a b] / %i[a b]

	// More keywords
	DO     = "DO"
	THEN   = "THEN"
	NEXT   = "NEXT"
	BEGIN  = "BEGIN"
	RESCUE = "RESCUE"
	ENSURE = "ENSURE"
	MODULE = "MODULE"
	YIELD  = "YIELD"
	KW_AND = "KW_AND"
	KW_OR  = "KW_OR"
	KW_NOT = "KW_NOT"
	ALIAS  = "ALIAS"
)

var keywords = map[string]TokenType{
	"fn":       FUNCTION,
	"def":      DEF,
	"let":      LET,
	"true":     TRUE,
	"false":    FALSE,
	"if":       IF,
	"else":     ELSE,
	"elsif":    ELSIF,
	"return":   RETURN,
	"class":    CLASS,
	"nil":      NIL,
	"end":      END,
	"struct":   STRUCT,
	"prop":     PROP,
	"Range":    IDENT,
	"for":      FOR,
	"in":       IN,
	"while":    WHILE,
	"break":    BREAK,
	"continue": CONTINUE,
	"import":   IMPORT,
	"try":      TRY,
	"catch":    CATCH,
	"throw":    THROW,
	"unless":   UNLESS,
	"until":    UNTIL,
	"case":     CASE,
	"when":     WHEN,
	"finally":  FINALLY,
	"self":     SELF,
	"super":    SUPER,
	"enum":     ENUM,
	"const":    CONST,
	"do":       DO,
	"then":     THEN,
	"next":     NEXT,
	"begin":    BEGIN,
	"rescue":   RESCUE,
	"ensure":   ENSURE,
	"module":   MODULE,
	"yield":    YIELD,
	"and":      KW_AND,
	"or":       KW_OR,
	"not":      KW_NOT,
	"alias":    ALIAS,
}

// LookupIdent checks if the given identifier is a keyword.
func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
