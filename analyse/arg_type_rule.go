package analyse

import (
	"fmt"
	"github.com/ayanozturk/go-php-parser/ast"
	"strings"
)

type ArgumentTypeRule struct{}

type semanticExpressionObserver func(filename string, expr ast.Node, scope *functionScope, ctx *AnalysisContext)

func (r *ArgumentTypeRule) CheckIssues(nodes []ast.Node, filename string, ctx *AnalysisContext) []AnalysisIssue {
	ctx = ensureArgCallDiagnostics(filename, nodes, ctx)
	return ctx.argTypeIssues
}

func ensureArgCallDiagnostics(filename string, nodes []ast.Node, ctx *AnalysisContext) *AnalysisContext {
	if ctx == nil {
		ctx = &AnalysisContext{}
	}
	if ctx.hasArgCallDiagnostics {
		return ctx
	}
	var typeIssues []AnalysisIssue
	var typeIssueSink *[]AnalysisIssue
	if analysisLevelAtLeast(ctx, 5) {
		typeIssueSink = &typeIssues
	}
	ctx.argCountSink = &ctx.argCountIssues
	ctx.deprecatedCallSink = &ctx.deprecatedCallIssues
	ctx.deprecatedCallSeen = make(map[ast.Node]struct{})
	if !ctx.hasAssignmentTypeIssues {
		ctx.assignmentTypeSink = &ctx.assignmentTypeIssues
	}
	var observe semanticExpressionObserver
	if ctx.Resolver != nil && analysisLevelAtLeast(ctx, 2) {
		observe = func(filename string, expr ast.Node, scope *functionScope, ctx *AnalysisContext) {
			call, ok := expr.(*ast.MethodCallNode)
			if !ok {
				return
			}
			appendMethodReceiverIssuesForCall(filename, call, scope, ctx, &ctx.methodReceiverIssues)
		}
	}
	fileCtx := analysisFileTypeContext(ctx, nodes)
	walkArgCallDeclarations(nodes, nil, ctx, filename, fileCtx, typeIssueSink, observe)
	ctx.argCountSink = nil
	ctx.deprecatedCallSink = nil
	ctx.deprecatedCallSeen = nil
	ctx.assignmentTypeSink = nil
	if observe != nil {
		ctx.hasMethodReceiverIssues = true
	}
	ctx.argTypeIssues = typeIssues
	ctx.hasArgCallDiagnostics = true
	ctx.hasAssignmentTypeIssues = true
	return ctx
}

func walkArgCallDeclarations(nodes []ast.Node, class *ast.ClassNode, ctx *AnalysisContext, filename string, fileCtx FileTypeContext, typeIssueSink *[]AnalysisIssue, observe semanticExpressionObserver) {
	var fileScope *functionScope
	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.NamespaceNode:
			walkArgCallDeclarations(n.Body, class, ctx, filename, fileCtx, typeIssueSink, observe)
		case *ast.ClassNode:
			for _, property := range n.Properties {
				walkArgCallProperty(property, n, ctx, filename, fileCtx, typeIssueSink, observe)
			}
			walkArgCallDeclarations(n.Methods, n, ctx, filename, fileCtx, typeIssueSink, observe)
		case *ast.InterfaceNode:
			walkArgCallDeclarations(n.Members, class, ctx, filename, fileCtx, typeIssueSink, observe)
		case *ast.TraitNode:
			traitClass := class
			if n.Name != nil {
				traitClass = &ast.ClassNode{Name: n.Name.Name}
			}
			walkArgCallDeclarations(n.Body, traitClass, ctx, filename, fileCtx, typeIssueSink, observe)
		case *ast.EnumNode:
			enumClass := &ast.ClassNode{Name: n.Name}
			walkArgCallDeclarations(n.Methods, enumClass, ctx, filename, fileCtx, typeIssueSink, observe)
		case *ast.FunctionNode:
			fnScope := analysisFunctionScope(ctx, class, n, fileCtx)
			walkStatementsForArgTypesUsing(n.Body, fnScope, ctx, filename, typeIssueSink, observe)
		case *ast.PropertyNode:
			walkArgCallProperty(n, class, ctx, filename, fileCtx, typeIssueSink, observe)
		default:
			if fileScope == nil {
				fileScope = newFunctionScopeWithContext(ctx, class, &ast.FunctionNode{}, fileCtx)
			}
			walkStatementsForArgTypesUsing([]ast.Node{node}, fileScope, ctx, filename, typeIssueSink, observe)
		}
	}
}

