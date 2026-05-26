package parser

import (
	"go-phpcs/ast"
	"go-phpcs/token"
)

// parseConstant parses a PHP constant declaration.
// Supports both legacy `const NAME = VALUE;` and typed `const TYPE NAME = VALUE;` forms.
func (p *Parser) parseConstant() *ast.ConstantNode {
	return p.parseConstantWithModifiers(nil)
}

func (p *Parser) parseConstantWithModifiers(modifiers []string) *ast.ConstantNode {
	pos := p.tok.Pos
	visibility := visibilityFromModifiers(modifiers)
	if len(modifiers) == 0 && (p.tok.Type == token.T_PUBLIC || p.tok.Type == token.T_PROTECTED || p.tok.Type == token.T_PRIVATE) {
		visibility = p.tok.Literal
		modifiers = append(modifiers, p.tok.Literal)
		p.nextToken()
	}
	if p.tok.Type != token.T_CONST {
		p.addError("expected 'const' after visibility, got %s", p.tok.Literal)
		return nil
	}
	p.nextToken() // consume 'const'
	typeStr := ""
	if isConstTypeToken(p.tok.Type) && p.peekToken().Type == token.T_STRING {
		typeStr = p.tok.Literal
		p.nextToken()
	}
	if p.tok.Type != token.T_STRING {
		p.addError("expected constant name after const, got %s", p.tok.Literal)
		return nil
	}
	name := p.tok.Literal
	p.nextToken() // consume name
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
	if _, ok := value.(*ast.HeredocNode); ok && p.tok.Type != token.T_SEMICOLON {
		p.nextToken()
	}
	if p.tok.Type != token.T_SEMICOLON {
		p.addError("expected ';' after constant value, got %s", p.tok.Literal)
		return nil
	}
	p.nextToken() // consume ';'
	return &ast.ConstantNode{
		Name:       name,
		Type:       typeStr,
		Visibility: visibility,
		Modifiers:  append([]string(nil), modifiers...),
		Value:      value,
		Pos:        ast.Position(pos),
	}
}

func isConstTypeToken(tokenType token.TokenType) bool {
	switch tokenType {
	case token.T_ARRAY, token.T_CALLABLE, token.T_STRING:
		return true
	default:
		return false
	}
}

func visibilityFromModifiers(modifiers []string) string {
	for _, modifier := range modifiers {
		switch modifier {
		case "public", "protected", "private":
			return modifier
		}
	}
	return ""
}
