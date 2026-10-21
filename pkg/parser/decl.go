package parser

import (
	"fmt"

	"github.com/vibe-lang/vibe/pkg/ast"
	"github.com/vibe-lang/vibe/pkg/lexer"
)

// noCloser marks a parameter list without a closing delimiter.
const noCloser lexer.TokenType = "NO_CLOSER"

// --- parameters --------------------------------------------------------------

type paramList struct {
	names      []*ast.Identifier
	types      []ast.Expression
	defaults   []ast.Expression
	variadic   bool
	blockParam *ast.Identifier
	kwRest     *ast.Identifier
	keyword    []bool
	prelude    []ast.Statement
}

// parseParamList parses parameters. The current token is the opener `(` or `|`
// and parsing ends on the matching closer. With bare set (a `def f a, b`
// style list) the current token is the token before the first parameter and
// parsing ends on the last parameter.
func (p *Parser) parseParamList(closer lexer.TokenType, bare bool) *paramList {
	pl := &paramList{}
	if !bare {
		leave := p.enterGroup()
		defer leave()
	}
	pipes := closer == lexer.PIPE
	if pipes {
		p.noUnion++
		defer func() { p.noUnion-- }()
	}
	for {
		if !bare && p.peekTokenIs(closer) {
			p.nextToken()
			return pl
		}
		p.nextToken()
		if !p.parseParam(pl, pipes) {
			return nil
		}
		if p.peekTokenIs(lexer.COMMA) {
			p.nextToken()
			continue
		}
		if !bare && !p.expectPeek(closer) {
			return nil
		}
		return pl
	}
}

func (p *Parser) parseParam(pl *paramList, inPipes bool) bool {
	tok := p.curToken
	var name *ast.Identifier
	switch tok.Type {
	case lexer.ASTERISK:
		pl.variadic = true
		name = &ast.Identifier{Token: tok, Value: "*"}
		if p.peekTokenIs(lexer.IDENT) && !p.peekToken.SpaceBefore {
			p.nextToken()
			name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		}
	case lexer.POWER_OP:
		kw := &ast.Identifier{Token: tok, Value: "**"}
		if p.peekTokenIs(lexer.IDENT) && !p.peekToken.SpaceBefore {
			p.nextToken()
			kw = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		}
		pl.kwRest = kw
		p.declare(kw.Value)
		return true
	case lexer.BITAND:
		if !p.expectPeek(lexer.IDENT) {
			return false
		}
		pl.blockParam = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		p.declare(pl.blockParam.Value)
		return true
	case lexer.LPAREN:
		synthetic := &ast.Identifier{Token: tok, Value: fmt.Sprintf("__arg%d", len(pl.names))}
		nested := &ast.DestructureAssignment{Token: tok, Value: synthetic}
		p.nextToken()
		nested.Targets = p.parseMultiTargets()
		if nested.Targets == nil || !p.expectPeek(lexer.RPAREN) {
			return false
		}
		nested.Names = identNames(nested.Targets)
		pl.prelude = append(pl.prelude, &ast.ExpressionStatement{Token: tok, Expression: nested})
		name = synthetic
	case lexer.IDENT:
		name = &ast.Identifier{Token: tok, Value: tok.Literal}
	default:
		p.addError(fmt.Sprintf("expected parameter name, got %s", tok.Type))
		return false
	}
	p.declare(name.Value)

	var typ, def ast.Expression
	isKw := false
	if p.peekIsLabel() && tok.Type == lexer.IDENT {
		switch {
		case p.peekPeekTokenIs(lexer.COMMA), p.peekPeekTokenIs(lexer.RPAREN), p.peekPeekTokenIs(lexer.PIPE),
			p.peekPeekToken.NLBefore, p.peekPeekTokenIs(lexer.EOF):
			p.nextToken()
			isKw = true
		case p.keywordDefaultFollows():
			p.nextToken()
			p.nextToken()
			if inPipes {
				def = p.parseExpression(BITOR)
			} else {
				def = p.parseExpression(ASSIGN)
			}
			if def == nil {
				return false
			}
			isKw = true
		}
	}
	if !isKw && p.peekTokenIs(lexer.COLON) {
		p.nextToken()
		p.nextToken()
		typ = p.parseType()
		if typ == nil {
			return false
		}
	}
	if !isKw && p.peekTokenIs(lexer.ASSIGN) {
		p.nextToken()
		p.nextToken()
		if inPipes {
			def = p.parseExpression(BITOR)
		} else {
			def = p.parseExpression(ASSIGN)
		}
		if def == nil {
			return false
		}
	}
	pl.keyword = append(pl.keyword, isKw)
	pl.names = append(pl.names, name)
	pl.types = append(pl.types, typ)
	pl.defaults = append(pl.defaults, def)
	return true
}