func walkArgCallProperty(property ast.Node, class *ast.ClassNode, ctx *AnalysisContext, filename string, fileCtx FileTypeContext, typeIssueSink *[]AnalysisIssue, observe semanticExpressionObserver) {
	prop, ok := property.(*ast.PropertyNode)
	if !ok {
		return
	}
	scope := newFunctionScopeWithContext(ctx, class, &ast.FunctionNode{}, fileCtx)
	walkExprForArgTypesUsing(prop.DefaultValue, scope, ctx, filename, typeIssueSink, observe)
	for _, hook := range prop.Hooks {
		walkExprForArgTypesUsing(hook.Expr, scope, ctx, filename, typeIssueSink, observe)
		walkStatementsForArgTypesUsing(hook.Body, scope, ctx, filename, typeIssueSink, observe)
	}
}

func walkStatementsForArgTypes(nodes []ast.Node, scope *functionScope, ctx *AnalysisContext, filename string, issues *[]AnalysisIssue) {
	walkStatementsForArgTypesUsing(nodes, scope, ctx, filename, issues, nil)
}

func walkStatementsForArgTypesUsing(nodes []ast.Node, scope *functionScope, ctx *AnalysisContext, filename string, issues *[]AnalysisIssue, observe semanticExpressionObserver) {
	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.ExpressionStmt:
			walkExprForArgTypesUsing(n.Expr, scope, ctx, filename, issues, observe)
			applyExpressionScope(scope, n.Expr, ctx)
		case *ast.NamespaceNode:
			walkStatementsForArgTypesUsing(n.Body, scope, ctx, filename, issues, observe)
		case *ast.AssignmentNode:
			walkExprForArgTypesUsing(n.Right, scope, ctx, filename, issues, observe)
			observeSemanticExpression(filename, n.Right, scope, ctx, observe)
			if observe != nil {
				assignedScope := scope.clone()
				applyAssignmentScope(assignedScope, n, ctx)
				walkExprForArgTypesUsing(n.Left, assignedScope, ctx, filename, issues, observe)
			}
			recordAssignmentTypeIssues(n, scope, ctx, filename)
			applyAssignmentScope(scope, n, ctx)
		case *ast.ReturnNode:
			walkExprForArgTypesUsing(n.Expr, scope, ctx, filename, issues, observe)
			observeSemanticExpression(filename, n.Expr, scope, ctx, observe)
		case *ast.BreakNode:
			walkExprForArgTypesUsing(n.Level, scope, ctx, filename, issues, observe)
		case *ast.ContinueNode:
			walkExprForArgTypesUsing(n.Level, scope, ctx, filename, issues, observe)
		case *ast.IfNode:
			walkExprForArgTypesUsing(n.Condition, scope, ctx, filename, issues, observe)
			thenScope := scopeForConditionTrue(scope, n.Condition)
			walkStatementsForArgTypesUsing(n.Body, thenScope, ctx, filename, issues, observe)
			elseifScopes := make([]*functionScope, len(n.ElseIfs))
			for i, elseif := range n.ElseIfs {
				walkExprForArgTypesUsing(elseif.Condition, scope, ctx, filename, issues, observe)
				elseifScopes[i] = scopeForConditionTrue(scope, elseif.Condition)
				walkStatementsForArgTypesUsing(elseif.Body, elseifScopes[i], ctx, filename, issues, observe)
			}
			var elseScope *functionScope
			if n.Else != nil {
				elseScope = scopeForConditionFalse(scope, n.Condition)
				walkStatementsForArgTypesUsing(n.Else.Body, elseScope, ctx, filename, issues, observe)
			}
			finishIfNodeScope(scope, n, thenScope, elseifScopes, elseScope, ctx)
		case *ast.BlockNode:
			walkStatementsForArgTypesUsing(n.Statements, scope.clone(), ctx, filename, issues, observe)
		case *ast.WhileNode:
			walkExprForArgTypesUsing(n.Condition, scope, ctx, filename, issues, observe)
			walkStatementsForArgTypesUsing(n.Body, scope.clone(), ctx, filename, issues, observe)
		case *ast.DoWhileNode:
			loopScope := scope.clone()
			walkStatementsForArgTypesUsing(n.Body, loopScope, ctx, filename, issues, observe)
			walkExprForArgTypesUsing(n.Condition, loopScope, ctx, filename, issues, observe)
		case *ast.ForNode:
			for _, expression := range n.Init {
				walkExprForArgTypesUsing(expression, scope, ctx, filename, issues, observe)
				applyExpressionScope(scope, expression, ctx)
			}
			loopScope := scope.clone()
			for _, condition := range n.Conditions {
				walkExprForArgTypesUsing(condition, loopScope, ctx, filename, issues, observe)
			}
			walkStatementsForArgTypesUsing(n.Body, loopScope, ctx, filename, issues, observe)
			for _, update := range n.Updates {
				walkExprForArgTypesUsing(update, loopScope, ctx, filename, issues, observe)
			}
		case *ast.ForeachNode:
			walkExprForArgTypesUsing(n.Expr, scope, ctx, filename, issues, observe)
			loopScope := scope.clone()
			applyForeachIterationTypes(loopScope, n, ctx, filename)
			walkStatementsForArgTypesUsing(n.Body, loopScope, ctx, filename, issues, observe)
			joinLoopAssignedVariables(scope, loopScope)
		case *ast.ThrowNode:
			walkExprForArgTypesUsing(n.Expr, scope, ctx, filename, issues, observe)
		case *ast.TryNode:
			walkStatementsForArgTypesUsing(n.Body, scope.clone(), ctx, filename, issues, observe)
			for _, catchNode := range n.Catches {
				walkStatementsForArgTypesUsing(catchNode.Body, scope.clone(), ctx, filename, issues, observe)
			}
			walkStatementsForArgTypesUsing(n.Finally, scope.clone(), ctx, filename, issues, observe)
		case *ast.SwitchNode:
			walkExprForArgTypesUsing(n.Expr, scope, ctx, filename, issues, observe)
			for _, switchCase := range n.Cases {
				walkExprForArgTypesUsing(switchCase.Expr, scope, ctx, filename, issues, observe)
				walkStatementsForArgTypesUsing(switchCase.Body, scope.clone(), ctx, filename, issues, observe)
			}
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

// scopeForConditionFalse keeps branch refinements local to the expression.
// Reuse the existing guard rules rather than treating every falsy value as null.
func scopeForConditionFalse(scope *functionScope, condition ast.Node) *functionScope {
	if unary, ok := condition.(*ast.UnaryExpr); ok && unary.Operator == "!" {
		return scopeForConditionTrue(scope, unary.Operand)
	}
	refined := scope.clone()
	if refined == nil {
		return nil
	}
	applyThisPropertyConditionScope(refined, condition, false)
	for _, name := range variablesNonNullWhenFalse(condition) {
		if typ, ok := nonNullVariableType(refined, name); ok {
			refined.setVariable(name, typ)
		}
	}
	return refined
}

func applyConditionTrueScope(scope *functionScope, condition ast.Node) {
	if scope == nil {
		return
	}
	applyThisPropertyConditionScope(scope, condition, true)
	for variableName, typ := range variablesTypedWhenTrue(condition, scope) {
		if !typ.IsEmpty() {
			scope.setVariable(variableName, typ)
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
		case "!=", "!==":
			if name, ok := nullComparisonVariable(n.Left, n.Right); ok {
				if typ, ok := nonNullVariableType(scope, name); ok {
					return map[string]Type{name: typ}
				}
			}
		}
	case *ast.UnaryExpr:
		if n.Operator == "!" {
			refined := map[string]Type{}
			for _, variableName := range variablesNonNullWhenFalse(n.Operand) {
				if typ, ok := nonNullVariableType(scope, variableName); ok {
					refined[variableName] = typ
				}
			}
			return refined
		}
	case *ast.VariableNode:
		if typ, ok := nonNullVariableType(scope, n.Name); ok {
			return map[string]Type{n.Name: typ}
		}
	case *ast.FunctionCallNode:
		if variableName, typ, ok := builtinTypePredicate(n); ok {
			return map[string]Type{variableName: typ}
		}
	}
	return map[string]Type{}
}

func nonNullVariableType(scope *functionScope, variableName string) (Type, bool) {
	if scope == nil {
		return EmptyType(), false
	}
	current, ok := scope.variable(variableName)
	if !ok {
		return EmptyType(), false
	}
	refined := current.withoutBuiltin("null")
	if refined.IsEmpty() {
		if current.hasBuiltin("null") {
			return MixedType(), true
		}
		return EmptyType(), false
	}
	return refined, true
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

	switch asciiLowerIdent(name) {
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

func finishIfNodeScope(scope *functionScope, node *ast.IfNode, thenScope *functionScope, elseifScopes []*functionScope, elseScope *functionScope, ctx *AnalysisContext) {
	applyTerminatingIfFalseScope(scope, node)
	applyIfElseJoinScope(scope, node, thenScope, elseifScopes, elseScope)
	applyLazyInitPropertyScope(scope, node, ctx)
}

func applyIfElseJoinScope(scope *functionScope, node *ast.IfNode, thenScope *functionScope, elseifScopes []*functionScope, elseScope *functionScope) {
	if scope == nil || node == nil || node.Else == nil {
		return
	}
	var fallthroughs []*functionScope
	if !statementsTerminate(node.Body) {
		fallthroughs = append(fallthroughs, thenScope)
	}
	for i, elseif := range node.ElseIfs {
		if statementsTerminate(elseif.Body) {
			continue
		}
		if i < len(elseifScopes) {
			fallthroughs = append(fallthroughs, elseifScopes[i])
		}
	}
	if !statementsTerminate(node.Else.Body) {
		fallthroughs = append(fallthroughs, elseScope)
	}
	joinFallthroughVariableTypes(scope, fallthroughs)
}

func joinFallthroughVariableTypes(outer *functionScope, fallthroughs []*functionScope) {
	if outer == nil || len(fallthroughs) == 0 {
		return
	}
	names := map[string]struct{}{}
	for _, branch := range fallthroughs {
		collectOwnedVariableNames(branch, names)
	}
	for name := range names {
		var joined Type
		found := false
		all := true
		for _, branch := range fallthroughs {
			if branch == nil {
				all = false
				continue
			}
			typ, ok := branch.variable(name)
			if !ok {
				all = false
				continue
			}
			if !found {
				joined = typ
				found = true
				continue
			}
			joined = unionInferredTypes(joined, typ)
		}
		if !found {
			continue
		}
		if !all {
			if outerType, ok := outer.variable(name); ok {
				joined = unionInferredTypes(joined, outerType)
			}
		}
		if !joined.IsEmpty() {
			outer.setVariable(name, joined)
		}
	}
}

func collectOwnedVariableNames(scope *functionScope, names map[string]struct{}) {
	if scope == nil || names == nil || scope.variables == nil || !scope.variablesOwned {
		return
	}
	layer := scope.variables
	if layer.hasOne && layer.name != "" {
		names[layer.name] = struct{}{}
	}
	for name := range layer.values {
		if name != "" {
			names[name] = struct{}{}
		}
	}
}

func applyTerminatingIfFalseScope(scope *functionScope, node *ast.IfNode) {
	if scope == nil || node == nil || node.Else != nil || len(node.ElseIfs) > 0 {
		return
	}
	if !statementsExitCurrentBlock(node.Body) {
		return
	}
	applyThisPropertyConditionScope(scope, node.Condition, false)
	// Strip null from variables that are non-null when the condition is false
	// (e.g. `if ($x === null) { return; }` → $x is non-null after).
	for _, variableName := range variablesNonNullWhenFalse(node.Condition) {
		current, ok := scope.variable(variableName)
		if !ok {
			continue
		}
		refined := current.withoutBuiltin("null")
		if !refined.IsEmpty() {
			scope.setVariable(variableName, refined)
		}
	}
	// When a negated instanceof guard terminates (return/throw/fail), later
	// statements see the asserted class. This covers `!($x instanceof T)`,
	// PHP's unparenthesized `!$x instanceof T`, and `||` combinations of those
	// checks (false means every operand is false).
	for _, cond := range negatedInstanceofGuardsWhenFalse(node.Condition) {
		if typ := typeFromInstanceofTarget(cond.target, scope); !typ.IsEmpty() {
			scope.setVariable(cond.variable, typ)
		}
	}
	if unary, ok := node.Condition.(*ast.UnaryExpr); ok && unary.Operator == "!" {
		for variableName, typ := range variablesTypedWhenTrue(unary.Operand, scope) {
			if typ.IsEmpty() {
				continue
			}
			scope.setVariable(variableName, typ)
		}
	}
}

func statementsExitCurrentBlock(stmts []ast.Node) bool {
	if statementsTerminate(stmts) {
		return true
	}
	if len(stmts) == 0 {
		return false
	}
	for _, stmt := range stmts {
		if exitsCurrentBlock(stmt) {
			return true
		}
	}
	return false
}

func exitsCurrentBlock(node ast.Node) bool {
	if isTerminatingStatement(node) {
		return true
	}
	switch n := node.(type) {
	case *ast.ExpressionStmt:
		return exitsCurrentBlock(n.Expr)
	case *ast.IdentifierNode:
		keyword := asciiLowerIdent(n.Value)
		return keyword == "break" || keyword == "continue"
	case *ast.IfNode:
		if n.Else == nil {
			return false
		}
		if !statementsExitCurrentBlock(n.Body) {
			return false
		}
		for _, elseif := range n.ElseIfs {
			if !statementsExitCurrentBlock(elseif.Body) {
				return false
			}
		}
		return statementsExitCurrentBlock(n.Else.Body)
	}
	return false
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
	case *ast.FunctionCallNode:
		if variableName, typ, ok := builtinTypePredicate(n); ok && typ.hasBuiltin("null") {
			return []string{variableName}
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

// nullComparisonProperty returns the property name when the expression is
// `null === $this->prop` or `$this->prop === null` (using == or ===).
func nullComparisonProperty(node ast.Node) (string, bool) {
	binary, ok := node.(*ast.BinaryExpr)
	if !ok {
		return "", false
	}
	if binary.Operator != "==" && binary.Operator != "===" {
		return "", false
	}
	check := func(maybeNull, maybeExpr ast.Node) (string, bool) {
		if !isNullLiteral(maybeNull) {
			return "", false
		}
		prop, ok := maybeExpr.(*ast.PropertyFetchNode)
		if !ok {
			return "", false
		}
		obj, ok := prop.Object.(*ast.VariableNode)
		if !ok || obj.Name != "this" {
			return "", false
		}
		return prop.Property, true
	}
	if name, ok := check(binary.Left, binary.Right); ok {
		return name, true
	}
	return check(binary.Right, binary.Left)
}

// bodyAssignsToThisProperty returns true when any top-level statement in body
// is an assignment to $this-><propertyName>.
func bodyAssignsToThisProperty(body []ast.Node, propertyName string) bool {
	for _, stmt := range body {
		var assignment *ast.AssignmentNode
		switch n := stmt.(type) {
		case *ast.AssignmentNode:
			assignment = n
		case *ast.ExpressionStmt:
			a, ok := n.Expr.(*ast.AssignmentNode)
			if !ok {
				continue
			}
			assignment = a
		default:
			continue
		}
		prop, ok := assignment.Left.(*ast.PropertyFetchNode)
		if !ok {
			continue
		}
		obj, ok := prop.Object.(*ast.VariableNode)
		if ok && obj.Name == "this" && strings.EqualFold(prop.Property, propertyName) {
			return true
		}
	}
	return false
}

// applyLazyInitPropertyScope handles the lazy-initialisation pattern:
//
//	if (null === $this->prop) { $this->prop = ...; }
//
// After such a block the property is guaranteed non-null regardless of which
// branch executed, so null is stripped from the property type in the outer scope.
func applyLazyInitPropertyScope(scope *functionScope, node *ast.IfNode, ctx *AnalysisContext) {
	if scope == nil || node == nil || node.Else != nil || len(node.ElseIfs) > 0 {
		return
	}
	if statementsTerminate(node.Body) {
		return
	}
	propertyName, ok := nullComparisonProperty(node.Condition)
	if !ok {
		return
	}
	if !bodyAssignsToThisProperty(node.Body, propertyName) {
		return
	}
	current, hasCurrent := scope.property(propertyName)
	if !hasCurrent {
		if declType, ok := scope.propertyDecls[propertyName]; ok {
			current = declType
		}
	}
	refined := current.withoutBuiltin("null")
	if !refined.IsEmpty() {
		scope.setProperty(propertyName, refined)
	}
}

func walkExprForArgTypesUsing(node ast.Node, scope *functionScope, ctx *AnalysisContext, filename string, issues *[]AnalysisIssue, observe semanticExpressionObserver) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *ast.StringLiteral, *ast.InterpolatedStringLiteral, *ast.StringNode,
		*ast.IntegerLiteral, *ast.IntegerNode,
		*ast.FloatLiteral, *ast.FloatNode,
		*ast.BooleanLiteral, *ast.BooleanNode,
		*ast.NullLiteral, *ast.NullNode,
		*ast.VariableNode:
		observeSemanticExpression(filename, n, scope, ctx, observe)
	case *ast.MethodCallNode:
		if issues != nil {
			checkMethodCallArgTypes(n, scope, ctx, filename, issues)
		}
		if ctx != nil && ctx.argCountSink != nil {
			checkMethodCallArgCount(n, scope, ctx, filename, ctx.argCountSink)
		}
		observeArgumentExpressions(n.Args, scope, ctx, filename, observe)
		walkExprForArgTypesUsing(n.Object, scope, ctx, filename, issues, observe)
		observeSemanticExpression(filename, n.Object, scope, ctx, observe)
		for _, arg := range n.Args {
			walkExprForArgTypesUsing(argumentValue(arg), scope, ctx, filename, issues, observe)
		}
		observeSemanticExpression(filename, n, scope, ctx, observe)
		appendDeprecatedCallFromExpr(filename, n, scope, ctx)
	case *ast.FunctionCallNode:
		if issues != nil {
			checkFunctionCallArgTypes(n, scope, ctx, filename, issues)
		}
		observeArgumentExpressions(n.Args, scope, ctx, filename, observe)
		walkExprForArgTypesUsing(n.Name, scope, ctx, filename, issues, observe)
		for _, arg := range n.Args {
			walkExprForArgTypesUsing(argumentValue(arg), scope, ctx, filename, issues, observe)
		}
		observeSemanticExpression(filename, n, scope, ctx, observe)
		appendDeprecatedCallFromExpr(filename, n, scope, ctx)
	case *ast.AssignmentNode:
		walkExprForArgTypesUsing(n.Right, scope, ctx, filename, issues, observe)
		observeSemanticExpression(filename, n.Right, scope, ctx, observe)
		if observe != nil {
			assignedScope := scope.clone()
			applyAssignmentScope(assignedScope, n, ctx)
			walkExprForArgTypesUsing(n.Left, assignedScope, ctx, filename, issues, observe)
		} else if ctx != nil && ctx.deprecatedCallSink != nil {
			assignedScope := scope.clone()
			applyAssignmentScope(assignedScope, n, ctx)
			walkExprForArgTypesUsing(n.Left, assignedScope, ctx, filename, nil, observe)
		}
		recordAssignmentTypeIssues(n, scope, ctx, filename)
	case *ast.PropertyFetchNode:
		walkExprForArgTypesUsing(n.Object, scope, ctx, filename, issues, observe)
		observeSemanticExpression(filename, n.Object, scope, ctx, observe)
		observeSemanticExpression(filename, n, scope, ctx, observe)
	case *ast.BinaryExpr:
		walkExprForArgTypesUsing(n.Left, scope, ctx, filename, issues, observe)
		switch n.Operator {
		case "&&", "and":
			walkExprForArgTypesUsing(n.Right, scopeForConditionTrue(scope, n.Left), ctx, filename, issues, observe)
		case "||", "or":
			walkExprForArgTypesUsing(n.Right, scopeForConditionFalse(scope, n.Left), ctx, filename, issues, observe)
		default:
			walkExprForArgTypesUsing(n.Right, scope, ctx, filename, issues, observe)
		}
		observeSemanticExpression(filename, n, scope, ctx, observe)
		recordBinaryOpIssue(n, scope, ctx, filename)
	case *ast.ConcatNode:
		for _, part := range n.Parts {
			walkExprForArgTypesUsing(part, scope, ctx, filename, issues, observe)
		}
		observeSemanticExpression(filename, n, scope, ctx, observe)
	case *ast.TernaryExpr:
		walkExprForArgTypesUsing(n.Condition, scope, ctx, filename, issues, observe)
		walkExprForArgTypesUsing(n.IfTrue, scopeForConditionTrue(scope, n.Condition), ctx, filename, issues, observe)
		walkExprForArgTypesUsing(n.IfFalse, scopeForConditionFalse(scope, n.Condition), ctx, filename, issues, observe)
		observeSemanticExpression(filename, n, scope, ctx, observe)
	case *ast.NewNode:
		if issues != nil {
			checkNewArgTypes(n, scope, ctx, filename, issues)
		}
		if ctx != nil && ctx.argCountSink != nil {
			checkNewArgCount(n, scope, ctx, filename, ctx.argCountSink)
		}
		observeArgumentExpressions(n.Args, scope, ctx, filename, observe)
		walkExprForArgTypesUsing(n.ClassExpr, scope, ctx, filename, issues, observe)
		for _, arg := range n.Args {
			walkExprForArgTypesUsing(argumentValue(arg), scope, ctx, filename, issues, observe)
		}
		observeSemanticExpression(filename, n, scope, ctx, observe)
	case *ast.NamedArgumentNode:
		walkExprForArgTypesUsing(n.Value, scope, ctx, filename, issues, observe)
	case *ast.UnpackedArgumentNode:
		walkExprForArgTypesUsing(n.Expr, scope, ctx, filename, issues, observe)
	case *ast.UnaryExpr:
		walkExprForArgTypesUsing(n.Operand, scope, ctx, filename, issues, observe)
		observeSemanticExpression(filename, n, scope, ctx, observe)
	case *ast.ArrayNode:
		for _, child := range n.Elements {
			walkExprForArgTypesUsing(child, scope, ctx, filename, issues, observe)
		}
		observeSemanticExpression(filename, n, scope, ctx, observe)
	case *ast.ArrayItemNode:
		walkExprForArgTypesUsing(n.Key, scope, ctx, filename, issues, observe)
		walkExprForArgTypesUsing(n.Value, scope, ctx, filename, issues, observe)
	case *ast.ArrayAccessNode:
		walkExprForArgTypesUsing(n.Var, scope, ctx, filename, issues, observe)
		walkExprForArgTypesUsing(n.Index, scope, ctx, filename, issues, observe)
		observeSemanticExpression(filename, n, scope, ctx, observe)
	case *ast.MatchNode:
		walkExprForArgTypesUsing(n.Condition, scope, ctx, filename, issues, observe)
		for _, arm := range n.Arms {
			for _, condition := range arm.Conditions {
				walkExprForArgTypesUsing(condition, scope, ctx, filename, issues, observe)
			}
			walkExprForArgTypesUsing(arm.Body, scope, ctx, filename, issues, observe)
		}
		observeSemanticExpression(filename, n, scope, ctx, observe)
	case *ast.ArrowFunctionNode:
		walkExprForArgTypesUsing(n.Expr, scope, ctx, filename, issues, observe)
		observeSemanticExpression(filename, n, scope, ctx, observe)
	}
}

func observeArgumentExpressions(args []ast.Node, scope *functionScope, ctx *AnalysisContext, filename string, observe semanticExpressionObserver) {
	for _, arg := range args {
		observeSemanticExpression(filename, argumentValue(arg), scope, ctx, observe)
	}
}

func observeSemanticExpression(filename string, expr ast.Node, scope *functionScope, ctx *AnalysisContext, observe semanticExpressionObserver) {
	if observe != nil && expr != nil {
		observe(filename, expr, scope, ctx)
	}
}

func checkMethodCallArgTypes(call *ast.MethodCallNode, scope *functionScope, ctx *AnalysisContext, filename string, issues *[]AnalysisIssue) {
	method, ok := resolveMethodForCall(call, scope, ctx, filename)
	if !ok || len(method.Params) == 0 {
		return
	}
	checkResolvedCallArgTypes(fmt.Sprintf("Method %s", method.Name), method, call.Args, scope, ctx, filename, issues, methodCalleeClass(method, call, scope, ctx, filename))
}

func checkFunctionCallArgTypes(call *ast.FunctionCallNode, scope *functionScope, ctx *AnalysisContext, filename string, issues *[]AnalysisIssue) {
	if call == nil || ctx == nil || ctx.Resolver == nil {
		return
	}
	name := functionCallName(call)
	if name == "" {
		return
	}
	var typeCtx FileTypeContext
	if scope != nil {
		typeCtx = scope.typeCtx
	}
	resolvedName := resolveFunctionNameForCall(name, typeCtx, ctx)
	function, ok := resolveFunctionView(ctx.Resolver, resolvedName)
	if !ok || len(function.Params) == 0 {
		return
	}
	checkResolvedCallArgTypes("Function "+function.Name, ResolvedMethod{Name: function.Name, Params: function.Params}, call.Args, scope, ctx, filename, issues, "")
}

func checkNewArgTypes(node *ast.NewNode, scope *functionScope, ctx *AnalysisContext, filename string, issues *[]AnalysisIssue) {
	className, method, ok := resolveConstructorForNew(node, scope, ctx)
	if !ok || len(method.Params) == 0 {
		return
	}
	checkResolvedCallArgTypes(fmt.Sprintf("Class %s constructor", className), method, node.Args, scope, ctx, filename, issues, className)
}

func methodCalleeClass(method ResolvedMethod, call *ast.MethodCallNode, scope *functionScope, ctx *AnalysisContext, filename string) string {
	if call != nil {
		if className, ok := inferTypeWithFacts(filename, call.Object, scope, ctx).SingleClassName(); ok {
			return className
		}
	}
	return method.DeclaringClass
}

func expectedCallParamType(param ResolvedParam, method ResolvedMethod, calleeClass string, ctx *AnalysisContext) Type {
	if name := openTemplateParamName(param.Type, ctx); name != "" {
		return MixedType()
	}
	return bindCalleeSignatureType(expandUnboundClassTemplates(param.Type, method.DeclaringClass, ctx), method.DeclaringClass, calleeClass, ctx)
}

func checkResolvedCallArgTypes(target string, method ResolvedMethod, args []ast.Node, scope *functionScope, ctx *AnalysisContext, filename string, issues *[]AnalysisIssue, calleeClass string) {
	if len(method.Params) == 0 {
		return
	}

	nameToIndex := make(map[string]int, len(method.Params))
	variadicIndex := -1
	for idx, param := range method.Params {
		nameToIndex[asciiLowerIdent(param.Name)] = idx
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
			idx, ok := nameToIndex[asciiLowerIdent(namedArg.Name)]
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
		expected := expectedCallParamType(param, method, calleeClass, ctx)
		if expected.IsEmpty() {
			usedParams[paramIndex] = struct{}{}
			continue
		}

		actual := inferTypeWithFacts(filename, argExpr, scope, ctx)
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

func resolveMethodForCall(call *ast.MethodCallNode, scope *functionScope, ctx *AnalysisContext, filename string) (ResolvedMethod, bool) {
	if call == nil {
		return ResolvedMethod{}, false
	}
	if object, ok := call.Object.(*ast.VariableNode); ok && object.Name == "this" {
		if method, ok := resolveSameClassMethod(scope, call.Method); ok {
			return method, true
		}
	}

	objectType := inferTypeWithFacts(filename, call.Object, scope, ctx)
	className, typeArgs, ok := genericReceiver(objectType)
	if !ok {
		return ResolvedMethod{}, false
	}
	if scope != nil && scope.className != "" && strings.EqualFold(className, scope.className) {
		if method, ok := resolveSameClassMethod(scope, call.Method); ok {
			return method, true
		}
	}

	if scope != nil && scope.genericContext != nil {
		if varNode, ok := call.Object.(*ast.VariableNode); ok {
			if genInst, hasGeneric := scope.genericContext[varNode.Name]; hasGeneric {
				if ctx != nil && ctx.Resolver != nil {
					if method, ok := resolveMethodWithGenerics(ctx.Resolver, genInst.ClassName, call.Method, genInst.TypeArguments); ok {
						return method, true
					}
				}
			}
		}
	}

	if ctx != nil && ctx.Resolver != nil {
		if method, ok := resolveMethodWithGenerics(ctx.Resolver, className, call.Method, typeArgs); ok {
			return method, true
		}
	}
	if scope != nil {
		if classData, ok := analysisClassScopeDataByName(ctx, className, scope.typeCtx); ok {
			if method, ok := classData.methods[asciiLowerIdent(call.Method)]; ok {
				return method, true
			}
		}
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
	RegisterAnalysisRuleWithLevel("A.ARG.TYPE", 5, "level5", func(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
		rule := &ArgumentTypeRule{}
		return rule.CheckIssues(nodes, filename, ctx)
	})
}
