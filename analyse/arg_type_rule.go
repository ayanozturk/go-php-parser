package analyse

import (
	"fmt"
	"go-phpcs/ast"
	"strings"
)

type ArgumentTypeRule struct{}

func (r *ArgumentTypeRule) CheckIssues(nodes []ast.Node, filename string, ctx *AnalysisContext) []AnalysisIssue {
	var issues []AnalysisIssue
	fileCtx := collectFileTypeContext(nodes)
	var walk func(node ast.Node, class *ast.ClassNode, scope *functionScope)

	walk = func(node ast.Node, class *ast.ClassNode, scope *functionScope) {
		switch n := node.(type) {
		case *ast.ClassNode:
			for _, methodNode := range n.Methods {
				walk(methodNode, n, nil)
			}
		case *ast.FunctionNode:
			fnScope := newFunctionScope(class, n, fileCtx)
			walkStatementsForArgTypes(n.Body, fnScope, ctx, filename, &issues)
		case *ast.NamespaceNode:
			for _, child := range n.Body {
				walk(child, class, scope)
			}
		}
	}

	for _, node := range nodes {
		walk(node, nil, nil)
	}

	return issues
}

func walkStatementsForArgTypes(nodes []ast.Node, scope *functionScope, ctx *AnalysisContext, filename string, issues *[]AnalysisIssue) {
	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.ExpressionStmt:
			walkExprForArgTypes(n.Expr, scope, ctx, filename, issues)
			applyExpressionScope(scope, n.Expr, ctx)
		case *ast.AssignmentNode:
			walkExprForArgTypes(n.Right, scope, ctx, filename, issues)
			applyAssignmentScope(scope, n, ctx)
		case *ast.ReturnNode:
			walkExprForArgTypes(n.Expr, scope, ctx, filename, issues)
		case *ast.IfNode:
			walkExprForArgTypes(n.Condition, scope, ctx, filename, issues)
			walkStatementsForArgTypes(n.Body, scope.clone(), ctx, filename, issues)
			for _, elseif := range n.ElseIfs {
				walkExprForArgTypes(elseif.Condition, scope, ctx, filename, issues)
				walkStatementsForArgTypes(elseif.Body, scope.clone(), ctx, filename, issues)
			}
			if n.Else != nil {
				walkStatementsForArgTypes(n.Else.Body, scope.clone(), ctx, filename, issues)
			}
			applyTerminatingIfFalseScope(scope, n)
		case *ast.BlockNode:
			walkStatementsForArgTypes(n.Statements, scope.clone(), ctx, filename, issues)
		case *ast.WhileNode:
			walkExprForArgTypes(n.Condition, scope, ctx, filename, issues)
			walkStatementsForArgTypes(n.Body, scope.clone(), ctx, filename, issues)
		case *ast.ForeachNode:
			walkExprForArgTypes(n.Expr, scope, ctx, filename, issues)
			walkStatementsForArgTypes(n.Body, scope.clone(), ctx, filename, issues)
		}
	}
}

func applyTerminatingIfFalseScope(scope *functionScope, node *ast.IfNode) {
	if scope == nil || node == nil || node.Else != nil || len(node.ElseIfs) > 0 {
		return
	}
	if !statementsTerminate(node.Body) {
		return
	}
	for _, variableName := range variablesNonNullWhenFalse(node.Condition) {
		current, ok := scope.variables[variableName]
		if !ok {
			continue
		}
		refined := current.withoutBuiltin("null")
		if !refined.IsEmpty() {
			scope.variables[variableName] = refined
		}
	}
}

func variablesNonNullWhenFalse(node ast.Node) []string {
	switch n := node.(type) {
	case *ast.BinaryExpr:
		switch n.Operator {
		case "||", "or":
			left := variablesNonNullWhenFalse(n.Left)
			return append(left, variablesNonNullWhenFalse(n.Right)...)
		case "==", "===":
			if name, ok := nullComparisonVariable(n.Left, n.Right); ok {
				return []string{name}
			}
		}
	case *ast.UnaryExpr:
		if n.Operator == "!" {
			if variable, ok := n.Operand.(*ast.VariableNode); ok {
				return []string{variable.Name}
			}
		}
	}
	return nil
}

func nullComparisonVariable(left, right ast.Node) (string, bool) {
	if isNullLiteral(left) {
		if variable, ok := right.(*ast.VariableNode); ok {
			return variable.Name, true
		}
	}
	if isNullLiteral(right) {
		if variable, ok := left.(*ast.VariableNode); ok {
			return variable.Name, true
		}
	}
	return "", false
}