// keywordDefaultFollows reports whether `name: <literal>` is a keyword
// parameter with a default value rather than a typed parameter. The current
// token is the name and the peek token is the colon.
func (p *Parser) keywordDefaultFollows() bool {
	t := p.peekPeekToken
	switch t.Type {
	case lexer.INT, lexer.FLOAT, lexer.STRING, lexer.SYMBOL, lexer.TRUE, lexer.FALSE, lexer.NIL,
		lexer.LBRACE, lexer.MINUS, lexer.WORDS, lexer.REGEX, lexer.ARROW, lexer.FUNCTION, lexer.SELF,
		lexer.IVAR, lexer.GVAR, lexer.CVAR:
		return true
	case lexer.LBRACKET:
		return p.tok(p.pos+3).Type == lexer.RBRACKET && p.tok(p.pos+4).Type != lexer.LBRACKET
	}
	return false
}

func withPrelude(body *ast.BlockStatement, prelude []ast.Statement) *ast.BlockStatement {
	if len(prelude) > 0 && body != nil {
		body.Statements = append(append([]ast.Statement{}, prelude...), body.Statements...)
	}
	return body
}

// --- bodies ------------------------------------------------------------------

// parseBodyWithRescue parses a body that ends with `end` and may contain
// rescue/else/ensure clauses. The current token is the first body token; on
// return it is the closing `end`.
func (p *Parser) parseBodyWithRescue(what string) *ast.BlockStatement {
	start := p.curToken
	body := p.parseBlockUntil(lexer.END, lexer.RESCUE, lexer.ENSURE)
	if p.curIn(lexer.RESCUE, lexer.ENSURE) {
		try := p.finishRescue(&ast.TryExpression{Token: start, Body: body}, what)
		if try == nil {
			return nil
		}
		return &ast.BlockStatement{Token: start, Statements: []ast.Statement{
			&ast.ExpressionStatement{Token: start, Expression: try},
		}}
	}
	if !p.expectEnd(what) {
		return nil
	}
	return body
}

// finishRescue parses rescue/else/ensure clauses after a body; the current
// token is `rescue`, `ensure` or `end`. It leaves the current token on `end`.
func (p *Parser) finishRescue(try *ast.TryExpression, what string) *ast.TryExpression {
	for p.curTokenIs(lexer.RESCUE) {
		rc := &ast.RescueClause{Token: p.curToken}
		for !p.peekToken.NLBefore && !p.peekIn(lexer.FAT_ARROW, lexer.THEN, lexer.SEMICOLON, lexer.EOF) {
			p.nextToken()
			t := p.parseExpression(ASSIGN)
			if t == nil {
				return nil
			}
			rc.Types = append(rc.Types, t)
			if !p.peekTokenIs(lexer.COMMA) {
				break
			}
			p.nextToken()
		}
		if p.peekTokenIs(lexer.FAT_ARROW) {
			p.nextToken()
			if !p.expectPeek(lexer.IDENT) {
				return nil
			}
			rc.Var = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
			p.declare(rc.Var.Value)
		}
		p.startBody(lexer.THEN)
		rc.Body = p.parseBlockUntil(lexer.RESCUE, lexer.ELSE, lexer.ENSURE, lexer.END)
		try.Rescues = append(try.Rescues, rc)
	}
	if p.curTokenIs(lexer.ELSE) {
		p.startBodyFromCur()
		try.ElseBody = p.parseBlockUntil(lexer.ENSURE, lexer.END)
	}
	if p.curTokenIs(lexer.ENSURE) {
		p.startBodyFromCur()
		try.FinallyBody = p.parseBlockUntil(lexer.END)
	}
	if !p.expectEnd(what) {
		return nil
	}
	return try
}

