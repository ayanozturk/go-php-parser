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
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
	}
	p.nextToken() // Initialize first token
	return p
}

func (p *Parser) nextToken() {
	p.tok = p.l.NextToken()
}

func (p *Parser) Parse() []ast.Node {
	var nodes []ast.Node

	// Expect PHP open tag first
	if p.tok.Type != token.T_OPEN_TAG {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected <?php at start of file, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
		return nodes
	}
	p.nextToken()

	for p.tok.Type != token.T_EOF {
		node := p.parseStatement()
		if node != nil {
			nodes = append(nodes, node)
		}
		// Don't consume token here - parseStatement handles that
	}

	return nodes
}

func (p *Parser) parseStatement() ast.Node {
	var node ast.Node

	switch p.tok.Type {
	case token.T_COMMENT, token.T_DOC_COMMENT:
		pos := p.tok.Pos
		comment := p.tok.Literal
		p.nextToken() // Skip comments
		return &ast.CommentNode{
			Value: comment,
			Pos:   ast.Position(pos),
		}
	case token.T_RETURN:
		pos := p.tok.Pos
		p.nextToken() // consume return
		expr := p.parseExpression()
		if expr == nil {
			return nil
		}
		if p.tok.Type != token.T_SEMICOLON {
			p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected ; after return statement, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
			return nil
		}
		p.nextToken() // consume ;
		return &ast.ReturnNode{
			Expr: expr,
			Pos:  ast.Position(pos),
		}
	case token.T_FUNCTION:
		node = p.parseFunction()
		return node
	case token.T_VARIABLE:
		if p.tok.Type == token.T_OBJECT_OP {
			p.nextToken() // consume ->
			if p.tok.Type != token.T_STRING {
				p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected method name after ->, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
				return nil
			}
			methodName := p.tok.Literal
			p.nextToken()

			if p.tok.Type != token.T_LPAREN {
				p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected ( after method name %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, methodName, p.tok.Literal))
				return nil
			}
			p.nextToken() // consume (

			var args []ast.Node
			if p.tok.Type != token.T_RPAREN {
				for {
					arg := p.parseExpression()
					if arg == nil {
						return nil
					}
					args = append(args, arg)

					if p.tok.Type == token.T_COMMA {
						p.nextToken()
						continue
					}
					break
				}
			}

			if p.tok.Type != token.T_RPAREN {
				p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected ) in argument list for method %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, methodName, p.tok.Literal))
				return nil
			}
			p.nextToken() // consume )

			if p.tok.Type != token.T_SEMICOLON {
				p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected ; after method call %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, methodName, p.tok.Literal))
				return nil
			}
			p.nextToken() // consume ;

			return &ast.ExpressionStmt{
				Expr: &ast.MethodCallNode{
					Object: &ast.VariableNode{
						Name: p.tok.Literal[1:],
						Pos:  ast.Position(p.tok.Pos),
					},
					Method: methodName,
					Args:   args,
					Pos:    ast.Position(p.tok.Pos),
				},
				Pos: ast.Position(p.tok.Pos),
			}
		}
		// If not a method call, handle as regular variable statement
		node = p.parseVariableStatement()
		return node
	case token.T_IF:
		node = p.parseIfStatement()
		return node
	case token.T_CLASS:
		node = p.parseClassDeclaration()
		return node
	case token.T_INTERFACE:
		node = p.parseInterfaceDeclaration()
		return node
	case token.T_ECHO:
		pos := p.tok.Pos
		p.nextToken() // consume echo
		expr := p.parseExpression()
		if expr == nil {
			return nil
		}
		if p.tok.Type != token.T_SEMICOLON {
			p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected ; after echo statement, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
			return nil
		}
		p.nextToken() // consume ;
		return &ast.ExpressionStmt{
			Expr: expr,
			Pos:  ast.Position(pos),
		}
	case token.T_STRING:
		// Handle other expressions
		return p.parseExpressionStatement()
	case token.T_SEMICOLON:
		p.nextToken() // skip empty statements
		return nil
	default:
		// Try parsing as expression statement
		if expr := p.parseExpression(); expr != nil {
			if p.tok.Type != token.T_SEMICOLON {
				p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected ; after expression, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
				return nil
			}
			p.nextToken() // consume ;
			return &ast.ExpressionStmt{
				Expr: expr,
				Pos:  expr.GetPos(),
			}
		}
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: unexpected token %s in statement", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
		p.nextToken()
		return nil
	}
}

