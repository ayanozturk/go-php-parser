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
			walkStatementsForArgTypes(n.Body, scopeForConditionTrue(scope, n.Condition), ctx, filename, issues)
			for _, elseif := range n.ElseIfs {
				walkExprForArgTypes(elseif.Condition, scope, ctx, filename, issues)
				walkStatementsForArgTypes(elseif.Body, scopeForConditionTrue(scope, elseif.Condition), ctx, filename, issues)
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

func scopeForConditionTrue(scope *functionScope, condition ast.Node) *functionScope {
	refined := scope.clone()
	if refined == nil {
		return nil
	}
	applyConditionTrueScope(refined, condition)
	return refined
}

func applyConditionTrueScope(scope *functionScope, condition ast.Node) {
	if scope == nil {
		return
	}
	for variableName, typ := range variablesTypedWhenTrue(condition, scope) {
		if !typ.IsEmpty() {
			scope.variables[variableName] = typ
		}
	}
}

func variablesTypedWhenTrue(node ast.Node, scope *functionScope) map[string]Type {
	switch n := node.(type) {
	case *ast.BinaryExpr:
		switch n.Operator {
		case "&&", "and":
			types := variablesTypedWhenTrue(n.Left, scope)
			for name, typ := range variablesTypedWhenTrue(n.Right, scope) {
				types[name] = typ
			}
			return types
		case "instanceof":
			if variable, ok := n.Left.(*ast.VariableNode); ok {
				if typ := typeFromInstanceofTarget(n.Right, scope); !typ.IsEmpty() {
					return map[string]Type{variable.Name: typ}
				}
			}
		case "==", "===":
			if name, ok := nullComparisonVariable(n.Left, n.Right); ok {
				return map[string]Type{name: ParseType("null")}
			}
		}
	case *ast.FunctionCallNode:
		if variableName, typ, ok := builtinTypePredicate(n); ok {
			return map[string]Type{variableName: typ}
		}
	}
	return map[string]Type{}
}

func builtinTypePredicate(call *ast.FunctionCallNode) (string, Type, bool) {
	if call == nil || len(call.Args) != 1 {
		return "", EmptyType(), false
	}

	nameNode, ok := call.Name.(*ast.IdentifierNode)
	if !ok {
		return "", EmptyType(), false
	}

	variable, ok := argumentValue(call.Args[0]).(*ast.VariableNode)
	if !ok {
		return "", EmptyType(), false
	}

	name := strings.TrimLeft(nameNode.Value, "\\")
	if idx := strings.LastIndex(name, "\\"); idx != -1 {
		name = name[idx+1:]
	}

	switch strings.ToLower(name) {
	case "is_string":
		return variable.Name, ParseType("string"), true
	case "is_int", "is_integer", "is_long":
		return variable.Name, ParseType("int"), true
	case "is_float", "is_double", "is_real":
		return variable.Name, ParseType("float"), true
	case "is_bool":
		return variable.Name, ParseType("bool"), true
	case "is_array":
		return variable.Name, ParseType("array"), true
	case "is_object":
		return variable.Name, ParseType("object"), true
	case "is_null":
		return variable.Name, ParseType("null"), true
	}

	return "", EmptyType(), false
}

func typeFromInstanceofTarget(node ast.Node, scope *functionScope) Type {
	identifier, ok := node.(*ast.IdentifierNode)
	if !ok {
		return EmptyType()
	}
	className := identifier.Value
	if scope != nil {
		className = scope.typeCtx.resolveClassLike(className)
	}
	return ClassType(className)
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
		checkNewArgTypes(n, scope, ctx, filename, issues)
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
	checkResolvedCallArgTypes(fmt.Sprintf("Method %s", method.Name), method, call.Args, scope, ctx, filename, issues)
}

func checkNewArgTypes(node *ast.NewNode, scope *functionScope, ctx *AnalysisContext, filename string, issues *[]AnalysisIssue) {
	className, method, ok := resolveConstructorForNew(node, scope, ctx)
	if !ok || len(method.Params) == 0 {
		return
	}
	checkResolvedCallArgTypes(fmt.Sprintf("Class %s constructor", className), method, node.Args, scope, ctx, filename, issues)
}

func checkResolvedCallArgTypes(target string, method ResolvedMethod, args []ast.Node, scope *functionScope, ctx *AnalysisContext, filename string, issues *[]AnalysisIssue) {
	if len(method.Params) == 0 {
		return
	}

	nameToIndex := make(map[string]int, len(method.Params))
	variadicIndex := -1
	for idx, param := range method.Params {
		nameToIndex[strings.ToLower(param.Name)] = idx
		if param.IsVariadic {
			variadicIndex = idx
		}
	}

	usedParams := map[int]struct{}{}
	nextPositionalParam := 0

	for _, argNode := range args {
		if _, ok := argNode.(*ast.UnpackedArgumentNode); ok {
			// Cannot infer concrete type count/order for unpacked arguments.
			return
		}

		argExpr := argumentValue(argNode)
		if argExpr == nil {
			continue
		}

		paramIndex := -1
		argLabel := ""

		if namedArg, ok := argNode.(*ast.NamedArgumentNode); ok {
			idx, ok := nameToIndex[strings.ToLower(namedArg.Name)]
			if !ok {
				continue
			}
			paramIndex = idx
			argLabel = "$" + method.Params[idx].Name
		} else {
			for nextPositionalParam < len(method.Params) {
				if _, alreadyUsed := usedParams[nextPositionalParam]; alreadyUsed {
					nextPositionalParam++
					continue
				}
				break
			}

			if nextPositionalParam < len(method.Params) {
				paramIndex = nextPositionalParam
				nextPositionalParam++
			} else if variadicIndex >= 0 {
				paramIndex = variadicIndex
			} else {
				return
			}
			argLabel = fmt.Sprintf("%d", paramIndex+1)
		}

		if paramIndex < 0 || paramIndex >= len(method.Params) {
			continue
		}

		param := method.Params[paramIndex]
		expected := ParseType(param.Type)
		if expected.IsEmpty() {
			usedParams[paramIndex] = struct{}{}
			continue
		}

		actual := inferType(argExpr, scope, ctx)
		if expected.AcceptsWithContext(actual, scope, ctx) {
			usedParams[paramIndex] = struct{}{}
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
			Message:  fmt.Sprintf("%s argument %s expects %s, got %s", target, argLabel, expected.String(), actualLabel),
		})

		usedParams[paramIndex] = struct{}{}
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
