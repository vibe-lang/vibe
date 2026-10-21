package parser

import (
	"fmt"

	"github.com/vibe-lang/vibe/pkg/ast"
	"github.com/vibe-lang/vibe/pkg/lexer"
)

// Parser transforms tokens from the lexer into an AST.
//
// The parser is a Pratt (top-down operator precedence) parser over a
// pre-tokenized slice, which allows cheap backtracking. Line breaks are
// carried on tokens (Token.NLBefore) instead of being separate tokens: an
// expression ends at a line break unless it is inside brackets or the next
// line clearly continues it (leading ".", "|>", "&&", ...).
//
// Convention: after any parse function returns, curToken is the last token
// of the construct that was parsed.
type Parser struct {
	toks []lexer.Token
	pos  int

	curToken      lexer.Token
	peekToken     lexer.Token
	peekPeekToken lexer.Token

	errors []string

	nlIgnore int // >0 while inside (), [] or {} where line breaks are insignificant
	noDo     int // >0 while parsing a header where `do` belongs to the enclosing construct
	noAssign int // >0 while parsing assignment targets
	noUnion  int // >0 while parsing types where `|` is a delimiter

	scopes []map[string]bool // known local variable names
	ctx    []ctxKind         // enclosing loop / function / block contexts
}

type ctxKind int

const (
	ctxTop ctxKind = iota
	ctxLoop
	ctxFunc
)

// Precedence levels for operators, lowest to highest.
const (
	LOWEST    = 1
	KWLOGIC   = 2  // and, or
	ASSIGN    = 3  // right-hand side of assignments
	TERNARY   = 4  // cond ? a : b
	NIL_COAL  = 5  // ??
	LOGIC_OR  = 6  // ||
	LOGIC_AND = 7  // &&
	CONTAINS  = 8  // in
	RANGE     = 9  // .. ...
	EQUALS    = 10 // == != <=> =~ !~
	COMPARE   = 11 // < > <= >=
	BITOR     = 12 // | ^
	BITAND_P  = 13 // &
	SHIFT     = 14 // << >>
	SUM       = 15 // + -
	PRODUCT   = 16 // * / %
	PREFIX    = 17 // -x !x
	POWER     = 18 // **
	PIPE      = 19 // |>
	CALL      = 20 // f(x)
	INDEX     = 21 // a[i]
	DOT       = 22 // a.b
)

var precedences = map[lexer.TokenType]int{
	lexer.KW_AND:       KWLOGIC,
	lexer.KW_OR:        KWLOGIC,
	lexer.QUESTION:     TERNARY,
	lexer.NIL_COALESCE: NIL_COAL,
	lexer.OR:           LOGIC_OR,
	lexer.AND:          LOGIC_AND,
	lexer.IN:           CONTAINS,
	lexer.KW_NOT:       CONTAINS,
	lexer.DOTDOT:       RANGE,
	lexer.DOTDOTDOT:    RANGE,
	lexer.EQ:           EQUALS,
	lexer.NOT_EQ:       EQUALS,
	lexer.SPACESHIP:    EQUALS,
	lexer.MATCH:        EQUALS,
	lexer.NOT_MATCH:    EQUALS,
	lexer.LT:           COMPARE,
	lexer.GT:           COMPARE,
	lexer.LTE:          COMPARE,
	lexer.GTE:          COMPARE,
	lexer.PIPE:         BITOR,
	lexer.BITXOR:       BITOR,
	lexer.BITAND:       BITAND_P,
	lexer.LSHIFT:       SHIFT,
	lexer.RSHIFT:       SHIFT,
	lexer.PLUS:         SUM,
	lexer.MINUS:        SUM,
	lexer.ASTERISK:     PRODUCT,
	lexer.SLASH:        PRODUCT,
	lexer.MODULO:       PRODUCT,
	lexer.POWER_OP:     POWER,
	lexer.PIPE_ARROW:   PIPE,
	lexer.LPAREN:       CALL,
	lexer.LBRACE:       CALL,
	lexer.DO:           CALL,
	lexer.LBRACKET:     INDEX,
	lexer.DOT:          DOT,
	lexer.SAFE_NAV:     DOT,
	lexer.DOUBLE_COLON: DOT,
}

// New creates a parser for the given lexer.
func New(l *lexer.Lexer) *Parser {
	p := &Parser{errors: []string{}, scopes: []map[string]bool{{}}, ctx: []ctxKind{ctxTop}}
	for {
		t := l.NextToken()
		p.toks = append(p.toks, t)
		if t.Type == lexer.EOF {
			break
		}
	}
	p.setPos(0)
	return p
}

