package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/vibe-lang/vibe/pkg/ast"
	"github.com/vibe-lang/vibe/pkg/lexer"
)

func (p *Parser) peekPrecedence() int {
	if prec, ok := precedences[p.peekToken.Type]; ok {
		return prec
	}
	return LOWEST
}

func (p *Parser) curPrecedence() int {
	if prec, ok := precedences[p.curToken.Type]; ok {
		return prec
	}
	return LOWEST
}

// enterGroup is called when entering (), [] or {}: line breaks become
// insignificant and `do`/assignment restrictions of the enclosing header are lifted.
func (p *Parser) enterGroup() func() {
	savedDo, savedAssign, savedUnion := p.noDo, p.noAssign, p.noUnion
	p.nlIgnore++
	p.noDo, p.noAssign, p.noUnion = 0, 0, 0
	return func() {
		p.nlIgnore--
		p.noDo, p.noAssign, p.noUnion = savedDo, savedAssign, savedUnion
	}
}

// continuesLine reports whether the peek token continues the current
// expression although it begins a new line (method chains and logical operators).
func continuesLine(t lexer.TokenType) bool {
	switch t {
	case lexer.DOT, lexer.SAFE_NAV, lexer.DOUBLE_COLON, lexer.PIPE_ARROW, lexer.AND, lexer.OR, lexer.NIL_COALESCE:
		return true
	}
	return false
}

func isAssignTok(t lexer.TokenType) bool {
	switch t {
	case lexer.ASSIGN, lexer.PLUS_ASSIGN, lexer.MINUS_ASSIGN, lexer.ASTERISK_ASSIGN, lexer.SLASH_ASSIGN,
		lexer.MODULO_ASSIGN, lexer.POWER_ASSIGN, lexer.OR_ASSIGN, lexer.AND_ASSIGN, lexer.NIL_ASSIGN,
		lexer.BITAND_ASSIGN, lexer.BITOR_ASSIGN, lexer.BITXOR_ASSIGN, lexer.LSHIFT_ASSIGN, lexer.RSHIFT_ASSIGN:
		return true
	}
	return false
}

func assignable(e ast.Expression) bool {
	switch n := e.(type) {
	case *ast.Identifier:
		return n.Value != "self" && n.Value != "super"
	case *ast.IndexExpression, *ast.DotExpression:
		return true
	}
	return false
}

// parseExpression parses an expression whose operators bind tighter than precedence.
func (p *Parser) parseExpression(precedence int) ast.Expression {
	left := p.parsePrefix()
	if left == nil {
		return nil
	}
	for {
		pk := p.peekToken
		if pk.NLBefore && p.nlIgnore == 0 && !continuesLine(pk.Type) {
			break
		}
		if isAssignTok(pk.Type) {
			if p.noAssign > 0 || !assignable(left) {
				break
			}
			left = p.parseAssignment(left)
			if left == nil {
				return nil
			}
			continue
		}
		if pk.Type == lexer.COLON && !pk.SpaceBefore && p.peekPeekTokenIs(lexer.COLON) && !p.peekPeekToken.SpaceBefore {
			p.nextToken()
			p.nextToken()
			left = p.parseDotExpression(left)
			if left == nil {
				return nil
			}
			continue
		}
		prec, ok := precedences[pk.Type]
		if !ok || precedence >= prec {
			break
		}
		if pk.Type == lexer.KW_NOT && !p.peekPeekTokenIs(lexer.IN) {
			break
		}
		switch pk.Type {
		case lexer.KW_NOT:
			p.nextToken()
			p.nextToken()
			tok := p.curToken
			p.nextToken()
			right := p.parseExpression(CONTAINS)
			if right == nil {
				return nil
			}
			bang := lexer.Token{Type: lexer.BANG, Literal: "!", Line: tok.Line, Column: tok.Column}
			left = &ast.PrefixExpression{Token: bang, Operator: "!", Right: &ast.InExpression{Token: tok, Left: left, Right: right}}
		case lexer.LPAREN:
			if id, ok := left.(*ast.Identifier); ok && pk.SpaceBefore && p.isLocal(id.Value) {
				return left
			}
			p.nextToken()
			left = p.parseCallExpression(left)
		case lexer.LBRACE, lexer.DO:
			if pk.Type == lexer.DO && p.noDo > 0 {
				return left
			}
			if !p.canTakeBlock(left) {
				return left
			}
			p.nextToken()
			left = p.attachBlock(left, p.parseBlock())
		case lexer.QUESTION:
			p.nextToken()
			left = p.parseTernary(left)
		case lexer.DOT, lexer.SAFE_NAV, lexer.DOUBLE_COLON:
			p.nextToken()
			left = p.parseDotExpression(left)
		case lexer.LBRACKET:
			p.nextToken()
			left = p.parseIndexExpression(left)
		case lexer.DOTDOT, lexer.DOTDOTDOT:
			p.nextToken()
			left = p.parseRangeExpression(left)
		case lexer.PIPE_ARROW:
			p.nextToken()
			tok := p.curToken
			p.nextToken()
			right := p.parseExpression(PIPE)
			left = &ast.PipeExpression{Token: tok, Left: left, Right: right}
		case lexer.NIL_COALESCE:
			p.nextToken()
			tok := p.curToken
			p.nextToken()
			right := p.parseExpression(NIL_COAL - 1)
			left = &ast.NilCoalesceExpression{Token: tok, Left: left, Right: right}
		case lexer.IN:
			p.nextToken()
			tok := p.curToken
			p.nextToken()
			right := p.parseExpression(CONTAINS)
			left = &ast.InExpression{Token: tok, Left: left, Right: right}
		default:
			p.nextToken()
			left = p.parseInfixExpression(left)
		}
		if left == nil {
			return nil
		}
	}
	return left
}

