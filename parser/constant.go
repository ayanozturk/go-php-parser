package parser

import (
	"go-phpcs/ast"
	"go-phpcs/token"
)

// parseConstant parses a PHP constant declaration: [visibility] const NAME [: TYPE] = VALUE;
func (p *Parser) parseConstant() *ast.ConstantNode {
	pos := p.tok.Pos
	visibility := ""
	// Check for visibility modifier
	if p.tok.Type == token.T_PUBLIC || p.tok.Type == token.T_PROTECTED || p.tok.Type == token.T_PRIVATE {
		visibility = p.tok.Literal
		p.nextToken()
	}
	if p.tok.Type != token.T_CONST {
		p.addError("expected 'const' after visibility, got %s", p.tok.Literal)
		return nil
	}
	p.nextToken() // consume 'const'
	if p.tok.Type != token.T_STRING {
		p.addError("expected constant name after const, got %s", p.tok.Literal)
		return nil
	}
	name := p.tok.Literal
	p.nextToken() // consume name
	typeStr := ""
	if p.tok.Type == token.T_COLON {
		p.nextToken() // consume ':'
		// Parse type (simple identifier or namespaced)
		if p.tok.Type == token.T_STRING {
			typeStr = p.tok.Literal
			p.nextToken()
		}
	}
	if p.tok.Type != token.T_ASSIGN {
		p.addError("expected '=' after constant name/type, got %s", p.tok.Literal)
		return nil
	}
	p.nextToken() // consume '='
	value := p.parseExpression()
	if p.tok.Type != token.T_SEMICOLON {
		p.addError("expected ';' after constant value, got %s", p.tok.Literal)
		return nil
	}
	p.nextToken() // consume ';'
	return &ast.ConstantNode{
		Name:       name,
		Type:       typeStr,
		Visibility: visibility,
		Value:      value,
		Pos:        ast.Position(pos),
	}
}
