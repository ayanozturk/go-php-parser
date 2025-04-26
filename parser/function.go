package parser

import (
	"go-phpcs/ast"
	"go-phpcs/token"
)

// parseFunction parses a PHP function declaration
func (p *Parser) parseFunction(modifiers []string) (ast.Node, error) {
	pos := p.tok.Pos
	p.nextToken() // consume 'function'

	var name string
	if p.tok.Type == token.T_STRING {
		name = p.tok.Literal
		p.nextToken()
	}

	if p.tok.Type != token.T_LPAREN {
		p.addError("line %d:%d: expected ( after function name %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume (

	var params []ast.Node
	for p.tok.Type != token.T_RPAREN {
		param := p.parseParameter()
		if param == nil {
			// Ensure forward progress to avoid infinite loop
			p.nextToken()
			continue
		}
		params = append(params, param)

		if p.tok.Type == token.T_COMMA {
			p.nextToken()
		}
	}
	p.nextToken() // consume )

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

	if p.tok.Type != token.T_LBRACE {
		p.addError("line %d:%d: expected { to start function body for %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume {

	body := p.parseBlockStatement()

	if p.tok.Type != token.T_RBRACE {
		p.addError("line %d:%d: expected } to close function %s body, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume }

	return &ast.FunctionNode{
		Name:       name,
		Params:     params,
		ReturnType: returnType,
		Body:       body,
		Modifiers:  modifiers,
		Pos:        ast.Position(pos),
	}, nil
}
