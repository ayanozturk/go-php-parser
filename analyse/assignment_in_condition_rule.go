package analyse

import (
	"go-phpcs/ast"
)

// AssignmentInConditionRule detects assignments inside conditional statements
type AssignmentInConditionRule struct{}

// CheckIssues walks through all nodes and checks for assignments in condition expressions
func (r *AssignmentInConditionRule) CheckIssues(nodes []ast.Node, filename string) []AnalysisIssue {
	var issues []AnalysisIssue
	addAssignmentIssues := func(expr ast.Node) {
		for _, assignment := range r.findAssignmentsInExpression(expr) {
			issues = append(issues, AnalysisIssue{
				Filename: filename,
				Line:     assignment.GetPos().Line,
				Column:   assignment.GetPos().Column,
				Code:     "Generic.CodeAnalysis.AssignmentInCondition",
				Message:  "Assignment in condition",
			})
		}
	}
	var checkFunc func(n ast.Node)
	checkFunc = func(n ast.Node) {
		switch node := n.(type) {
		case *ast.IfNode:
			// Check main if condition
			addAssignmentIssues(node.Condition)
			// Check elseif conditions
			for _, elseif := range node.ElseIfs {
				addAssignmentIssues(elseif.Condition)
				// Recursively check elseif body
				for _, bodyNode := range elseif.Body {
					checkFunc(bodyNode)
				}
			}
			// Recursively check if body
			for _, bodyNode := range node.Body {
				checkFunc(bodyNode)
			}
			// Recursively check else body
			if node.Else != nil {
				for _, bodyNode := range node.Else.Body {
					checkFunc(bodyNode)
				}
			}

		case *ast.WhileNode:
			// Check while condition
			addAssignmentIssues(node.Condition)
			// Recursively check while body
			for _, bodyNode := range node.Body {
				checkFunc(bodyNode)
			}

		case *ast.MatchNode:
			// Check match condition
			addAssignmentIssues(node.Condition)
			// Check match arm conditions
			for _, arm := range node.Arms {
				for _, condition := range arm.Conditions {
					addAssignmentIssues(condition)
				}
			}

		case *ast.TernaryExpr:
			// Check ternary condition
			addAssignmentIssues(node.Condition)

		case *ast.FunctionNode:
			// Recursively check function body
			for _, bodyNode := range node.Body {
				checkFunc(bodyNode)
			}

		case *ast.ClassNode:
			// Recursively check class methods
			for _, method := range node.Methods {
				checkFunc(method)
			}

		case *ast.BlockNode:
			// Recursively check block statements
			for _, stmt := range node.Statements {
				checkFunc(stmt)
			}
		}
	}

	for _, n := range nodes {
		checkFunc(n)
	}

	return issues
}

// findAssignmentsInExpression recursively searches for all assignment nodes within an expression.
func (r *AssignmentInConditionRule) findAssignmentsInExpression(expr ast.Node) []*ast.AssignmentNode {
	if expr == nil {
		return nil
	}

	switch node := expr.(type) {
	case *ast.AssignmentNode:
		assignments := []*ast.AssignmentNode{node}
		assignments = append(assignments, r.findAssignmentsInExpression(node.Left)...)
		assignments = append(assignments, r.findAssignmentsInExpression(node.Right)...)
		return assignments

	case *ast.BinaryExpr:
		// Check both sides of binary expressions
		var assignments []*ast.AssignmentNode
		assignments = append(assignments, r.findAssignmentsInExpression(node.Left)...)
		assignments = append(assignments, r.findAssignmentsInExpression(node.Right)...)
		return assignments

	case *ast.ExpressionStmt:
		// Unwrap expression statements
		return r.findAssignmentsInExpression(node.Expr)

	case *ast.FunctionCallNode:
		// Check function call arguments
		var assignments []*ast.AssignmentNode
		for _, arg := range node.Args {
			assignments = append(assignments, r.findAssignmentsInExpression(arg)...)
		}
		return assignments

	case *ast.PropertyFetchNode:
		// Check property fetch expressions
		return r.findAssignmentsInExpression(node.Object)

	case *ast.ConcatNode:
		// Check concatenation parts
		var assignments []*ast.AssignmentNode
		for _, part := range node.Parts {
			assignments = append(assignments, r.findAssignmentsInExpression(part)...)
		}
		return assignments

	case *ast.TypeCastNode:
		// Check type cast expression
		return r.findAssignmentsInExpression(node.Expr)

	case *ast.TernaryExpr:
		// Check all parts of ternary expression
		var assignments []*ast.AssignmentNode
		assignments = append(assignments, r.findAssignmentsInExpression(node.Condition)...)
		assignments = append(assignments, r.findAssignmentsInExpression(node.IfTrue)...)
		assignments = append(assignments, r.findAssignmentsInExpression(node.IfFalse)...)
		return assignments
	}

	return nil
}

func init() {
	RegisterAnalysisRule("Generic.CodeAnalysis.AssignmentInCondition", func(filename string, nodes []ast.Node) []AnalysisIssue {
		rule := &AssignmentInConditionRule{}
		return rule.CheckIssues(nodes, filename)
	})
}
