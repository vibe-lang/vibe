package parser

import (
	"fmt"

	"github.com/vibe-lang/vibe/pkg/ast"
	"github.com/vibe-lang/vibe/pkg/lexer"
)

// parseInClause parses `in pattern [if guard] [then] body`; the current token is `in`.
func (p *Parser) parseInClause() *ast.InClause {
	clause := &ast.InClause{Token: p.curToken}
	p.nextToken()
	if clause.Pattern = p.parseTopPattern(); clause.Pattern == nil {
		return nil
	}
	if p.peekIn(lexer.IF, lexer.UNLESS) && !p.peekToken.NLBefore {
		p.nextToken()
		clause.Unless = p.curTokenIs(lexer.UNLESS)
		p.nextToken()
		if clause.Guard = p.parseExpression(LOWEST); clause.Guard == nil {
			return nil
		}
	}
	p.startBody(lexer.THEN)
	clause.Body = p.parseBlockUntil(lexer.WHEN, lexer.IN, lexer.ELSE, lexer.END)
	return clause
}

func (p *Parser) isHashKeyStart() bool {
	t := p.curToken
	return (t.Type == lexer.IDENT || t.Type == lexer.STRING || isWordTok(t)) &&
		p.peekIsLabel()
}

func (p *Parser) patternEnds() bool {
	t := p.peekToken
	return t.NLBefore || t.Type == lexer.THEN || t.Type == lexer.IF || t.Type == lexer.UNLESS ||
		t.Type == lexer.SEMICOLON || t.Type == lexer.EOF
}

// parseTopPattern parses a pattern where brackets and braces may be omitted:
// `a, *rest` and `name:, age:`.
func (p *Parser) parseTopPattern() ast.Expression {
	tok := p.curToken
	if p.isHashKeyStart() || p.curTokenIs(lexer.POWER_OP) {
		return p.parseHashPatternBody(tok, nil, "")
	}
	if p.curTokenIs(lexer.ASTERISK) {
		return p.parseArrayPatternBody(tok, nil, "")
	}
	first := p.parsePattern()
	if first == nil {
		return nil
	}
	if p.peekTokenIs(lexer.COMMA) {
		arr := &ast.ArrayPattern{Token: tok, Elements: []ast.Expression{first}, RestIdx: -1}
		for p.peekTokenIs(lexer.COMMA) {
			p.nextToken()
			p.nextToken()
			if !p.parseArrayElement(arr) {
				return nil
			}
		}
		return arr
	}
	return first
}

// parsePattern parses alternatives and an optional `=> name` binding.
func (p *Parser) parsePattern() ast.Expression {
	tok := p.curToken
	first := p.parsePrimaryPattern()
	if first == nil {
		return nil
	}
	result := first
	if p.peekTokenIs(lexer.PIPE) {
		alt := &ast.AltPattern{Token: tok, Alts: []ast.Expression{first}}
		for p.peekTokenIs(lexer.PIPE) {
			p.nextToken()
			p.nextToken()
			next := p.parsePrimaryPattern()
			if next == nil {
				return nil
			}
			alt.Alts = append(alt.Alts, next)
		}
		result = alt
	}
	if p.peekTokenIs(lexer.FAT_ARROW) {
		p.nextToken()
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		name := p.curToken.Literal
		p.declare(name)
		result = &ast.CapturePattern{Token: tok, Pattern: result, Name: name}
	}
	return result
}

func (p *Parser) rangeEndFollows() bool {
	t := p.peekToken
	if t.NLBefore {
		return false
	}
	switch t.Type {
	case lexer.INT, lexer.FLOAT, lexer.STRING, lexer.MINUS, lexer.SYMBOL:
		return true
	}
	return false
}

