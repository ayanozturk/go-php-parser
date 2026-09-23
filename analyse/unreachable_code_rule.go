package analyse

import (
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

// UnreachableCodeRule reports statements that can never execute because a
// previous statement in the same block always terminates control flow.
//
// Inspired by modern static analyzers (like mago), this keeps the check cheap by
// performing a single pass per block and short-circuiting once a terminator is found.
type UnreachableCodeRule struct{}

func (r *UnreachableCodeRule) CheckIssues(nodes []ast.Node, filename string) []AnalysisIssue {
	return r.CheckIssuesWithContext(nodes, filename, nil)
}

func (r *UnreachableCodeRule) CheckIssuesWithContext(nodes []ast.Node, filename string, ctx *AnalysisContext) []AnalysisIssue {
	if ctx != nil && len(ctx.Content) > 0 {
		return r.checkIssuesFromParsedCST(filename, sharedParseResult(ctx, ctx.Content), ctx.Flow)
	}
	issues := make([]AnalysisIssue, 0, 4)
	r.walkStatements(nodes, filename, ctx, &issues)
	return issues
}

// checkIssuesFromParsedCST is the production path. Statement offsets match
// FlowStatementKey, so richer shared flow facts remain available without
// lowering the file to AST nodes.
func (r *UnreachableCodeRule) checkIssuesFromParsedCST(filename string, res *syntax.ParseResult, flow FlowGraphReader) []AnalysisIssue {
	if res == nil || res.File == nil || res.File.Root == nil {
		return nil
	}
	var issues []AnalysisIssue
	var walk func([]*syntax.RedNode)
	var walkStmt func(*syntax.RedNode)
	walk = func(stmts []*syntax.RedNode) {
		terminated := false
		for _, stmt := range stmts {
			if stmt == nil || stmt.Kind() == syntax.KindToken {
				continue
			}
			reachable := !terminated
			if flow != nil {
				p, e := stmt.Pos(), stmt.EndPos()
				key := FlowStatementKey{File: filename, StartOffset: p.Offset, EndOffset: e.Offset}
				fromGraph, ok := flow.StatementReachable(key)
				// Lowered ExpressionStmt spans exclude the trailing semicolon,
				// while the lossless CST span includes it. Preserve the shared
				// flow lookup by trying that AST-compatible end offset as well.
				if !ok && stmt.Kind() == syntax.KindExpressionStmt && e.Offset > p.Offset {
					key.EndOffset--
					fromGraph, ok = flow.StatementReachable(key)
				}
				if ok {
					reachable = fromGraph
				}
			}
			if !reachable {
				p := stmt.Pos()
				issues = append(issues, AnalysisIssue{Filename: filename, Line: p.Line, Column: p.Column, Code: "Generic.CodeAnalysis.UnreachableCode", Message: "Unreachable statement after terminating statement"})
				continue
			}
			walkStmt(stmt)
			if cstTerminatingStatement(stmt) {
				terminated = true
			}
		}
	}
	walkBody := func(body *syntax.RedNode) { walk(syntax.StatementBodyList(body)) }
	walkStmt = func(n *syntax.RedNode) {
		if n == nil {
			return
		}
		switch n.Kind() {
		case syntax.KindStatementList:
			walkBody(n)
		case syntax.KindFunctionDecl, syntax.KindMethodDecl:
			walkBody(syntax.FunctionBody(n))
		case syntax.KindClassDecl:
			for _, method := range syntax.ClassMethods(n) {
				walkStmt(method)
			}
		case syntax.KindNamespaceDecl:
			var children []*syntax.RedNode
			n.ForEachChild(func(ch *syntax.RedNode) bool {
				if ch.Kind() == syntax.KindStatementList {
					children = syntax.StatementBodyList(ch)
				}
				return true
			})
			walk(children)
		case syntax.KindIfStmt:
			walkBody(syntax.IfBody(n))
			for _, ei := range syntax.IfElseIfs(n) {
				walkBody(syntax.ElseIfBody(ei))
			}
			if els := syntax.IfElse(n); els != nil {
				walkBody(syntax.ElseBody(els))
			}
		case syntax.KindWhileStmt:
			walkBody(syntax.WhileBody(n))
		case syntax.KindDoWhileStmt:
			walkBody(syntax.DoWhileBody(n))
		case syntax.KindForStmt:
			walkBody(syntax.ForBody(n))
		case syntax.KindForeachStmt:
			n.ForEachChild(func(ch *syntax.RedNode) bool {
				if isCSTStatement(ch.Kind()) {
					walkStmt(ch)
				}
				return true
			})
		}
	}
	tops := res.File.Root.Children()
	for i := 0; i < len(tops); i++ {
		if tops[i].Kind() == syntax.KindToken {
			continue
		}
		if tops[i].Kind() == syntax.KindNamespaceDecl {
			body, consumed := syntax.NamespaceBody(tops[i], tops, i)
			walk(body)
			i += consumed
			continue
		}
		walk([]*syntax.RedNode{tops[i]})
	}
	return issues
}

func isCSTStatement(k syntax.Kind) bool {
	return k == syntax.KindStatementList || syntax.IsStatementKind(k)
}

func cstTerminatingStatement(n *syntax.RedNode) bool {
	if n == nil {
		return false
	}
	switch n.Kind() {
	case syntax.KindReturnStmt, syntax.KindThrowStmt, syntax.KindBreakStmt, syntax.KindContinueStmt:
		return true
	case syntax.KindExpressionStmt:
		return cstTerminatingExpr(syntax.ExpressionStmtExpr(n))
	case syntax.KindIfStmt:
		if syntax.IfElse(n) == nil || !cstStatementsTerminate(syntax.StatementBodyList(syntax.IfBody(n))) {
			return false
		}
		for _, ei := range syntax.IfElseIfs(n) {
			if !cstStatementsTerminate(syntax.StatementBodyList(syntax.ElseIfBody(ei))) {
				return false
			}
		}
		return cstStatementsTerminate(syntax.StatementBodyList(syntax.ElseBody(syntax.IfElse(n))))
	}
	return false
}

func cstStatementsTerminate(stmts []*syntax.RedNode) bool {
	for _, s := range stmts {
		if cstTerminatingStatement(s) {
			return true
		}
	}
	return false
}

func cstTerminatingExpr(n *syntax.RedNode) bool {
	if n == nil || n.Kind() != syntax.KindCallExpr {
		return false
	}
	callee := syntax.CallCallee(n)
	if callee == nil {
		return false
	}
	if !syntax.CallIsMethodLike(n) {
		name := strings.TrimLeft(asciiLowerIdent(syntax.NameText(callee)), `\\`)
		if name == "exit" || name == "die" {
			return true
		}
		return isPHPUnitNeverMethod(staticCallMethodName(name)) && strings.Contains(name, "::")
	}
	access := callee
	if callee.Kind() != syntax.KindMemberAccessExpr && callee.Kind() != syntax.KindNullsafeMemberAccessExpr && callee.Kind() != syntax.KindStaticMemberAccessExpr {
		return false
	}
	obj := syntax.MemberAccessObject(access)
	return obj != nil && obj.Kind() == syntax.KindVariableExpr && syntax.VariableExprName(obj) == "this" && isPHPUnitNeverMethod(syntax.MemberAccessName(access))
}

func (r *UnreachableCodeRule) walkStatements(stmts []ast.Node, filename string, ctx *AnalysisContext, issues *[]AnalysisIssue) {
	terminated := false
	for _, stmt := range stmts {
		if isNonExecutableStatement(stmt) {
			continue
		}
		reachable := !terminated
		if ctx != nil && ctx.Flow != nil {
			if fromGraph, ok := ctx.Flow.StatementReachable(flowStatementKey(filename, stmt)); ok {
				reachable = fromGraph
			}
		}
		if !reachable {
			pos := stmt.GetPos()
			*issues = append(*issues, AnalysisIssue{
				Filename: filename,
				Line:     pos.Line,
				Column:   pos.Column,
				Code:     "Generic.CodeAnalysis.UnreachableCode",
				Message:  "Unreachable statement after terminating statement",
			})
			continue
		}

		r.walkChildren(stmt, filename, ctx, issues)

		// Only mark as terminated if this statement always terminates control flow in all code paths
		if isTerminatingStatement(stmt) {
			terminated = true
		}
	}
}

func (r *UnreachableCodeRule) walkChildren(node ast.Node, filename string, ctx *AnalysisContext, issues *[]AnalysisIssue) {
	switch n := node.(type) {
	case *ast.FunctionNode:
		r.walkStatements(n.Body, filename, ctx, issues)
	case *ast.ClassNode:
		for _, m := range n.Methods {
			r.walkChildren(m, filename, ctx, issues)
		}
	case *ast.BlockNode:
		r.walkStatements(n.Statements, filename, ctx, issues)
	case *ast.IfNode:
		r.walkStatements(n.Body, filename, ctx, issues)
		for _, elseif := range n.ElseIfs {
			r.walkStatements(elseif.Body, filename, ctx, issues)
		}
		if n.Else != nil {
			r.walkStatements(n.Else.Body, filename, ctx, issues)
		}
	case *ast.WhileNode:
		r.walkStatements(n.Body, filename, ctx, issues)
	case *ast.DoWhileNode:
		r.walkStatements(n.Body, filename, ctx, issues)
	case *ast.ForNode:
		r.walkStatements(n.Body, filename, ctx, issues)
	case *ast.ForeachNode:
		r.walkStatements(n.Body, filename, ctx, issues)
	case *ast.NamespaceNode:
		r.walkStatements(n.Body, filename, ctx, issues)
	}
}

func isNonExecutableStatement(node ast.Node) bool {
	switch node.(type) {
	case *ast.CommentNode, *ast.PHPDocNode, *ast.AttributeNode:
		return true
	default:
		return false
	}
}

func isTerminatingStatement(node ast.Node) bool {
	switch n := node.(type) {
	case *ast.ReturnNode, *ast.ThrowNode, *ast.BreakNode, *ast.ContinueNode:
		return true
	case *ast.ExpressionStmt:
		return isTerminatingStatement(n.Expr)
	case *ast.FunctionCallNode:
		return isNeverReturningCall(n)
	case *ast.MethodCallNode:
		return isNeverReturningCall(n)
	case *ast.IfNode:
		if n.Else == nil {
			return false
		}
		if !statementsTerminate(n.Body) {
			return false
		}
		for _, elseif := range n.ElseIfs {
			if !statementsTerminate(elseif.Body) {
				return false
			}
		}
		return statementsTerminate(n.Else.Body)
	}
	return false
}

func statementsTerminate(stmts []ast.Node) bool {
	if len(stmts) == 0 {
		return false
	}
	for _, stmt := range stmts {
		if isTerminatingStatement(stmt) {
			return true
		}
	}
	return false
}

func isNeverReturningCall(node ast.Node) bool {
	switch n := node.(type) {
	case *ast.FunctionCallNode:
		if isBuiltinTerminatorCall(n) {
			return true
		}
		identifier, ok := n.Name.(*ast.IdentifierNode)
		if !ok {
			return false
		}
		return isPHPUnitNeverMethod(staticCallMethodName(identifier.Value))
	case *ast.MethodCallNode:
		receiver, ok := n.Object.(*ast.VariableNode)
		return ok && receiver.Name == "this" && isPHPUnitNeverMethod(n.Method)
	default:
		return false
	}
}

func isBuiltinTerminatorCall(call *ast.FunctionCallNode) bool {
	if call == nil || call.Name == nil {
		return false
	}
	identifier, ok := call.Name.(*ast.IdentifierNode)
	if !ok {
		return false
	}
	name := strings.TrimLeft(asciiLowerIdent(identifier.Value), `\`)
	return name == "exit" || name == "die"
}

func staticCallMethodName(name string) string {
	if index := strings.LastIndex(name, "::"); index >= 0 {
		return name[index+2:]
	}
	return name
}

func isPHPUnitNeverMethod(name string) bool {
	switch asciiLowerIdent(name) {
	case "fail", "marktestskipped", "marktestincomplete":
		return true
	default:
		return false
	}
}

func init() {
	RegisterAnalysisRuleWithLevel("Generic.CodeAnalysis.UnreachableCode", 4, "deadCode", func(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
		rule := &UnreachableCodeRule{}
		return rule.CheckIssuesWithContext(nodes, filename, ctx)
	})
}