// startBodyFromCur moves past a keyword (else, ensure) that is the current token.
func (p *Parser) startBodyFromCur() { p.nextToken() }

// --- functions ---------------------------------------------------------------

func (p *Parser) parseFunctionLiteral() ast.Expression {
	lit := &ast.FunctionLiteral{Token: p.curToken}
	p.pushScope()
	defer p.popScope()
	p.pushCtx(ctxFunc)
	defer p.popCtx()

	if p.peekTokenIs(lexer.IDENT) {
		p.nextToken()
		lit.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	}
	if p.peekTokenIs(lexer.LT) && !p.peekToken.SpaceBefore {
		if lit.TypeParams = p.parseTypeParams(); lit.TypeParams == nil {
			return nil
		}
	}
	var pl *paramList
	if p.peekTokenIs(lexer.LPAREN) {
		p.nextToken()
		if pl = p.parseParamList(lexer.RPAREN, false); pl == nil {
			return nil
		}
	} else {
		pl = &paramList{}
	}
	p.applyParams(lit, pl)
	if p.peekTokenIs(lexer.COLON) {
		p.nextToken()
		p.nextToken()
		if lit.ReturnType = p.parseType(); lit.ReturnType == nil {
			return nil
		}
	}
	switch {
	case p.peekTokenIs(lexer.LBRACE):
		p.nextToken()
		lit.Body = p.parseBraceBody()
	case p.peekTokenIs(lexer.DO):
		p.nextToken()
		p.nextToken()
		lit.Body = p.parseBodyWithRescue("function")
	default:
		p.peekError(lexer.LBRACE)
		return nil
	}
	if lit.Body == nil {
		return nil
	}
	withPrelude(lit.Body, pl.prelude)
	return lit
}

func (p *Parser) applyParams(lit *ast.FunctionLiteral, pl *paramList) {
	lit.Parameters = pl.names
	if lit.Parameters == nil {
		lit.Parameters = []*ast.Identifier{}
	}
	lit.ParamTypes = pl.types
	lit.ParamDefaults = pl.defaults
	lit.Variadic = pl.variadic
	lit.BlockParam = pl.blockParam
	lit.KwRest = pl.kwRest
	lit.Keyword = pl.keyword
}

// parseBraceBody parses `{ statements }`; the current token is `{`.
func (p *Parser) parseBraceBody() *ast.BlockStatement {
	tok := p.curToken
	p.nextToken()
	stmts := p.parseStatements(lexer.RBRACE)
	if !p.curTokenIs(lexer.RBRACE) {
		p.addError("expected '}' to close block")
		return nil
	}
	return &ast.BlockStatement{Token: tok, Statements: stmts}
}

var operatorMethodNames = map[lexer.TokenType]bool{
	lexer.PLUS: true, lexer.MINUS: true, lexer.ASTERISK: true, lexer.SLASH: true, lexer.MODULO: true,
	lexer.EQ: true, lexer.NOT_EQ: true, lexer.LT: true, lexer.GT: true, lexer.LTE: true, lexer.GTE: true,
	lexer.SPACESHIP: true, lexer.POWER_OP: true, lexer.LSHIFT: true, lexer.RSHIFT: true, lexer.BITAND: true,
	lexer.PIPE: true, lexer.BITXOR: true, lexer.BANG: true, lexer.BITNOT: true, lexer.MATCH: true,
}

