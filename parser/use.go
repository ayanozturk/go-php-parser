package parser

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/token"
	"strings"
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
		path := p.parseQualifiedName()
		if path == "" {
			p.addError("line %d:%d: expected imported symbol after use, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
			return nil, nil
		}

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

func (p *Parser) parseQualifiedName() string {
	if (p.tok.Type == token.T_STRING || p.tok.Type == token.T_STATIC || p.tok.Type == token.T_SELF || p.tok.Type == token.T_PARENT) &&
		p.peekToken().Type != token.T_NS_SEPARATOR && p.peekToken().Literal != "\\" {
		name := p.tok.Literal
		p.nextToken()
		return name
	}

	p.nameBuf.Reset()
	if p.tok.Type == token.T_NS_SEPARATOR || p.tok.Literal == "\\" {
		p.nameBuf.WriteString("\\")
		p.nextToken()
	}

	for {
		if p.tok.Type != token.T_STRING && p.tok.Type != token.T_STATIC && p.tok.Type != token.T_SELF && p.tok.Type != token.T_PARENT {
			break
		}
		p.nameBuf.WriteString(p.tok.Literal)
		p.nextToken()
		if p.tok.Type == token.T_NS_SEPARATOR || p.tok.Literal == "\\" {
			p.nameBuf.WriteString("\\")
			p.nextToken()
			continue
		}
		break
	}

	name := p.nameBuf.String()
	return strings.TrimSuffix(name, "\\")
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
