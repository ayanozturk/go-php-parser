package analyse

import (
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

func ensureArgCallDiagnosticsFromCST(filename string, nodes []ast.Node, ctx *AnalysisContext) *AnalysisContext {
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
		seen := make(map[*ast.MethodCallNode]struct{})
		observe = func(filename string, expr ast.Node, scope *functionScope, ctx *AnalysisContext) {
			call, ok := expr.(*ast.MethodCallNode)
			if !ok {
				return
			}
			if _, exists := seen[call]; exists {
				return
			}
			seen[call] = struct{}{}
			appendMethodReceiverIssuesForCall(filename, call, scope, ctx, &ctx.methodReceiverIssues)
		}
	}
	fileCtx := analysisFileTypeContext(ctx, nodes)
	res := sharedParseResult(ctx, ctx.Content)
	if res != nil && res.File != nil && res.File.Root != nil {
		walker := argCallCSTWalker{filename: filename, ctx: ctx, fileCtx: fileCtx, typeIssues: typeIssueSink, observe: observe, file: res.File}
		walker.walkTopLevel(res.File.Root)
	}
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

type argCallCSTWalker struct {
	filename   string
	ctx        *AnalysisContext
	fileCtx    FileTypeContext
	typeIssues *[]AnalysisIssue
	observe    semanticExpressionObserver
	file       *syntax.File
}

func (w *argCallCSTWalker) walkTopLevel(root *syntax.RedNode) {
	siblings := syntax.TopLevelStatements(root)
	fileScope := newFunctionScopeWithContext(w.ctx, nil, &ast.FunctionNode{}, w.fileCtx)
	for i := 0; i < len(siblings); i++ {
		n := siblings[i]
		if n.Kind() == syntax.KindNamespaceDecl {
			body, consumed := syntax.NamespaceBody(n, siblings, i)
			namespaceScope := newFunctionScopeWithContext(w.ctx, nil, &ast.FunctionNode{}, w.fileCtx)
			w.walkDeclarations(body, nil, namespaceScope)
			i += consumed
			continue
		}
		w.walkDeclaration(n, nil, fileScope)
	}
}

func (w *argCallCSTWalker) walkDeclarations(nodes []*syntax.RedNode, class *ast.ClassNode, fileScope *functionScope) {
	for _, n := range nodes {
		w.walkDeclaration(n, class, fileScope)
	}
}

func (w *argCallCSTWalker) walkDeclaration(n *syntax.RedNode, class *ast.ClassNode, fileScope *functionScope) {
	if n == nil {
		return
	}
	switch n.Kind() {
	case syntax.KindNamespaceDecl:
		body, _ := syntax.NamespaceBody(n, nil, 0)
		namespaceScope := newFunctionScopeWithContext(w.ctx, class, &ast.FunctionNode{}, w.fileCtx)
		w.walkDeclarations(body, class, namespaceScope)
	case syntax.KindClassDecl, syntax.KindTraitDecl, syntax.KindEnumDecl:
		classCtx := syntax.LowerClassLikeHeaderNode(n, w.file)
		if classCtx == nil {
			return
		}
		for _, member := range n.Children() {
			if member.Kind() == syntax.KindMemberList {
				for _, child := range member.Children() {
					switch child.Kind() {
					case syntax.KindPropertyDecl:
						for _, prop := range syntax.LowerPropertyDeclNode(child, w.file) {
							walkArgCallProperty(prop, classCtx, w.ctx, w.filename, w.fileCtx, w.typeIssues, w.observe)
						}
					case syntax.KindFunctionDecl, syntax.KindMethodDecl:
						fn := syntax.LowerFunctionHeaderNode(child, w.file)
						if fn == nil {
							continue
						}
						scope := analysisFunctionScope(w.ctx, classCtx, fn, w.fileCtx)
						if body := syntax.FunctionBody(child); body != nil {
							w.walkStatementList(body, scope, nil)
						}
					}
				}
			}
		}
	case syntax.KindFunctionDecl:
		fn := syntax.LowerFunctionHeaderNode(n, w.file)
		if fn == nil {
			return
		}
		scope := analysisFunctionScope(w.ctx, class, fn, w.fileCtx)
		if body := syntax.FunctionBody(n); body != nil {
			w.walkStatementList(body, scope, nil)
		}
	default:
		if syntax.IsStatementKind(n.Kind()) {
			w.walkStatement(n, fileScope, nil)
		}
	}
}

func (w *argCallCSTWalker) walkStatementList(list *syntax.RedNode, scope *functionScope, prefix []*syntax.RedNode) {
	if list == nil || scope == nil {
		return
	}
	siblings := syntax.StatementSiblings(list)
	for i, stmt := range siblings {
		prior := append(append([]*syntax.RedNode(nil), prefix...), siblings[:i]...)
		w.walkStatement(stmt, scope, prior)
	}
}

func (w *argCallCSTWalker) walkBody(body *syntax.RedNode, scope *functionScope) {
	if body != nil && body.Kind() == syntax.KindStatementList {
		w.walkStatementList(body, scope, nil)
		return
	}
	for _, stmt := range syntax.StatementBodyList(body) {
		w.walkStatement(stmt, scope, nil)
	}
}

func (w *argCallCSTWalker) lowerExpr(n *syntax.RedNode) ast.Node {
	if n == nil {
		return nil
	}
	return syntax.LowerExprNode(n, w.file)
}

func (w *argCallCSTWalker) walkStatement(n *syntax.RedNode, scope *functionScope, prefix []*syntax.RedNode) {
	if n == nil || scope == nil {
		return
	}
	// Simple statements are lowered individually, never as a function/file
	// body. The existing expression visitors remain the leaf semantic logic.
	switch n.Kind() {
	case syntax.KindIfStmt:
		w.walkIf(n, scope)
	case syntax.KindWhileStmt:
		cond := w.lowerExpr(syntax.WhileCondition(n))
		walkExprForArgTypesUsing(cond, scope, w.ctx, w.filename, w.typeIssues, w.observe)
		w.walkBody(syntax.WhileBody(n), scopeForConditionTrue(scope, cond, w.ctx))
		w.applyGuaranteedWhileExit(prefix, n, scope, cond)
	case syntax.KindDoWhileStmt:
		loop := scope.clone()
		w.walkBody(syntax.DoWhileBody(n), loop)
		cond := w.lowerExpr(syntax.DoWhileCondition(n))
		walkExprForArgTypesUsing(cond, loop, w.ctx, w.filename, w.typeIssues, w.observe)
	case syntax.KindForStmt:
		w.walkFor(n, scope)
	case syntax.KindForeachStmt:
		w.walkForeach(n, scope)
	case syntax.KindTryStmt:
		w.walkTry(n, scope)
	case syntax.KindSwitchStmt:
		w.walkSwitch(n, scope)
	case syntax.KindExpressionStmt, syntax.KindEchoStmt, syntax.KindUnsetStmt,
		syntax.KindReturnStmt, syntax.KindThrowStmt,
		syntax.KindBreakStmt, syntax.KindContinueStmt:
		stmt := syntax.LowerStmtNode(n, w.file)
		if stmt != nil {
			walkStatementsForArgTypesUsing([]ast.Node{stmt}, scope, w.ctx, w.filename, w.typeIssues, w.observe)
		}
	}
}

func (w *argCallCSTWalker) walkIf(n *syntax.RedNode, scope *functionScope) {
	cond := w.lowerExpr(syntax.IfCondition(n))
	walkExprForArgTypesUsing(cond, scope, w.ctx, w.filename, w.typeIssues, w.observe)
	thenScope := scopeForConditionTrue(scope, cond, w.ctx)
	w.walkBody(syntax.IfBody(n), thenScope)
	elseifNodes := syntax.IfElseIfs(n)
	elseifScopes := make([]*functionScope, len(elseifNodes))
	elseifConds := make([]ast.Node, len(elseifNodes))
	for i, elseif := range elseifNodes {
		elseifConds[i] = w.lowerExpr(syntax.ElseIfCondition(elseif))
		walkExprForArgTypesUsing(elseifConds[i], scope, w.ctx, w.filename, w.typeIssues, w.observe)
		elseifScopes[i] = scopeForConditionTrue(scope, elseifConds[i], w.ctx)
		w.walkBody(syntax.ElseIfBody(elseif), elseifScopes[i])
	}
	elseNode := syntax.IfElse(n)
	var elseScope *functionScope
	if elseNode != nil {
		elseScope = scopeForConditionFalse(scope, cond, w.ctx)
		w.walkBody(syntax.ElseBody(elseNode), elseScope)
	}
	w.finishIfScope(scope, cond, thenScope, elseifConds, elseifScopes, elseNode != nil, elseScope, syntax.IfBody(n), elseifNodes, elseNode)
}

func (w *argCallCSTWalker) finishIfScope(scope *functionScope, cond ast.Node, thenScope *functionScope, elseifConds []ast.Node, elseifScopes []*functionScope, hasElse bool, elseScope *functionScope, thenBody *syntax.RedNode, elseifNodes []*syntax.RedNode, elseNode *syntax.RedNode) {
	if scope == nil {
		return
	}
	if !hasElse && len(elseifNodes) == 0 && w.cstBodyExits(thenBody) {
		applyThisPropertyConditionScope(scope, cond, false, w.ctx)
		for _, name := range variablesNonNullWhenFalse(cond) {
			if current, ok := scope.variable(name); ok {
				if refined := current.withoutBuiltin("null"); !refined.IsEmpty() {
					scope.setVariable(name, refined)
				}
			}
		}
		for _, guard := range negatedInstanceofGuardsWhenFalse(cond) {
			if typ := typeFromInstanceofTarget(guard.target, scope); !typ.IsEmpty() {
				current := EmptyType()
				if existing, ok := scope.variable(guard.variable); ok {
					current = existing
				}
				scope.setVariable(guard.variable, refineTypeByInstanceof(current, typ))
			}
		}
		for name, typ := range variablesTypedWhenFalse(cond, scope) {
			if !typ.IsEmpty() {
				scope.setVariable(name, typ)
			}
		}
	}
	if hasElse {
		var fallthroughs []*functionScope
		if !w.cstBodyTerminates(thenBody) {
			fallthroughs = append(fallthroughs, thenScope)
		}
		for i, body := range elseifNodes {
			if !w.cstBodyTerminates(syntax.ElseIfBody(body)) && i < len(elseifScopes) {
				fallthroughs = append(fallthroughs, elseifScopes[i])
			}
		}
		if !w.cstBodyTerminates(syntax.ElseBody(elseNode)) {
			fallthroughs = append(fallthroughs, elseScope)
		}
		joinFallthroughVariableTypes(scope, fallthroughs)
	}
	if !hasElse && len(elseifNodes) == 0 && !w.cstBodyTerminates(thenBody) {
		w.applyIfSpecialScope(scope, cond, thenScope, thenBody)
	}
}

func (w *argCallCSTWalker) applyIfSpecialScope(scope *functionScope, cond ast.Node, thenScope *functionScope, body *syntax.RedNode) {
	name, ok := falsyInitGuardVariable(cond)
	if ok && w.cstBodyAssignsVariable(body, name) {
		thenType, thenOK := thenScope.variable(name)
		elseType, elseOK := variablesTypedWhenFalse(cond, scope)[name]
		if !elseOK {
			elseType, elseOK = nonNullVariableType(scope, name)
		}
		joined := thenType
		if thenOK && elseOK {
			joined = unionInferredTypes(thenType, elseType)
		} else if elseOK {
			joined = elseType
		}
		if !joined.IsEmpty() {
			scope.setVariable(name, joined)
		}
	}
	if propertyName, ok := nullComparisonProperty(cond); ok && w.cstBodyAssignsThisProperty(body, propertyName) {
		current, hasCurrent := scope.property(propertyName)
		if !hasCurrent {
			current = scope.propertyDecls[propertyName]
		}
		if refined := current.withoutBuiltin("null"); !refined.IsEmpty() {
			scope.setProperty(propertyName, refined)
		}
	}
}

func (w *argCallCSTWalker) cstBodyTerminates(body *syntax.RedNode) bool {
	for _, stmt := range syntax.StatementBodyList(body) {
		if w.cstStatementTerminates(stmt) {
			return true
		}
	}
	return false
}

func (w *argCallCSTWalker) cstStatementTerminates(stmt *syntax.RedNode) bool {
	if stmt == nil {
		return false
	}
	switch stmt.Kind() {
	case syntax.KindReturnStmt, syntax.KindThrowStmt:
		return true
	case syntax.KindExpressionStmt:
		expr := syntax.ExpressionStmtExpr(stmt)
		if lowered := w.lowerExpr(expr); lowered != nil {
			return statementsTerminate([]ast.Node{&ast.ExpressionStmt{Expr: lowered}})
		}
	case syntax.KindIfStmt:
		elseNode := syntax.IfElse(stmt)
		if elseNode == nil || !w.cstBodyTerminates(syntax.IfBody(stmt)) {
			return false
		}
		for _, elseif := range syntax.IfElseIfs(stmt) {
			if !w.cstBodyTerminates(syntax.ElseIfBody(elseif)) {
				return false
			}
		}
		return w.cstBodyTerminates(syntax.ElseBody(elseNode))
	}
	return false
}

func (w *argCallCSTWalker) cstBodyExits(body *syntax.RedNode) bool {
	stmts := syntax.StatementBodyList(body)
	if w.cstBodyTerminates(body) {
		return true
	}
	for _, stmt := range stmts {
		if stmt.Kind() == syntax.KindBreakStmt || stmt.Kind() == syntax.KindContinueStmt {
			return true
		}
		if stmt.Kind() == syntax.KindIfStmt {
			if syntax.IfElse(stmt) == nil || !w.cstBodyExits(syntax.IfBody(stmt)) {
				continue
			}
			allExit := true
			for _, elseif := range syntax.IfElseIfs(stmt) {
				if !w.cstBodyExits(syntax.ElseIfBody(elseif)) {
					allExit = false
					break
				}
			}
			if allExit && w.cstBodyExits(syntax.ElseBody(syntax.IfElse(stmt))) {
				return true
			}
		}
	}
	return false
}

func (w *argCallCSTWalker) cstBodyAssignsVariable(body *syntax.RedNode, name string) bool {
	for _, stmt := range syntax.StatementBodyList(body) {
		if stmt.Kind() != syntax.KindExpressionStmt {
			continue
		}
		expr := syntax.ExpressionStmtExpr(stmt)
		if expr == nil || expr.Kind() != syntax.KindAssignExpr {
			continue
		}
		left, _, _ := syntax.AssignmentParts(expr)
		if left != nil && left.Kind() == syntax.KindVariableExpr && strings.TrimPrefix(strings.TrimSpace(left.Text()), "$") == name {
			return true
		}
	}
	return false
}

func (w *argCallCSTWalker) cstBodyAssignsThisProperty(body *syntax.RedNode, name string) bool {
	for _, stmt := range syntax.StatementBodyList(body) {
		if stmt.Kind() != syntax.KindExpressionStmt {
			continue
		}
		expr := syntax.ExpressionStmtExpr(stmt)
		if expr == nil || expr.Kind() != syntax.KindAssignExpr {
			continue
		}
		left, _, _ := syntax.AssignmentParts(expr)
		if left == nil || (left.Kind() != syntax.KindMemberAccessExpr && left.Kind() != syntax.KindNullsafeMemberAccessExpr) {
			continue
		}
		property := syntax.MemberAccessName(left)
		object := syntax.MemberAccessObject(left)
		if strings.EqualFold(property, name) && object != nil && strings.TrimSpace(object.Text()) == "$this" {
			return true
		}
	}
	return false
}

func (w *argCallCSTWalker) walkFor(n *syntax.RedNode, scope *functionScope) {
	for _, node := range syntax.ForInitExpressions(n) {
		expr := w.lowerExpr(node)
		walkExprForArgTypesUsing(expr, scope, w.ctx, w.filename, w.typeIssues, w.observe)
		applyExpressionScope(scope, expr, w.ctx)
	}
	loopScope := scope.clone()
	for _, cond := range syntax.ForConditions(n) {
		expr := w.lowerExpr(cond)
		walkExprForArgTypesUsing(expr, loopScope, w.ctx, w.filename, w.typeIssues, w.observe)
	}
	w.walkBody(syntax.ForBody(n), loopScope)
	for _, node := range syntax.ForUpdateExpressions(n) {
		expr := w.lowerExpr(node)
		walkExprForArgTypesUsing(expr, loopScope, w.ctx, w.filename, w.typeIssues, w.observe)
	}
}

func (w *argCallCSTWalker) walkForeach(n *syntax.RedNode, scope *functionScope) {
	iterable, key, value := syntax.ForeachParts(n)
	expr := w.lowerExpr(iterable)
	walkExprForArgTypesUsing(expr, scope, w.ctx, w.filename, w.typeIssues, w.observe)
	loopScope := scope.clone()
	fe := &ast.ForeachNode{Expr: expr}
	fe.KeyVar, fe.ValueVar = w.lowerExpr(key), w.lowerExpr(value)
	applyForeachIterationTypes(loopScope, fe, w.ctx, w.filename)
	w.walkBody(syntax.ForeachBody(n), loopScope)
	joinLoopAssignedVariables(scope, loopScope)
}

func (w *argCallCSTWalker) walkTry(n *syntax.RedNode, scope *functionScope) {
	w.walkBody(syntax.TryBody(n), scope.clone())
	for _, catch := range syntax.TryCatches(n) {
		w.walkBody(syntax.CatchBody(catch), scope.clone())
	}
	w.walkBody(syntax.TryFinallyBody(n), scope.clone())
}

func (w *argCallCSTWalker) walkSwitch(n *syntax.RedNode, scope *functionScope) {
	expr := w.lowerExpr(syntax.SwitchExpr(n))
	walkExprForArgTypesUsing(expr, scope, w.ctx, w.filename, w.typeIssues, w.observe)
	for _, clause := range syntax.SwitchCases(n) {
		caseExpr := w.lowerExpr(syntax.SwitchCaseExpr(clause))
		walkExprForArgTypesUsing(caseExpr, scope, w.ctx, w.filename, w.typeIssues, w.observe)
		w.walkBody(syntax.SwitchCaseBody(clause), scope.clone())
	}
}

func (w *argCallCSTWalker) applyGuaranteedWhileExit(prefix []*syntax.RedNode, loop *syntax.RedNode, scope *functionScope, condition ast.Node) {
	if loop == nil || scope == nil {
		return
	}
	var prefixAST []ast.Node
	for _, stmt := range prefix {
		if stmt == nil || !syntax.IsStatementKind(stmt.Kind()) {
			continue
		}
		if lowered := syntax.LowerStmtNode(stmt, w.file); lowered != nil {
			prefixAST = append(prefixAST, lowered)
		}
	}
	if !whileConditionTrueAtEntry(prefixAST, condition, scope) {
		return
	}
	entry := loopFlowPath{scope: scopeForConditionTrue(scope, condition, w.ctx), assigned: map[string]struct{}{}}
	result := w.simulateCSTLoopStatements(syntax.StatementBodyList(syntax.WhileBody(loop)), []loopFlowPath{entry})
	exits := append(append([]loopFlowPath(nil), result.normal...), result.continues...)
	exits = append(exits, result.breaks...)
	if len(exits) == 0 {
		return
	}
	for name := range exits[0].assigned {
		var joined Type
		for i, exit := range exits {
			if _, ok := exit.assigned[name]; !ok || exit.scope == nil {
				joined = EmptyType()
				break
			}
			typ, ok := exit.scope.variable(name)
			if !ok || typ.IsEmpty() {
				joined = EmptyType()
				break
			}
			if i == 0 {
				joined = typ
			} else {
				joined = unionInferredTypes(joined, typ)
			}
		}
		if !joined.IsEmpty() {
			scope.setVariable(name, joined)
		}
	}
}

func (w *argCallCSTWalker) simulateCSTLoopStatements(stmts []*syntax.RedNode, inputs []loopFlowPath) loopFlowResult {
	result := loopFlowResult{normal: inputs}
	for _, stmt := range stmts {
		if len(result.normal) == 0 {
			break
		}
		next := loopFlowResult{}
		for _, input := range result.normal {
			step := w.simulateCSTLoopStatement(stmt, input)
			next.normal = append(next.normal, step.normal...)
			next.breaks = append(next.breaks, step.breaks...)
			next.continues = append(next.continues, step.continues...)
		}
		result.normal = next.normal
		result.breaks = append(result.breaks, next.breaks...)
		result.continues = append(result.continues, next.continues...)
	}
	return result
}

func (w *argCallCSTWalker) simulateCSTLoopStatement(stmt *syntax.RedNode, input loopFlowPath) loopFlowResult {
	if stmt == nil {
		return loopFlowResult{normal: []loopFlowPath{input}}
	}
	switch stmt.Kind() {
	case syntax.KindReturnStmt, syntax.KindThrowStmt:
		return loopFlowResult{}
	case syntax.KindBreakStmt:
		return loopFlowResult{breaks: []loopFlowPath{input}}
	case syntax.KindContinueStmt:
		return loopFlowResult{continues: []loopFlowPath{input}}
	case syntax.KindIfStmt:
		condition := w.lowerExpr(syntax.IfCondition(stmt))
		branches := []loopFlowResult{w.simulateCSTLoopStatements(syntax.StatementBodyList(syntax.IfBody(stmt)), []loopFlowPath{cloneLoopFlowPath(input, scopeForConditionTrue(input.scope, condition, w.ctx))})}
		for _, clause := range syntax.IfElseIfs(stmt) {
			clauseCond := w.lowerExpr(syntax.ElseIfCondition(clause))
			branches = append(branches, w.simulateCSTLoopStatements(syntax.StatementBodyList(syntax.ElseIfBody(clause)), []loopFlowPath{cloneLoopFlowPath(input, scopeForConditionTrue(input.scope, clauseCond, w.ctx))}))
		}
		if els := syntax.IfElse(stmt); els != nil {
			branches = append(branches, w.simulateCSTLoopStatements(syntax.StatementBodyList(syntax.ElseBody(els)), []loopFlowPath{cloneLoopFlowPath(input, scopeForConditionFalse(input.scope, condition, w.ctx))}))
		} else {
			branches = append(branches, loopFlowResult{normal: []loopFlowPath{cloneLoopFlowPath(input, input.scope.clone())}})
		}
		return mergeLoopFlowResults(branches...)
	case syntax.KindTryStmt:
		if syntax.TryFinallyBody(stmt) != nil {
			return loopFlowResult{normal: []loopFlowPath{input}}
		}
		branches := []loopFlowResult{w.simulateCSTLoopStatements(syntax.StatementSiblings(syntax.TryBody(stmt)), []loopFlowPath{cloneLoopFlowPath(input, input.scope.clone())})}
		for _, catch := range syntax.TryCatches(stmt) {
			catchScope := input.scope.clone()
			if catchScope != nil && syntax.CatchVariable(catch) != "" {
				var types []string
				for _, raw := range syntax.CatchTypes(catch) {
					types = append(types, catchScope.typeCtx.resolveClassLike(raw))
				}
				if typ := ParseType(strings.Join(types, "|")); !typ.IsEmpty() {
					catchScope.setVariable(syntax.CatchVariable(catch), typ)
				}
			}
			branches = append(branches, w.simulateCSTLoopStatements(syntax.StatementSiblings(syntax.CatchBody(catch)), []loopFlowPath{cloneLoopFlowPath(input, catchScope)}))
		}
		return mergeLoopFlowResults(branches...)
	case syntax.KindExpressionStmt:
		lowered, _ := syntax.LowerStmtNode(stmt, w.file).(*ast.ExpressionStmt)
		if lowered != nil {
			applyExpressionStmtScope(input.scope, lowered, w.ctx)
			markLoopFlowAssignment(&input, assignmentExpr(lowered.Expr))
		}
		return loopFlowResult{normal: []loopFlowPath{input}}
	case syntax.KindStatementList:
		return w.simulateCSTLoopStatements(syntax.StatementSiblings(stmt), []loopFlowPath{input})
	default:
		return loopFlowResult{normal: []loopFlowPath{input}}
	}
}
