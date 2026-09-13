package parser

import (
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/token"
)

// parseTypeNode parses a native type hint into nested type/name AST nodes.
// Types are built token-by-token; there is no concatenate-then-reparse path.
func (p *Parser) parseTypeNode() ast.Node {
	return p.parseTypeNodeExpr()
}

// parseFullTypeNode is the parenthesized-capable entry used by params/properties.
func parseFullTypeNode(p *Parser) ast.Node {
	return p.parseTypeNodeExpr()
}

func (p *Parser) parseTypeNodeExpr() ast.Node {
	pos := p.tok.Pos
	if p.tok.Type == token.T_QUESTION {
		p.nextToken()
		inner := p.parseTypeAtomNode()
		if inner == nil {
			return nil
		}
		return &ast.NullableTypeNode{Inner: inner, Pos: ast.Position(pos)}
	}
	first := p.parseTypeAtomNode()
	if first == nil {
		return nil
	}
	if p.tok.Type == token.T_PIPE {
		types := []ast.Node{first}
		for p.tok.Type == token.T_PIPE {
			p.nextToken()
			next := p.parseTypeAtomNode()
			if next == nil {
				p.addError("empty type segment in union type")
				break
			}
			types = append(types, next)
		}
		return &ast.UnionTypeNode{Types: types, Pos: ast.Position(pos)}
	}
	if p.tok.Type == token.T_AMPERSAND {
		types := []ast.Node{first}
		for p.tok.Type == token.T_AMPERSAND {
			// Reference marker before $param is not an intersection.
			if p.peekToken().Type == token.T_VARIABLE {
				break
			}
			p.nextToken()
			next := p.parseTypeAtomNode()
			if next == nil {
				p.addError("empty type segment in intersection type")
				break
			}
			types = append(types, next)
		}
		if len(types) == 1 {
			return first
		}
		return &ast.IntersectionTypeNode{Types: types, Pos: ast.Position(pos)}
	}
	return first
}

func (p *Parser) parseTypeAtomNode() ast.Node {
	pos := p.tok.Pos
	if p.tok.Type == token.T_LPAREN {
		p.nextToken()
		inner := p.parseTypeNodeExpr()
		if p.tok.Type != token.T_RPAREN {
			p.addError("expected ')' after parenthesized type hint, got %s", p.tok.Literal)
		} else {
			p.nextToken()
		}
		return &ast.ParenthesizedTypeNode{Inner: inner, Pos: ast.Position(pos)}
	}
	if p.tok.Type == token.T_CALLABLE {
		return p.parseCallableTypeNode()
	}
	name := p.parseTypeNameNode()
	if name == nil {
		return nil
	}
	// Legacy PHPDoc-style Foo[] suffix still appears in some fixtures.
	if p.tok.Type == token.T_LBRACKET {
		p.nextToken()
		if p.tok.Type != token.T_RBRACKET {
			p.addError("expected ']' after array type in type hint")
		} else {
			p.nextToken()
		}
		if id, ok := name.(*ast.IdentifierNode); ok {
			id.Value += "[]"
		}
	}
	return name
}

func (p *Parser) parseTypeNameNode() ast.Node {
	pos := p.tok.Pos
	var b strings.Builder
	if p.tok.Type == token.T_NS_SEPARATOR || p.tok.Literal == "\\" {
		b.WriteString("\\")
		p.nextToken()
	}
	wrote := false
	for {
		switch p.tok.Type {
		case token.T_STRING, token.T_NEW, token.T_STATIC, token.T_SELF, token.T_PARENT,
			token.T_NEVER, token.T_ARRAY, token.T_NULL, token.T_MIXED,
			token.T_FALSE, token.T_TRUE:
			b.WriteString(p.tok.Literal)
			wrote = true
			p.nextToken()
			if p.tok.Type == token.T_NS_SEPARATOR || p.tok.Literal == "\\" {
				b.WriteString("\\")
				p.nextToken()
				continue
			}
		default:
			if p.tok.Literal == "mixed" {
				b.WriteString(p.tok.Literal)
				wrote = true
				p.nextToken()
			}
		}
		break
	}
	if !wrote {
		return nil
	}
	return &ast.IdentifierNode{Value: b.String(), Pos: ast.Position(pos)}
}

// parseCallableTypeNode builds a CallableTypeNode for callable / callable(...): T.
// Parameter and return types inside the signature are parsed as nested type nodes.
func (p *Parser) parseCallableTypeNode() ast.Node {
	pos := p.tok.Pos
	p.nextToken()
	if p.tok.Type != token.T_LPAREN {
		return &ast.CallableTypeNode{HasSignature: false, Pos: ast.Position(pos)}
	}
	p.nextToken()
	var params []*ast.CallableParamNode
	for p.tok.Type != token.T_RPAREN && p.tok.Type != token.T_EOF {
		for p.tok.Type == token.T_WHITESPACE || p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT {
			p.nextToken()
		}
		if p.tok.Type == token.T_RPAREN {
			break
		}
		if len(params) > 0 {
			if p.tok.Type != token.T_COMMA {
				p.addError("expected ',' between callable parameters, got %s", p.tok.Literal)
				break
			}
			p.nextToken()
			for p.tok.Type == token.T_WHITESPACE || p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT {
				p.nextToken()
			}
		}
		paramPos := p.tok.Pos
		var typeHint ast.Node
		if isNativeTypeStart(p.tok) {
			typeHint = p.parseTypeNodeExpr()
		}
		var name string
		if p.tok.Type == token.T_VARIABLE {
			name = p.tok.Literal[1:]
			p.nextToken()
		} else if typeHint == nil {
			p.addError("expected parameter type or name in callable, got %s", p.tok.Literal)
			break
		}
		params = append(params, &ast.CallableParamNode{
			TypeHint: typeHint,
			Name:     name,
			Pos:      ast.Position(paramPos),
		})
	}
	if p.tok.Type != token.T_RPAREN {
		p.addError("expected ')' to close callable parameter list, got %s", p.tok.Literal)
	} else {
		p.nextToken()
	}
	var returnType ast.Node
	if p.tok.Type == token.T_COLON {
		p.nextToken()
		for p.tok.Type == token.T_WHITESPACE || p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT {
			p.nextToken()
		}
		if isNativeTypeStart(p.tok) {
			returnType = p.parseTypeNodeExpr()
		} else {
			p.addError("expected return type after ':' in callable, got %s", p.tok.Literal)
		}
	}
	return &ast.CallableTypeNode{
		Params:       params,
		HasSignature: true,
		ReturnType:   returnType,
		Pos:          ast.Position(pos),
	}
}

func isNativeTypeStart(tok token.Token) bool {
	switch tok.Type {
	case token.T_QUESTION, token.T_LPAREN, token.T_NS_SEPARATOR, token.T_STRING,
		token.T_CALLABLE, token.T_ARRAY, token.T_NULL, token.T_MIXED, token.T_STATIC,
		token.T_SELF, token.T_PARENT, token.T_NEVER, token.T_FALSE, token.T_TRUE, token.T_NEW:
		return true
	default:
		return tok.Literal == "mixed" || tok.Literal == "\\"
	}
}
