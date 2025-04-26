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
		if p.tok.Type == token.T_RPAREN {
			break
		}

		if p.tok.Type == token.T_STRING {
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
		} else {
			p.nextToken()
		}
	}

	if !p.expect(token.T_RPAREN) {
		return nil
	}

	return declare
}
