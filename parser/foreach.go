package parser

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/token"
)

// parseExpressionWithStop parses an expression up to (but not including) a stop token.
func (p *Parser) parseExpressionWithStop(stopTypes ...token.TokenType) ast.Node {
	return p.parseExpressionWithPrecedenceStop(0, true, stopTypes...)
}

// parseExpressionWithPrecedenceStop is like parseExpressionWithPrecedence but stops if it sees a stop token.
func (p *Parser) parseExpressionWithPrecedenceStop(minPrec int, validateAssignmentTarget bool, stopTypes ...token.TokenType) ast.Node {
	prevBuf := p.stopBuf
	prevLen := p.stopLen
	p.stopLen = copy(p.stopBuf[:], stopTypes)
	res := p.parseExpressionWithPrecedence(minPrec, validateAssignmentTarget)
	p.stopBuf = prevBuf
	p.stopLen = prevLen
	return res
}

// parseForeachStatement parses a PHP foreach statement
func (p *Parser) parseForeachStatement() (ast.Node, error) {
	pos := p.tok.Pos
	p.nextToken() // consume 'foreach'

	if p.tok.Type != token.T_LPAREN {
		p.addError("line %d:%d: expected ( after foreach, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume '('

	// Use the new helper to parse the expression up to 'as'
	expr := p.parseExpressionWithStop(token.T_AS)
	if expr == nil {
		p.addError("line %d:%d: expected expression after foreach (", p.tok.Pos.Line, p.tok.Pos.Column)
		return nil, nil
	}
	if p.tok.Type != token.T_AS {
		p.addError("line %d:%d: expected 'as' after foreach expression, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume 'as'

	// Parse key => value or just value, with optional &
	var keyVar ast.Node
	var valueVar ast.Node
	byRef := false

	// Helper to parse a variable, optionally extended with property/array-index
	// access, e.g. "$this->webUser" or "$stub->position" as a foreach target.
	parseVar := func() ast.Node {
		if p.tok.Type == token.T_VARIABLE {
			varName := p.tok.Literal[1:]
			varPos := p.tok.Pos
			p.nextToken()
			node := ast.Node(&ast.VariableNode{Name: varName, Pos: ast.Position(varPos)})
			return p.parseNewClassPostfixExpression(node)
		}
		return nil
	}

	parseValueTarget := func() ast.Node {
		if p.tok.Type == token.T_VARIABLE {
			return parseVar()
		}
		if p.tok.Type == token.T_LBRACKET {
			return p.parseArrayLiteral(true)
		}
		if p.tok.Type == token.T_LIST {
			return p.parseListLiteral(true)
		}
		return nil
	}

	if p.tok.Type == token.T_AMPERSAND {
		// Could be foreach ($arr as &$v) or foreach ($arr as $k => &$v)
		p.nextToken()
		if p.tok.Type == token.T_VARIABLE {
			// Peek ahead: if next is =>, this is key, not value
			if p.peekToken().Type == token.T_DOUBLE_ARROW {
				// &var is key, so parse key, then =>, then value (possibly by-ref)
				keyVar = &ast.UnaryExpr{Operator: "&", Operand: parseVar(), Pos: ast.Position(p.tok.Pos)}
				p.nextToken() // consume =>
				if p.tok.Type == token.T_AMPERSAND {
					byRef = true
					p.nextToken()
				}
				valueVar = parseVar()
			} else {
				// &var is value
				byRef = true
				valueVar = parseVar()
			}
		} else {
			p.addError("line %d:%d: expected variable after & in foreach", p.tok.Pos.Line, p.tok.Pos.Column)
			return nil, nil
		}
	} else if p.tok.Type == token.T_VARIABLE {
		// Could be key or value
		varNode := parseVar()
		if p.tok.Type == token.T_DOUBLE_ARROW {
			// This variable is the key
			keyVar = varNode
			p.nextToken() // consume =>
			if p.tok.Type == token.T_AMPERSAND {
				byRef = true
				p.nextToken()
			}
			valueVar = parseValueTarget()
		} else {
			// This variable is the value
			valueVar = varNode
		}
	} else if p.tok.Type == token.T_LBRACKET || p.tok.Type == token.T_LIST {
		valueVar = parseValueTarget()
	} else {
		p.addError("line %d:%d: expected variable, destructuring target, or & after 'as' in foreach, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}

	if valueVar == nil {
		p.addError("line %d:%d: expected foreach target after 'as' or '=>', got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}

	if p.tok.Type != token.T_RPAREN {
		p.addError("line %d:%d: expected ) after foreach variables, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume ')'

	if p.tok.Type == token.T_COLON {
		p.nextToken() // consume ':'
		body := p.parseAltBody(token.T_ENDFOREACH)
		if p.tok.Type != token.T_ENDFOREACH {
			p.addError("line %d:%d: expected endforeach to close alternative foreach syntax, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
			return nil, nil
		}
		p.nextToken() // consume endforeach
		if !p.consumeAltTerminator("endforeach") {
			return nil, nil
		}
		return &ast.ForeachNode{
			Expr:     expr,
			KeyVar:   keyVar,
			ValueVar: valueVar,
			ByRef:    byRef,
			Body:     body,
			Pos:      ast.Position(pos),
		}, nil
	}

	// Parse body (block or single statement)
	var body []ast.Node
	if p.tok.Type == token.T_LBRACE {
		p.nextToken() // consume '{'
		body = p.parseBlockStatement()
		if p.tok.Type != token.T_RBRACE {
			p.addError("line %d:%d: expected } to close foreach body, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
			return nil, nil
		}
		p.nextToken() // consume '}'
	} else {
		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		if stmt != nil {
			body = append(body, stmt)
		}
	}

	return &ast.ForeachNode{
		Expr:     expr,
		KeyVar:   keyVar,
		ValueVar: valueVar,
		ByRef:    byRef,
		Body:     body,
		Pos:      ast.Position(pos),
	}, nil
}
