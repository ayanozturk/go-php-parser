package parser

import (
	"go-phpcs/ast"
	"go-phpcs/token"
	"strings"
)

func (p *Parser) parseClassDeclarationWithModifier(modifier string) (ast.Node, error) {
	return p.parseClassDeclarationWithModifiers([]string{modifier})
}

func (p *Parser) parseClassDeclarationWithModifiers(modifiers []string) (ast.Node, error) {
	for {
		switch {
		case p.tok.Type == token.T_READONLY:
			modifiers = append(modifiers, p.tok.Literal)
			p.nextToken()
		case p.tok.Type == token.T_STRING && (p.tok.Literal == "final" || p.tok.Literal == "abstract"):
			modifiers = append(modifiers, p.tok.Literal)
			p.nextToken()
		default:
			if p.tok.Type != token.T_CLASS {
				p.addError("line %d:%d: expected 'class' after modifier %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, strings.Join(modifiers, " "), p.tok.Literal)
				return nil, nil
			}

			node, err := p.parseClassDeclaration()
			if classNode, ok := node.(*ast.ClassNode); ok {
				classNode.Modifier = strings.Join(modifiers, " ")
			}
			return node, err
		}
	}
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
			if modifier, ok := p.parsePropertyModifier(); ok {
				modifiers = append(modifiers, modifier)
				continue
			}
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

func (p *Parser) parsePropertyModifier() (string, bool) {
	if p.tok.Type == token.T_READONLY {
		p.nextToken()
		return "readonly", true
	}
	if p.tok.Type != token.T_PUBLIC && p.tok.Type != token.T_PROTECTED && p.tok.Type != token.T_PRIVATE {
		return "", false
	}
	modifier := p.tok.Literal
	if p.peekToken().Type != token.T_LPAREN {
		return "", false
	}
	p.nextToken() // consume visibility, land on '('
	if p.tok.Type != token.T_LPAREN {
		return "", false
	}
	p.nextToken() // consume '('
	if p.tok.Type != token.T_STRING || p.tok.Literal != "set" {
		p.addError("line %d:%d: expected set in asymmetric visibility modifier, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return modifier, true
	}
	p.nextToken() // consume 'set'
	if p.tok.Type != token.T_RPAREN {
		p.addError("line %d:%d: expected ) after asymmetric visibility modifier, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return modifier, true
	}
	p.nextToken() // consume ')'
	return modifier + "(set)", true
}

func (p *Parser) parsePropertyDeclaration(modifiers []string, typeHint string) (ast.Node, error) {
	p.debugTokenContext("parsePropertyDeclaration entry")
	pos := p.tok.Pos
	// Interpret modifiers
	var visibility, setVisibility string
	var isStatic, isReadonly bool
	for _, m := range modifiers {
		switch m {
		case "public", "protected", "private":
			visibility = m
		case "public(set)", "protected(set)", "private(set)":
			setVisibility = m[:len(m)-5]
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
	var hooks []ast.PropertyHookNode
	requiresSemicolon := true
	if p.tok.Type == token.T_LBRACE {
		hooks = p.parsePropertyHooks(name)
		requiresSemicolon = false
	}
	if requiresSemicolon && p.tok.Type != token.T_SEMICOLON {
		p.debugTokenContext("expected semicolon after property")
		p.addError("line %d:%d: expected ; after property declaration $%s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
		return nil, nil
	}
	if requiresSemicolon {
		p.nextToken()
	}
	return &ast.PropertyNode{
		Name:          name,
		TypeHint:      typeHint,
		DefaultValue:  defaultValue,
		Visibility:    visibility,
		SetVisibility: setVisibility,
		IsStatic:      isStatic,
		IsReadonly:    isReadonly,
		Hooks:         hooks,
		Pos:           ast.Position(pos),
	}, nil
}

func (p *Parser) parsePropertyHooks(propertyName string) []ast.PropertyHookNode {
	var hooks []ast.PropertyHookNode
	p.nextToken() // consume '{'
	for p.tok.Type != token.T_RBRACE && p.tok.Type != token.T_EOF {
		for p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT {
			p.nextToken()
		}
		if p.tok.Type == token.T_RBRACE || p.tok.Type == token.T_EOF {
			break
		}

		hookPos := p.tok.Pos
		isByRef := false
		if p.tok.Type == token.T_AMPERSAND {
			isByRef = true
			p.nextToken()
		}
		if p.tok.Type != token.T_STRING {
			p.addError("line %d:%d: expected property hook name for $%s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, propertyName, p.tok.Literal)
			for p.tok.Type != token.T_SEMICOLON && p.tok.Type != token.T_RBRACE && p.tok.Type != token.T_EOF {
				p.nextToken()
			}
			if p.tok.Type == token.T_SEMICOLON {
				p.nextToken()
			}
			continue
		}

		hook := ast.PropertyHookNode{Name: p.tok.Literal, IsByRef: isByRef, Pos: ast.Position(hookPos)}
		p.nextToken() // consume hook name

		if p.tok.Type == token.T_LPAREN {
			hook.Parameter = p.readBalancedPropertyHookHeader()
		}

		switch p.tok.Type {
		case token.T_DOUBLE_ARROW:
			p.nextToken() // consume =>
			hook.Expr = p.parseExpression()
			if p.tok.Type != token.T_SEMICOLON {
				p.addError("line %d:%d: expected ; after %s hook for $%s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, hook.Name, propertyName, p.tok.Literal)
				return hooks
			}
			p.nextToken() // consume ;
		case token.T_LBRACE:
			p.nextToken() // consume {
			hook.Body = p.parseBlockStatement()
			if p.tok.Type != token.T_RBRACE {
				p.addError("line %d:%d: expected } after %s hook for $%s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, hook.Name, propertyName, p.tok.Literal)
				return hooks
			}
			p.nextToken() // consume }
		default:
			p.addError("line %d:%d: expected => or { after %s hook for $%s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, hook.Name, propertyName, p.tok.Literal)
			return hooks
		}

		hooks = append(hooks, hook)
	}

	if p.tok.Type != token.T_RBRACE {
		p.addError("line %d:%d: expected } to close property hooks for $%s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, propertyName, p.tok.Literal)
		return hooks
	}
	p.nextToken() // consume '}'
	return hooks
}

func (p *Parser) readBalancedPropertyHookHeader() string {
	depth := 0
	header := ""
	for {
		if p.tok.Type == token.T_LPAREN {
			depth++
		} else if p.tok.Type == token.T_RPAREN {
			depth--
		}
		header += p.tok.Literal
		p.nextToken()
		if depth == 0 || p.tok.Type == token.T_EOF {
			break
		}
	}
	return header
}
