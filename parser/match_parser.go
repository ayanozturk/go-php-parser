package parser

import (
	"go-phpcs/ast"
	"go-phpcs/token"
)

// parseMatchExpression parses a PHP 8.0+ match expression
// Syntax: match (condition) { value1[, value2]* => expression [, ...] }
func (p *Parser) parseMatchExpression() ast.Node {
	matchPos := p.tok.Pos

	// Expect 'match' keyword
	if !p.expect(token.T_MATCH) {
		return nil
	}

	// Expect opening parenthesis
	if !p.expect(token.T_LPAREN) {
		p.addError("line %d:%d: expected '(' after 'match'", p.tok.Pos.Line, p.tok.Pos.Column)
		return nil
	}

	// Parse condition expression
	condition := p.parseExpression()
	if condition == nil {
		p.addError("line %d:%d: expected condition expression in match", p.tok.Pos.Line, p.tok.Pos.Column)
		return nil
	}

	// Expect closing parenthesis
	if !p.expect(token.T_RPAREN) {
		p.addError("line %d:%d: expected ')' after match condition", p.tok.Pos.Line, p.tok.Pos.Column)
		return nil
	}

	// Expect opening brace
	if !p.expect(token.T_LBRACE) {
		p.addError("line %d:%d: expected '{' after match condition", p.tok.Pos.Line, p.tok.Pos.Column)
		return nil
	}

	var arms []ast.MatchArmNode

	// Parse match arms
	for p.tok.Type != token.T_RBRACE && p.tok.Type != token.T_EOF {
		arm := p.parseMatchArm()
		if arm == nil {
			return nil
		}
		arms = append(arms, *arm)

		// Handle optional comma after arm (trailing comma support)
		if p.tok.Type == token.T_COMMA {
			p.nextToken()
		} else if p.tok.Type != token.T_RBRACE {
			p.addError("line %d:%d: expected ',' or '}' after match arm", p.tok.Pos.Line, p.tok.Pos.Column)
			return nil
		}
	}

	// Expect closing brace
	if !p.expect(token.T_RBRACE) {
		p.addError("line %d:%d: expected '}' to close match expression", p.tok.Pos.Line, p.tok.Pos.Column)
		return nil
	}

	return &ast.MatchNode{
		Condition: condition,
		Arms:      arms,
		Pos:       ast.Position(matchPos),
	}
}

// parseMatchArm parses a single match arm
// Syntax: condition1[, condition2]* => expression
func (p *Parser) parseMatchArm() *ast.MatchArmNode {
	armPos := p.tok.Pos
	var conditions []ast.Node

	// Handle default case
	if p.tok.Type == token.T_DEFAULT {
		defaultNode := &ast.IdentifierNode{
			Value: "default",
			Pos:   ast.Position(p.tok.Pos),
		}
		conditions = append(conditions, defaultNode)
		p.nextToken()
	} else {
		// Parse conditions (can be multiple separated by commas)
		for {
			condition := p.parseExpression()
			if condition == nil {
				p.addError("line %d:%d: expected condition in match arm", p.tok.Pos.Line, p.tok.Pos.Column)
				return nil
			}
			conditions = append(conditions, condition)

			// Check for comma (multiple conditions)
			if p.tok.Type == token.T_COMMA {
				p.nextToken()
			} else {
				break
			}
		}
	}

	// Expect arrow operator
	if !p.expect(token.T_DOUBLE_ARROW) {
		p.addError("line %d:%d: expected '=>' after match conditions", p.tok.Pos.Line, p.tok.Pos.Column)
		return nil
	}

	// Parse body expression
	body := p.parseExpression()
	if body == nil {
		p.addError("line %d:%d: expected expression after '=>' in match arm", p.tok.Pos.Line, p.tok.Pos.Column)
		return nil
	}

	return &ast.MatchArmNode{
		Conditions: conditions,
		Body:       body,
		Pos:        ast.Position(armPos),
	}
}
