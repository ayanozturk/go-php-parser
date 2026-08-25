package parser

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/token"
)

// parseForStatement parses a PHP for-loop:
// for (init; cond; post) { ... }
func (p *Parser) parseForStatement() (ast.Node, error) {
	pos := p.tok.Pos
	p.nextToken() // consume 'for'

	if p.tok.Type != token.T_LPAREN {
		p.addError("line %d:%d: expected ( after for, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume '('
	init, ok := p.parseForExpressionList(token.T_SEMICOLON)
	if !ok {
		return nil, nil
	}
	p.nextToken() // consume first ';'
	conditions, ok := p.parseForExpressionList(token.T_SEMICOLON)
	if !ok {
		return nil, nil
	}
	p.nextToken() // consume second ';'
	updates, ok := p.parseForExpressionList(token.T_RPAREN)
	if !ok {
		return nil, nil
	}
	p.nextToken() // consume ')'

	if p.tok.Type == token.T_COLON {
		p.nextToken() // consume ':'
		body := p.parseAltBody(token.T_ENDFOR)
		if p.tok.Type != token.T_ENDFOR {
			p.addError("line %d:%d: expected endfor to close alternative for syntax, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
			return nil, nil
		}
		p.nextToken() // consume endfor
		if !p.consumeAltTerminator("endfor") {
			return nil, nil
		}
		return &ast.ForNode{Init: init, Conditions: conditions, Updates: updates, Body: body, Pos: ast.Position(pos)}, nil
	}

	// Body
	var body []ast.Node
	if p.tok.Type == token.T_LBRACE {
		p.nextToken() // consume '{'
		body = p.parseBlockStatement()
		if p.tok.Type != token.T_RBRACE {
			p.addError("line %d:%d: expected } to close for body, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
			return nil, nil
		}
		p.nextToken() // consume '}'
	} else {
		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		if stmt != nil {
			body = append(body, stmt)
		}
	}

	return &ast.ForNode{Init: init, Conditions: conditions, Updates: updates, Body: body, Pos: ast.Position(pos)}, nil
}

// parseForExpressionList parses one control clause and leaves its terminator
// as the current token. PHP permits every clause to be empty and permits
// multiple expressions separated by commas.
func (p *Parser) parseForExpressionList(terminator token.TokenType) ([]ast.Node, bool) {
	if p.tok.Type == terminator {
		return nil, true
	}

	var expressions []ast.Node
	for {
		expr := p.parseExpressionWithStop(token.T_COMMA, terminator)
		if expr == nil {
			p.addError("line %d:%d: expected expression in for control clause", p.tok.Pos.Line, p.tok.Pos.Column)
			return nil, false
		}
		expressions = append(expressions, expr)
		switch p.tok.Type {
		case token.T_COMMA:
			p.nextToken()
			if p.tok.Type == terminator {
				p.addError("line %d:%d: expected expression after comma in for control clause", p.tok.Pos.Line, p.tok.Pos.Column)
				return nil, false
			}
		case terminator:
			return expressions, true
		case token.T_EOF:
			p.addError("line %d:%d: unexpected EOF in for control clause", p.tok.Pos.Line, p.tok.Pos.Column)
			return nil, false
		default:
			p.addError("line %d:%d: expected comma or clause terminator in for control, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
			return nil, false
		}
	}
}