// parseDefName reads the method name after `def`, including operator and
// setter names. The current token is `def`.
func (p *Parser) parseDefName(lit *ast.FunctionLiteral) bool {
	p.nextToken()
	tok := p.curToken
	if (tok.Type == lexer.SELF || tok.Type == lexer.IDENT) && p.peekTokenIs(lexer.DOT) {
		lit.Receiver = &ast.Identifier{Token: tok, Value: tok.Literal}
		if tok.Type == lexer.SELF {
			lit.Receiver.Value = "self"
		}
		p.nextToken()
		p.nextToken()
		tok = p.curToken
	}
	switch {
	case operatorMethodNames[tok.Type]:
		lit.Name = &ast.Identifier{Token: tok, Value: tok.Literal}
	case tok.Type == lexer.LBRACKET && p.peekTokenIs(lexer.RBRACKET):
		p.nextToken()
		name := "[]"
		if p.peekTokenIs(lexer.ASSIGN) && !p.peekToken.SpaceBefore {
			p.nextToken()
			name = "[]="
		}
		lit.Name = &ast.Identifier{Token: tok, Value: name}
	case tok.Type == lexer.IDENT || isWordTok(tok):
		name := tok.Literal
		if p.peekTokenIs(lexer.ASSIGN) && !p.peekToken.SpaceBefore && p.peekPeekTokenIs(lexer.LPAREN) {
			p.nextToken()
			name += "="
		}
		lit.Name = &ast.Identifier{Token: tok, Value: name}
	default:
		p.addError(fmt.Sprintf("expected method name after def, got %s", tok.Type))
		return false
	}
	return true
}

func (p *Parser) parseDef() ast.Expression {
	lit := &ast.FunctionLiteral{Token: p.curToken}
	if !p.parseDefName(lit) {
		return nil
	}
	savedScopes := p.scopes
	p.scopes = []map[string]bool{{}}
	defer func() { p.scopes = savedScopes }()
	p.pushCtx(ctxFunc)
	defer p.popCtx()

	if p.peekTokenIs(lexer.LT) && !p.peekToken.SpaceBefore {
		if lit.TypeParams = p.parseTypeParams(); lit.TypeParams == nil {
			return nil
		}
	}
	pl := &paramList{}
	switch {
	case p.peekTokenIs(lexer.LPAREN):
		p.nextToken()
		if pl = p.parseParamList(lexer.RPAREN, false); pl == nil {
			return nil
		}
	case p.peekTokenIs(lexer.IDENT) && !p.peekToken.NLBefore, p.peekIn(lexer.ASTERISK, lexer.POWER_OP, lexer.BITAND) && !p.peekToken.NLBefore:
		if pl = p.parseParamList(noCloser, true); pl == nil {
			return nil
		}
	}
	p.applyParams(lit, pl)
	if p.peekTokenIs(lexer.COLON) && !p.peekToken.NLBefore {
		p.nextToken()
		p.nextToken()
		if lit.ReturnType = p.parseType(); lit.ReturnType == nil {
			return nil
		}
	}
	if p.peekTokenIs(lexer.ASSIGN) && !p.peekToken.NLBefore {
		// endless method: def name(args) = expr
		p.nextToken()
		tok := p.curToken
		p.nextToken()
		e := p.parseExpression(LOWEST)
		if e == nil {
			return nil
		}
		lit.Body = &ast.BlockStatement{Token: tok, Statements: []ast.Statement{
			&ast.ExpressionStatement{Token: tok, Expression: e},
		}}
		withPrelude(lit.Body, pl.prelude)
		return lit
	}
	p.startBody()
	if lit.Body = p.parseBodyWithRescue("function definition"); lit.Body == nil {
		return nil
	}
	withPrelude(lit.Body, pl.prelude)
	return lit
}

// --- lambdas and blocks --------------------------------------------------------

func (p *Parser) parseLambda() ast.Expression {
	af := &ast.ArrowFunction{Token: p.curToken, Parameters: []*ast.Identifier{}}
	p.pushScope()
	defer p.popScope()
	p.pushCtx(ctxFunc)
	defer p.popCtx()
	var pl *paramList
	switch {
	case p.peekTokenIs(lexer.LPAREN):
		p.nextToken()
		if pl = p.parseParamList(lexer.RPAREN, false); pl == nil {
			return nil
		}
	case p.peekTokenIs(lexer.IDENT) && !p.peekToken.NLBefore:
		if pl = p.parseParamList(noCloser, true); pl == nil {
			return nil
		}
	default:
		pl = &paramList{}
	}
	p.applyArrowParams(af, pl)
	if p.peekTokenIs(lexer.COLON) {
		p.nextToken()
		p.nextToken()
		if af.ReturnType = p.parseType(); af.ReturnType == nil {
			return nil
		}
	}
	switch {
	case p.peekTokenIs(lexer.LBRACE):
		p.nextToken()
		af.Body = p.parseBraceBody()
	case p.peekTokenIs(lexer.DO):
		p.nextToken()
		p.nextToken()
		af.Body = p.parseBodyWithRescue("lambda")
	default:
		p.peekError(lexer.LBRACE)
		return nil
	}
	if af.Body == nil {
		return nil
	}
	withPrelude(af.Body, pl.prelude)
	return af
}

