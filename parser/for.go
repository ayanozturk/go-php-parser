package parser

import (
	"go-phpcs/ast"
	"go-phpcs/token"
)

// parseForStatement parses a minimal PHP for-loop:
// for (init; cond; post) { ... }
// init/cond/post are parsed as generic expressions with permissive stopping rules.
func (p *Parser) parseForStatement() (ast.Node, error) {
	pos := p.tok.Pos
	p.nextToken() // consume 'for'

	if p.tok.Type != token.T_LPAREN {
		p.addError("line %d:%d: expected ( after for, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	// Tolerant skip until the matching ')', to avoid strict expression parsing for for-control
	depth := 1
	for depth > 0 {
		p.nextToken()
		if p.tok.Type == token.T_LPAREN {
			depth++
		} else if p.tok.Type == token.T_RPAREN {
			depth--
		} else if p.tok.Type == token.T_EOF {
			p.addError("line %d:%d: unexpected EOF in for control", p.tok.Pos.Line, p.tok.Pos.Column)
			return nil, nil
		}
	}
	p.nextToken() // consume token after ')'

	// Body
	var body []ast.Node
	if p.tok.Type == token.T_LBRACE {
		p.nextToken() // consume '{'
		body = p.parseBlockStatement()
		if p.tok.Type != token.T_RBRACE {
			p.addError("line %d:%d: expected } to close for body, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
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

	// Represent for-loop as a BlockNode wrapper to keep AST stable without new node type
	return &ast.BlockNode{Statements: body, Pos: ast.Position(pos)}, nil
}
