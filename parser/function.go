package parser

import (
	"go-phpcs/ast"
	"go-phpcs/token"
)

// parseFunction parses a PHP function declaration
func (p *Parser) parseFunction(modifiers []string) (ast.Node, error) {
	pos := p.tok.Pos
	p.nextToken() // consume 'function'

	var savedModifiers []string
	if len(modifiers) > 0 {
		savedModifiers = make([]string, len(modifiers))
		copy(savedModifiers, modifiers)
	}

	var name string
	if isValidMethodNameToken(p.tok.Type) {
		name = p.tok.Literal
		p.nextToken()
	}

	if p.tok.Type != token.T_LPAREN {
		p.addError("line %d:%d: expected ( after function name %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
		p.syncToNextClassMember()
		return nil, nil
	}
	p.nextToken() // consume (

	var params []ast.Node
	for p.tok.Type != token.T_RPAREN && p.tok.Type != token.T_EOF {
		// Skip comments before parameter or after trailing comma
		for p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT || p.tok.Type == token.T_WHITESPACE {
			p.nextToken()
		}
		// If after skipping comments we see a closing parenthesis or EOF, stop.
		if p.tok.Type == token.T_RPAREN || p.tok.Type == token.T_EOF {
			break
		}
		param := p.parseParameter()
		if param == nil {
			// Enhanced error recovery: skip to next comma, closing parenthesis, or opening brace
			for p.tok.Type != token.T_COMMA && p.tok.Type != token.T_RPAREN && p.tok.Type != token.T_LBRACE && p.tok.Type != token.T_EOF {
				p.nextToken()
			}
			if p.tok.Type == token.T_COMMA {
				p.nextToken()
				continue
			}
			// T_RPAREN, T_LBRACE, or T_EOF — stop parameter parsing
			break
		}
		params = append(params, param)

		if p.tok.Type == token.T_COMMA {
			p.nextToken()
			// Allow trailing comma before )
			for p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT || p.tok.Type == token.T_WHITESPACE {
				p.nextToken()
			}
			if p.tok.Type == token.T_RPAREN {
				break
			}
		}
	}
	p.nextToken() // consume )

	if name == "" && p.tok.Type == token.T_USE {
		p.nextToken() // consume use
		if p.tok.Type != token.T_LPAREN {
			p.addError("line %d:%d: expected ( after closure use, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
			return nil, nil
		}
		p.nextToken() // consume (
		for p.tok.Type != token.T_RPAREN && p.tok.Type != token.T_EOF {
			if p.tok.Type == token.T_AMPERSAND {
				p.nextToken()
			}
			if p.tok.Type != token.T_VARIABLE {
				p.addError("line %d:%d: expected closure use variable, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
				return nil, nil
			}
			p.nextToken()
			if p.tok.Type == token.T_COMMA {
				p.nextToken()
				continue
			}
		}
		if p.tok.Type != token.T_RPAREN {
			p.addError("line %d:%d: expected ) after closure use list, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
			return nil, nil
		}
		p.nextToken() // consume )
	}

	// Parse return type hint
	var returnType string
	if p.tok.Type == token.T_COLON {
		p.nextToken()
		// Accept static, self, parent as return types
		if p.tok.Type == token.T_STATIC || p.tok.Type == token.T_SELF || p.tok.Type == token.T_PARENT {
			returnType = p.tok.Literal
			p.nextToken()
		} else {
			returnType = p.parseTypeHint()
		}
	}

	// Skip whitespace, comments, and attributes before function body
	for p.tok.Type == token.T_WHITESPACE || p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT || p.tok.Type == token.T_ATTRIBUTE {
		p.nextToken()
	}

	for _, modifier := range savedModifiers {
		if modifier == "abstract" && p.tok.Type == token.T_SEMICOLON {
			p.nextToken() // consume ;
			return &ast.FunctionNode{
				Name:       name,
				Params:     params,
				ReturnType: returnType,
				Modifiers:  savedModifiers,
				Body:       nil,
				PHPDoc:     p.consumeCurrentDoc(pos),
				Pos:        ast.Position(pos),
			}, nil
		}
	}

	// Parse function body
	if p.tok.Type != token.T_LBRACE {
		p.addError("line %d:%d: expected { to start function body for %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
		p.syncToNextClassMember()
		return nil, nil
	}
	p.nextToken() // consume {

	var body []ast.Node
	braceDepth := 1
	for braceDepth > 0 && p.tok.Type != token.T_EOF {
		if p.tok.Type == token.T_LBRACE {
			braceDepth++
			p.nextToken()
			continue
		}
		if p.tok.Type == token.T_RBRACE {
			braceDepth--
			if braceDepth == 0 {
				break
			}
			p.nextToken()
			continue
		}
		prevOffset := p.tok.Pos.Offset
		stmt, err := p.parseStatement()
		if stmt != nil {
			body = append(body, stmt)
		}
		if err != nil {
			p.addError(err.Error())
		}
		// Safety: only advance if parseStatement did not consume any token
		if stmt == nil && p.tok.Pos.Offset == prevOffset {
			p.nextToken()
		}
	}

	if p.tok.Type == token.T_RBRACE {
		p.nextToken() // consume }
	} else {
		p.addError("line %d:%d: expected } to close function %s body, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
		p.syncToNextClassMember()
		return nil, nil
	}

	return &ast.FunctionNode{
		Name:       name,
		Params:     params,
		ReturnType: returnType,
		Modifiers:  savedModifiers,
		Body:       body,
		PHPDoc:     p.consumeCurrentDoc(pos),
		Pos:        ast.Position(pos),
	}, nil
}
