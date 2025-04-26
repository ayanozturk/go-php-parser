package parser

import (
	"go-phpcs/ast"
	"go-phpcs/token"
)

// Parse the declare statement
func (p *Parser) parseDeclare() ast.Node {
	pos := p.tok.Pos
	declare := &ast.DeclareNode{
		Directives: make(map[string]ast.Node),
		Pos:        ast.Position(pos),
	}

	if !p.expect(token.T_LPAREN) {
		return nil
	}

	for {
		if p.tok.Type != token.T_STRING {
			p.addError("line %d:%d: expected directive name in declare(), got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
			return nil
		}
		key := p.tok.Literal
		p.nextToken()

		if !p.expect(token.T_ASSIGN) {
			return nil
		}

		value := p.parseExpression()
		if value == nil {
			return nil
		}
		declare.Directives[key] = value

		if p.tok.Type == token.T_COMMA {
			p.nextToken()
			continue
		}
		break
	}

	if !p.expect(token.T_RPAREN) {
		return nil
	}

	// Parse the statement or block after declare(...)
	stmt, _ := p.parseStatement()
	declare.Body = stmt

	return declare
}
