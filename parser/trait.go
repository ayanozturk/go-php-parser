package parser

import (
	"go-phpcs/ast"
	"go-phpcs/token"
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

	// Expect opening brace
	if p.tok.Type != token.T_LBRACE {
		p.addError("line %d:%d: expected { to start trait body for %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume {

	// Parse methods and constants inside the trait
	var body []ast.Node
	for p.tok.Type != token.T_RBRACE && p.tok.Type != token.T_EOF {
		// Collect all modifiers (public, protected, private, static, final, abstract) and comments/docblocks before 'function' or 'const'
		var modifiers []string
		for {
			switch p.tok.Type {
			case token.T_PUBLIC, token.T_PROTECTED, token.T_PRIVATE, token.T_STATIC, token.T_FINAL, token.T_ABSTRACT:
				modifiers = append(modifiers, p.tok.Literal)
				p.nextToken()
				continue
			case token.T_COMMENT, token.T_DOC_COMMENT:
				p.nextToken()
				continue
			}
			break
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
			if constant := p.parseConstant(); constant != nil {
				body = append(body, constant)
			}
			continue
		}
		if len(modifiers) > 0 {
			// If we saw modifiers but not a function, emit error and skip
			p.addError("line %d:%d: expected function after modifiers in trait %s body, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
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