func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	tok := p.curToken
	op := tok.Literal
	switch tok.Type {
	case lexer.KW_AND:
		op = "&&"
		tok = lexer.Token{Type: lexer.AND, Literal: "&&", Line: tok.Line, Column: tok.Column}
	case lexer.KW_OR:
		op = "||"
		tok = lexer.Token{Type: lexer.OR, Literal: "||", Line: tok.Line, Column: tok.Column}
	}
	prec := p.curPrecedence()
	rightPrec := prec
	if tok.Type == lexer.POWER_OP {
		rightPrec = prec - 1 // right associative
	}
	p.nextToken()
	right := p.parseExpression(rightPrec)
	if right == nil {
		return nil
	}
	return &ast.InfixExpression{Token: tok, Left: left, Operator: op, Right: right}
}

func (p *Parser) parseTernary(cond ast.Expression) ast.Expression {
	expr := &ast.TernaryExpression{Token: p.curToken, Condition: cond}
	p.nextToken()
	expr.Consequence = p.parseExpression(ASSIGN)
	if !p.expectPeek(lexer.COLON) {
		p.addError("expected ':' in ternary expression")
		return nil
	}
	p.nextToken()
	expr.Alternative = p.parseExpression(TERNARY - 1)
	if expr.Consequence == nil || expr.Alternative == nil {
		return nil
	}
	return expr
}

func (p *Parser) canStartExpr(t lexer.Token) bool {
	if t.NLBefore && p.nlIgnore == 0 {
		return false
	}
	switch t.Type {
	case lexer.INT, lexer.FLOAT, lexer.STRING, lexer.SYMBOL, lexer.REGEX, lexer.WORDS, lexer.IDENT,
		lexer.IVAR, lexer.CVAR, lexer.GVAR, lexer.TRUE, lexer.FALSE, lexer.NIL, lexer.SELF, lexer.SUPER,
		lexer.LPAREN, lexer.LBRACKET, lexer.LBRACE, lexer.MINUS, lexer.BANG, lexer.ARROW, lexer.FUNCTION,
		lexer.KW_NOT, lexer.IF, lexer.CASE, lexer.UNLESS:
		return true
	}
	return false
}

func (p *Parser) parseRangeExpression(left ast.Expression) ast.Expression {
	expr := &ast.RangeExpression{Token: p.curToken, Start: left, Exclusive: p.curToken.Type == lexer.DOTDOTDOT}
	if !p.canStartExpr(p.peekToken) {
		return expr
	}
	p.nextToken()
	expr.End = p.parseExpression(RANGE)
	if expr.End == nil {
		return nil
	}
	return expr
}

// --- assignment ------------------------------------------------------------

func (p *Parser) parseAssignment(left ast.Expression) ast.Expression {
	p.nextToken()
	opTok := p.curToken
	p.nextToken()
	if opTok.Type == lexer.ASSIGN {
		if id, ok := left.(*ast.Identifier); ok {
			p.declare(id.Value)
		}
	}
	value := p.parseExpression(ASSIGN)
	if value == nil {
		return nil
	}
	if opTok.Type == lexer.ASSIGN {
		if sp, ok := value.(*ast.SplatExpression); ok {
			value = &ast.ArrayLiteral{Token: sp.Token, Elements: []ast.Expression{sp}}
		}
		return p.makeAssign(left, opTok, value)
	}
	op := strings.TrimSuffix(opTok.Literal, "=")
	switch op {
	case "||", "&&", "??":
		return &ast.LogicalAssignment{Token: opTok, Target: left, Operator: op, Value: value}
	}
	infixTok := lexer.Token{Type: lexer.TokenType(op), Literal: op, Line: opTok.Line, Column: opTok.Column}
	combined := &ast.InfixExpression{Token: infixTok, Left: left, Operator: op, Right: value}
	assignTok := lexer.Token{Type: lexer.ASSIGN, Literal: "=", Line: opTok.Line, Column: opTok.Column}
	return p.makeAssign(left, assignTok, combined)
}

func (p *Parser) makeAssign(left ast.Expression, tok lexer.Token, value ast.Expression) ast.Expression {
	switch t := left.(type) {
	case *ast.Identifier:
		return &ast.AssignmentExpression{Token: tok, Name: t, Value: value}
	case *ast.IndexExpression:
		return &ast.IndexAssignment{Token: t.Token, Left: t.Left, Index: t.Index, Value: value}
	case *ast.DotExpression:
		return &ast.DotAssignment{Token: t.Token, Left: t.Left, Field: t.Field.Value, Value: value}
	}
	p.addError("invalid assignment target")
	return nil
}

// --- prefix ----------------------------------------------------------------

