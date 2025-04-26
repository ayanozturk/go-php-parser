package parser

import (
	"go-phpcs/ast"
	"go-phpcs/token"
)

// parseNamespaceDeclaration parses a PHP namespace declaration (block or inline)
func (p *Parser) parseNamespaceDeclaration() (ast.Node, error) {
	pos := p.tok.Pos
	p.nextToken() // consume 'namespace'

	// Parse namespace name (can be multiple T_STRING separated by T_NS_SEPARATOR)
	name := ""
	for {
		if p.tok.Type == token.T_STRING {
			if name != "" {
				name += "\\"
			}
			name += p.tok.Literal
			p.nextToken()
		} else if p.tok.Type == token.T_NS_SEPARATOR {
			p.nextToken()
			continue
		} else {
			break
		}
	}

	// Inline namespace: namespace Foo\\Bar;
	if p.tok.Type == token.T_SEMICOLON {
		p.nextToken() // consume ;
		return &ast.NamespaceNode{
			Name: name,
			Body: nil,
			Pos:  ast.Position(pos),
		}, nil
	}

	// Block namespace: namespace Foo\\Bar { ... }
	if p.tok.Type == token.T_LBRACE {
		p.nextToken() // consume {
		var body []ast.Node
		for p.tok.Type != token.T_RBRACE && p.tok.Type != token.T_EOF {
			stmt, err := p.parseStatement()
			if err != nil {
				p.addError(err.Error())
				p.nextToken()
				continue
			}
			if stmt != nil {
				body = append(body, stmt)
			}
		}
		if p.tok.Type != token.T_RBRACE {
			p.addError("line %d:%d: expected } to close namespace %s body, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
			return nil, nil
		}
		p.nextToken() // consume }
		return &ast.NamespaceNode{
			Name: name,
			Body: body,
			Pos:  ast.Position(pos),
		}, nil
	}

	p.addError("line %d:%d: expected ; or { after namespace name, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
	return nil, nil
}
