package lexer

import (
	"bytes"
	"strconv"
	"strings"
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
// - Recording whether a line break precedes each token (Token.NLBefore)
// - Ignoring whitespace and comments where appropriate
type Lexer struct {
	input        string // the source code being analyzed
	position     int    // current position in input (points to current char)
	readPosition int    // current reading position in input (after current char)
	ch           rune   // current character under examination
	line         int    // current line number for error reporting
	column       int    // current column number for error reporting

	prevType    TokenType // type of the previously emitted token
	sawNewline  bool      // a newline was skipped since the last token
	sawSpace    bool      // whitespace or a comment was skipped since the last token
	heredocSkip map[int]int
	heredocNext int // where the next heredoc body on the current line starts (0 = none)
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
// It handles UTF-8 encoded input by properly decoding runes, tracks line
// and column positions, and jumps over heredoc bodies that were already consumed.
func (l *Lexer) readChar() {
	if l.ch == '\n' {
		l.line++
		l.column = 0
		l.heredocNext = 0
		if skip, ok := l.heredocSkip[l.readPosition]; ok {
			// l.readPosition is the first byte after the newline; jump past the body.
			l.line += strings.Count(l.input[l.readPosition:skip], "\n")
			delete(l.heredocSkip, l.readPosition)
			l.readPosition = skip
		}
	}
	if l.readPosition >= len(l.input) {
		l.ch = 0 // EOF
		l.position = l.readPosition
	} else {
		r, size := utf8.DecodeRuneInString(l.input[l.readPosition:])
		l.ch = r
		l.position = l.readPosition
		l.readPosition += size
	}
	l.column++
}

// peekChar returns the next character in the input without advancing the lexer.
// Returns 0 (EOF) if at the end of input.
func (l *Lexer) peekChar() rune {
	if l.readPosition >= len(l.input) {
		return 0 // EOF
	}
	r, _ := utf8.DecodeRuneInString(l.input[l.readPosition:])
	return r
}

// peekAt returns the rune n runes after the current one (peekAt(1) == peekChar()).
func (l *Lexer) peekAt(n int) rune {
	pos := l.position
	var r rune
	for k := 0; k <= n; k++ {
		if pos >= len(l.input) {
			return 0
		}
		var size int
		r, size = utf8.DecodeRuneInString(l.input[pos:])
		pos += size
	}
	return r
}

// prevChar returns the character just before the current one, or 0 at the start.
func (l *Lexer) prevChar() rune {
	if l.position == 0 {
		return 0
	}
	r, _ := utf8.DecodeLastRuneInString(l.input[:l.position])
	return r
}

// operators lists multi-character operators, longest first.
var operators = []struct {
	text string
	typ  TokenType
}{
	{"<=>", SPACESHIP}, {"**=", POWER_ASSIGN}, {"||=", OR_ASSIGN}, {"&&=", AND_ASSIGN},
	{"??=", NIL_ASSIGN}, {"<<=", LSHIFT_ASSIGN}, {">>=", RSHIFT_ASSIGN}, {"...", DOTDOTDOT},
	{"&.", SAFE_NAV}, {"::", DOUBLE_COLON}, {"&&", AND}, {"||", OR}, {"|>", PIPE_ARROW}, {"<<", LSHIFT}, {">>", RSHIFT},
	{"<=", LTE}, {">=", GTE}, {"==", EQ}, {"!=", NOT_EQ}, {"=>", FAT_ARROW}, {"=~", MATCH},
	{"!~", NOT_MATCH}, {"->", ARROW}, {"+=", PLUS_ASSIGN}, {"-=", MINUS_ASSIGN}, {"*=", ASTERISK_ASSIGN},
	{"/=", SLASH_ASSIGN}, {"%=", MODULO_ASSIGN}, {"&=", BITAND_ASSIGN}, {"|=", BITOR_ASSIGN},
	{"^=", BITXOR_ASSIGN}, {"**", POWER_OP}, {"??", NIL_COALESCE}, {"..", DOTDOT},
}

var singleOps = map[rune]TokenType{
	'=': ASSIGN, '+': PLUS, '-': MINUS, '!': BANG, '*': ASTERISK, '/': SLASH, '%': MODULO,
	'|': PIPE, '&': BITAND, '^': BITXOR, '~': BITNOT, '?': QUESTION, '<': LT, '>': GT,
	';': SEMICOLON, ':': COLON, ',': COMMA, '.': DOT, '(': LPAREN, ')': RPAREN,
	'{': LBRACE, '}': RBRACE, '[': LBRACKET, ']': RBRACKET,
}

// NextToken returns the next token from the input.
// This is the main method called by the parser to get tokens one at a time.
func (l *Lexer) NextToken() Token {
	tok := l.nextToken()
	tok.NLBefore = l.sawNewline
	tok.SpaceBefore = l.sawSpace || l.sawNewline
	l.sawNewline = false
	l.sawSpace = false
	l.prevType = tok.Type
	return tok
}

// valueEnded reports whether the previous token ends a value, in which case
// an ambiguous character like '/' or '%' is a binary operator.
func (l *Lexer) valueEnded() bool {
	if l.sawNewline {
		return false
	}
	switch l.prevType {
	case IDENT, INT, FLOAT, STRING, RPAREN, RBRACKET, RBRACE, TRUE, FALSE, NIL, SELF, END,
		IVAR, CVAR, GVAR, SYMBOL, REGEX, WORDS, SUPER:
		return true
	}
	return false
}

// commandArgStart reports whether the current char starts what looks like the first
// argument of a paren-less call, e.g. `puts /x/` or `puts %w[a b]`: previous token is an
// identifier, there is whitespace before the char and none after.
func (l *Lexer) commandArgStart() bool {
	if l.prevType != IDENT {
		return false
	}
	prev := l.prevChar()
	next := l.peekChar()
	return (prev == ' ' || prev == '\t') && next != ' ' && next != '\t' && next != '=' && next != '\n'
}

func (l *Lexer) nextToken() Token {
	l.skipWhitespaceAndComments()

	startLine, startCol := l.line, l.column
	tok := Token{Line: startLine, Column: startCol}

	switch l.ch {
	case 0:
		tok.Type = EOF
		tok.Literal = ""
		return tok
	case '"':
		if l.peekChar() == '"' && l.peekAt(2) == '"' {
			tok.Type = STRING
			tok.Literal = l.readTripleQuotedString()
		} else {
			tok.Type = STRING
			tok.Literal = l.readString()
		}
		tok.Line = startLine
		tok.Column = l.column
		l.readChar()
		return tok
	case '\'':
		tok.Type = STRING
		tok.Literal = l.readString()
		tok.Line = startLine
		tok.Column = l.column
		l.readChar()
		return tok
	case '@':
		if t, ok := l.readInstanceVar(startLine, startCol); ok {
			return t
		}
	case '$':
		if isLetter(l.peekChar()) {
			l.readChar()
			tok.Type = GVAR
			tok.Literal = "$" + l.readIdentifier()
			return tok
		}
	case ':':
		if t, ok := l.readSymbol(startLine, startCol); ok {
			return t
		}
	case '/':
		if (!l.valueEnded() || l.commandArgStart()) && l.regexClosesOnLine() {
			return l.readRegex(startLine, startCol)
		}
	case '%':
		if (!l.valueEnded() || l.commandArgStart()) && (l.peekChar() == 'w' || l.peekChar() == 'i') && isOpenDelim(l.peekAt(2)) {
			return l.readWordList(startLine, startCol)
		}
	case '<':
		if t, ok := l.readHeredoc(startLine, startCol); ok {
			return t
		}
	}

	if isLetter(l.ch) {
		tok.Literal = l.readIdentifier()
		tok.Type = LookupIdent(tok.Literal)
		tok.Line = l.line
		tok.Column = l.column
		return tok
	}
	if isDigit(l.ch) {
		t := l.readNumber()
		return t
	}

	// Multi-character operators, longest match first.
	rest := l.input[l.position:]
	for _, op := range operators {
		if strings.HasPrefix(rest, op.text) {
			n := utf8.RuneCountInString(op.text)
			for k := 0; k < n-1; k++ {
				l.readChar()
			}
			tok = Token{Type: op.typ, Literal: op.text, Line: l.line, Column: l.column - (n - 1)}
			l.readChar()
			return tok
		}
	}

	if typ, ok := singleOps[l.ch]; ok {
		tok = Token{Type: typ, Literal: string(l.ch), Line: l.line, Column: l.column}
		l.readChar()
		return tok
	}

	tok = Token{Type: ILLEGAL, Literal: string(l.ch), Line: l.line, Column: l.column}
	l.readChar()
	return tok
}

// regexClosesOnLine reports whether an unescaped, unbracketed '/' closes the regex
// that starts at the current '/' before the end of the line.
func (l *Lexer) regexClosesOnLine() bool {
	inClass := false
	rest := l.input[l.readPosition:]
	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case '\\':
			i++
		case '\n':
			return false
		case '[':
			inClass = true
		case ']':
			inClass = false
		case '/':
			if !inClass {
				return true
			}
		}
	}
	return false
}