func (p *Parser) parsePrefix() ast.Expression {
	tok := p.curToken
	switch tok.Type {
	case lexer.IDENT:
		return p.parseIdentifier()
	case lexer.INT:
		v, err := strconv.ParseInt(tok.Literal, 0, 64)
		if err != nil {
			p.addError(fmt.Sprintf("could not parse %q as integer", tok.Literal))
			return nil
		}
		return &ast.IntegerLiteral{Token: tok, Value: v}
	case lexer.FLOAT:
		v, err := strconv.ParseFloat(strings.ReplaceAll(tok.Literal, "_", ""), 64)
		if err != nil {
			p.addError(fmt.Sprintf("could not parse %q as float", tok.Literal))
			return nil
		}
		return &ast.FloatLiteral{Token: tok, Value: v}
	case lexer.STRING:
		return p.parseStringLiteral()
	case lexer.SYMBOL:
		return &ast.SymbolLiteral{Token: tok, Value: tok.Literal}
	case lexer.REGEX:
		return &ast.RegexLiteral{Token: tok, Pattern: tok.Literal, Flags: tok.Extra}
	case lexer.WORDS:
		return p.parseWordList()
	case lexer.TRUE, lexer.FALSE:
		return &ast.BooleanLiteral{Token: tok, Value: tok.Type == lexer.TRUE}
	case lexer.NIL:
		return &ast.NilLiteral{Token: tok}
	case lexer.IVAR:
		name := strings.TrimPrefix(tok.Literal, "@")
		self := &ast.Identifier{Token: lexer.Token{Type: lexer.SELF, Literal: "self", Line: tok.Line, Column: tok.Column}, Value: "self"}
		return &ast.DotExpression{Token: tok, Left: self, Field: &ast.Identifier{Token: tok, Value: name}}
	case lexer.CVAR, lexer.GVAR:
		return &ast.Identifier{Token: tok, Value: tok.Literal}
	case lexer.SELF:
		return &ast.Identifier{Token: tok, Value: "self"}
	case lexer.SUPER:
		return p.parseSuper()
	case lexer.BANG, lexer.MINUS, lexer.PLUS, lexer.BITNOT:
		p.nextToken()
		right := p.parseExpression(PREFIX)
		if right == nil {
			return nil
		}
		if tok.Type == lexer.MINUS && negateLeadingLiteral(right) {
			return right
		}
		return &ast.PrefixExpression{Token: tok, Operator: tok.Literal, Right: right}
	case lexer.KW_NOT:
		p.nextToken()
		right := p.parseExpression(KWLOGIC)
		if right == nil {
			return nil
		}
		bang := lexer.Token{Type: lexer.BANG, Literal: "!", Line: tok.Line, Column: tok.Column}
		return &ast.PrefixExpression{Token: bang, Operator: "!", Right: right}
	case lexer.ASTERISK:
		p.nextToken()
		v := p.parseExpression(PREFIX)
		if v == nil {
			return nil
		}
		return &ast.SplatExpression{Token: tok, Value: v}
	case lexer.POWER_OP:
		p.nextToken()
		v := p.parseExpression(PREFIX)
		if v == nil {
			return nil
		}
		return &ast.DoubleSplatExpression{Token: tok, Value: v}
	case lexer.BITAND:
		p.nextToken()
		v := p.parseExpression(PREFIX)
		if v == nil {
			return nil
		}
		return &ast.BlockPass{Token: tok, Value: v}
	case lexer.LPAREN:
		return p.parseGroupedExpression()
	case lexer.LBRACKET:
		return p.parseArrayLiteral()
	case lexer.LBRACE:
		return p.parseHashLiteral()
	case lexer.FUNCTION:
		return p.parseFunctionLiteral()
	case lexer.DEF:
		return p.parseDef()
	case lexer.ARROW:
		return p.parseLambda()
	case lexer.IF:
		return p.parseIfExpression()
	case lexer.UNLESS:
		return p.parseUnlessExpression()
	case lexer.CASE:
		return p.parseCaseExpression()
	case lexer.TRY:
		return p.parseTryExpression()
	case lexer.BEGIN:
		return p.parseBeginExpression()
	case lexer.CLASS:
		return p.parseClassLiteral(false)
	case lexer.MODULE:
		return p.parseClassLiteral(true)
	case lexer.YIELD:
		return p.parseYield()
	case lexer.ASSIGN:
		p.addError("unexpected assignment operator. Did you mean to use a variable name before '='?")
		return nil
	}
	p.addError(fmt.Sprintf("no prefix parse function for %s found", tok.Type))
	return nil
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	leave := p.enterGroup()
	defer leave()
	p.nextToken()
	exp := p.parseExpression(LOWEST)
	if exp == nil {
		return nil
	}
	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}
	return exp
}

func (p *Parser) parseWordList() ast.Expression {
	tok := p.curToken
	arr := &ast.ArrayLiteral{Token: tok, Elements: []ast.Expression{}}
	if tok.Literal == "" {
		return arr
	}
	for _, w := range strings.Split(tok.Literal, "\x00") {
		if tok.Extra == "i" {
			arr.Elements = append(arr.Elements, &ast.SymbolLiteral{Token: tok, Value: w})
		} else {
			arr.Elements = append(arr.Elements, &ast.StringLiteral{Token: tok, Value: w})
		}
	}
	return arr
}