func (p *Parser) parseFunction() ast.Node {
	pos := p.tok.Pos
	p.nextToken() // consume 'function'

	// Parse function name if present (could be anonymous function)
	var name string
	if p.tok.Type == token.T_STRING {
		name = p.tok.Literal
		p.nextToken()
	}

	if p.tok.Type != token.T_LPAREN {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected ( after function name %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal))
		return nil
	}
	p.nextToken() // consume (

	// Parse parameters
	var params []ast.Node
	for p.tok.Type != token.T_RPAREN && p.tok.Type != token.T_EOF {
		if param := p.parseParameter(); param != nil {
			params = append(params, param)
		}

		if p.tok.Type == token.T_COMMA {
			p.nextToken()
			continue
		}

		if p.tok.Type != token.T_RPAREN {
			p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected , or ) in parameter list for function %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal))
			return nil
		}
	}
	p.nextToken() // consume )

	// Parse return type
	var returnType string
	if p.tok.Type == token.T_COLON {
		p.nextToken()
		if p.tok.Type == token.T_STRING || p.tok.Type == token.T_ARRAY {
			returnType = p.tok.Literal
			p.nextToken()

			// Handle array type
			if p.tok.Type == token.T_LBRACKET {
				returnType += "["
				p.nextToken()

				// Handle array items
				for p.tok.Type != token.T_RBRACKET && p.tok.Type != token.T_EOF {
					if p.tok.Type == token.T_STRING || p.tok.Type == token.T_ARRAY {
						returnType += p.tok.Literal
						p.nextToken()
					} else {
						p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected array item type, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
						return nil
					}

					if p.tok.Type == token.T_COMMA {
						returnType += ", "
						p.nextToken()
						continue
					}
					break
				}

				if p.tok.Type != token.T_RBRACKET {
					p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected ']' after array type in return type for function %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal))
					return nil
				}
				returnType += "]"
				p.nextToken()
			}
		} else {
			p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected return type for function %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal))
			return nil
		}
	}

	// Parse function body
	if p.tok.Type != token.T_LBRACE {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected { to start function body for %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal))
		return nil
	}
	body := p.parseBlockStatement()

	return &ast.FunctionNode{
		Name:       name,
		Params:     params,
		Body:       body,
		ReturnType: returnType,
		Pos:        ast.Position(pos),
	}
}

func (p *Parser) parseVariableStatement() ast.Node {
	varPos := p.tok.Pos
	varName := p.tok.Literal[1:] // Remove leading $ from variable name
	p.nextToken()

	// If this is an assignment
	if p.tok.Type == token.T_ASSIGN {
		p.nextToken() // consume =
		right := p.parseExpression()
		if right == nil {
			return nil
		}

		if p.tok.Type != token.T_SEMICOLON {
			p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected ; after assignment to $%s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, varName, p.tok.Literal))
			return nil
		}

		return &ast.AssignmentNode{
			Left: &ast.VariableNode{
				Name: varName,
				Pos:  ast.Position(varPos),
			},
			Right: right,
			Pos:   ast.Position(varPos),
		}
	}

	return &ast.VariableNode{
		Name: varName,
		Pos:  ast.Position(varPos),
	}
}

func (p *Parser) parseExpression() ast.Node {
	switch p.tok.Type {
	case token.T_LBRACKET:
		return p.parseArrayLiteral()
	case token.T_ARRAY:
		if p.peekToken().Type == token.T_LPAREN {
			return p.parseArrayLiteral()
		}
		fallthrough
	case token.T_STRING:
		// Handle array keys
		if p.peekToken().Type == token.T_DOUBLE_ARROW {
			return p.parseArrayLiteral()
		}
		fallthrough
	default:
		return p.parseSimpleExpression()
	}
}

