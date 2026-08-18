package parser

import (
	"go-phpcs/token"
)

// parseFullTypeHint parses a type hint, including nested parentheses and unions/intersections, until a non-type token or variable is encountered
func parseFullTypeHint(p *Parser) string {
	parenLevel := 0
	p.nameBuf.Reset()
	for {
		// Skip whitespace and comments between type tokens
		if p.tok.Type == token.T_WHITESPACE || p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT {
			p.nextToken()
			continue
		}
		if p.tok.Type == token.T_LPAREN {
			parenLevel++
			p.nameBuf.WriteString("(")
			p.nextToken()
			continue
		}
		if p.tok.Type == token.T_RPAREN {
			parenLevel--
			p.nameBuf.WriteString(")")
			p.nextToken()
			if parenLevel == 0 && (p.tok.Type != token.T_PIPE && p.tok.Type != token.T_AMPERSAND) {
				break
			}
			continue
		}
		if p.tok.Type == token.T_AMPERSAND && parenLevel == 0 && p.peekToken().Type == token.T_VARIABLE {
			break
		}
		if p.tok.Type == token.T_PIPE || p.tok.Type == token.T_AMPERSAND || p.tok.Type == token.T_QUESTION {
			p.nameBuf.WriteString(p.tok.Literal)
			p.nextToken()
			continue
		}
		if p.tok.Type == token.T_NS_SEPARATOR || p.tok.Type == token.T_CALLABLE || p.tok.Type == token.T_ARRAY ||
			p.tok.Type == token.T_STATIC || p.tok.Type == token.T_SELF || p.tok.Type == token.T_PARENT ||
			p.tok.Type == token.T_NEW || p.tok.Type == token.T_MIXED || p.tok.Type == token.T_NULL ||
			p.tok.Type == token.T_FALSE {
			p.nameBuf.WriteString(p.tok.Literal)
			p.nextToken()
			continue
		}
		if p.tok.Type == token.T_STRING {
			p.nameBuf.WriteString(p.tok.Literal)
			p.nextToken()
			continue
		}
		// Stop if we hit the variable name or anything that can't be part of a type
		break
	}
	return p.nameBuf.String()
}