func (p *Parser) parseSuper() ast.Expression {
	id := &ast.Identifier{Token: p.curToken, Value: "super"}
	if p.peekStartsCommandArg(false) {
		return p.finishCommandCall(id)
	}
	if p.peekTokenIs(lexer.DOT) || (p.peekTokenIs(lexer.LPAREN) && !p.peekToken.SpaceBefore) {
		return id
	}
	return &ast.CallExpression{Token: p.curToken, Function: id, Implicit: true}
}

func (p *Parser) parseYield() ast.Expression {
	y := &ast.YieldExpression{Token: p.curToken}
	switch {
	case p.peekTokenIs(lexer.LPAREN) && !p.peekToken.SpaceBefore:
		p.nextToken()
		y.Arguments = p.parseParenArgs()
		if y.Arguments == nil {
			return nil
		}
	case p.peekStartsCommandArg(false):
		y.Arguments = p.parseCommandArgs()
	}
	return y
}

// --- identifiers and calls --------------------------------------------------

func isUpper(s string) bool { return s != "" && s[0] >= 'A' && s[0] <= 'Z' }

func (p *Parser) parseIdentifier() ast.Expression {
	tok := p.curToken
	name := tok.Literal
	id := &ast.Identifier{Token: tok, Value: name}

	if name == "Range" && p.peekTokenIs(lexer.LPAREN) && !p.peekToken.SpaceBefore {
		p.nextToken()
		return p.parseRangeCall()
	}

	if p.peekTokenIs(lexer.LT) && !p.peekToken.SpaceBefore && !p.isLocal(name) {
		if args, ok := p.tryTypeArgs(); ok {
			if p.peekTokenIs(lexer.LPAREN) && !p.peekToken.SpaceBefore {
				p.nextToken()
				return p.finishCall(id, args)
			}
			if p.peekTokenIs(lexer.DOT) && isUpper(name) {
				p.nextToken()
				if p.peekToken.Literal != "new" {
					p.addError("expected 'new' after type arguments")
					return nil
				}
				p.nextToken()
				if !p.expectPeek(lexer.LPAREN) {
					return nil
				}
				return p.finishCall(id, args)
			}
			if isUpper(name) {
				return &ast.StructLiteral{Token: tok, Type: name, TypeArgs: args, Fields: map[string]ast.Expression{}}
			}
			p.addError("unexpected type arguments")
			return nil
		}
	}

	if p.isLocal(name) {
		return id
	}
	if p.peekStartsCommandArg(false) {
		return p.finishCommandCall(id)
	}
	id.NonLocal = !isUpper(name)
	return id
}

// peekStartsCommandArg reports whether the token after the current one begins
// an argument of a parenthesis-less call such as `puts "hi"`.
func (p *Parser) peekStartsCommandArg(dot bool) bool {
	t := p.peekToken
	if t.NLBefore || !t.SpaceBefore {
		return false
	}
	switch t.Type {
	case lexer.INT, lexer.FLOAT, lexer.STRING, lexer.SYMBOL, lexer.REGEX, lexer.WORDS, lexer.IDENT,
		lexer.IVAR, lexer.CVAR, lexer.GVAR, lexer.TRUE, lexer.FALSE, lexer.NIL, lexer.SELF,
		lexer.SUPER, lexer.FUNCTION, lexer.ARROW, lexer.KW_NOT:
		return true
	case lexer.DEF, lexer.YIELD:
		return !dot
	case lexer.LBRACKET, lexer.LPAREN:
		return !dot
	case lexer.MINUS, lexer.ASTERISK, lexer.POWER_OP, lexer.BITAND, lexer.BANG, lexer.BITNOT:
		if dot {
			return false
		}
		pp := p.peekPeekToken
		return !pp.SpaceBefore && !pp.NLBefore
	}
	return false
}

func (p *Parser) finishCommandCall(fn ast.Expression) ast.Expression {
	call := &ast.CallExpression{Token: p.curToken, Function: fn}
	call.Arguments = p.parseCommandArgs()
	if call.Arguments == nil {
		return nil
	}
	return call
}

// parseCommandArgs parses comma separated arguments without parentheses.
// The current token is the callee; the first argument is the peek token.
func (p *Parser) parseCommandArgs() []ast.Expression {
	p.noDo++
	defer func() { p.noDo-- }()
	args := []ast.Expression{}
	var rocket *ast.HashLiteral
	for {
		p.nextToken()
		arg := p.parseArgument(&rocket, ASSIGN)
		if arg == nil {
			return nil
		}
		if arg != skipArg {
			args = append(args, arg)
		}
		if p.peekTokenIs(lexer.COMMA) {
			p.nextToken()
			continue
		}
		break
	}
	if rocket != nil {
		args = append(args, rocket)
	}
	return args
}

// skipArg is returned by parseArgument for arguments that were folded into the trailing hash.
var skipArg ast.Expression = &ast.NilLiteral{}

