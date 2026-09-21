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
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		c := n.bindChild(green, offset)
		switch green.Kind() {
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
			if isGreenTokenType(green, token.T_COLON) {
				seenColon = true
				headerEnd = spanEnd(file, c.Span())
				return true
			}
			if seenColon && isTypeKind(green.Kind()) {
				en.BackedBy = TypeText(c)
				headerEnd = spanEnd(file, c.Span())
				seenColon = false
			}
		}
		return true
	})
	en.HeaderEndPos = headerEnd
	if members != nil {
		var pendingAttrs []ast.Node
		memberKinds := func(k Kind) bool {
			switch k {
			case KindAttributeList, KindEnumCase, KindFunctionDecl, KindMethodDecl:
				return true
			default:
				return false
			}
		}
		members.ForEachChildDesc(func(green *GreenNode, offset int) bool {
			k := green.Kind()
			if !memberKinds(k) {
				pendingAttrs = nil
				return true
			}
			m := members.bindChild(green, offset)
			switch k {
			case KindAttributeList:
				pendingAttrs = append(pendingAttrs, lowerAttributeList(m, file)...)
			case KindEnumCase:
				// Classic clears PHPDoc before cases — do not attach leading
				// T_DOC_COMMENT trivia on KindEnumCase to the case node.
				pendingAttrs = nil
				if ec := lowerEnumCase(m, file); ec != nil {
					en.Cases = append(en.Cases, ec)
				}
			case KindFunctionDecl, KindMethodDecl:
				if fn := lowerFunction(m, file); fn != nil {
					if len(pendingAttrs) > 0 {
						fn.Attributes = pendingAttrs
						pendingAttrs = nil
					}
					en.Methods = append(en.Methods, fn)
				}
			default:
				pendingAttrs = nil
			}
			return true
		})
	}
	return en
}

func lowerEnumCase(n *RedNode, file *File) *ast.EnumCaseNode {
	if n == nil {
		return nil
	}
	pos, end := nodePos(file, n)
	ec := &ast.EnumCaseNode{Pos: pos, EndPos: end}
	seenAssign := false
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		c := n.bindChild(green, offset)
		if isGreenTokenType(green, token.T_CASE) {
			pos, _ = nodePos(file, c)
			ec.Pos = pos
			return true
		}
		if isNameKind(green.Kind()) && ec.Name == "" {
			ec.Name = NameText(c)
			return true
		}
		if isGreenTokenType(green, token.T_ASSIGN) {
			seenAssign = true
			return true
		}
		if seenAssign && (isExprKind(green.Kind()) || isNameKind(green.Kind())) {
			ec.Value = lowerExpr(c, file)
			seenAssign = false
		}
		return true
	})
	if ec.Name == "" {
		return nil
	}
	return ec
}