func (p *Parser) parsePrimaryPattern() ast.Expression {
	tok := p.curToken
	switch tok.Type {
	case lexer.LBRACKET:
		return p.parseArrayPatternBody(tok, nil, "]")
	case lexer.LBRACE:
		return p.parseHashPatternBody(tok, nil, "}")
	case lexer.LPAREN:
		p.nextToken()
		inner := p.parsePattern()
		if inner == nil || !p.expectPeek(lexer.RPAREN) {
			return nil
		}
		return inner
	case lexer.BITXOR:
		p.nextToken()
		if p.curTokenIs(lexer.LPAREN) {
			p.nextToken()
			e := p.parseExpression(LOWEST)
			if e == nil || !p.expectPeek(lexer.RPAREN) {
				return nil
			}
			return &ast.PinPattern{Token: tok, Value: e}
		}
		switch p.curToken.Type {
		case lexer.IDENT, lexer.IVAR, lexer.GVAR, lexer.CVAR:
			return &ast.PinPattern{Token: tok, Value: &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}}
		}
		p.addError("expected a variable or parenthesized expression after '^'")
		return nil
	case lexer.DOTDOT, lexer.DOTDOTDOT:
		p.nextToken()
		end := p.parseExpression(BITOR)
		if end == nil {
			return nil
		}
		return &ast.ValuePattern{Token: tok, Value: &ast.RangeExpression{Token: tok, End: end, Exclusive: tok.Type == lexer.DOTDOTDOT}}
	case lexer.IDENT:
		if isUpper(tok.Literal) {
			return p.parseConstPattern()
		}
		if p.peekTokenIs(lexer.DOT) || (p.peekTokenIs(lexer.LPAREN) && !p.peekToken.SpaceBefore) {
			break
		}
		p.declare(tok.Literal)
		return &ast.BindPattern{Token: tok, Name: tok.Literal}
	case lexer.NIL:
		return &ast.ValuePattern{Token: tok, Value: &ast.NilLiteral{Token: tok}}
	}
	p.noAssign++
	v := p.parseExpression(BITOR)
	p.noAssign--
	if v == nil {
		return nil
	}
	if p.peekIn(lexer.DOTDOT, lexer.DOTDOTDOT) && !p.peekToken.NLBefore {
		p.nextToken()
		r := &ast.RangeExpression{Token: p.curToken, Start: v, Exclusive: p.curTokenIs(lexer.DOTDOTDOT)}
		if p.rangeEndFollows() {
			p.nextToken()
			if r.End = p.parseExpression(BITOR); r.End == nil {
				return nil
			}
		}
		v = r
	}
	return &ast.ValuePattern{Token: tok, Value: v}
}

// parseConstPattern handles `Const`, `Const(...)` and `Const[...]`.
func (p *Parser) parseConstPattern() ast.Expression {
	tok := p.curToken
	class := &ast.Identifier{Token: tok, Value: tok.Literal}
	if p.peekTokenIs(lexer.LPAREN) && !p.peekToken.SpaceBefore {
		p.nextToken()
		if p.peekTokenIs(lexer.RPAREN) {
			p.nextToken()
			return &ast.ArrayPattern{Token: tok, Class: class, RestIdx: -1}
		}
		p.nextToken()
		if p.isHashKeyStart() || p.curTokenIs(lexer.POWER_OP) {
			return p.parseHashPatternBody(tok, class, ")")
		}
		return p.parseArrayPatternBody(tok, class, ")")
	}
	if p.peekTokenIs(lexer.LBRACKET) && !p.peekToken.SpaceBefore {
		p.nextToken()
		return p.parseArrayPatternBody(tok, class, "]")
	}
	return &ast.ValuePattern{Token: tok, Value: class}
}

func (p *Parser) parseArrayElement(arr *ast.ArrayPattern) bool {
	if p.curTokenIs(lexer.ASTERISK) {
		if arr.RestIdx >= 0 {
			p.addError("only one splat is allowed in an array pattern")
			return false
		}
		arr.RestIdx = len(arr.Elements)
		if p.peekTokenIs(lexer.IDENT) && !p.peekToken.NLBefore {
			p.nextToken()
			arr.RestName = p.curToken.Literal
			p.declare(arr.RestName)
		}
		arr.Elements = append(arr.Elements, nil)
		return true
	}
	el := p.parsePattern()
	if el == nil {
		return false
	}
	arr.Elements = append(arr.Elements, el)
	return true
}