// parseParenArgs parses `( args )`; the current token is the opening parenthesis.
func (p *Parser) parseParenArgs() []ast.Expression {
	leave := p.enterGroup()
	defer leave()
	args := []ast.Expression{}
	var rocket *ast.HashLiteral
	for {
		if p.peekTokenIs(lexer.RPAREN) {
			p.nextToken()
			break
		}
		p.nextToken()
		arg := p.parseArgument(&rocket, LOWEST)
		if arg == nil {
			return nil
		}
		if arg != skipArg {
			args = append(args, arg)
		}
		if p.peekTokenIs(lexer.COMMA) {
			p.nextToken()
			continue
		}
		if !p.expectPeek(lexer.RPAREN) {
			return nil
		}
		break
	}
	if rocket != nil {
		args = append(args, rocket)
	}
	return args
}

// peekIsLabel reports whether the peek token is the colon of a `name:` label
// (adjacent to the name and not the first half of `::`).
func (p *Parser) peekIsLabel() bool {
	if !p.peekTokenIs(lexer.COLON) || p.peekToken.SpaceBefore {
		return false
	}
	pp := p.peekPeekToken
	return !(pp.Type == lexer.COLON && !pp.SpaceBefore)
}

func isWordTok(t lexer.Token) bool {
	if t.Literal == "" {
		return false
	}
	c := t.Literal[0]
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// parseArgument parses one call argument: positional, name: value, key => value,
// *splat, **splat or &block.
func (p *Parser) parseArgument(rocket **ast.HashLiteral, prec int) ast.Expression {
	tok := p.curToken
	if (tok.Type == lexer.IDENT || isWordTok(tok) && tok.Type != lexer.STRING) && p.peekIsLabel() {
		p.nextToken()
		kw := &ast.KeywordArgument{Token: tok, Name: tok.Literal}
		if p.peekIn(lexer.COMMA, lexer.RPAREN) || (p.peekToken.NLBefore && p.nlIgnore == 0) {
			kw.Value = &ast.Identifier{Token: tok, Value: tok.Literal}
			return kw
		}
		p.nextToken()
		kw.Value = p.parseExpression(prec)
		if kw.Value == nil {
			return nil
		}
		return kw
	}
	if tok.Type == lexer.STRING && p.peekIsLabel() {
		p.nextToken()
		p.nextToken()
		v := p.parseExpression(ASSIGN)
		if v == nil {
			return nil
		}
		return &ast.KeywordArgument{Token: tok, Name: tok.Literal, Value: v}
	}
	expr := p.parseExpression(prec)
	if expr == nil {
		return nil
	}
	if p.peekTokenIs(lexer.FAT_ARROW) {
		p.nextToken()
		arrow := p.curToken
		p.nextToken()
		v := p.parseExpression(ASSIGN)
		if v == nil {
			return nil
		}
		if *rocket == nil {
			*rocket = &ast.HashLiteral{Token: arrow, Pairs: map[ast.Expression]ast.Expression{}}
		}
		(*rocket).Entries = append((*rocket).Entries, ast.HashEntry{Key: expr, Value: v})
		(*rocket).Pairs[expr] = v
		return skipArg
	}
	return expr
}

func (p *Parser) parseCallExpression(function ast.Expression) ast.Expression {
	return p.finishCall(function, nil)
}

// finishCall parses the argument list of a call; the current token is `(`.
func (p *Parser) finishCall(function ast.Expression, typeArgs []string) ast.Expression {
	tok := p.curToken
	args := p.parseParenArgs()
	if args == nil {
		return nil
	}
	if id, ok := function.(*ast.Identifier); ok && isUpper(id.Value) && allKeywordArgs(args) {
		if !(len(args) == 0 && p.peekTokenIs(lexer.LBRACE)) {
			lit := &ast.StructLiteral{Token: id.Token, Type: id.Value, TypeArgs: typeArgs, Fields: map[string]ast.Expression{}}
			for _, a := range args {
				kw := a.(*ast.KeywordArgument)
				lit.Fields[kw.Name] = kw.Value
				lit.FieldOrder = append(lit.FieldOrder, kw.Name)
			}
			return lit
		}
	}
	return &ast.CallExpression{Token: tok, Function: function, Arguments: args, TypeArgs: typeArgs}
}

func allKeywordArgs(args []ast.Expression) bool {
	for _, a := range args {
		if _, ok := a.(*ast.KeywordArgument); !ok {
			return false
		}
	}
	return true
}

func (p *Parser) parseRangeCall() ast.Expression {
	tok := p.curToken
	args := p.parseParenArgs()
	if len(args) < 2 || len(args) > 3 {
		p.addError("Range expects 2 or 3 arguments")
		return nil
	}
	expr := &ast.RangeCallExpression{Token: tok, Start: args[0], End: args[1]}
	if len(args) == 3 {
		b, ok := args[2].(*ast.BooleanLiteral)
		if !ok {
			p.addError("third argument to Range must be a boolean")
			return nil
		}
		expr.Exclusive = b.Value
	}
	return expr
}

// --- dot and index ---------------------------------------------------------

func (p *Parser) parseDotExpression(left ast.Expression) ast.Expression {
	dot := p.curToken
	safe := dot.Type == lexer.SAFE_NAV
	if p.peekTokenIs(lexer.LPAREN) {
		// callable.(args)
		p.nextToken()
		call := p.finishCall(left, nil)
		return call
	}
	p.nextToken()
	if !p.curTokenIs(lexer.IDENT) && !isWordTok(p.curToken) {
		p.addError(fmt.Sprintf("expected identifier after dot, got %s", p.curToken.Type))
		return nil
	}
	field := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	dotExpr := &ast.DotExpression{Token: dot, Left: left, Field: field, Safe: safe}
	if p.peekStartsCommandArg(true) && !isAssignTok(p.peekToken.Type) {
		call := &ast.CallExpression{Token: p.curToken, Function: dotExpr}
		call.Arguments = p.parseCommandArgs()
		if call.Arguments == nil {
			return nil
		}
		return call
	}
	return dotExpr
}

func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	tok := p.curToken
	leave := p.enterGroup()
	defer leave()
	p.nextToken()
	index := p.parseExpression(LOWEST)
	if index == nil {
		return nil
	}
	if p.peekTokenIs(lexer.COMMA) {
		// a[start, length] is sugar for a.slice(start, length)
		p.nextToken()
		p.nextToken()
		length := p.parseExpression(LOWEST)
		if length == nil || !p.expectPeek(lexer.RBRACKET) {
			return nil
		}
		fn := &ast.DotExpression{Token: tok, Left: left, Field: &ast.Identifier{Token: tok, Value: "slice"}}
		return &ast.CallExpression{Token: tok, Function: fn, Arguments: []ast.Expression{index, length}}
	}
	if !p.expectPeek(lexer.RBRACKET) {
		return nil
	}
	return &ast.IndexExpression{Token: tok, Left: left, Index: index}
}

