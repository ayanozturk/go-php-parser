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
		p.errors = append(p.errors, "expected <?php at start of file")
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
		p.nextToken() // Skip comments
		return nil
	case token.T_FUNCTION:
		node = p.parseFunctionDeclaration()
		return node
	case token.T_VARIABLE:
		// Don't consume semicolon here - let parseVariableStatement handle it
		node = p.parseVariableStatement()
		return node
	case token.T_IF:
		node = p.parseIfStatement()
		return node
	case token.T_STRING:
		if p.tok.Literal == "echo" {
			p.nextToken() // consume echo
			expr := p.parseExpression()
			if expr == nil {
				return nil
			}
			if p.tok.Type != token.T_SEMICOLON {
				p.errors = append(p.errors, "expected ; after echo statement")
				return nil
			}
			p.nextToken() // consume ;
			return &ast.ExpressionStmt{
				Expr: expr,
				Pos:  ast.Position(p.tok.Pos),
			}
		}
		// Handle other expressions
		return p.parseExpressionStatement()
	case token.T_SEMICOLON:
		p.nextToken() // skip empty statements
		return nil
	default:
		// Try parsing as expression statement
		if expr := p.parseExpression(); expr != nil {
			if p.tok.Type != token.T_SEMICOLON {
				p.errors = append(p.errors, "expected ; after expression")
				return nil
			}
			p.nextToken() // consume ;
			return &ast.ExpressionStmt{
				Expr: expr,
				Pos:  expr.GetPos(),
			}
		}
		p.errors = append(p.errors, fmt.Sprintf("unexpected token %s in statement", p.tok.Type))
		p.nextToken()
		return nil
	}
}

func (p *Parser) parseFunctionDeclaration() ast.Node {
	pos := p.tok.Pos
	p.nextToken() // consume 'function'

	if p.tok.Type != token.T_STRING {
		p.errors = append(p.errors, "expected function name")
		return nil
	}

	name := p.tok.Literal
	p.nextToken()

	if p.tok.Type != token.T_LPAREN {
		p.errors = append(p.errors, "expected ( after function name")
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
			p.errors = append(p.errors, "expected , or ) in parameter list")
			return nil
		}
	}
	p.nextToken() // consume )

	if p.tok.Type != token.T_LBRACE {
		p.errors = append(p.errors, "expected { after function parameters")
		return nil
	}
	var body []ast.Node
	p.nextToken() // consume {

	for p.tok.Type != token.T_RBRACE && p.tok.Type != token.T_EOF {
		if stmt := p.parseStatement(); stmt != nil {
			body = append(body, stmt)
		}
		// Don't consume tokens here - parseStatement handles that
	}

	if p.tok.Type != token.T_RBRACE {
		p.errors = append(p.errors, "expected } to close function body")
		return nil
	}
	p.nextToken() // consume closing brace

	return &ast.FunctionNode{
		Name:   name,
		Params: params,
		Body:   body,
		Pos:    ast.Position(pos),
	}
}

