package parser

import (
	"errors"
	"fmt"

	"go-phpcs/ast"
	"go-phpcs/lexer"
)

// Parser converts tokens into an abstract syntax tree
type Parser struct {
	lexer      *lexer.Lexer
	tokens     []lexer.Token
	pos        int
	errorNodes []*ast.Error
}

// New creates a new parser instance
func New(lex *lexer.Lexer) *Parser {
	return &Parser{
		lexer:      lex,
		tokens:     make([]lexer.Token, 0),
		errorNodes: make([]*ast.Error, 0),
	}
}

// Parse parses the input and returns an AST
func (p *Parser) Parse() (*ast.Program, error) {
	// Initialize tokens
	p.fetchTokens()

	// Parse statements until we reach the end
	statements := make([]ast.Stmt, 0)

	for !p.isAtEnd() {
		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}

		if stmt != nil {
			statements = append(statements, stmt)
		}
	}

	// Create and return the program node
	program := &ast.Program{
		Statements: statements,
	}

	if len(p.errorNodes) > 0 {
		return program, fmt.Errorf("parse completed with %d errors", len(p.errorNodes))
	}

	return program, nil
}

// fetchTokens reads all tokens from the lexer
func (p *Parser) fetchTokens() {
	for {
		token := p.lexer.GetNextToken()
		p.tokens = append(p.tokens, token)

		if token.Type == lexer.T_EOF {
			break
		}
	}
}

// isAtEnd checks if we've reached the end of the token stream
func (p *Parser) isAtEnd() bool {
	return p.pos >= len(p.tokens) || p.tokens[p.pos].Type == lexer.T_EOF
}

// parseStatement parses a single statement
func (p *Parser) parseStatement() (ast.Stmt, error) {
	// Here we would have a large switch statement to handle different statement types
	// ...existing code...

	return nil, errors.New("not implemented")
}

// Helper methods for token handling
func (p *Parser) advance() lexer.Token {
	if !p.isAtEnd() {
		p.pos++
	}
	return p.previous()
}

func (p *Parser) current() lexer.Token {
	if p.isAtEnd() {
		return p.tokens[len(p.tokens)-1]
	}
	return p.tokens[p.pos]
}

func (p *Parser) previous() lexer.Token {
	return p.tokens[p.pos-1]
}