// parseArrayPatternBody parses array pattern elements. For bracketed forms the
// current token is the opener (or the first element when class is set and
// closer is ")"); closer "" means a braceless pattern starting at the current token.
func (p *Parser) parseArrayPatternBody(tok lexer.Token, class ast.Expression, closer string) ast.Expression {
	arr := &ast.ArrayPattern{Token: tok, Class: class, RestIdx: -1}
	if closer == "" {
		for {
			if !p.parseArrayElement(arr) {
				return nil
			}
			if !p.peekTokenIs(lexer.COMMA) {
				return arr
			}
			p.nextToken()
			p.nextToken()
		}
	}
	var end lexer.TokenType = lexer.RBRACKET
	if closer == ")" {
		end = lexer.RPAREN
	}
	leave := p.enterGroup()
	defer leave()
	if closer == ")" {
		// current token is the first element
		for {
			if !p.parseArrayElement(arr) {
				return nil
			}
			if p.peekTokenIs(lexer.COMMA) {
				p.nextToken()
				p.nextToken()
				continue
			}
			if !p.expectPeek(end) {
				return nil
			}
			return arr
		}
	}
	for {
		if p.peekTokenIs(end) {
			p.nextToken()
			return arr
		}
		p.nextToken()
		if !p.parseArrayElement(arr) {
			return nil
		}
		if p.peekTokenIs(lexer.COMMA) {
			p.nextToken()
			continue
		}
		if !p.expectPeek(end) {
			return nil
		}
		return arr
	}
}

// parseHashPatternBody parses `key: pattern` entries. closer is "}" or ")"
// for delimited forms (current token is the opener, or the first key when the
// closer is ")"), or "" for a braceless pattern starting at the current token.
func (p *Parser) parseHashPatternBody(tok lexer.Token, class ast.Expression, closer string) ast.Expression {
	hp := &ast.HashPattern{Token: tok, Class: class}
	var end lexer.TokenType
	switch closer {
	case "}":
		end = lexer.RBRACE
	case ")":
		end = lexer.RPAREN
	}
	if closer != "" {
		leave := p.enterGroup()
		defer leave()
	}
	first := closer == "" || closer == ")"
	for {
		if closer == "}" || !first {
			if closer != "" && p.peekTokenIs(end) {
				p.nextToken()
				return hp
			}
			p.nextToken()
		}
		first = false
		switch {
		case p.curTokenIs(lexer.POWER_OP):
			if p.peekTokenIs(lexer.NIL) {
				p.nextToken()
				hp.Exact = true
			} else if p.peekTokenIs(lexer.IDENT) {
				p.nextToken()
				hp.HasRest = true
				hp.RestName = p.curToken.Literal
				p.declare(hp.RestName)
			} else {
				hp.HasRest = true
			}
		case p.isHashKeyStart():
			key := p.curToken.Literal
			p.nextToken() // colon
			hp.Keys = append(hp.Keys, key)
			noValue := p.peekIn(lexer.COMMA, lexer.RBRACE, lexer.RPAREN) || p.patternEnds()
			if noValue {
				p.declare(key)
				hp.Patterns = append(hp.Patterns, nil)
			} else {
				p.nextToken()
				sub := p.parsePattern()
				if sub == nil {
					return nil
				}
				hp.Patterns = append(hp.Patterns, sub)
			}
		default:
			p.addError(fmt.Sprintf("expected key in hash pattern, got %s", p.curToken.Type))
			return nil
		}
		if p.peekTokenIs(lexer.COMMA) {
			p.nextToken()
			if closer == "" {
				p.nextToken()
				first = true
			}
			continue
		}
		if closer == "" {
			return hp
		}
		if !p.expectPeek(end) {
			return nil
		}
		return hp
	}
}
