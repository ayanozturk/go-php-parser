package parser

import (
	"go-phpcs/ast"
	"go-phpcs/token"
	"strconv"
)

const errExpectedRParenFunctionCall = "line %d:%d: expected ) after arguments for function call %s, got %s"

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

func (p *Parser) parseSimpleExpression() ast.Node {
	// Handle unary minus and plus
	if p.tok.Type == token.T_MINUS || p.tok.Type == token.T_PLUS {
		return p.parseSimpleUnary()
	}

	switch p.tok.Type {
	case token.T_NEW:
		return p.parseSimpleNew()
	case token.T_STRING, token.T_STATIC, token.T_SELF, token.T_PARENT, token.T_NS_SEPARATOR:
		return p.parseSimpleFQCNOrFunctionCall()
	case token.T_CONSTANT_ENCAPSED_STRING:
		return p.parseSimpleStringOrConcat()
	case token.T_CONSTANT_STRING:
		return p.parseSimpleConstantString()
	case token.T_LNUMBER:
		return p.parseSimpleLNumber()
	case token.T_DNUMBER:
		return p.parseSimpleDNumber()
	case token.T_TRUE, token.T_FALSE, token.T_NULL:
		return p.parseSimpleBoolOrNull()
	case token.T_VARIABLE:
		return p.parseSimpleVariable()
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
	if p.tok.Type != token.T_STRING && p.tok.Type != token.T_STATIC && p.tok.Type != token.T_SELF && p.tok.Type != token.T_PARENT && p.tok.Type != token.T_NS_SEPARATOR {
		p.addError("line %d:%d: expected class name after new, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil
	}
	classNameNode := p.parseFQCN()
	className := ""
	if id, ok := classNameNode.(*ast.IdentifierNode); ok {
		className = id.Value
	} else {
		p.addError("line %d:%d: expected identifier node for class name after new", p.tok.Pos.Line, p.tok.Pos.Column)
		return nil
	}
	if p.tok.Type != token.T_LPAREN {
		p.addError("line %d:%d: expected ( after class name %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, className, p.tok.Literal)
		return nil
	}
	p.nextToken() // consume (
	var args []ast.Node
	for p.tok.Type != token.T_RPAREN && p.tok.Type != token.T_EOF {
		if arg := p.parseExpression(); arg != nil {
			args = append(args, arg)
		}
		if p.tok.Type == token.T_COMMA {
			p.nextToken()
			continue
		}
		break
	}
	if p.tok.Type != token.T_RPAREN {
		p.addError("line %d:%d: expected ) after arguments for %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, className, p.tok.Literal)
		return nil
	}
	p.nextToken() // consume )
	return &ast.NewNode{
		ClassName: className,
		Args:      args,
		Pos:       ast.Position(pos),
	}
}

func (p *Parser) parseSimpleFQCNOrFunctionCall() ast.Node {
	fqcnPos := p.tok.Pos
	fqcn := ""
	// Handle leading \ for fully qualified names
	if p.tok.Type == token.T_NS_SEPARATOR {
		fqcn += "\\"
		p.nextToken()
	}
	fqcn += p.collectFQCNString()
	if p.tok.Type == token.T_DOUBLE_COLON {
		return p.parseSimpleClassConstFetch(fqcn, fqcnPos)
	}
	if p.tok.Type == token.T_LPAREN {
		return p.parseSimpleFunctionCall(fqcn, fqcnPos)
	}
	return &ast.IdentifierNode{
		Value: fqcn,
		Pos:   ast.Position(fqcnPos),
	}
}

func (p *Parser) collectFQCNString() string {
	fqcn := ""
	for {
		if p.tok.Type == token.T_NS_SEPARATOR {
			fqcn += "\\"
			p.nextToken()
		} else if p.tok.Type == token.T_STRING || p.tok.Type == token.T_STATIC || p.tok.Type == token.T_SELF || p.tok.Type == token.T_PARENT {
			fqcn += p.tok.Literal
			p.nextToken()
		} else {
			break
		}
	}
	return fqcn
}

func (p *Parser) parseSimpleClassConstFetch(fqcn string, fqcnPos token.Position) ast.Node {
	p.nextToken() // consume '::'
	if p.tok.Type == token.T_STRING {
		constName := p.tok.Literal
		p.nextToken()
		return &ast.ClassConstFetchNode{
			Class: fqcn,
			Const: constName,
			Pos:   ast.Position(fqcnPos),
		}
	} else if p.tok.Type == token.T_CLASS_CONST {
		p.nextToken()
		return &ast.ClassConstFetchNode{
			Class: fqcn,
			Const: "class",
			Pos:   ast.Position(fqcnPos),
		}
	} else {
		p.addError("line %d:%d: expected constant name or 'class' after '::', got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil
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

func (p *Parser) parseFunctionCallArguments() []ast.Node {
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
	return args
}

func (p *Parser) parseSimpleObjectOrMethod(expr ast.Node) ast.Node {
	objOpPos := p.tok.Pos
	p.nextToken() // consume '->'
	if p.tok.Type != token.T_STRING {
		p.addError("line %d:%d: expected property/method name after ->, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
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
	for {
		if p.tok.Type == token.T_OBJECT_OPERATOR {
			expr = p.parseSimpleObjectOrMethod(expr)
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

func (p *Parser) parseSimpleArrayAccess(expr ast.Node) ast.Node {
	bracketPos := p.tok.Pos
	p.nextToken() // consume [
	index := p.parseExpression()
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
