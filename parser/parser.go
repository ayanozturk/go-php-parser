package parser

import (
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"go-phpcs/token"
)

type Parser struct {
	l   *lexer.Lexer
	tok token.Token
}

func New(l *lexer.Lexer) *Parser {
	return &Parser{l: l, tok: l.NextToken()}
}

func (p *Parser) Parse() []ast.Node {
	var nodes []ast.Node
	for p.tok.Type != token.T_EOF {
		// very basic: just a stub
		p.tok = p.l.NextToken()
	}
	return nodes
}
