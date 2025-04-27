package parser

import (
	"fmt"
	"go-phpcs/token"
)

// parseTypeHint parses a type hint (nullable, union, FQCN, etc.)
func (p *Parser) parseTypeHint() string {
	// Only emit errors for real type hints, not when inside docblocks/comments
	isDocblockContext := p.tok.Type == token.T_DOC_COMMENT
	typeHint := ""
	lastWasPipe := false
	segmentCount := 0
	for {
		// Nullable type
		if p.tok.Type == token.T_QUESTION {
			typeHint += "?"
			p.nextToken()
		}
		// Parse FQCN or namespaced type: (\|NS_SEPARATOR)*STRING (repeated)
		typeSegment := ""
		// Accept leading backslash
		if p.tok.Type == token.T_NS_SEPARATOR || p.tok.Literal == "\\" {
			typeSegment += "\\"
			p.nextToken()
		}
		for {
			if p.tok.Type == token.T_STRING || p.tok.Type == token.T_NEW || p.tok.Type == token.T_STATIC || (p.tok.Type == token.T_FALSE && p.tok.Literal == "false") {
				typeSegment += p.tok.Literal
				p.nextToken()
				// Accept chained namespaces: \Foo\Bar
				if p.tok.Type == token.T_NS_SEPARATOR || p.tok.Literal == "\\" {
					typeSegment += "\\"
					p.nextToken()
					continue
				}
			}
			break
		}
		if typeSegment == "" {
			// If we saw a type hint starter but couldn't form a valid segment, advance to avoid infinite loop
			if p.tok.Type == token.T_NS_SEPARATOR || p.tok.Literal == "\\" {
				if !isDocblockContext {
					p.errors = append(p.errors, "unexpected namespace separator in type hint")
				}
				p.nextToken()
				break
			}
			if p.tok.Type == token.T_CALLABLE {
				callableType, err := p.parseCallableType()
				typeSegment += callableType
				if err != nil && !isDocblockContext {
					p.errors = append(p.errors, err.Error())
				}
			} else if p.tok.Type == token.T_ARRAY || p.tok.Type == token.T_NULL || p.tok.Type == token.T_MIXED || p.tok.Literal == "mixed" {
				typeSegment += p.tok.Literal
				p.nextToken()
			}
		}
		if typeSegment != "" {
			typeHint += typeSegment
			segmentCount++
			lastWasPipe = false
			if p.tok.Type == token.T_LBRACKET {
				typeHint += "[]"
				p.nextToken()
				if p.tok.Type != token.T_RBRACKET {
					if !isDocblockContext {
						p.errors = append(p.errors, "expected ']' after array type in type hint")
					}
					return typeHint
				}
				p.nextToken()
			}
		} else {
			// Only emit error if the segment is truly empty and not just at start/end
			if lastWasPipe && segmentCount > 0 {
				if !isDocblockContext {
					p.errors = append(p.errors, "empty type segment in union type")
				}
			}
			break
		}
		if p.tok.Type == token.T_PIPE {
			if lastWasPipe {
				if !isDocblockContext {
					p.errors = append(p.errors, "consecutive '|' in union type")
				}
			}
			typeHint += "|"
			p.nextToken()
			lastWasPipe = true
			continue
		}
		break
	}
	if lastWasPipe {
		if !isDocblockContext {
			p.errors = append(p.errors, "union type ends with '|' or has empty segment")
		}
	}
	if segmentCount == 0 && lastWasPipe {
		if !isDocblockContext {
			p.errors = append(p.errors, "empty union type")
		}
	}

	return typeHint
}

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
