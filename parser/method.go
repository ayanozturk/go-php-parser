package parser

import (
	"fmt"
	"go-phpcs/ast"
	"go-phpcs/token"
)

// parseInterfaceDeclaration parses a PHP interface declaration
func (p *Parser) parseInterfaceDeclaration() ast.Node {
	pos := p.tok.Pos
	p.nextToken() // consume 'interface'

	if p.tok.Type != token.T_STRING {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected interface name, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
		return nil
	}

	name := p.tok.Literal
	p.nextToken()

	// Parse optional 'extends' clause
	var extends []string
	if p.tok.Type == token.T_EXTENDS {
		p.nextToken() // consume 'extends'
		for {
			fqcn := ""
			// Accept one or more leading backslashes for FQCN
			for p.tok.Literal == "\\" {
				fqcn += p.tok.Literal
				p.nextToken()
			}
			if p.tok.Type != token.T_STRING {
				p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected interface name after extends, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
				return nil
			}
			fqcn += p.tok.Literal
			extends = append(extends, fqcn)
			p.nextToken()
			if p.tok.Type != token.T_COMMA {
				break
			}
			p.nextToken() // consume ','
		}
	}

	if p.tok.Type != token.T_LBRACE {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected { after interface name %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal))
		return nil
	}
	p.nextToken() // consume {

	var methods []ast.Node
	for p.tok.Type != token.T_RBRACE && p.tok.Type != token.T_EOF {
		// Skip doc comments and regular comments in interface body
		if p.tok.Type == token.T_DOC_COMMENT || p.tok.Type == token.T_COMMENT {
			p.nextToken()
			continue
		}
		// Interface methods can have visibility modifiers
		if p.tok.Type == token.T_PUBLIC || p.tok.Type == token.T_PRIVATE || p.tok.Type == token.T_PROTECTED || p.tok.Type == token.T_FUNCTION {
			if method := p.parseInterfaceMethod(); method != nil {
				methods = append(methods, method)
			}
		} else {
			p.errors = append(p.errors, fmt.Sprintf("line %d:%d: unexpected token %s in interface %s body", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal, name))
			p.nextToken()
		}
	}

	if p.tok.Type != token.T_RBRACE {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected } to close interface %s body, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal))
		return nil
	}
	p.nextToken() // consume }

	return &ast.InterfaceNode{
		Name:    name,
		Extends: extends,
		Methods: methods,
		Pos:     ast.Position(pos),
	}
}

// parseInterfaceMethod parses a method declaration in an interface
func (p *Parser) parseInterfaceMethod() ast.Node {
	pos := p.tok.Pos

	// Skip doc comments and regular comments before method signature
	for p.tok.Type == token.T_DOC_COMMENT || p.tok.Type == token.T_COMMENT {
		p.nextToken()
	}

	// Parse visibility modifier if present
	var visibility string
	if p.tok.Type == token.T_PUBLIC || p.tok.Type == token.T_PRIVATE || p.tok.Type == token.T_PROTECTED {
		visibility = p.tok.Literal
		p.nextToken()
	}

	// Parse function keyword
	if p.tok.Type != token.T_FUNCTION {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected 'function' keyword, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
		return nil
	}
	p.nextToken()

	// Parse method name
	if p.tok.Type != token.T_STRING {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected method name, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
		return nil
	}
	name := p.tok.Literal
	p.nextToken()

	// Parse opening parenthesis
	if p.tok.Type != token.T_LPAREN {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected '(' after method name %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal))
		return nil
	}
	p.nextToken()

	// Parse parameters
	var params []ast.Node
	for p.tok.Type != token.T_RPAREN && p.tok.Type != token.T_EOF {
		// Allow for trailing comma before closing parenthesis
		if p.tok.Type == token.T_COMMA {
			p.nextToken()
			if p.tok.Type == token.T_RPAREN {
				break
			}
		}

		param := p.parseParameter()
		if param == nil {
			// If the parameter is nil and we're at a closing parenthesis, break (tolerate trailing comma)
			if p.tok.Type == token.T_RPAREN {
				break
			}
			return nil
		}
		params = append(params, param)

		if p.tok.Type == token.T_COMMA {
			p.nextToken()
			// Accept trailing comma
			if p.tok.Type == token.T_RPAREN {
				break
			}
		} else if p.tok.Type == token.T_RPAREN {
			break
		} else {
			// Instead of erroring immediately, skip ignorable tokens and try to continue
			// If the token is whitespace, comment, or doc comment, skip it
			if p.tok.Type == token.T_WHITESPACE || p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT {
				p.nextToken()
				continue
			}
			// Otherwise, error and stop
			p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected ',' or ')' in parameter list for method %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal))
			return nil
		}
	}

	// Parse closing parenthesis
	if p.tok.Type != token.T_RPAREN {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected ')' after parameter list for method %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal))
		return nil
	}
	p.nextToken()

	// Parse return type
	var returnType string
	if p.tok.Type == token.T_COLON {
		p.nextToken()
		if p.tok.Type == token.T_STRING || p.tok.Type == token.T_ARRAY {
			returnType = p.tok.Literal
			p.nextToken()

			// Handle array type
			if p.tok.Type == token.T_LBRACKET {
				returnType += "[]"
				p.nextToken()
				if p.tok.Type != token.T_RBRACKET {
					p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected ']' after array type in return type for method %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal))
					return nil
				}
				p.nextToken()
			}
		} else {
			p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected return type for method %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal))
			return nil
		}
	}

	// Parse semicolon
	if p.tok.Type != token.T_SEMICOLON {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected ';' after method declaration %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal))
		return nil
	}
	p.nextToken()

	return &ast.InterfaceMethodNode{
		Name:       name,
		Visibility: visibility,
		ReturnType: returnType,
		Params:     params,
		Pos:        ast.Position(pos),
	}
}
