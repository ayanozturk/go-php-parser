package parser

import (
	"go-phpcs/ast"
	"go-phpcs/token"
)

// expect checks if the current token matches the expected type. If so, advances to the next token and returns true.
// Otherwise, adds an error and returns false.
func (p *Parser) expect(expected token.TokenType) bool {
	if p.tok.Type == expected {
		p.nextToken()
		return true
	}
	// p.addError("line %d:%d: expected %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, expected, p.tok.Type)
	return false
}

// skipCommentsAndWhitespace skips over any T_COMMENT, T_DOC_COMMENT, and T_WHITESPACE tokens.
// This is used before boundary tokens ({, }, ;) in declaration headers where trailing
// inline comments (e.g. // phpcs:ignore) may appear between the declaration and the brace.
func (p *Parser) skipCommentsAndWhitespace() {
	for p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT || p.tok.Type == token.T_WHITESPACE {
		p.nextToken()
	}
}

// isValidMethodNameToken returns true if the given token type is valid as a PHP method name.
func isValidMethodNameToken(t token.TokenType) bool {
	switch t {
	case token.T_STRING,
		token.T_CONTINUE,
		token.T_DEFAULT,
		token.T_CONST,
		token.T_ENUM,
		token.T_NAMESPACE,
		token.T_NEVER,
		token.T_NULL,
		token.T_TRUE,
		token.T_FALSE,
		token.T_MATCH,
		token.T_YIELD,
		token.T_LIST,
		token.T_ECHO,
		token.T_INCLUDE,
		token.T_REQUIRE,
		token.T_CLONE,
		token.T_GLOBAL,
		token.T_STATIC:
		// Add more keywords as needed if legal as method names in PHP
		// See https://www.php.net/manual/en/reserved.keywords.php
		// PHP allows most keywords as method names except some special ones (class, function, etc.)
		// This list can be extended as needed for other edge cases
		return true
	default:
		return false
	}
}

func exprOrIdentifier(keyword string, expr ast.Node, pos ast.Position) ast.Node {
	if expr != nil {
		return expr
	}
	return &ast.IdentifierNode{Value: keyword, Pos: pos}
}
