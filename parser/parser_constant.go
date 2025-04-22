package parser

import (
	"go-phpcs/ast"
	"go-phpcs/token"
)

// parseConstant parses a PHP constant declaration: const NAME = VALUE;
func (p *Parser) parseConstant() *ast.ConstantNode {
	pos := p.tok.Pos
	p.nextToken() // consume 'const'
	if p.tok.Type != token.T_STRING {
		p.addError("expected constant name after const, got %s", p.tok.Literal)
		return nil
	}
	name := p.tok.Literal
	p.nextToken() // consume name
	if p.tok.Type != token.T_ASSIGN {
		p.addError("expected '=' after constant name, got %s", p.tok.Literal)
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
		Name:  name,
		Value: value,
		Pos:   ast.Position(pos),
	}
}
