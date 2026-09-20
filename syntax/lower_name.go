package syntax

import (
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/token"
)

func lowerModifiers(n *RedNode) ast.ModifierList {
	if n == nil {
		return nil
	}
	list := n.FirstChildOfKind(KindModifierList)
	if list == nil {
		return nil
	}
	var out ast.ModifierList
	var children []*RedNode
	list.ForEachChild(func(c *RedNode) bool {
		children = append(children, c)
		return true
	})
	for i := 0; i < len(children); i++ {
		c := children[i]
		if c.Green == nil || !c.Green.IsToken() {
			continue
		}
		tok, ok := c.Green.Token()
		if !ok {
			continue
		}
		switch tok.Type {
		case token.T_PUBLIC, token.T_PROTECTED, token.T_PRIVATE,
			token.T_STATIC, token.T_FINAL, token.T_ABSTRACT, token.T_READONLY:
			m := ast.Modifier{Tok: tok.Type, Text: tok.Literal}
			// Asymmetric visibility: public(set)
			if (tok.Type == token.T_PUBLIC || tok.Type == token.T_PROTECTED || tok.Type == token.T_PRIVATE) &&
				i+3 < len(children) &&
				isTokenType(children[i+1], token.T_LPAREN) &&
				tokenLiteral(children[i+2]) == "set" &&
				isTokenType(children[i+3], token.T_RPAREN) {
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

func firstNameChild(n *RedNode) *RedNode {
	if n == nil {
		return nil
	}
	var found *RedNode
	n.ForEachChild(func(c *RedNode) bool {
		if isNameKind(c.Kind()) {
			found = c
			return false
		}
		return true
	})
	return found
}

func clauseNames(clause *RedNode) []string {
	if clause == nil {
		return nil
	}
	var out []string
	clause.ForEachChild(func(c *RedNode) bool {
		if isNameKind(c.Kind()) {
			out = append(out, NameText(c))
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