func (p *Parser) parseArrayElement() ast.Node {
	var key ast.Node
	var value ast.Node

	// Parse key if present
	if p.tok.Type == token.T_STRING || p.tok.Type == token.T_CONSTANT_ENCAPSED_STRING {
		key = &ast.StringNode{
			Value: p.tok.Literal,
			Pos:   ast.Position(p.tok.Pos),
		}
		p.nextToken()

		if p.tok.Type == token.T_DOUBLE_ARROW {
			p.nextToken() // consume =>
		} else {
			// If no =>, treat the string as a value
			return &ast.KeyValueNode{
				Key:   nil,
				Value: key,
				Pos:   key.GetPos(),
			}
		}
	}

	// Parse value
	value = p.parseExpression()
	if value == nil {
		return nil
	}

	// Don't consume the token here - let the caller handle commas
	return &ast.KeyValueNode{
		Key:   key,
		Value: value,
		Pos:   value.GetPos(),
	}
}

func (p *Parser) parseArrayLiteral() ast.Node {
	pos := p.tok.Pos

	// Handle array() syntax
	if p.tok.Type == token.T_ARRAY {
		p.nextToken() // consume array
		if p.tok.Type != token.T_LPAREN {
			p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected ( after array, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
			return nil
		}
		p.nextToken() // consume (

		var elements []ast.Node
		for p.tok.Type != token.T_RPAREN && p.tok.Type != token.T_EOF {
			if element := p.parseArrayElement(); element != nil {
				elements = append(elements, element)
			}

			if p.tok.Type == token.T_COMMA {
				p.nextToken() // consume comma
				continue
			}

			if p.tok.Type != token.T_RPAREN {
				p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected , or ) in array literal, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
				return nil
			}
			break
		}
		p.nextToken() // consume )

		return &ast.ArrayNode{
			Elements: elements,
			Pos:      ast.Position(pos),
		}
	}

	// Handle [] syntax
	if p.tok.Type == token.T_LBRACKET {
		p.nextToken() // consume [

		var elements []ast.Node
		for p.tok.Type != token.T_RBRACKET && p.tok.Type != token.T_EOF {
			// Parse key if present
			var key ast.Node
			var value ast.Node

			// Parse key if it's a string
			if p.tok.Type == token.T_STRING || p.tok.Type == token.T_CONSTANT_STRING || p.tok.Type == token.T_CONSTANT_ENCAPSED_STRING {
				key = &ast.StringNode{
					Value: p.tok.Literal,
					Pos:   ast.Position(p.tok.Pos),
				}
				p.nextToken()

				if p.tok.Type == token.T_DOUBLE_ARROW {
					p.nextToken() // consume =>
					value = p.parseExpression()
					if value == nil {
						return nil
					}
				} else {
					// If no =>, treat the string as a value
					value = key
					key = nil
				}
			} else {
				// No key, just parse value
				value = p.parseExpression()
				if value == nil {
					return nil
				}
			}

			elements = append(elements, &ast.KeyValueNode{
				Key:   key,
				Value: value,
				Pos:   value.GetPos(),
			})

			if p.tok.Type == token.T_COMMA {
				p.nextToken() // consume comma
				continue
			}

			if p.tok.Type != token.T_RBRACKET {
				p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected , or ] in array literal, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
				return nil
			}
		}
		p.nextToken() // consume ]

		return &ast.ArrayNode{
			Elements: elements,
			Pos:      ast.Position(pos),
		}
	}

	return nil
}

func (p *Parser) parseExpressionStatement() ast.Node {
	expr := p.parseExpression()
	if expr == nil {
		return nil
	}

	if p.tok.Type != token.T_SEMICOLON {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected ; after expression, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
		return nil
	}
	p.nextToken() // consume ;

	return &ast.ExpressionStmt{
		Expr: expr,
		Pos:  expr.GetPos(),
	}
}

