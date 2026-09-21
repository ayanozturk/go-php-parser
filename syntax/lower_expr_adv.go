package syntax

import (
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/token"
)

func lowerClosureExpr(n *RedNode, file *File) ast.Node {
	if n == nil {
		return nil
	}
	pos, end := nodePos(file, n)
	fn := &ast.FunctionNode{
		Name:   "",
		Pos:    pos,
		EndPos: end,
		PHPDoc: leadingDocFromNode(n),
	}
	seenColon := false
	headerEnd := pos
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		switch k {
		case KindParamList:
			c := n.bindChild(green, offset)
			fn.Params = lowerParamList(c, file)
			headerEnd = spanEnd(file, c.Span())
		case KindClosureUseClause:
			c := n.bindChild(green, offset)
			fn.Uses = lowerClosureUses(c, file)
			headerEnd = spanEnd(file, c.Span())
		case KindStatementList:
			c := n.bindChild(green, offset)
			fn.Body = lowerStatements(c, file)
		case KindTokenList:
			fn.Body = nil
		default:
			if k == KindToken {
				switch green.TokenType() {
				case token.T_STATIC:
					fn.Modifiers = append(fn.Modifiers, ast.Modifier{Tok: token.T_STATIC, Text: "static"})
				case token.T_COLON:
					seenColon = true
				}
				return true
			}
			if seenColon && isTypeKind(k) {
				c := n.bindChild(green, offset)
				fn.ReturnType = lowerType(c, file)
				headerEnd = spanEnd(file, c.Span())
				seenColon = false
			}
		}
		return true
	})
	fn.HeaderEndPos = headerEnd
	return fn
}

func lowerClosureUses(n *RedNode, file *File) []ast.ClosureUse {
	if n == nil {
		return nil
	}
	var out []ast.ClosureUse
	byRef := false
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if green.Kind() != KindToken {
			return true
		}
		switch green.TokenType() {
		case token.T_AMPERSAND:
			byRef = true
		case token.T_VARIABLE:
			c := n.bindChild(green, offset)
			p, e := nodePos(file, c)
			out = append(out, ast.ClosureUse{
				Name:   stripVarDollar(tokenLiteral(c)),
				ByRef:  byRef,
				Pos:    p,
				EndPos: e,
			})
			byRef = false
		}
		return true
	})
	return out
}

func lowerArrowFunctionExpr(n *RedNode, file *File) ast.Node {
	if n == nil {
		return nil
	}
	pos, end := nodePos(file, n)
	af := &ast.ArrowFunctionNode{Pos: pos, EndPos: end}
	seenColon, seenArrow := false, false
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		switch k {
		case KindParamList:
			c := n.bindChild(green, offset)
			af.Params = lowerParamList(c, file)
		default:
			if k == KindToken {
				switch green.TokenType() {
				case token.T_COLON:
					seenColon = true
				case token.T_DOUBLE_ARROW:
					seenArrow = true
				}
				return true
			}
			if seenColon && isTypeKind(k) {
				c := n.bindChild(green, offset)
				af.ReturnType = lowerType(c, file)
				seenColon = false
				return true
			}
			if seenArrow && (isExprKind(k) || isNameKind(k)) {
				c := n.bindChild(green, offset)
				af.Expr = lowerExpr(c, file)
			}
		}
		return true
	})
	return af
}

func lowerAnonymousClass(n *RedNode, file *File) (ast.Node, []ast.Node) {
	if n == nil {
		return nil, nil
	}
	pos, end := nodePos(file, n)
	cls := &ast.ClassNode{
		Pos:       pos,
		EndPos:    end,
		Modifiers: lowerModifiers(n),
		PHPDoc:    leadingDocFromNode(n),
	}
	// Readonly may appear as a bare token before T_CLASS on anonymous classes.
	n.ForEachChildDesc(func(green *GreenNode, _ int) bool {
		if isGreenTokenType(green, token.T_READONLY) {
			cls.Modifiers = append(cls.Modifiers, ast.Modifier{Tok: token.T_READONLY, Text: "readonly"})
		}
		return true
	})
	var ctorArgs []ast.Node
	var members *RedNode
	headerEnd := pos
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		switch k {
		case KindArgList:
			c := n.bindChild(green, offset)
			ctorArgs = lowerArgList(c, file)
			headerEnd = spanEnd(file, c.Span())
		case KindExtendsClause:
			c := n.bindChild(green, offset)
			names := clauseNames(c)
			if len(names) > 0 {
				cls.Extends = names[0]
			}
			headerEnd = spanEnd(file, c.Span())
		case KindImplementsClause:
			c := n.bindChild(green, offset)
			cls.Implements = clauseNames(c)
			headerEnd = spanEnd(file, c.Span())
		case KindMemberList:
			members = n.bindChild(green, offset)
		case KindModifierList:
			c := n.bindChild(green, offset)
			headerEnd = spanEnd(file, c.Span())
		}
		return true
	})
	cls.HeaderEndPos = headerEnd
	if members != nil {
		lowerClassMembers(members, file, cls)
	}
	return cls, ctorArgs
}

