package syntax

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/token"
)

func lowerEnum(n *RedNode, file *File) *ast.EnumNode {
	if n == nil {
		return nil
	}
	pos, end := nodePos(file, n)
	en := &ast.EnumNode{Pos: pos, EndPos: end}
	var members *RedNode
	headerEnd := pos
	seenColon := false
	for _, c := range n.Children() {
		switch c.Kind() {
		case KindUnqualifiedName, KindQualifiedName,
			KindFullyQualifiedName, KindRelativeName:
			if en.Name == "" {
				en.Name = unqualifiedTail(NameText(c))
				headerEnd = spanEnd(file, c.Span())
			}
		case KindImplementsClause:
			en.Implements = clauseNames(c)
			headerEnd = spanEnd(file, c.Span())
		case KindMemberList:
			members = c
		default:
			if isTokenType(c, token.T_COLON) {
				seenColon = true
				headerEnd = spanEnd(file, c.Span())
				continue
			}
			if seenColon && isTypeKind(c.Kind()) {
				en.BackedBy = TypeText(c)
				headerEnd = spanEnd(file, c.Span())
				seenColon = false
			}
		}
	}
	en.HeaderEndPos = headerEnd
	if members != nil {
		for _, m := range members.Children() {
			switch m.Kind() {
			case KindEnumCase:
				// Classic clears PHPDoc before cases — do not attach leading
				// T_DOC_COMMENT trivia on KindEnumCase to the case node.
				if ec := lowerEnumCase(m, file); ec != nil {
					en.Cases = append(en.Cases, ec)
				}
			case KindFunctionDecl, KindMethodDecl:
				if fn := lowerFunction(m, file); fn != nil {
					en.Methods = append(en.Methods, fn)
				}
			}
		}
	}
	return en
}

func lowerEnumCase(n *RedNode, file *File) *ast.EnumCaseNode {
	if n == nil {
		return nil
	}
	pos, end := nodePos(file, n)
	ec := &ast.EnumCaseNode{Pos: pos, EndPos: end}
	for _, c := range n.Children() {
		if isNameKind(c.Kind()) && ec.Name == "" {
			ec.Name = NameText(c)
		}
		// Value expressions stay nil in index mode (no expr lowering).
	}
	if ec.Name == "" {
		return nil
	}
	return ec
}
