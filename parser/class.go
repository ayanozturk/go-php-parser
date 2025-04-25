package parser

import (
	"go-phpcs/ast"
	"go-phpcs/token"
)

func (p *Parser) parseClassDeclarationWithModifier(modifier string) (ast.Node, error) {
	// This is the same as parseClassDeclaration but attaches the modifier
	pos := p.tok.Pos
	p.nextToken() // consume 'class'

	if p.tok.Type != token.T_STRING {
		p.addError("line %d:%d: expected class name, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}

	name := p.tok.Literal
	p.nextToken()

	// Check for extends clause
	var extends string
	if p.tok.Type == token.T_EXTENDS {
		p.nextToken() // consume 'extends'
		if p.tok.Type != token.T_STRING {
			p.addError("line %d:%d: expected parent class name after extends, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
			return nil, nil
		}
		extends = p.tok.Literal
		p.nextToken()
	}

	// Check for implements clause
	var implements []string
	if p.tok.Type == token.T_IMPLEMENTS {
		p.nextToken() // consume 'implements'
		for {
			if p.tok.Type != token.T_STRING {
				p.addError("line %d:%d: expected interface name after implements, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
				return nil, nil
			}
			implements = append(implements, p.tok.Literal)
			p.nextToken()

			if p.tok.Type != token.T_COMMA {
				break
			}
			p.nextToken() // consume comma
		}
	}

	if p.tok.Type != token.T_LBRACE {
		p.addError("line %d:%d: expected { after class declaration for %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume {

	var properties []ast.Node
	var methods []ast.Node
	for p.tok.Type != token.T_RBRACE && p.tok.Type != token.T_EOF {
		// Collect all modifiers (public, protected, private, static, final, abstract) and comments/docblocks before 'function'
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
			if method, err := p.parseFunction(modifiers); method != nil {
				methods = append(methods, method)
			} else if err != nil {
				return nil, err
			}
			continue
		}
		if len(modifiers) > 0 {
			// If we saw modifiers but not a function, emit error and skip
			p.addError("line %d:%d: expected function after modifiers in class %s body, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
			p.nextToken()
			continue
		}
		if p.tok.Type == token.T_VARIABLE {
			if prop, err := p.parsePropertyDeclaration(); prop != nil {
				properties = append(properties, prop)
			} else if err != nil {
				return nil, err
			}
			continue
		}
		p.addError("line %d:%d: unexpected token %s in class %s body", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal, name)
		p.nextToken()
	}

	if p.tok.Type != token.T_RBRACE {
		p.addError("line %d:%d: expected } to close class %s body, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume }

	return &ast.ClassNode{
		Name:       name,
		Extends:    extends,
		Implements: implements,
		Properties: properties,
		Methods:    methods,
		Pos:        ast.Position(pos),
		Modifier:   modifier,
	}, nil
}

func (p *Parser) parseClassDeclaration() (ast.Node, error) {
	pos := p.tok.Pos
	p.nextToken() // consume 'class'

	if p.tok.Type != token.T_STRING {
		p.addError("line %d:%d: expected class name, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}

	name := p.tok.Literal
	p.nextToken()

	// Check for extends clause
	var extends string
	if p.tok.Type == token.T_EXTENDS {
		p.nextToken() // consume 'extends'
		if p.tok.Type != token.T_STRING {
			p.addError("line %d:%d: expected parent class name after extends, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
			return nil, nil
		}
		extends = p.tok.Literal
		p.nextToken()
	}

	// Check for implements clause
	var implements []string
	if p.tok.Type == token.T_IMPLEMENTS {
		p.nextToken() // consume 'implements'
		for {
			if p.tok.Type != token.T_STRING {
				p.addError("line %d:%d: expected interface name after implements, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
				return nil, nil
			}
			implements = append(implements, p.tok.Literal)
			p.nextToken()

			if p.tok.Type != token.T_COMMA {
				break
			}
			p.nextToken() // consume comma
		}
	}

	if p.tok.Type != token.T_LBRACE {
		p.addError("line %d:%d: expected { after class declaration for %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume {

	var properties []ast.Node
	var methods []ast.Node
	for p.tok.Type != token.T_RBRACE && p.tok.Type != token.T_EOF {
		// Collect all modifiers (public, protected, private, static, final, abstract) and comments/docblocks before 'function'
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
			if method, err := p.parseFunction(modifiers); method != nil {
				methods = append(methods, method)
			} else if err != nil {
				return nil, err
			}
			continue
		}
		if len(modifiers) > 0 {
			// If we saw modifiers but not a function, emit error and skip
			p.addError("line %d:%d: expected function after modifiers in class %s body, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
			p.nextToken()
			continue
		}
		if p.tok.Type == token.T_VARIABLE {
			if prop, err := p.parsePropertyDeclaration(); prop != nil {
				properties = append(properties, prop)
			} else if err != nil {
				return nil, err
			}
			continue
		}
		p.addError("line %d:%d: unexpected token %s in class %s body", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal, name)
		p.nextToken()
	}

	if p.tok.Type != token.T_RBRACE {
		p.addError("line %d:%d: expected } to close class %s body, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume }

	return &ast.ClassNode{
		Name:       name,
		Extends:    extends,
		Implements: implements,
		Properties: properties,
		Methods:    methods,
		Pos:        ast.Position(pos),
	}, nil
}

func (p *Parser) parsePropertyDeclaration() (ast.Node, error) {
	pos := p.tok.Pos
	name := p.tok.Literal[1:] // Remove $ prefix
	p.nextToken()

	if p.tok.Type != token.T_SEMICOLON {
		p.addError("line %d:%d: expected ; after property declaration $%s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
		return nil, nil
	}
	p.nextToken()

	return &ast.PropertyNode{
		Name: name,
		Pos:  ast.Position(pos),
	}, nil
}
