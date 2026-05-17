package parser

import (
	"go-phpcs/ast"
	"go-phpcs/token"
)

func (p *Parser) parseArrayElement() ast.Node {
	pos := p.tok.Pos
	byRef, unpack := p.parseArrayElementFlags()
	var key ast.Node
	value := p.parseExpressionWithPrecedence(0, false, token.T_DOUBLE_ARROW, token.T_COMMA, token.T_RBRACKET, token.T_RPAREN)
	if value == nil {
		return nil
	}

	if p.tok.Type == token.T_DOUBLE_ARROW {
		key = value
		p.nextToken() // consume =>
		if byRef && p.tok.Type != token.T_VARIABLE {
			p.addError("line %d:%d: by-reference must be followed by a variable", p.tok.Pos.Line, p.tok.Pos.Column)
			return nil
		}
		value = p.parseExpression()
		if value == nil {
			return nil
		}
	} else if byRef {
		if _, ok := value.(*ast.VariableNode); !ok {
			p.addError("line %d:%d: by-reference must be followed by a variable", p.tok.Pos.Line, p.tok.Pos.Column)
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

func (p *Parser) parseArrayLiteral(allowSkippedElements bool) ast.Node {
	pos := p.tok.Pos

	// Handle array() syntax
	if p.tok.Type == token.T_ARRAY {
		p.nextToken() // consume array
		if p.tok.Type != token.T_LPAREN {
			p.addError("line %d:%d: expected ( after array, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
			return nil
		}
		p.nextToken() // consume (

		elements, ok := p.parseDelimitedArrayElements(token.T_RPAREN, ")", "array literal", allowSkippedElements)
		if !ok {
			return nil
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

		elements, ok := p.parseDelimitedArrayElements(token.T_RBRACKET, "]", "array literal", allowSkippedElements)
		if !ok {
			return nil
		}
		p.nextToken() // consume ]

		return &ast.ArrayNode{
			Elements: elements,
			Pos:      ast.Position(pos),
		}
	}

	return nil
}

func (p *Parser) parseDelimitedArrayElements(end token.TokenType, endLiteral, context string, allowSkippedElements bool) ([]ast.Node, bool) {
	var elements []ast.Node
	for p.tok.Type != end && p.tok.Type != token.T_EOF {
		for p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT {
			p.nextToken()
		}
		if p.tok.Type == end {
			break
		}
		if allowSkippedElements && p.tok.Type == token.T_COMMA {
			p.nextToken()
			continue
		}
		if element := p.parseArrayElement(); element != nil {
			elements = append(elements, element)
		}
		for p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT {
			p.nextToken()
		}

		if p.tok.Type == token.T_COMMA {
			p.nextToken()
			continue
		}

		if p.tok.Type != end {
			p.addError("line %d:%d: expected , or %s in %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, endLiteral, context, p.tok.Literal)
			return nil, false
		}
		break
	}

	return elements, true
}

func (p *Parser) parseListLiteral(allowSkippedElements bool) ast.Node {
	pos := p.tok.Pos
	p.nextToken() // consume list
	if p.tok.Type != token.T_LPAREN {
		p.addError("line %d:%d: expected ( after list, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil
	}
	p.nextToken() // consume (

	elements, ok := p.parseDelimitedArrayElements(token.T_RPAREN, ")", "list expression", allowSkippedElements)
	if !ok {
		return nil
	}
	p.nextToken() // consume )

	return &ast.ArrayNode{
		Elements: elements,
		Pos:      ast.Position(pos),
	}
}
