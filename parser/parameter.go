package parser

import (
	"fmt"
	"go-phpcs/ast"
	"go-phpcs/token"
)

// parseParameter parses a function or method parameter
func (p *Parser) parseParameter() ast.Node {
	pos := p.tok.Pos

	// PHP8 constructor property promotion: check for visibility modifier
	var visibility string
	var isPromoted bool
	if p.tok.Type == token.T_PUBLIC || p.tok.Type == token.T_PROTECTED || p.tok.Type == token.T_PRIVATE {
		visibility = p.tok.Literal
		isPromoted = true
		p.nextToken() // consume visibility
	}

	// Parse type hint if present (support nullable type: ?Bar)
	var typeHint string
	if p.tok.Type == token.T_QUESTION {
		typeHint = "?"
		p.nextToken() // consume ?
	}
	if p.tok.Type == token.T_STRING || p.tok.Type == token.T_ARRAY {
		typeHint += p.tok.Literal
		p.nextToken()

		// Handle array type with []
		if p.tok.Type == token.T_LBRACKET {
			typeHint += "[]"
			p.nextToken() // consume [
			if p.tok.Type != token.T_RBRACKET {
				p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected ']' after array type %s[], got %s", p.tok.Pos.Line, p.tok.Pos.Column, typeHint, p.tok.Literal))
				return nil
			}
			p.nextToken() // consume ]
		}
	}

	// Parse variable name
	if p.tok.Type != token.T_VARIABLE {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected variable name in parameter, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
		return nil
	}
	name := p.tok.Literal[1:] // Remove $ prefix
	p.nextToken()

	// Handle default value if present
	var defaultValue ast.Node
	if p.tok.Type == token.T_ASSIGN {
		p.nextToken() // consume =
		defaultValue = p.parseExpression()
	}

	return &ast.ParameterNode{
		Name:         name,
		TypeHint:     typeHint,
		DefaultValue: defaultValue,
		Visibility:   visibility,
		IsPromoted:   isPromoted,
		Pos:          ast.Position(pos),
	}
}
