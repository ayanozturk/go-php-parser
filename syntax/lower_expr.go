package syntax

import (
	"strconv"
	"strings"
	"sync"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/token"
)

var redNodePool = sync.Pool{New: func() any { return new(RedNode) }}

// forEachChildDescGreen walks green's children at absolute offset without RedNode allocation.
func forEachChildDescGreen(green *GreenNode, offset int, fn func(green *GreenNode, offset int) bool) {
	if green == nil || len(green.children) == 0 || fn == nil {
		return
	}
	off := offset
	for _, g := range green.children {
		if g == nil {
			continue
		}
		if !fn(g, off) {
			return
		}
		off += g.width
	}
}

// withPooledRed binds green+offset to a pooled RedNode for the duration of fn.
func withPooledRed(file *File, parent *RedNode, green *GreenNode, offset int, fn func(*RedNode)) {
	if file == nil || green == nil || fn == nil {
		return
	}
	n := redNodePool.Get().(*RedNode)
	n.File = file
	n.Parent = parent
	n.Green = green
	n.Offset = offset
	defer func() {
		*n = RedNode{}
		redNodePool.Put(n)
	}()
	fn(n)
}

// lowerExprAt lowers the child described by green+offset without a prior bindChild.
func lowerExprAt(file *File, green *GreenNode, offset int) ast.Node {
	var out ast.Node
	withPooledRed(file, nil, green, offset, func(n *RedNode) {
		out = lowerExpr(n, file)
	})
	return out
}

func lowerTypeAt(file *File, green *GreenNode, offset int) ast.Node {
	var out ast.Node
	withPooledRed(file, nil, green, offset, func(n *RedNode) {
		out = lowerType(n, file)
	})
	return out
}

func lowerStatementsAt(file *File, green *GreenNode, offset int) []ast.Node {
	var out []ast.Node
	withPooledRed(file, nil, green, offset, func(n *RedNode) {
		out = lowerStatements(n, file)
	})
	return out
}

func lowerParamListAt(file *File, green *GreenNode, offset int) []ast.Node {
	var out []ast.Node
	withPooledRed(file, nil, green, offset, func(n *RedNode) {
		out = lowerParamList(n, file)
	})
	return out
}

func lowerParenExprAt(file *File, green *GreenNode, offset int) ast.Node {
	var out ast.Node
	withPooledRed(file, nil, green, offset, func(n *RedNode) {
		out = lowerParenExpr(n, file)
	})
	return out
}

func lowerEncapsulatedExprAt(file *File, green *GreenNode, offset int) ast.Node {
	var out ast.Node
	withPooledRed(file, nil, green, offset, func(n *RedNode) {
		out = lowerEncapsulatedExpr(n, file)
	})
	return out
}

// lowerExpr lowers one expression CST node to a classic AST node.
// Unsupported kinds return nil without panicking.
func lowerExpr(n *RedNode, file *File) ast.Node {
	if n == nil {
		return nil
	}
	switch n.Kind() {
	case KindVariableExpr:
		return lowerVariableExpr(n, file)
	case KindLiteralExpr:
		return lowerLiteralExpr(n, file)
	case KindUnqualifiedName, KindQualifiedName, KindFullyQualifiedName, KindRelativeName, KindName:
		pos, end := nodePos(file, n)
		return &ast.IdentifierNode{Value: NameText(n), Pos: pos, EndPos: end}
	case KindBinaryExpr:
		return lowerBinaryExpr(n, file)
	case KindAssignExpr:
		return lowerAssignExpr(n, file)
	case KindCallExpr:
		return lowerCallExpr(n, file)
	case KindMemberAccessExpr, KindNullsafeMemberAccessExpr:
		return lowerMemberAccessExpr(n, file)
	case KindParenExpr:
		return lowerParenExpr(n, file)
	case KindUnaryExpr:
		return lowerUnaryExpr(n, file)
	case KindCastExpr:
		return lowerCastExpr(n, file)
	case KindStaticMemberAccessExpr:
		return lowerStaticMemberAccessExpr(n, file)
	case KindArrayAccessExpr:
		return lowerArrayAccessExpr(n, file)
	case KindNewExpr:
		return lowerNewExpr(n, file)
	case KindArrayExpr:
		return lowerArrayExpr(n, file)
	case KindTernaryExpr:
		return lowerTernaryExpr(n, file)
	case KindThrowExpr:
		return lowerThrowExpr(n, file)
	case KindClosureExpr:
		return lowerClosureExpr(n, file)
	case KindArrowFunctionExpr:
		return lowerArrowFunctionExpr(n, file)
	case KindMatchExpr:
		return lowerMatchExpr(n, file)
	case KindCloneExpr:
		return lowerCloneExpr(n, file)
	case KindListExpr:
		return lowerListExpr(n, file)
	case KindYieldExpr:
		return lowerYieldExpr(n, file)
	case KindIncludeExpr, KindPrintExpr:
		return lowerKeywordUnaryExpr(n, file)
	case KindStringLiteral:
		return lowerInterpolatedStringLiteral(n, file)
	case KindHeredoc, KindNowdoc:
		return lowerHeredoc(n, file)
	case KindVariableVariableExpr:
		return lowerVariableVariableExpr(n, file)
	case KindFirstClassCallableExpr:
		return lowerFirstClassCallableExpr(n, file)
	default:
		return nil
	}
}

