package parser

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/token"
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
		token.T_FOR,
		token.T_NEW,
		token.T_CONST,
		token.T_EMPTY,
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
		token.T_STATIC,
		token.T_PUBLIC,
		token.T_PRIVATE,
		token.T_PROTECTED,
		token.T_ABSTRACT,
		token.T_FINAL,
		token.T_READONLY,
		token.T_USE,
		token.T_ARRAY,
		token.T_BREAK,
		token.T_CALLABLE,
		token.T_CASE,
		token.T_CATCH,
		token.T_CLASS,
		token.T_DECLARE,
		token.T_DO,
		token.T_ELSE,
		token.T_ELSEIF,
		token.T_ENDDECLARE,
		token.T_ENDFOR,
		token.T_ENDFOREACH,
		token.T_ENDIF,
		token.T_ENDSWITCH,
		token.T_ENDWHILE,
		token.T_EXIT,
		token.T_EXTENDS,
		token.T_FINALLY,
		token.T_FN,
		token.T_FOREACH,
		token.T_FUNCTION,
		token.T_GOTO,
		token.T_IF,
		token.T_IMPLEMENTS,
		token.T_INCLUDE_ONCE,
		token.T_INSTANCEOF,
		token.T_INSTEADOF,
		token.T_INTERFACE,
		token.T_ISSET,
		token.T_LOGICAL_AND,
		token.T_LOGICAL_OR,
		token.T_LOGICAL_XOR,
		token.T_PRINT,
		token.T_REQUIRE_ONCE,
		token.T_RETURN,
		token.T_SWITCH,
		token.T_THROW,
		token.T_TRAIT,
		token.T_TRY,
		token.T_UNSET,
		token.T_VAR,
		token.T_MIXED,
		token.T_AS,
		token.T_WHILE,
		token.T_YIELD_FROM:
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
