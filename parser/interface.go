package parser

import (
	"fmt"
	"go-phpcs/ast"
	"go-phpcs/token"
)

// parseInterfaceDeclaration parses a PHP interface declaration
func (p *Parser) parseInterfaceDeclaration() ast.Node {
	pos := p.tok.Pos
	p.nextToken() // consume 'interface'

	if p.tok.Type != token.T_STRING {
		p.errors = append(p.errors, "expected interface name")
		return nil
	}

	name := p.tok.Literal
	p.nextToken()

	if p.tok.Type != token.T_LBRACE {
		p.errors = append(p.errors, "expected { after interface name")
		return nil
	}
	p.nextToken() // consume {

	var methods []ast.Node
	for p.tok.Type != token.T_RBRACE && p.tok.Type != token.T_EOF {
		// Interface methods can have visibility modifiers
		if p.tok.Type == token.T_PUBLIC || p.tok.Type == token.T_PRIVATE || p.tok.Type == token.T_PROTECTED {
			if method := p.parseFunctionDeclaration(); method != nil {
				methods = append(methods, method)
			}
		} else if p.tok.Type == token.T_FUNCTION {
			if method := p.parseFunctionDeclaration(); method != nil {
				methods = append(methods, method)
			}
		} else {
			p.errors = append(p.errors, fmt.Sprintf("unexpected token %s in interface body", p.tok.Type))
			p.nextToken()
		}
	}

	if p.tok.Type != token.T_RBRACE {
		p.errors = append(p.errors, "expected } to close interface body")
		return nil
	}
	p.nextToken() // consume }

	return &ast.InterfaceNode{
		Name:    name,
		Methods: methods,
		Pos:     ast.Position(pos),
	}
}