func lowerVariableExpr(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	name := ""
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if isGreenTokenType(green, token.T_VARIABLE) {
			tt, ok := green.Token()
			if ok {
				lit := tt.Literal
				if lit == "" {
					lit = greenTokenLiteral(green)
				}
				name = stripVarDollar(lit)
			}
			return false
		}
		return true
	})
	if name == "" {
		name = stripVarDollar(strings.TrimSpace(n.Text()))
	}
	if name == "" {
		return nil
	}
	return &ast.VariableNode{Name: name, Pos: pos, EndPos: end}
}

func lowerLiteralExpr(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	var litTok token.Token
	var haveTok bool
	var nameLit ast.Node
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if green.IsToken() {
			tok, ok := green.Token()
			if ok {
				litTok = tok
				haveTok = true
			}
			return false
		}
		if isNameKind(green.Kind()) {
			// bare name used as literal-ish (e.g. array keyword path)
			nameLit = &ast.IdentifierNode{Value: nameTextAt(file, green, offset), Pos: pos, EndPos: end}
			return false
		}
		return true
	})
	if nameLit != nil {
		return nameLit
	}
	if !haveTok {
		return nil
	}
	lit := litTok.Literal
	switch litTok.Type {
	case token.T_LNUMBER:
		v, _ := strconv.ParseInt(strings.ReplaceAll(lit, "_", ""), 0, 64)
		return &ast.IntegerNode{Value: v, Pos: pos, EndPos: end}
	case token.T_DNUMBER:
		v, _ := strconv.ParseFloat(strings.ReplaceAll(lit, "_", ""), 64)
		return &ast.FloatNode{Value: v, Pos: pos, EndPos: end}
	case token.T_CONSTANT_ENCAPSED_STRING, token.T_CONSTANT_STRING:
		return &ast.StringLiteral{Value: decodeLowerStringLiteral(lit), Pos: pos, EndPos: end}
	case token.T_TRUE:
		return &ast.BooleanNode{Value: true, Pos: pos, EndPos: end}
	case token.T_FALSE:
		return &ast.BooleanNode{Value: false, Pos: pos, EndPos: end}
	case token.T_NULL:
		return &ast.NullNode{Pos: pos, EndPos: end}
	default:
		trimmed := strings.TrimSpace(strings.ToLower(lit))
		switch trimmed {
		case "true":
			return &ast.BooleanNode{Value: true, Pos: pos, EndPos: end}
		case "false":
			return &ast.BooleanNode{Value: false, Pos: pos, EndPos: end}
		case "null":
			return &ast.NullNode{Pos: pos, EndPos: end}
		}
		return &ast.IdentifierNode{Value: lit, Pos: pos, EndPos: end}
	}
}

func lowerBinaryExpr(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	var leftGreen, rightGreen *GreenNode
	var leftOff, rightOff int
	var haveLeft bool
	var op string
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if green.IsToken() {
			tok, ok := green.Token()
			if ok {
				op = strings.TrimSpace(tok.Literal)
			}
			return true
		}
		k := green.Kind()
		if isExprKind(k) || isNameKind(k) {
			if !haveLeft {
				leftGreen, leftOff = green, offset
				haveLeft = true
			} else {
				rightGreen, rightOff = green, offset
			}
		}
		return true
	})
	if !haveLeft || op == "" {
		return nil
	}
	return &ast.BinaryExpr{
		Left:     lowerExprAt(file, leftGreen, leftOff),
		Operator: op,
		Right:    lowerExprAt(file, rightGreen, rightOff),
		Pos:      pos,
		EndPos:   end,
	}
}