// Errors returns all errors encountered during parsing.
func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) tok(i int) lexer.Token {
	if i < 0 {
		i = 0
	}
	if i >= len(p.toks) {
		return p.toks[len(p.toks)-1]
	}
	return p.toks[i]
}

func (p *Parser) setPos(i int) {
	p.pos = i
	p.curToken = p.tok(i)
	p.peekToken = p.tok(i + 1)
	p.peekPeekToken = p.tok(i + 2)
}

func (p *Parser) nextToken() {
	if p.curToken.Type == lexer.EOF {
		return
	}
	p.setPos(p.pos + 1)
}

func (p *Parser) addError(msg string) {
	p.errors = append(p.errors, fmt.Sprintf("[%d:%d] %s", p.curToken.Line, p.curToken.Column, msg))
}

func (p *Parser) peekError(t lexer.TokenType) {
	p.addError(fmt.Sprintf("expected next token to be %s, got %s instead", t, p.peekToken.Type))
}

func (p *Parser) curTokenIs(t lexer.TokenType) bool  { return p.curToken.Type == t }
func (p *Parser) peekTokenIs(t lexer.TokenType) bool { return p.peekToken.Type == t }
func (p *Parser) peekPeekTokenIs(t lexer.TokenType) bool {
	return p.peekPeekToken.Type == t
}

func (p *Parser) expectPeek(t lexer.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

func (p *Parser) curIn(types ...lexer.TokenType) bool {
	for _, t := range types {
		if p.curToken.Type == t {
			return true
		}
	}
	return false
}

func (p *Parser) peekIn(types ...lexer.TokenType) bool {
	for _, t := range types {
		if p.peekToken.Type == t {
			return true
		}
	}
	return false
}

// --- speculation -----------------------------------------------------------

type mark struct {
	pos  int
	errs int
	toks []lexer.Token
}

func (p *Parser) mark() mark { return mark{p.pos, len(p.errors), p.toks} }

func (p *Parser) reset(m mark) {
	p.toks = m.toks
	p.errors = p.errors[:m.errs]
	p.setPos(m.pos)
}

// --- scopes and contexts ---------------------------------------------------

func (p *Parser) pushScope() { p.scopes = append(p.scopes, map[string]bool{}) }
func (p *Parser) popScope()  { p.scopes = p.scopes[:len(p.scopes)-1] }

func (p *Parser) declare(name string) {
	if name != "" {
		p.scopes[len(p.scopes)-1][name] = true
	}
}

func (p *Parser) isLocal(name string) bool {
	for i := len(p.scopes) - 1; i >= 0; i-- {
		if p.scopes[i][name] {
			return true
		}
	}
	return false
}

func (p *Parser) pushCtx(k ctxKind) { p.ctx = append(p.ctx, k) }
func (p *Parser) popCtx()           { p.ctx = p.ctx[:len(p.ctx)-1] }
func (p *Parser) inLoop() bool      { return p.ctx[len(p.ctx)-1] == ctxLoop }

// --- program and statements ------------------------------------------------

// ParseProgram parses a complete program.
func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{Statements: []ast.Statement{}}
	program.Statements = p.parseStatements()
	return program
}

// parseStatements parses statements starting at curToken until one of the stop
// tokens (or EOF) is current. curToken is left on the stop token.
func (p *Parser) parseStatements(stops ...lexer.TokenType) []ast.Statement {
	savedNL, savedDo, savedAssign, savedUnion := p.nlIgnore, p.noDo, p.noAssign, p.noUnion
	p.nlIgnore, p.noDo, p.noAssign, p.noUnion = 0, 0, 0, 0
	defer func() { p.nlIgnore, p.noDo, p.noAssign, p.noUnion = savedNL, savedDo, savedAssign, savedUnion }()

	stmts := []ast.Statement{}
	for {
		for p.curTokenIs(lexer.SEMICOLON) {
			p.nextToken()
		}
		if p.curTokenIs(lexer.EOF) || p.curIn(stops...) {
			return stmts
		}
		start := p.pos
		if stmt := p.parseStatement(); stmt != nil {
			stmts = append(stmts, stmt)
		}
		if p.pos == start && !p.curIn(stops...) {
			// Make sure a failed statement still makes progress.
		}
		p.nextToken()
	}
}

func (p *Parser) parseBlockUntil(stops ...lexer.TokenType) *ast.BlockStatement {
	tok := p.curToken
	return &ast.BlockStatement{Token: tok, Statements: p.parseStatements(stops...)}
}

