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
	// Handle parameter list
	for p.tok.Type != token.T_RPAREN && p.tok.Type != token.T_EOF {

		// Skip comments and commas before each parameter
		for p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT || p.tok.Type == token.T_COMMA {
			p.nextToken()
		}
		// After skipping, if next token is ')' or EOF, end the parameter list
		if p.tok.Type == token.T_RPAREN || p.tok.Type == token.T_EOF {
			break
		}

		// Now, expect a parameter

		// --- FIX: skip PHP attributes before parameter (like in parseParameter) ---
		for {
			if p.tok.Type == token.T_ATTRIBUTE {
				p.nextToken()
				continue
			}
			if p.tok.Type == token.T_WHITESPACE || p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT {
				p.nextToken()
				continue
			}
			break
		}

		// Handle nullable type hint (e.g. ?string)
		var typeHint string
		var unionTypeNode *ast.UnionTypeNode
		typePos := p.tok.Pos

		if p.tok.Type == token.T_QUESTION {
			typeHint = "?"
			p.nextToken()
		}

		// Handle type hint (including 'mixed')
		if p.tok.Type == token.T_STRING || p.tok.Type == token.T_ARRAY || p.tok.Type == token.T_BACKSLASH || p.tok.Literal == "mixed" {
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

			// Continue collecting parts of a class name if we encounter backslashes
			for p.tok.Type == token.T_STRING || p.tok.Type == token.T_BACKSLASH {
				typeName.WriteString(p.tok.Literal)
				p.nextToken()
			}

			typeHint += typeName.String()

			// Handle array syntax
			if p.tok.Type == token.T_LBRACKET {
				typeHint += "[]"
				p.nextToken()
				if p.tok.Type != token.T_RBRACKET {
					p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected ']' after array type in parameter, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
					return nil
				}
				p.nextToken()
			}

			// Handle union types in parameters
			if p.tok.Type == token.T_PIPE {
				// Create a union type
				unionTypes := []string{typeHint}
				typeHint = "" // Clear the string-based type hint

				for p.tok.Type == token.T_PIPE {
					p.nextToken() // consume |

					// Parse each union type member
					if p.tok.Type == token.T_STRING || p.tok.Type == token.T_ARRAY ||
						p.tok.Literal == "mixed" || p.tok.Literal == "null" ||
						p.tok.Literal == "int" || p.tok.Literal == "float" ||
						p.tok.Literal == "bool" || p.tok.Literal == "string" ||
						p.tok.Type == token.T_BACKSLASH {

						// Handle fully qualified class names with backslashes
						var memberTypeName strings.Builder
						if p.tok.Type == token.T_BACKSLASH {
							memberTypeName.WriteString(p.tok.Literal)
							p.nextToken()
						} else {
							memberTypeName.WriteString(p.tok.Literal)
							p.nextToken()
						}

						// Continue collecting parts of a class name
						for p.tok.Type == token.T_STRING || p.tok.Type == token.T_BACKSLASH {
							memberTypeName.WriteString(p.tok.Literal)
							p.nextToken()
						}

						unionTypes = append(unionTypes, memberTypeName.String())

						// Handle array syntax for union type member
						if p.tok.Type == token.T_LBRACKET {
							unionTypes[len(unionTypes)-1] += "[]"
							p.nextToken()
							if p.tok.Type != token.T_RBRACKET {
								p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected ']' after array type in parameter union type, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
								return nil
							}
							p.nextToken()
						}
					} else {
						p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected type name in parameter union type, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
						return nil
					}
				}

				unionTypeNode = &ast.UnionTypeNode{
					Types: unionTypes,
					Pos:   ast.Position(typePos),
				}
			}
		}

		// Parameter name
		if p.tok.Type != token.T_VARIABLE {
			// Allow end of parameter list after skipping comments/commas/trailing comments
			if p.tok.Type == token.T_RPAREN {
				break
			}
			// If we see a single trailing comment (e.g., /* , ... */) or comma before ')', skip and break
			if (p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT) && (p.peekToken().Type == token.T_RPAREN) {
				p.nextToken() // skip trailing comment
				break
			}
			if p.tok.Type == token.T_COMMA && (p.peekToken().Type == token.T_RPAREN) {
				p.nextToken() // skip trailing comma
				break
			}
			p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected parameter name in parameter, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
			return nil
		}
		varName := p.tok.Literal[1:] // Remove $ prefix
		paramPos := p.tok.Pos
		p.nextToken()

		// Handle default value
		var defaultValue ast.Node
		if p.tok.Type == token.T_ASSIGN {
			p.nextToken() // consume =
			defaultValue = p.parseExpression()
		}

		param := &ast.ParamNode{
			Name:         varName,
			TypeHint:     typeHint,
			UnionType:    unionTypeNode,
			DefaultValue: defaultValue,
			Pos:          ast.Position(paramPos),
		}

		params = append(params, param)

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
		// If not end, loop will skip comments/commas and try to parse next parameter
	}

	p.nextToken() // consume )

	// Parse return type if present
	var returnType ast.Node
	if p.tok.Type == token.T_COLON {
		p.nextToken() // consume :

		// Support nullable return types (e.g. ?Node)
		var nullable bool
		if p.tok.Type == token.T_QUESTION {
			nullable = true
			p.nextToken()
		}

		// Check for valid return type token
		if p.tok.Type == token.T_STRING || p.tok.Type == token.T_ARRAY ||
			p.tok.Literal == "mixed" || p.tok.Type == token.T_BACKSLASH {
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
						p.tok.Type == token.T_BACKSLASH {

						// Handle fully qualified class names with backslashes
						var memberTypeName strings.Builder
						if p.tok.Type == token.T_BACKSLASH {
							memberTypeName.WriteString(p.tok.Literal)
							p.nextToken()
						} else {
							memberTypeName.WriteString(p.tok.Literal)
							p.nextToken()
						}

						// Continue collecting parts of a class name
						for p.tok.Type == token.T_STRING || p.tok.Type == token.T_BACKSLASH {
							memberTypeName.WriteString(p.tok.Literal)
							p.nextToken()
						}

						unionTypes = append(unionTypes, memberTypeName.String())

						// Handle array syntax for union type member
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
