package parser

import (
	"go-phpcs/ast"
	"go-phpcs/token"
	"strings"
)

func (p *Parser) parseArrayElement() ast.Node {
	pos := p.tok.Pos
	var key ast.Node
	var value ast.Node
	var byRef bool
	var unpack bool

	// Check for spread operator (...)
	if p.tok.Type == token.T_ELLIPSIS {
		unpack = true
		p.nextToken()
	}

	// Check for by-reference operator (&)
	if p.tok.Type == token.T_AMPERSAND {
		byRef = true
		p.nextToken()
	}

	// Parse key if present (support class constant fetches as keys)
	if p.tok.Type == token.T_STRING || p.tok.Type == token.T_NS_SEPARATOR || p.tok.Type == token.T_MIXED ||
		p.tok.Type == token.T_SELF || p.tok.Type == token.T_PARENT || p.tok.Type == token.T_STATIC {
		// Accumulate fully qualified class name
		var className strings.Builder
		classPos := p.tok.Pos
		for p.tok.Type == token.T_STRING || p.tok.Type == token.T_NS_SEPARATOR {
			className.WriteString(p.tok.Literal)
			p.nextToken()
		}
		if p.tok.Type == token.T_DOUBLE_COLON {
			p.nextToken() // consume ::
			if p.tok.Type == token.T_STRING {
				constName := p.tok.Literal
				key = &ast.ClassConstFetchNode{
					Class: className.String(),
					Const: constName,
					Pos:   ast.Position(classPos),
				}
				p.nextToken()
			} else if p.tok.Type == token.T_CLASS_CONST || p.tok.Type == token.T_CLASS {
				// Support Foo::class
				key = &ast.ClassConstFetchNode{
					Class: className.String(),
					Const: "class",
					Pos:   ast.Position(classPos),
				}
				p.nextToken()
			} else {
				p.addError("line %d:%d: expected constant name after :: in array key, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
				return nil
			}
		} else {
			// Not a class constant fetch, fallback to fully qualified name
			fqdn := className.String()
			key = &ast.IdentifierNode{
				Value: fqdn,
				Pos:   ast.Position(classPos),
			}
		}
	} else if p.tok.Type == token.T_CONSTANT_STRING {
		key = p.parseSimpleExpression()
	}

	if p.tok.Type == token.T_DOUBLE_ARROW {
		p.nextToken() // consume =>
	} else {
		// If no =>, treat the expression as a value
		value = key
		key = nil
	}

	// Parse value if not already set
	if value == nil {
		if byRef && p.tok.Type != token.T_VARIABLE {
			p.addError("line %d:%d: by-reference must be followed by a variable", p.tok.Pos.Line, p.tok.Pos.Column)
			return nil
		}
		value = p.parseExpression()
		if value == nil {
			return nil
		}
	}

	return &ast.ArrayItemNode{
		Key:    key,
		Value:  value,
		ByRef:  byRef,
		Unpack: unpack,
		Pos:    ast.Position(pos),
	}
}

func (p *Parser) parseArrayLiteral() ast.Node {
	pos := p.tok.Pos

	// Handle array() syntax
	if p.tok.Type == token.T_ARRAY {
		p.nextToken() // consume array
		if p.tok.Type != token.T_LPAREN {
			p.addError("line %d:%d: expected ( after array, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
			return nil
		}
		p.nextToken() // consume (

		var elements []ast.Node
		for p.tok.Type != token.T_RPAREN && p.tok.Type != token.T_EOF {
			if element := p.parseArrayElement(); element != nil {
				elements = append(elements, element)
			}

			if p.tok.Type == token.T_COMMA {
				p.nextToken() // consume comma
				continue
			}

			if p.tok.Type != token.T_RPAREN {
				p.addError("line %d:%d: expected , or ) in array literal, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
				return nil
			}
			break
		}
		p.nextToken() // consume )

		return &ast.ArrayNode{
			Elements: elements,
			Pos:      ast.Position(pos),
		}
	}

	// Handle [] syntax
	if p.tok.Type == token.T_LBRACKET {
		p.nextToken() // consume [

		var elements []ast.Node
		for p.tok.Type != token.T_RBRACKET && p.tok.Type != token.T_EOF {
			if element := p.parseArrayElement(); element != nil {
				elements = append(elements, element)
			}

			if p.tok.Type == token.T_COMMA {
				p.nextToken() // consume comma
				continue
			}

			if p.tok.Type != token.T_RBRACKET {
				p.addError("line %d:%d: expected , or ] in array literal, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
				return nil
			}
		}
		p.nextToken() // consume ]

		return &ast.ArrayNode{
			Elements: elements,
			Pos:      ast.Position(pos),
		}
	}

	return nil
}