func lowerAssignExpr(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	var leftGreen, rightGreen *GreenNode
	var leftOff, rightOff int
	var haveLeft bool
	var op string
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if green.IsToken() {
			tok, ok := green.Token()
			if ok {
				op = strings.TrimSpace(tok.Literal)
			}
			return true
		}
		k := green.Kind()
		if isExprKind(k) || isNameKind(k) {
			if !haveLeft {
				leftGreen, leftOff = green, offset
				haveLeft = true
			} else {
				rightGreen, rightOff = green, offset
			}
		}
		return true
	})
	if !haveLeft || op == "" {
		return nil
	}
	leftNode := lowerExprAt(file, leftGreen, leftOff)
	rightNode := lowerExprAt(file, rightGreen, rightOff)
	if leftNode != nil {
		pos = leftNode.GetPos()
	}
	if rightNode != nil {
		end = rightNode.GetEndPos()
	}
	return &ast.AssignmentNode{
		Left:     leftNode,
		Operator: op,
		Right:    rightNode,
		Pos:      pos,
		EndPos:   end,
	}
}

func lowerCallExpr(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	var calleeGreen, argsGreen, loneArgGreen *GreenNode
	var calleeOff, argsOff, loneArgOff int
	var haveCallee, haveArgs, haveBuiltin, haveLoneArg bool
	var builtinGreen *GreenNode
	var builtinOff int
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		switch {
		case k == KindArgList:
			argsGreen, argsOff = green, offset
			haveArgs = true
		case k == KindToken:
			if isGreenBuiltinCallNameToken(green) {
				builtinGreen, builtinOff = green, offset
				haveBuiltin = true
			}
		case isExprKind(k) || isNameKind(k):
			if haveBuiltin && !haveArgs && !haveLoneArg {
				loneArgGreen, loneArgOff = green, offset
				haveLoneArg = true
			} else if !haveCallee {
				calleeGreen, calleeOff = green, offset
				haveCallee = true
			}
		}
		return true
	})
	var argNodes []ast.Node
	if haveArgs {
		argNodes = lowerArgListFromGreen(file, argsGreen, argsOff)
	}
	if haveBuiltin {
		tt, ok := builtinGreen.Token()
		if !ok {
			return nil
		}
		namePos, nameEnd := nodePosGreen(file, builtinGreen, builtinOff)
		if haveLoneArg && len(argNodes) == 0 {
			if a := lowerExprAt(file, loneArgGreen, loneArgOff); a != nil {
				argNodes = []ast.Node{a}
			}
		}
		return &ast.FunctionCallNode{
			Name: &ast.IdentifierNode{
				Value:  strings.ToLower(strings.TrimSpace(tt.Literal)),
				Pos:    namePos,
				EndPos: nameEnd,
			},
			Args:   argNodes,
			Pos:    pos,
			EndPos: end,
		}
	}
	if !haveCallee {
		return nil
	}

	switch calleeGreen.Kind() {
	case KindMemberAccessExpr, KindNullsafeMemberAccessExpr:
		var obj ast.Node
		var method string
		withPooledRed(file, nil, calleeGreen, calleeOff, func(callee *RedNode) {
			obj, method = splitMemberAccess(callee, file)
		})
		if method == "" {
			return nil
		}
		return &ast.MethodCallNode{
			Object:   obj,
			Method:   method,
			Args:     argNodes,
			Pos:      pos,
			EndPos:   end,
			Nullsafe: calleeGreen.Kind() == KindNullsafeMemberAccessExpr,
		}
	case KindStaticMemberAccessExpr:
		var class, member string
		var constExpr ast.Node
		withPooledRed(file, nil, calleeGreen, calleeOff, func(callee *RedNode) {
			class, member, constExpr = splitStaticMemberAccessParts(callee, file)
		})
		if class == "" {
			return nil
		}
		if constExpr != nil {
			return &ast.FunctionCallNode{
				Name: &ast.ClassConstFetchNode{
					Class:     class,
					Const:     "$",
					ConstExpr: constExpr,
					Pos:       pos,
					EndPos:    end,
				},
				Args:   argNodes,
				Pos:    pos,
				EndPos: end,
			}
		}
		if member == "" {
			return nil
		}
		return &ast.FunctionCallNode{
			Name: &ast.IdentifierNode{
				Value:  class + "::" + member,
				Pos:    pos,
				EndPos: end,
			},
			Args:   argNodes,
			Pos:    pos,
			EndPos: end,
		}
	default:
		name := lowerExprAt(file, calleeGreen, calleeOff)
		callPos, callEnd := pos, end
		if name != nil {
			callPos = name.GetPos()
		}
		return &ast.FunctionCallNode{
			Name:   name,
			Args:   argNodes,
			Pos:    callPos,
			EndPos: callEnd,
		}
	}
}