// startBody moves from the last header token to the first body token,
// skipping separators and an optional keyword such as `then` or `do`.
func (p *Parser) startBody(kw ...lexer.TokenType) {
	for p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken()
	}
	if len(kw) > 0 && p.peekIn(kw...) {
		p.nextToken()
	}
	p.nextToken()
}

func (p *Parser) expectEnd(what string) bool {
	if p.curTokenIs(lexer.END) {
		return true
	}
	p.addError(fmt.Sprintf("expected 'end' to close %s", what))
	return false
}

// valueFollows reports whether an optional value (after return/break/next)
// starts on the same line.
func (p *Parser) valueFollows() bool {
	t := p.peekToken
	if t.NLBefore {
		return false
	}
	switch t.Type {
	case lexer.SEMICOLON, lexer.END, lexer.RBRACE, lexer.RPAREN, lexer.RBRACKET, lexer.EOF,
		lexer.IF, lexer.UNLESS, lexer.WHILE, lexer.UNTIL, lexer.RESCUE, lexer.ELSE, lexer.ELSIF,
		lexer.WHEN, lexer.ENSURE, lexer.FINALLY, lexer.CATCH, lexer.COLON, lexer.COMMA:
		return false
	}
	return true
}

func (p *Parser) parseStatement() ast.Statement {
	first := p.curToken
	var stmt ast.Statement

	switch p.curToken.Type {
	case lexer.RETURN:
		stmt = p.parseReturnStatement()
	case lexer.BREAK:
		b := &ast.BreakStatement{Token: p.curToken}
		if p.valueFollows() {
			p.nextToken()
			b.Value = p.parseExpression(LOWEST)
		}
		stmt = b
	case lexer.CONTINUE:
		stmt = &ast.ContinueStatement{Token: p.curToken}
	case lexer.NEXT:
		stmt = p.parseNextStatement()
	case lexer.THROW:
		s := &ast.ThrowStatement{Token: p.curToken}
		p.nextToken()
		s.Value = p.parseExpression(LOWEST)
		stmt = s
	case lexer.IMPORT:
		stmt = p.parseImportStatement()
	case lexer.STRUCT:
		stmt = p.parseStructStatement()
	case lexer.ENUM:
		stmt = p.parseEnumStatement()
	case lexer.CONST:
		stmt = p.parseConstStatement()
	case lexer.LET:
		p.nextToken()
		return p.parseStatement()
	case lexer.FOR:
		stmt = p.parseForStatement()
	case lexer.WHILE:
		stmt = p.parseWhileStatement()
	case lexer.UNTIL:
		stmt = p.parseUntilStatement()
	case lexer.ALIAS:
		stmt = p.parseAliasStatement()
	case lexer.IDENT:
		stmt = p.parseIdentStatement()
	default:
		stmt = p.parseSimpleStatement(first)
	}
	if stmt == nil {
		return nil
	}
	return p.applyModifiers(stmt)
}

// parseIdentStatement handles statements that start with an identifier:
// contextual keywords, typed declarations, multiple assignment and
// ordinary expressions.
func (p *Parser) parseIdentStatement() ast.Statement {
	lit := p.curToken.Literal
	switch {
	case lit == "retry" && !p.peekIn(lexer.LPAREN, lexer.ASSIGN, lexer.DOT):
		return &ast.RetryStatement{Token: p.curToken}
	case lit == "type" && p.peekTokenIs(lexer.IDENT) && p.peekPeekTokenIs(lexer.ASSIGN) && !p.peekToken.NLBefore:
		return p.parseTypeAliasStatement()
	case lit == "interface" && p.peekTokenIs(lexer.IDENT) && !p.peekToken.NLBefore:
		return p.parseInterfaceStatement()
	case p.peekIsLabel():
		return p.parseTypedDeclaration()
	}
	return p.parseSimpleStatement(p.curToken)
}

// parseSimpleStatement parses a multiple assignment or an expression statement.
func (p *Parser) parseSimpleStatement(first lexer.Token) ast.Statement {
	if p.looksLikeMultiAssign() {
		if s := p.parseMultiAssign(); s != nil {
			return s
		}
		return nil
	}
	expr := p.parseExpression(LOWEST)
	if expr == nil {
		return nil
	}
	if p.peekTokenIs(lexer.FAT_ARROW) && !p.peekToken.NLBefore {
		p.nextToken()
		tok := p.curToken
		p.nextToken()
		pat := p.parseTopPattern()
		if pat == nil {
			return nil
		}
		return &ast.ExpressionStatement{Token: first, Expression: &ast.MatchPattern{Token: tok, Value: expr, Pattern: pat, Strict: true}}
	}
	return &ast.ExpressionStatement{Token: first, Expression: expr}
}

