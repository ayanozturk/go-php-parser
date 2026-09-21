package syntax

import (
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/token"
)

func isGreenTokenType(g *GreenNode, tt token.TokenType) bool {
	return g != nil && g.IsToken() && g.TokenType() == tt
}

func greenTokenLiteral(g *GreenNode) string {
	if g == nil || !g.IsToken() {
		return ""
	}
	tok, ok := g.Token()
	if !ok {
		return ""
	}
	return tok.Literal
}

func firstGreenTokenLiteral(green *GreenNode, offset int) string {
	var lit string
	forEachChildDescGreen(green, offset, func(g *GreenNode, off int) bool {
		if g.IsToken() {
			lit = greenTokenLiteral(g)
			return false
		}
		return true
	})
	return lit
}

func lowerModifiers(n *RedNode) ast.ModifierList {
	if n == nil {
		return nil
	}
	list := n.FirstChildOfKind(KindModifierList)
	if list == nil {
		return nil
	}
	var out ast.ModifierList
	var parts []struct {
		green  *GreenNode
		offset int
	}
	list.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		parts = append(parts, struct {
			green  *GreenNode
			offset int
		}{green, offset})
		return true
	})
	for i := 0; i < len(parts); i++ {
		g := parts[i].green
		if !g.IsToken() {
			continue
		}
		tok, ok := g.Token()
		if !ok {
			continue
		}
		switch tok.Type {
		case token.T_PUBLIC, token.T_PROTECTED, token.T_PRIVATE,
			token.T_STATIC, token.T_FINAL, token.T_ABSTRACT, token.T_READONLY:
			m := ast.Modifier{Tok: tok.Type, Text: tok.Literal}
			// Asymmetric visibility: public(set)
			if (tok.Type == token.T_PUBLIC || tok.Type == token.T_PROTECTED || tok.Type == token.T_PRIVATE) &&
				i+3 < len(parts) &&
				isGreenTokenType(parts[i+1].green, token.T_LPAREN) &&
				tokenLiteral(list.bindChild(parts[i+2].green, parts[i+2].offset)) == "set" &&
				isGreenTokenType(parts[i+3].green, token.T_RPAREN) {
				m.Set = true
				i += 3
			}
			out = append(out, m)
		}
	}
	return out
}

func isTokenType(n *RedNode, tt token.TokenType) bool {
	if n == nil || n.Green == nil || !n.Green.IsToken() {
		return false
	}
	tok, ok := n.Green.Token()
	return ok && tok.Type == tt
}

func tokenLiteral(n *RedNode) string {
	if n == nil {
		return ""
	}
	if n.Green != nil && n.Green.IsToken() {
		tok, ok := n.Green.Token()
		if ok && tok.Literal != "" {
			return tok.Literal
		}
	}
	return strings.TrimSpace(n.Text())
}

func isNameKind(k Kind) bool {
	switch k {
	case KindUnqualifiedName, KindQualifiedName,
		KindFullyQualifiedName, KindRelativeName, KindName:
		return true
	default:
		return false
	}
}

func nameString(n *RedNode) string {
	if n == nil {
		return ""
	}
	return strings.TrimPrefix(NameText(n), `\`)
}

func nameTextAt(file *File, green *GreenNode, offset int) string {
	if file == nil || green == nil || !isNameKind(green.Kind()) {
		return ""
	}
	var out string
	withPooledRed(file, nil, green, offset, func(n *RedNode) {
		out = NameText(n)
	})
	return out
}

func firstNameChildGreen(n *RedNode) (*GreenNode, int) {
	if n == nil {
		return nil, 0
	}
	var foundGreen *GreenNode
	var foundOff int
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if isNameKind(green.Kind()) {
			foundGreen = green
			foundOff = offset
			return false
		}
		return true
	})
	return foundGreen, foundOff
}

func clauseNames(clause *RedNode) []string {
	if clause == nil {
		return nil
	}
	return clauseNamesGreen(clause.File, clause.Green, clause.Offset)
}

func clauseNamesGreen(file *File, green *GreenNode, offset int) []string {
	if file == nil || green == nil {
		return nil
	}
	var out []string
	forEachChildDescGreen(green, offset, func(g *GreenNode, off int) bool {
		if isNameKind(g.Kind()) {
			out = append(out, nameTextAt(file, g, off))
		}
		return true
	})
	return out
}

func unqualifiedTail(path string) string {
	path = strings.Trim(path, `\`)
	if i := strings.LastIndexByte(path, '\\'); i >= 0 {
		return path[i+1:]
	}
	return path
}

func stripVarDollar(s string) string {
	return strings.TrimPrefix(strings.TrimSpace(s), "$")
}
