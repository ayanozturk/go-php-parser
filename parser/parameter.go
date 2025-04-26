package parser

import (
	"fmt"
	"go-phpcs/ast"
	"go-phpcs/token"
)

// parseParameter parses a function or method parameter
func (p *Parser) parseParameter() ast.Node {
	// Skip PHP attributes (#[...]) and comments before parameter, and allow attributes before any parameter element
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

	// Parse all modifiers (visibility, readonly) in any order
	var visibility string
	var isPromoted bool
	for {
		if p.tok.Type == token.T_PUBLIC || p.tok.Type == token.T_PROTECTED || p.tok.Type == token.T_PRIVATE {
			visibility = p.tok.Literal
			isPromoted = true
			p.nextToken()
			continue
		}
		if p.tok.Literal == "readonly" {
			isPromoted = true
			p.nextToken()
			continue
		}
		break
	}
	pos := p.tok.Pos

	// Parse type hint if present (support nullable type: ?Bar, union types: Foo|Bar, FQCNs, parenthesized types)
	var typeHint string
	if p.tok.Type == token.T_LPAREN {
		typeHint = parseFullTypeHint(p)
	} else {
		switch p.tok.Type {
		case token.T_NS_SEPARATOR, token.T_STRING, token.T_CALLABLE, token.T_ARRAY, token.T_STATIC, token.T_SELF, token.T_PARENT, token.T_NEW, token.T_QUESTION, token.T_MIXED:
			typeHint = p.parseTypeHint()
		default:
			// Also allow literal backslash for robustness
			if p.tok.Literal == "\\" {
				typeHint = p.parseTypeHint()
			}
		}
	}

	// After type hint, skip whitespace/comments before checking for & or ... or $var
	for p.tok.Type == token.T_WHITESPACE || p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT {
		p.nextToken()
	}

	// Parse by-reference parameter (&$var)
	isByRef := false
	if p.tok.Type == token.T_AMPERSAND {
		isByRef = true
		p.nextToken() // consume &
	}

	// After &, skip whitespace/comments
	for p.tok.Type == token.T_WHITESPACE || p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT {
		p.nextToken()
	}

	// Parse variadic parameter (...$var)
	isVariadic := false
	if p.tok.Type == token.T_ELLIPSIS {
		isVariadic = true
		p.nextToken() // consume ...
	}

	// After ..., skip whitespace/comments
	for p.tok.Type == token.T_WHITESPACE || p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT {
		p.nextToken()
	}

	// Parse variable name (must be $var)
	if p.tok.Type != token.T_VARIABLE {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected variable name in parameter, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
		return nil
	}
	name := p.tok.Literal[1:] // Remove $ prefix
	p.nextToken()

	// Handle default value if present
	var defaultValue ast.Node
	if p.tok.Type == token.T_ASSIGN {
		p.nextToken() // consume =
		defaultValue = p.parseExpression()
	}

	return &ast.ParamNode{
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
