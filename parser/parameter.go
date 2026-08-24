package parser

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/token"
	"strings"
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

	// Parse all modifiers (visibility, asymmetric visibility "(set)", readonly)
	// in any order, e.g. "public private(set) readonly string $x".
	var visibility string
	var setVisibility string
	var isPromoted bool
	var isReadonly bool
	for {
		mod, ok := p.parsePropertyModifier()
		if !ok {
			// parsePropertyModifier only matches "readonly" or a visibility
			// keyword followed by "(" (asymmetric visibility); fall back to
			// a plain visibility keyword here (not followed by "(").
			if p.tok.Type == token.T_PUBLIC || p.tok.Type == token.T_PROTECTED || p.tok.Type == token.T_PRIVATE {
				mod = p.tok.Literal
				p.nextToken()
			} else {
				break
			}
		}
		isPromoted = true
		switch mod {
		case "readonly":
			isReadonly = true
		case "public(set)", "protected(set)", "private(set)":
			setVisibility = mod[:len(mod)-5]
		default:
			visibility = mod
		}
	}
	pos := p.tok.Pos

	// Parse type hint if present (support nullable, union, intersection, FQCNs, parenthesized types)
	var typeHint string
	switch p.tok.Type {
	case token.T_LPAREN, token.T_NS_SEPARATOR, token.T_STRING, token.T_CALLABLE, token.T_ARRAY, token.T_STATIC, token.T_SELF, token.T_PARENT, token.T_NEW, token.T_QUESTION, token.T_MIXED, token.T_NULL, token.T_FALSE, token.T_TRUE:
		typeHint = parseFullTypeHint(p)
	default:
		if p.tok.Literal == "\\" {
			typeHint = parseFullTypeHint(p)
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
		p.addError("line %d:%d: expected variable name in parameter, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		// Enhanced error recovery: skip to next comma or closing parenthesis
		for p.tok.Type != token.T_COMMA && p.tok.Type != token.T_RPAREN && p.tok.Type != token.T_EOF {
			p.nextToken()
		}
		return nil
	}
	name := p.tok.Literal[1:] // Remove $ prefix
	p.nextToken()

	// Allow spacing/comments between the variable and default assignment.
	for p.tok.Type == token.T_WHITESPACE || p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT {
		p.nextToken()
	}

	// Handle default value if present
	var defaultValue ast.Node
	if p.tok.Type == token.T_ASSIGN {
		p.nextToken() // consume =
		defaultValue = p.parseExpression()
	}

	// If we see a comment after a parameter, skip it (for commented-out or inline params)
	for p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT {
		// Skip any comment that looks like a commented-out parameter or is inline
		if strings.HasPrefix(p.tok.Literal, "/*") || strings.HasPrefix(p.tok.Literal, "//") || strings.HasPrefix(p.tok.Literal, ",") {
			p.nextToken()
			continue
		}
		break
	}

	return &ast.ParamNode{
		Name:          name,
		TypeHint:      typeHint,
		DefaultValue:  defaultValue,
		Visibility:    visibility,
		SetVisibility: setVisibility,
		IsPromoted:    isPromoted,
		IsReadonly:    isReadonly,
		IsVariadic:    isVariadic,
		IsByRef:       isByRef,
		Pos:           ast.Position(pos),
	}
}
