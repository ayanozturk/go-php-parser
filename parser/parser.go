package parser

import (
	"fmt"
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"go-phpcs/token"
	"strconv"
	"strings"
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
		p.errors = append(p.errors, fmt.Sprintf(format, args...))
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
	default:
		// Try parsing as expression statement
		if expr := p.parseExpression(); expr != nil {
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
		p.addError("line %d:%d: unexpected token %s in statement", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		p.nextToken()
		return nil, nil
	}
}

// parseTypeHint parses a type hint (nullable, union, FQCN, etc.)
func (p *Parser) parseTypeHint() string {
	// Only emit errors for real type hints, not when inside docblocks/comments
	isDocblockContext := p.tok.Type == token.T_DOC_COMMENT
	typeHint := ""
	lastWasPipe := false
	segmentCount := 0
	for {
		// Nullable type
		if p.tok.Type == token.T_QUESTION {
			typeHint += "?"
			p.nextToken()
		}
		typeSegment := ""
		for (p.tok.Literal == "\\" || p.tok.Type == token.T_BACKSLASH) && p.peekToken().Type == token.T_STRING {
			typeSegment += "\\"
			p.nextToken() // consume backslash
			typeSegment += p.tok.Literal
			p.nextToken() // consume string
		}
		if typeSegment == "" {
			if p.tok.Type == token.T_STRING || p.tok.Type == token.T_ARRAY || p.tok.Type == token.T_NULL || p.tok.Type == token.T_MIXED || p.tok.Type == token.T_CALLABLE || p.tok.Literal == "mixed" {
				typeSegment += p.tok.Literal
				p.nextToken()
			}
		}
		if typeSegment != "" {
			typeHint += typeSegment
			segmentCount++
			lastWasPipe = false
			if p.tok.Type == token.T_LBRACKET {
				typeHint += "[]"
				p.nextToken()
				if p.tok.Type != token.T_RBRACKET {
					if !isDocblockContext {
						p.errors = append(p.errors, "expected ']' after array type in type hint")
					}
					return typeHint
				}
				p.nextToken()
			}
		} else {
			if lastWasPipe {
				if !isDocblockContext {
					p.errors = append(p.errors, "empty type segment in union type")
				}
			}
			break
		}
		if p.tok.Type == token.T_PIPE {
			if lastWasPipe {
				if !isDocblockContext {
					p.errors = append(p.errors, "consecutive '|' in union type")
				}
			}
			typeHint += "|"
			p.nextToken()
			lastWasPipe = true
			continue
		}
		break
	}
	if lastWasPipe {
		if !isDocblockContext {
			p.errors = append(p.errors, "union type ends with '|' or has empty segment")
		}
	}
	if segmentCount == 0 {
		if !isDocblockContext {
			p.errors = append(p.errors, "empty union type")
		}
	}
	return typeHint
}

func (p *Parser) parseFunction(modifiers []string) (ast.Node, error) {
	pos := p.tok.Pos
	p.nextToken() // consume 'function'

	var name string
	if p.tok.Type == token.T_STRING {
		name = p.tok.Literal
		p.nextToken()
	}

	if p.tok.Type != token.T_LPAREN {
		p.addError("line %d:%d: expected ( after function name %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume (

	var params []ast.Node
	for p.tok.Type != token.T_RPAREN {
		param := p.parseParameter()
		if param == nil {
			return nil, nil
		}
		params = append(params, param)

		if p.tok.Type == token.T_COMMA {
			p.nextToken()
		}
	}
	p.nextToken() // consume )

	// Parse return type hint
	var returnType string
	if p.tok.Type == token.T_COLON {
		p.nextToken()
		returnType = p.parseTypeHint()
	}

	if p.tok.Type != token.T_LBRACE {
		p.addError("line %d:%d: expected { to start function body for %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume {

	body := p.parseBlockStatement()

	if p.tok.Type != token.T_RBRACE {
		p.addError("line %d:%d: expected } to close function %s body, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume }

	return &ast.FunctionNode{
		Name:       name,
		Params:     params,
		ReturnType: returnType,
		Body:       body,
		Modifiers:  modifiers,
		Pos:        ast.Position(pos),
	}, nil
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

// precedence table for PHP operators (higher number = higher precedence)
var phpOperatorPrecedence = map[token.TokenType]int{
	token.T_ASSIGN:         1,
	token.T_PLUS_EQUAL:     1,
	token.T_MINUS_EQUAL:    1,
	token.T_MUL_EQUAL:      1,
	token.T_DIV_EQUAL:      1,
	token.T_MOD_EQUAL:      1,
	token.T_AND_EQUAL:      1,
	token.T_CONCAT_EQUAL:   1,
	token.T_XOR_EQUAL:      1,
	token.T_COALESCE_EQUAL: 1,

	token.T_BOOLEAN_OR:     2, // ||
	token.T_BOOLEAN_AND:    3, // &&
	token.T_PIPE:           4, // |
	token.T_AMPERSAND:      5, // &
	// token.T_XOR_EQUAL:   5, // ^ (already included as assignment above)
	token.T_IS_EQUAL:         6,
	token.T_IS_NOT_EQUAL:     6,
	token.T_IS_IDENTICAL:     6,
	token.T_IS_NOT_IDENTICAL: 6,
	token.T_IS_SMALLER:       7,
	token.T_IS_GREATER:       7,
	token.T_SPACESHIP:        7,
	token.T_INSTANCEOF:       8,

	token.T_COALESCE:      9, // ??
	token.T_PLUS:         10,
	token.T_MINUS:        10,
	token.T_DOT:          10,
	token.T_MULTIPLY:     11,
	token.T_DIVIDE:       11,
	token.T_MODULO:       11,
}

// operator associativity (true = right-associative)
var phpOperatorRightAssoc = map[token.TokenType]bool{
	token.T_ASSIGN:         true,
	token.T_PLUS_EQUAL:     true,
	token.T_MINUS_EQUAL:    true,
	token.T_MUL_EQUAL:      true,
	token.T_DIV_EQUAL:      true,
	token.T_MOD_EQUAL:      true,
	token.T_AND_EQUAL:      true,
	token.T_CONCAT_EQUAL:   true,
	token.T_XOR_EQUAL:      true,
	token.T_COALESCE_EQUAL: true,
	token.T_COALESCE:       true,
}

func (p *Parser) parseExpression() ast.Node {
	return p.parseExpressionWithPrecedence(0, true)
}

// parseExpressionWithPrecedence parses expressions with correct precedence. Only validateAssignmentTarget for top-level expressions.
func (p *Parser) parseExpressionWithPrecedence(minPrec int, validateAssignmentTarget bool) ast.Node {
	// Array literals
	if p.tok.Type == token.T_LBRACKET || p.tok.Type == token.T_ARRAY {
		return p.parseArrayLiteral()
	}
	left := p.parseSimpleExpression()
	if left == nil {
		p.addError("line %d:%d: expected left operand, got nil", p.tok.Pos.Line, p.tok.Pos.Column)
		return nil
	}
	for {
		prec, isOp := phpOperatorPrecedence[p.tok.Type]
		if !isOp || prec < minPrec {
			break
		}
		op := p.tok.Type
		operator := p.tok.Literal
		if op == token.T_BOOLEAN_OR {
			operator = "||"
		}
		pos := p.tok.Pos
		assocRight := phpOperatorRightAssoc[op]
		nextMinPrec := prec + 1
		if assocRight {
			nextMinPrec = prec
		}
		p.nextToken()
		right := p.parseExpressionWithPrecedence(nextMinPrec, false)
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



// isAssignmentOperator returns true if the operator is an assignment
func isAssignmentOperator(op token.TokenType) bool {
	switch op {
	case token.T_ASSIGN, token.T_PLUS_EQUAL, token.T_MINUS_EQUAL, token.T_MUL_EQUAL, token.T_DIV_EQUAL, token.T_MOD_EQUAL, token.T_AND_EQUAL, token.T_CONCAT_EQUAL, token.T_XOR_EQUAL, token.T_COALESCE_EQUAL:
		return true
	default:
		return false
	}
}

// isValidAssignmentTarget returns true if node is a valid assignment target (VariableNode only for now)
func isValidAssignmentTarget(node ast.Node) bool {
	if node == nil {
		return false
	}
	switch node.(type) {
	case *ast.VariableNode:
		return true
	default:
		return false
	}
}

func (p *Parser) isBinaryOperator(tokenType token.TokenType) bool {
	switch tokenType {
	case token.T_PLUS, token.T_MINUS, token.T_MULTIPLY, token.T_DIVIDE, token.T_MODULO,
		token.T_IS_EQUAL, token.T_IS_NOT_EQUAL, token.T_IS_SMALLER, token.T_IS_GREATER,
		token.T_DOT,        // Support string concatenation
		token.T_COALESCE,   // Support null coalescing operator ??
		token.T_BOOLEAN_OR, // Support double pipe || operator
		token.T_BOOLEAN_AND, // Support double ampersand && operator
		token.T_PIPE: // Support single pipe | operator
		return true
	case token.T_ASSIGN:
		return true
	default:
		return false
	}
}

func (p *Parser) parseArrayElement() ast.Node {
	pos := p.tok.Pos
	var key ast.Node
	var value ast.Node
	var byRef bool
	var unpack bool

	// Check for spread operator (...)
	if p.tok.Type == token.T_ELLIPSIS {
		unpack = true
		p.nextToken()
	}

	// Check for by-reference operator (&)
	if p.tok.Type == token.T_AMPERSAND {
		byRef = true
		p.nextToken()
	}

	// Parse key if present (support class constant fetches as keys)
	if p.tok.Type == token.T_STRING || p.tok.Type == token.T_BACKSLASH ||
		p.tok.Type == token.T_SELF || p.tok.Type == token.T_PARENT || p.tok.Type == token.T_STATIC {
		// Accumulate fully qualified class name
		var className strings.Builder
		classPos := p.tok.Pos
		for p.tok.Type == token.T_STRING || p.tok.Type == token.T_BACKSLASH {
			className.WriteString(p.tok.Literal)
			p.nextToken()
		}
		if p.tok.Type == token.T_DOUBLE_COLON {
			p.nextToken() // consume ::
			if p.tok.Type == token.T_STRING {
				constName := p.tok.Literal
				key = &ast.ClassConstFetchNode{
					Class: className.String(),
					Const: constName,
					Pos:   ast.Position(classPos),
				}
				p.nextToken()
			} else if p.tok.Type == token.T_CLASS_CONST || p.tok.Type == token.T_CLASS {
				// Support Foo::class
				key = &ast.ClassConstFetchNode{
					Class: className.String(),
					Const: "class",
					Pos:   ast.Position(classPos),
				}
				p.nextToken()
			} else {
				p.addError("line %d:%d: expected constant name after :: in array key, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
				return nil
			}
		} else {
			// Not a class constant fetch, fallback to fully qualified name
			fqdn := className.String()
			key = &ast.IdentifierNode{
				Value: fqdn,
				Pos:   ast.Position(classPos),
			}
		}
	} else if p.tok.Type == token.T_CONSTANT_STRING {
		key = p.parseSimpleExpression()
	}

	if p.tok.Type == token.T_DOUBLE_ARROW {
		p.nextToken() // consume =>
	} else {
		// If no =>, treat the expression as a value
		value = key
		key = nil
	}

	// Parse value if not already set
	if value == nil {
		if byRef && p.tok.Type != token.T_VARIABLE {
			p.addError("line %d:%d: by-reference must be followed by a variable", p.tok.Pos.Line, p.tok.Pos.Column)
			return nil
		}
		value = p.parseExpression()
		if value == nil {
			return nil
		}
	}

	return &ast.ArrayItemNode{
		Key:    key,
		Value:  value,
		ByRef:  byRef,
		Unpack: unpack,
		Pos:    ast.Position(pos),
	}
}

func (p *Parser) parseArrayLiteral() ast.Node {
	pos := p.tok.Pos

	// Handle array() syntax
	if p.tok.Type == token.T_ARRAY {
		p.nextToken() // consume array
		if p.tok.Type != token.T_LPAREN {
			p.addError("line %d:%d: expected ( after array, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
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
				p.addError("line %d:%d: expected , or ) in array literal, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
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
			if element := p.parseArrayElement(); element != nil {
				elements = append(elements, element)
			}

			if p.tok.Type == token.T_COMMA {
				p.nextToken() // consume comma
				continue
			}

			if p.tok.Type != token.T_RBRACKET {
				p.addError("line %d:%d: expected , or ] in array literal, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
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

func (p *Parser) parseIfStatement() (ast.Node, error) {
	pos := p.tok.Pos
	p.nextToken() // consume if

	if p.tok.Type != token.T_LPAREN {
		p.addError("line %d:%d: expected ( after if, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	p.nextToken()

	condition := p.parseExpression()
	if condition == nil {
		return nil, nil
	}

	if p.tok.Type != token.T_RPAREN {
		p.addError("line %d:%d: expected ) after if condition, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	p.nextToken()

	if p.tok.Type != token.T_LBRACE {
		p.addError("line %d:%d: expected { after if condition, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	p.nextToken()

	var body []ast.Node
	for p.tok.Type != token.T_RBRACE && p.tok.Type != token.T_EOF {
		if stmt, err := p.parseStatement(); stmt != nil {
			body = append(body, stmt)
		} else if err != nil {
			return nil, err
		}
		// Don't consume tokens here - parseStatement handles that
	}

	if p.tok.Type != token.T_RBRACE {
		p.addError("line %d:%d: expected } to close if body, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume }

	var elseifs []*ast.ElseIfNode
	var elseNode *ast.ElseNode

	// Parse any elseif clauses
	for p.tok.Type == token.T_ELSEIF {
		elseifNode, err := p.parseElseIfClause()
		if elseifNode == nil || err != nil {
			return nil, err
		}
		elseifs = append(elseifs, elseifNode)
	}

	// Parse optional else clause
	if p.tok.Type == token.T_ELSE {
		var err error
		elseNode, err = p.parseElseClause()
		if elseNode == nil || err != nil {
			return nil, err
		}
	}

	return &ast.IfNode{
		Condition: condition,
		Body:      body,
		ElseIfs:   elseifs,
		Else:      elseNode,
		Pos:       ast.Position(pos),
	}, nil
}

func (p *Parser) parseElseIfClause() (*ast.ElseIfNode, error) {
	pos := p.tok.Pos
	p.nextToken() // consume elseif

	if p.tok.Type != token.T_LPAREN {
		p.addError("line %d:%d: expected ( after elseif, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	p.nextToken()

	condition := p.parseExpression()
	if condition == nil {
		return nil, nil
	}

	if p.tok.Type != token.T_RPAREN {
		p.addError("line %d:%d: expected ) after elseif condition, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	p.nextToken()

	if p.tok.Type != token.T_LBRACE {
		p.addError("line %d:%d: expected { after elseif condition, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	p.nextToken()

	var body []ast.Node
	for p.tok.Type != token.T_RBRACE && p.tok.Type != token.T_EOF {
		if stmt, err := p.parseStatement(); stmt != nil {
			body = append(body, stmt)
		} else if err != nil {
			return nil, err
		}
		p.nextToken()
	}

	if p.tok.Type != token.T_RBRACE {
		p.addError("line %d:%d: expected } to close elseif body, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume }

	return &ast.ElseIfNode{
		Condition: condition,
		Body:      body,
		Pos:       ast.Position(pos),
	}, nil
}

func (p *Parser) parseElseClause() (*ast.ElseNode, error) {
	pos := p.tok.Pos
	p.nextToken() // consume else

	if p.tok.Type != token.T_LBRACE {
		p.addError("line %d:%d: expected { after else, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	p.nextToken()

	var body []ast.Node
	for p.tok.Type != token.T_RBRACE && p.tok.Type != token.T_EOF {
		if stmt, err := p.parseStatement(); stmt != nil {
			body = append(body, stmt)
		} else if err != nil {
			return nil, err
		}
		p.nextToken()
	}

	if p.tok.Type != token.T_RBRACE {
		p.addError("line %d:%d: expected } to close else body, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume }

	return &ast.ElseNode{
		Body: body,
		Pos:  ast.Position(pos),
	}, nil
}

func (p *Parser) parseClassDeclarationWithModifier(modifier string) (ast.Node, error) {
	// This is the same as parseClassDeclaration but attaches the modifier
	pos := p.tok.Pos
	p.nextToken() // consume 'class'

	if p.tok.Type != token.T_STRING {
		p.addError("line %d:%d: expected class name, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}

	name := p.tok.Literal
	p.nextToken()

	// Check for extends clause
	var extends string
	if p.tok.Type == token.T_EXTENDS {
		p.nextToken() // consume 'extends'
		if p.tok.Type != token.T_STRING {
			p.addError("line %d:%d: expected parent class name after extends, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
			return nil, nil
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
				p.addError("line %d:%d: expected interface name after implements, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
				return nil, nil
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
		p.addError("line %d:%d: expected { after class declaration for %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume {

	var properties []ast.Node
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
			if method, err := p.parseFunction(modifiers); method != nil {
				methods = append(methods, method)
			} else if err != nil {
				return nil, err
			}
			continue
		}
		if len(modifiers) > 0 {
			// If we saw modifiers but not a function, emit error and skip
			p.addError("line %d:%d: expected function after modifiers in class %s body, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
			p.nextToken()
			continue
		}
		if p.tok.Type == token.T_VARIABLE {
			if prop, err := p.parsePropertyDeclaration(); prop != nil {
				properties = append(properties, prop)
			} else if err != nil {
				return nil, err
			}
			continue
		}
		p.addError("line %d:%d: unexpected token %s in class %s body", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal, name)
		p.nextToken()
	}

	if p.tok.Type != token.T_RBRACE {
		p.addError("line %d:%d: expected } to close class %s body, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume }

	return &ast.ClassNode{
		Name:       name,
		Extends:    extends,
		Implements: implements,
		Properties: properties,
		Methods:    methods,
		Pos:        ast.Position(pos),
		Modifier:   modifier,
	}, nil
}

func (p *Parser) parseClassDeclaration() (ast.Node, error) {
	pos := p.tok.Pos
	p.nextToken() // consume 'class'

	if p.tok.Type != token.T_STRING {
		p.addError("line %d:%d: expected class name, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}

	name := p.tok.Literal
	p.nextToken()

	// Check for extends clause
	var extends string
	if p.tok.Type == token.T_EXTENDS {
		p.nextToken() // consume 'extends'
		if p.tok.Type != token.T_STRING {
			p.addError("line %d:%d: expected parent class name after extends, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
			return nil, nil
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
				p.addError("line %d:%d: expected interface name after implements, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
				return nil, nil
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
		p.addError("line %d:%d: expected { after class declaration for %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
		return nil, nil
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
				if method, err := p.parseFunction(nil); method != nil {
					if fn, ok := method.(*ast.FunctionNode); ok {
						fn.Visibility = visibility
					}
					methods = append(methods, method)
				} else if err != nil {
					return nil, err
				}
			} else if p.tok.Type == token.T_VARIABLE {
				if prop, err := p.parsePropertyDeclaration(); prop != nil {
					if pn, ok := prop.(*ast.PropertyNode); ok {
						pn.Visibility = visibility
					}
					properties = append(properties, prop)
				} else if err != nil {
					return nil, err
				}
			} else {
				p.addError("line %d:%d: expected function or property declaration after visibility modifier %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, visibility, p.tok.Literal)
				p.nextToken()
			}
		} else if p.tok.Type == token.T_FUNCTION {
			if method, err := p.parseFunction(nil); method != nil {
				methods = append(methods, method)
			} else if err != nil {
				return nil, err
			}
		} else if p.tok.Type == token.T_VARIABLE {
			// Parse property declaration
			if prop, err := p.parsePropertyDeclaration(); prop != nil {
				properties = append(properties, prop)
			} else if err != nil {
				return nil, err
			}
		} else {
			p.addError("line %d:%d: unexpected token %s in class %s body", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal, name)
			p.nextToken()
		}
	}

	if p.tok.Type != token.T_RBRACE {
		p.addError("line %d:%d: expected } to close class %s body, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
		return nil, nil
	}
	p.nextToken() // consume }

	return &ast.ClassNode{
		Name:       name,
		Extends:    extends,
		Implements: implements,
		Properties: properties,
		Methods:    methods,
		Pos:        ast.Position(pos),
	}, nil
}

func (p *Parser) parsePropertyDeclaration() (ast.Node, error) {
	pos := p.tok.Pos
	name := p.tok.Literal[1:] // Remove $ prefix
	p.nextToken()

	if p.tok.Type != token.T_SEMICOLON {
		p.addError("line %d:%d: expected ; after property declaration $%s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
		return nil, nil
	}
	p.nextToken()

	return &ast.PropertyNode{
		Name: name,
		Pos:  ast.Position(pos),
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
		className := p.tok.Literal
		pos := p.tok.Pos
		p.nextToken()
		// Class constant fetch: self::CONST, static::CONST, Foo::CONST, Foo::class
		if p.tok.Type == token.T_DOUBLE_COLON {
			p.nextToken() // consume '::'
			if p.tok.Type == token.T_STRING {
				constName := p.tok.Literal
				p.nextToken()
				return &ast.ClassConstFetchNode{
					Class: className,
					Const: constName,
					Pos:   ast.Position(pos),
				}
			} else if p.tok.Type == token.T_CLASS_CONST {
				// Support Foo::class
				p.nextToken()
				return &ast.ClassConstFetchNode{
					Class: className,
					Const: "class",
					Pos:   ast.Position(pos),
				}
			} else {
				p.addError("line %d:%d: expected constant name or 'class' after '::', got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
				return nil
			}
		}
		// Check for function call: identifier followed by '('
		if p.tok.Type == token.T_LPAREN {
			p.nextToken() // consume '('
			var args []ast.Node
			for p.tok.Type != token.T_RPAREN && p.tok.Type != token.T_EOF {
				isUnpacked := false
				if p.tok.Type == token.T_ELLIPSIS {
					isUnpacked = true
					p.nextToken() // consume ...
				}
				// Parse a full expression as argument
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
				// Only break if next token is ')' (end of arguments)
				if p.tok.Type == token.T_COMMA {
					p.nextToken()
					continue
				} else if p.tok.Type == token.T_RPAREN {
					break
				} else if p.tok.Type == token.T_EOF {
					break
				} else {
					// If not a comma or parenthesis, it's likely a parse error, but allow parseExpression to consume as much as possible
					// and rely on error handling
					continue
				}
			}
			if p.tok.Type != token.T_RPAREN {
				p.addError("line %d:%d: expected ) after arguments for function call %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, className, p.tok.Literal)
				return nil
			}
			p.nextToken() // consume )
			return &ast.FunctionCallNode{
				Name: className,
				Args: args,
				Pos:  ast.Position(pos),
			}
		}
		// Not a function call, just an identifier
		return &ast.IdentifierNode{
			Value: className,
			Pos:   ast.Position(pos),
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
		node := &ast.VariableNode{
			Name: p.tok.Literal[1:], // Remove $ prefix
			Pos:  ast.Position(p.tok.Pos),
		}
		p.nextToken()
		return node
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

// UnpackedArgumentNode and FunctionCallNode should be defined in ast package if not already present.

// parseEnum parses an enum declaration
func (p *Parser) parseEnum() (*ast.EnumNode, error) {
	pos := p.tok.Pos
	p.nextToken() // consume "enum"

	// Get enum name
	if p.tok.Type != token.T_STRING {
		return nil, fmt.Errorf("expected enum name, got %s", p.tok.Type)
	}
	name := p.tok.Literal
	p.nextToken()

	// Check for backed enum type
	var backedBy string
	if p.tok.Type == token.T_COLON {
		p.nextToken() // consume ":"
		if p.tok.Type != token.T_STRING {
			return nil, fmt.Errorf("expected enum backing type, got %s", p.tok.Type)
		}
		backedBy = p.tok.Literal
		p.nextToken()
	}

	// Expect opening brace
	if p.tok.Type != token.T_LBRACE {
		return nil, fmt.Errorf("expected {, got %s", p.tok.Type)
	}
	p.nextToken()

	// Parse cases
	var cases []*ast.EnumCaseNode
	for p.tok.Type != token.T_RBRACE {
		if p.tok.Type == token.T_CASE {
			enumCase, err := p.parseEnumCase()
			if err != nil {
				return nil, err
			}
			cases = append(cases, enumCase)
		} else {
			p.nextToken()
		}
	}

	// Consume closing brace
	p.nextToken()

	return &ast.EnumNode{
		Name:     name,
		BackedBy: backedBy,
		Cases:    cases,
		Pos:      ast.Position(pos),
	}, nil
}

// parseEnumCase parses a single enum case
func (p *Parser) parseEnumCase() (*ast.EnumCaseNode, error) {
	pos := p.tok.Pos
	p.nextToken() // consume "case"

	// Get case name
	if p.tok.Type != token.T_STRING {
		return nil, fmt.Errorf("expected case name, got %s", p.tok.Type)
	}
	name := p.tok.Literal
	p.nextToken()

	// Check for value (for backed enums)
	var value ast.Node
	if p.tok.Type == token.T_ASSIGN {
		p.nextToken() // consume "="
		value = p.parseExpression()
		if value == nil {
			return nil, fmt.Errorf("expected value after = in enum case")
		}
	}

	// Expect semicolon
	if p.tok.Type != token.T_SEMICOLON {
		return nil, fmt.Errorf("expected ;, got %s", p.tok.Type)
	}
	p.nextToken()

	return &ast.EnumCaseNode{
		Name:  name,
		Value: value,
		Pos:   ast.Position(pos),
	}, nil
}