func (p *Parser) applyArrowParams(af *ast.ArrowFunction, pl *paramList) {
	if pl.names != nil {
		af.Parameters = pl.names
	}
	af.ParamTypes = pl.types
	af.ParamDefaults = pl.defaults
	af.Variadic = pl.variadic
	af.BlockParam = pl.blockParam
	af.KwRest = pl.kwRest
	af.Keyword = pl.keyword
}

func (p *Parser) canTakeBlock(left ast.Expression) bool {
	switch n := left.(type) {
	case *ast.CallExpression:
		return n.Block == nil
	case *ast.Identifier:
		return !p.isLocal(n.Value) && !isUpper(n.Value) && n.Value != "self"
	case *ast.DotExpression:
		return true
	}
	return false
}

func (p *Parser) attachBlock(left ast.Expression, blk *ast.ArrowFunction) ast.Expression {
	if blk == nil {
		return nil
	}
	switch n := left.(type) {
	case *ast.CallExpression:
		n.Block = blk
		return n
	}
	return &ast.CallExpression{Token: blk.Token, Function: left, Arguments: []ast.Expression{}, Block: blk}
}

// parseBlock parses a `{ |params| ... }` or `do |params| ... end` block; the
// current token is `{` or `do`.
func (p *Parser) parseBlock() *ast.ArrowFunction {
	af := &ast.ArrowFunction{Token: p.curToken, IsBlock: true, Parameters: []*ast.Identifier{}}
	brace := p.curTokenIs(lexer.LBRACE)
	p.pushScope()
	defer p.popScope()
	p.pushCtx(ctxFunc)
	defer p.popCtx()

	pl := &paramList{}
	switch {
	case p.peekTokenIs(lexer.PIPE):
		p.nextToken()
		if pl = p.parseParamList(lexer.PIPE, false); pl == nil {
			return nil
		}
	case p.peekTokenIs(lexer.OR):
		p.nextToken()
	}
	p.applyArrowParams(af, pl)
	if brace {
		af.Body = p.parseBraceBody()
	} else {
		p.startBody()
		af.Body = p.parseBodyWithRescue("do block")
	}
	if af.Body == nil {
		return nil
	}
	withPrelude(af.Body, pl.prelude)
	return af
}

// --- classes, structs, enums -----------------------------------------------------

func (p *Parser) parseClassLiteral(module bool) ast.Expression {
	lit := &ast.ClassLiteral{Token: p.curToken, IsModule: module}
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	lit.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	if p.peekTokenIs(lexer.LT) && p.peekPeekTokenIs(lexer.IDENT) {
		third := p.tok(p.pos + 3).Type
		if third == lexer.GT || third == lexer.COMMA || third == lexer.RSHIFT {
			if lit.TypeParams = p.parseTypeParams(); lit.TypeParams == nil {
				return nil
			}
		}
	}
	if p.peekTokenIs(lexer.LT) && !module {
		p.nextToken()
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		lit.Parent = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		if p.peekTokenIs(lexer.LT) && !p.peekToken.SpaceBefore {
			if _, ok := p.tryTypeArgs(); !ok {
				return nil
			}
		}
	}
	if !module && p.peekTokenIs(lexer.IDENT) && p.peekToken.Literal == "implements" {
		p.nextToken()
		for {
			if !p.expectPeek(lexer.IDENT) {
				return nil
			}
			lit.Implements = append(lit.Implements, p.curToken.Literal)
			if !p.peekTokenIs(lexer.COMMA) {
				break
			}
			p.nextToken()
		}
	}
	savedScopes := p.scopes
	p.scopes = []map[string]bool{{}}
	defer func() { p.scopes = savedScopes }()
	p.pushCtx(ctxFunc)
	defer p.popCtx()
	p.startBody()
	lit.Body = p.parseBlockUntil(lexer.END)
	if !p.expectEnd("class definition") {
		return nil
	}
	return lit
}

