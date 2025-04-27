package parser

import (
	"fmt"
	"go-phpcs/ast"
	"go-phpcs/token"
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
	if p.tok.Type == token.T_LPAREN || p.tok.Type == token.T_NS_SEPARATOR || p.tok.Type == token.T_STRING || p.tok.Type == token.T_CALLABLE || p.tok.Type == token.T_ARRAY || p.tok.Type == token.T_STATIC || p.tok.Type == token.T_SELF || p.tok.Type == token.T_PARENT || p.tok.Type == token.T_NEW || p.tok.Type == token.T_QUESTION || p.tok.Type == token.T_MIXED {
		typeHint = parseFullTypeHint(p)
	} else if p.tok.Literal == "\\" {
		typeHint = parseFullTypeHint(p)
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

	// If we see a comment like /* ,... */ after a parameter, skip it (for commented-out trailing params)
	for p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT {
		// Only skip if the comment starts with '/* ,'
		if strings.HasPrefix(p.tok.Literal, "/* ,") || strings.HasPrefix(p.tok.Literal, ",") {
			p.nextToken()
			continue
		}
		break
	}

	paramNode := &ast.ParamNode{
		Name:         name,
		TypeHint:     typeHint,
		DefaultValue: defaultValue,
		Visibility:   visibility,
		IsPromoted:   isPromoted,
		IsVariadic:   isVariadic,
		IsByRef:      isByRef,
		Pos:          ast.Position(pos),
	}
	// Attach AST node for union/intersection/single type
	typeNode := parseTypeHintAST(typeHint, ast.Position(pos))
	switch t := typeNode.(type) {
	case *ast.UnionTypeNode:
		paramNode.UnionType = t
	case *ast.IntersectionTypeNode:
		paramNode.IntersectionType = t
	}
	return paramNode
}

// parseTypeHintAST parses a type hint string and returns the appropriate AST node (IdentifierNode, UnionTypeNode, IntersectionTypeNode)
func parseTypeHintAST(typeHint string, pos ast.Position) ast.Node {
	typeHint = strings.TrimSpace(typeHint)
	if typeHint == "" {
		return nil
	}
	// Intersection type (has & and not |)
	if strings.Contains(typeHint, "&") && !strings.Contains(typeHint, "|") {
		parts := strings.Split(typeHint, "&")
		var types []string
		for _, part := range parts {
			t := strings.TrimSpace(part)
			if t != "" {
				types = append(types, t)
			}
		}
		return &ast.IntersectionTypeNode{Types: types, Pos: pos}
	}
	// Union type (has |)
	if strings.Contains(typeHint, "|") {
		parts := strings.Split(typeHint, "|")
		var types []string
		for _, part := range parts {
			t := strings.TrimSpace(part)
			if t != "" {
				types = append(types, t)
			}
		}
		return &ast.UnionTypeNode{Types: types, Pos: pos}
	}
	// Single type
	return &ast.IdentifierNode{Value: typeHint, Pos: pos}
}
