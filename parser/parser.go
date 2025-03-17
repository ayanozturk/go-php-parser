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
		pos := p.tok.Pos
		comment := p.tok.Literal
		p.nextToken() // Skip comments
		return &ast.CommentNode{
			Value: comment,
			Pos:   ast.Position(pos),
		}
	case token.T_FUNCTION:
		node = p.parseFunctionDeclaration()
		return node
	case token.T_VARIABLE:
		// Check if this is a method call
		pos := p.tok.Pos
		name := p.tok.Literal[1:] // Remove $ prefix
		p.nextToken()

		if p.tok.Type == token.T_OBJECT_OP {
			p.nextToken() // consume ->
			if p.tok.Type != token.T_STRING {
				p.errors = append(p.errors, "expected method name after ->")
				return nil
			}
			methodName := p.tok.Literal
			p.nextToken()

			if p.tok.Type != token.T_LPAREN {
				p.errors = append(p.errors, "expected ( after method name")
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
				p.errors = append(p.errors, "expected ) in argument list")
				return nil
			}
			p.nextToken() // consume )

			if p.tok.Type != token.T_SEMICOLON {
				p.errors = append(p.errors, "expected ; after method call")
				return nil
			}
			p.nextToken() // consume ;

			return &ast.ExpressionStmt{
				Expr: &ast.MethodCallNode{
					Object: &ast.VariableNode{
						Name: name,
						Pos:  ast.Position(pos),
					},
					Method: methodName,
					Args:   args,
					Pos:    ast.Position(pos),
				},
				Pos: ast.Position(pos),
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
			p.errors = append(p.errors, "expected ; after echo statement")
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

	// Parse visibility modifier if present
	var visibility string
	if p.tok.Type == token.T_PUBLIC || p.tok.Type == token.T_PRIVATE || p.tok.Type == token.T_PROTECTED {
		visibility = p.tok.Literal
		p.nextToken()
	}

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

	// Parse return type if present
	var returnType string
	if p.tok.Type == token.T_COLON {
		p.nextToken() // consume :
		if p.tok.Type != token.T_STRING {
			p.errors = append(p.errors, "expected return type after :")
			return nil
		}
		returnType = p.tok.Literal
		p.nextToken()
	}

	// Check if this is an interface method declaration (no body, just semicolon)
	if p.tok.Type == token.T_SEMICOLON {
		p.nextToken() // consume ;
		return &ast.FunctionNode{
			Name:       name,
			Visibility: visibility,
			ReturnType: returnType,
			Params:     params,
			Body:       nil, // Interface methods have no body
			Pos:        ast.Position(pos),
		}
	}

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
		Name:       name,
		Visibility: visibility,
		ReturnType: returnType,
		Params:     params,
		Body:       body,
		Pos:        ast.Position(pos),
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

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

func (p *Parser) parseExpression() ast.Node {
	var expr ast.Node

	switch p.tok.Type {
	case token.T_NEW:
		pos := p.tok.Pos
		p.nextToken() // consume 'new'
		if p.tok.Type != token.T_STRING {
			p.errors = append(p.errors, "expected class name after new")
			return nil
		}
		className := p.tok.Literal
		p.nextToken()

		if p.tok.Type != token.T_LPAREN {
			p.errors = append(p.errors, "expected ( after class name")
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
			p.errors = append(p.errors, "expected ) in argument list")
			return nil
		}
		p.nextToken() // consume )

		expr = &ast.NewNode{
			ClassName: className,
			Args:      args,
			Pos:       ast.Position(pos),
		}
	case token.T_VARIABLE:
		pos := p.tok.Pos
		name := p.tok.Literal[1:] // Remove $ prefix
		p.nextToken()

		// Check for method call
		if p.tok.Type == token.T_OBJECT_OP {
			p.nextToken() // consume ->
			if p.tok.Type != token.T_STRING {
				p.errors = append(p.errors, "expected method name after ->")
				return nil
			}
			methodName := p.tok.Literal
			p.nextToken()

			if p.tok.Type != token.T_LPAREN {
				p.errors = append(p.errors, "expected ( after method name")
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
				p.errors = append(p.errors, "expected ) in argument list")
				return nil
			}
			p.nextToken() // consume )

			expr = &ast.MethodCallNode{
				Object: &ast.VariableNode{
					Name: name,
					Pos:  ast.Position(pos),
				},
				Method: methodName,
				Args:   args,
				Pos:    ast.Position(pos),
			}
		} else {
			expr = &ast.VariableNode{
				Name: name,
				Pos:  ast.Position(pos),
			}
		}
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
			// Split string into parts based on variable interpolation
			current := ""
			for i := 0; i < len(str); i++ {
				if str[i] == '$' {
					// Add the current string part if any
					if current != "" {
						parts = append(parts, &ast.StringLiteral{
							Value: current,
							Pos:   ast.Position(strPos),
						})
						current = ""
					}
					// Look for variable name
					varName := ""
					for j := i + 1; j < len(str); j++ {
						if isLetter(str[j]) || isDigit(str[j]) || str[j] == '_' {
							varName += string(str[j])
						} else {
							break
						}
					}
					if varName != "" {
						parts = append(parts, &ast.VariableNode{
							Name: varName,
							Pos:  ast.Position(strPos),
						})
						i += len(varName) // Skip past variable name
					}
				} else {
					current += string(str[i])
				}
			}
			// Add any remaining string part
			if current != "" {
				parts = append(parts, &ast.StringLiteral{
					Value: current,
					Pos:   ast.Position(strPos),
				})
			}
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

func (p *Parser) parseClassDeclaration() ast.Node {
	pos := p.tok.Pos
	p.nextToken() // consume 'class'

	if p.tok.Type != token.T_STRING {
		p.errors = append(p.errors, "expected class name")
		return nil
	}

	name := p.tok.Literal
	p.nextToken()

	// Check for extends clause
	var extends string
	if p.tok.Type == token.T_EXTENDS {
		p.nextToken() // consume 'extends'
		if p.tok.Type != token.T_STRING {
			p.errors = append(p.errors, "expected parent class name after extends")
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
				p.errors = append(p.errors, "expected interface name after implements")
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
		p.errors = append(p.errors, "expected { after class declaration")
		return nil
	}
	p.nextToken() // consume {

	var properties []ast.Node
	var methods []ast.Node
	for p.tok.Type != token.T_RBRACE && p.tok.Type != token.T_EOF {
		// Handle visibility modifiers for methods
		if p.tok.Type == token.T_PUBLIC || p.tok.Type == token.T_PRIVATE || p.tok.Type == token.T_PROTECTED {
			if method := p.parseFunctionDeclaration(); method != nil {
				methods = append(methods, method)
			}
		} else if p.tok.Type == token.T_FUNCTION {
			if method := p.parseFunctionDeclaration(); method != nil {
				methods = append(methods, method)
			}
		} else if p.tok.Type == token.T_VARIABLE {
			// Parse property declaration
			if prop := p.parsePropertyDeclaration(); prop != nil {
				properties = append(properties, prop)
			}
		} else {
			p.errors = append(p.errors, fmt.Sprintf("unexpected token %s in class body", p.tok.Type))
			p.nextToken()
		}
	}

	if p.tok.Type != token.T_RBRACE {
		p.errors = append(p.errors, "expected } to close class body")
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
		p.errors = append(p.errors, "expected ; after property declaration")
		return nil
	}
	p.nextToken()

	return &ast.PropertyNode{
		Name: name,
		Pos:  ast.Position(pos),
	}
}

func (p *Parser) Errors() []string {
	return p.errors
}
