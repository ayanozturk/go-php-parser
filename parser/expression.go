package parser

import (
	"go-phpcs/ast"
	"go-phpcs/token"
)

func (p *Parser) parseExpression() ast.Node {
	return p.parseExpressionWithPrecedence(0, true)
}

// parseExpressionWithPrecedence parses expressions with correct precedence. Only validateAssignmentTarget for top-level expressions.
func (p *Parser) parseExpressionWithPrecedence(minPrec int, validateAssignmentTarget bool, stopTypes ...token.TokenType) ast.Node {
	if p.debug {

	}

	// Array literals
	if p.tok.Type == token.T_LBRACKET || p.tok.Type == token.T_ARRAY {
		return p.parseArrayLiteral()
	}

	// Handle unary operators
	switch p.tok.Type {
	case token.T_PLUS, token.T_MINUS:
		return p.parseUnaryExpression()
	case token.T_NOT:
		return p.parseNotExpression()
	case token.T_THROW:
		return p.parseThrowExpression()
	case token.T_STRING:
		if p.tok.Literal == "!" {
			return p.parseUnaryExpression()
		}
	}

	left := p.parseSimpleExpression()
	if left == nil {
		p.addError("line %d:%d: expected left operand, got nil (error recovery)", p.tok.Pos.Line, p.tok.Pos.Column)
		// Error recovery: skip to next semicolon or closing parenthesis
		for p.tok.Type != token.T_SEMICOLON && p.tok.Type != token.T_RPAREN && p.tok.Type != token.T_EOF {
			p.nextToken()
		}
		if p.tok.Type == token.T_SEMICOLON {
			p.nextToken()
		}
		return nil
	}
	for {
		// Only parse ternary if minPrec <= precedence of ternary
		ternaryPrec := PhpOperatorPrecedence[token.T_QUESTION]
		if p.tok.Type == token.T_QUESTION && minPrec <= ternaryPrec {
			left = p.parseTernaryExpression(left, ternaryPrec)
			continue
		}
		// Check for stop tokens
		for _, stop := range stopTypes {
			if p.tok.Type == stop {
				return left
			}
		}
		prec, isOp := PhpOperatorPrecedence[p.tok.Type]
		if !isOp || prec < minPrec {
			break
		}
		op := p.tok.Type
		operator := p.tok.Literal
		if op == token.T_BOOLEAN_OR {
			operator = "||"
		}
		pos := p.tok.Pos
		assocRight := PhpOperatorRightAssoc[op]
		if p.debug {

		}
		nextMinPrec := prec + 1
		if assocRight {
			nextMinPrec = prec
		}
		p.nextToken()
		var right ast.Node
		if op == token.T_BOOLEAN_OR || op == token.T_BOOLEAN_AND {
			// Logical operators
			right = p.parseExpressionWithPrecedence(0, true, stopTypes...)
		} else if op == token.T_INSTANCEOF {
			// For instanceof, right side must be a class name or FQCN
			right = p.parseSimpleExpression()
		} else {
			right = p.parseExpressionWithPrecedence(nextMinPrec, false, stopTypes...)
		}
		if p.debug {

		}
		if right == nil {
			p.addError("line %d:%d: expected right operand after operator %s", pos.Line, pos.Column, operator)
			return nil
		}
		// Only validate assignment target for the outermost assignment (not for nested assignments in logical expressions)
		if isAssignmentOperator(op) && validateAssignmentTarget && minPrec == 0 {
			if !isValidAssignmentTarget(left) {
				p.addError("line %d:%d: invalid assignment target for operator %s", pos.Line, pos.Column, operator)
				return nil
			}
		}
		if isAssignmentOperator(op) {
			left = &ast.AssignmentNode{
				Left:  left,
				Right: right,
				Pos:   ast.Position(pos),
			}
		} else {
			left = &ast.BinaryExpr{
				Left:     left,
				Operator: operator,
				Right:    right,
				Pos:      ast.Position(pos),
			}
		}
	}
	return left
}

func (p *Parser) parseUnaryExpression() ast.Node {
	opTok := p.tok
	p.nextToken()
	right := p.parseExpressionWithPrecedence(100, false)
	if right == nil {
		p.addError("line %d:%d: expected operand after unary operator %s", opTok.Pos.Line, opTok.Pos.Column, opTok.Literal)
		return nil
	}
	return &ast.UnaryExpr{
		Operator: opTok.Literal,
		Operand:  right,
		Pos:      ast.Position(opTok.Pos),
	}
}

func (p *Parser) parseNotExpression() ast.Node {
	notTok := p.tok
	p.nextToken()
	right := p.parseExpressionWithPrecedence(100, false)
	if right == nil {
		p.addError("line %d:%d: expected operand after unary operator !", notTok.Pos.Line, notTok.Pos.Column)
		return nil
	}
	return &ast.UnaryExpr{
		Operator: "!",
		Operand:  right,
		Pos:      ast.Position(notTok.Pos),
	}
}

func (p *Parser) parseThrowExpression() ast.Node {
	throwTok := p.tok
	p.nextToken() // consume 'throw'
	expr := p.parseExpressionWithPrecedence(100, false)
	if expr == nil {
		p.addError("line %d:%d: expected expression after throw, got %s", throwTok.Pos.Line, throwTok.Pos.Column, p.tok.Literal)
		return nil
	}
	return &ast.ThrowNode{
		Expr: expr,
		Pos:  ast.Position(throwTok.Pos),
	}
}

func (p *Parser) parseTernaryExpression(left ast.Node, ternaryPrec int) ast.Node {
	qPos := p.tok.Pos
	p.nextToken() // consume '?'
	ifTrue := p.parseExpressionWithPrecedence(0, false)
	if p.tok.Type != token.T_COLON {
		p.addError("line %d:%d: expected ':' in ternary expression, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil
	}
	p.nextToken() // consume ':'
	ifFalse := p.parseExpressionWithPrecedence(ternaryPrec, false)
	return &ast.TernaryExpr{
		Condition: left,
		IfTrue:    ifTrue,
		IfFalse:   ifFalse,
		Pos:       ast.Position(qPos),
	}
}
