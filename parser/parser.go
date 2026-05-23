package parser

import (
	"context"
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"go-phpcs/token"
	"strings"
)

type Parser struct {
	Ctx context.Context
	// SkipFunctionBodies parses signatures and class members but skips statement
	// bodies. Indexers use this to build symbol tables without paying for a full
	// analysis-grade AST.
	SkipFunctionBodies bool
	l                  *lexer.Lexer
	tok                token.Token
	errors             []error
	debug              bool
	currentDoc         string // Current PHPDoc comment being tracked
	modifierBuf        []string
	nameBuf            strings.Builder
}

func New(l *lexer.Lexer, debug bool) *Parser {
	p := &Parser{
		l:           l,
		errors:      []error{},
		debug:       debug,
		modifierBuf: make([]string, 0, 4),
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

	for p.tok.Type == token.T_WHITESPACE || p.tok.Type == token.T_COMMENT {
		p.nextToken()
	}

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
		if p.Ctx != nil && p.Ctx.Err() != nil {
			p.addError("parser context cancelled: %v", p.Ctx.Err())
			break
		}
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
	p.nameBuf.Reset()
	for {
		if p.tok.Type == token.T_NS_SEPARATOR {
			p.nameBuf.WriteString("\\")
			p.nextToken()
		}
		if p.tok.Type == token.T_STRING || p.tok.Type == token.T_STATIC || p.tok.Type == token.T_SELF || p.tok.Type == token.T_PARENT {
			p.nameBuf.WriteString(p.tok.Literal)
			p.nextToken()
		} else {
			break
		}
	}
	fqcn := p.nameBuf.String()
	if fqcn == "" {
		p.addError("line %d:%d: expected fully qualified class name, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil
	}

	return &ast.IdentifierNode{
		Value: fqcn,
		Pos:   ast.Position(pos),
	}
}

// parseModifiers parses and returns member modifiers, reusing the internal modifierBuf.
func (p *Parser) parseModifiers() []string {
	p.modifierBuf = p.modifierBuf[:0]
	for {
		if modifier, ok := p.parsePropertyModifier(); ok {
			p.modifierBuf = append(p.modifierBuf, modifier)
			continue
		}
		switch p.tok.Type {
		case token.T_PUBLIC, token.T_PROTECTED, token.T_PRIVATE, token.T_STATIC, token.T_FINAL, token.T_ABSTRACT:
			p.modifierBuf = append(p.modifierBuf, p.tok.Literal)
			p.nextToken()
			continue
		case token.T_COMMENT, token.T_DOC_COMMENT, token.T_ATTRIBUTE:
			p.nextToken()
			continue
		}
		break
	}
	return p.modifierBuf
}