func (p *Parser) parseReturnStatement() ast.Statement {
	stmt := &ast.ReturnStatement{Token: p.curToken}
	if !p.valueFollows() {
		return stmt
	}
	p.nextToken()
	v := p.parseExpression(LOWEST)
	if p.peekTokenIs(lexer.COMMA) {
		arr := &ast.ArrayLiteral{Token: p.curToken, Elements: []ast.Expression{v}}
		for p.peekTokenIs(lexer.COMMA) {
			p.nextToken()
			p.nextToken()
			arr.Elements = append(arr.Elements, p.parseExpression(ASSIGN))
		}
		v = arr
	}
	stmt.Value = v
	return stmt
}

// parseNextStatement desugars `next`: inside a loop it is `continue`, inside a
// block or function it returns the given value from the block.
func (p *Parser) parseNextStatement() ast.Statement {
	tok := p.curToken
	var val ast.Expression
	if p.valueFollows() {
		p.nextToken()
		val = p.parseExpression(LOWEST)
	}
	if p.inLoop() {
		return &ast.ContinueStatement{Token: tok}
	}
	return &ast.ReturnStatement{Token: tok, Value: val}
}

func (p *Parser) parseImportStatement() ast.Statement {
	stmt := &ast.ImportStatement{Token: p.curToken}
	if !p.expectPeek(lexer.STRING) {
		return nil
	}
	stmt.Path = p.curToken.Literal
	return stmt
}

func (p *Parser) parseAliasStatement() ast.Statement {
	stmt := &ast.AliasStatement{Token: p.curToken}
	name := func() (string, bool) {
		p.nextToken()
		if p.curIn(lexer.IDENT, lexer.SYMBOL) || isWordTok(p.curToken) {
			return p.curToken.Literal, true
		}
		p.addError("expected a method name in alias")
		return "", false
	}
	var ok bool
	if stmt.NewName, ok = name(); !ok {
		return nil
	}
	if stmt.OldName, ok = name(); !ok {
		return nil
	}
	return stmt
}

func (p *Parser) parseConstStatement() ast.Statement {
	stmt := &ast.ConstStatement{Token: p.curToken}
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	if p.peekTokenIs(lexer.COLON) {
		p.nextToken()
		p.nextToken()
		stmt.TypeAnnotation = p.parseType()
	}
	if !p.expectPeek(lexer.ASSIGN) {
		return nil
	}
	p.nextToken()
	stmt.Value = p.parseExpression(ASSIGN)
	p.declare(stmt.Name.Value)
	return stmt
}

// parseTypedDeclaration parses `name: type` and `name: type = value`.
func (p *Parser) parseTypedDeclaration() ast.Statement {
	first := p.curToken
	ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	p.nextToken() // colon
	p.nextToken() // type
	typeAnnotation := p.parseType()
	if typeAnnotation == nil {
		return nil
	}
	if p.peekTokenIs(lexer.ASSIGN) {
		p.nextToken()
		assign := &ast.AssignmentExpression{Token: p.curToken, Name: ident, TypeAnnotation: typeAnnotation}
		p.nextToken()
		assign.Value = p.parseExpression(ASSIGN)
		if _, ok := typeAnnotation.(*ast.CompoundTypeAnnotation); ok {
			if arr, ok := assign.Value.(*ast.ArrayLiteral); ok {
				assign.Value = &ast.CompoundLiteral{Token: arr.Token, Elements: arr.Elements}
			}
		}
		p.declare(ident.Value)
		return &ast.ExpressionStatement{Token: first, Expression: assign}
	}
	p.declare(ident.Value)
	typeAnno, ok := typeAnnotation.(*ast.TypeAnnotation)
	if !ok {
		typeAnno = &ast.TypeAnnotation{Token: ident.Token, Name: typeAnnotation.String()}
	}
	return &ast.ExpressionStatement{Token: first, Expression: &ast.TypedIdentifier{
		Token: ident.Token, Identifier: ident, Type: typeAnno,
	}}
}