// --- literals --------------------------------------------------------------

func (p *Parser) parseArrayLiteral() ast.Expression {
	arr := &ast.ArrayLiteral{Token: p.curToken, Elements: []ast.Expression{}}
	elems := p.parseListUntil(lexer.RBRACKET)
	if elems == nil {
		return nil
	}
	arr.Elements = elems
	return arr
}

// parseListUntil parses comma separated expressions up to the closing token.
// The current token is the opener.
func (p *Parser) parseListUntil(end lexer.TokenType) []ast.Expression {
	leave := p.enterGroup()
	defer leave()
	list := []ast.Expression{}
	for {
		if p.peekTokenIs(end) {
			p.nextToken()
			return list
		}
		p.nextToken()
		e := p.parseExpression(ASSIGN)
		if e == nil {
			return nil
		}
		list = append(list, e)
		if p.peekTokenIs(lexer.COMMA) {
			p.nextToken()
			continue
		}
		if !p.expectPeek(end) {
			return nil
		}
		return list
	}
}

func (p *Parser) parseHashLiteral() ast.Expression {
	hash := &ast.HashLiteral{Token: p.curToken, Pairs: map[ast.Expression]ast.Expression{}}
	leave := p.enterGroup()
	defer leave()
	for {
		if p.peekTokenIs(lexer.RBRACE) {
			p.nextToken()
			return hash
		}
		p.nextToken()
		tok := p.curToken
		switch {
		case tok.Type == lexer.POWER_OP:
			p.nextToken()
			v := p.parseExpression(PREFIX)
			if v == nil {
				return nil
			}
			hash.Entries = append(hash.Entries, ast.HashEntry{Value: v, Splat: true})
		case (tok.Type == lexer.IDENT || isWordTok(tok) && tok.Type != lexer.STRING) && p.peekIsLabel():
			p.nextToken()
			key := &ast.StringLiteral{Token: tok, Value: tok.Literal}
			var val ast.Expression
			if p.peekIn(lexer.COMMA, lexer.RBRACE) {
				val = &ast.Identifier{Token: tok, Value: tok.Literal}
			} else {
				p.nextToken()
				val = p.parseExpression(ASSIGN)
				if val == nil {
					return nil
				}
			}
			hash.Entries = append(hash.Entries, ast.HashEntry{Key: key, Value: val})
			hash.Pairs[key] = val
		default:
			key := p.parseExpression(TERNARY)
			if key == nil {
				return nil
			}
			if !p.peekIn(lexer.COLON, lexer.FAT_ARROW) {
				p.peekError(lexer.COLON)
				return nil
			}
			p.nextToken()
			p.nextToken()
			val := p.parseExpression(ASSIGN)
			if val == nil {
				return nil
			}
			hash.Entries = append(hash.Entries, ast.HashEntry{Key: key, Value: val})
			hash.Pairs[key] = val
		}
		if p.peekTokenIs(lexer.COMMA) {
			p.nextToken()
			continue
		}
		if !p.expectPeek(lexer.RBRACE) {
			return nil
		}
		return hash
	}
}

func (p *Parser) parseStringLiteral() ast.Expression {
	tok := p.curToken
	value := tok.Literal
	if !strings.Contains(value, "${") {
		return &ast.StringLiteral{Token: tok, Value: strings.ReplaceAll(value, "\x01", "$")}
	}
	interp := &ast.StringInterpolationLiteral{
		Token:       tok,
		Value:       strings.ReplaceAll(value, "\x01", "$"),
		Expressions: []ast.Expression{},
		Parts:       []string{},
	}
	rest := value
	for {
		start := strings.Index(rest, "${")
		if start < 0 {
			interp.Parts = append(interp.Parts, strings.ReplaceAll(rest, "\x01", "$"))
			break
		}
		interp.Parts = append(interp.Parts, strings.ReplaceAll(rest[:start], "\x01", "$"))
		end := matchingBrace(rest, start+2)
		if end < 0 {
			p.addError("unterminated string interpolation")
			break
		}
		expr := p.parseSnippet(rest[start+2 : end])
		if expr == nil {
			return nil
		}
		interp.Expressions = append(interp.Expressions, expr)
		rest = rest[end+1:]
	}
	if len(interp.Parts) == len(interp.Expressions) {
		interp.Parts = append(interp.Parts, "")
	}
	return interp
}

