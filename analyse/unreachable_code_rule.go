package analyse

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"strings"
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
	issues := make([]AnalysisIssue, 0, 4)
	r.walkStatements(nodes, filename, ctx, &issues)
	return issues
}

func (r *UnreachableCodeRule) walkStatements(stmts []ast.Node, filename string, ctx *AnalysisContext, issues *[]AnalysisIssue) {
	terminated := false
	for _, stmt := range stmts {
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

func isTerminatingStatement(node ast.Node) bool {
	switch n := node.(type) {
	case *ast.ReturnNode, *ast.ThrowNode, *ast.BreakNode, *ast.ContinueNode:
		return true
	case *ast.ExpressionStmt:
		return isTerminatingStatement(n.Expr)
	case *ast.FunctionCallNode:
		return isBuiltinTerminatorCall(n)
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

func init() {
	RegisterAnalysisRuleWithLevel("Generic.CodeAnalysis.UnreachableCode", 4, "deadCode", func(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
		rule := &UnreachableCodeRule{}
		return rule.CheckIssuesWithContext(nodes, filename, ctx)
	})
}