// applyModifiers wraps a statement in trailing `if`, `unless`, `while`,
// `until` and `rescue` modifiers that sit on the same line.
func (p *Parser) applyModifiers(stmt ast.Statement) ast.Statement {
	for {
		pk := p.peekToken
		if pk.NLBefore {
			return stmt
		}
		switch pk.Type {
		case lexer.IF, lexer.UNLESS:
			p.nextToken()
			tok := p.curToken
			p.nextToken()
			cond := p.parseExpression(LOWEST)
			stmt = &ast.PostfixCondition{Token: tok, Statement: stmt, Condition: cond, Unless: tok.Type == lexer.UNLESS}
		case lexer.WHILE, lexer.UNTIL:
			p.nextToken()
			tok := p.curToken
			p.nextToken()
			cond := p.parseExpression(LOWEST)
			stmt = &ast.PostfixLoop{
				Token: tok, Statement: stmt, Condition: cond,
				Until: tok.Type == lexer.UNTIL, DoWhile: isBeginBlock(stmt),
			}
		case lexer.RESCUE:
			p.nextToken()
			tok := p.curToken
			p.nextToken()
			alt := p.parseExpression(LOWEST)
			stmt = wrapRescueModifier(stmt, alt, tok)
		default:
			return stmt
		}
	}
}

func isBeginBlock(stmt ast.Statement) bool {
	es, ok := stmt.(*ast.ExpressionStatement)
	if !ok {
		return false
	}
	t, ok := es.Expression.(*ast.TryExpression)
	return ok && t.Token.Type == lexer.BEGIN
}

func wrapRescueModifier(stmt ast.Statement, alt ast.Expression, tok lexer.Token) ast.Statement {
	wrap := func(inner ast.Expression) ast.Expression {
		return &ast.TryExpression{
			Token: tok,
			Body: &ast.BlockStatement{Token: tok, Statements: []ast.Statement{
				&ast.ExpressionStatement{Token: tok, Expression: inner},
			}},
			Rescues: []*ast.RescueClause{{
				Token: tok,
				Body: &ast.BlockStatement{Token: tok, Statements: []ast.Statement{
					&ast.ExpressionStatement{Token: tok, Expression: alt},
				}},
			}},
		}
	}
	if es, ok := stmt.(*ast.ExpressionStatement); ok {
		switch e := es.Expression.(type) {
		case *ast.AssignmentExpression:
			e.Value = wrap(e.Value)
			return stmt
		case *ast.IndexAssignment:
			e.Value = wrap(e.Value)
			return stmt
		case *ast.DotAssignment:
			e.Value = wrap(e.Value)
			return stmt
		}
		return &ast.ExpressionStatement{Token: es.Token, Expression: wrap(es.Expression)}
	}
	return &ast.ExpressionStatement{Token: tok, Expression: &ast.TryExpression{
		Token: tok,
		Body:  &ast.BlockStatement{Token: tok, Statements: []ast.Statement{stmt}},
		Rescues: []*ast.RescueClause{{
			Token: tok,
			Body: &ast.BlockStatement{Token: tok, Statements: []ast.Statement{
				&ast.ExpressionStatement{Token: tok, Expression: alt},
			}},
		}},
	}}
}

// --- loops -----------------------------------------------------------------

func (p *Parser) parseWhileStatement() ast.Statement {
	loop := &ast.WhileLoop{Token: p.curToken}
	p.nextToken()
	p.noDo++
	loop.Condition = p.parseExpression(LOWEST)
	p.noDo--
	p.startBody(lexer.DO)
	p.pushCtx(ctxLoop)
	loop.Body = p.parseBlockUntil(lexer.END)
	p.popCtx()
	if !p.expectEnd("while loop") {
		return nil
	}
	return loop
}

func (p *Parser) parseUntilStatement() ast.Statement {
	loop := &ast.UntilLoop{Token: p.curToken}
	p.nextToken()
	p.noDo++
	loop.Condition = p.parseExpression(LOWEST)
	p.noDo--
	p.startBody(lexer.DO)
	p.pushCtx(ctxLoop)
	loop.Body = p.parseBlockUntil(lexer.END)
	p.popCtx()
	if !p.expectEnd("until loop") {
		return nil
	}
	return loop
}

func (p *Parser) parseForStatement() ast.Statement {
	loop := &ast.ForLoop{Token: p.curToken}
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	loop.Iterator = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	p.declare(loop.Iterator.Value)
	if p.peekTokenIs(lexer.COMMA) {
		p.nextToken()
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		loop.ValueIterator = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		p.declare(loop.ValueIterator.Value)
	}
	if !p.expectPeek(lexer.IN) {
		return nil
	}
	p.nextToken()
	p.noDo++
	loop.Collection = p.parseExpression(LOWEST)
	p.noDo--
	p.startBody(lexer.DO)
	p.pushCtx(ctxLoop)
	loop.Body = p.parseBlockUntil(lexer.END)
	p.popCtx()
	if !p.expectEnd("for loop") {
		return nil
	}
	return loop
}

