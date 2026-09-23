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
			fn.Params = lowerParamListAt(file, green, offset)
			headerEnd = spanEnd(file, Span{Start: offset, End: offset + green.width})
		case KindClosureUseClause:
			fn.Uses = lowerClosureUsesAt(file, green, offset)
			headerEnd = spanEnd(file, Span{Start: offset, End: offset + green.width})
		case KindStatementList:
			fn.Body = lowerStatementsAt(file, green, offset)
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
				fn.ReturnType = lowerTypeAt(file, green, offset)
				headerEnd = spanEnd(file, Span{Start: offset, End: offset + green.width})
				seenColon = false
			}
		}
		return true
	})
	fn.HeaderEndPos = headerEnd
	return fn
}

func lowerClosureUsesAt(file *File, green *GreenNode, offset int) []ast.ClosureUse {
	if file == nil || green == nil {
		return nil
	}
	var out []ast.ClosureUse
	byRef := false
	forEachChildDescGreen(green, offset, func(g *GreenNode, off int) bool {
		if !g.IsToken() {
			return true
		}
		switch g.TokenType() {
		case token.T_AMPERSAND:
			byRef = true
		case token.T_VARIABLE:
			p, e := nodePosGreen(file, g, off)
			out = append(out, ast.ClosureUse{
				Name:   stripVarDollar(greenTokenLiteral(g)),
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
			af.Params = lowerParamListAt(file, green, offset)
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
				af.ReturnType = lowerTypeAt(file, green, offset)
				seenColon = false
				return true
			}
			if seenArrow && (isExprKind(k) || isNameKind(k)) {
				af.Expr = lowerExprAt(file, green, offset)
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
	var membersGreen *GreenNode
	var membersOff int
	headerEnd := pos
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		switch k {
		case KindArgList:
			ctorArgs = lowerArgListFromGreen(file, green, offset)
			headerEnd = spanEnd(file, Span{Start: offset, End: offset + green.width})
		case KindExtendsClause:
			names := clauseNamesGreen(file, green, offset)
			if len(names) > 0 {
				cls.Extends = names[0]
			}
			headerEnd = spanEnd(file, Span{Start: offset, End: offset + green.width})
		case KindImplementsClause:
			cls.Implements = clauseNamesGreen(file, green, offset)
			headerEnd = spanEnd(file, Span{Start: offset, End: offset + green.width})
		case KindMemberList:
			membersGreen, membersOff = green, offset
		case KindModifierList:
			headerEnd = spanEnd(file, Span{Start: offset, End: offset + green.width})
		}
		return true
	})
	cls.HeaderEndPos = headerEnd
	if membersGreen != nil {
		lowerClassMembers(file, membersGreen, membersOff, cls, true)
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
			m.Condition = lowerExprAt(file, green, offset)
			seenCond = true
			return true
		}
		if k == KindMatchArm {
			withPooledRed(file, n, green, offset, func(armNode *RedNode) {
				if arm := lowerMatchArm(armNode, file); arm != nil {
					m.Arms = append(m.Arms, *arm)
				}
			})
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
			dp, de := nodePosGreen(file, green, offset)
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
			arm.Conditions = append(arm.Conditions, lowerExprAt(file, green, offset))
			return true
		}
		if seenArrow && (isExprKind(k) || isNameKind(k)) {
			arm.Body = lowerExprAt(file, green, offset)
		}
		return true
	})
	return arm
}

func lowerCloneExpr(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	var operandGreen *GreenNode
	var operandOff int
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		if isExprKind(k) || isNameKind(k) {
			operandGreen, operandOff = green, offset
			return false
		}
		return true
	})
	if operandGreen == nil {
		return nil
	}
	return &ast.UnaryExpr{
		Operator: "clone",
		Operand:  lowerExprAt(file, operandGreen, operandOff),
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
		if item := lowerArrayElementFromGreen(file, green, offset); item != nil {
			elements = append(elements, item)
		}
		return true
	})
	return &ast.ArrayNode{Elements: elements, Pos: pos, EndPos: end}
}

func lowerKeywordUnaryExpr(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	var op string
	var operandGreen *GreenNode
	var operandOff int
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
			operandGreen, operandOff = green, offset
		}
		return true
	})
	if op == "" || operandGreen == nil {
		return nil
	}
	return &ast.UnaryExpr{
		Operator: op,
		Operand:  lowerExprAt(file, operandGreen, operandOff),
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
			if name == "" && file != nil {
				start := offset
				end := offset + green.width
				if start >= 0 && end <= len(file.Source) {
					name = stripVarDollar(strings.TrimSpace(string(file.Source[start:end])))
				}
			}
			parts = append(parts, &ast.VariableNode{Name: name, Pos: p, EndPos: e})
		case KindEncapsulatedExpr:
			if e := lowerEncapsulatedExprAt(file, green, offset); e != nil {
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
	var foundGreen *GreenNode
	var foundOff int
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if green.IsToken() {
			foundGreen, foundOff = green, offset
			return false
		}
		return true
	})
	if foundGreen != nil {
		return &RedNode{File: n.File, Parent: n, Green: foundGreen, Offset: foundOff}
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
			structured = lowerExprAt(file, green, offset)
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
		switch green.TokenType() {
		case token.T_VARIABLE:
			p, e := nodePosGreen(file, green, offset)
			structured = &ast.VariableNode{
				Name:   stripVarDollar(greenTokenLiteral(green)),
				Pos:    p,
				EndPos: e,
			}
			return false
		case token.T_STRING_VARNAME, token.T_STRING:
			if varname == "" {
				varname = greenTokenLiteral(green)
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
