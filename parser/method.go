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
		if p.tok.Type == token.T_PUBLIC || p.tok.Type == token.T_PRIVATE || p.tok.Type == token.T_PROTECTED || p.tok.Type == token.T_FUNCTION {
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
	p.nextToken() // consume )

	// Parse return type if present
	var returnType ast.Node
	if p.tok.Type == token.T_COLON {
		p.nextToken() // consume :

		// Handle nullable return types (e.g. ?Node)
		var nullable bool
		if p.tok.Type == token.T_QUESTION {
			nullable = true
			p.nextToken()
		}

		// Handle callable return type
		if p.tok.Type == token.T_CALLABLE {
			callableType, err := p.parseCallableType()
			if err != nil {
				p.errors = append(p.errors, fmt.Sprintf("line %d:%d: %v", p.tok.Pos.Line, p.tok.Pos.Column, err))
				return nil
			}
			returnType = &ast.IdentifierNode{
				Value: callableType,
				Pos:   ast.Position(p.tok.Pos),
			}
		} else if p.tok.Type == token.T_STRING || p.tok.Type == token.T_ARRAY ||
			p.tok.Literal == "mixed" || p.tok.Literal == "null" ||
			p.tok.Type == token.T_BACKSLASH ||
			p.tok.Type == token.T_STATIC || p.tok.Type == token.T_SELF || p.tok.Type == token.T_PARENT {
			typePos := p.tok.Pos

			// Handle fully qualified class names with backslashes
			var typeName strings.Builder
			if p.tok.Literal == "mixed" {
				typeName.WriteString("mixed")
				p.nextToken()
			} else if p.tok.Type == token.T_BACKSLASH {
				typeName.WriteString(p.tok.Literal)
				p.nextToken()
			} else {
				typeName.WriteString(p.tok.Literal)
				p.nextToken()
			}

			// Continue collecting parts of a class name
			for p.tok.Type == token.T_STRING || p.tok.Type == token.T_BACKSLASH {
				typeName.WriteString(p.tok.Literal)
				p.nextToken()
			}

			// Handle array type
			if p.tok.Type == token.T_LBRACKET {
				typeName.WriteString("[]")
				p.nextToken()
				if p.tok.Type != token.T_RBRACKET {
					p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected ']' after array type in return type for method %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal))
					return nil
				}
				p.nextToken()
			}

			// Check for union type (|)
			if p.tok.Type == token.T_PIPE {
				// Create a union type
				unionPos := typePos
				unionTypes := []string{typeName.String()}

				// Continue collecting types until we don't see more pipes
				for p.tok.Type == token.T_PIPE {
					p.nextToken() // consume |

					// Parse each union type member
					if p.tok.Type == token.T_STRING || p.tok.Type == token.T_ARRAY ||
						p.tok.Literal == "mixed" || p.tok.Literal == "null" ||
						p.tok.Literal == "int" || p.tok.Literal == "float" ||
						p.tok.Literal == "bool" || p.tok.Literal == "string" ||
						p.tok.Type == token.T_BACKSLASH || p.tok.Type == token.T_NS_SEPARATOR ||
						p.tok.Type == token.T_STATIC || p.tok.Type == token.T_SELF || p.tok.Type == token.T_PARENT {
						var memberTypeName strings.Builder
						if p.tok.Literal == "null" {
							memberTypeName.WriteString("null")
							p.nextToken()
						} else {
							for p.tok.Type == token.T_BACKSLASH || p.tok.Type == token.T_NS_SEPARATOR {
								memberTypeName.WriteString("\\")
								p.nextToken()
							}
							if p.tok.Type == token.T_STRING || p.tok.Type == token.T_STATIC || p.tok.Type == token.T_SELF || p.tok.Type == token.T_PARENT {
								memberTypeName.WriteString(p.tok.Literal)
								p.nextToken()
								for p.tok.Type == token.T_STRING || p.tok.Type == token.T_BACKSLASH || p.tok.Type == token.T_NS_SEPARATOR {
									if p.tok.Type == token.T_BACKSLASH || p.tok.Type == token.T_NS_SEPARATOR {
										memberTypeName.WriteString("\\")
									} else {
										memberTypeName.WriteString(p.tok.Literal)
									}
									p.nextToken()
								}
							}
						}
						unionTypes = append(unionTypes, memberTypeName.String())
						if p.tok.Type == token.T_LBRACKET {
							unionTypes[len(unionTypes)-1] += "[]"
							p.nextToken()
							if p.tok.Type != token.T_RBRACKET {
								p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected ']' after array type in union type for method %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal))
								return nil
							}
							p.nextToken()
						}
					} else {
						p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected type name in union type for method %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal))
						return nil
					}
				}

				// Create the union type node
				returnType = &ast.UnionTypeNode{
					Types: unionTypes,
					Pos:   ast.Position(unionPos),
				}
			} else {
				// Create a simple type node
				returnType = &ast.IdentifierNode{
					Value: typeName.String(),
					Pos:   ast.Position(typePos),
				}
			}

			// If nullable, wrap the type in a nullable type node
			if nullable {
				// If it's already a union type, just add null to it
				if ut, ok := returnType.(*ast.UnionTypeNode); ok {
					ut.Types = append(ut.Types, "null")
				} else {
					// Otherwise, create a union type with the base type and null
					returnType = &ast.UnionTypeNode{
						Types: []string{returnType.TokenLiteral(), "null"},
						Pos:   ast.Position(typePos),
					}
				}
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