func isOpenDelim(r rune) bool {
	switch r {
	case '[', '(', '{', '<', '|', '!', '/':
		return true
	}
	return false
}

func closeDelim(r rune) rune {
	switch r {
	case '[':
		return ']'
	case '(':
		return ')'
	case '{':
		return '}'
	case '<':
		return '>'
	}
	return r
}

// skipWhitespaceAndComments skips spaces, newlines, "//" and "#" line comments and
// "=begin"/"=end" block comments, recording whether a newline was crossed.
func (l *Lexer) skipWhitespaceAndComments() {
	for {
		switch {
		case l.ch == '\n':
			l.sawNewline = true
			l.readChar()
		case unicode.IsSpace(l.ch) && l.ch != 0:
			l.sawSpace = true
			l.readChar()
		case l.ch == '#' || (l.ch == '/' && l.peekChar() == '/'):
			l.sawSpace = true
			l.skipComment()
		case l.ch == '=' && l.column == 1 && strings.HasPrefix(l.input[l.position:], "=begin"):
			l.skipBlockComment()
		default:
			return
		}
	}
}

// skipComment skips a line comment (// ... or # ...) up to the newline.
func (l *Lexer) skipComment() {
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
}

// skipBlockComment skips a "=begin ... =end" comment.
func (l *Lexer) skipBlockComment() {
	for l.ch != 0 {
		if l.ch == '\n' {
			l.sawNewline = true
			l.readChar()
			if strings.HasPrefix(l.input[l.position:], "=end") {
				l.skipComment()
				return
			}
			continue
		}
		l.readChar()
	}
}

