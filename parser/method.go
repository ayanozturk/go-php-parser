package parser

import (
	"fmt"
	"go-phpcs/ast"
	"go-phpcs/token"
	"strings"
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

	// Skip comments and whitespace before opening brace
	for p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT || p.tok.Type == token.T_WHITESPACE {
		p.nextToken()
	}
	if p.tok.Type != token.T_LBRACE {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected { after interface name %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal))
		return nil
	}
	p.nextToken() // consume {

	var members []ast.Node
	for p.tok.Type != token.T_RBRACE && p.tok.Type != token.T_EOF {
		// Skip doc comments and regular comments in interface body
		if p.tok.Type == token.T_DOC_COMMENT || p.tok.Type == token.T_COMMENT {
			p.nextToken()
			continue
		}
		// Interface members: methods and constants
		if p.tok.Type == token.T_PUBLIC || p.tok.Type == token.T_PRIVATE || p.tok.Type == token.T_PROTECTED {
			// Advance past visibility
			p.nextToken()
			// Skip any number of 'static', comments, and whitespace
			for {
				if p.tok.Type == token.T_STATIC || p.tok.Type == token.T_DOC_COMMENT || p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_WHITESPACE {
					p.nextToken()
				} else {
					break
				}
			}
			if p.tok.Type == token.T_CONST {
				if constant := p.parseConstant(); constant != nil {
					members = append(members, constant)
				}
			} else if p.tok.Type == token.T_FUNCTION {
				if method := p.parseInterfaceMethod(); method != nil {
					members = append(members, method)
				}
			} else {
				p.errors = append(p.errors, fmt.Sprintf("line %d:%d: unexpected token %s after visibility modifier in interface %s body", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal, name))
				p.nextToken()
			}
		} else if p.tok.Type == token.T_FUNCTION {
			if method := p.parseInterfaceMethod(); method != nil {
				members = append(members, method)
			}
		} else if p.tok.Type == token.T_CONST {
			if constant := p.parseConstant(); constant != nil {
				members = append(members, constant)
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
		Members: members,
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
		p.syncToNextClassMember()
		return nil
	}
	p.nextToken()

	// Accept PHP keywords as method names (not just T_STRING)
	if !isValidMethodNameToken(p.tok.Type) {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected method name, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
		p.syncToNextClassMember()
		return nil
	}
	name := p.tok.Literal
	p.nextToken()

	// Parse opening parenthesis
	if p.tok.Type != token.T_LPAREN {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected '(' after method name %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal))
		p.syncToNextClassMember()
		return nil
	}
	p.nextToken()

	// Parse parameters
	var params []ast.Node
	for p.tok.Type != token.T_RPAREN && p.tok.Type != token.T_EOF {
		// Skip comments and commas before each parameter
		for p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT || p.tok.Type == token.T_COMMA {
			p.nextToken()
		}
		if p.tok.Type == token.T_RPAREN || p.tok.Type == token.T_EOF {
			break
		}
		param := p.parseParameter()
		if param != nil {
			params = append(params, param)
		}
		// After a parameter, skip any comments and commas before checking for next parameter or end
		for p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT || p.tok.Type == token.T_COMMA {
			p.nextToken()
		}
		if p.tok.Type == token.T_RPAREN {
			break
		}
		if p.tok.Type == token.T_EOF {
			break
		}
	}
	if p.tok.Type != token.T_RPAREN {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected ')' after parameter list for method %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal))
		p.syncToNextClassMember()
		return nil
	}
	p.nextToken() // consume )

	// Skip comments after parameter list (e.g., /* : self */)
	for p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT {
		p.nextToken()
	}

	// Parse return type if present
	var returnType ast.Node
	if p.tok.Type == token.T_COLON {
		p.nextToken() // consume :
		typePos := p.tok.Pos
		typeStr := p.parseTypeHint()
		if typeStr != "" {
			if strings.Contains(typeStr, "|") {
				parts := strings.Split(typeStr, "|")
				for i := range parts {
					parts[i] = strings.TrimSpace(parts[i])
				}
				returnType = &ast.UnionTypeNode{
					Types: parts,
					Pos:   ast.Position(typePos),
				}
			} else if strings.Contains(typeStr, "&") {
				parts := strings.Split(typeStr, "&")
				for i := range parts {
					parts[i] = strings.TrimSpace(parts[i])
				}
				returnType = &ast.IntersectionTypeNode{
					Types: parts,
					Pos:   ast.Position(typePos),
				}
			} else {
				returnType = &ast.IdentifierNode{
					Value: typeStr,
					Pos:   ast.Position(typePos),
				}
			}
		} else {
			p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected return type for method %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal))
			p.syncToNextClassMember()
			return nil
		}
	}

	// Skip comments before semicolon (defensive, in case comments appear here)
	for p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT {
		p.nextToken()
	}

	// Parse semicolon
	if p.tok.Type != token.T_SEMICOLON {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected ';' after method declaration %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal))
		p.syncToNextClassMember()
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
