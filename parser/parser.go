package parser

import (
	"fmt"
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"go-phpcs/token"
	"strconv"
)

type Parser struct {
	l      *lexer.Lexer
	tok    token.Token
	errors []string
	debug  bool
}

func New(l *lexer.Lexer, debug bool) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
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
		errMsg := fmt.Sprintf(format, args...)
		p.errors = append(p.errors, errMsg)
	}
}

// Errors returns the list of errors encountered during parsing
func (p *Parser) Errors() []string {
	return p.errors
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

	// Skip whitespace/comments/doc comments after open tag
	for p.tok.Type == token.T_WHITESPACE || p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT {
		p.nextToken()
	}

	for p.tok.Type != token.T_EOF {
		// Also skip whitespace/comments/doc comments between statements
		for p.tok.Type == token.T_WHITESPACE || p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT {
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

func (p *Parser) parseStatement() (ast.Node, error) {
	// Skip attributes before statement (PHP 8+)
	for p.tok.Type == token.T_ATTRIBUTE {
		p.nextToken()
	}
	if p.tok.Type == token.T_NAMESPACE {
		return p.parseNamespaceDeclaration()
	}
	if p.tok.Type == token.T_LBRACE {
		pos := p.tok.Pos
		p.nextToken() // consume {
		stmts := p.parseBlockStatement()
		if p.tok.Type != token.T_RBRACE {
			p.addError("line %d:%d: expected } to close block, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
			return nil, nil
		}
		p.nextToken() // consume }
		return &ast.BlockNode{Statements: stmts, Pos: ast.Position(pos)}, nil
	}
	// Handle declare statement
	if p.tok.Type == token.T_DECLARE {
		p.nextToken() // consume 'declare'
		return p.parseDeclare(), nil
	}
	switch p.tok.Type {
	case token.T_TRAIT:
		return p.parseTraitDeclaration()
	case token.T_COMMENT:
		pos := p.tok.Pos
		comment := p.tok.Literal
		p.nextToken() // consume comment
		return &ast.CommentNode{
			Value: comment,
			Pos:   ast.Position(pos),
		}, nil
	case token.T_RETURN:
		pos := p.tok.Pos
		p.nextToken() // consume return
		expr := p.parseExpression()
		if expr == nil {
			return nil, nil
		}
		if p.tok.Type != token.T_SEMICOLON {
			p.addError("line %d:%d: expected ; after return statement, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
			return nil, nil
		}
		p.nextToken() // consume ;
		return &ast.ReturnNode{
			Expr: expr,
			Pos:  ast.Position(pos),
		}, nil
	case token.T_FUNCTION:
		return p.parseFunction(nil)

	case token.T_IF:
		return p.parseIfStatement()
	case token.T_STRING:
		if p.tok.Literal == "final" || p.tok.Literal == "abstract" {
			modifier := p.tok.Literal
			p.nextToken()
			if p.tok.Type == token.T_CLASS {
				return p.parseClassDeclarationWithModifier(modifier)
			}
			p.addError("line %d:%d: expected 'class' after modifier %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, modifier, p.tok.Literal)
			return nil, nil
		}
		return p.parseExpressionStatement()
	case token.T_CLASS:
		return p.parseClassDeclaration()
	case token.T_INTERFACE:
		node := p.parseInterfaceDeclaration()
		if node == nil {
			return nil, fmt.Errorf("failed to parse interface declaration")
		}
		return node, nil
	case token.T_ECHO:
		pos := p.tok.Pos
		p.nextToken() // consume echo
		expr := p.parseExpression()
		if expr == nil {
			return nil, nil
		}
		if p.tok.Type != token.T_SEMICOLON {
			p.addError("line %d:%d: expected ; after echo statement, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
			return nil, nil
		}
		p.nextToken() // consume ;
		return &ast.ExpressionStmt{
			Expr: expr,
			Pos:  ast.Position(pos),
		}, nil

	case token.T_SEMICOLON:
		p.nextToken() // skip empty statements
		return nil, nil
	case token.T_ENUM:
		return p.parseEnum()
	case token.T_FOREACH:
		return p.parseForeachStatement()
	default:
		// Try parsing as expression statement
		if expr := p.parseExpression(); expr != nil {
			// Accept function calls as statements even if last token is ')', as long as next is semicolon
			if p.tok.Type != token.T_SEMICOLON {
				p.addError("line %d:%d: expected ; after expression, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
				return nil, nil
			}
			p.nextToken() // consume ;
			return &ast.ExpressionStmt{
				Expr: expr,
				Pos:  expr.GetPos(),
			}, nil
		}
		// Enhanced error recovery: skip tokens until semicolon or closing paren to avoid cascading errors
		p.addError("line %d:%d: unexpected token %s in statement (error recovery)", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		for p.tok.Type != token.T_SEMICOLON && p.tok.Type != token.T_RPAREN && p.tok.Type != token.T_EOF {
			p.nextToken()
		}
		if p.tok.Type == token.T_SEMICOLON {
			p.nextToken()
		}
		return nil, nil
	}
}

func (p *Parser) parseVariableStatement() (ast.Node, error) {
	varPos := p.tok.Pos
	varName := p.tok.Literal[1:] // Remove leading $ from variable name
	p.nextToken()

	// If this is an assignment
	if p.tok.Type == token.T_ASSIGN {
		p.nextToken() // consume =
		right := p.parseExpression()
		if right == nil {
			return nil, fmt.Errorf("failed to parse right-hand side of assignment")
		}

		if p.tok.Type != token.T_SEMICOLON {
			p.addError("line %d:%d: expected ; after assignment to $%s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, varName, p.tok.Literal)
			return nil, nil
		}
		p.nextToken() // consume ;

		return &ast.AssignmentNode{
			Left: &ast.VariableNode{
				Name: varName,
				Pos:  ast.Position(varPos),
			},
			Right: right,
			Pos:   ast.Position(varPos),
		}, nil
	}

	return &ast.VariableNode{
		Name: varName,
		Pos:  ast.Position(varPos),
	}, nil
}

// Use PhpOperatorPrecedence and PhpOperatorRightAssoc from domain.go

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

// Remove isAssignmentOperator and isValidAssignmentTarget (now in domain.go)

func (p *Parser) isBinaryOperator(tokenType token.TokenType) bool {
	switch tokenType {
	case token.T_PLUS, token.T_MINUS, token.T_MULTIPLY, token.T_DIVIDE, token.T_MODULO,
		token.T_IS_EQUAL, token.T_IS_NOT_EQUAL, token.T_IS_SMALLER, token.T_IS_GREATER,
		token.T_DOT,         // Support string concatenation
		token.T_COALESCE,    // Support null coalescing operator ??
		token.T_BOOLEAN_OR,  // Support double pipe || operator
		token.T_BOOLEAN_AND, // Support double ampersand && operator
		token.T_PIPE:        // Support single pipe | operator
		return true
	case token.T_ASSIGN:
		return true
	case token.T_IS_GREATER_OR_EQUAL, token.T_IS_SMALLER_OR_EQUAL:
		return true
	default:
		return false
	}
}

func (p *Parser) parseExpressionStatement() (ast.Node, error) {
	expr := p.parseExpressionWithPrecedence(0, true)
	if expr == nil {
		return nil, nil
	}

	if p.tok.Type != token.T_SEMICOLON {
		p.addError("line %d:%d: expected ; after expression, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume ;

	return &ast.ExpressionStmt{
		Expr: expr,
		Pos:  expr.GetPos(),
	}, nil
}

func (p *Parser) parseBlockStatement() []ast.Node {
	var statements []ast.Node
	for p.tok.Type != token.T_RBRACE && p.tok.Type != token.T_EOF {
		stmt, err := p.parseStatement()
		if err != nil {
			p.addError(err.Error())
			p.nextToken()
			continue
		} else if stmt != nil {
			statements = append(statements, stmt)
		}
		if stmt == nil {
			p.nextToken()
		}
	}
	return statements
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
		if p.tok.Type == token.T_STRING {
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
	if p.debug {

	}
	return &ast.IdentifierNode{
		Value: fqcn,
		Pos:   ast.Position(pos),
	}
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

		if p.tok.Type != token.T_STRING {
			p.addError("line %d:%d: expected class name after new, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
			return nil
		}
		className := p.tok.Literal
		p.nextToken()

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
		varNode := &ast.VariableNode{
			Name: p.tok.Literal[1:], // Remove $ prefix
			Pos:  ast.Position(p.tok.Pos),
		}
		p.nextToken()
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
			return &ast.FunctionCallNode{
				Name: varNode,
				Args: args,
				Pos:  varNode.Pos,
			}
		}
		// Support array access: $foo[0], $foo[$bar], etc.
		var expr ast.Node = varNode
		for {
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
				return expr
			}
			// Support property fetch: $foo->bar
			if p.tok.Type == token.T_OBJECT_OPERATOR {
				objOpPos := p.tok.Pos
				p.nextToken() // consume '->'
				if p.tok.Type != token.T_STRING {
					p.addError("line %d:%d: expected property name after ->, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
					return nil
				}
				property := p.tok.Literal
				p.nextToken() // consume property name
				expr = &ast.PropertyFetchNode{
					Object:   expr,
					Property: property,
					Pos:      ast.Position(objOpPos),
				}
				continue // allow chaining: $foo->bar->baz
			}
			break
		}
		return expr
	case token.T_NS_SEPARATOR:
		if p.debug {

		}
		return p.parseFQCN()
	default:
		p.addError("line %d:%d: unexpected token %s in expression", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		p.nextToken()
		return nil
	}
}

// parseTraitDeclaration parses a PHP trait declaration
func (p *Parser) parseTraitDeclaration() (ast.Node, error) {
	pos := p.tok.Pos
	p.nextToken() // consume 'trait'

	if p.tok.Type != token.T_STRING {
		p.addError("line %d:%d: expected trait name, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	name := p.tok.Literal
	p.nextToken()

	// Expect opening brace
	if p.tok.Type != token.T_LBRACE {
		p.addError("line %d:%d: expected { to start trait body for %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume {

	// Parse methods inside the trait
	var methods []ast.Node
	for p.tok.Type != token.T_RBRACE && p.tok.Type != token.T_EOF {
		// Collect all modifiers (public, protected, private, static, final, abstract) and comments/docblocks before 'function'
		var modifiers []string
		for {
			switch p.tok.Type {
			case token.T_PUBLIC, token.T_PROTECTED, token.T_PRIVATE, token.T_STATIC, token.T_FINAL, token.T_ABSTRACT:
				modifiers = append(modifiers, p.tok.Literal)
				p.nextToken()
				continue
			case token.T_COMMENT, token.T_DOC_COMMENT:
				p.nextToken()
				continue
			}
			break
		}
		if p.tok.Type == token.T_FUNCTION {
			fn, err := p.parseFunction(modifiers)
			if err != nil {
				p.addError(err.Error())
				p.nextToken()
				continue
			}
			if fn != nil {
				methods = append(methods, fn)
			}
			continue
		}
		if len(modifiers) > 0 {
			// If we saw modifiers but not a function, emit error and skip
			p.addError("line %d:%d: expected function after modifiers in trait %s body, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
			p.nextToken()
			continue
		}
		// Skip unexpected tokens inside trait body
		p.addError("line %d:%d: unexpected token %s in trait %s body", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal, name)
		p.nextToken()
	}

	if p.tok.Type != token.T_RBRACE {
		p.addError("line %d:%d: expected } to close trait %s body, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume }

	return &ast.TraitNode{
		Name:    name,
		Methods: methods,
		Pos:     ast.Position(pos),
	}, nil
}
