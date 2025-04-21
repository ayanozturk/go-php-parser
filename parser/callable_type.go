package parser

import (
	"fmt"
	"go-phpcs/token"
)

// parseCallableType parses a callable type hint, e.g. callable(int $a, string): void
// Returns the string representation or an error if malformed
func (p *Parser) parseCallableType() (string, error) {
	typeHint := "callable"
	if p.tok.Type == token.T_CALLABLE {
		p.nextToken()
		// If there is no parameter list, just return "callable"
		if p.tok.Type != token.T_LPAREN {
			return typeHint, nil
		}
		if p.tok.Type == token.T_LPAREN {
			typeHint += "("
			p.nextToken()
			paramCount := 0
			for p.tok.Type != token.T_RPAREN && p.tok.Type != token.T_EOF {
				// Allow whitespace/comments between params
				for p.tok.Type == token.T_WHITESPACE || p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT {
					p.nextToken()
				}
				if paramCount > 0 {
					if p.tok.Type != token.T_COMMA {
						return typeHint, fmt.Errorf("expected ',' between callable parameters, got %s", p.tok.Literal)
					}
					typeHint += ","
					p.nextToken()
					for p.tok.Type == token.T_WHITESPACE || p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT {
						p.nextToken()
					}
				}
				// Accept type and variable name, or just type
				paramType := ""
				// Support union/nullable types for params
				if p.tok.Type == token.T_QUESTION || p.tok.Type == token.T_STRING || p.tok.Type == token.T_CALLABLE || p.tok.Type == token.T_ARRAY || p.tok.Type == token.T_NULL || p.tok.Type == token.T_MIXED || p.tok.Literal == "mixed" {
					paramType = p.parseTypeHint()
				}
				if p.tok.Type == token.T_VARIABLE {
					paramType += p.tok.Literal
					p.nextToken()
				} else if paramType == "" {
					return typeHint, fmt.Errorf("expected parameter type or name in callable, got %s", p.tok.Literal)
				}
				typeHint += paramType
				paramCount++
			}
			if p.tok.Type != token.T_RPAREN {
				return typeHint, fmt.Errorf("expected ')' to close callable parameter list, got %s", p.tok.Literal)
			}
			typeHint += ")"
			p.nextToken()
		}
		// Optional return type
		if p.tok.Type == token.T_COLON {
			typeHint += ":"
			p.nextToken()
			// Allow whitespace/comments before return type
			for p.tok.Type == token.T_WHITESPACE || p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT {
				p.nextToken()
			}
			// Support union/nullable types for return type
			if p.tok.Type == token.T_QUESTION || p.tok.Type == token.T_STRING || p.tok.Type == token.T_CALLABLE || p.tok.Type == token.T_ARRAY || p.tok.Type == token.T_NULL || p.tok.Type == token.T_MIXED || p.tok.Literal == "mixed" {
				typeHint += p.parseTypeHint()
			} else {
				return typeHint, fmt.Errorf("expected return type after ':' in callable, got %s", p.tok.Literal)
			}
		}
	}
	return typeHint, nil
}
