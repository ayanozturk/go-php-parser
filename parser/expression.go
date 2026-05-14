package parser

import (
	"go-phpcs/ast"
	"go-phpcs/token"
	"strconv"
	"strings"
)

const errExpectedRParenFunctionCall = "line %d:%d: expected ) after arguments for function call %s, got %s"

func (p *Parser) parseExpression() ast.Node {
	return p.parseExpressionWithPrecedence(0, true)
}

// parseExpressionWithPrecedence parses expressions with correct precedence. Only validateAssignmentTarget for top-level expressions.
func (p *Parser) parseExpressionWithPrecedence(minPrec int, validateAssignmentTarget bool, stopTypes ...token.TokenType) ast.Node {
	for p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT || p.tok.Type == token.T_WHITESPACE {
		p.nextToken()
	}
	if p.tok.Type == token.T_LBRACKET || p.tok.Type == token.T_ARRAY {
		left := p.parseArrayLiteral(validateAssignmentTarget)
		if left == nil {
			return nil
		}
		left = p.parsePostfixExpression(left)
		return p.parseBinaryAndTernaryOperators(left, minPrec, validateAssignmentTarget, stopTypes...)
	}
	if p.tok.Type == token.T_LIST && validateAssignmentTarget {
		left := p.parseListLiteral(true)
		if left == nil {
			return nil
		}
		return p.parseBinaryAndTernaryOperators(left, minPrec, validateAssignmentTarget, stopTypes...)
	}

	if left := p.parseUnaryExpression(stopTypes...); left != nil {
		return p.parseBinaryAndTernaryOperators(left, minPrec, validateAssignmentTarget, stopTypes...)
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
	case token.T_INC, token.T_DEC:
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
	case token.T_AT:
		atTok := p.tok
		p.nextToken()
		right := p.parseExpressionWithPrecedence(100, false, stopTypes...)
		if right == nil {
			p.addError("line %d:%d: expected operand after unary operator @", atTok.Pos.Line, atTok.Pos.Column)
			return nil
		}
		return &ast.UnaryExpr{
			Operator: "@",
			Operand:  right,
			Pos:      ast.Position(atTok.Pos),
		}
	case token.T_AMPERSAND:
		ampTok := p.tok
		p.nextToken()
		right := p.parseExpressionWithPrecedence(100, false, stopTypes...)
		if right == nil {
			p.addError("line %d:%d: expected operand after unary operator &", ampTok.Pos.Line, ampTok.Pos.Column)
			return nil
		}
		return &ast.UnaryExpr{
			Operator: "&",
			Operand:  right,
			Pos:      ast.Position(ampTok.Pos),
		}
	case token.T_TILDE:
		tildeTok := p.tok
		p.nextToken()
		right := p.parseExpressionWithPrecedence(100, false, stopTypes...)
		if right == nil {
			p.addError("line %d:%d: expected operand after unary operator ~", tildeTok.Pos.Line, tildeTok.Pos.Column)
			return nil
		}
		return &ast.UnaryExpr{
			Operator: "~",
			Operand:  right,
			Pos:      ast.Position(tildeTok.Pos),
		}
	case token.T_CLONE:
		cloneTok := p.tok
		p.nextToken()
		right := p.parseExpressionWithPrecedence(100, false, stopTypes...)
		if right == nil {
			p.addError("line %d:%d: expected operand after clone", cloneTok.Pos.Line, cloneTok.Pos.Column)
			return nil
		}
		return &ast.UnaryExpr{
			Operator: "clone",
			Operand:  right,
			Pos:      ast.Position(cloneTok.Pos),
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
		for p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT || p.tok.Type == token.T_WHITESPACE {
			p.nextToken()
		}
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
	ifTrue := left
	if p.tok.Type != token.T_COLON {
		ifTrue = p.parseExpressionWithPrecedence(0, false)
	}
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
	if !isAssignmentOperator(op) && isAssignmentOperator(p.tok.Type) && isValidAssignmentTarget(right) {
		right = p.parseBinaryOperator(right, PhpOperatorPrecedence[p.tok.Type], false, stopTypes...)
		if right == nil {
			return nil
		}
	}
	if isAssignmentOperator(op) {
		if unary, ok := left.(*ast.UnaryExpr); ok && unary.Operator == "!" && isValidAssignmentTarget(unary.Operand) {
			assignment := &ast.AssignmentNode{
				Left:  unary.Operand,
				Right: right,
				Pos:   ast.Position(pos),
			}
			return &ast.UnaryExpr{
				Operator: unary.Operator,
				Operand:  assignment,
				Pos:      unary.Pos,
			}
		}
	}
	if isAssignmentOperator(op) && validateAssignmentTarget {
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

func (p *Parser) parseSimpleExpression() ast.Node {
	// Handle unary minus and plus
	if p.tok.Type == token.T_MINUS || p.tok.Type == token.T_PLUS {
		return p.parseSimpleUnary()
	}

	switch p.tok.Type {
	case token.T_NEW:
		return p.parseSimpleNew()
	case token.T_FUNCTION:
		if fn, err := p.parseFunction(nil); err == nil {
			if fn != nil {
				return fn
			}
		}
		return nil
	case token.T_FN:
		return p.parseArrowFunction()
	case token.T_STATIC:
		if p.peekToken().Type == token.T_FN {
			return p.parseArrowFunction()
		}
		if p.peekToken().Type == token.T_FUNCTION {
			p.nextToken() // consume static
			if fn, err := p.parseFunction([]string{"static"}); err == nil {
				if fn != nil {
					return fn
				}
			}
			return nil
		}
		return p.parseSimpleFQCNOrFunctionCall()
	case token.T_STRING, token.T_SELF, token.T_PARENT, token.T_NS_SEPARATOR:
		return p.parseSimpleFQCNOrFunctionCall()
	case token.T_CONSTANT_ENCAPSED_STRING:
		return p.parseSimpleStringOrConcat()
	case token.T_CONSTANT_STRING:
		return p.parseSimpleConstantString()
	case token.T_START_HEREDOC, token.T_START_NOWDOC:
		return p.parseSimpleHeredoc()
	case token.T_LNUMBER:
		return p.parseSimpleLNumber()
	case token.T_DNUMBER:
		return p.parseSimpleDNumber()
	case token.T_TRUE, token.T_FALSE, token.T_NULL:
		return p.parseSimpleBoolOrNull()
	case token.T_VARIABLE:
		return p.parseSimpleVariable()
	case token.T_YIELD:
		return p.parseSimpleYieldExpression()
	case token.T_ISSET, token.T_EMPTY, token.T_EXIT, token.T_DIE:
		return p.parseSimpleBuiltinCall()
	case token.T_INCLUDE, token.T_INCLUDE_ONCE, token.T_REQUIRE, token.T_REQUIRE_ONCE:
		return p.parseSimpleIncludeExpression()
	case token.T_LPAREN:
		return p.parseGroupedExpression()
	case token.T_MATCH:
		return p.parseMatchExpression()
	// case token.T_NS_SEPARATOR: (now handled above)
	default:
		return p.parseSimpleUnexpected()
	}
}

// --- Helper methods for parseSimpleExpression ---

func (p *Parser) parseSimpleUnary() ast.Node {
	op := p.tok.Type
	pos := p.tok.Pos
	p.nextToken()
	right := p.parseSimpleExpression()
	if intNode, ok := right.(*ast.IntegerNode); ok {
		if op == token.T_MINUS {
			intNode.Value = -intNode.Value
		}
		intNode.Pos = ast.Position(pos)
		return intNode
	} else if floatNode, ok := right.(*ast.FloatNode); ok {
		if op == token.T_MINUS {
			floatNode.Value = -floatNode.Value
		}
		floatNode.Pos = ast.Position(pos)
		return floatNode
	} else {
		return right
	}
}

func (p *Parser) parseSimpleNew() ast.Node {
	pos := p.tok.Pos
	p.nextToken() // consume 'new'
	if p.tok.Type == token.T_CLASS {
		classExpr, args := p.parseAnonymousClassExpression()
		if classExpr == nil {
			return nil
		}
		return &ast.NewNode{
			ClassExpr: classExpr,
			Args:      args,
			Pos:       ast.Position(pos),
		}
	}
	if p.tok.Type != token.T_STRING && p.tok.Type != token.T_STATIC && p.tok.Type != token.T_SELF && p.tok.Type != token.T_PARENT && p.tok.Type != token.T_NS_SEPARATOR && p.tok.Type != token.T_VARIABLE && p.tok.Type != token.T_LPAREN {
		p.addError("line %d:%d: expected class name after new, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil
	}
	className := ""
	var classExpr ast.Node
	if p.tok.Type == token.T_VARIABLE {
		classExpr = &ast.VariableNode{Name: p.tok.Literal[1:], Pos: ast.Position(p.tok.Pos)}
		p.nextToken()
		classExpr = p.parseNewClassPostfixExpression(classExpr)
		if variable, ok := classExpr.(*ast.VariableNode); ok {
			className = "$" + variable.Name
			classExpr = nil
		}
	} else if p.tok.Type == token.T_LPAREN {
		p.nextToken() // consume (
		classExpr = p.parseExpressionWithStop(token.T_RPAREN)
		if classExpr == nil {
			p.addError("line %d:%d: expected class expression after new (", p.tok.Pos.Line, p.tok.Pos.Column)
			return nil
		}
		if p.tok.Type != token.T_RPAREN {
			p.addError("line %d:%d: expected ) after dynamic class expression, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
			return nil
		}
		p.nextToken() // consume )
	} else {
		classNameNode := p.parseFQCN()
		if id, ok := classNameNode.(*ast.IdentifierNode); ok {
			className = id.Value
		} else {
			p.addError("line %d:%d: expected identifier node for class name after new", p.tok.Pos.Line, p.tok.Pos.Column)
			return nil
		}
	}
	var args []ast.Node
	if p.tok.Type == token.T_LPAREN {
		p.nextToken() // consume (
		args = p.parseFunctionCallArguments()
		if p.tok.Type != token.T_RPAREN {
			p.addError("line %d:%d: expected ) after arguments for %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, className, p.tok.Literal)
			return nil
		}
		p.nextToken() // consume )
	}
	return &ast.NewNode{
		ClassName: className,
		ClassExpr: classExpr,
		Args:      args,
		Pos:       ast.Position(pos),
	}
}

func (p *Parser) parseNewClassPostfixExpression(expr ast.Node) ast.Node {
	for {
		for p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT || p.tok.Type == token.T_WHITESPACE {
			p.nextToken()
		}
		if p.tok.Type == token.T_OBJECT_OPERATOR || p.tok.Type == token.T_NULLSAFE_OBJECT_OPERATOR {
			expr = p.parseSimpleObjectOrMethod(expr)
			continue
		}
		if p.tok.Type == token.T_DOUBLE_COLON {
			expr = p.parseStaticAccessOnNode(expr)
			continue
		}
		if p.tok.Type == token.T_LBRACKET {
			expr = p.parseSimpleArrayAccess(expr)
			continue
		}
		break
	}
	return expr
}

func (p *Parser) parseSimpleFQCNOrFunctionCall() ast.Node {
	fqcnPos := p.tok.Pos
	var fqcnBuilder strings.Builder
	// Handle leading \ for fully qualified names
	if p.tok.Type == token.T_NS_SEPARATOR {
		fqcnBuilder.WriteString("\\")
		p.nextToken()
	}
	fqcnBuilder.WriteString(p.collectFQCNString())
	fqcn := fqcnBuilder.String()
	if p.tok.Type == token.T_DOUBLE_COLON {
		return p.parseSimpleStaticAccess(fqcn, fqcnPos)
	}
	var expr ast.Node = &ast.IdentifierNode{
		Value: fqcn,
		Pos:   ast.Position(fqcnPos),
	}
	if p.tok.Type == token.T_LPAREN {
		// Check if this is first-class callable syntax: name(...)
		p.nextToken() // consume '('
		if p.tok.Type == token.T_ELLIPSIS {
			if p.peekToken().Type == token.T_RPAREN {
				p.nextToken() // consume '...'
				p.nextToken() // consume ')'
				expr = &ast.FirstClassCallableNode{
					Name: &ast.IdentifierNode{
						Value: fqcn,
						Pos:   ast.Position(fqcnPos),
					},
					Pos: ast.Position(fqcnPos),
				}
				return p.parsePostfixExpression(expr)
			}
		}
		// Not first-class callable, parse as regular function call
		args := p.parseFunctionCallArguments()
		if p.tok.Type != token.T_RPAREN {
			p.addError(errExpectedRParenFunctionCall, p.tok.Pos.Line, p.tok.Pos.Column, fqcn, p.tok.Literal)
			return nil
		}
		p.nextToken() // consume )
		expr = &ast.FunctionCallNode{
			Name: expr,
			Args: args,
			Pos:  ast.Position(fqcnPos),
		}
		return p.parsePostfixExpression(expr)
	}
	return p.parsePostfixExpression(expr)
}

func (p *Parser) collectFQCNString() string {
	var fqcnBuilder strings.Builder
	for {
		if p.tok.Type == token.T_NS_SEPARATOR {
			fqcnBuilder.WriteString("\\")
			p.nextToken()
		} else if p.tok.Type == token.T_STRING || p.tok.Type == token.T_STATIC || p.tok.Type == token.T_SELF || p.tok.Type == token.T_PARENT {
			fqcnBuilder.WriteString(p.tok.Literal)
			p.nextToken()
		} else {
			break
		}
	}
	return fqcnBuilder.String()
}

func (p *Parser) parseSimpleStaticAccess(fqcn string, fqcnPos token.Position) ast.Node {
	p.nextToken() // consume '::'
	if p.tok.Type == token.T_VARIABLE {
		memberName := p.tok.Literal
		p.nextToken()
		return p.parsePostfixExpression(&ast.ClassConstFetchNode{
			Class: fqcn,
			Const: memberName,
			Pos:   ast.Position(fqcnPos),
		})
	}
	if p.tok.Type == token.T_STRING || isValidMethodNameToken(p.tok.Type) {
		memberName := p.tok.Literal
		p.nextToken()
		if p.tok.Type == token.T_LPAREN {
			p.nextToken() // consume '('
			args := p.parseFunctionCallArguments()
			if p.tok.Type != token.T_RPAREN {
				p.addError("line %d:%d: expected ) after arguments for static call %s::%s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, fqcn, memberName, p.tok.Literal)
				return nil
			}
			p.nextToken() // consume ')'
			return p.parsePostfixExpression(&ast.FunctionCallNode{
				Name: &ast.IdentifierNode{
					Value: fqcn + "::" + memberName,
					Pos:   ast.Position(fqcnPos),
				},
				Args: args,
				Pos:  ast.Position(fqcnPos),
			})
		}
		return p.parsePostfixExpression(&ast.ClassConstFetchNode{
			Class: fqcn,
			Const: memberName,
			Pos:   ast.Position(fqcnPos),
		})
	} else if p.tok.Type == token.T_CLASS_CONST || p.tok.Type == token.T_CLASS {
		p.nextToken()
		return p.parsePostfixExpression(&ast.ClassConstFetchNode{
			Class: fqcn,
			Const: "class",
			Pos:   ast.Position(fqcnPos),
		})
	} else {
		p.addError("line %d:%d: expected constant name or 'class' after '::', got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil
	}
}

func (p *Parser) parseArrowFunction() ast.Node {
	pos := p.tok.Pos
	if p.tok.Type == token.T_STATIC {
		p.nextToken()
	}
	if p.tok.Type != token.T_FN {
		p.addError("line %d:%d: expected fn after static, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil
	}
	p.nextToken() // consume 'fn'
	if p.tok.Type != token.T_LPAREN {
		p.addError("line %d:%d: expected ( after fn, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil
	}
	p.nextToken() // consume '('

	var params []ast.Node
	for p.tok.Type != token.T_RPAREN {
		for p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT || p.tok.Type == token.T_WHITESPACE {
			p.nextToken()
		}
		if p.tok.Type == token.T_RPAREN {
			break
		}
		param := p.parseParameter()
		if param == nil {
			for p.tok.Type != token.T_COMMA && p.tok.Type != token.T_RPAREN && p.tok.Type != token.T_EOF {
				p.nextToken()
			}
			if p.tok.Type == token.T_COMMA {
				p.nextToken()
				continue
			}
			break
		}
		params = append(params, param)
		if p.tok.Type == token.T_COMMA {
			p.nextToken()
			for p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT || p.tok.Type == token.T_WHITESPACE {
				p.nextToken()
			}
			if p.tok.Type == token.T_RPAREN {
				break
			}
		}
	}
	if p.tok.Type != token.T_RPAREN {
		p.addError("line %d:%d: expected ) after arrow function parameters, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil
	}
	p.nextToken() // consume ')'

	var returnType string
	if p.tok.Type == token.T_COLON {
		p.nextToken()
		returnType = p.parseTypeHint()
	}

	if p.tok.Type != token.T_DOUBLE_ARROW {
		p.addError("line %d:%d: expected => after arrow function signature, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil
	}
	p.nextToken() // consume '=>'

	body := p.parseExpression()
	if body == nil {
		p.addError("line %d:%d: expected expression after => in arrow function", p.tok.Pos.Line, p.tok.Pos.Column)
		return nil
	}

	return &ast.ArrowFunctionNode{
		Params:     params,
		ReturnType: returnType,
		Expr:       body,
		Pos:        ast.Position(pos),
	}
}

func (p *Parser) parseSimpleFunctionCall(fqcn string, fqcnPos token.Position) ast.Node {
	p.nextToken() // consume '('
	args := p.parseFunctionCallArguments()
	if p.tok.Type != token.T_RPAREN {
		p.addError(errExpectedRParenFunctionCall, p.tok.Pos.Line, p.tok.Pos.Column, fqcn, p.tok.Literal)
		return nil
	}
	p.nextToken() // consume )
	return &ast.FunctionCallNode{
		Name: &ast.IdentifierNode{
			Value: fqcn,
			Pos:   ast.Position(fqcnPos),
		},
		Args: args,
		Pos:  ast.Position(fqcnPos),
	}
}

// parseSimpleFunctionCallWithConsumedParen parses a function call when '(' has already been consumed
func (p *Parser) parseSimpleFunctionCallWithConsumedParen(fqcn string, fqcnPos token.Position) ast.Node {
	args := p.parseFunctionCallArguments()
	if p.tok.Type != token.T_RPAREN {
		p.addError(errExpectedRParenFunctionCall, p.tok.Pos.Line, p.tok.Pos.Column, fqcn, p.tok.Literal)
		return nil
	}
	p.nextToken() // consume )
	return &ast.FunctionCallNode{
		Name: &ast.IdentifierNode{
			Value: fqcn,
			Pos:   ast.Position(fqcnPos),
		},
		Args: args,
		Pos:  ast.Position(fqcnPos),
	}
}

func (p *Parser) parseFunctionCallArguments() []ast.Node {
	var args []ast.Node
	for p.tok.Type != token.T_RPAREN && p.tok.Type != token.T_EOF {
		for p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT {
			p.nextToken()
		}
		if p.tok.Type == token.T_RPAREN {
			break
		}
		isUnpacked := false
		if p.tok.Type == token.T_ELLIPSIS {
			isUnpacked = true
			p.nextToken() // consume ...
		}
		var arg ast.Node
		if p.tok.Type == token.T_STRING && p.peekToken().Type == token.T_COLON {
			argPos := p.tok.Pos
			name := p.tok.Literal
			p.nextToken() // consume name
			p.nextToken() // consume :
			value := p.parseExpression()
			arg = &ast.NamedArgumentNode{Name: name, Value: value, Pos: ast.Position(argPos)}
		} else {
			arg = p.parseExpression()
		}
		if arg != nil {
			if isUnpacked {
				arg = &ast.UnpackedArgumentNode{
					Expr: arg,
					Pos:  arg.GetPos(),
				}
			}
			args = append(args, arg)
		}
		for p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT {
			p.nextToken()
		}
		if p.tok.Type == token.T_COMMA {
			p.nextToken()
			continue
		} else if p.tok.Type == token.T_RPAREN {
			break
		} else if p.tok.Type == token.T_EOF {
			break
		} else {
			continue
		}
	}
	return args
}

func (p *Parser) parseSimpleObjectOrMethod(expr ast.Node) ast.Node {
	objOpPos := p.tok.Pos
	operator := p.tok.Literal
	p.nextToken() // consume object operator
	if p.tok.Type == token.T_LBRACE {
		p.nextToken() // consume {
		memberExpr := p.parseExpressionWithStop(token.T_RBRACE)
		if memberExpr == nil {
			p.addError("line %d:%d: expected expression after %s{, got %s", p.tok.Pos.Line, p.tok.Pos.Column, operator, p.tok.Literal)
			return nil
		}
		if p.tok.Type != token.T_RBRACE {
			p.addError("line %d:%d: expected } after dynamic member expression, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
			return nil
		}
		p.nextToken() // consume }
		return &ast.PropertyFetchNode{
			Object:   expr,
			Property: memberExpr.TokenLiteral(),
			Pos:      ast.Position(objOpPos),
		}
	}
	if !isMemberIdentifierToken(p.tok.Type) && !isValidMethodNameToken(p.tok.Type) {
		p.addError("line %d:%d: expected property/method name after %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, operator, p.tok.Literal)
		return nil
	}
	member := p.tok.Literal
	p.nextToken() // consume property/method name
	if p.tok.Type == token.T_LPAREN {
		return p.parseSimpleMethodCall(expr, member, objOpPos)
	}
	return &ast.PropertyFetchNode{
		Object:   expr,
		Property: member,
		Pos:      ast.Position(objOpPos),
	}
}

func (p *Parser) parseSimpleMethodCall(expr ast.Node, member string, objOpPos token.Position) ast.Node {
	p.nextToken() // consume '('
	args := p.parseFunctionCallArguments()
	if p.tok.Type != token.T_RPAREN {
		p.addError("line %d:%d: expected ) after arguments for method call %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, member, p.tok.Literal)
		return nil
	}
	p.nextToken() // consume )
	return &ast.MethodCallNode{
		Object: expr,
		Method: member,
		Args:   args,
		Pos:    ast.Position(objOpPos),
	}
}

func (p *Parser) parseSimpleStringOrConcat() ast.Node {
	pos := p.tok.Pos
	value := p.tok.Literal
	p.nextToken()
	if p.tok.Type == token.T_VARIABLE {
		var parts []ast.Node
		parts = append(parts, &ast.StringNode{
			Value: value,
			Pos:   ast.Position(pos),
		})
		for p.tok.Type == token.T_VARIABLE {
			varNode := &ast.VariableNode{
				Name: p.tok.Literal[1:],
				Pos:  ast.Position(p.tok.Pos),
			}
			parts = append(parts, varNode)
			p.nextToken()
		}
		return &ast.ConcatNode{
			Parts: parts,
			Pos:   ast.Position(pos),
		}
	}
	return &ast.StringNode{
		Value: value,
		Pos:   ast.Position(pos),
	}
}

func (p *Parser) parseSimpleConstantString() ast.Node {
	node := &ast.StringLiteral{
		Value: p.tok.Literal,
		Pos:   ast.Position(p.tok.Pos),
	}
	p.nextToken()
	return node
}

func (p *Parser) parseSimpleLNumber() ast.Node {
	val, _ := strconv.ParseInt(p.tok.Literal, 10, 64)
	node := &ast.IntegerNode{
		Value: val,
		Pos:   ast.Position(p.tok.Pos),
	}
	p.nextToken()
	return node
}

func (p *Parser) parseSimpleDNumber() ast.Node {
	val, _ := strconv.ParseFloat(p.tok.Literal, 64)
	node := &ast.FloatNode{
		Value: val,
		Pos:   ast.Position(p.tok.Pos),
	}
	p.nextToken()
	return node
}

func (p *Parser) parseSimpleBoolOrNull() ast.Node {
	switch p.tok.Type {
	case token.T_TRUE:
		node := &ast.BooleanNode{
			Value: true,
			Pos:   ast.Position(p.tok.Pos),
		}
		p.nextToken()
		return node
	case token.T_FALSE:
		node := &ast.BooleanNode{
			Value: false,
			Pos:   ast.Position(p.tok.Pos),
		}
		p.nextToken()
		return node
	case token.T_NULL:
		node := &ast.NullNode{
			Pos: ast.Position(p.tok.Pos),
		}
		p.nextToken()
		return node
	}
	return nil
}

func (p *Parser) parseSimpleVariable() ast.Node {
	var expr ast.Node = &ast.VariableNode{
		Name: p.tok.Literal[1:],
		Pos:  ast.Position(p.tok.Pos),
	}
	p.nextToken()
	return p.parsePostfixExpression(expr)
}

func (p *Parser) parseSimpleBuiltinCall() ast.Node {
	pos := p.tok.Pos
	name := p.tok.Literal
	p.nextToken()
	if (name == "exit" || name == "die") && p.tok.Type == token.T_SEMICOLON {
		return p.parsePostfixExpression(&ast.FunctionCallNode{
			Name: &ast.IdentifierNode{
				Value: name,
				Pos:   ast.Position(pos),
			},
			Args: nil,
			Pos:  ast.Position(pos),
		})
	}
	if p.tok.Type != token.T_LPAREN {
		p.addError("line %d:%d: expected ( after %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
		return nil
	}
	p.nextToken() // consume '('
	args := p.parseFunctionCallArguments()
	if p.tok.Type != token.T_RPAREN {
		p.addError(errExpectedRParenFunctionCall, p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
		return nil
	}
	p.nextToken() // consume ')'
	return p.parsePostfixExpression(&ast.FunctionCallNode{
		Name: &ast.IdentifierNode{
			Value: name,
			Pos:   ast.Position(pos),
		},
		Args: args,
		Pos:  ast.Position(pos),
	})
}

func (p *Parser) parseSimpleIncludeExpression() ast.Node {
	pos := p.tok.Pos
	name := p.tok.Literal
	p.nextToken()
	expr := p.parseExpressionWithPrecedence(100, false)
	if expr == nil {
		p.addError("line %d:%d: expected expression after %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
		return nil
	}
	return &ast.UnaryExpr{
		Operator: name,
		Operand:  expr,
		Pos:      ast.Position(pos),
	}
}

func (p *Parser) parseSimpleYieldExpression() ast.Node {
	pos := p.tok.Pos
	p.nextToken() // consume yield
	if p.tok.Type == token.T_STRING && p.tok.Literal == "from" {
		p.nextToken() // consume from
		expr := p.parseExpressionWithPrecedence(100, false)
		if expr == nil {
			p.addError("line %d:%d: expected expression after yield from", pos.Line, pos.Column)
			return nil
		}
		return &ast.YieldNode{
			Value: expr,
			From:  true,
			Pos:   ast.Position(pos),
		}
	}

	value := p.parseExpressionWithPrecedence(0, false, token.T_DOUBLE_ARROW, token.T_SEMICOLON)
	if value == nil {
		p.addError("line %d:%d: expected expression after yield", pos.Line, pos.Column)
		return nil
	}

	var key ast.Node
	if p.tok.Type == token.T_DOUBLE_ARROW {
		key = value
		p.nextToken() // consume =>
		value = p.parseExpressionWithPrecedence(0, false, token.T_SEMICOLON)
		if value == nil {
			p.addError("line %d:%d: expected expression after => in yield", p.tok.Pos.Line, p.tok.Pos.Column)
			return nil
		}
	}

	return &ast.YieldNode{
		Key:   key,
		Value: value,
		From:  false,
		Pos:   ast.Position(pos),
	}
}

func (p *Parser) parseGroupedExpression() ast.Node {
	groupPos := p.tok.Pos
	p.nextToken() // consume '('
	if castType, ok := p.readCastType(); ok {
		if p.tok.Type != token.T_RPAREN {
			p.addError("line %d:%d: expected ) after cast type, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
			return nil
		}
		p.nextToken() // consume ')'
		expr := p.parseExpressionWithPrecedence(100, false)
		if expr == nil {
			p.addError("line %d:%d: expected expression after cast %s", p.tok.Pos.Line, p.tok.Pos.Column, castType)
			return nil
		}
		return p.parsePostfixExpression(&ast.TypeCastNode{Type: castType, Expr: expr, Pos: ast.Position(groupPos)})
	}
	expr := p.parseExpression()
	if expr == nil {
		return nil
	}
	if p.tok.Type != token.T_RPAREN {
		p.addError("line %d:%d: expected ) after grouped expression, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil
	}
	p.nextToken() // consume ')'
	if expr.GetPos().Line == 0 {
		expr.SetPos(ast.Position(groupPos))
	}
	return p.parsePostfixExpression(expr)
}

func (p *Parser) readCastType() (string, bool) {
	if p.tok.Type == token.T_STRING {
		switch p.tok.Literal {
		case "string", "int", "integer", "float", "double", "bool", "boolean", "object", "unset":
			castType := p.tok.Literal
			if p.peekToken().Type == token.T_RPAREN {
				p.nextToken()
				return castType, true
			}
		}
	}
	if p.tok.Type == token.T_ARRAY && p.peekToken().Type == token.T_RPAREN {
		p.nextToken()
		return "array", true
	}
	return "", false
}

func (p *Parser) parsePostfixExpression(expr ast.Node) ast.Node {
	for {
		for p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT || p.tok.Type == token.T_WHITESPACE {
			p.nextToken()
		}
		if p.tok.Type == token.T_INC || p.tok.Type == token.T_DEC {
			opTok := p.tok
			p.nextToken()
			expr = &ast.UnaryExpr{
				Operator: opTok.Literal,
				Operand:  expr,
				Pos:      ast.Position(opTok.Pos),
			}
			continue
		}
		if p.tok.Type == token.T_OBJECT_OPERATOR || p.tok.Type == token.T_NULLSAFE_OBJECT_OPERATOR {
			expr = p.parseSimpleObjectOrMethod(expr)
			continue
		}
		if p.tok.Type == token.T_DOUBLE_COLON {
			expr = p.parseStaticAccessOnNode(expr)
			continue
		}
		if p.tok.Type == token.T_LBRACKET {
			expr = p.parseSimpleArrayAccess(expr)
			continue
		}
		if p.tok.Type == token.T_LPAREN {
			expr = p.parseSimpleVariableFunctionCall(expr)
			continue
		}
		break
	}
	return expr
}

func (p *Parser) parseStaticAccessOnNode(expr ast.Node) ast.Node {
	className := expr.TokenLiteral()
	if variable, ok := expr.(*ast.VariableNode); ok {
		className = "$" + variable.Name
	}
	p.nextToken() // consume '::'
	if p.tok.Type == token.T_VARIABLE {
		memberName := p.tok.Literal
		p.nextToken()
		return p.parsePostfixExpression(&ast.ClassConstFetchNode{Class: className, Const: memberName, Pos: expr.GetPos()})
	}
	if isMemberIdentifierToken(p.tok.Type) || isValidMethodNameToken(p.tok.Type) {
		memberName := p.tok.Literal
		p.nextToken()
		if p.tok.Type == token.T_LPAREN {
			p.nextToken() // consume '('
			args := p.parseFunctionCallArguments()
			if p.tok.Type != token.T_RPAREN {
				p.addError("line %d:%d: expected ) after arguments for static call %s::%s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, className, memberName, p.tok.Literal)
				return nil
			}
			p.nextToken() // consume ')'
			return p.parsePostfixExpression(&ast.FunctionCallNode{
				Name: &ast.IdentifierNode{Value: className + "::" + memberName, Pos: expr.GetPos()},
				Args: args,
				Pos:  expr.GetPos(),
			})
		}
		return p.parsePostfixExpression(&ast.ClassConstFetchNode{Class: className, Const: memberName, Pos: expr.GetPos()})
	}
	if p.tok.Type == token.T_CLASS_CONST || p.tok.Type == token.T_CLASS {
		p.nextToken()
		return p.parsePostfixExpression(&ast.ClassConstFetchNode{Class: className, Const: "class", Pos: expr.GetPos()})
	}
	p.addError("line %d:%d: expected constant name or 'class' after '::', got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
	return nil
}

func isMemberIdentifierToken(tokType token.TokenType) bool {
	switch tokType {
	case token.T_STRING, token.T_CLASS, token.T_CALLABLE, token.T_MATCH, token.T_FN, token.T_LIST, token.T_STATIC, token.T_SELF, token.T_PARENT, token.T_VARIABLE:
		return true
	default:
		return false
	}
}

func (p *Parser) parseSimpleArrayAccess(expr ast.Node) ast.Node {
	bracketPos := p.tok.Pos
	p.nextToken() // consume [
	var index ast.Node
	if p.tok.Type != token.T_RBRACKET {
		index = p.parseExpression()
	}
	if p.tok.Type != token.T_RBRACKET {
		p.addError("line %d:%d: expected ] after array index, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil
	}
	p.nextToken() // consume ]
	return &ast.ArrayAccessNode{
		Var:   expr,
		Index: index,
		Pos:   ast.Position(bracketPos),
	}
}

func (p *Parser) parseSimpleHeredoc() ast.Node {
	pos := p.tok.Pos
	identifier := p.tok.Literal
	p.nextToken() // consume heredoc start token

	parts := []ast.Node{}
	if p.tok.Type == token.T_ENCAPSED_AND_WHITESPACE {
		parts = append(parts, &ast.StringNode{
			Value: p.tok.Literal,
			Pos:   ast.Position(p.tok.Pos),
		})
		p.nextToken()
	}

	if p.tok.Type != token.T_END_HEREDOC && p.tok.Type != token.T_END_NOWDOC {
		p.addError("line %d:%d: expected heredoc terminator for %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, identifier, p.tok.Literal)
		return nil
	}
	p.nextToken() // consume heredoc terminator

	return &ast.HeredocNode{
		Identifier: identifier,
		Parts:      parts,
		Pos:        ast.Position(pos),
	}
}

func (p *Parser) parseSimpleVariableFunctionCall(expr ast.Node) ast.Node {
	p.nextToken() // consume '('
	var args []ast.Node
	for p.tok.Type != token.T_RPAREN && p.tok.Type != token.T_EOF {
		isUnpacked := false
		if p.tok.Type == token.T_ELLIPSIS {
			isUnpacked = true
			p.nextToken() // consume ...
		}
		arg := p.parseExpression()
		if arg != nil {
			if isUnpacked {
				arg = &ast.UnpackedArgumentNode{
					Expr: arg,
					Pos:  arg.GetPos(),
				}
			}
			args = append(args, arg)
		}
		if p.tok.Type == token.T_COMMA {
			p.nextToken()
			continue
		} else if p.tok.Type == token.T_RPAREN {
			break
		} else if p.tok.Type == token.T_EOF {
			break
		} else {
			continue
		}
	}
	if p.tok.Type != token.T_RPAREN {
		p.addError(errExpectedRParenFunctionCall, p.tok.Pos.Line, p.tok.Pos.Column, expr.TokenLiteral(), p.tok.Literal)
		return nil
	}
	p.nextToken() // consume )
	return &ast.FunctionCallNode{
		Name: expr,
		Args: args,
		Pos:  expr.GetPos(),
	}
}

func (p *Parser) parseSimpleUnexpected() ast.Node {
	p.addError("line %d:%d: unexpected token %s in expression", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
	p.nextToken()
	return nil
}