func lowerMatchExpr(n *RedNode, file *File) ast.Node {
	if n == nil {
		return nil
	}
	pos, end := nodePos(file, n)
	m := &ast.MatchNode{Pos: pos, EndPos: end}
	seenLParen, seenCond := false, false
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if isGreenTokenType(green, token.T_LPAREN) {
			seenLParen = true
			return true
		}
		k := green.Kind()
		if seenLParen && !seenCond && (isExprKind(k) || isNameKind(k)) {
			c := n.bindChild(green, offset)
			m.Condition = lowerExpr(c, file)
			seenCond = true
			return true
		}
		if k == KindMatchArm {
			c := n.bindChild(green, offset)
			if arm := lowerMatchArm(c, file); arm != nil {
				m.Arms = append(m.Arms, *arm)
			}
		}
		return true
	})
	return m
}

func lowerMatchArm(n *RedNode, file *File) *ast.MatchArmNode {
	if n == nil {
		return nil
	}
	pos, end := nodePos(file, n)
	arm := &ast.MatchArmNode{Pos: pos, EndPos: end}
	seenArrow := false
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if isGreenTokenType(green, token.T_DEFAULT) {
			c := n.bindChild(green, offset)
			dp, de := nodePos(file, c)
			arm.Conditions = []ast.Node{&ast.IdentifierNode{Value: "default", Pos: dp, EndPos: de}}
			return true
		}
		if isGreenTokenType(green, token.T_DOUBLE_ARROW) {
			seenArrow = true
			return true
		}
		if isGreenTokenType(green, token.T_COMMA) {
			return true
		}
		k := green.Kind()
		if !seenArrow && (isExprKind(k) || isNameKind(k)) {
			c := n.bindChild(green, offset)
			arm.Conditions = append(arm.Conditions, lowerExpr(c, file))
			return true
		}
		if seenArrow && (isExprKind(k) || isNameKind(k)) {
			c := n.bindChild(green, offset)
			arm.Body = lowerExpr(c, file)
		}
		return true
	})
	return arm
}

func lowerCloneExpr(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	var operand *RedNode
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		if isExprKind(k) || isNameKind(k) {
			operand = n.bindChild(green, offset)
			return false
		}
		return true
	})
	if operand == nil {
		return nil
	}
	return &ast.UnaryExpr{
		Operator: "clone",
		Operand:  lowerExpr(operand, file),
		Pos:      pos,
		EndPos:   end,
	}
}

func lowerListExpr(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	var elements []ast.Node
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if green.Kind() != KindArrayElement {
			return true
		}
		if item := lowerArrayElement(n.bindChild(green, offset), file); item != nil {
			elements = append(elements, item)
		}
		return true
	})
	return &ast.ArrayNode{Elements: elements, Pos: pos, EndPos: end}
}

func lowerKeywordUnaryExpr(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	var op string
	var operand *RedNode
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if green.IsToken() {
			tok, ok := green.Token()
			if ok {
				lit := strings.TrimSpace(tok.Literal)
				if lit != "" && op == "" {
					op = lit
				}
			}
			return true
		}
		k := green.Kind()
		if isExprKind(k) || isNameKind(k) {
			operand = n.bindChild(green, offset)
		}
		return true
	})
	if op == "" || operand == nil {
		return nil
	}
	return &ast.UnaryExpr{
		Operator: op,
		Operand:  lowerExpr(operand, file),
		Pos:      pos,
		EndPos:   end,
	}
}

