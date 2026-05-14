package parser

import (
	"go-phpcs/ast"
	"go-phpcs/token"
)

func (p *Parser) parseWhileStatement() (ast.Node, error) {
	pos := p.tok.Pos
	p.nextToken() // consume while

	if p.tok.Type != token.T_LPAREN {
		p.addError("line %d:%d: expected ( after while, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume (

	condition := p.parseExpressionWithStop(token.T_RPAREN)
	if condition == nil {
		p.addError("line %d:%d: expected condition after while (", p.tok.Pos.Line, p.tok.Pos.Column)
		return nil, nil
	}
	if p.tok.Type != token.T_RPAREN {
		p.addError("line %d:%d: expected ) after while condition, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume )

	body, err := p.parseLoopBody("while")
	if err != nil {
		return nil, err
	}

	return &ast.WhileNode{Condition: condition, Body: body, Pos: ast.Position(pos)}, nil
}

func (p *Parser) parseDoWhileStatement() (ast.Node, error) {
	pos := p.tok.Pos
	p.nextToken() // consume do

	body, err := p.parseLoopBody("do")
	if err != nil {
		return nil, err
	}
	if p.tok.Type != token.T_WHILE {
		p.addError("line %d:%d: expected while after do body, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume while

	if p.tok.Type != token.T_LPAREN {
		p.addError("line %d:%d: expected ( after while in do-while, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume (

	condition := p.parseExpressionWithStop(token.T_RPAREN)
	if condition == nil {
		p.addError("line %d:%d: expected condition after while ( in do-while", p.tok.Pos.Line, p.tok.Pos.Column)
		return nil, nil
	}
	if p.tok.Type != token.T_RPAREN {
		p.addError("line %d:%d: expected ) after do-while condition, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume )
	if p.tok.Type != token.T_SEMICOLON {
		p.addError("line %d:%d: expected ; after do-while, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume ;

	return &ast.DoWhileNode{Condition: condition, Body: body, Pos: ast.Position(pos)}, nil
}

func (p *Parser) parseLoopBody(loopName string) ([]ast.Node, error) {
	var body []ast.Node
	if p.tok.Type == token.T_LBRACE {
		p.nextToken() // consume {
		body = p.parseBlockStatement()
		if p.tok.Type != token.T_RBRACE {
			p.addError("line %d:%d: expected } to close %s body, got %s", p.tok.Pos.Line, p.tok.Pos.Column, loopName, p.tok.Literal)
			return nil, nil
		}
		p.nextToken() // consume }
		return body, nil
	}

	stmt, err := p.parseStatement()
	if err != nil {
		return nil, err
	}
	if stmt != nil {
		body = append(body, stmt)
	}
	return body, nil
}
