package syntax

import (
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/token"
)

func lowerTrait(n *RedNode, file *File) *ast.TraitNode {
	if n == nil {
		return nil
	}
	pos, end := nodePos(file, n)
	var name string
	var namePos, nameEnd ast.Position
	var members *RedNode
	for _, c := range n.Children() {
		switch c.Kind() {
		case KindUnqualifiedName, KindQualifiedName,
			KindFullyQualifiedName, KindRelativeName:
			if name == "" {
				name = unqualifiedTail(NameText(c))
				namePos, nameEnd = nodePos(file, c)
			}
		case KindMemberList:
			members = c
		}
	}
	if name == "" {
		return nil
	}
	tr := &ast.TraitNode{
		Name:   &ast.Identifier{Name: name, Pos: namePos, EndPos: nameEnd},
		Pos:    pos,
		EndPos: end,
	}
	if members != nil {
		tr.Body = lowerTraitMembers(members, file)
	}
	return tr
}

// lowerTraitMembers lowers trait body members into a single Body slice in
// source order (methods, properties, constants, trait uses) like classic.
func lowerTraitMembers(members *RedNode, file *File) []ast.Node {
	var body []ast.Node
	var pendingAttrs []ast.Node
	for _, m := range members.Children() {
		switch m.Kind() {
		case KindAttributeList:
			pendingAttrs = append(pendingAttrs, lowerAttributeList(m, file)...)
		case KindFunctionDecl, KindMethodDecl:
			if fn := lowerFunction(m, file); fn != nil {
				if len(pendingAttrs) > 0 {
					fn.Attributes = pendingAttrs
					pendingAttrs = nil
				}
				body = append(body, fn)
			}
		case KindPropertyDecl:
			props := lowerProperties(m, file)
			if len(pendingAttrs) > 0 {
				for _, node := range props {
					if prop, ok := node.(*ast.PropertyNode); ok {
						prop.Attributes = pendingAttrs
					}
				}
				pendingAttrs = nil
			}
			body = append(body, props...)
		case KindClassConstDecl:
			consts := lowerClassConsts(m, file)
			if len(pendingAttrs) > 0 {
				for _, node := range consts {
					if c, ok := node.(*ast.ConstantNode); ok {
						c.Attributes = pendingAttrs
					}
				}
				pendingAttrs = nil
			}
			body = append(body, consts...)
		case KindUseTraitClause:
			pendingAttrs = nil
			if tu := lowerUseTraitClause(m, file); tu != nil {
				body = append(body, tu)
			}
		default:
			pendingAttrs = nil
		}
	}
	return body
}

func lowerUseTraitClause(n *RedNode, file *File) *ast.TraitUseNode {
	if n == nil {
		return nil
	}
	pos, end := nodePos(file, n)
	var traits []string
	var adaptations []ast.TraitAdaptation
	for _, c := range n.Children() {
		switch {
		case isNameKind(c.Kind()):
			traits = append(traits, NameText(c))
		case c.Kind() == KindTraitAdaptationList:
			adaptations = lowerTraitAdaptations(c)
		}
	}
	if len(traits) == 0 {
		return nil
	}
	return &ast.TraitUseNode{
		Traits:      traits,
		Adaptations: adaptations,
		Pos:         pos,
		EndPos:      end,
	}
}

func lowerTraitAdaptations(list *RedNode) []ast.TraitAdaptation {
	if list == nil {
		return nil
	}
	var out []ast.TraitAdaptation
	for _, c := range list.Children() {
		if c.Kind() != KindTraitAdaptation {
			continue
		}
		if a, ok := lowerTraitAdaptation(c); ok {
			out = append(out, a)
		}
	}
	return out
}

func lowerTraitAdaptation(n *RedNode) (ast.TraitAdaptation, bool) {
	var a ast.TraitAdaptation
	children := n.Children()
	i := 0
	// Optional Trait::Method or bare Method.
	if i < len(children) && isNameKind(children[i].Kind()) {
		first := NameText(children[i])
		i++
		if i < len(children) && isTokenType(children[i], token.T_DOUBLE_COLON) {
			a.Trait = first
			i++
			if i < len(children) && isNameKind(children[i].Kind()) {
				a.Method = NameText(children[i])
				i++
			}
		} else {
			a.Method = first
		}
	}
	if a.Method == "" {
		return a, false
	}
	for ; i < len(children); i++ {
		c := children[i]
		switch {
		case isTokenType(c, token.T_AS):
			i++
			for ; i < len(children); i++ {
				c = children[i]
				if c.Kind() == KindModifierList {
					a.Modifiers = lowerModifiersFromList(c)
					continue
				}
				if isNameKind(c.Kind()) {
					a.As = NameText(c)
					continue
				}
				if isTokenType(c, token.T_SEMICOLON) {
					break
				}
				// Visibility as T_STRING under ModifierList already handled;
				// bare T_STRING visibility after AS (rare) via ModifierList only.
			}
			return a, true
		case isTokenType(c, token.T_INSTEADOF):
			i++
			for ; i < len(children); i++ {
				c = children[i]
				if isNameKind(c.Kind()) {
					a.InsteadOf = append(a.InsteadOf, NameText(c))
					continue
				}
				if isTokenType(c, token.T_COMMA) {
					continue
				}
				if isTokenType(c, token.T_SEMICOLON) {
					break
				}
			}
			return a, true
		}
	}
	return a, a.Method != ""
}

func lowerModifiersFromList(list *RedNode) ast.ModifierList {
	if list == nil {
		return nil
	}
	var out ast.ModifierList
	for _, c := range list.Children() {
		if c.Green == nil || !c.Green.IsToken() {
			continue
		}
		tok, ok := c.Green.Token()
		if !ok {
			continue
		}
		switch tok.Type {
		case token.T_PUBLIC, token.T_PROTECTED, token.T_PRIVATE:
			out = append(out, ast.Modifier{Tok: tok.Type, Text: tok.Literal})
		case token.T_STRING:
			lit := strings.ToLower(tok.Literal)
			switch lit {
			case "public", "protected", "private":
				out = append(out, ast.ModifierFromText(tok.Literal))
			}
		}
	}
	return out
}