func lowerArgList(list *RedNode, file *File) []ast.Node {
	if list == nil {
		return nil
	}
	return lowerArgListFromGreen(file, list.Green, list.Offset)
}

func lowerArgListFromGreen(file *File, green *GreenNode, offset int) []ast.Node {
	if file == nil || green == nil {
		return nil
	}
	var out []ast.Node
	forEachChildDescGreen(green, offset, func(g *GreenNode, off int) bool {
		switch g.Kind() {
		case KindArg:
			if a := lowerArgFromGreen(file, g, off); a != nil {
				out = append(out, a)
			}
		case KindNamedArg:
			if a := lowerNamedArgFromGreen(file, g, off); a != nil {
				out = append(out, a)
			}
		}
		return true
	})
	return out
}

func lowerArg(n *RedNode, file *File) ast.Node {
	if n == nil {
		return nil
	}
	return lowerArgFromGreen(file, n.Green, n.Offset)
}

func lowerArgFromGreen(file *File, green *GreenNode, offset int) ast.Node {
	unpacked := false
	var exprGreen *GreenNode
	var exprOff int
	var haveExpr bool
	forEachChildDescGreen(green, offset, func(g *GreenNode, off int) bool {
		if isGreenTokenType(g, token.T_ELLIPSIS) {
			unpacked = true
			return true
		}
		k := g.Kind()
		if isExprKind(k) || isNameKind(k) {
			exprGreen, exprOff = g, off
			haveExpr = true
		}
		return true
	})
	if !haveExpr {
		return nil
	}
	lowered := lowerExprAt(file, exprGreen, exprOff)
	if lowered == nil {
		return nil
	}
	if unpacked {
		pos, end := nodePosGreen(file, green, offset)
		return &ast.UnpackedArgumentNode{Expr: lowered, Pos: pos, EndPos: end}
	}
	return lowered
}

func lowerNamedArg(n *RedNode, file *File) ast.Node {
	if n == nil {
		return nil
	}
	return lowerNamedArgFromGreen(file, n.Green, n.Offset)
}

func lowerNamedArgFromGreen(file *File, green *GreenNode, offset int) ast.Node {
	name := ""
	var exprGreen *GreenNode
	var exprOff int
	var haveExpr bool
	forEachChildDescGreen(green, offset, func(g *GreenNode, off int) bool {
		if g.IsToken() {
			tt, ok := g.Token()
			if !ok {
				return true
			}
			if tt.Type == token.T_STRING || tt.Type == token.T_CLASS {
				if name == "" {
					name = tt.Literal
				}
			}
			return true
		}
		k := g.Kind()
		if isExprKind(k) || isNameKind(k) {
			exprGreen, exprOff = g, off
			haveExpr = true
		}
		return true
	})
	if name == "" || !haveExpr {
		return nil
	}
	pos, end := nodePosGreen(file, green, offset)
	return &ast.NamedArgumentNode{
		Name:   name,
		Value:  lowerExprAt(file, exprGreen, exprOff),
		Pos:    pos,
		EndPos: end,
	}
}

func lowerMemberAccessExpr(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	obj, prop := splitMemberAccess(n, file)
	// Empty property is valid for incomplete `$obj->` (completion / recovery).
	if obj == nil && prop == "" {
		return nil
	}
	return &ast.PropertyFetchNode{
		Object:   obj,
		Property: prop,
		Pos:      pos,
		EndPos:   end,
	}
}

func splitMemberAccess(n *RedNode, file *File) (object ast.Node, member string) {
	var objGreen *GreenNode
	var objOff int
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if green.Kind() == KindToken {
			tt, ok := green.Token()
			if !ok {
				return true
			}
			switch tt.Type {
			case token.T_OBJECT_OPERATOR, token.T_NULLSAFE_OBJECT_OPERATOR:
				return true
			case token.T_STRING, token.T_VARIABLE:
				member = strings.TrimSpace(tt.Literal)
			default:
				if isContextualIdent(tt.Type, tt.Literal) {
					member = strings.TrimSpace(tt.Literal)
				}
			}
			return true
		}
		k := green.Kind()
		if (isExprKind(k) || isNameKind(k)) && objGreen == nil {
			objGreen, objOff = green, offset
		}
		return true
	})
	if objGreen != nil {
		object = lowerExprAt(file, objGreen, objOff)
	}
	return object, member
}

