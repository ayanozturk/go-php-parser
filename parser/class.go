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
	// Parse class body
	for p.tok.Type != token.T_RBRACE && p.tok.Type != token.T_EOF {
		// Collect all modifiers before method/property
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
		// Parse type hint if present (for property)
		var typeHint string
		if p.tok.Type == token.T_STRING || p.tok.Type == token.T_NS_SEPARATOR || p.tok.Type == token.T_CALLABLE || p.tok.Type == token.T_ARRAY || p.tok.Type == token.T_QUESTION {
			typeHint = p.parseTypeHint()
		}
		if p.tok.Type == token.T_FUNCTION {
			if method, err := p.parseFunction(modifiers); method != nil {
				methods = append(methods, method)
			} else if err != nil {
				return nil, err
			}
			continue
		}
		if p.tok.Type == token.T_VARIABLE {
			if prop, err := p.parsePropertyDeclaration(modifiers, typeHint); prop != nil {
				properties = append(properties, prop)
			} else if err != nil {
				return nil, err
			}
			continue
		}
		if len(modifiers) > 0 || typeHint != "" {
			// If we saw modifiers or type hint but not a function/property, emit error and skip
			p.addError("line %d:%d: expected property or function after modifiers/type in class %s body, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
			p.syncToNextClassMember()
			continue
		}
		// Skip unexpected tokens inside class body
		p.addError("line %d:%d: unexpected token %s in class %s body", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal, name)
		p.syncToNextClassMember()
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
		PHPDoc:     p.consumeCurrentDoc(pos),
	}, nil
}

func (p *Parser) debugTokenContext(context string) {
	// fmt.Printf("[DEBUG] %s: token=%v, literal=%q, line=%d\n", context, p.tok.Type, p.tok.Literal, p.tok.Pos.Line)
}

func (p *Parser) parseClassDeclaration() (ast.Node, error) {
	pos := p.tok.Pos
	p.nextToken() // consume 'class'

	if p.tok.Type != token.T_STRING {
		p.debugTokenContext("expected class name")
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
	var constants []ast.Node
	// Parse class body
	for p.tok.Type != token.T_RBRACE && p.tok.Type != token.T_EOF {
		p.debugTokenContext("class body loop entry")
		// Collect all modifiers before method/property/constant
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
		// Parse type hint if present (for property)
		var typeHint string
		if p.tok.Type == token.T_STRING || p.tok.Type == token.T_NS_SEPARATOR || p.tok.Type == token.T_CALLABLE || p.tok.Type == token.T_ARRAY || p.tok.Type == token.T_QUESTION {
			typeHint = p.parseTypeHint()
		}
		if p.tok.Type == token.T_FUNCTION {
			p.debugTokenContext("parsing function")
			if method, err := p.parseFunction(modifiers); method != nil {
				methods = append(methods, method)
			} else if err != nil {
				p.debugTokenContext("parseFunction error")
				return nil, err
			}
			continue
		}
		if p.tok.Type == token.T_VARIABLE {
			p.debugTokenContext("parsing property")
			if prop, err := p.parsePropertyDeclaration(modifiers, typeHint); prop != nil {
				properties = append(properties, prop)
			} else if err != nil {
				p.debugTokenContext("parsePropertyDeclaration error")
				return nil, err
			}
			continue
		}
		if p.tok.Type == token.T_CONST {
			p.debugTokenContext("parsing constant")
			if constant := p.parseConstant(); constant != nil {
				constants = append(constants, constant)
			}
			continue
		}
		if len(modifiers) > 0 || typeHint != "" {
			p.debugTokenContext("unexpected after modifiers/typeHint")
			p.addError("line %d:%d: expected property or function after modifiers/type in class %s body, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
			p.syncToNextClassMember()
			continue
		}
		p.debugTokenContext("unexpected token in class body")
		p.addError("line %d:%d: unexpected token %s in class %s body", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal, name)
		p.syncToNextClassMember()
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
		Constants:  constants,
		Pos:        ast.Position(pos),
		PHPDoc:     p.consumeCurrentDoc(pos),
	}, nil
}

// Helper: skip to next class member or end of class on parse error
func (p *Parser) syncToNextClassMember() {
	for {
		switch p.tok.Type {
		case token.T_PUBLIC, token.T_PROTECTED, token.T_PRIVATE, token.T_STATIC, token.T_FINAL, token.T_ABSTRACT, token.T_FUNCTION, token.T_VARIABLE, token.T_RBRACE, token.T_EOF:
			return
		}
		p.nextToken()
	}
}

func (p *Parser) parsePropertyDeclaration(modifiers []string, typeHint string) (ast.Node, error) {
	p.debugTokenContext("parsePropertyDeclaration entry")
	pos := p.tok.Pos
	// Interpret modifiers
	var visibility string
	var isStatic, isReadonly bool
	for _, m := range modifiers {
		switch m {
		case "public", "protected", "private":
			visibility = m
		case "static":
			isStatic = true
		case "readonly":
			isReadonly = true
		}
	}
	if p.tok.Type != token.T_VARIABLE {
		p.debugTokenContext("expected property name")
		p.addError("line %d:%d: expected property name, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	name := p.tok.Literal[1:]
	p.nextToken()
	// Default value
	var defaultValue ast.Node
	if p.tok.Type == token.T_ASSIGN {
		p.nextToken() // consume =
		defaultValue = p.parseExpression()
	}
	if p.tok.Type != token.T_SEMICOLON {
		p.debugTokenContext("expected semicolon after property")
		p.addError("line %d:%d: expected ; after property declaration $%s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
		return nil, nil
	}
	p.nextToken()
	return &ast.PropertyNode{
		Name:         name,
		TypeHint:     typeHint,
		DefaultValue: defaultValue,
		Visibility:   visibility,
		IsStatic:     isStatic,
		IsReadonly:   isReadonly,
		Pos:          ast.Position(pos),
	}, nil
}
