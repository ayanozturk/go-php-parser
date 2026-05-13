package parser

import (
	"go-phpcs/ast"
	"go-phpcs/token"
)

func (p *Parser) parseTryStatement() (ast.Node, error) {
	pos := p.tok.Pos
	p.nextToken() // consume try
	if p.tok.Type != token.T_LBRACE {
		p.addError("line %d:%d: expected { after try, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume {
	body := p.parseBlockStatement()
	if p.tok.Type != token.T_RBRACE {
		p.addError("line %d:%d: expected } to close try body, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume }

	var catches []*ast.CatchNode
	for p.tok.Type == token.T_CATCH {
		catchNode, err := p.parseCatchClause()
		if err != nil {
			return nil, err
		}
		if catchNode == nil {
			return nil, nil
		}
		catches = append(catches, catchNode)
	}
	if len(catches) == 0 {
		p.addError("line %d:%d: expected catch after try block, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}

	return &ast.TryNode{Body: body, Catches: catches, Pos: ast.Position(pos)}, nil
}

func (p *Parser) parseCatchClause() (*ast.CatchNode, error) {
	pos := p.tok.Pos
	p.nextToken() // consume catch
	if p.tok.Type != token.T_LPAREN {
		p.addError("line %d:%d: expected ( after catch, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume (

	var types []string
	for {
		typeNode := p.parseFQCN()
		identifier, ok := typeNode.(*ast.IdentifierNode)
		if !ok || identifier == nil {
			p.addError("line %d:%d: expected catch type, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
			return nil, nil
		}
		types = append(types, identifier.Value)
		if p.tok.Type != token.T_PIPE {
			break
		}
		p.nextToken() // consume |
	}

	if p.tok.Type != token.T_VARIABLE {
		p.addError("line %d:%d: expected catch variable, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	variable := p.tok.Literal[1:]
	p.nextToken()

	if p.tok.Type != token.T_RPAREN {
		p.addError("line %d:%d: expected ) after catch signature, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume )

	if p.tok.Type != token.T_LBRACE {
		p.addError("line %d:%d: expected { after catch signature, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume {
	body := p.parseBlockStatement()
	if p.tok.Type != token.T_RBRACE {
		p.addError("line %d:%d: expected } to close catch body, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume }

	return &ast.CatchNode{Types: types, Variable: variable, Body: body, Pos: ast.Position(pos)}, nil
}