func lowerParenExpr(n *RedNode, file *File) ast.Node {
	var found ast.Node
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		if isExprKind(k) || isNameKind(k) {
			found = lowerExprAt(file, green, offset)
			return false
		}
		return true
	})
	return found
}

func lowerUnaryExpr(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	var op string
	var operandGreen *GreenNode
	var operandOff int
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if green.IsToken() {
			tok, ok := green.Token()
			if ok {
				op = strings.TrimSpace(tok.Literal)
			}
			return true
		}
		k := green.Kind()
		if (isExprKind(k) || isNameKind(k)) && operandGreen == nil {
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

func lowerCastExpr(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	castType := ""
	var operandGreen *GreenNode
	var operandOff int
	seenOpen := false
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if green.IsToken() {
			tt, ok := green.Token()
			if !ok {
				return true
			}
			switch tt.Type {
			case token.T_LPAREN:
				seenOpen = true
			case token.T_RPAREN:
				seenOpen = false
			default:
				if seenOpen && castType == "" {
					castType = strings.TrimSpace(tt.Literal)
				}
			}
			return true
		}
		k := green.Kind()
		if (isExprKind(k) || isNameKind(k)) && operandGreen == nil {
			operandGreen, operandOff = green, offset
		}
		return true
	})
	if castType == "" || operandGreen == nil {
		return nil
	}
	return &ast.TypeCastNode{
		Type:   castType,
		Expr:   lowerExprAt(file, operandGreen, operandOff),
		Pos:    pos,
		EndPos: end,
	}
}

func lowerStaticMemberAccessExpr(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	class, constName, constExpr := splitStaticMemberAccessParts(n, file)
	if class == "" {
		return nil
	}
	if constExpr != nil {
		// Dynamic name: Foo::{$m} / Foo::{$m->n} / Foo::${expr}
		return &ast.ClassConstFetchNode{
			Class:     class,
			Const:     "$",
			ConstExpr: constExpr,
			Pos:       pos,
			EndPos:    end,
		}
	}
	if constName == "" {
		return nil
	}
	return &ast.ClassConstFetchNode{
		Class:  class,
		Const:  constName,
		Pos:    pos,
		EndPos: end,
	}
}

func splitStaticMemberAccess(n *RedNode, file *File) (class, member string) {
	class, member, _ = splitStaticMemberAccessParts(n, file)
	return class, member
}

func splitStaticMemberAccessParts(n *RedNode, file *File) (class, member string, constExpr ast.Node) {
	if n == nil {
		return "", "", nil
	}
	seenColon := false
	done := false
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if green.Kind() == KindToken && green.TokenType() == token.T_DOUBLE_COLON {
			seenColon = true
			return true
		}
		k := green.Kind()
		if !seenColon {
			if isNameKind(k) && class == "" {
				class = nameTextAt(file, green, offset)
				return true
			}
			if (isExprKind(k) || isNameKind(k)) && class == "" {
				// Dynamic class expr — classic ClassConstFetch only stores string.
				// VariableNode.TokenLiteral omits "$"; keep it so Level0/symbols
				// treat `$var::method()` as a variable class, not bare `var`.
				if e := lowerExprAt(file, green, offset); e != nil {
					if v, ok := e.(*ast.VariableNode); ok {
						class = "$" + v.Name
					} else {
						class = e.TokenLiteral()
					}
				}
			}
			return true
		}
		// After :: — member name (literal token) or dynamic expr.
		switch k {
		case KindParenExpr:
			constExpr = lowerParenExprAt(file, green, offset)
			member = "$"
			done = true
			return false
		case KindEncapsulatedExpr:
			constExpr = lowerEncapsulatedExprAt(file, green, offset)
			member = "$"
			done = true
			return false
		default:
			if green.IsToken() {
				tok, ok := green.Token()
				if ok {
					member = strings.TrimSpace(tok.Literal)
				}
			}
		}
		return true
	})
	if done {
		return class, member, constExpr
	}
	return class, member, nil
}

func isBuiltinCallNameToken(n *RedNode) bool {
	return isTokenType(n, token.T_EXIT) || isTokenType(n, token.T_DIE) ||
		isTokenType(n, token.T_ISSET) || isTokenType(n, token.T_EMPTY) ||
		isTokenType(n, token.T_UNSET)
}

