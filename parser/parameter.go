package parser

import (
	"fmt"
	"go-phpcs/ast"
	"go-phpcs/token"
)

// parseParameter parses a function or method parameter
func (p *Parser) parseParameter() ast.Node {
	// Skip PHP attributes and comments before parameter
	for p.tok.Type == token.T_ATTRIBUTE || p.tok.Type == token.T_WHITESPACE || p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT {
		p.nextToken()
	}

	// PHP8+ constructor property promotion: check for visibility and readonly modifier (in any order)
	var visibility string
	var isPromoted bool
	for {
		if p.tok.Type == token.T_PUBLIC || p.tok.Type == token.T_PROTECTED || p.tok.Type == token.T_PRIVATE {
			visibility = p.tok.Literal
			isPromoted = true
			p.nextToken()
		} else if p.tok.Literal == "readonly" { // fallback for readonly if token.T_READONLY is not defined
			isPromoted = true
			p.nextToken()
		} else {
			break
		}
	}
	pos := p.tok.Pos

	if p.tok.Type == token.T_PUBLIC || p.tok.Type == token.T_PROTECTED || p.tok.Type == token.T_PRIVATE {
		visibility = p.tok.Literal
		isPromoted = true
		p.nextToken() // consume visibility
	}

	// Parse type hint if present (support nullable type: ?Bar, union types: Foo|Bar, FQCNs)
	var typeHint string
	typeHint = p.parseTypeHint()

	// Parse by-reference parameter (&$var)
	isByRef := false
	if p.tok.Type == token.T_AMPERSAND {
		isByRef = true
		p.nextToken() // consume &
	}

	// Parse variadic parameter (...$var)
	isVariadic := false
	if p.tok.Type == token.T_ELLIPSIS {
		isVariadic = true
		p.nextToken() // consume ...
	}

	// Parse variable name (allow $var, or edge-case: 'mixed' or 'string' as parameter names)
	var name string
	if p.tok.Type == token.T_VARIABLE {
		name = p.tok.Literal[1:] // Remove $ prefix
		p.nextToken()
	} else if p.tok.Type == token.T_MIXED || p.tok.Type == token.T_STRING {
		// Accept 'mixed' or 'string' as parameter names (no $ prefix)
		name = p.tok.Literal
		p.nextToken()
	} else {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected variable name in parameter, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
		return nil
	}

	// Handle default value if present
	var defaultValue ast.Node
	if p.tok.Type == token.T_ASSIGN {
		p.nextToken() // consume =
		defaultValue = p.parseExpression()
	}

	return &ast.ParameterNode{
		Name:         name,
		TypeHint:     typeHint,
		DefaultValue: defaultValue,
		Visibility:   visibility,
		IsPromoted:   isPromoted,
		IsVariadic:   isVariadic,
		IsByRef:      isByRef,
		Pos:          ast.Position(pos),
	}
}