// readIdentifier reads an identifier or keyword from the input.
// Identifiers start with a letter or underscore and can contain letters, digits and
// underscores, optionally ending in '?' or '!' (e.g. empty?, save!).
func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	if l.ch == '?' || l.ch == '!' {
		next := l.peekChar()
		if next != '=' && next != '?' && !isLetter(next) && !isDigit(next) && next != '"' && next != '\'' {
			l.readChar()
		}
	}
	return l.input[position:l.position]
}

// readNumber reads a numeric literal: decimal, hex (0x), binary (0b), octal (0o),
// with optional '_' digit separators, fractions and exponents. The returned literal
// is normalised so the parser can convert it with strconv.
func (l *Lexer) readNumber() Token {
	start := l.position
	line := l.line

	if l.ch == '0' {
		var base int
		switch l.peekChar() {
		case 'x', 'X':
			base = 16
		case 'b', 'B':
			base = 2
		case 'o', 'O':
			base = 8
		}
		if base != 0 && isBaseDigit(l.peekAt(2), base) {
			l.readChar() // 0
			l.readChar() // x/b/o
			for isBaseDigit(l.ch, base) || (l.ch == '_' && isBaseDigit(l.peekChar(), base)) {
				l.readChar()
			}
			text := strings.ReplaceAll(l.input[start:l.position], "_", "")
			if n, err := strconv.ParseInt(text, 0, 64); err == nil {
				text = strconv.FormatInt(n, 10)
			}
			return Token{Type: INT, Literal: text, Line: line, Column: l.column}
		}
	}

	isFloat := false
	for isDigit(l.ch) || (l.ch == '_' && isDigit(l.peekChar())) {
		l.readChar()
		if l.ch == '.' && !isFloat && isDigit(l.peekChar()) {
			isFloat = true
			l.readChar()
		}
	}
	if l.ch == 'e' || l.ch == 'E' {
		n := l.peekChar()
		if isDigit(n) || ((n == '+' || n == '-') && isDigit(l.peekAt(2))) {
			isFloat = true
			l.readChar()
			if l.ch == '+' || l.ch == '-' {
				l.readChar()
			}
			for isDigit(l.ch) {
				l.readChar()
			}
		}
	}

	literal := strings.ReplaceAll(l.input[start:l.position], "_", "")
	if isFloat {
		return Token{Type: FLOAT, Literal: literal, Line: line, Column: l.column}
	}
	return Token{Type: INT, Literal: literal, Line: line, Column: l.column}
}

