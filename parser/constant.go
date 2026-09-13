package parser

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/token"
)

// parseConstant parses a PHP constant declaration.
// Supports both legacy `const NAME = VALUE;` and typed `const TYPE NAME = VALUE;` forms.
func (p *Parser) parseConstant() []*ast.ConstantNode {
	return p.parseConstantWithModifiers(nil)
}

func (p *Parser) parseConstantWithModifiers(modifiers ast.ModifierList) []*ast.ConstantNode {
	if len(modifiers) == 0 && (p.tok.Type == token.T_PUBLIC || p.tok.Type == token.T_PROTECTED || p.tok.Type == token.T_PRIVATE) {
		modifiers = ast.ModifierList{{Tok: p.tok.Type, Text: p.tok.Literal}}
		p.nextToken()
	}
	if p.tok.Type != token.T_CONST {
		p.addError("expected 'const' after visibility, got %s", p.tok.Literal)
		return nil
	}
	phpdoc := p.consumeCurrentDoc(p.tok.Pos)
	p.nextToken() // consume 'const'
	var typeNode ast.Node
	if isConstTypeToken(p.tok.Type) {
		// Speculatively parse a (possibly union/intersection) type; only
		// keep it if a constant name (T_STRING) follows, e.g.
		// "const string|int BAR = 'bar';" vs. plain "const BAR = 1;"
		// (where "BAR" itself would otherwise look like a type token).
		cp := p.checkpoint()
		candidate := p.parseTypeNode()
		if p.tok.Type == token.T_STRING {
			typeNode = candidate
		} else {
			p.restore(cp)
		}
	}
	var constants []*ast.ConstantNode
	for {
		p.skipCommentsAndWhitespace()
		pos := p.tok.Pos
		if !isConstantNameToken(p.tok.Type) {
			p.addError("expected constant name after const, got %s", p.tok.Literal)
			return constants
		}
		name := p.tok.Literal
		p.nextToken() // consume name
		itemType := typeNode
		if p.tok.Type == token.T_COLON {
			p.nextToken() // consume ':'
			// Parse type (simple identifier or namespaced)
			if p.tok.Type == token.T_STRING {
				itemType = &ast.IdentifierNode{Value: p.tok.Literal, Pos: ast.Position(p.tok.Pos)}
				p.nextToken()
			}
		}
		if p.tok.Type != token.T_ASSIGN {
			p.addError("expected '=' after constant name/type, got %s", p.tok.Literal)
			return constants
		}
		p.nextToken() // consume '='
		value := p.parseExpression()
		if _, ok := value.(*ast.HeredocNode); ok && p.tok.Type != token.T_SEMICOLON && p.tok.Type != token.T_COMMA {
			p.nextToken()
		}
		constants = append(constants, &ast.ConstantNode{
			Name:      name,
			Type:      itemType,
			Modifiers: append(ast.ModifierList(nil), modifiers...),
			Value:     value,
			PHPDoc:    phpdoc,
			Pos:       ast.Position(pos),
		})
		if p.tok.Type == token.T_COMMA {
			p.nextToken() // consume ',' and parse the next grouped constant
			continue
		}
		break
	}
	if p.tok.Type != token.T_SEMICOLON {
		p.addError("expected ';' after constant value, got %s", p.tok.Literal)
		return constants
	}
	p.nextToken() // consume ';'
	for _, constant := range constants {
		constant.SetEndPos(ast.Position(p.prevTokEnd))
	}
	return constants
}

func isConstantNameToken(tokenType token.TokenType) bool {
	return tokenType == token.T_STRING || tokenType == token.T_AS
}

func isConstTypeToken(tokenType token.TokenType) bool {
	return isNativeTypeStart(token.Token{Type: tokenType})
}
