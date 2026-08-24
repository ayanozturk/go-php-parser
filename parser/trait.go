package parser

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/token"
)

// parseTraitDeclaration parses a PHP trait declaration
func (p *Parser) parseTraitDeclaration() (ast.Node, error) {
	pos := p.tok.Pos
	p.nextToken() // consume 'trait'

	if p.tok.Type != token.T_STRING {
		p.addError("line %d:%d: expected trait name, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	name := p.tok.Literal
	p.nextToken()

	// Skip trailing comments/whitespace before opening brace
	p.skipCommentsAndWhitespace()

	// Expect opening brace
	if p.tok.Type != token.T_LBRACE {
		p.addError("line %d:%d: expected { to start trait body for %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume {

	// Parse methods and constants inside the trait
	var body []ast.Node
	for p.tok.Type != token.T_RBRACE && p.tok.Type != token.T_EOF {
		modifiers := p.parseModifiers()
		var typeHint string
		if p.tok.Type == token.T_STRING || p.tok.Type == token.T_NS_SEPARATOR || p.tok.Type == token.T_CALLABLE || p.tok.Type == token.T_ARRAY || p.tok.Type == token.T_MIXED || p.tok.Type == token.T_QUESTION || p.tok.Type == token.T_TRUE || p.tok.Type == token.T_FALSE || p.tok.Type == token.T_NULL || p.tok.Type == token.T_STATIC {
			typeHint = p.parseTypeHint()
		}
		if p.tok.Type == token.T_FUNCTION {
			fn, err := p.parseFunction(modifiers)
			if err != nil {
				p.addError(err.Error())
				p.nextToken()
				continue
			}
			if fn != nil {
				body = append(body, fn)
			}
			continue
		}
		if p.tok.Type == token.T_CONST {
			for _, constant := range p.parseConstantWithModifiers(modifiers) {
				body = append(body, constant)
			}
			continue
		}
		if p.tok.Type == token.T_USE {
			// A trait can itself compose other traits, e.g.
			// "trait T { use OtherTrait; }".
			if traitUse := p.parseTraitUseStatement(); traitUse != nil {
				body = append(body, traitUse)
			}
			continue
		}
		if p.tok.Type == token.T_VARIABLE {
			if prop, err := p.parsePropertyDeclaration(modifiers, typeHint); prop != nil {
				body = append(body, prop)
			} else if err != nil {
				p.addError(err.Error())
				p.nextToken()
			}
			continue
		}
		if len(modifiers) > 0 || typeHint != "" {
			p.addError("line %d:%d: expected property or function after modifiers/type in trait %s body, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
			p.nextToken()
			continue
		}
		// Skip unexpected tokens inside trait body
		p.addError("line %d:%d: unexpected token %s in trait %s body", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal, name)
		p.nextToken()
	}

	if p.tok.Type != token.T_RBRACE {
		p.addError("line %d:%d: expected } to close trait %s body, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume }

	return &ast.TraitNode{
		Name: &ast.Identifier{Name: name, Pos: ast.Position(pos)},
		Body: body,
		Pos:  ast.Position(pos),
	}, nil
}