func isBaseDigit(ch rune, base int) bool {
	switch base {
	case 16:
		return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
	case 2:
		return ch == '0' || ch == '1'
	case 8:
		return ch >= '0' && ch <= '7'
	}
	return false
}

// readString reads a string literal, handling escape sequences and interpolation.
// String literals are enclosed in double or single quotes; interpolation uses
// ${expression}. On return l.ch is the closing quote (not consumed).
func (l *Lexer) readString() string {
	openQuote := l.ch
	l.readChar() // skip the opening quote
	return l.readStringBody(openQuote)
}

// escapedDollar marks a literal '$' that was written as "\$" so the parser's
// interpolation scan does not treat "${" as the start of an expression.
const escapedDollar = '\x01'

// EscapedDollar is exported for the parser, which turns the marker back into '$'.
const EscapedDollar = string(rune(escapedDollar))

// readStringBody reads string contents until the terminator rune (0 means end of input).
func (l *Lexer) readStringBody(term rune) string {
	var out bytes.Buffer

	for {
		if l.ch == 0 || (term != 0 && l.ch == term) {
			break
		}

		if l.ch == '\\' {
			l.readChar() // consume the backslash
			switch l.ch {
			case 'n':
				out.WriteRune('\n')
			case 't':
				out.WriteRune('\t')
			case 'r':
				out.WriteRune('\r')
			case '0':
				out.WriteRune(0)
			case 'a':
				out.WriteRune('\a')
			case 'b':
				out.WriteRune('\b')
			case 'e':
				out.WriteRune(0x1b)
			case 'f':
				out.WriteRune('\f')
			case 'v':
				out.WriteRune('\v')
			case 's':
				out.WriteRune(' ')
			case '\\':
				out.WriteRune('\\')
			case '"':
				out.WriteRune('"')
			case '\'':
				out.WriteRune('\'')
			case '$':
				out.WriteRune(escapedDollar)
			case 'u':
				if r, ok := l.readUnicodeEscape(); ok {
					out.WriteRune(r)
					continue
				}
				out.WriteString("\\u")
			case 'x':
				if v, ok := l.readHexEscape(); ok {
					out.WriteRune(rune(v))
					continue
				}
				out.WriteString("\\x")
			case '\n':
				// line continuation: swallow the newline
			case 0:
				out.WriteRune('\\')
				return out.String()
			default:
				// Unknown escape: keep the backslash and the character.
				out.WriteRune('\\')
				out.WriteRune(l.ch)
			}
			l.readChar()
			continue
		}

		// String interpolation with ${ ... }; the expression text is copied verbatim.
		if l.ch == '$' && l.peekChar() == '{' {
			out.WriteString("${")
			l.readChar() // consume $
			l.readChar() // consume {
			l.copyInterpolation(&out)
			out.WriteRune('}')
			l.readChar() // consume the closing brace
			continue
		}

		out.WriteRune(l.ch)
		l.readChar()
	}

	return out.String()
}

