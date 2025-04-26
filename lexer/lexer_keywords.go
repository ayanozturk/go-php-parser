package lexer

import (
	"go-phpcs/token"
)

// keywordTokenMap maps PHP keywords to their token types.
var keywordTokenMap = map[string]token.TokenType{
	"function":     token.T_FUNCTION,
	"if":           token.T_IF,
	"else":         token.T_ELSE,
	"elseif":       token.T_ELSEIF,
	"endif":        token.T_ENDIF,
	"array":        token.T_ARRAY,
	"mixed":        token.T_MIXED,
	"string":       token.T_STRING,
	"callable":     token.T_CALLABLE,
	"true":         token.T_TRUE,
	"false":        token.T_FALSE,
	"null":         token.T_NULL,
	"class":        token.T_CLASS,
	"extends":      token.T_EXTENDS,
	"interface":    token.T_INTERFACE,
	"instanceof":   token.T_INSTANCEOF,
	"implements":   token.T_IMPLEMENTS,
	"echo":         token.T_ECHO,
	"new":          token.T_NEW,
	"public":       token.T_PUBLIC,
	"private":      token.T_PRIVATE,
	"protected":    token.T_PROTECTED,
	"static":       token.T_STATIC,
	"return":       token.T_RETURN,
	"declare":      token.T_DECLARE,
	"enum":         token.T_ENUM,
	"match":        token.T_MATCH,
	"fn":           token.T_FN,
	"readonly":     token.T_READONLY,
	"case":         token.T_CASE,
	"trait":        token.T_TRAIT,
	"const":        token.T_CONST,
	"break":        token.T_BREAK,
	"for":          token.T_FOR,
	"foreach":      token.T_FOREACH,
	"as":           token.T_AS,
	"while":        token.T_WHILE,
	"do":           token.T_DO,
	"switch":       token.T_SWITCH,
	"goto":         token.T_GOTO,
	"continue":     token.T_CONTINUE,
	"throw":        token.T_THROW,
	"try":          token.T_TRY,
	"catch":        token.T_CATCH,
	"finally":      token.T_FINALLY,
	"isset":        token.T_ISSET,
	"empty":        token.T_EMPTY,
	"unset":        token.T_UNSET,
	"die":          token.T_DIE,
	"exit":         token.T_EXIT,
	"include":      token.T_INCLUDE,
	"include_once": token.T_INCLUDE_ONCE,
	"require":      token.T_REQUIRE,
	"require_once": token.T_REQUIRE_ONCE,
	"global":       token.T_GLOBAL,
	"list":         token.T_LIST,
	"namespace":    token.T_NAMESPACE,
}

// LookupKeyword returns the token.Token for a given identifier if it's a keyword, else returns T_STRING.
func LookupKeyword(ident string, pos token.Position) token.Token {
	if tokType, ok := keywordTokenMap[ident]; ok {
		return token.Token{Type: tokType, Literal: ident, Pos: pos}
	}

	return token.Token{Type: token.T_STRING, Literal: ident, Pos: pos}
}