// --- multiple assignment ---------------------------------------------------

func isTargetTok(t lexer.TokenType) bool {
	switch t {
	case lexer.IDENT, lexer.IVAR, lexer.CVAR, lexer.GVAR, lexer.SELF, lexer.STRING, lexer.INT, lexer.SYMBOL:
		return true
	}
	return false
}

// looksLikeMultiAssign scans ahead (without consuming) for `a, b = ...`,
// `*a, b = ...` and `(a, b), c = ...`.
func (p *Parser) looksLikeMultiAssign() bool {
	depth := 0
	sawComma := p.curTokenIs(lexer.ASTERISK)
	prevTarget := false
	for i := p.pos; ; i++ {
		t := p.tok(i)
		if i > p.pos && t.NLBefore && depth == 0 {
			return false
		}
		switch t.Type {
		case lexer.EOF:
			return false
		case lexer.SEMICOLON:
			if depth == 0 {
				return false
			}
		case lexer.LPAREN, lexer.LBRACKET:
			depth++
			prevTarget = false
		case lexer.RPAREN, lexer.RBRACKET:
			depth--
			if depth < 0 {
				return false
			}
			prevTarget = true
		case lexer.COMMA:
			if depth == 0 {
				sawComma = true
			}
			prevTarget = false
		case lexer.ASSIGN:
			if depth == 0 {
				return sawComma
			}
		case lexer.DOT, lexer.ASTERISK, lexer.SAFE_NAV, lexer.DOUBLE_COLON:
			prevTarget = false
		default:
			if depth == 0 {
				if !isTargetTok(t.Type) {
					return false
				}
				if prevTarget {
					return false
				}
				prevTarget = true
			}
		}
	}
}

func (p *Parser) parseMultiTargets() []ast.Expression {
	var targets []ast.Expression
	for {
		var target ast.Expression
		switch {
		case p.curTokenIs(lexer.ASTERISK):
			splat := &ast.SplatExpression{Token: p.curToken}
			if p.peekIn(lexer.IDENT, lexer.IVAR, lexer.GVAR, lexer.CVAR) && !p.peekToken.NLBefore {
				p.nextToken()
				splat.Value = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
				p.declare(p.curToken.Literal)
			}
			target = splat
		case p.curTokenIs(lexer.LPAREN):
			nested := &ast.DestructureAssignment{Token: p.curToken}
			p.nextToken()
			nested.Targets = p.parseMultiTargets()
			nested.Names = identNames(nested.Targets)
			if !p.expectPeek(lexer.RPAREN) {
				return nil
			}
			target = nested
		default:
			p.noAssign++
			target = p.parseExpression(PIPE)
			p.noAssign--
			if id, ok := target.(*ast.Identifier); ok {
				p.declare(id.Value)
			}
		}
		if target == nil {
			return nil
		}
		targets = append(targets, target)
		if p.peekTokenIs(lexer.COMMA) {
			p.nextToken()
			p.nextToken()
			continue
		}
		return targets
	}
}

func identNames(targets []ast.Expression) []*ast.Identifier {
	names := make([]*ast.Identifier, 0, len(targets))
	for _, t := range targets {
		id, ok := t.(*ast.Identifier)
		if !ok {
			return nil
		}
		names = append(names, id)
	}
	return names
}

func (p *Parser) parseMultiAssign() ast.Statement {
	first := p.curToken
	targets := p.parseMultiTargets()
	if targets == nil || !p.expectPeek(lexer.ASSIGN) {
		return nil
	}
	assign := &ast.DestructureAssignment{Token: p.curToken, Targets: targets, Names: identNames(targets)}
	p.nextToken()
	values := []ast.Expression{p.parseExpression(ASSIGN)}
	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken()
		p.nextToken()
		values = append(values, p.parseExpression(ASSIGN))
	}
	_, isSplat := values[0].(*ast.SplatExpression)
	if len(values) == 1 && !isSplat {
		assign.Value = values[0]
	} else {
		assign.Value = &ast.ArrayLiteral{Token: first, Elements: values}
	}
	if assign.Value == nil {
		return nil
	}
	return &ast.ExpressionStatement{Token: first, Expression: assign}
}