// readUnicodeEscape reads \uXXXX or \u{X...}; l.ch is 'u' on entry and the next
// unread character on successful return.
func (l *Lexer) readUnicodeEscape() (rune, bool) {
	rest := l.input[l.readPosition:]
	if strings.HasPrefix(rest, "{") {
		end := strings.IndexByte(rest, '}')
		if end < 0 {
			return 0, false
		}
		v, err := strconv.ParseUint(rest[1:end], 16, 32)
		if err != nil {
			return 0, false
		}
		for k := 0; k <= end+1; k++ {
			l.readChar()
		}
		return rune(v), true
	}
	if len(rest) >= 4 {
		v, err := strconv.ParseUint(rest[:4], 16, 32)
		if err == nil {
			for k := 0; k < 5; k++ {
				l.readChar()
			}
			return rune(v), true
		}
	}
	return 0, false
}

// readHexEscape reads \xHH; l.ch is 'x' on entry.
func (l *Lexer) readHexEscape() (int, bool) {
	rest := l.input[l.readPosition:]
	if len(rest) >= 2 {
		v, err := strconv.ParseUint(rest[:2], 16, 8)
		if err == nil {
			for k := 0; k < 3; k++ {
				l.readChar()
			}
			return int(v), true
		}
	}
	return 0, false
}

// copyInterpolation copies the raw source of a ${...} expression into out, honouring
// nested braces and nested string literals. On return l.ch is the closing '}'.
func (l *Lexer) copyInterpolation(out *bytes.Buffer) {
	depth := 1
	for l.ch != 0 {
		switch l.ch {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return
			}
		case '"', '\'':
			l.copyQuoted(out)
			continue
		}
		out.WriteRune(l.ch)
		l.readChar()
	}
}

// copyQuoted copies a quoted string (including its quotes) verbatim.
func (l *Lexer) copyQuoted(out *bytes.Buffer) {
	q := l.ch
	out.WriteRune(q)
	l.readChar()
	for l.ch != 0 && l.ch != q {
		if l.ch == '\\' {
			out.WriteRune(l.ch)
			l.readChar()
			if l.ch == 0 {
				return
			}
			out.WriteRune(l.ch)
			l.readChar()
			continue
		}
		if l.ch == '$' && l.peekChar() == '{' && q == '"' {
			out.WriteString("${")
			l.readChar()
			l.readChar()
			l.copyInterpolation(out)
			out.WriteRune('}')
			l.readChar()
			continue
		}
		out.WriteRune(l.ch)
		l.readChar()
	}
	if l.ch == q {
		out.WriteRune(q)
		l.readChar()
	}
}

// readTripleQuotedString reads a triple-quoted multi-line string: """..."""
// On return l.ch is the last closing quote (not consumed).
func (l *Lexer) readTripleQuotedString() string {
	var out bytes.Buffer

	l.readChar() // skip first "
	l.readChar() // skip second "
	l.readChar() // skip third ", now at first content char

	for l.ch != 0 {
		if l.ch == '"' && l.peekChar() == '"' && l.peekAt(2) == '"' {
			l.readChar()
			l.readChar()
			break
		}
		out.WriteRune(l.ch)
		l.readChar()
	}

	return out.String()
}

// readInstanceVar reads @name and @@name.
func (l *Lexer) readInstanceVar(line, col int) (Token, bool) {
	if isLetter(l.peekChar()) {
		l.readChar()
		return Token{Type: IVAR, Literal: "@" + l.readIdentifier(), Line: line, Column: col}, true
	}
	if l.peekChar() == '@' && isLetter(l.peekAt(2)) {
		l.readChar()
		l.readChar()
		return Token{Type: CVAR, Literal: "@@" + l.readIdentifier(), Line: line, Column: col}, true
	}
	return Token{}, false
}

