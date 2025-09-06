package parser

import (
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"go-phpcs/token"
)

type Parser struct {
	l          *lexer.Lexer
	tok        token.Token
	errors     []error
	debug      bool
	currentDoc string // Current PHPDoc comment being tracked
}

func New(l *lexer.Lexer, debug bool) *Parser {
	p := &Parser{
		l:      l,
		errors: []error{},
		debug:  debug,
	}
	p.nextToken() // Initialize first token
	return p
}

func (p *Parser) nextToken() {
	p.tok = p.l.NextToken()
}

func (p *Parser) addError(format string, args ...interface{}) {
	if p.debug {
		p.errors = append(p.errors, ErrorDeferred{Format: format, Args: args})
	}
}

// Errors returns the list of errors encountered during parsing
func (p *Parser) Errors() []string {
	res := make([]string, len(p.errors))
	for i, err := range p.errors {
		res[i] = err.Error()
	}
	return res
}

// consumeCurrentDoc consumes the current PHPDoc comment and returns a PHPDocNode
func (p *Parser) consumeCurrentDoc(pos token.Position) *ast.PHPDocNode {
	if p.currentDoc == "" {
		return nil
	}
	phpdoc := ast.ExtractPHPDocFromComment(p.currentDoc)
	if phpdoc != nil {
		phpdoc.Pos = ast.Position(pos)
	}
	p.currentDoc = "" // Clear the current doc
	return phpdoc
}

func (p *Parser) Parse() []ast.Node {
	// Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			p.addError("Parser panic: %v", r)
		}
	}()

	var nodes []ast.Node

	// Expect PHP open tag first
	if p.tok.Type != token.T_OPEN_TAG {
		p.addError("line %d:%d: expected <?php at start of file, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nodes
	}
	p.nextToken()

	// Skip whitespace/comments after open tag (but not doc comments - let statement parsing handle them)
	for p.tok.Type == token.T_WHITESPACE || p.tok.Type == token.T_COMMENT {
		p.nextToken()
	}

	for p.tok.Type != token.T_EOF {
		// Skip whitespace/comments between statements (but not doc comments - let statement parsing handle them)
		for p.tok.Type == token.T_WHITESPACE || p.tok.Type == token.T_COMMENT {
			p.nextToken()
		}
		if p.tok.Type == token.T_EOF {
			break
		}
		node, err := p.parseStatement()
		if err != nil {
			p.addError(err.Error())
			p.nextToken() // Ensure forward progress
			continue
		}
		if node != nil {
			nodes = append(nodes, node)
		}
	}

	return nodes
}

// peekToken returns the next token without consuming it
func (p *Parser) peekToken() token.Token {
	return p.l.PeekToken()
}

// parseSimpleExpression parses a simple expression (identifier, literal, etc.)
// parseFQCN parses a fully qualified class name, e.g. \Foo\Bar
func (p *Parser) parseFQCN() ast.Node {
	pos := p.tok.Pos
	fqcn := ""
	for {
		if p.tok.Type == token.T_NS_SEPARATOR {
			fqcn += "\\"
			p.nextToken()
		}
		if p.tok.Type == token.T_STRING || p.tok.Type == token.T_STATIC || p.tok.Type == token.T_SELF || p.tok.Type == token.T_PARENT {
			fqcn += p.tok.Literal
			p.nextToken()
		} else {
			break
		}
	}
	if fqcn == "" {
		p.addError("line %d:%d: expected fully qualified class name, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil
	}

	return &ast.IdentifierNode{
		Value: fqcn,
		Pos:   ast.Position(pos),
	}
}
