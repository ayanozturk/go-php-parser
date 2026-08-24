package parser

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/token"
	"strings"
)

// parseInterfaceDeclaration parses a PHP interface declaration
func (p *Parser) parseInterfaceDeclaration() ast.Node {
	pos := p.tok.Pos
	phpdoc := p.consumeCurrentDoc(pos)
	p.nextToken() // consume 'interface'

	if p.tok.Type != token.T_STRING {
		p.addError("line %d:%d: expected interface name, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		return nil
	}

	name := p.tok.Literal
	p.nextToken()

	// Parse optional 'extends' clause
	var extends []string
	if p.tok.Type == token.T_EXTENDS {
		p.nextToken() // consume 'extends'
		for {
			fqcn := ""
			// Loop to collect full FQCN: (T_NS_SEPARATOR T_STRING)+
			for {
				if p.tok.Type == token.T_NS_SEPARATOR || p.tok.Literal == "\\" {
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
				p.addError("line %d:%d: expected interface name after extends, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
				return nil
			}
			extends = append(extends, fqcn)
			if p.tok.Type != token.T_COMMA {
				break
			}
			p.nextToken() // consume ','
		}
	}

	// Skip comments and whitespace before opening brace
	for p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT || p.tok.Type == token.T_WHITESPACE {
		p.nextToken()
	}
	if p.tok.Type != token.T_LBRACE {
		p.addError("line %d:%d: expected { after interface name %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
		return nil
	}
	p.nextToken() // consume {

	var members []ast.Node
	for p.tok.Type != token.T_RBRACE && p.tok.Type != token.T_EOF {
		// Skip doc comments, regular comments, and attributes in interface body
		if p.tok.Type == token.T_DOC_COMMENT || p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_ATTRIBUTE {
			p.nextToken()
			continue
		}
		// Interface members: methods and constants
		if p.tok.Type == token.T_PUBLIC || p.tok.Type == token.T_PRIVATE || p.tok.Type == token.T_PROTECTED {
			visibility := p.tok.Literal
			p.nextToken()
			// Skip any number of 'static', comments, and whitespace
			for {
				if p.tok.Type == token.T_STATIC || p.tok.Type == token.T_DOC_COMMENT || p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_WHITESPACE {
					p.nextToken()
				} else {
					break
				}
			}
			if p.tok.Type == token.T_CONST {
				for _, constant := range p.parseConstantWithModifiers([]string{visibility}) {
					members = append(members, constant)
				}
			} else if p.tok.Type == token.T_FUNCTION {
				if method := p.parseInterfaceMethodWithVisibility(visibility); method != nil {
					members = append(members, method)
				}
			} else if isInterfacePropertyTypeStart(p.tok.Type) {
				if prop := p.parseInterfaceProperty(visibility); prop != nil {
					members = append(members, prop)
				}
			} else {
				p.addError("line %d:%d: unexpected token %s after visibility modifier in interface %s body", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal, name)
				p.nextToken()
			}
		} else if p.tok.Type == token.T_FUNCTION {
			if method := p.parseInterfaceMethod(); method != nil {
				members = append(members, method)
			}
		} else if p.tok.Type == token.T_CONST {
			for _, constant := range p.parseConstant() {
				members = append(members, constant)
			}
		} else {
			p.addError("line %d:%d: unexpected token %s in interface %s body", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal, name)
			p.nextToken()
		}
	}

	if p.tok.Type != token.T_RBRACE {
		p.addError("line %d:%d: expected } to close interface %s body, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
		return nil
	}
	p.nextToken() // consume }

	return &ast.InterfaceNode{
		Name:    name,
		Extends: extends,
		Members: members,
		PHPDoc:  phpdoc,
		Pos:     ast.Position(pos),
	}
}

// parseInterfaceMethod parses a method declaration in an interface
func (p *Parser) parseInterfaceMethod() ast.Node {
	return p.parseInterfaceMethodWithVisibility("")
}

// isInterfacePropertyTypeStart reports whether tokenType can start a
// property type hint for a PHP 8.4 interface property hook declaration,
// e.g. "public string $foo { get; }".
func isInterfacePropertyTypeStart(tokenType token.TokenType) bool {
	switch tokenType {
	case token.T_STRING, token.T_NS_SEPARATOR, token.T_CALLABLE, token.T_ARRAY, token.T_MIXED,
		token.T_QUESTION, token.T_TRUE, token.T_FALSE, token.T_NULL, token.T_STATIC, token.T_LPAREN,
		token.T_VARIABLE:
		return true
	default:
		return false
	}
}

// parseInterfaceProperty parses a PHP 8.4 property-hook-only property
// declaration in an interface body, e.g. "public string $foo { get; }".
// Interface properties may not have a default value or a hook body/expr,
// only bare hook declarations like "{ get; }" or "{ get; set; }".
func (p *Parser) parseInterfaceProperty(visibility string) ast.Node {
	pos := p.tok.Pos
	var typeHint string
	if p.tok.Type != token.T_VARIABLE {
		if p.tok.Type == token.T_LPAREN {
			typeHint = parseFullTypeHint(p)
		} else {
			typeHint = p.parseTypeHint()
		}
		p.skipCommentsAndWhitespace()
	}
	if p.tok.Type != token.T_VARIABLE {
		p.addError("line %d:%d: expected property name in interface, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		p.nextToken()
		return nil
	}
	name := p.tok.Literal[1:]
	p.nextToken()

	var hooks []ast.PropertyHookNode
	if p.tok.Type == token.T_LBRACE {
		hooks = p.parsePropertyHooks(name)
	}
	if p.tok.Type == token.T_SEMICOLON {
		p.nextToken()
	}
	return &ast.PropertyNode{
		Name:       name,
		TypeHint:   typeHint,
		Visibility: visibility,
		Hooks:      hooks,
		Pos:        ast.Position(pos),
	}
}

func (p *Parser) parseInterfaceMethodWithVisibility(initialVisibility string) ast.Node {
	pos := p.tok.Pos

	// Skip doc comments and regular comments before method signature
	for p.tok.Type == token.T_DOC_COMMENT || p.tok.Type == token.T_COMMENT {
		if p.tok.Type == token.T_DOC_COMMENT {
			p.currentDoc = p.tok.Literal
		}
		p.nextToken()
	}

	// Parse visibility modifier if present
	visibility := initialVisibility
	if p.tok.Type == token.T_PUBLIC || p.tok.Type == token.T_PRIVATE || p.tok.Type == token.T_PROTECTED {
		visibility = p.tok.Literal
		p.nextToken()
	}

	// Parse function keyword
	if p.tok.Type != token.T_FUNCTION {
		p.addError("line %d:%d: expected 'function' keyword, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		p.syncToNextClassMember()
		return nil
	}
	p.nextToken()

	// Optional by-reference return marker: "function &name(...)"
	if p.tok.Type == token.T_AMPERSAND {
		p.nextToken()
	}

	// Accept PHP keywords as method names (not just T_STRING)
	if !isValidMethodNameToken(p.tok.Type) {
		p.addError("line %d:%d: expected method name, got %s", p.tok.Pos.Line, p.tok.Pos.Column, p.tok.Literal)
		p.syncToNextClassMember()
		return nil
	}
	name := p.tok.Literal
	p.nextToken()

	// Parse opening parenthesis
	if p.tok.Type != token.T_LPAREN {
		p.addError("line %d:%d: expected '(' after method name %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
		p.syncToNextClassMember()
		return nil
	}
	p.nextToken()

	// Parse parameters
	var params []ast.Node
	for p.tok.Type != token.T_RPAREN && p.tok.Type != token.T_EOF {
		// Skip comments and commas before each parameter
		for p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT || p.tok.Type == token.T_COMMA {
			p.nextToken()
		}
		if p.tok.Type == token.T_RPAREN || p.tok.Type == token.T_EOF {
			break
		}
		param := p.parseParameter()
		if param != nil {
			params = append(params, param)
		}
		// After a parameter, skip any comments and commas before checking for next parameter or end
		for p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT || p.tok.Type == token.T_COMMA {
			p.nextToken()
		}
		if p.tok.Type == token.T_RPAREN {
			break
		}
		if p.tok.Type == token.T_EOF {
			break
		}
	}
	if p.tok.Type != token.T_RPAREN {
		p.addError("line %d:%d: expected ')' after parameter list for method %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
		p.syncToNextClassMember()
		return nil
	}
	p.nextToken() // consume )

	// Skip comments after parameter list (e.g., /* : self */)
	for p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT {
		p.nextToken()
	}

	// Parse return type if present
	var returnType ast.Node
	if p.tok.Type == token.T_COLON {
		p.nextToken() // consume :
		typePos := p.tok.Pos
		typeStr := p.parseTypeHint()
		if typeStr != "" {
			if strings.Contains(typeStr, "|") {
				parts := strings.Split(typeStr, "|")
				for i := range parts {
					parts[i] = strings.TrimSpace(parts[i])
				}
				returnType = &ast.UnionTypeNode{
					Types: parts,
					Pos:   ast.Position(typePos),
				}
			} else if strings.Contains(typeStr, "&") {
				parts := strings.Split(typeStr, "&")
				for i := range parts {
					parts[i] = strings.TrimSpace(parts[i])
				}
				returnType = &ast.IntersectionTypeNode{
					Types: parts,
					Pos:   ast.Position(typePos),
				}
			} else {
				returnType = &ast.IdentifierNode{
					Value: typeStr,
					Pos:   ast.Position(typePos),
				}
			}
		} else {
			p.addError("line %d:%d: expected return type for method %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
			p.syncToNextClassMember()
			return nil
		}
	}

	// Skip comments before semicolon (defensive, in case comments appear here)
	for p.tok.Type == token.T_COMMENT || p.tok.Type == token.T_DOC_COMMENT {
		p.nextToken()
	}

	// Parse semicolon
	if p.tok.Type != token.T_SEMICOLON {
		p.addError("line %d:%d: expected ';' after method declaration %s, got %s", p.tok.Pos.Line, p.tok.Pos.Column, name, p.tok.Literal)
		p.syncToNextClassMember()
		return nil
	}
	p.nextToken()

	return &ast.InterfaceMethodNode{
		Name:       name,
		Visibility: visibility,
		ReturnType: returnType,
		Params:     params,
		PHPDoc:     p.consumeCurrentDoc(pos),
		Pos:        ast.Position(pos),
	}
}
