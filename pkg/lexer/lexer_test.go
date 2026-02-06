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
		{IDENT, "x"},
		{ASSIGN, "="},
		{INT, "5"},
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

// ---------------------------------------------------------------------------
// Phase 4a: Additional lexer token tests
// ---------------------------------------------------------------------------

func TestFloatTokens(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"3.14", "3.14"},
		{"0.5", "0.5"},
		{"100.0", "100.0"},
	}

	for _, tt := range tests {
		l := New(tt.input)
		tok := l.NextToken()

		if tok.Type != FLOAT {
			t.Errorf("input %q: expected FLOAT, got %s", tt.input, tok.Type)
		}
		if tok.Literal != tt.expected {
			t.Errorf("input %q: expected literal %q, got %q", tt.input, tt.expected, tok.Literal)
		}
	}
}

func TestComparisonOperators(t *testing.T) {
	input := `<= >= < >`

	expected := []struct {
		typ TokenType
		lit string
	}{
		{LTE, "<="},
		{GTE, ">="},
		{LT, "<"},
		{GT, ">"},
		{EOF, ""},
	}

	l := New(input)
	for i, exp := range expected {
		tok := l.NextToken()
		if tok.Type != exp.typ {
			t.Fatalf("tests[%d] - type wrong. expected=%q, got=%q", i, exp.typ, tok.Type)
		}
		if tok.Literal != exp.lit {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q", i, exp.lit, tok.Literal)
		}
	}
}

func TestModuloOperator(t *testing.T) {
	l := New("10 % 3")
	tok := l.NextToken() // 10
	if tok.Type != INT || tok.Literal != "10" {
		t.Fatalf("expected INT(10), got %s(%s)", tok.Type, tok.Literal)
	}
	tok = l.NextToken() // %
	if tok.Type != MODULO || tok.Literal != "%" {
		t.Fatalf("expected MODULO(%%), got %s(%s)", tok.Type, tok.Literal)
	}
	tok = l.NextToken() // 3
	if tok.Type != INT || tok.Literal != "3" {
		t.Fatalf("expected INT(3), got %s(%s)", tok.Type, tok.Literal)
	}
}

func TestSingleQuoteStrings(t *testing.T) {
	l := New(`'hello world'`)
	tok := l.NextToken()

	if tok.Type != STRING {
		t.Fatalf("expected STRING, got %s", tok.Type)
	}
	if tok.Literal != "hello world" {
		t.Fatalf("expected literal %q, got %q", "hello world", tok.Literal)
	}
}

func TestDoubleSlashComments(t *testing.T) {
	input := `x = 5 // this is a comment
y = 10`

	expected := []struct {
		typ TokenType
		lit string
	}{
		{IDENT, "x"},
		{ASSIGN, "="},
		{INT, "5"},
		{IDENT, "y"},
		{ASSIGN, "="},
		{INT, "10"},
		{EOF, ""},
	}

	l := New(input)
	for i, exp := range expected {
		tok := l.NextToken()
		if tok.Type != exp.typ {
			t.Fatalf("tests[%d] - type wrong. expected=%q, got=%q", i, exp.typ, tok.Type)
		}
		if tok.Literal != exp.lit {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q", i, exp.lit, tok.Literal)
		}
	}
}

func TestIllegalToken(t *testing.T) {
	l := New("@")
	tok := l.NextToken()

	if tok.Type != ILLEGAL {
		t.Fatalf("expected ILLEGAL, got %s", tok.Type)
	}
}

func TestAllKeywords(t *testing.T) {
	tests := []struct {
		input    string
		expected TokenType
	}{
		{"fn", FUNCTION},
		{"def", DEF},
		{"let", LET},
		{"true", TRUE},
		{"false", FALSE},
		{"if", IF},
		{"else", ELSE},
		{"elsif", ELSIF},
		{"return", RETURN},
		{"class", CLASS},
		{"nil", NIL},
		{"end", END},
		{"struct", STRUCT},
		{"prop", PROP},
		{"for", FOR},
		{"in", IN},
	}

	for _, tt := range tests {
		l := New(tt.input)
		tok := l.NextToken()

		if tok.Type != tt.expected {
			t.Errorf("keyword %q: expected type %s, got %s", tt.input, tt.expected, tok.Type)
		}
		if tok.Literal != tt.input {
			t.Errorf("keyword %q: expected literal %q, got %q", tt.input, tt.input, tok.Literal)
		}
	}
}

func TestLineColumnTracking(t *testing.T) {
	input := "x = 5\ny = 10"

	l := New(input)

	// x on line 1
	tok := l.NextToken()
	if tok.Line != 1 {
		t.Errorf("x: expected line 1, got %d", tok.Line)
	}

	l.NextToken() // =
	l.NextToken() // 5

	// y on line 2
	tok = l.NextToken()
	if tok.Literal != "y" {
		t.Fatalf("expected y, got %s", tok.Literal)
	}
	if tok.Line != 2 {
		t.Errorf("y: expected line 2, got %d", tok.Line)
	}
}

func TestDotOperator(t *testing.T) {
	l := New("a.b")

	tok := l.NextToken()
	if tok.Type != IDENT || tok.Literal != "a" {
		t.Fatalf("expected IDENT(a), got %s(%s)", tok.Type, tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != DOT || tok.Literal != "." {
		t.Fatalf("expected DOT(.), got %s(%s)", tok.Type, tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != IDENT || tok.Literal != "b" {
		t.Fatalf("expected IDENT(b), got %s(%s)", tok.Type, tok.Literal)
	}
}

func TestSemicolonToken(t *testing.T) {
	l := New("x = 5;")
	l.NextToken() // x
	l.NextToken() // =
	l.NextToken() // 5

	tok := l.NextToken()
	if tok.Type != SEMICOLON || tok.Literal != ";" {
		t.Fatalf("expected SEMICOLON, got %s(%s)", tok.Type, tok.Literal)
	}
}

func TestBraceTokens(t *testing.T) {
	l := New("{ }")

	tok := l.NextToken()
	if tok.Type != LBRACE {
		t.Fatalf("expected LBRACE, got %s", tok.Type)
	}

	tok = l.NextToken()
	if tok.Type != RBRACE {
		t.Fatalf("expected RBRACE, got %s", tok.Type)
	}
}
