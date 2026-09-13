package parser

import (
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/token"
)

func (p *Parser) parseUseDeclaration() (ast.Node, error) {
	pos := p.tok.Pos
	p.nextToken() // consume use

	useType := "class"
	if p.tok.Type == token.T_FUNCTION {
		useType = "function"
		p.nextToken()
	} else if p.tok.Type == token.T_CONST {
		useType = "const"
		p.nextToken()
	}

	var uses []ast.Node
	for {
		path := p.parseQualifiedName(useType == "const")
		if path == "" {
			p.addError("line %d:%d: expected imported symbol after use, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
			return nil, nil
		}

		// Group-use: use Foo\Bar\{A, B as C};
		if p.tok.Type == token.T_BACKSLASH || p.tok.Type == token.T_NS_SEPARATOR {
			if p.peekToken().Type == token.T_LBRACE {
				p.nextToken() // consume trailing \
			}
		}
		if p.tok.Type == token.T_LBRACE {
			prefix := strings.TrimSuffix(path, `\`)
			groupUses := p.parseUseGroup(prefix, useType, pos)
			uses = append(uses, groupUses...)
		} else {
			alias := defaultUseAlias(path)
			if p.tok.Type == token.T_AS {
				p.nextToken()
				if p.tok.Type != token.T_STRING {
					p.addError("line %d:%d: expected alias after 'as', got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
					return nil, nil
				}
				alias = p.tok.Literal
				p.nextToken()
			}
			uses = append(uses, &ast.UseNode{
				Path:  path,
				Alias: alias,
				Type:  useType,
				Pos:   ast.Position(pos),
			})
		}

		if p.tok.Type != token.T_COMMA {
			break
		}
		p.nextToken() // consume ','
	}

	if p.tok.Type != token.T_SEMICOLON {
		p.addError("line %d:%d: expected ; after use declaration, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil, nil
	}
	p.nextToken()

	if len(uses) == 1 {
		return uses[0], nil
	}
	return &ast.BlockNode{Statements: uses, Pos: ast.Position(pos)}, nil
}

func (p *Parser) parseUseGroup(prefix, useType string, pos token.Position) []ast.Node {
	p.nextToken() // consume '{'
	var uses []ast.Node
	for p.tok.Type != token.T_RBRACE && p.tok.Type != token.T_EOF {
		itemType := useType
		if p.tok.Type == token.T_FUNCTION {
			itemType = "function"
			p.nextToken()
		} else if p.tok.Type == token.T_CONST {
			itemType = "const"
			p.nextToken()
		}
		name := p.parseQualifiedName(itemType == "const")
		if name == "" {
			p.addError("line %d:%d: expected name in use group, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
			break
		}
		path := strings.Trim(prefix, `\`) + `\` + strings.Trim(name, `\`)
		alias := defaultUseAlias(path)
		if p.tok.Type == token.T_AS {
			p.nextToken()
			if p.tok.Type == token.T_STRING {
				alias = p.tok.Literal
				p.nextToken()
			}
		}
		uses = append(uses, &ast.UseNode{
			Path:  path,
			Alias: alias,
			Type:  itemType,
			Pos:   ast.Position(pos),
		})
		if p.tok.Type == token.T_COMMA {
			p.nextToken()
			continue
		}
		break
	}
	if p.tok.Type != token.T_RBRACE {
		p.addError("line %d:%d: expected } to close use group, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return uses
	}
	p.nextToken() // consume '}'
	return uses
}

func (p *Parser) parseQualifiedName(allowConstantLiteral bool) string {
	if isUseNameToken(p.tok.Type, allowConstantLiteral) &&
		p.peekToken().Type != token.T_NS_SEPARATOR && p.peekToken().Literal != "\\" &&
		p.peekToken().Type != token.T_BACKSLASH {
		name := p.tok.Literal
		p.nextToken()
		return name
	}

	p.nameBuf.Reset()
	if p.tok.Type == token.T_NS_SEPARATOR || p.tok.Literal == "\\" || p.tok.Type == token.T_BACKSLASH {
		p.nameBuf.WriteString("\\")
		p.nextToken()
	}

	for {
		if !isUseNameToken(p.tok.Type, allowConstantLiteral) {
			break
		}
		if isConstantLiteralToken(p.tok.Type) &&
			(p.peekToken().Type == token.T_NS_SEPARATOR || p.peekToken().Literal == "\\" || p.peekToken().Type == token.T_BACKSLASH) {
			break
		}
		p.nameBuf.WriteString(p.tok.Literal)
		p.nextToken()
		if p.tok.Type == token.T_NS_SEPARATOR || p.tok.Literal == "\\" {
			// Stop before group-use `\{` so the caller can parse the brace list.
			if p.peekToken().Type == token.T_LBRACE {
				break
			}
			p.nameBuf.WriteString("\\")
			p.nextToken()
			continue
		}
		if p.tok.Type == token.T_BACKSLASH {
			if p.peekToken().Type == token.T_LBRACE {
				break
			}
			p.nameBuf.WriteString("\\")
			p.nextToken()
			continue
		}
		break
	}

	name := p.nameBuf.String()
	return strings.TrimSuffix(name, "\\")
}

func isUseNameToken(tokenType token.TokenType, allowConstantLiteral bool) bool {
	if tokenType == token.T_STRING || tokenType == token.T_STATIC || tokenType == token.T_SELF || tokenType == token.T_PARENT {
		return true
	}
	return allowConstantLiteral && isConstantLiteralToken(tokenType)
}

func isConstantLiteralToken(tokenType token.TokenType) bool {
	return tokenType == token.T_TRUE || tokenType == token.T_FALSE || tokenType == token.T_NULL
}

func defaultUseAlias(path string) string {
	path = strings.TrimPrefix(path, "\\")
	if path == "" {
		return ""
	}
	if idx := strings.LastIndex(path, "\\"); idx >= 0 {
		return path[idx+1:]
	}
	return path
}