func (p *Parser) parseIfStatement() ast.Node {
	pos := p.tok.Pos
	p.nextToken() // consume if

	if p.tok.Type != token.T_LPAREN {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected ( after if, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
		return nil
	}
	p.nextToken()

	condition := p.parseExpression()
	if condition == nil {
		return nil
	}

	if p.tok.Type != token.T_RPAREN {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected ) after if condition, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
		return nil
	}
	p.nextToken()

	if p.tok.Type != token.T_LBRACE {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected { after if condition, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
		return nil
	}
	p.nextToken()

	var body []ast.Node
	for p.tok.Type != token.T_RBRACE && p.tok.Type != token.T_EOF {
		if stmt := p.parseStatement(); stmt != nil {
			body = append(body, stmt)
		}
		// Don't consume tokens here - parseStatement handles that
	}

	if p.tok.Type != token.T_RBRACE {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected } to close if body, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
		return nil
	}
	p.nextToken() // consume }

	var elseifs []*ast.ElseIfNode
	var elseNode *ast.ElseNode

	// Parse any elseif clauses
	for p.tok.Type == token.T_ELSEIF {
		elseifNode := p.parseElseIfClause()
		if elseifNode == nil {
			return nil
		}
		elseifs = append(elseifs, elseifNode)
	}

	// Parse optional else clause
	if p.tok.Type == token.T_ELSE {
		elseNode = p.parseElseClause()
		if elseNode == nil {
			return nil
		}
	}

	return &ast.IfNode{
		Condition: condition,
		Body:      body,
		ElseIfs:   elseifs,
		Else:      elseNode,
		Pos:       ast.Position(pos),
	}
}

func (p *Parser) parseElseIfClause() *ast.ElseIfNode {
	pos := p.tok.Pos
	p.nextToken() // consume elseif

	if p.tok.Type != token.T_LPAREN {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected ( after elseif, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
		return nil
	}
	p.nextToken()

	condition := p.parseExpression()
	if condition == nil {
		return nil
	}

	if p.tok.Type != token.T_RPAREN {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected ) after elseif condition, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
		return nil
	}
	p.nextToken()

	if p.tok.Type != token.T_LBRACE {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected { after elseif condition, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
		return nil
	}
	p.nextToken()

	var body []ast.Node
	for p.tok.Type != token.T_RBRACE && p.tok.Type != token.T_EOF {
		if stmt := p.parseStatement(); stmt != nil {
			body = append(body, stmt)
		}
		p.nextToken()
	}

	if p.tok.Type != token.T_RBRACE {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected } to close elseif body, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
		return nil
	}
	p.nextToken() // consume }

	return &ast.ElseIfNode{
		Condition: condition,
		Body:      body,
		Pos:       ast.Position(pos),
	}
}

func (p *Parser) parseElseClause() *ast.ElseNode {
	pos := p.tok.Pos
	p.nextToken() // consume else

	if p.tok.Type != token.T_LBRACE {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected { after else, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
		return nil
	}
	p.nextToken()

	var body []ast.Node
	for p.tok.Type != token.T_RBRACE && p.tok.Type != token.T_EOF {
		if stmt := p.parseStatement(); stmt != nil {
			body = append(body, stmt)
		}
		p.nextToken()
	}

	if p.tok.Type != token.T_RBRACE {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected } to close else body, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
		return nil
	}
	p.nextToken() // consume }

	return &ast.ElseNode{
		Body: body,
		Pos:  ast.Position(pos),
	}
}

func (p *Parser) parseClassDeclaration() ast.Node {
	pos := p.tok.Pos
	p.nextToken() // consume 'class'

	if p.tok.Type != token.T_STRING {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected class name, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
		return nil
	}

	name := p.tok.Literal
	p.nextToken()

	// Check for extends clause
	var extends string
	if p.tok.Type == token.T_EXTENDS {
		p.nextToken() // consume 'extends'
		if p.tok.Type != token.T_STRING {
			p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected parent class name after extends, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
			return nil
		}
		extends = p.tok.Literal
		p.nextToken()
	}

	// Check for implements clause
	var implements []string
	if p.tok.Type == token.T_IMPLEMENTS {
		p.nextToken() // consume 'implements'
		for {
			if p.tok.Type != token.T_STRING {
				p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected interface name after implements, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
				return nil
			}
			implements = append(implements, p.tok.Literal)
			p.nextToken()

			if p.tok.Type != token.T_COMMA {
				break
			}
			p.nextToken() // consume comma
		}
	}

	if p.tok.Type != token.T_LBRACE {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected { after class declaration for %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal))
		return nil
	}
	p.nextToken() // consume {

	var properties []ast.Node
	var methods []ast.Node
	for p.tok.Type != token.T_RBRACE && p.tok.Type != token.T_EOF {
		// Handle visibility modifiers for methods and properties
		if p.tok.Type == token.T_PUBLIC || p.tok.Type == token.T_PRIVATE || p.tok.Type == token.T_PROTECTED {
			visibility := p.tok.Literal
			p.nextToken()

			if p.tok.Type == token.T_FUNCTION {
				if method := p.parseFunction(); method != nil {
					if fn, ok := method.(*ast.FunctionNode); ok {
						fn.Visibility = visibility
					}
					methods = append(methods, method)
				}
			} else if p.tok.Type == token.T_VARIABLE {
				if prop := p.parsePropertyDeclaration(); prop != nil {
					if pn, ok := prop.(*ast.PropertyNode); ok {
						pn.Visibility = visibility
					}
					properties = append(properties, prop)
				}
			} else {
				p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected function or property declaration after visibility modifier %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, visibility, p.tok.Literal))
				p.nextToken()
			}
		} else if p.tok.Type == token.T_FUNCTION {
			if method := p.parseFunction(); method != nil {
				methods = append(methods, method)
			}
		} else if p.tok.Type == token.T_VARIABLE {
			// Parse property declaration
			if prop := p.parsePropertyDeclaration(); prop != nil {
				properties = append(properties, prop)
			}
		} else {
			p.errors = append(p.errors, fmt.Sprintf("line %d:%d: unexpected token %s in class %s body", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal, name))
			p.nextToken()
		}
	}

	if p.tok.Type != token.T_RBRACE {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected } to close class %s body, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal))
		return nil
	}
	p.nextToken() // consume }

	return &ast.ClassNode{
		Name:       name,
		Extends:    extends,
		Implements: implements,
		Properties: properties,
		Methods:    methods,
		Pos:        ast.Position(pos),
	}
}

