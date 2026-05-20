package parser

import (
	"fmt"
	"go-phpcs/ast"
	"go-phpcs/token"
)

// parseEnum parses an enum declaration
func (p *Parser) parseEnum() (*ast.EnumNode, error) {
	pos := p.tok.Pos
	p.nextToken() // consume "enum"

	// Get enum name
	if p.tok.Type != token.T_STRING {
		return nil, fmt.Errorf("expected enum name, got %s", p.tok.Type)
	}
	name := p.tok.Literal
	p.nextToken()

	// Skip trailing comments/whitespace before colon or opening brace
	p.skipCommentsAndWhitespace()

	// Check for backed enum type
	var backedBy string
	if p.tok.Type == token.T_COLON {
		p.nextToken() // consume ":"
		if p.tok.Type != token.T_STRING {
			return nil, fmt.Errorf("expected enum backing type, got %s", p.tok.Type)
		}
		backedBy = p.tok.Literal
		p.nextToken()
	}

	// Skip trailing comments/whitespace before opening brace
	p.skipCommentsAndWhitespace()

	// Expect opening brace
	if p.tok.Type != token.T_LBRACE {
		return nil, fmt.Errorf("expected {, got %s", p.tok.Type)
	}
	p.nextToken()

	// Parse enum body
	var cases []*ast.EnumCaseNode
	var methods []ast.Node
	for p.tok.Type != token.T_RBRACE && p.tok.Type != token.T_EOF {
		for p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT || p.tok.Type == token.T_ATTRIBUTE {
			p.nextToken()
		}
		if p.tok.Type == token.T_CASE {
			enumCase, err := p.parseEnumCase()
			if err != nil {
				return nil, err
			}
			cases = append(cases, enumCase)
			continue
		}

		modifiers := p.parseModifiers()
		if p.tok.Type == token.T_FUNCTION {
			method, err := p.parseFunction(modifiers)
			if err != nil {
				return nil, err
			}
			if method != nil {
				methods = append(methods, method)
			}
			continue
		} else {
			p.nextToken()
		}
	}

	// Consume closing brace
	p.nextToken()

	return &ast.EnumNode{
		Name:     name,
		BackedBy: backedBy,
		Cases:    cases,
		Methods:  methods,
		Pos:      ast.Position(pos),
	}, nil
}

// parseEnumCase parses a single enum case
func (p *Parser) parseEnumCase() (*ast.EnumCaseNode, error) {
	pos := p.tok.Pos
	p.nextToken() // consume "case"

	// Get case name
	if p.tok.Type != token.T_STRING {
		return nil, fmt.Errorf("expected case name, got %s", p.tok.Type)
	}
	name := p.tok.Literal
	p.nextToken()

	// Check for value (for backed enums)
	var value ast.Node
	if p.tok.Type == token.T_ASSIGN {
		p.nextToken() // consume "="
		value = p.parseExpression()
		if value == nil {
			return nil, fmt.Errorf("expected value after = in enum case")
		}
	}

	// Expect semicolon
	if p.tok.Type != token.T_SEMICOLON {
		return nil, fmt.Errorf("expected ;, got %s", p.tok.Type)
	}
	p.nextToken()

	return &ast.EnumCaseNode{
		Name:  name,
		Value: value,
		Pos:   ast.Position(pos),
	}, nil
}
