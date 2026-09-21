package analyse

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
	"github.com/ayanozturk/go-php-parser/token"
)

// tryCSTCallExprForMemo builds a lightweight FunctionCallNode / MethodCallNode
// for KindCallExpr without LowerExprNode (no recursive arg/object lowering).
// ok=false means fall back to full LowerExprNode; ok=true with nil means the
// call cannot be represented (same as lowerCallExpr nil → suppress subtree).
//
// Covers high-volume symbol shapes: simple/variable function names, Foo::bar(),
// -> / ?-> method calls (incl. chained / (new T) / paren receivers), and
// keyword-token builtins (isset/empty/exit/die).
func tryCSTCallExprForMemo(n *syntax.RedNode, file *syntax.File) (ast.Node, bool) {
	if n == nil || n.Kind() != syntax.KindCallExpr || file == nil {
		return nil, false
	}
	callee := syntax.CallCallee(n)
	if callee == nil {
		return tryCSTBuiltinCallForMemo(n)
	}
	args := stubCallArgsFromCST(n)
	callPos, callEnd := n.Pos(), n.EndPos()

	switch callee.Kind() {
	case syntax.KindUnqualifiedName, syntax.KindQualifiedName, syntax.KindFullyQualifiedName, syntax.KindRelativeName, syntax.KindName:
		name := syntax.NameText(callee)
		namePos := callee.Pos()
		return &ast.FunctionCallNode{
			Name:   &ast.IdentifierNode{Value: name, Pos: namePos, EndPos: callee.EndPos()},
			Args:   args,
			Pos:    namePos,
			EndPos: callEnd,
		}, true

	case syntax.KindVariableExpr:
		// $fn() — symbols ignore (functionCallName empty); still avoid Lower*.
		name := syntax.VariableExprName(callee)
		if name == "" {
			return nil, false
		}
		namePos := callee.Pos()
		return &ast.FunctionCallNode{
			Name:   &ast.VariableNode{Name: name, Pos: namePos, EndPos: callee.EndPos()},
			Args:   args,
			Pos:    namePos,
			EndPos: callEnd,
		}, true

	case syntax.KindMemberAccessExpr, syntax.KindNullsafeMemberAccessExpr:
		method := syntax.MemberAccessName(callee)
		if method == "" {
			// Dynamic $obj->{$expr}() — lowerCallExpr returns nil.
			return nil, true
		}
		objRed := syntax.MemberAccessObject(callee)
		obj, ok := lightweightCallObject(objRed)
		if !ok {
			return nil, false
		}
		return &ast.MethodCallNode{
			Object:   obj,
			Method:   method,
			Args:     args,
			Pos:      callPos,
			EndPos:   callEnd,
			Nullsafe: callee.Kind() == syntax.KindNullsafeMemberAccessExpr,
		}, true

	case syntax.KindStaticMemberAccessExpr:
		class, member, dynamic := syntax.StaticMemberAccessParts(callee)
		if class == "" {
			return nil, false
		}
		if dynamic {
			// Foo::{$m}() — FunctionCall with ClassConstFetch Name; rare, fall back.
			return nil, false
		}
		if member == "" {
			return nil, true
		}
		return &ast.FunctionCallNode{
			Name: &ast.IdentifierNode{
				Value:  class + "::" + member,
				Pos:    callPos,
				EndPos: callEnd,
			},
			Args:   args,
			Pos:    callPos,
			EndPos: callEnd,
		}, true

	case syntax.KindParenExpr:
		// (expr)() — uncommon; fall back unless inner is a simple name/var.
		inner := syntax.ParenInner(callee)
		if inner == nil {
			return nil, false
		}
		switch inner.Kind() {
		case syntax.KindUnqualifiedName, syntax.KindQualifiedName, syntax.KindFullyQualifiedName, syntax.KindRelativeName, syntax.KindName:
			name := syntax.NameText(inner)
			namePos := inner.Pos()
			return &ast.FunctionCallNode{
				Name:   &ast.IdentifierNode{Value: name, Pos: namePos, EndPos: inner.EndPos()},
				Args:   args,
				Pos:    namePos,
				EndPos: callEnd,
			}, true
		case syntax.KindVariableExpr:
			name := syntax.VariableExprName(inner)
			if name == "" {
				return nil, false
			}
			namePos := inner.Pos()
			return &ast.FunctionCallNode{
				Name:   &ast.VariableNode{Name: name, Pos: namePos, EndPos: inner.EndPos()},
				Args:   args,
				Pos:    namePos,
				EndPos: callEnd,
			}, true
		default:
			return nil, false
		}

	default:
		return nil, false
	}
}

