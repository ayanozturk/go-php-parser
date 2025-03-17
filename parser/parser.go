package parser

import (
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"go-phpcs/token"
)

type Parser struct {
	l      *lexer.Lexer
	tok    token.Token
	errors []string
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
	}
	p.nextToken() // Initialize first token
	return p
}

func (p *Parser) nextToken() {
	p.tok = p.l.NextToken()
}

func (p *Parser) Parse() []ast.Node {
	var nodes []ast.Node

	// Expect PHP open tag first
	if p.tok.Type != token.T_OPEN_TAG {
		p.errors = append(p.errors, "expected <?php at start of file")
		return nodes
	}
	p.nextToken()

	for p.tok.Type != token.T_EOF {
		if node := p.parseStatement(); node != nil {
			nodes = append(nodes, node)
		}
		p.nextToken()
	}

	return nodes
}

func (p *Parser) parseStatement() ast.Node {
	switch p.tok.Type {
	case token.T_FUNCTION:
		return p.parseFunctionDeclaration()
	case token.T_VARIABLE:
		return p.parseVariableStatement()
	default:
		return nil
	}
}

func (p *Parser) parseFunctionDeclaration() ast.Node {
	pos := p.tok.Pos
	p.nextToken() // consume 'function'

	if p.tok.Type != token.T_IDENTIFIER {
		p.errors = append(p.errors, "expected function name")
		return nil
	}

	name := p.tok.Literal
	p.nextToken()

	if p.tok.Type != token.T_LPAREN {
		p.errors = append(p.errors, "expected ( after function name")
		return nil
	}
	p.nextToken()

	// Parse parameters (to be implemented)
	if p.tok.Type != token.T_RPAREN {
		p.errors = append(p.errors, "expected )")
		return nil
	}
	p.nextToken()

	// Parse function body
	if p.tok.Type != token.T_LBRACE {
		p.errors = append(p.errors, "expected { after function parameters")
		return nil
	}
	p.nextToken()

	var body []ast.Node
	for p.tok.Type != token.T_RBRACE && p.tok.Type != token.T_EOF {
		if stmt := p.parseStatement(); stmt != nil {
			body = append(body, stmt)
		}
		p.nextToken()
	}

	if p.tok.Type != token.T_RBRACE {
		p.errors = append(p.errors, "expected } to close function body")
		return nil
	}

	return &ast.FunctionNode{
		Name:   name,
		Params: nil, // TODO: Add parameter parsing
		Body:   body,
		Pos:    ast.Position(pos),
	}
}

func (p *Parser) parseVariableStatement() ast.Node {
	varPos := p.tok.Pos
	varName := p.tok.Literal[1:] // Remove leading $ from variable name
	p.nextToken()

	// If this is an assignment
	if p.tok.Type == token.T_ASSIGN {
		p.nextToken() // consume =
		right := p.parseExpression()
		if right == nil {
			return nil
		}

		if p.tok.Type != token.T_SEMICOLON {
			p.errors = append(p.errors, "expected ; after assignment")
			return nil
		}

		return &ast.AssignmentNode{
			Left: &ast.VariableNode{
				Name: varName,
				Pos:  ast.Position(varPos),
			},
			Right: right,
			Pos:   ast.Position(varPos),
		}
	}

	return &ast.VariableNode{
		Name: varName,
		Pos:  ast.Position(varPos),
	}
}

func (p *Parser) parseExpression() ast.Node {
	switch p.tok.Type {
	case token.T_CONSTANT_STRING:
		strPos := p.tok.Pos
		str := p.tok.Literal
		p.nextToken()
		return &ast.LiteralNode{
			Value: str,
			Pos:   ast.Position(strPos),
		}
	case token.T_VARIABLE:
		return p.parseVariableStatement()
	default:
		p.errors = append(p.errors, "unexpected token in expression")
		return nil
	}
}

func (p *Parser) Errors() []string {
	return p.errors
}
