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
	if p.tok.Type == token.T_LBRACKET || p.tok.Type == token.T_ARRAY {
		return p.parseArrayLiteral()
	}

	if node := p.parseUnaryExpression(stopTypes...); node != nil {
		return node
	}

	left := p.parseSimpleExpression()
	if left == nil {
		return p.recoverFromExpressionError()
	}

	return p.parseBinaryAndTernaryOperators(left, minPrec, validateAssignmentTarget, stopTypes...)
}

// parseUnaryExpression handles unary and throw expressions.
func (p *Parser) parseUnaryExpression(stopTypes ...token.TokenType) ast.Node {
	switch p.tok.Type {
	case token.T_PLUS, token.T_MINUS:
		opTok := p.tok
		p.nextToken()
		right := p.parseExpressionWithPrecedence(100, false, stopTypes...)
		if right == nil {
			p.addError("line %d:%d: expected operand after unary operator %s", opTok.Pos.Line, opTok.Pos.Column, opTok.Literal)
			return nil
		}
		return &ast.UnaryExpr{
			Operator: opTok.Literal,
			Operand:  right,
			Pos:      ast.Position(opTok.Pos),
		}
	case token.T_NOT:
		notTok := p.tok
		p.nextToken()
		right := p.parseExpressionWithPrecedence(100, false, stopTypes...)
		if right == nil {
			p.addError("line %d:%d: expected operand after unary operator !", notTok.Pos.Line, notTok.Pos.Column)
			return nil
		}
		return &ast.UnaryExpr{
			Operator: "!",
			Operand:  right,
			Pos:      ast.Position(notTok.Pos),
		}
	case token.T_THROW:
		throwTok := p.tok
		p.nextToken()
		expr := p.parseExpressionWithPrecedence(100, false, stopTypes...)
		if expr == nil {
			p.addError("line %d:%d: expected expression after throw, got %s", throwTok.Pos.Line, throwTok.Pos.Column, p.tok.Literal)
			return nil
		}
		return &ast.ThrowNode{
			Expr: expr,
			Pos:  ast.Position(throwTok.Pos),
		}
	case token.T_STRING:
		if p.tok.Literal == "!" {
			opTok := p.tok
			p.nextToken()
			right := p.parseExpressionWithPrecedence(100, false, stopTypes...)
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
	}
	return nil
}

// parseBinaryAndTernaryOperators handles binary and ternary expressions.
func (p *Parser) parseBinaryAndTernaryOperators(left ast.Node, minPrec int, validateAssignmentTarget bool, stopTypes ...token.TokenType) ast.Node {
	for {
		// Ternary
		ternaryPrec := PhpOperatorPrecedence[token.T_QUESTION]
		if p.tok.Type == token.T_QUESTION && minPrec <= ternaryPrec {
			left = p.parseTernaryExpression(left, ternaryPrec)
			continue
		}
		// Stop tokens
		for _, stop := range stopTypes {
			if p.tok.Type == stop {
				return left
			}
		}
		prec, isOp := PhpOperatorPrecedence[p.tok.Type]
		if !isOp || prec < minPrec {
			break
		}
		left = p.parseBinaryOperator(left, prec, validateAssignmentTarget, stopTypes...)
		if left == nil {
			return nil
		}
	}
	return left
}

// parseTernaryExpression handles ternary expressions.
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

// parseBinaryOperator handles a single binary operator application.
func (p *Parser) parseBinaryOperator(left ast.Node, prec int, validateAssignmentTarget bool, stopTypes ...token.TokenType) ast.Node {
	op := p.tok.Type
	operator := p.tok.Literal
	if op == token.T_BOOLEAN_OR {
		operator = "||"
	}
	pos := p.tok.Pos
	assocRight := PhpOperatorRightAssoc[op]

	nextMinPrec := prec + 1
	if assocRight {
		nextMinPrec = prec
	}
	p.nextToken()
	var right ast.Node
	if op == token.T_BOOLEAN_OR || op == token.T_BOOLEAN_AND {
		right = p.parseExpressionWithPrecedence(0, true, stopTypes...)
	} else if op == token.T_INSTANCEOF {
		right = p.parseSimpleExpression()
	} else {
		right = p.parseExpressionWithPrecedence(nextMinPrec, false, stopTypes...)
	}
	if right == nil {
		p.addError("line %d:%d: expected right operand after operator %s", pos.Line, pos.Column, operator)
		return nil
	}
	if isAssignmentOperator(op) && validateAssignmentTarget && prec == 0 {
		if !isValidAssignmentTarget(left) {
			p.addError("line %d:%d: invalid assignment target for operator %s", pos.Line, pos.Column, operator)
			return nil
		}
	}
	if isAssignmentOperator(op) {
		return &ast.AssignmentNode{
			Left:  left,
			Right: right,
			Pos:   ast.Position(pos),
		}
	}
	return &ast.BinaryExpr{
		Left:     left,
		Operator: operator,
		Right:    right,
		Pos:      ast.Position(pos),
	}
}

// recoverFromExpressionError handles error recovery for invalid expressions.
func (p *Parser) recoverFromExpressionError() ast.Node {
	p.addError("line %d:%d: expected left operand, got nil (error recovery)", p.tok.Pos.Line, p.tok.Pos.Column)
	for p.tok.Type != token.T_SEMICOLON && p.tok.Type != token.T_RPAREN && p.tok.Type != token.T_EOF {
		p.nextToken()
	}
	if p.tok.Type == token.T_SEMICOLON {
		p.nextToken()
	}
	return nil
}