func lowerInterpolatedStringLiteral(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	parts := lowerStringParts(n, file)
	if len(parts) == 0 {
		return &ast.StringLiteral{Value: "", Pos: pos, EndPos: end}
	}
	// Single plain string part with no variables → StringLiteral for parity with
	// empty interpolations; classic uses InterpolatedStringLiteral whenever the
	// double-quoted form was parsed as parts.
	return &ast.InterpolatedStringLiteral{Parts: parts, Pos: pos, EndPos: end}
}

func lowerHeredoc(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	ident := ""
	n.ForEachChildDesc(func(green *GreenNode, _ int) bool {
		if !green.IsToken() {
			return true
		}
		tok, ok := green.Token()
		if !ok {
			return true
		}
		if tok.Type == token.T_START_HEREDOC || tok.Type == token.T_START_NOWDOC {
			ident = heredocIdentifier(tok.Literal)
			return false
		}
		return true
	})
	return &ast.HeredocNode{
		Identifier: ident,
		Parts:      lowerStringParts(n, file),
		Pos:        pos,
		EndPos:     end,
	}
}

func lowerStringParts(n *RedNode, file *File) []ast.Node {
	if n == nil {
		return nil
	}
	var parts []ast.Node
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		switch green.Kind() {
		case KindStringPart:
			p, e := nodePosGreen(file, green, offset)
			lit := firstGreenTokenLiteral(green, offset)
			parts = append(parts, &ast.StringNode{
				Value:  lit,
				Pos:    p,
				EndPos: e,
			})
		case KindVariablePart:
			p, e := nodePosGreen(file, green, offset)
			name := stripVarDollar(firstGreenTokenLiteral(green, offset))
			if name == "" {
				part := RedNode{File: n.File, Green: green, Offset: offset}
				name = stripVarDollar(strings.TrimSpace(part.Text()))
			}
			parts = append(parts, &ast.VariableNode{Name: name, Pos: p, EndPos: e})
		case KindEncapsulatedExpr:
			enc := RedNode{File: n.File, Green: green, Offset: offset}
			if e := lowerEncapsulatedExpr(&enc, file); e != nil {
				parts = append(parts, e)
			}
		}
		return true
	})
	return parts
}

func firstTokenChild(n *RedNode) *RedNode {
	if n == nil {
		return nil
	}
	var found *RedNode
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if green.IsToken() {
			found = n.bindChild(green, offset)
			return false
		}
		return true
	})
	if found != nil {
		return found
	}
	if n.Green != nil && n.Green.IsToken() {
		return n
	}
	return nil
}

// lowerEncapsulatedExpr lowers {$expr} / ${…} inside strings or as a static
// member name. Prefer a structured expr child when the parser emitted one;
// otherwise recover a simple variable from token children.
func lowerEncapsulatedExpr(n *RedNode, file *File) ast.Node {
	if n == nil {
		return nil
	}
	var structured ast.Node
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		if isExprKind(k) || isNameKind(k) {
			structured = lowerExpr(n.bindChild(green, offset), file)
			return false
		}
		return true
	})
	if structured != nil {
		return structured
	}
	// Token-blob fallback: ${name} / {$var} before structured parse.
	var varname string
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if green.Kind() != KindToken {
			return true
		}
		c := n.bindChild(green, offset)
		switch green.TokenType() {
		case token.T_VARIABLE:
			structured = &ast.VariableNode{
				Name:   stripVarDollar(tokenLiteral(c)),
				Pos:    spanStart(file, c.Span()),
				EndPos: spanEnd(file, c.Span()),
			}
			return false
		case token.T_STRING_VARNAME, token.T_STRING:
			if varname == "" {
				varname = tokenLiteral(c)
			}
		}
		return true
	})
	if structured != nil {
		return structured
	}
	if varname != "" {
		pos, end := nodePos(file, n)
		return &ast.VariableNode{Name: varname, Pos: pos, EndPos: end}
	}
	return nil
}

func heredocIdentifier(lit string) string {
	lit = strings.TrimSpace(lit)
	lit = strings.TrimPrefix(lit, "<<<")
	lit = strings.TrimSpace(lit)
	lit = strings.Trim(lit, "'\"")
	return lit
}
