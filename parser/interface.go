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
			if method := p.parseInterfaceMethod(); method != nil {
				methods = append(methods, method)
			}
		} else if p.tok.Type == token.T_FUNCTION {
			if method := p.parseInterfaceMethod(); method != nil {
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

// parseInterfaceMethod parses a method declaration in an interface
func (p *Parser) parseInterfaceMethod() ast.Node {
	pos := p.tok.Pos

	// Parse visibility modifier if present
	var visibility string
	if p.tok.Type == token.T_PUBLIC || p.tok.Type == token.T_PRIVATE || p.tok.Type == token.T_PROTECTED {
		visibility = p.tok.Literal
		p.nextToken()
	}

	p.nextToken() // consume 'function'

	if p.tok.Type != token.T_STRING {
		p.errors = append(p.errors, "expected method name")
		return nil
	}

	name := p.tok.Literal
	p.nextToken()

	if p.tok.Type != token.T_LPAREN {
		p.errors = append(p.errors, "expected ( after method name")
		return nil
	}
	p.nextToken() // consume (

	// Parse parameters
	var params []ast.Node
	for p.tok.Type != token.T_RPAREN && p.tok.Type != token.T_EOF {
		if param := p.parseParameter(); param != nil {
			params = append(params, param)
		}

		if p.tok.Type == token.T_COMMA {
			p.nextToken()
			continue
		}

		if p.tok.Type != token.T_RPAREN {
			p.errors = append(p.errors, "expected , or ) in parameter list")
			return nil
		}
	}
	p.nextToken() // consume )

	// Parse return type if present
	var returnType string
	if p.tok.Type == token.T_COLON {
		p.nextToken() // consume :
		if p.tok.Type != token.T_STRING {
			p.errors = append(p.errors, "expected return type after :")
			return nil
		}
		returnType = p.tok.Literal
		p.nextToken()
	}

	// Interface methods must end with a semicolon
	if p.tok.Type != token.T_SEMICOLON {
		p.errors = append(p.errors, "expected ; after interface method declaration")
		return nil
	}
	p.nextToken() // consume ;

	return &ast.InterfaceMethodNode{
		Name:       name,
		Visibility: visibility,
		ReturnType: returnType,
		Params:     params,
		Pos:        ast.Position(pos),
	}
}