// matchingBrace returns the index of the `}` that closes an interpolation
// whose body starts at i, skipping nested braces and string literals.
func matchingBrace(s string, i int) int {
	depth := 1
	for ; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		case '"', '\'':
			q := s[i]
			for i++; i < len(s) && s[i] != q; i++ {
				if s[i] == '\\' {
					i++
				}
			}
		}
	}
	return -1
}

// parseSnippet parses the source text of an interpolation as a single expression.
func (p *Parser) parseSnippet(src string) ast.Expression {
	sub := New(lexer.New(src))
	sub.scopes = p.scopes
	stmts := sub.parseStatements()
	for _, e := range sub.errors {
		p.errors = append(p.errors, fmt.Sprintf("[%d:%d] in string interpolation: %s", p.curToken.Line, p.curToken.Column, e))
	}
	if len(sub.errors) > 0 {
		return nil
	}
	if len(stmts) == 1 {
		if es, ok := stmts[0].(*ast.ExpressionStatement); ok {
			return es.Expression
		}
	}
	p.addError(fmt.Sprintf("invalid expression %q in string interpolation", src))
	return nil
}

// --- generics --------------------------------------------------------------

// expectGT consumes a closing `>`, splitting a `>>` token when types are nested.
func (p *Parser) expectGT() bool {
	if p.peekTokenIs(lexer.RSHIFT) {
		t := p.peekToken
		gt := lexer.Token{Type: lexer.GT, Literal: ">", Line: t.Line, Column: t.Column}
		gt2 := lexer.Token{Type: lexer.GT, Literal: ">", Line: t.Line, Column: t.Column + 1}
		toks := make([]lexer.Token, 0, len(p.toks)+1)
		toks = append(toks, p.toks[:p.pos+1]...)
		toks = append(toks, gt, gt2)
		toks = append(toks, p.toks[p.pos+2:]...)
		p.toks = toks
		p.setPos(p.pos)
	}
	return p.expectPeek(lexer.GT)
}

// tryTypeArgs speculatively parses `<type, type>` after an identifier.
func (p *Parser) tryTypeArgs() ([]string, bool) {
	m := p.mark()
	p.nextToken() // <
	var args []string
	for {
		p.nextToken()
		t := p.parseType()
		if t == nil {
			p.reset(m)
			return nil, false
		}
		args = append(args, t.String())
		if p.peekTokenIs(lexer.COMMA) {
			p.nextToken()
			continue
		}
		break
	}
	if !p.expectGT() {
		p.reset(m)
		return nil, false
	}
	return args, true
}

// parseTypeParams parses `<T, U>` after a declaration name; the peek token is `<`.
func (p *Parser) parseTypeParams() ([]string, map[string]ast.Expression) {
	p.nextToken()
	var params []string
	var bounds map[string]ast.Expression
	for {
		if !p.expectPeek(lexer.IDENT) {
			return nil, nil
		}
		name := p.curToken.Literal
		params = append(params, name)
		if p.peekTokenIs(lexer.COLON) {
			p.nextToken()
			p.nextToken()
			bound := p.parseType()
			if bound == nil {
				return nil, nil
			}
			if bounds == nil {
				bounds = map[string]ast.Expression{}
			}
			bounds[name] = bound
		}
		if p.peekTokenIs(lexer.COMMA) {
			p.nextToken()
			continue
		}
		break
	}
	if !p.expectGT() {
		return nil, nil
	}
	return params, bounds
}

// --- types -----------------------------------------------------------------

func (p *Parser) parseType() ast.Expression {
	base := p.parseTypeAtom()
	if base == nil {
		return nil
	}
	if ta, ok := base.(*ast.TypeAnnotation); ok && len(ta.Name) > 1 && strings.HasSuffix(ta.Name, "?") {
		ta.Name = strings.TrimSuffix(ta.Name, "?")
		base = &ast.UnionTypeAnnotation{
			Token: ta.Token,
			Types: []ast.Expression{ta, &ast.TypeAnnotation{Token: ta.Token, Name: "nil"}},
		}
	}
	for p.peekTokenIs(lexer.QUESTION) && !p.peekToken.SpaceBefore {
		p.nextToken()
		base = &ast.UnionTypeAnnotation{
			Token: p.curToken,
			Types: []ast.Expression{base, &ast.TypeAnnotation{Token: p.curToken, Name: "nil"}},
		}
	}
	if p.noUnion == 0 && p.peekTokenIs(lexer.PIPE) {
		types := []ast.Expression{base}
		if u, ok := base.(*ast.UnionTypeAnnotation); ok {
			types = u.Types
		}
		tok := p.curToken
		for p.peekTokenIs(lexer.PIPE) {
			p.nextToken()
			p.nextToken()
			next := p.parseTypeAtom()
			if next == nil {
				return nil
			}
			if p.peekTokenIs(lexer.QUESTION) && !p.peekToken.SpaceBefore {
				p.nextToken()
				types = append(types, next, &ast.TypeAnnotation{Token: p.curToken, Name: "nil"})
				continue
			}
			types = append(types, next)
		}
		return &ast.UnionTypeAnnotation{Token: tok, Types: types}
	}
	return base
}