func isGreenBuiltinCallNameToken(g *GreenNode) bool {
	if g == nil || !g.IsToken() {
		return false
	}
	switch g.TokenType() {
	case token.T_EXIT, token.T_DIE, token.T_ISSET, token.T_EMPTY, token.T_UNSET:
		return true
	default:
		return false
	}
}

func lowerYieldExpr(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	from := false
	hasArrow := false
	var exprGreens []struct {
		green  *GreenNode
		offset int
	}
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if green.IsToken() {
			tt, ok := green.Token()
			if !ok {
				return true
			}
			// T_YIELD_FROM is never actually produced by the lexer (no
			// keyword table entry maps to it) - real "yield from" lexes as
			// plain T_YIELD followed by a separate T_STRING("from") token,
			// which is what the parser now appends as a child here. The
			// T_YIELD_FROM check is kept in case that ever changes.
			if tt.Type == token.T_YIELD_FROM {
				from = true
			}
			if tt.Type == token.T_STRING {
				lit := tt.Literal
				if lit == "" && file != nil {
					start := offset
					end := offset + green.width
					if start >= 0 && end <= len(file.Source) {
						lit = strings.TrimSpace(string(file.Source[start:end]))
					}
				}
				if strings.EqualFold(lit, "from") {
					from = true
				}
			}
			if tt.Type == token.T_DOUBLE_ARROW {
				hasArrow = true
			}
			return true
		}
		k := green.Kind()
		if isExprKind(k) || isNameKind(k) {
			exprGreens = append(exprGreens, struct {
				green  *GreenNode
				offset int
			}{green, offset})
		}
		return true
	})
	if len(exprGreens) == 0 {
		return &ast.YieldNode{From: from, Pos: pos, EndPos: end}
	}
	if from {
		return &ast.YieldNode{
			Value:  lowerExprAt(file, exprGreens[0].green, exprGreens[0].offset),
			From:   true,
			Pos:    pos,
			EndPos: end,
		}
	}
	var key, value ast.Node
	if hasArrow && len(exprGreens) >= 2 {
		key = lowerExprAt(file, exprGreens[0].green, exprGreens[0].offset)
		value = lowerExprAt(file, exprGreens[1].green, exprGreens[1].offset)
	} else {
		value = lowerExprAt(file, exprGreens[0].green, exprGreens[0].offset)
	}
	return &ast.YieldNode{
		Key:    key,
		Value:  value,
		From:   false,
		Pos:    pos,
		EndPos: end,
	}
}

func lowerArrayAccessExpr(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	var arrGreen, idxGreen *GreenNode
	var arrOff, idxOff int
	var haveArr, haveIdx bool
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		if isExprKind(k) || isNameKind(k) {
			if !haveArr {
				arrGreen, arrOff = green, offset
				haveArr = true
			} else {
				idxGreen, idxOff = green, offset
				haveIdx = true
			}
		}
		return true
	})
	if !haveArr {
		return nil
	}
	var index ast.Node
	if haveIdx {
		index = lowerExprAt(file, idxGreen, idxOff)
	}
	return &ast.ArrayAccessNode{
		Var:    lowerExprAt(file, arrGreen, arrOff),
		Index:  index,
		Pos:    pos,
		EndPos: end,
	}
}

func lowerNewExpr(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	var classGreen *GreenNode
	var classOff int
	var argsGreen *GreenNode
	var argsOff int
	var anonNew ast.Node
	var anonDone bool
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		switch k {
		case KindArgList:
			argsGreen, argsOff = green, offset
		case KindAttributeList:
			return true
		case KindAnonymousClass:
			var cls ast.Node
			var ctorArgs []ast.Node
			withPooledRed(file, n, green, offset, func(anon *RedNode) {
				cls, ctorArgs = lowerAnonymousClass(anon, file)
			})
			if cls == nil {
				anonDone = true
				return false
			}
			anonNew = &ast.NewNode{
				ClassExpr: cls,
				Args:      ctorArgs,
				Pos:       pos,
				EndPos:    end,
			}
			anonDone = true
			return false
		default:
			if (isExprKind(k) || isNameKind(k)) && classGreen == nil {
				classGreen, classOff = green, offset
			}
		}
		return true
	})
	if anonDone {
		return anonNew
	}
	if classGreen == nil {
		return nil
	}
	argNodes := lowerArgListFromGreen(file, argsGreen, argsOff)
	classKind := classGreen.Kind()
	switch classKind {
	case KindUnqualifiedName, KindQualifiedName, KindFullyQualifiedName, KindRelativeName, KindName:
		return &ast.NewNode{
			ClassName: nameTextAt(file, classGreen, classOff),
			Args:      argNodes,
			Pos:       pos,
			EndPos:    end,
		}
	case KindVariableExpr:
		v, ok := lowerExprAt(file, classGreen, classOff).(*ast.VariableNode)
		if !ok || v == nil {
			return &ast.NewNode{
				ClassExpr: lowerExprAt(file, classGreen, classOff),
				Args:      argNodes,
				Pos:       pos,
				EndPos:    end,
			}
		}
		// Classic stores bare `new $var` as ClassName="$name".
		return &ast.NewNode{
			ClassName: "$" + v.Name,
			Args:      argNodes,
			Pos:       pos,
			EndPos:    end,
		}
	default:
		return &ast.NewNode{
			ClassExpr: lowerExprAt(file, classGreen, classOff),
			Args:      argNodes,
			Pos:       pos,
			EndPos:    end,
		}
	}
}