func lightweightCallObject(n *syntax.RedNode) (ast.Node, bool) {
	if n == nil {
		return nil, false
	}
	pos, end := n.Pos(), n.EndPos()
	switch n.Kind() {
	case syntax.KindVariableExpr:
		name := syntax.VariableExprName(n)
		if name == "" {
			return nil, false
		}
		return &ast.VariableNode{Name: name, Pos: pos, EndPos: end}, true
	case syntax.KindUnqualifiedName, syntax.KindQualifiedName, syntax.KindFullyQualifiedName, syntax.KindRelativeName, syntax.KindName:
		return &ast.IdentifierNode{Value: syntax.NameText(n), Pos: pos, EndPos: end}, true
	case syntax.KindParenExpr:
		return lightweightCallObject(syntax.ParenInner(n))
	case syntax.KindNewExpr:
		return tryCSTNewExprForMemo(n)
	case syntax.KindMemberAccessExpr, syntax.KindNullsafeMemberAccessExpr:
		return tryCSTMemberAccessForMemo(n)
	case syntax.KindStaticMemberAccessExpr:
		class, member, dynamic := syntax.StaticMemberAccessParts(n)
		if dynamic || class == "" || member == "" {
			return nil, false
		}
		if member == "class" {
			return &ast.ClassConstFetchNode{Class: class, Const: "class", Pos: pos, EndPos: end}, true
		}
		// Foo::CONST as receiver is unusual for methodCallClassName; fall back.
		return nil, false
	default:
		return nil, false
	}
}

// tryCSTNewExprForMemo builds NewNode for simple `new Name` / `new $var`
// without lowering constructor args or anonymous classes.
func tryCSTNewExprForMemo(n *syntax.RedNode) (ast.Node, bool) {
	if n == nil || n.Kind() != syntax.KindNewExpr {
		return nil, false
	}
	if syntax.NewIsAnonymous(n) {
		return nil, false
	}
	cls := syntax.NewClass(n)
	if cls == nil {
		return nil, false
	}
	args := stubCallArgsFromCST(n)
	pos, end := n.Pos(), n.EndPos()
	switch cls.Kind() {
	case syntax.KindUnqualifiedName, syntax.KindQualifiedName, syntax.KindFullyQualifiedName, syntax.KindRelativeName, syntax.KindName:
		return &ast.NewNode{
			ClassName: syntax.NameText(cls),
			Args:      args,
			Pos:       pos,
			EndPos:    end,
		}, true
	case syntax.KindVariableExpr:
		name := syntax.VariableExprName(cls)
		if name == "" {
			return nil, false
		}
		return &ast.NewNode{
			ClassName: "$" + name,
			Args:      args,
			Pos:       pos,
			EndPos:    end,
		}, true
	default:
		return nil, false
	}
}