func isNullLiteral(node ast.Node) bool {
	switch node.(type) {
	case *ast.NullLiteral, *ast.NullNode:
		return true
	default:
		return false
	}
}

func walkExprForArgTypes(node ast.Node, scope *functionScope, ctx *AnalysisContext, filename string, issues *[]AnalysisIssue) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *ast.MethodCallNode:
		checkMethodCallArgTypes(n, scope, ctx, filename, issues)
		walkExprForArgTypes(n.Object, scope, ctx, filename, issues)
		for _, arg := range n.Args {
			walkExprForArgTypes(argumentValue(arg), scope, ctx, filename, issues)
		}
	case *ast.FunctionCallNode:
		for _, arg := range n.Args {
			walkExprForArgTypes(argumentValue(arg), scope, ctx, filename, issues)
		}
	case *ast.AssignmentNode:
		walkExprForArgTypes(n.Right, scope, ctx, filename, issues)
	case *ast.PropertyFetchNode:
		walkExprForArgTypes(n.Object, scope, ctx, filename, issues)
	case *ast.BinaryExpr:
		walkExprForArgTypes(n.Left, scope, ctx, filename, issues)
		walkExprForArgTypes(n.Right, scope, ctx, filename, issues)
	case *ast.ConcatNode:
		for _, part := range n.Parts {
			walkExprForArgTypes(part, scope, ctx, filename, issues)
		}
	case *ast.TernaryExpr:
		walkExprForArgTypes(n.Condition, scope, ctx, filename, issues)
		walkExprForArgTypes(n.IfTrue, scope, ctx, filename, issues)
		walkExprForArgTypes(n.IfFalse, scope, ctx, filename, issues)
	case *ast.NewNode:
		for _, arg := range n.Args {
			walkExprForArgTypes(argumentValue(arg), scope, ctx, filename, issues)
		}
	case *ast.NamedArgumentNode:
		walkExprForArgTypes(n.Value, scope, ctx, filename, issues)
	case *ast.UnpackedArgumentNode:
		walkExprForArgTypes(n.Expr, scope, ctx, filename, issues)
	}
}

func checkMethodCallArgTypes(call *ast.MethodCallNode, scope *functionScope, ctx *AnalysisContext, filename string, issues *[]AnalysisIssue) {
	method, ok := resolveMethodForCall(call, scope, ctx)
	if !ok || len(method.Params) == 0 {
		return
	}

	for idx, argNode := range call.Args {
		if idx >= len(method.Params) {
			return
		}

		argExpr := argumentValue(argNode)
		if argExpr == nil {
			continue
		}

		expected := ParseType(method.Params[idx].Type)
		if expected.IsEmpty() {
			continue
		}
		actual := inferType(argExpr, scope, ctx)
		if expected.AcceptsWithContext(actual, scope, ctx) {
			continue
		}

		actualLabel := actual.String()
		if actualLabel == "" {
			actualLabel = "mixed"
		}
		pos := argExpr.GetPos()
		*issues = append(*issues, AnalysisIssue{
			Filename: filename,
			Line:     pos.Line,
			Column:   pos.Column,
			Code:     "A.ARG.TYPE",
			Message:  fmt.Sprintf("Method %s argument %d expects %s, got %s", method.Name, idx+1, expected.String(), actualLabel),
		})
	}
}

func resolveMethodForCall(call *ast.MethodCallNode, scope *functionScope, ctx *AnalysisContext) (ResolvedMethod, bool) {
	if call == nil {
		return ResolvedMethod{}, false
	}
	if object, ok := call.Object.(*ast.VariableNode); ok && object.Name == "this" {
		if method, ok := resolveSameClassMethod(scope, call.Method); ok {
			return method, true
		}
	}

	objectType := inferType(call.Object, scope, ctx)
	className, ok := objectType.SingleClassName()
	if !ok {
		return ResolvedMethod{}, false
	}
	if scope != nil && scope.className != "" && strings.EqualFold(className, scope.className) {
		if method, ok := resolveSameClassMethod(scope, call.Method); ok {
			return method, true
		}
	}
	if ctx != nil && ctx.Resolver != nil {
		return ctx.Resolver.ResolveMethod(className, call.Method)
	}
	return ResolvedMethod{}, false
}

func argumentValue(node ast.Node) ast.Node {
	switch n := node.(type) {
	case *ast.NamedArgumentNode:
		return n.Value
	case *ast.UnpackedArgumentNode:
		return n.Expr
	default:
		return node
	}
}

func init() {
	RegisterAnalysisRuleWithContext("A.ARG.TYPE", func(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
		rule := &ArgumentTypeRule{}
		return rule.CheckIssues(nodes, filename, ctx)
	})
}
