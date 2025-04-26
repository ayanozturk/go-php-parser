package parser

import "go-phpcs/token"

// expect checks if the current token matches the expected type. If so, advances to the next token and returns true.
// Otherwise, adds an error and returns false.
func (p *Parser) expect(expected token.TokenType) bool {
	if p.tok.Type == expected {
		p.nextToken()
		return true
	}
	// p.addError("line %d:%d: expected %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, expected, p.tok.Type)
	return false
}
