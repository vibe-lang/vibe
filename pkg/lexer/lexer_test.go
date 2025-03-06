package lexer

import (
	"testing"
)

func TestNextToken(t *testing.T) {
	input := `five = 5
ten = 10

def add_numbers(x: int, y, z: string): SomeStruct
  result = x + y
  SomeStruct(g: result)
end

result = add_numbers(five, ten, "hello")

# simple function without type annotations
def no_types(a, b)
  a + b
end

# function with return type but no parameter types
def calculate(x, y): int
  return x * y
end

# function with parameter types but no return type
def log(message: string, level: int)
  # just a function that logs something
  "Logged: " + message
end

# one-liner function
def square(n: int): int n * n end

!-/*5
5 < 10 > 5
if (5 < 10)
  return true
else
  return false
end
10 == 10
10 != 9
"hello world"
"hello \"world\""
`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{IDENT, "five"},
		{ASSIGN, "="},
		{INT, "5"},
		{IDENT, "ten"},
		{ASSIGN, "="},
		{INT, "10"},

		{FUNCTION, "def"},
		{IDENT, "add_numbers"},
		{LPAREN, "("},
		{IDENT, "x"},
		{COLON, ":"},
		{IDENT, "int"},
		{COMMA, ","},
		{IDENT, "y"},
		{COMMA, ","},
		{IDENT, "z"},
		{COLON, ":"},
		{IDENT, "string"},
		{RPAREN, ")"},
		{COLON, ":"},
		{IDENT, "SomeStruct"},
		{IDENT, "result"},
		{ASSIGN, "="},
		{IDENT, "x"},
		{PLUS, "+"},
		{IDENT, "y"},
		{IDENT, "SomeStruct"},
		{LPAREN, "("},
		{IDENT, "g"},
		{COLON, ":"},
		{IDENT, "result"},
		{RPAREN, ")"},
		{END, "end"},

		{IDENT, "result"},
		{ASSIGN, "="},
		{IDENT, "add_numbers"},
		{LPAREN, "("},
		{IDENT, "five"},
		{COMMA, ","},
		{IDENT, "ten"},
		{COMMA, ","},
		{STRING, "hello"},
		{RPAREN, ")"},

		// Simple function without type annotations
		{HASH, "#"},
		{FUNCTION, "def"},
		{IDENT, "no_types"},
		{LPAREN, "("},
		{IDENT, "a"},
		{COMMA, ","},
		{IDENT, "b"},
		{RPAREN, ")"},
		{IDENT, "a"},
		{PLUS, "+"},
		{IDENT, "b"},
		{END, "end"},

		// Function with return type but no parameter types
		{HASH, "#"},
		{FUNCTION, "def"},
		{IDENT, "calculate"},
		{LPAREN, "("},
		{IDENT, "x"},
		{COMMA, ","},
		{IDENT, "y"},
		{RPAREN, ")"},
		{COLON, ":"},
		{IDENT, "int"},
		{RETURN, "return"},
		{IDENT, "x"},
		{ASTERISK, "*"},
		{IDENT, "y"},
		{END, "end"},

		// Function with parameter types but no return type
		{HASH, "#"},
		{FUNCTION, "def"},
		{IDENT, "log"},
		{LPAREN, "("},
		{IDENT, "message"},
		{COLON, ":"},
		{IDENT, "string"},
		{COMMA, ","},
		{IDENT, "level"},
		{COLON, ":"},
		{IDENT, "int"},
		{RPAREN, ")"},
		{HASH, "#"},
		{STRING, "Logged: "},
		{PLUS, "+"},
		{IDENT, "message"},
		{END, "end"},

		// One-liner function
		{HASH, "#"},
		{FUNCTION, "def"},
		{IDENT, "square"},
		{LPAREN, "("},
		{IDENT, "n"},
		{COLON, ":"},
		{IDENT, "int"},
		{RPAREN, ")"},
		{COLON, ":"},
		{IDENT, "int"},
		{IDENT, "n"},
		{ASTERISK, "*"},
		{IDENT, "n"},
		{END, "end"},

		{BANG, "!"},
		{MINUS, "-"},
		{SLASH, "/"},
		{ASTERISK, "*"},
		{INT, "5"},
		{INT, "5"},
		{LT, "<"},
		{INT, "10"},
		{GT, ">"},
		{INT, "5"},
		{IF, "if"},
		{LPAREN, "("},
		{INT, "5"},
		{LT, "<"},
		{INT, "10"},
		{RPAREN, ")"},
		{RETURN, "return"},
		{TRUE, "true"},
		{ELSE, "else"},
		{RETURN, "return"},
		{FALSE, "false"},
		{END, "end"},
		{INT, "10"},
		{EQ, "=="},
		{INT, "10"},
		{INT, "10"},
		{NOT_EQ, "!="},
		{INT, "9"},
		{STRING, "hello world"},
		{STRING, "hello \\\"world\\\""},
		{EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestComment(t *testing.T) {
	input := `# This is a comment
x = 5 # This is another comment`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{HASH, "#"},
		{IDENT, "x"},
		{ASSIGN, "="},
		{INT, "5"},
		{HASH, "#"},
		{EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}