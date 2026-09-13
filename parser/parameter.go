package parser

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/token"
	"strings"
)

// tokenHasLeadingDocComment reports whether tok carries a T_DOC_COMMENT in
// LeadingTrivia (trivia-on-token attachment).
func tokenHasLeadingDocComment(tok token.Token) bool {
	for _, tr := range tok.LeadingTrivia {
		if tr.Type == token.T_DOC_COMMENT {
			return true
		}
	}
	return false
}

// parseParameter parses a function or method parameter
func (p *Parser) parseParameter() ast.Node {
	var phpdoc *ast.PHPDocNode
	for {
		if p.tok.Type == token.T_ATTRIBUTE {
			p.skipAttributeGroups()
			continue
		}
		if p.tok.Type == token.T_WHITESPACE || p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT {
			if p.tok.Type == token.T_DOC_COMMENT {
				// Standalone doc token (legacy path): bind to this parameter now.
				phpdoc = ast.ExtractPHPDocFromComment(p.tok.Literal)
				if phpdoc != nil {
					phpdoc.Pos = ast.Position(p.tok.Pos)
				}
				p.currentDoc = ""
			}
			p.nextToken()
			continue
		}
		break
	}
	// Trivia-on-token: only take currentDoc when it was harvested from this
	// parameter's first significant token, not a stale method/class PHPDoc.
	if phpdoc == nil && p.currentDoc != "" && tokenHasLeadingDocComment(p.tok) {
		phpdoc = p.consumeCurrentDoc(p.tok.Pos)
	}
	if p.tok.Type == token.T_RPAREN || p.tok.Type == token.T_EOF {
		return nil
	}

	// Parse all modifiers (visibility, asymmetric visibility "(set)", readonly)
	// in any order, e.g. "public private(set) readonly string $x".
	var mods ast.ModifierList
	var isPromoted bool
	var isReadonly bool
	for {
		mod, ok := p.parsePropertyModifier()
		if !ok {
			// parsePropertyModifier only matches "readonly" or a visibility
			// keyword followed by "(" (asymmetric visibility); fall back to
			// a plain visibility keyword here (not followed by "(").
			if p.tok.Type == token.T_PUBLIC || p.tok.Type == token.T_PROTECTED || p.tok.Type == token.T_PRIVATE {
				mod = ast.Modifier{Tok: p.tok.Type, Text: p.tok.Literal}
				p.nextToken()
			} else {
				break
			}
		}
		isPromoted = true
		if mod.Tok == token.T_READONLY {
			isReadonly = true
		}
		mods = append(mods, mod)
	}
	pos := p.tok.Pos

	// Parse type hint if present (support nullable, union, intersection, FQCNs, parenthesized types)
	var typeHint ast.Node
	switch p.tok.Type {
	case token.T_LPAREN, token.T_NS_SEPARATOR, token.T_STRING, token.T_CALLABLE, token.T_ARRAY, token.T_STATIC, token.T_SELF, token.T_PARENT, token.T_NEW, token.T_QUESTION, token.T_MIXED, token.T_NULL, token.T_FALSE, token.T_TRUE:
		typeHint = parseFullTypeNode(p)
	default:
		if p.tok.Literal == "\\" {
			typeHint = parseFullTypeNode(p)
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
		Name:         name,
		TypeHint:     typeHint,
		DefaultValue: defaultValue,
		Modifiers:    mods,
		IsPromoted:   isPromoted,
		IsReadonly:   isReadonly,
		IsVariadic:   isVariadic,
		IsByRef:      isByRef,
		PHPDoc:       phpdoc,
		Pos:          ast.Position(pos),
	}
}
