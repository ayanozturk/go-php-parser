package analyse

import "go-phpcs/ast"

// UnreachableCodeRule reports statements that can never execute because a
// previous statement in the same block always terminates control flow.
//
// Inspired by modern static analyzers (like mago), this keeps the check cheap by
// performing a single pass per block and short-circuiting once a terminator is found.
type UnreachableCodeRule struct{}

func (r *UnreachableCodeRule) CheckIssues(nodes []ast.Node, filename string) []AnalysisIssue {
	issues := make([]AnalysisIssue, 0, 4)
	r.walkStatements(nodes, filename, &issues)
	return issues
}

func (r *UnreachableCodeRule) walkStatements(stmts []ast.Node, filename string, issues *[]AnalysisIssue) {
	terminated := false
	for _, stmt := range stmts {
		if terminated {
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

		r.walkChildren(stmt, filename, issues)
		if isTerminatingStatement(stmt) {
			terminated = true
		}
	}
}

func (r *UnreachableCodeRule) walkChildren(node ast.Node, filename string, issues *[]AnalysisIssue) {
	switch n := node.(type) {
	case *ast.FunctionNode:
		r.walkStatements(n.Body, filename, issues)
	case *ast.ClassNode:
		for _, m := range n.Methods {
			r.walkChildren(m, filename, issues)
		}
	case *ast.BlockNode:
		r.walkStatements(n.Statements, filename, issues)
	case *ast.IfNode:
		r.walkStatements(n.Body, filename, issues)
		for _, elseif := range n.ElseIfs {
			r.walkStatements(elseif.Body, filename, issues)
		}
		if n.Else != nil {
			r.walkStatements(n.Else.Body, filename, issues)
		}
	case *ast.WhileNode:
		r.walkStatements(n.Body, filename, issues)
	case *ast.ForeachNode:
		r.walkStatements(n.Body, filename, issues)
	case *ast.NamespaceNode:
		r.walkStatements(n.Body, filename, issues)
	}
}

func isTerminatingStatement(node ast.Node) bool {
	switch node.(type) {
	case *ast.ReturnNode, *ast.ThrowNode:
		return true
	}
	return false
}

func init() {
	RegisterAnalysisRule("Generic.CodeAnalysis.UnreachableCode", func(filename string, nodes []ast.Node) []AnalysisIssue {
		rule := &UnreachableCodeRule{}
		return rule.CheckIssues(nodes, filename)
	})
}
