package parser

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/token"
)

func (p *Parser) parseSwitchStatement() (ast.Node, error) {
	pos := p.tok.Pos
	p.nextToken() // consume switch

	if p.tok.Type != token.T_LPAREN {
		p.addError("line %d:%d: expected ( after switch, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume (

	expr := p.parseExpressionWithStop(token.T_RPAREN)
	if expr == nil {
		p.addError("line %d:%d: expected expression after switch (", p.tok.Pos.Line, p.tok.Pos.Column)
		return nil, nil
	}
	if p.tok.Type != token.T_RPAREN {
		p.addError("line %d:%d: expected ) after switch expression, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume )

	if p.tok.Type != token.T_LBRACE && p.tok.Type != token.T_COLON {
		p.addError("line %d:%d: expected { after switch, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	altSyntax := p.tok.Type == token.T_COLON
	p.nextToken() // consume { or :

	closeTok := token.T_RBRACE
	if altSyntax {
		closeTok = token.T_ENDSWITCH
	}

	var cases []*ast.SwitchCaseNode
	for p.tok.Type != closeTok && p.tok.Type != token.T_EOF {
		for p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT || p.tok.Type == token.T_OPEN_TAG || p.tok.Type == token.T_CLOSE_TAG {
			p.nextToken()
		}
		if p.tok.Type == closeTok {
			break
		}
		if p.tok.Type != token.T_CASE && p.tok.Type != token.T_DEFAULT {
			p.addError("line %d:%d: expected case or default in switch, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
			return nil, nil
		}

		casePos := p.tok.Pos
		switchCase := &ast.SwitchCaseNode{IsDefault: p.tok.Type == token.T_DEFAULT, Pos: ast.Position(casePos)}
		p.nextToken() // consume case/default

		if !switchCase.IsDefault {
			switchCase.Expr = p.parseExpressionWithStop(token.T_COLON, token.T_SEMICOLON)
			if switchCase.Expr == nil {
				p.addError("line %d:%d: expected expression after case", p.tok.Pos.Line, p.tok.Pos.Column)
				return nil, nil
			}
		}
		if p.tok.Type != token.T_COLON && p.tok.Type != token.T_SEMICOLON {
			p.addError("line %d:%d: expected : after switch case, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
			return nil, nil
		}
		p.nextToken() // consume : or ;

		for p.tok.Type != token.T_CASE && p.tok.Type != token.T_DEFAULT && p.tok.Type != closeTok && p.tok.Type != token.T_EOF {
			// Consume open/close PHP tag transitions ourselves so we return
			// to this loop's own stop-condition check (case/default/close)
			// after each one, rather than letting parseStatement's internal
			// retry jump straight from an open tag into whatever follows
			// (which could be the very "}"/"case" that should end this body).
			if p.tok.Type == token.T_OPEN_TAG || p.tok.Type == token.T_CLOSE_TAG {
				p.nextToken()
				continue
			}
			stmt, err := p.parseStatement()
			if err != nil {
				return nil, err
			}
			if stmt != nil {
				switchCase.Body = append(switchCase.Body, stmt)
			}
		}
		cases = append(cases, switchCase)
	}

	if p.tok.Type != closeTok {
		p.addError("line %d:%d: expected } to close switch, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume } or endswitch

	if altSyntax {
		if !p.consumeAltTerminator("endswitch") {
			return nil, nil
		}
	}

	return &ast.SwitchNode{Expr: expr, Cases: cases, Pos: ast.Position(pos)}, nil
}
