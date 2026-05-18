package analyse

import (
	"fmt"
	"go-phpcs/ast"
	"strings"
)

const argumentCountRuleCode = "A.ARG.COUNT"

type ArgumentCountRule struct{}

func (r *ArgumentCountRule) CheckIssues(nodes []ast.Node, filename string, ctx *AnalysisContext) []AnalysisIssue {
	var issues []AnalysisIssue
	fileCtx := collectFileTypeContext(nodes)
	var walk func(node ast.Node, class *ast.ClassNode)

	walk = func(node ast.Node, class *ast.ClassNode) {
		switch n := node.(type) {
		case *ast.ClassNode:
			for _, methodNode := range n.Methods {
				walk(methodNode, n)
			}
		case *ast.FunctionNode:
			scope := newFunctionScope(class, n, fileCtx)
			walkStatementsForArgCounts(n.Body, scope, ctx, filename, &issues)
		case *ast.NamespaceNode:
			for _, child := range n.Body {
				walk(child, class)
			}
		}
	}

	for _, node := range nodes {
		walk(node, nil)
	}

	return issues
}

func walkStatementsForArgCounts(nodes []ast.Node, scope *functionScope, ctx *AnalysisContext, filename string, issues *[]AnalysisIssue) {
	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.ExpressionStmt:
			walkExprForArgCounts(n.Expr, scope, ctx, filename, issues)
			applyExpressionScope(scope, n.Expr, ctx)
		case *ast.AssignmentNode:
			walkExprForArgCounts(n.Right, scope, ctx, filename, issues)
			applyAssignmentScope(scope, n, ctx)
		case *ast.ReturnNode:
			walkExprForArgCounts(n.Expr, scope, ctx, filename, issues)
		case *ast.IfNode:
			walkExprForArgCounts(n.Condition, scope, ctx, filename, issues)
			walkStatementsForArgCounts(n.Body, scope.clone(), ctx, filename, issues)
			for _, elseif := range n.ElseIfs {
				walkExprForArgCounts(elseif.Condition, scope, ctx, filename, issues)
				walkStatementsForArgCounts(elseif.Body, scope.clone(), ctx, filename, issues)
			}
			if n.Else != nil {
				walkStatementsForArgCounts(n.Else.Body, scope.clone(), ctx, filename, issues)
			}
		case *ast.BlockNode:
			walkStatementsForArgCounts(n.Statements, scope.clone(), ctx, filename, issues)
		case *ast.WhileNode:
			walkExprForArgCounts(n.Condition, scope, ctx, filename, issues)
			walkStatementsForArgCounts(n.Body, scope.clone(), ctx, filename, issues)
		case *ast.ForeachNode:
			walkExprForArgCounts(n.Expr, scope, ctx, filename, issues)
			walkStatementsForArgCounts(n.Body, scope.clone(), ctx, filename, issues)
		}
	}
}

func walkExprForArgCounts(node ast.Node, scope *functionScope, ctx *AnalysisContext, filename string, issues *[]AnalysisIssue) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *ast.MethodCallNode:
		checkMethodCallArgCount(n, scope, ctx, filename, issues)
		walkExprForArgCounts(n.Object, scope, ctx, filename, issues)
		for _, arg := range n.Args {
			walkExprForArgCounts(argumentValue(arg), scope, ctx, filename, issues)
		}
	case *ast.FunctionCallNode:
		for _, arg := range n.Args {
			walkExprForArgCounts(argumentValue(arg), scope, ctx, filename, issues)
		}
	case *ast.AssignmentNode:
		walkExprForArgCounts(n.Right, scope, ctx, filename, issues)
	case *ast.PropertyFetchNode:
		walkExprForArgCounts(n.Object, scope, ctx, filename, issues)
	case *ast.BinaryExpr:
		walkExprForArgCounts(n.Left, scope, ctx, filename, issues)
		walkExprForArgCounts(n.Right, scope, ctx, filename, issues)
	case *ast.ConcatNode:
		for _, part := range n.Parts {
			walkExprForArgCounts(part, scope, ctx, filename, issues)
		}
	case *ast.TernaryExpr:
		walkExprForArgCounts(n.Condition, scope, ctx, filename, issues)
		walkExprForArgCounts(n.IfTrue, scope, ctx, filename, issues)
		walkExprForArgCounts(n.IfFalse, scope, ctx, filename, issues)
	case *ast.NewNode:
		checkNewArgCount(n, scope, ctx, filename, issues)
		for _, arg := range n.Args {
			walkExprForArgCounts(argumentValue(arg), scope, ctx, filename, issues)
		}
	case *ast.NamedArgumentNode:
		walkExprForArgCounts(n.Value, scope, ctx, filename, issues)
	case *ast.UnpackedArgumentNode:
		walkExprForArgCounts(n.Expr, scope, ctx, filename, issues)
	}
}

func checkMethodCallArgCount(call *ast.MethodCallNode, scope *functionScope, ctx *AnalysisContext, filename string, issues *[]AnalysisIssue) {
	method, ok := resolveMethodForCall(call, scope, ctx)
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

	switch strings.ToLower(className) {
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
		method, ok := ctx.Resolver.ResolveMethod(resolvedClassName, "__construct")
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
		key := strings.ToLower(current)
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
				if method, ok := ctx.Resolver.ResolveMethod(parent, "__construct"); ok {
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