func lowerArrayExpr(n *RedNode, file *File) ast.Node {
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

func lowerArrayElement(n *RedNode, file *File) ast.Node {
	if n == nil {
		return nil
	}
	return lowerArrayElementFromGreen(file, n.Green, n.Offset)
}

func lowerArrayElementFromGreen(file *File, green *GreenNode, offset int) ast.Node {
	if file == nil || green == nil {
		return nil
	}
	pos, end := nodePosGreen(file, green, offset)

	var firstChild *GreenNode
	childCount := 0
	forEachChildDescGreen(green, offset, func(g *GreenNode, off int) bool {
		childCount++
		if firstChild == nil {
			firstChild = g
		}
		return true
	})
	if childCount == 0 {
		return nil
	}

	if isGreenTokenType(firstChild, token.T_ELLIPSIS) {
		var valGreen *GreenNode
		var valOff int
		var haveVal bool
		skip := true
		forEachChildDescGreen(green, offset, func(g *GreenNode, off int) bool {
			if skip {
				skip = false
				return true
			}
			k := g.Kind()
			if isExprKind(k) || isNameKind(k) {
				valGreen, valOff = g, off
				haveVal = true
				return false
			}
			return true
		})
		if !haveVal {
			return nil
		}
		return &ast.ArrayItemNode{
			Value:  lowerExprAt(file, valGreen, valOff),
			Unpack: true,
			Pos:    pos,
			EndPos: end,
		}
	}

	i := 0
	byRefBeforeFirst := isGreenTokenType(firstChild, token.T_AMPERSAND)
	if byRefBeforeFirst {
		i = 1
	}
	var firstGreen *GreenNode
	var firstExprOff int
	idx := 0
	forEachChildDescGreen(green, offset, func(g *GreenNode, off int) bool {
		if idx < i {
			idx++
			return true
		}
		k := g.Kind()
		if firstGreen == nil && (isExprKind(k) || isNameKind(k)) {
			firstGreen, firstExprOff = g, off
			i = idx + 1
			return false
		}
		idx++
		return true
	})
	if firstGreen == nil {
		return nil
	}

	hasArrow := false
	idx = 0
	forEachChildDescGreen(green, offset, func(g *GreenNode, off int) bool {
		if idx < i {
			idx++
			return true
		}
		if isGreenTokenType(g, token.T_DOUBLE_ARROW) {
			hasArrow = true
			i = idx + 1
			return false
		}
		idx++
		return true
	})
	if !hasArrow {
		return &ast.ArrayItemNode{
			Value:  lowerExprAt(file, firstGreen, firstExprOff),
			ByRef:  byRefBeforeFirst,
			Pos:    pos,
			EndPos: end,
		}
	}

	key := lowerExprAt(file, firstGreen, firstExprOff)
	if byRefBeforeFirst && key != nil {
		kp, ke := nodePosGreen(file, firstGreen, firstExprOff)
		key = &ast.UnaryExpr{Operator: "&", Operand: key, Pos: kp, EndPos: ke}
	}
	byRefValue := false
	idx = 0
	forEachChildDescGreen(green, offset, func(g *GreenNode, off int) bool {
		if idx < i {
			idx++
			return true
		}
		if isGreenTokenType(g, token.T_AMPERSAND) {
			byRefValue = true
			i = idx + 1
		}
		return false
	})
	var valGreen *GreenNode
	var valOff int
	var haveVal bool
	idx = 0
	forEachChildDescGreen(green, offset, func(g *GreenNode, off int) bool {
		if idx < i {
			idx++
			return true
		}
		k := g.Kind()
		if isExprKind(k) || isNameKind(k) {
			valGreen, valOff = g, off
			haveVal = true
			return false
		}
		idx++
		return true
	})
	var value ast.Node
	if haveVal {
		value = lowerExprAt(file, valGreen, valOff)
	}
	return &ast.ArrayItemNode{
		Key:    key,
		Value:  value,
		ByRef:  byRefValue,
		Pos:    pos,
		EndPos: end,
	}
}

func lowerTernaryExpr(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	var condGreen *GreenNode
	var condOff int
	seenQ := false
	seenColon := false
	var thenGreen *GreenNode
	var thenOff int
	var elseGreen *GreenNode
	var elseOff int
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if green.IsToken() {
			tt, ok := green.Token()
			if !ok {
				return true
			}
			switch tt.Type {
			case token.T_QUESTION:
				seenQ = true
			case token.T_COLON:
				seenColon = true
			}
			return true
		}
		k := green.Kind()
		if !(isExprKind(k) || isNameKind(k)) {
			return true
		}
		if condGreen == nil {
			condGreen = green
			condOff = offset
			return true
		}
		if seenQ && !seenColon && thenGreen == nil {
			thenGreen = green
			thenOff = offset
			return true
		}
		if seenColon && elseGreen == nil {
			elseGreen = green
			elseOff = offset
		}
		return true
	})
	if condGreen == nil {
		return nil
	}
	condNode := lowerExprAt(file, condGreen, condOff)
	var ifTrue ast.Node
	if thenGreen == nil {
		// Elvis / short ternary: classic sets IfTrue = Condition.
		ifTrue = condNode
	} else {
		ifTrue = lowerExprAt(file, thenGreen, thenOff)
	}
	var ifFalse ast.Node
	if elseGreen != nil {
		ifFalse = lowerExprAt(file, elseGreen, elseOff)
	}
	return &ast.TernaryExpr{
		Condition: condNode,
		IfTrue:    ifTrue,
		IfFalse:   ifFalse,
		Pos:       pos,
		EndPos:    end,
	}
}