func (p *Parser) parseTypeAtom() ast.Expression {
	tok := p.curToken
	switch {
	case tok.Type == lexer.LBRACKET:
		return p.parseCompoundType()
	case tok.Type == lexer.LPAREN:
		var params []string
		var first ast.Expression
		if !p.peekTokenIs(lexer.RPAREN) {
			for {
				p.nextToken()
				t := p.parseType()
				if t == nil {
					return nil
				}
				if first == nil {
					first = t
				}
				params = append(params, t.String())
				if !p.peekTokenIs(lexer.COMMA) {
					break
				}
				p.nextToken()
			}
		}
		if !p.expectPeek(lexer.RPAREN) {
			return nil
		}
		if p.peekTokenIs(lexer.ARROW) {
			p.nextToken()
			p.nextToken()
			ret := p.parseType()
			if ret == nil {
				return nil
			}
			return &ast.TypeAnnotation{Token: tok, Name: "(" + strings.Join(params, ", ") + ") -> " + ret.String()}
		}
		if len(params) != 1 {
			p.addError("expected '->' after function type parameters")
			return nil
		}
		return p.typeDims(first)
	case tok.Type == lexer.NIL:
		return &ast.TypeAnnotation{Token: tok, Name: "nil"}
	case tok.Type == lexer.IDENT, tok.Type == lexer.FUNCTION, isWordTok(tok) && tok.Type != lexer.STRING:
		name := tok.Literal
		if p.peekTokenIs(lexer.LT) && !p.peekToken.SpaceBefore {
			p.nextToken()
			var args []string
			for {
				p.nextToken()
				t := p.parseType()
				if t == nil {
					return nil
				}
				args = append(args, t.String())
				if p.peekTokenIs(lexer.COMMA) {
					p.nextToken()
					continue
				}
				break
			}
			if !p.expectGT() {
				return nil
			}
			name += "<" + strings.Join(args, ", ") + ">"
		}
		return p.typeDims(&ast.TypeAnnotation{Token: tok, Name: name})
	}
	p.addError(fmt.Sprintf("expected type name, got %s", tok.Type))
	return nil
}

// typeDims appends `[]` array suffixes to a type.
func (p *Parser) typeDims(t ast.Expression) ast.Expression {
	dims := 0
	for p.peekTokenIs(lexer.LBRACKET) && p.peekPeekTokenIs(lexer.RBRACKET) {
		p.nextToken()
		p.nextToken()
		dims++
	}
	if dims == 0 {
		return t
	}
	suffix := strings.Repeat("[]", dims)
	if ta, ok := t.(*ast.TypeAnnotation); ok {
		ta.Name += suffix
		return ta
	}
	return &ast.TypeAnnotation{Token: p.curToken, Name: "(" + t.String() + ")" + suffix}
}

func (p *Parser) parseCompoundType() ast.Expression {
	ann := &ast.CompoundTypeAnnotation{Token: p.curToken, Types: []ast.Expression{}}
	p.noUnion++
	defer func() { p.noUnion-- }()
	for {
		if p.peekTokenIs(lexer.RBRACKET) {
			p.nextToken()
			break
		}
		p.nextToken()
		t := p.parseType()
		if t == nil {
			return nil
		}
		if ta, ok := t.(*ast.TypeAnnotation); ok {
			t = &ast.Identifier{Token: ta.Token, Value: ta.Name}
		}
		ann.Types = append(ann.Types, t)
		if p.peekTokenIs(lexer.COMMA) {
			p.nextToken()
			continue
		}
		if !p.expectPeek(lexer.RBRACKET) {
			p.addError("expected comma or closing bracket in compound type")
			return nil
		}
		break
	}
	dims := 0
	for p.peekTokenIs(lexer.LBRACKET) && p.peekPeekTokenIs(lexer.RBRACKET) {
		p.nextToken()
		p.nextToken()
		dims++
	}
	if dims == 0 {
		return ann
	}
	types := []string{}
	for _, t := range ann.Types {
		types = append(types, t.String())
	}
	return &ast.TypeAnnotation{
		Token:          ann.Token,
		Name:           "[" + strings.Join(types, ", ") + "]" + strings.Repeat("[]", dims),
		IsCompoundType: true,
		CompoundTypes:  types,
	}
}

// negateLeadingLiteral folds a unary minus into a numeric literal that is the
// receiver of a method chain (-5.abs is (-5).abs). It reports whether it did.
func negateLeadingLiteral(e ast.Expression) bool {
	for {
		switch x := e.(type) {
		case *ast.DotExpression:
			switch lit := x.Left.(type) {
			case *ast.IntegerLiteral:
				lit.Value = -lit.Value
				return true
			case *ast.FloatLiteral:
				lit.Value = -lit.Value
				return true
			}
			e = x.Left
		case *ast.CallExpression:
			e = x.Function
		default:
			return false
		}
	}
}