// tryCSTBuiltinCallForMemo handles KindCallExpr whose name is a keyword token
// (isset/empty/exit/die) with no expr/name callee child. Stub args avoid
// recursive LowerExprNode on ArgList. Lone-arg forms without ArgList
// (rare; e.g. legacy isset $a) fall back to full lower.
func tryCSTBuiltinCallForMemo(n *syntax.RedNode) (ast.Node, bool) {
	if n == nil || n.Kind() != syntax.KindCallExpr {
		return nil, false
	}
	var name string
	var namePos, nameEnd ast.Position
	found := false
	n.ForEachChildDesc(func(green *syntax.GreenNode, offset int) bool {
		if green == nil || green.Kind() != syntax.KindToken {
			return true
		}
		switch green.TokenType() {
		case token.T_ISSET:
			name = "isset"
		case token.T_EMPTY:
			name = "empty"
		case token.T_EXIT:
			name = "exit"
		case token.T_DIE:
			name = "die"
		default:
			return true
		}
		tok := &syntax.RedNode{File: n.File, Green: green, Offset: offset}
		namePos, nameEnd = tok.Pos(), tok.EndPos()
		found = true
		return false
	})
	if !found {
		return nil, false
	}
	if syntax.CallArgList(n) == nil {
		// exit;/die; with no parens is fine (no args). Any non-token child
		// without ArgList is the lone-arg form — leave to lowerCallExpr.
		hasLone := false
		n.ForEachChildDesc(func(green *syntax.GreenNode, _ int) bool {
			if green == nil || green.Kind() == syntax.KindToken {
				return true
			}
			hasLone = true
			return false
		})
		if hasLone {
			return nil, false
		}
	}
	callPos, callEnd := n.Pos(), n.EndPos()
	return &ast.FunctionCallNode{
		Name:   &ast.IdentifierNode{Value: name, Pos: namePos, EndPos: nameEnd},
		Args:   stubCallArgsFromCST(n),
		Pos:    callPos,
		EndPos: callEnd,
	}, true
}

// stubCallArgsFromCST builds arg nodes sufficient for checkCallArguments /
// checkNamedArguments (count, named names, unpacked flags) without lowering
// argument expressions. Works for KindCallExpr and KindNewExpr (ArgList child).
func stubCallArgsFromCST(call *syntax.RedNode) []ast.Node {
	raw := syntax.CallArgs(call)
	if len(raw) == 0 {
		return nil
	}
	out := make([]ast.Node, 0, len(raw))
	for _, a := range raw {
		pos, end := a.Pos(), a.EndPos()
		switch a.Kind() {
		case syntax.KindNamedArg:
			name := syntax.NamedArgName(a)
			out = append(out, &ast.NamedArgumentNode{Name: name, Pos: pos, EndPos: end})
		case syntax.KindArg:
			if syntax.ArgIsUnpacked(a) {
				out = append(out, &ast.UnpackedArgumentNode{Pos: pos, EndPos: end})
			} else {
				out = append(out, &ast.IdentifierNode{Pos: pos, EndPos: end})
			}
		}
	}
	return out
}

// tryCSTMemberAccessForMemo builds PropertyFetchNode for non-call member
// access without lowering the receiver expression tree.
func tryCSTMemberAccessForMemo(n *syntax.RedNode) (ast.Node, bool) {
	if n == nil {
		return nil, false
	}
	switch n.Kind() {
	case syntax.KindMemberAccessExpr, syntax.KindNullsafeMemberAccessExpr:
	default:
		return nil, false
	}
	prop := syntax.MemberAccessName(n)
	if prop == "" {
		return nil, false
	}
	obj, ok := lightweightCallObject(syntax.MemberAccessObject(n))
	if !ok {
		return nil, false
	}
	pos, end := n.Pos(), n.EndPos()
	return &ast.PropertyFetchNode{
		Object:   obj,
		Property: prop,
		Pos:      pos,
		EndPos:   end,
	}, true
}

// tryCSTStaticMemberForMemo builds ClassConstFetchNode for Foo::BAR /
// Foo::$prop without full lower. Dynamic Foo::{$m} falls back.
func tryCSTStaticMemberForMemo(n *syntax.RedNode) (ast.Node, bool) {
	if n == nil || n.Kind() != syntax.KindStaticMemberAccessExpr {
		return nil, false
	}
	class, member, dynamic := syntax.StaticMemberAccessParts(n)
	if dynamic || class == "" || member == "" {
		return nil, false
	}
	pos, end := n.Pos(), n.EndPos()
	return &ast.ClassConstFetchNode{
		Class:  class,
		Const:  member,
		Pos:    pos,
		EndPos: end,
	}, true
}
