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
	for _, c := range n.Children() {
		switch c.Kind() {
		case KindParamList:
			fn.Params = lowerParamList(c, file)
			headerEnd = spanEnd(file, c.Span())
		case KindClosureUseClause:
			fn.Uses = lowerClosureUses(c, file)
			headerEnd = spanEnd(file, c.Span())
		case KindStatementList:
			fn.Body = lowerStatements(c, file)
		case KindTokenList:
			fn.Body = nil
		default:
			if isTokenType(c, token.T_STATIC) {
				fn.Modifiers = append(fn.Modifiers, ast.Modifier{Tok: token.T_STATIC, Text: "static"})
				continue
			}
			if isTokenType(c, token.T_COLON) {
				seenColon = true
				continue
			}
			if seenColon && isTypeKind(c.Kind()) {
				fn.ReturnType = lowerType(c, file)
				headerEnd = spanEnd(file, c.Span())
				seenColon = false
			}
		}
	}
	fn.HeaderEndPos = headerEnd
	return fn
}

func lowerClosureUses(n *RedNode, file *File) []ast.ClosureUse {
	if n == nil {
		return nil
	}
	var out []ast.ClosureUse
	byRef := false
	for _, c := range n.Children() {
		if isTokenType(c, token.T_AMPERSAND) {
			byRef = true
			continue
		}
		if isTokenType(c, token.T_VARIABLE) {
			p, e := nodePos(file, c)
			out = append(out, ast.ClosureUse{
				Name:   stripVarDollar(tokenLiteral(c)),
				ByRef:  byRef,
				Pos:    p,
				EndPos: e,
			})
			byRef = false
		}
	}
	return out
}

func lowerArrowFunctionExpr(n *RedNode, file *File) ast.Node {
	if n == nil {
		return nil
	}
	pos, end := nodePos(file, n)
	af := &ast.ArrowFunctionNode{Pos: pos, EndPos: end}
	seenColon, seenArrow := false, false
	for _, c := range n.Children() {
		switch c.Kind() {
		case KindParamList:
			af.Params = lowerParamList(c, file)
		default:
			if isTokenType(c, token.T_COLON) {
				seenColon = true
				continue
			}
			if seenColon && isTypeKind(c.Kind()) {
				af.ReturnType = lowerType(c, file)
				seenColon = false
				continue
			}
			if isTokenType(c, token.T_DOUBLE_ARROW) {
				seenArrow = true
				continue
			}
			if seenArrow && (isExprKind(c.Kind()) || isNameKind(c.Kind())) {
				af.Expr = lowerExpr(c, file)
			}
		}
	}
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
	for _, c := range n.Children() {
		if isTokenType(c, token.T_READONLY) {
			cls.Modifiers = append(cls.Modifiers, ast.Modifier{Tok: token.T_READONLY, Text: "readonly"})
		}
	}
	var ctorArgs []ast.Node
	var members *RedNode
	headerEnd := pos
	for _, c := range n.Children() {
		switch c.Kind() {
		case KindArgList:
			ctorArgs = lowerArgList(c, file)
			headerEnd = spanEnd(file, c.Span())
		case KindExtendsClause:
			names := clauseNames(c)
			if len(names) > 0 {
				cls.Extends = names[0]
			}
			headerEnd = spanEnd(file, c.Span())
		case KindImplementsClause:
			cls.Implements = clauseNames(c)
			headerEnd = spanEnd(file, c.Span())
		case KindMemberList:
			members = c
		case KindModifierList:
			headerEnd = spanEnd(file, c.Span())
		}
	}
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
	for _, c := range n.Children() {
		if isTokenType(c, token.T_LPAREN) {
			seenLParen = true
			continue
		}
		if seenLParen && !seenCond && (isExprKind(c.Kind()) || isNameKind(c.Kind())) {
			m.Condition = lowerExpr(c, file)
			seenCond = true
			continue
		}
		if c.Kind() == KindMatchArm {
			if arm := lowerMatchArm(c, file); arm != nil {
				m.Arms = append(m.Arms, *arm)
			}
		}
	}
	return m
}

func lowerMatchArm(n *RedNode, file *File) *ast.MatchArmNode {
	if n == nil {
		return nil
	}
	pos, end := nodePos(file, n)
	arm := &ast.MatchArmNode{Pos: pos, EndPos: end}
	seenArrow := false
	for _, c := range n.Children() {
		if isTokenType(c, token.T_DEFAULT) {
			dp, de := nodePos(file, c)
			arm.Conditions = []ast.Node{&ast.IdentifierNode{Value: "default", Pos: dp, EndPos: de}}
			continue
		}
		if isTokenType(c, token.T_DOUBLE_ARROW) {
			seenArrow = true
			continue
		}
		if isTokenType(c, token.T_COMMA) {
			continue
		}
		if !seenArrow && (isExprKind(c.Kind()) || isNameKind(c.Kind())) {
			arm.Conditions = append(arm.Conditions, lowerExpr(c, file))
			continue
		}
		if seenArrow && (isExprKind(c.Kind()) || isNameKind(c.Kind())) {
			arm.Body = lowerExpr(c, file)
		}
	}
	return arm
}

func lowerCloneExpr(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	var operand *RedNode
	for _, c := range n.Children() {
		if isExprKind(c.Kind()) || isNameKind(c.Kind()) {
			operand = c
			break
		}
	}
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
	for _, c := range n.Children() {
		if c.Kind() != KindArrayElement {
			continue
		}
		if item := lowerArrayElement(c, file); item != nil {
			elements = append(elements, item)
		}
	}
	return &ast.ArrayNode{Elements: elements, Pos: pos, EndPos: end}
}

func lowerKeywordUnaryExpr(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	var op string
	var operand *RedNode
	for _, c := range n.Children() {
		if c.Green != nil && c.Green.IsToken() {
			lit := strings.TrimSpace(tokenLiteral(c))
			if lit != "" && op == "" {
				op = lit
			}
			continue
		}
		if isExprKind(c.Kind()) || isNameKind(c.Kind()) {
			operand = c
		}
	}
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
	for _, c := range n.Children() {
		if c.Green == nil || !c.Green.IsToken() {
			continue
		}
		tt, ok := tokenOf(c)
		if !ok {
			continue
		}
		if tt.Type == token.T_START_HEREDOC || tt.Type == token.T_START_NOWDOC {
			ident = heredocIdentifier(tt.Literal)
			break
		}
	}
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
	for _, c := range n.Children() {
		switch c.Kind() {
		case KindStringPart:
			p, e := nodePos(file, c)
			parts = append(parts, &ast.StringNode{
				Value:  tokenLiteral(firstTokenChild(c)),
				Pos:    p,
				EndPos: e,
			})
		case KindVariablePart:
			p, e := nodePos(file, c)
			name := stripVarDollar(tokenLiteral(firstTokenChild(c)))
			if name == "" {
				name = stripVarDollar(strings.TrimSpace(c.Text()))
			}
			parts = append(parts, &ast.VariableNode{Name: name, Pos: p, EndPos: e})
		case KindEncapsulatedExpr:
			// Classic skips {$expr} / ${expr} brace forms in interpolated strings.
			continue
		}
	}
	return parts
}

func firstTokenChild(n *RedNode) *RedNode {
	if n == nil {
		return nil
	}
	for _, c := range n.Children() {
		if c.Green != nil && c.Green.IsToken() {
			return c
		}
	}
	if n.Green != nil && n.Green.IsToken() {
		return n
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