func (p *Parser) parseStructStatement() ast.Statement {
	stmt := &ast.StructStatement{Token: p.curToken}
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	if p.peekTokenIs(lexer.LT) {
		if stmt.TypeParams = p.parseTypeParams(); stmt.TypeParams == nil {
			return nil
		}
	}
	savedScopes := p.scopes
	p.scopes = []map[string]bool{{}}
	defer func() { p.scopes = savedScopes }()
	p.startBody()
	stmt.Fields = p.parseStatements(lexer.END)
	if !p.curTokenIs(lexer.END) {
		p.addError(fmt.Sprintf("expected 'end' to close struct definition, got %s", p.curToken.Type))
		return nil
	}
	return stmt
}

func (p *Parser) parseEnumStatement() ast.Statement {
	stmt := &ast.EnumStatement{Token: p.curToken}
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	p.nextToken()
	for !p.curTokenIs(lexer.END) && !p.curTokenIs(lexer.EOF) {
		if p.curTokenIs(lexer.IDENT) {
			stmt.Values = append(stmt.Values, p.curToken.Literal)
		}
		p.nextToken()
	}
	if !p.expectEnd("enum definition") {
		return nil
	}
	return stmt
}

func (p *Parser) parsePropStatement() ast.Statement {
	stmt := &ast.PropertyStatement{Token: p.curToken}
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	if p.peekTokenIs(lexer.COLON) {
		p.nextToken()
		p.nextToken()
		t := p.parseType()
		if t == nil {
			return nil
		}
		if ta, ok := t.(*ast.TypeAnnotation); ok {
			stmt.Type = ta
		} else {
			stmt.Type = &ast.TypeAnnotation{Token: stmt.Name.Token, Name: t.String()}
		}
	}
	return stmt
}

func (p *Parser) parseTypeAliasStatement() ast.Statement {
	stmt := &ast.TypeAliasStatement{Token: p.curToken}
	p.nextToken()
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	p.nextToken() // =
	p.nextToken()
	if stmt.Type = p.parseType(); stmt.Type == nil {
		return nil
	}
	return stmt
}

// parseInterfaceStatement parses
//
//	interface Name
//	  def method(a: int): string
//	end
func (p *Parser) parseInterfaceStatement() ast.Statement {
	stmt := &ast.InterfaceStatement{Token: p.curToken}
	p.nextToken()
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	p.nextToken()
	savedScopes := p.scopes
	defer func() { p.scopes = savedScopes }()
	for {
		for p.curTokenIs(lexer.SEMICOLON) {
			p.nextToken()
		}
		if p.curTokenIs(lexer.END) || p.curTokenIs(lexer.EOF) {
			break
		}
		if !p.curTokenIs(lexer.DEF) {
			p.addError(fmt.Sprintf("expected 'def' in interface, got %s", p.curToken.Type))
			return nil
		}
		sig := &ast.MethodSignature{}
		fn := &ast.FunctionLiteral{}
		if !p.parseDefName(fn) {
			return nil
		}
		sig.Name = fn.Name.Value
		p.scopes = []map[string]bool{{}}
		if p.peekTokenIs(lexer.LPAREN) {
			p.nextToken()
			pl := p.parseParamList(lexer.RPAREN, false)
			if pl == nil {
				return nil
			}
			sig.Parameters, sig.ParamTypes = pl.names, pl.types
		}
		if p.peekTokenIs(lexer.COLON) {
			p.nextToken()
			p.nextToken()
			if sig.ReturnType = p.parseType(); sig.ReturnType == nil {
				return nil
			}
		}
		stmt.Methods = append(stmt.Methods, sig)
		p.nextToken()
	}
	if !p.expectEnd("interface definition") {
		return nil
	}
	return stmt
}

// --- conditionals --------------------------------------------------------------

