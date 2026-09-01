package analyse

import (
	"fmt"
	"github.com/ayanozturk/go-php-parser/ast"
	"strings"
)

const argumentCountRuleCode = "A.ARG.COUNT"

type ArgumentCountRule struct{}

func (r *ArgumentCountRule) CheckIssues(nodes []ast.Node, filename string, ctx *AnalysisContext) []AnalysisIssue {
	ctx = ensureArgCallDiagnostics(filename, nodes, ctx)
	return ctx.argCountIssues
}

func checkMethodCallArgCount(call *ast.MethodCallNode, scope *functionScope, ctx *AnalysisContext, filename string, issues *[]AnalysisIssue) {
	method, ok := resolveMethodForCall(call, scope, ctx, filename)
	if !ok {
		return
	}
	if issue, ok := validateResolvedMethodCallCount(method, call.Args, filename, call.GetPos(), fmt.Sprintf("Method %s", method.Name)); ok {
		*issues = append(*issues, issue)
	}
}

func checkNewArgCount(node *ast.NewNode, scope *functionScope, ctx *AnalysisContext, filename string, issues *[]AnalysisIssue) {
	className, method, ok := resolveConstructorForNew(node, scope, ctx)
	if !ok {
		return
	}
	label := fmt.Sprintf("Class %s constructor", className)
	if issue, ok := validateResolvedMethodCallCount(method, node.Args, filename, node.GetPos(), label); ok {
		*issues = append(*issues, issue)
	}
}

func validateResolvedMethodCallCount(method ResolvedMethod, args []ast.Node, filename string, pos ast.Position, target string) (AnalysisIssue, bool) {
	actualCount, hasUnpacked := countCallArguments(args)
	if hasUnpacked {
		return AnalysisIssue{}, false
	}

	requiredCount := 0
	maxCount := 0
	variadic := false
	for _, param := range method.Params {
		if !param.HasDefault && !param.IsVariadic {
			requiredCount++
		}
		if param.IsVariadic {
			variadic = true
			continue
		}
		maxCount++
	}

	if actualCount < requiredCount {
		return AnalysisIssue{
			Filename: filename,
			Line:     pos.Line,
			Column:   pos.Column,
			Code:     argumentCountRuleCode,
			Message:  fmt.Sprintf("%s invoked with %d %s, at least %d required.", target, actualCount, pluralizeParameters(actualCount), requiredCount),
		}, true
	}
	if !variadic && actualCount > maxCount {
		return AnalysisIssue{
			Filename: filename,
			Line:     pos.Line,
			Column:   pos.Column,
			Code:     argumentCountRuleCode,
			Message:  fmt.Sprintf("%s invoked with %d %s, at most %d allowed.", target, actualCount, pluralizeParameters(actualCount), maxCount),
		}, true
	}

	return AnalysisIssue{}, false
}

func resolveConstructorForNew(node *ast.NewNode, scope *functionScope, ctx *AnalysisContext) (string, ResolvedMethod, bool) {
	if node == nil {
		return "", ResolvedMethod{}, false
	}

	className := node.ClassName
	if className == "" {
		if ident, ok := node.ClassExpr.(*ast.IdentifierNode); ok {
			className = ident.Value
		}
	}
	className = strings.TrimSpace(className)
	if className == "" {
		return "", ResolvedMethod{}, false
	}

	resolvedClassName := className
	if scope != nil {
		resolvedClassName = scope.typeCtx.resolveClassLike(className)
	}

	switch asciiLowerIdent(className) {
	case "self", "static":
		if method, ok := resolveSameClassMethod(scope, "__construct"); ok {
			return strings.TrimPrefix(scope.className, `\`), method, true
		}
		if method, ok := resolveInheritedConstructor(scope.className, scope, ctx); ok {
			return strings.TrimPrefix(scope.className, `\`), method, true
		}
		return strings.TrimPrefix(scope.className, `\`), ResolvedMethod{Name: "__construct"}, true
	}

	if scope != nil && scope.className != "" && strings.EqualFold(strings.TrimPrefix(resolvedClassName, `\`), strings.TrimPrefix(scope.className, `\`)) {
		if method, ok := resolveSameClassMethod(scope, "__construct"); ok {
			return strings.TrimPrefix(scope.className, `\`), method, true
		}
		if method, ok := resolveInheritedConstructor(scope.className, scope, ctx); ok {
			return strings.TrimPrefix(scope.className, `\`), method, true
		}
		return strings.TrimPrefix(scope.className, `\`), ResolvedMethod{Name: "__construct"}, true
	}

	if ctx != nil && ctx.Resolver != nil {
		method, ok := resolveMethodView(ctx.Resolver, resolvedClassName, "__construct")
		if ok {
			return strings.TrimPrefix(resolvedClassName, `\`), method, true
		}
	}

	return "", ResolvedMethod{}, false
}

func resolveInheritedConstructor(className string, scope *functionScope, ctx *AnalysisContext) (ResolvedMethod, bool) {
	className = strings.TrimPrefix(strings.TrimSpace(className), `\`)
	if className == "" {
		return ResolvedMethod{}, false
	}
	if isBuiltinExceptionClass(className, scope, ctx) {
		return builtinExceptionConstructor(), true
	}

	seen := map[string]struct{}{}
	queue := []string{className}
	for len(queue) > 0 {
		current := strings.TrimPrefix(queue[0], `\`)
		queue = queue[1:]
		key := asciiLowerIdent(current)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		resolved, ok := resolveHierarchyClass(current, scope, ctx)
		if !ok {
			continue
		}
		for _, parent := range resolved.Extends {
			parent = canonicalClassName(parent, scope, ctx)
			if parent == "" {
				continue
			}
			if ctx != nil && ctx.Resolver != nil {
				if method, ok := resolveMethodView(ctx.Resolver, parent, "__construct"); ok {
					return method, true
				}
			}
			if isBuiltinExceptionClass(parent, scope, ctx) {
				return builtinExceptionConstructor(), true
			}
			queue = append(queue, parent)
		}
	}

	return ResolvedMethod{}, false
}

func isBuiltinExceptionClass(className string, scope *functionScope, ctx *AnalysisContext) bool {
	className = strings.TrimPrefix(strings.TrimSpace(className), `\`)
	return strings.EqualFold(className, "Exception") || classHierarchyCompatible("Exception", className, scope, ctx)
}

func builtinExceptionConstructor() ResolvedMethod {
	return ResolvedMethod{
		Name: "__construct",
		Params: []ResolvedParam{
			{Name: "message", Type: "string", HasDefault: true},
			{Name: "code", Type: "int", HasDefault: true},
			{Name: "previous", Type: "Throwable", HasDefault: true},
		},
	}
}

func countCallArguments(args []ast.Node) (int, bool) {
	count := 0
	for _, arg := range args {
		if _, ok := arg.(*ast.UnpackedArgumentNode); ok {
			return 0, true
		}
		count++
	}
	return count, false
}

func pluralizeParameters(count int) string {
	if count == 1 {
		return "parameter"
	}
	return "parameters"
}

func init() {
	RegisterAnalysisRuleWithContext(argumentCountRuleCode, func(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
		rule := &ArgumentCountRule{}
		return rule.CheckIssues(nodes, filename, ctx)
	})
}