var operatorSymbols = []string{"<=>", "===", "[]=", "**", "==", "!=", "<=", ">=", "<<", ">>", "[]", "+", "-", "*", "/", "%", "<", ">", "!", "&", "|", "^", "~"}

// readSymbol reads :name, :"quoted" and operator symbols such as :+.
func (l *Lexer) readSymbol(line, col int) (Token, bool) {
	prev := l.prevChar()
	// A symbol cannot directly follow an identifier/closing token ("x:y", "a ?b:c"),
	// which are type annotations or hash keys.
	if isLetter(prev) || isDigit(prev) || prev == ')' || prev == ']' || prev == '}' || prev == '"' || prev == '\'' || prev == ':' {
		return Token{}, false
	}
	next := l.peekChar()
	if next == ':' {
		return Token{}, false
	}
	if isLetter(next) {
		l.readChar()
		name := l.readIdentifier()
		return Token{Type: SYMBOL, Literal: name, Line: line, Column: col}, true
	}
	if next == '"' {
		l.readChar()
		name := l.readString()
		l.readChar()
		return Token{Type: SYMBOL, Literal: name, Line: line, Column: col}, true
	}
	if prev == '(' || prev == ',' || prev == ' ' || prev == '\t' || prev == '[' {
		rest := l.input[l.readPosition:]
		for _, op := range operatorSymbols {
			if strings.HasPrefix(rest, op) {
				after := ' '
				if len(rest) > len(op) {
					after = rune(rest[len(op)])
				}
				if after == ')' || after == ',' || after == ' ' || after == '\n' || after == '\t' || after == ']' || after == 0 {
					for k := 0; k <= len(op); k++ {
						l.readChar()
					}
					return Token{Type: SYMBOL, Literal: op, Line: line, Column: col}, true
				}
			}
		}
	}
	return Token{}, false
}

// readRegex reads /pattern/flags. The pattern is the Literal and the flags are Extra.
func (l *Lexer) readRegex(line, col int) Token {
	l.readChar() // skip opening /
	var out bytes.Buffer
	inClass := false
	for l.ch != 0 {
		if l.ch == '\\' {
			out.WriteRune(l.ch)
			l.readChar()
			if l.ch == '/' {
				// "\/" is just a slash
				out.Truncate(out.Len() - 1)
			}
			if l.ch != 0 {
				out.WriteRune(l.ch)
				l.readChar()
			}
			continue
		}
		if l.ch == '[' {
			inClass = true
		} else if l.ch == ']' {
			inClass = false
		} else if l.ch == '/' && !inClass {
			break
		}
		out.WriteRune(l.ch)
		l.readChar()
	}
	l.readChar() // skip closing /
	flags := ""
	for l.ch == 'i' || l.ch == 'm' || l.ch == 'x' || l.ch == 's' {
		flags += string(l.ch)
		l.readChar()
	}
	return Token{Type: REGEX, Literal: out.String(), Extra: flags, Line: line, Column: col}
}

// readWordList reads %w[a b c] and %i[a b c]. Words are joined by a NUL byte in Literal.
func (l *Lexer) readWordList(line, col int) Token {
	l.readChar() // %
	kind := string(l.ch)
	l.readChar() // w or i
	closer := closeDelim(l.ch)
	l.readChar() // opening delimiter
	var words []string
	var cur bytes.Buffer
	flush := func() {
		if cur.Len() > 0 {
			words = append(words, cur.String())
			cur.Reset()
		}
	}
	for l.ch != 0 && l.ch != closer {
		if unicode.IsSpace(l.ch) {
			flush()
		} else if l.ch == '\\' && l.peekChar() != 0 {
			l.readChar()
			cur.WriteRune(l.ch)
		} else {
			cur.WriteRune(l.ch)
		}
		l.readChar()
	}
	flush()
	l.readChar() // closing delimiter
	return Token{Type: WORDS, Literal: strings.Join(words, "\x00"), Extra: kind, Line: line, Column: col}
}

