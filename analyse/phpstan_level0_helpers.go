package analyse

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"strings"
)

func functionCallName(call *ast.FunctionCallNode) string {
	switch name := call.Name.(type) {
	case *ast.IdentifierNode:
		return strings.TrimPrefix(name.Value, `\`)
	case *ast.Identifier:
		return strings.TrimPrefix(name.Name, `\`)
	}
	return ""
}

func resolveNewClassName(node *ast.NewNode, ft FileTypeContext) string {
	if node.ClassName != "" {
		return ft.resolveClassLike(node.ClassName)
	}
	if ident, ok := node.ClassExpr.(*ast.IdentifierNode); ok {
		return ft.resolveClassLike(ident.Value)
	}
	return ""
}

func resolveClassLikeForCall(name string, current *ast.ClassNode, ft FileTypeContext, ctx *AnalysisContext) string {
	switch strings.ToLower(strings.TrimPrefix(name, `\`)) {
	case "self", "static":
		if current != nil {
			return ft.resolveClassLike(current.Name)
		}
	case "parent":
		if current != nil {
			if ctx != nil && ctx.Resolver != nil {
				currentName := ft.resolveClassLike(current.Name)
				if class, ok := ctx.Resolver.ResolveClass(currentName); ok && len(class.Extends) > 0 {
					return class.Extends[0]
				}
			}
			if class, ok := ft.resolveClass(ft.resolveClassLike(current.Name)); ok && len(class.Extends) > 0 {
				return class.Extends[0]
			}
		}
	}
	return ft.resolveClassLike(name)
}

func resolveFunctionNameForCall(name string, ft FileTypeContext, ctx *AnalysisContext) string {
	name = strings.TrimPrefix(strings.TrimSpace(name), `\`)
	if name == "" || strings.Contains(name, "::") {
		return name
	}
	if ctx != nil && ctx.Resolver != nil && ctx.Resolver.FunctionExists(name) {
		return name
	}
	resolved := ft.resolveClassLike(name)
	if ctx != nil && ctx.Resolver != nil && ctx.Resolver.FunctionExists(resolved) {
		return resolved
	}
	return name
}

func currentClassName(class *ast.ClassNode, ft FileTypeContext) string {
	if class == nil {
		return ""
	}
	return ft.resolveClassLike(class.Name)
}

func isSpecialClassName(name string) bool {
	switch strings.ToLower(strings.TrimPrefix(name, `\`)) {
	case "", "self", "static", "parent":
		return true
	default:
		return false
	}
}

func isWritableExpr(node ast.Node) bool {
	switch node.(type) {
	case *ast.VariableNode, *ast.ArrayAccessNode, *ast.PropertyFetchNode:
		return true
	default:
		return false
	}
}

func titleKind(kind string) string {
	if kind == "" {
		return ""
	}
	return strings.ToUpper(kind[:1]) + kind[1:]
}

func stringLiteralValue(node ast.Node) (string, bool) {
	switch n := node.(type) {
	case *ast.StringLiteral:
		return n.Value, true
	case *ast.StringNode:
		return n.Value, true
	}
	return "", false
}

func cloneBoolMap(in map[string]bool) map[string]bool {
	out := make(map[string]bool, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func issue(filename string, pos ast.Position, code, message string) AnalysisIssue {
	return AnalysisIssue{Filename: filename, Line: pos.Line, Column: pos.Column, Code: code, Message: message}
}

// issueSpan is like issue but also records the node's end position, giving
// the diagnostic a full [start, end) source span instead of a single
// point. Prefer this over issue() wherever a concrete offending ast.Node
// (rather than a bare position) is available.
func issueSpan(filename string, n ast.Node, code, message string) AnalysisIssue {
	start := n.GetPos()
	end := n.GetEndPos()
	return AnalysisIssue{
		Filename:  filename,
		Line:      start.Line,
		Column:    start.Column,
		EndLine:   end.Line,
		EndColumn: end.Column,
		Code:      code,
		Message:   message,
	}
}

func resolveThrownClassName(node ast.Node, ft FileTypeContext) string {
	switch n := node.(type) {
	case *ast.NewNode:
		return resolveNewClassName(n, ft)
	case *ast.IdentifierNode:
		return ft.resolveClassLike(n.Value)
	case *ast.Identifier:
		return ft.resolveClassLike(n.Name)
	case *ast.ClassConstFetchNode:
		if n.Const == "class" {
			return ft.resolveClassLike(n.Class)
		}
	}
	return ""
}

func isSubclassOf(resolver SymbolResolver, current, target string) bool {
	return isSubclassOfSeen(resolver, current, target, map[string]struct{}{})
}

func isSubclassOfSeen(resolver SymbolResolver, current, target string, seen map[string]struct{}) bool {
	if resolver == nil || current == "" || target == "" {
		return false
	}
	if indexKey(current) == indexKey(target) {
		return true
	}
	key := indexKey(current)
	if _, visited := seen[key]; visited {
		return false
	}
	seen[key] = struct{}{}
	class, ok := resolver.ResolveClass(current)
	if !ok {
		return false
	}
	for _, parent := range class.Extends {
		if isSubclassOfSeen(resolver, parent, target, seen) {
			return true
		}
	}
	return false
}

func callerClassName(class *ast.ClassNode, ft FileTypeContext) string {
	if class == nil {
		return ""
	}
	return ft.resolveClassLike(class.Name)
}

func isThrowableClass(name string, resolver SymbolResolver) bool {
	if name == "" || isSpecialClassName(name) {
		return false
	}
	seen := map[string]struct{}{}
	var walk func(string) bool
	walk = func(className string) bool {
		key := indexKey(className)
		if key == "" {
			return false
		}
		if _, ok := seen[key]; ok {
			return false
		}
		seen[key] = struct{}{}
		if indexKey(className) == indexKey("Throwable") {
			return true
		}
		class, ok := resolver.ResolveClass(className)
		if !ok {
			return false
		}
		if class.Kind == "interface" {
			for _, iface := range class.Extends {
				if walk(iface) {
					return true
				}
			}
			return indexKey(class.Name) == indexKey("Throwable")
		}
		for _, parent := range class.Extends {
			if walk(parent) {
				return true
			}
		}
		for _, iface := range class.Implements {
			if walk(iface) {
				return true
			}
		}
		return false
	}
	return walk(name)
}
