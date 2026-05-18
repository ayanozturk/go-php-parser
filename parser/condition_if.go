package parser

import (
	"go-phpcs/ast"
	"go-phpcs/token"
)

func (p *Parser) parseIfStatement() (ast.Node, error) {
	pos := p.tok.Pos
	p.nextToken() // consume if

	if p.tok.Type != token.T_LPAREN {
		p.addError("line %d:%d: expected ( after if, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	p.nextToken()

	condition := p.parseExpression()
	if condition == nil {
		return nil, nil
	}

	if p.tok.Type != token.T_RPAREN {
		p.addError("line %d:%d: expected ) after if condition, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	p.nextToken()

	body, err := p.parseConditionalBody("if")
	if err != nil {
		return nil, err
	}
	for p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT || p.tok.Type == token.T_WHITESPACE {
		p.nextToken()
	}

	var elseifs []*ast.ElseIfNode
	var elseNode *ast.ElseNode

	// Parse any elseif clauses
	for p.tok.Type == token.T_ELSEIF {
		elseifNode, err := p.parseElseIfClause()
		if elseifNode == nil || err != nil {
			return nil, err
		}
		elseifs = append(elseifs, elseifNode)
	}

	// Parse optional else clause
	if p.tok.Type == token.T_ELSE {
		var err error
		elseNode, err = p.parseElseClause()
		if elseNode == nil || err != nil {
			return nil, err
		}
	}

	return &ast.IfNode{
		Condition: condition,
		Body:      body,
		ElseIfs:   elseifs,
		Else:      elseNode,
		Pos:       ast.Position(pos),
	}, nil
}

func (p *Parser) parseElseIfClause() (*ast.ElseIfNode, error) {
	pos := p.tok.Pos
	p.nextToken() // consume elseif

	if p.tok.Type != token.T_LPAREN {
		p.addError("line %d:%d: expected ( after elseif, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	p.nextToken()

	condition := p.parseExpression()
	if condition == nil {
		return nil, nil
	}

	if p.tok.Type != token.T_RPAREN {
		p.addError("line %d:%d: expected ) after elseif condition, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	p.nextToken()

	body, err := p.parseConditionalBody("elseif")
	if err != nil {
		return nil, err
	}
	for p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT || p.tok.Type == token.T_WHITESPACE {
		p.nextToken()
	}

	return &ast.ElseIfNode{
		Condition: condition,
		Body:      body,
		Pos:       ast.Position(pos),
	}, nil
}

func (p *Parser) parseElseClause() (*ast.ElseNode, error) {
	pos := p.tok.Pos
	p.nextToken() // consume else

	body, err := p.parseConditionalBody("else")
	if err != nil {
		return nil, err
	}
	for p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT || p.tok.Type == token.T_WHITESPACE {
		p.nextToken()
	}

	return &ast.ElseNode{
		Body: body,
		Pos:  ast.Position(pos),
	}, nil
}

func (p *Parser) parseConditionalBody(keyword string) ([]ast.Node, error) {
	if p.tok.Type == token.T_LBRACE {
		p.nextToken()
		body := p.parseBlockStatement()
		if p.tok.Type != token.T_RBRACE {
			p.addError("line %d:%d: expected } to close %s body, got %s", p.tok.Pos.Line, p.tok.Pos.Column, keyword, p.tok.Literal)
			return nil, nil
		}
		p.nextToken()
		return body, nil
	}

	stmt, err := p.parseStatement()
	if err != nil {
		return nil, err
	}
	if stmt == nil {
		return nil, nil
	}
	return []ast.Node{stmt}, nil
}
