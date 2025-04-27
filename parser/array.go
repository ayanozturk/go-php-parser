package parser

import (
	"errors"
	"go-phpcs/ast"
	"go-phpcs/token"
	"strings"
)

func (p *Parser) parseArrayElement() ast.Node {
	pos := p.tok.Pos
	byRef, unpack := p.parseArrayElementFlags()
	key, keyErr := p.parseArrayElementKey()
	if keyErr != nil {
		return nil
	}

	var value ast.Node
	if p.tok.Type == token.T_DOUBLE_ARROW {
		p.nextToken() // consume =>
	} else {
		value = key
		key = nil
	}

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

// parseArrayElementFlags parses the unpack and byRef flags for an array element.
func (p *Parser) parseArrayElementFlags() (byRef, unpack bool) {
	if p.tok.Type == token.T_ELLIPSIS {
		unpack = true
		p.nextToken()
	}
	if p.tok.Type == token.T_AMPERSAND {
		byRef = true
		p.nextToken()
	}
	return
}

// parseArrayElementKey parses the key for an array element, including class constant fetches and identifiers.
func (p *Parser) parseArrayElementKey() (ast.Node, error) {
	tok := p.tok.Type
	if tok == token.T_STRING || tok == token.T_NS_SEPARATOR || tok == token.T_MIXED ||
		tok == token.T_SELF || tok == token.T_PARENT || tok == token.T_STATIC {
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
				key := &ast.ClassConstFetchNode{
					Class: className.String(),
					Const: constName,
					Pos:   ast.Position(classPos),
				}
				p.nextToken()
				return key, nil
			} else if p.tok.Type == token.T_CLASS_CONST || p.tok.Type == token.T_CLASS {
				key := &ast.ClassConstFetchNode{
					Class: className.String(),
					Const: "class",
					Pos:   ast.Position(classPos),
				}
				p.nextToken()
				return key, nil
			} else {
				p.addError("line %d:%d: expected constant name after :: in array key, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
				return nil, ErrInvalidArrayKey
			}
		} else {
			fqdn := className.String()
			key := &ast.IdentifierNode{
				Value: fqdn,
				Pos:   ast.Position(classPos),
			}
			return key, nil
		}
	} else if tok == token.T_CONSTANT_STRING {
		return p.parseSimpleExpression(), nil
	}
	return nil, nil
}

var ErrInvalidArrayKey = errors.New("invalid array key")

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
