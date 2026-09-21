package analyse

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

// tryCSTCallExprForMemo builds a lightweight FunctionCallNode / MethodCallNode
// for KindCallExpr without LowerExprNode (no recursive arg/object lowering).
// ok=false means fall back to full LowerExprNode; ok=true with nil means the
// call cannot be represented (same as lowerCallExpr nil → suppress subtree).
//
// Covers the high-volume symbol shapes: simple function names, Foo::bar(),
// and -> / ?-> method calls with simple receivers ($this / $var / Name).
func tryCSTCallExprForMemo(n *syntax.RedNode, file *syntax.File) (ast.Node, bool) {
	if n == nil || n.Kind() != syntax.KindCallExpr || file == nil {
		return nil, false
	}
	callee := syntax.CallCallee(n)
	if callee == nil {
		// Builtin token calls (isset/empty/…) — leave to full lower.
		return nil, false
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
	case syntax.KindStaticMemberAccessExpr:
		class, member, dynamic := syntax.StaticMemberAccessParts(n)
		if dynamic || class == "" || member == "" {
			return nil, false
		}
		if member == "class" {
			return &ast.ClassConstFetchNode{Class: class, Const: "class", Pos: pos, EndPos: end}, true
		}
		return nil, false
	default:
		return nil, false
	}
}

// stubCallArgsFromCST builds arg nodes sufficient for checkCallArguments /
// checkNamedArguments (count, named names, unpacked flags) without lowering
// argument expressions.
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
