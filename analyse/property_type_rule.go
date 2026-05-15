package analyse

import (
	"fmt"
	"go-phpcs/ast"
	"strings"
)

type PropertyTypeRule struct{}

func (r *PropertyTypeRule) CheckIssues(nodes []ast.Node, filename string, ctx *AnalysisContext) []AnalysisIssue {
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
			walkStatementsForPropertyTypes(n.Body, scope, ctx, filename, &issues)
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

func walkStatementsForPropertyTypes(nodes []ast.Node, scope *functionScope, ctx *AnalysisContext, filename string, issues *[]AnalysisIssue) {
	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.ExpressionStmt:
			walkExprForPropertyTypes(n.Expr, scope, ctx, filename, issues)
			applyExpressionScope(scope, n.Expr, ctx)
		case *ast.AssignmentNode:
			walkAssignmentForPropertyTypes(n, scope, ctx, filename, issues)
			applyAssignmentScope(scope, n, ctx)
		case *ast.ReturnNode:
			walkExprForPropertyTypes(n.Expr, scope, ctx, filename, issues)
		case *ast.IfNode:
			walkExprForPropertyTypes(n.Condition, scope, ctx, filename, issues)
			walkStatementsForPropertyTypes(n.Body, scope.clone(), ctx, filename, issues)
			for _, elseif := range n.ElseIfs {
				walkExprForPropertyTypes(elseif.Condition, scope, ctx, filename, issues)
				walkStatementsForPropertyTypes(elseif.Body, scope.clone(), ctx, filename, issues)
			}
			if n.Else != nil {
				walkStatementsForPropertyTypes(n.Else.Body, scope.clone(), ctx, filename, issues)
			}
		case *ast.BlockNode:
			walkStatementsForPropertyTypes(n.Statements, scope.clone(), ctx, filename, issues)
		case *ast.WhileNode:
			walkExprForPropertyTypes(n.Condition, scope, ctx, filename, issues)
			walkStatementsForPropertyTypes(n.Body, scope.clone(), ctx, filename, issues)
		case *ast.ForeachNode:
			walkExprForPropertyTypes(n.Expr, scope, ctx, filename, issues)
			walkStatementsForPropertyTypes(n.Body, scope.clone(), ctx, filename, issues)
		}
	}
}

func walkExprForPropertyTypes(node ast.Node, scope *functionScope, ctx *AnalysisContext, filename string, issues *[]AnalysisIssue) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *ast.AssignmentNode:
		walkAssignmentForPropertyTypes(n, scope, ctx, filename, issues)
	case *ast.MethodCallNode:
		walkExprForPropertyTypes(n.Object, scope, ctx, filename, issues)
		for _, arg := range n.Args {
			walkExprForPropertyTypes(argumentValue(arg), scope, ctx, filename, issues)
		}
	case *ast.FunctionCallNode:
		for _, arg := range n.Args {
			walkExprForPropertyTypes(argumentValue(arg), scope, ctx, filename, issues)
		}
	case *ast.PropertyFetchNode:
		walkExprForPropertyTypes(n.Object, scope, ctx, filename, issues)
	case *ast.BinaryExpr:
		walkExprForPropertyTypes(n.Left, scope, ctx, filename, issues)
		walkExprForPropertyTypes(n.Right, scope, ctx, filename, issues)
	case *ast.ConcatNode:
		for _, part := range n.Parts {
			walkExprForPropertyTypes(part, scope, ctx, filename, issues)
		}
	case *ast.TernaryExpr:
		walkExprForPropertyTypes(n.Condition, scope, ctx, filename, issues)
		walkExprForPropertyTypes(n.IfTrue, scope, ctx, filename, issues)
		walkExprForPropertyTypes(n.IfFalse, scope, ctx, filename, issues)
	case *ast.NewNode:
		for _, arg := range n.Args {
			walkExprForPropertyTypes(argumentValue(arg), scope, ctx, filename, issues)
		}
	case *ast.NamedArgumentNode:
		walkExprForPropertyTypes(n.Value, scope, ctx, filename, issues)
	case *ast.UnpackedArgumentNode:
		walkExprForPropertyTypes(n.Expr, scope, ctx, filename, issues)
	}
}

func walkAssignmentForPropertyTypes(assign *ast.AssignmentNode, scope *functionScope, ctx *AnalysisContext, filename string, issues *[]AnalysisIssue) {
	if assign == nil {
		return
	}

	walkExprForPropertyTypes(assign.Right, scope, ctx, filename, issues)

	propertyFetch, ok := assign.Left.(*ast.PropertyFetchNode)
	if !ok {
		return
	}

	expected, propertyName, ok := resolvePropertyTypeForAssignment(propertyFetch, scope, ctx)
	if !ok || expected.IsEmpty() {
		return
	}

	actual := inferType(assign.Right, scope, ctx)
	if expected.AcceptsWithContext(actual, scope, ctx) {
		return
	}

	actualLabel := actual.String()
	if actualLabel == "" {
		actualLabel = "mixed"
	}
	pos := assign.Right.GetPos()
	*issues = append(*issues, AnalysisIssue{
		Filename: filename,
		Line:     pos.Line,
		Column:   pos.Column,
		Code:     "A.PROP.TYPE",
		Message:  fmt.Sprintf("Property %s expects %s, got %s", propertyName, expected.String(), actualLabel),
	})
}

func resolvePropertyTypeForAssignment(fetch *ast.PropertyFetchNode, scope *functionScope, ctx *AnalysisContext) (Type, string, bool) {
	if fetch == nil {
		return EmptyType(), "", false
	}

	if object, ok := fetch.Object.(*ast.VariableNode); ok && object.Name == "this" {
		if propertyType, ok := resolveSameClassPropertyType(scope, fetch.Property); ok {
			return propertyType, "$this->" + fetch.Property, true
		}
	}

	objectType := inferType(fetch.Object, scope, ctx)
	className, ok := objectType.SingleClassName()
	if !ok {
		return EmptyType(), "", false
	}
	if scope != nil && scope.className != "" && strings.EqualFold(className, scope.className) {
		if propertyType, ok := resolveSameClassPropertyType(scope, fetch.Property); ok {
			return propertyType, className + "::$" + fetch.Property, true
		}
	}
	if ctx != nil && ctx.Resolver != nil {
		if property, ok := ctx.Resolver.ResolveProperty(className, fetch.Property); ok {
			return ParseType(property.Type), className + "::$" + property.Name, true
		}
	}

	return EmptyType(), "", false
}

func init() {
	RegisterAnalysisRuleWithContext("A.PROP.TYPE", func(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
		rule := &PropertyTypeRule{}
		return rule.CheckIssues(nodes, filename, ctx)
	})
}