func (p *Parser) parsePropertyDeclaration() ast.Node {
	pos := p.tok.Pos
	name := p.tok.Literal[1:] // Remove $ prefix
	p.nextToken()

	if p.tok.Type != token.T_SEMICOLON {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected ; after property declaration $%s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal))
		return nil
	}
	p.nextToken()

	return &ast.PropertyNode{
		Name: name,
		Pos:  ast.Position(pos),
	}
}

func (p *Parser) parseBlockStatement() []ast.Node {
	p.nextToken() // consume {

	var statements []ast.Node
	for p.tok.Type != token.T_RBRACE && p.tok.Type != token.T_EOF {
		if stmt := p.parseStatement(); stmt != nil {
			statements = append(statements, stmt)
		}
		// Don't consume tokens here - parseStatement handles that
	}

	if p.tok.Type != token.T_RBRACE {
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected } to close block, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
		return nil
	}
	p.nextToken() // consume closing brace

	return statements
}

func (p *Parser) Errors() []string {
	return p.errors
}

// peekToken returns the next token without consuming it
func (p *Parser) peekToken() token.Token {
	return p.l.PeekToken()
}

// parseSimpleExpression parses a simple expression (identifier, literal, etc.)
func (p *Parser) parseSimpleExpression() ast.Node {
	switch p.tok.Type {
	case token.T_NEW:
		pos := p.tok.Pos
		p.nextToken() // consume 'new'

		if p.tok.Type != token.T_STRING {
			p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected class name after new, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
			return nil
		}
		className := p.tok.Literal
		p.nextToken()

		if p.tok.Type != token.T_LPAREN {
			p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected ( after class name %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, className, p.tok.Literal))
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
			p.errors = append(p.errors, fmt.Sprintf("line %d:%d: expected ) after arguments for %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, className, p.tok.Literal))
			return nil
		}
		p.nextToken() // consume )

		return &ast.NewNode{
			ClassName: className,
			Args:      args,
			Pos:       ast.Position(pos),
		}
	case token.T_IDENTIFIER:
		node := &ast.IdentifierNode{
			Value: p.tok.Literal,
			Pos:   ast.Position(p.tok.Pos),
		}
		p.nextToken()
		return node
	case token.T_STRING:
		node := &ast.StringNode{
			Value: p.tok.Literal,
			Pos:   ast.Position(p.tok.Pos),
		}
		p.nextToken()
		return node
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
		node := &ast.StringNode{
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
		node := &ast.VariableNode{
			Name: p.tok.Literal[1:], // Remove $ prefix
			Pos:  ast.Position(p.tok.Pos),
		}
		p.nextToken()
		return node
	default:
		p.errors = append(p.errors, fmt.Sprintf("line %d:%d: unexpected token %s in expression", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal))
		p.nextToken()
		return nil
	}
}
