package lexer

import (
	"testing"
)

func TestNextToken(t *testing.T) {
	input := `five = 5;
ten = 10;
add = def add_numbers(x: int, y, z: string): SomeStruct
  result = x + y;
  SomeStruct(g: result)
end;
result = add_numbers(five, ten, "hello");
!-/*5;
5 < 10 > 5;
if (5 < 10) {
  return true;
} else {
  return false;
}
10 == 10;
10 != 9;
"hello world";
"hello \"world\"";
`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{IDENT, "five"},
		{ASSIGN, "="},
		{INT, "5"},
		{SEMICOLON, ";"},
		{IDENT, "ten"},
		{ASSIGN, "="},
		{INT, "10"},
		{SEMICOLON, ";"},
		{IDENT, "add"},
		{ASSIGN, "="},
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
		{SEMICOLON, ";"},
		{IDENT, "SomeStruct"},
		{LPAREN, "("},
		{IDENT, "g"},
		{COLON, ":"},
		{IDENT, "result"},
		{RPAREN, ")"},
		{END, "end"},
		{SEMICOLON, ";"},
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
		{SEMICOLON, ";"},

		// !-/*5;
		{BANG, "!"},
		{MINUS, "-"},
		{SLASH, "/"},
		{ASTERISK, "*"},
		{INT, "5"},
		{SEMICOLON, ";"},

		// 5 < 10 > 5;
		{INT, "5"},
		{LT, "<"},
		{INT, "10"},
		{GT, ">"},
		{INT, "5"},
		{SEMICOLON, ";"},

		// if (5 < 10) {
		{IF, "if"},
		{LPAREN, "("},
		{INT, "5"},
		{LT, "<"},
		{INT, "10"},
		{RPAREN, ")"},
		{LBRACE, "{"},

		// return true;
		{RETURN, "return"},
		{TRUE, "true"},
		{SEMICOLON, ";"},

		// } else {
		{RBRACE, "}"},
		{ELSE, "else"},
		{LBRACE, "{"},

		// return false;
		{RETURN, "return"},
		{FALSE, "false"},
		{SEMICOLON, ";"},

		// }
		{RBRACE, "}"},

		// 10 == 10;
		{INT, "10"},
		{EQ, "=="},
		{INT, "10"},
		{SEMICOLON, ";"},

		// 10 != 9;
		{INT, "10"},
		{NOT_EQ, "!="},
		{INT, "9"},
		{SEMICOLON, ";"},

		// "hello world";
		{STRING, "hello world"},
		{SEMICOLON, ";"},

		// "hello \"world\"";
		{STRING, "hello \\\"world\\\""},
		{SEMICOLON, ";"},

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
x = 5; # This is another comment`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{HASH, "#"},
		{IDENT, "x"},
		{ASSIGN, "="},
		{INT, "5"},
		{SEMICOLON, ";"},
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