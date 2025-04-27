package parser

import (
	"go-phpcs/ast"
	"go-phpcs/token"
	"strconv"
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

func (p *Parser) parseSimpleExpression() ast.Node {
	// Handle unary minus and plus
	if p.tok.Type == token.T_MINUS || p.tok.Type == token.T_PLUS {
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
			// Fallback: treat as BinaryExpr or error
			return right
		}
	}
	switch p.tok.Type {
	case token.T_NEW:
		pos := p.tok.Pos
		p.nextToken() // consume 'new'

		// Accept T_STRING, T_STATIC, T_SELF, T_PARENT, or T_NS_SEPARATOR for FQCN
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
	case token.T_STRING, token.T_STATIC, token.T_SELF, token.T_PARENT:
		// Support fully qualified class names (FQCN) like \Symfony\Component\ClassName
		fqcnPos := p.tok.Pos
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
		// Class constant fetch: FQCN::CONST or FQCN::class
		if p.tok.Type == token.T_DOUBLE_COLON {
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
				// Support FQCN::class
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
		// Check for function call: FQCN(...)
		if p.tok.Type == token.T_LPAREN {
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
				p.addError("line %d:%d: expected ) after arguments for function call %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, fqcn, p.tok.Literal)
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
		// Not a function call, just an identifier
		return &ast.IdentifierNode{
			Value: fqcn,
			Pos:   ast.Position(fqcnPos),
		}
	case token.T_CONSTANT_ENCAPSED_STRING:
		// Handle string interpolation
		pos := p.tok.Pos
		value := p.tok.Literal
		p.nextToken()

		// Check for variable interpolation
		if p.tok.Type == token.T_VARIABLE {
			var parts []ast.Node
			parts = append(parts, &ast.StringNode{
				Value: value,
				Pos:   ast.Position(pos),
			})
			for p.tok.Type == token.T_VARIABLE {
				varNode := &ast.VariableNode{
					Name: p.tok.Literal[1:], // Remove $ prefix
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
	case token.T_CONSTANT_STRING:
		node := &ast.StringLiteral{
			Value: p.tok.Literal,
			Pos:   ast.Position(p.tok.Pos),
		}
		p.nextToken()
		return node
	case token.T_LNUMBER:
		val, _ := strconv.ParseInt(p.tok.Literal, 10, 64)
		node := &ast.IntegerNode{
			Value: val,
			Pos:   ast.Position(p.tok.Pos),
		}
		p.nextToken()
		return node
	case token.T_DNUMBER:
		val, _ := strconv.ParseFloat(p.tok.Literal, 64)
		node := &ast.FloatNode{
			Value: val,
			Pos:   ast.Position(p.tok.Pos),
		}
		p.nextToken()
		return node
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
	case token.T_VARIABLE:
		var expr ast.Node = &ast.VariableNode{
			Name: p.tok.Literal[1:], // Remove $ prefix
			Pos:  ast.Position(p.tok.Pos),
		}
		p.nextToken()
		for {
			if p.tok.Type == token.T_OBJECT_OPERATOR {
				objOpPos := p.tok.Pos
				p.nextToken() // consume '->'
				if p.tok.Type != token.T_STRING {
					p.addError("line %d:%d: expected property/method name after ->, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
					return nil
				}
				member := p.tok.Literal
				p.nextToken() // consume property/method name
				// Check for method call
				if p.tok.Type == token.T_LPAREN {
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
						p.addError("line %d:%d: expected ) after arguments for method call %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, member, p.tok.Literal)
						return nil
					}
					p.nextToken() // consume )
					expr = &ast.MethodCallNode{
						Object: expr,
						Method: member,
						Args:   args,
						Pos:    ast.Position(objOpPos),
					}
				} else {
					expr = &ast.PropertyFetchNode{
						Object:   expr,
						Property: member,
						Pos:      ast.Position(objOpPos),
					}
				}
				continue // allow chaining: $foo->bar()->baz
			}
			if p.tok.Type == token.T_LBRACKET {
				bracketPos := p.tok.Pos
				p.nextToken() // consume [
				index := p.parseExpression()
				if p.tok.Type != token.T_RBRACKET {
					p.addError("line %d:%d: expected ] after array index, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
					return nil
				}
				p.nextToken() // consume ]
				expr = &ast.ArrayAccessNode{
					Var:   expr,
					Index: index,
					Pos:   ast.Position(bracketPos),
				}
				continue
			}
			// Check for function call: $var()
			if p.tok.Type == token.T_LPAREN {
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
					p.addError("line %d:%d: expected ) after arguments for function call on variable, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
					return nil
				}
				p.nextToken() // consume )
				expr = &ast.FunctionCallNode{
					Name: expr,
					Args: args,
					Pos:  expr.GetPos(),
				}
				continue
			}
			break
		}
		return expr
	case token.T_NS_SEPARATOR:
		fqcnNode := p.parseFQCN()
		// Check for function call: \FQCN(...)
		if p.tok.Type == token.T_LPAREN {
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
				p.addError("line %d:%d: expected ) after arguments for function call %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, fqcnNode.TokenLiteral(), p.tok.Literal)
				return nil
			}
			p.nextToken() // consume )
			return &ast.FunctionCallNode{
				Name: fqcnNode,
				Args: args,
				Pos:  fqcnNode.GetPos(),
			}
		}
		return fqcnNode
	default:
		p.addError("line %d:%d: unexpected token %s in expression", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		p.nextToken()
		return nil
	}
}