func (p *Parser) parseParameter() ast.Node {
	pos := p.tok.Pos

	// Parse parameter type
	var paramType string
	if p.tok.Type == token.T_STRING || p.tok.Type == token.T_ARRAY || p.tok.Type == token.T_CALLABLE {
		paramType = p.tok.Literal
		p.nextToken()
	}

	// Check for by-reference
	byRef := false
	if p.tok.Type == token.T_AMPERSAND {
		byRef = true
		p.nextToken()
	}

	// Check for variadic
	variadic := false
	if p.tok.Type == token.T_ELLIPSIS {
		variadic = true
		p.nextToken()
	}

	// Parse parameter name
	if p.tok.Type != token.T_VARIABLE {
		p.errors = append(p.errors, "expected parameter name")
		return nil
	}
	name := p.tok.Literal[1:] // Remove $ prefix
	p.nextToken()

	// Parse default value if present
	var defaultValue ast.Node
	if p.tok.Type == token.T_ASSIGN {
		p.nextToken()
		defaultValue = p.parseExpression()
	}

	// Variadic parameters cannot have default values
	if variadic && defaultValue != nil {
		p.errors = append(p.errors, "variadic parameter cannot have a default value")
		return nil
	}

	return &ast.ParameterNode{
		Name:         name,
		Type:         paramType,
		ByRef:        byRef,
		Variadic:     variadic,
		DefaultValue: defaultValue,
		Pos:          ast.Position(pos),
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
			p.errors = append(p.errors, "expected ; after assignment")
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
	var expr ast.Node

	switch p.tok.Type {
	case token.T_STRING:
		// Check for function calls
		strPos := p.tok.Pos
		name := p.tok.Literal
		p.nextToken()

		if p.tok.Type == token.T_LPAREN {
			p.nextToken() // consume (

			// Parse arguments
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
				p.errors = append(p.errors, "expected ) in argument list")
				return nil
			}
			p.nextToken() // consume )

			expr = &ast.FunctionCall{
				Name:      name,
				Arguments: args,
				Pos:       ast.Position(strPos),
			}
		} else {
			expr = &ast.StringLiteral{
				Value: name,
				Pos:   ast.Position(strPos),
			}
		}
	case token.T_CONSTANT_STRING:
		strPos := p.tok.Pos
		str := p.tok.Literal
		p.nextToken()
		expr = &ast.StringLiteral{
			Value: str,
			Pos:   ast.Position(strPos),
		}
	case token.T_CONSTANT_ENCAPSED_STRING:
		strPos := p.tok.Pos
		str := p.tok.Literal[1 : len(p.tok.Literal)-1] // Remove quotes

		// Handle string interpolation
		var parts []ast.Node
		if len(str) > 0 {
			// Simple case: no interpolation, just a string literal
			parts = append(parts, &ast.StringLiteral{
				Value: str,
				Pos:   ast.Position(strPos),
			})
		}

		p.nextToken()
		expr = &ast.InterpolatedStringLiteral{
			Parts: parts,
			Pos:   ast.Position(strPos),
		}
	case token.T_LNUMBER:
		numPos := p.tok.Pos
		num, err := strconv.ParseInt(p.tok.Literal, 10, 64)
		if err != nil {
			p.errors = append(p.errors, fmt.Sprintf("invalid integer literal: %s", p.tok.Literal))
			return nil
		}
		p.nextToken()
		expr = &ast.IntegerLiteral{
			Value: num,
			Pos:   ast.Position(numPos),
		}
	case token.T_DNUMBER:
		numPos := p.tok.Pos
		num, err := strconv.ParseFloat(p.tok.Literal, 64)
		if err != nil {
			p.errors = append(p.errors, fmt.Sprintf("invalid float literal: %s", p.tok.Literal))
			return nil
		}
		p.nextToken()
		expr = &ast.FloatLiteral{
			Value: num,
			Pos:   ast.Position(numPos),
		}
	case token.T_TRUE:
		pos := p.tok.Pos
		p.nextToken()
		expr = &ast.BooleanLiteral{
			Value: true,
			Pos:   ast.Position(pos),
		}
	case token.T_FALSE:
		pos := p.tok.Pos
		p.nextToken()
		expr = &ast.BooleanLiteral{
			Value: false,
			Pos:   ast.Position(pos),
		}
	case token.T_NULL:
		pos := p.tok.Pos
		p.nextToken()
		expr = &ast.NullLiteral{
			Pos: ast.Position(pos),
		}
	case token.T_VARIABLE:
		expr = p.parseVariableStatement()
	default:
		p.errors = append(p.errors, "unexpected token in expression")
		return nil
	}

	return expr
}

func (p *Parser) parseExpressionStatement() ast.Node {
	expr := p.parseExpression()
	if expr == nil {
		return nil
	}

	if p.tok.Type != token.T_SEMICOLON {
		p.errors = append(p.errors, "expected ; after expression")
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
		p.errors = append(p.errors, "expected ( after if")
		return nil
	}
	p.nextToken()

	condition := p.parseExpression()
	if condition == nil {
		return nil
	}

	if p.tok.Type != token.T_RPAREN {
		p.errors = append(p.errors, "expected ) after if condition")
		return nil
	}
	p.nextToken()

	if p.tok.Type != token.T_LBRACE {
		p.errors = append(p.errors, "expected { after if condition")
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
		p.errors = append(p.errors, "expected } to close if body")
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
		p.errors = append(p.errors, "expected ( after elseif")
		return nil
	}
	p.nextToken()

	condition := p.parseExpression()
	if condition == nil {
		return nil
	}

	if p.tok.Type != token.T_RPAREN {
		p.errors = append(p.errors, "expected ) after elseif condition")
		return nil
	}
	p.nextToken()

	if p.tok.Type != token.T_LBRACE {
		p.errors = append(p.errors, "expected { after elseif condition")
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
		p.errors = append(p.errors, "expected } to close elseif body")
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
		p.errors = append(p.errors, "expected { after else")
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
		p.errors = append(p.errors, "expected } to close else body")
		return nil
	}
	p.nextToken() // consume }

	return &ast.ElseNode{
		Body: body,
		Pos:  ast.Position(pos),
	}
}

func (p *Parser) Errors() []string {
	return p.errors
}