// readHeredoc reads <<~ID, <<-ID and <<ID heredocs. The body starts on the line after
// the current one; it is consumed here and skipped when the lexer reaches that line.
func (l *Lexer) readHeredoc(line, col int) (Token, bool) {
	if l.peekChar() != '<' {
		return Token{}, false
	}
	squiggly, dash := false, false
	idx := 2
	switch l.peekAt(2) {
	case '~':
		squiggly = true
		idx = 3
	case '-':
		dash = true
		idx = 3
	}
	first := l.peekAt(idx)
	quote := rune(0)
	if first == '"' || first == '\'' {
		quote = first
		idx++
		first = l.peekAt(idx)
	}
	if !(unicode.IsUpper(first) || first == '_') {
		return Token{}, false
	}
	if !squiggly && !dash && l.valueEnded() && !(l.prevType == IDENT && l.prevChar() == ' ') {
		return Token{}, false
	}

	// Read the identifier.
	rest := l.input[l.position:]
	skip := 2
	if squiggly || dash {
		skip = 3
	}
	if quote != 0 {
		skip++
	}
	i := skip
	for i < len(rest) && (isLetter(rune(rest[i])) || isDigit(rune(rest[i]))) {
		i++
	}
	id := rest[skip:i]
	if id == "" {
		return Token{}, false
	}
	if quote != 0 {
		if i >= len(rest) || rune(rest[i]) != quote {
			return Token{}, false
		}
		i++
	}
	// Find where the body begins: after the end of the current line (or the previous heredoc).
	lineEnd := strings.IndexByte(l.input[l.position:], '\n')
	if lineEnd < 0 {
		return Token{}, false
	}
	lineStart := l.position + lineEnd + 1
	bodyStart := lineStart
	if l.heredocNext > bodyStart {
		bodyStart = l.heredocNext
	}

	var lines []string
	pos := bodyStart
	found := false
	for pos <= len(l.input) {
		end := strings.IndexByte(l.input[pos:], '\n')
		var ln string
		next := 0
		if end < 0 {
			ln = l.input[pos:]
			next = len(l.input)
		} else {
			ln = l.input[pos : pos+end]
			next = pos + end + 1
		}
		trimmed := strings.TrimRight(ln, "\r")
		check := trimmed
		if squiggly || dash {
			check = strings.TrimSpace(trimmed)
		}
		if check == id {
			pos = next
			found = true
			break
		}
		lines = append(lines, trimmed)
		if end < 0 {
			break
		}
		pos = next
	}
	if !found {
		pos = len(l.input)
	}

	if squiggly {
		lines = dedent(lines)
	}
	body := strings.Join(lines, "\n")
	if len(lines) > 0 {
		body += "\n"
	}

	// Mark the region so the lexer jumps over it once it reaches the newline.
	if l.heredocSkip == nil {
		l.heredocSkip = map[int]int{}
	}
	l.heredocSkip[lineStart] = pos
	l.heredocNext = pos

	// Advance past the heredoc marker.
	for k := 0; k < utf8.RuneCountInString(rest[:i]); k++ {
		l.readChar()
	}

	if quote == '\'' {
		return Token{Type: STRING, Literal: strings.ReplaceAll(body, "$", EscapedDollar), Line: line, Column: col}, true
	}
	sub := &Lexer{input: body, line: 1}
	sub.readChar()
	text := sub.readStringBody(0)
	return Token{Type: STRING, Literal: text, Line: line, Column: col}, true
}

// dedent removes the common leading whitespace of non-blank lines.
func dedent(lines []string) []string {
	min := -1
	for _, ln := range lines {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		n := len(ln) - len(strings.TrimLeft(ln, " \t"))
		if min < 0 || n < min {
			min = n
		}
	}
	if min <= 0 {
		return lines
	}
	out := make([]string, len(lines))
	for i, ln := range lines {
		if len(ln) >= min {
			out[i] = ln[min:]
		} else {
			out[i] = strings.TrimLeft(ln, " \t")
		}
	}
	return out
}

// isLetter returns true if the character is a letter or underscore.
func isLetter(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_'
}

// isDigit returns true if the character is a decimal digit.
func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}
