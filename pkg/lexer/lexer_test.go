package lexer

import (
	"fmt"
	"testing"
)

func TestNextToken(t *testing.T) {
	input := `five = 5
ten = 10

def add(x: int, y: int): int
    x + y
end

result = add(five, ten)
!-/*5
5 < 10 > 5
if (5 < 10)
    return true
else
    return false
end

10 == 10
10 != 9
"hello \"world\""
"Apple"
"hello \\\"world\\\""
[1, 2]
"Hello, ${name}!"
1..5
10...20
nil
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

		{DEF, "def"},
		{IDENT, "add"},
		{LPAREN, "("},
		{IDENT, "x"},
		{COLON, ":"},
		{IDENT, "int"},
		{COMMA, ","},
		{IDENT, "y"},
		{COLON, ":"},
		{IDENT, "int"},
		{RPAREN, ")"},
		{COLON, ":"},
		{IDENT, "int"},
		{IDENT, "x"},
		{PLUS, "+"},
		{IDENT, "y"},
		{END, "end"},

		{IDENT, "result"},
		{ASSIGN, "="},
		{IDENT, "add"},
		{LPAREN, "("},
		{IDENT, "five"},
		{COMMA, ","},
		{IDENT, "ten"},
		{RPAREN, ")"},

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

		{STRING, "hello \"world\""},
		{STRING, "Apple"},
		{STRING, "hello \\\"world\\\""},

		{LBRACKET, "["},
		{INT, "1"},
		{COMMA, ","},
		{INT, "2"},
		{RBRACKET, "]"},

		{STRING, "Hello, ${name}!"},

		{INT, "1"},
		{DOTDOT, ".."},
		{INT, "5"},

		{INT, "10"},
		{DOTDOTDOT, "..."},
		{INT, "20"},

		{NIL, "nil"},
		{EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		// For debugging
		fmt.Printf("Token %d: expected=%q(%q), got=%q(%q)\n",
			i, tt.expectedType, tt.expectedLiteral, tok.Type, tok.Literal)

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

func TestRangeOperators(t *testing.T) {
	input := `
a = 1..5
b = 10...20
c = start..finish
d = Range(1, 10)
`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{IDENT, "a"},
		{ASSIGN, "="},
		{INT, "1"},
		{DOTDOT, ".."},
		{INT, "5"},

		{IDENT, "b"},
		{ASSIGN, "="},
		{INT, "10"},
		{DOTDOTDOT, "..."},
		{INT, "20"},

		{IDENT, "c"},
		{ASSIGN, "="},
		{IDENT, "start"},
		{DOTDOT, ".."},
		{IDENT, "finish"},

		{IDENT, "d"},
		{ASSIGN, "="},
		{IDENT, "Range"},
		{LPAREN, "("},
		{INT, "1"},
		{COMMA, ","},
		{INT, "10"},
		{RPAREN, ")"},

		{EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		// Add debugging
		t.Logf("Token %d: expected=%q(%q), got=%q(%q)",
			i, tt.expectedType, tt.expectedLiteral, tok.Type, tok.Literal)

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