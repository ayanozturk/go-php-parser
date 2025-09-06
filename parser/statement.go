package parser

import (
	"fmt"
	"go-phpcs/ast"
	"go-phpcs/token"
)

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
retry:
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
	case token.T_DOC_COMMENT:
		// Store PHPDoc comment for next node, don't return it as a separate statement
		p.currentDoc = p.tok.Literal
		p.nextToken() // consume doc comment
		// Continue parsing with the current token
		goto retry
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
	case token.T_THROW:
		pos := p.tok.Pos
		p.nextToken() // consume throw
		expr := p.parseExpression()
		if expr == nil {
			return nil, nil
		}
		if p.tok.Type != token.T_SEMICOLON {
			p.addError("line %d:%d: expected ; after throw statement, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
			return nil, nil
		}
		p.nextToken() // consume ;
		return &ast.ThrowNode{
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
	case token.T_FOR:
		return p.parseForStatement()
	case token.T_UNSET:
		pos := p.tok.Pos
		p.nextToken() // consume 'unset'
		if p.tok.Type != token.T_LPAREN {
			p.addError("line %d:%d: expected ( after unset, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
			return nil, nil
		}
		p.nextToken() // consume '('
		var args []ast.Node
		for {
			arg := p.parseExpression()
			if arg != nil {
				args = append(args, arg)
			}
			if p.tok.Type == token.T_COMMA {
				p.nextToken()
				continue
			}
			break
		}
		if p.tok.Type != token.T_RPAREN {
			p.addError("line %d:%d: expected ) after unset arguments, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
			return nil, nil
		}
		p.nextToken() // consume ')'
		if p.tok.Type != token.T_SEMICOLON {
			p.addError("line %d:%d: expected ; after unset statement, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
			return nil, nil
		}
		p.nextToken() // consume ;
		return &ast.ExpressionStmt{
			Expr: &ast.FunctionCallNode{
				Name: &ast.IdentifierNode{Value: "unset", Pos: ast.Position(pos)},
				Args: args,
				Pos:  ast.Position(pos),
			},
			Pos: ast.Position(pos),
		}, nil
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