func (p *Parser) parseIfExpression() ast.Expression {
	expr := &ast.IfExpression{Token: p.curToken}
	p.nextToken()
	if expr.Condition = p.parseExpression(LOWEST); expr.Condition == nil {
		return nil
	}
	p.startBody(lexer.THEN)
	expr.Consequence = p.parseBlockUntil(lexer.ELSE, lexer.ELSIF, lexer.END)
	switch p.curToken.Type {
	case lexer.ELSIF:
		tok := p.curToken
		nested := p.parseIfExpression()
		if nested == nil {
			return nil
		}
		expr.Alternative = &ast.BlockStatement{Token: tok, Statements: []ast.Statement{
			&ast.ExpressionStatement{Token: tok, Expression: nested},
		}}
		return expr
	case lexer.ELSE:
		p.startBodyFromCur()
		expr.Alternative = p.parseBlockUntil(lexer.END)
	}
	if !p.expectEnd("if expression") {
		return nil
	}
	return expr
}

func (p *Parser) parseUnlessExpression() ast.Expression {
	expr := &ast.UnlessExpression{Token: p.curToken}
	p.nextToken()
	if expr.Condition = p.parseExpression(LOWEST); expr.Condition == nil {
		return nil
	}
	p.startBody(lexer.THEN)
	expr.Consequence = p.parseBlockUntil(lexer.ELSE, lexer.END)
	if p.curTokenIs(lexer.ELSE) {
		p.startBodyFromCur()
		expr.Alternative = p.parseBlockUntil(lexer.END)
	}
	if !p.expectEnd("unless expression") {
		return nil
	}
	return expr
}

func (p *Parser) parseCaseExpression() ast.Expression {
	expr := &ast.CaseExpression{Token: p.curToken}
	if !p.peekToken.NLBefore && !p.peekIn(lexer.SEMICOLON) {
		p.nextToken()
		if expr.Subject = p.parseExpression(LOWEST); expr.Subject == nil {
			return nil
		}
	}
	p.nextToken()
	for p.curTokenIs(lexer.SEMICOLON) {
		p.nextToken()
	}
	for {
		switch p.curToken.Type {
		case lexer.WHEN:
			when := &ast.WhenClause{Token: p.curToken}
			for {
				p.nextToken()
				v := p.parseExpression(ASSIGN)
				if v == nil {
					return nil
				}
				when.Values = append(when.Values, v)
				if !p.peekTokenIs(lexer.COMMA) {
					break
				}
				p.nextToken()
			}
			p.startBody(lexer.THEN)
			when.Body = p.parseBlockUntil(lexer.WHEN, lexer.IN, lexer.ELSE, lexer.END)
			expr.Whens = append(expr.Whens, when)
			continue
		case lexer.IN:
			clause := p.parseInClause()
			if clause == nil {
				return nil
			}
			expr.Ins = append(expr.Ins, clause)
			continue
		}
		break
	}
	if p.curTokenIs(lexer.ELSE) {
		p.startBodyFromCur()
		expr.Default = p.parseBlockUntil(lexer.END)
	}
	if !p.expectEnd("case expression") {
		return nil
	}
	return expr
}

// --- exceptions -----------------------------------------------------------------

func (p *Parser) parseTryExpression() ast.Expression {
	expr := &ast.TryExpression{Token: p.curToken}
	p.startBody()
	expr.Body = p.parseBlockUntil(lexer.CATCH, lexer.END)
	if p.curTokenIs(lexer.CATCH) {
		if p.peekTokenIs(lexer.IDENT) && !p.peekToken.NLBefore {
			p.nextToken()
			expr.CatchVar = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
			p.declare(expr.CatchVar.Value)
		}
		p.startBody()
		expr.CatchBody = p.parseBlockUntil(lexer.FINALLY, lexer.END)
	} else {
		p.addError("expected 'catch' in try expression")
		return nil
	}
	if p.curTokenIs(lexer.FINALLY) {
		p.startBodyFromCur()
		expr.FinallyBody = p.parseBlockUntil(lexer.END)
	}
	if !p.expectEnd("try expression") {
		return nil
	}
	return expr
}

func (p *Parser) parseBeginExpression() ast.Expression {
	expr := &ast.TryExpression{Token: p.curToken}
	p.startBody()
	expr.Body = p.parseBlockUntil(lexer.RESCUE, lexer.ELSE, lexer.ENSURE, lexer.END)
	return p.finishRescueExpr(expr)
}

func (p *Parser) finishRescueExpr(expr *ast.TryExpression) ast.Expression {
	if r := p.finishRescue(expr, "begin block"); r != nil {
		return r
	}
	return nil
}