func lowerThrowExpr(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	expr := firstExprChild(n)
	if expr == nil {
		return nil
	}
	return &ast.ThrowNode{
		Expr:   lowerExpr(expr, file),
		Pos:    pos,
		EndPos: end,
	}
}

func lowerFirstClassCallableExpr(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	expr := firstExprChild(n)
	if expr == nil {
		return nil
	}
	lowered := lowerExpr(expr, file)
	if lowered == nil {
		return nil
	}
	fcc := &ast.FirstClassCallableNode{Pos: pos, EndPos: end}
	if name, ok := lowered.(*ast.IdentifierNode); ok {
		fcc.Name = name
	} else {
		fcc.Target = lowered
	}
	return fcc
}

func tokenOf(n *RedNode) (token.Token, bool) {
	if n == nil || n.Green == nil || !n.Green.IsToken() {
		return token.Token{}, false
	}
	return n.Green.Token()
}

func decodeLowerStringLiteral(lit string) string {
	if len(lit) >= 2 {
		q := lit[0]
		if (q == '"' || q == '\'') && lit[len(lit)-1] == q {
			inner := lit[1 : len(lit)-1]
			if q == '\'' {
				return unescapeLowerSingleQuoted(inner)
			}
			return unescapeLowerDoubleQuoted(inner)
		}
	}
	return lit
}

// DecodeStringLiteral returns the PHP string contents of a quoted literal
// token text (single- or double-quoted), matching lowering's
// decodeLowerStringLiteral. Unquoted input is returned unchanged.
func DecodeStringLiteral(lit string) string {
	return decodeLowerStringLiteral(lit)
}

func unescapeLowerSingleQuoted(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) && (s[i+1] == '\\' || s[i+1] == '\'') {
			b.WriteByte(s[i+1])
			i++
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func unescapeLowerDoubleQuoted(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			i++
			switch s[i] {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			case '"', '\\', '$':
				b.WriteByte(s[i])
			default:
				b.WriteByte('\\')
				b.WriteByte(s[i])
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func lowerVariableVariableExpr(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	var innerGreen *GreenNode
	var innerOff int
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		if isExprKind(k) || isNameKind(k) {
			innerGreen, innerOff = green, offset
			return false
		}
		return true
	})
	if innerGreen == nil {
		return nil
	}
	expr := lowerExprAt(file, innerGreen, innerOff)
	if expr == nil {
		return nil
	}
	return &ast.VariableVariableNode{
		Expr:   expr,
		Pos:    pos,
		EndPos: end,
	}
}
