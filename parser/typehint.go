package parser

import (
	"go-phpcs/token"
	"strings"
)

// parseFullTypeHint parses a type hint, including nested parentheses and unions/intersections, until a non-type token or variable is encountered
func parseFullTypeHint(p *Parser) string {
	parenLevel := 0
	typeHintBuilder := strings.Builder{}
	for {
		if p.tok.Type == token.T_LPAREN {
			parenLevel++
			typeHintBuilder.WriteString("(")
			p.nextToken()
			continue
		}
		if p.tok.Type == token.T_RPAREN {
			parenLevel--
			typeHintBuilder.WriteString(")")
			p.nextToken()
			if parenLevel == 0 && (p.tok.Type != token.T_PIPE && p.tok.Type != token.T_AMPERSAND) {
				break
			}
			continue
		}
		if p.tok.Type == token.T_PIPE || p.tok.Type == token.T_AMPERSAND || p.tok.Type == token.T_QUESTION {
			typeHintBuilder.WriteString(p.tok.Literal)
			p.nextToken()
			continue
		}
		if p.tok.Type == token.T_NS_SEPARATOR || p.tok.Type == token.T_STRING || p.tok.Type == token.T_CALLABLE || p.tok.Type == token.T_ARRAY || p.tok.Type == token.T_STATIC || p.tok.Type == token.T_SELF || p.tok.Type == token.T_PARENT || p.tok.Type == token.T_NEW || p.tok.Type == token.T_MIXED || p.tok.Type == token.T_NULL {
			typeHintBuilder.WriteString(p.tok.Literal)
			p.nextToken()
			continue
		}
		// Stop if we hit the variable name or anything that can't be part of a type
		break
	}
	return typeHintBuilder.String()
}
