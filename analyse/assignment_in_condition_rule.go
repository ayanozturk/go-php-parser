package analyse

import (
	"go-phpcs/ast"
)

// AssignmentInConditionRule detects assignments inside conditional statements
type AssignmentInConditionRule struct{}

// CheckIssues walks through all nodes and checks for assignments in condition expressions
func (r *AssignmentInConditionRule) CheckIssues(nodes []ast.Node, filename string) []AnalysisIssue {
	var issues []AnalysisIssue
	var checkFunc func(n ast.Node)
	checkFunc = func(n ast.Node) {
		switch node := n.(type) {
		case *ast.IfNode:
			// Check main if condition
			if assignment := r.findAssignmentInExpression(node.Condition); assignment != nil {
				issues = append(issues, AnalysisIssue{
					Filename: filename,
					Line:     assignment.GetPos().Line,
					Column:   assignment.GetPos().Column,
					Code:     "Generic.CodeAnalysis.AssignmentInCondition",
					Message:  "Assignment in condition",
				})
			}
			// Check elseif conditions
			for _, elseif := range node.ElseIfs {
				if assignment := r.findAssignmentInExpression(elseif.Condition); assignment != nil {
					issues = append(issues, AnalysisIssue{
						Filename: filename,
						Line:     assignment.GetPos().Line,
						Column:   assignment.GetPos().Column,
						Code:     "Generic.CodeAnalysis.AssignmentInCondition",
						Message:  "Assignment in condition",
					})
				}
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
			if assignment := r.findAssignmentInExpression(node.Condition); assignment != nil {
				issues = append(issues, AnalysisIssue{
					Filename: filename,
					Line:     assignment.GetPos().Line,
					Column:   assignment.GetPos().Column,
					Code:     "Generic.CodeAnalysis.AssignmentInCondition",
					Message:  "Assignment in condition",
				})
			}
			// Recursively check while body
			for _, bodyNode := range node.Body {
				checkFunc(bodyNode)
			}

		case *ast.MatchNode:
			// Check match condition
			if assignment := r.findAssignmentInExpression(node.Condition); assignment != nil {
				issues = append(issues, AnalysisIssue{
					Filename: filename,
					Line:     assignment.GetPos().Line,
					Column:   assignment.GetPos().Column,
					Code:     "Generic.CodeAnalysis.AssignmentInCondition",
					Message:  "Assignment in condition",
				})
			}
			// Check match arm conditions
			for _, arm := range node.Arms {
				for _, condition := range arm.Conditions {
					if assignment := r.findAssignmentInExpression(condition); assignment != nil {
						issues = append(issues, AnalysisIssue{
							Filename: filename,
							Line:     assignment.GetPos().Line,
							Column:   assignment.GetPos().Column,
							Code:     "Generic.CodeAnalysis.AssignmentInCondition",
							Message:  "Assignment in condition",
						})
					}
				}
			}

		case *ast.TernaryExpr:
			// Check ternary condition
			if assignment := r.findAssignmentInExpression(node.Condition); assignment != nil {
				issues = append(issues, AnalysisIssue{
					Filename: filename,
					Line:     assignment.GetPos().Line,
					Column:   assignment.GetPos().Column,
					Code:     "Generic.CodeAnalysis.AssignmentInCondition",
					Message:  "Assignment in condition",
				})
			}

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

// findAssignmentInExpression recursively searches for assignment nodes within an expression
func (r *AssignmentInConditionRule) findAssignmentInExpression(expr ast.Node) *ast.AssignmentNode {
	if expr == nil {
		return nil
	}

	switch node := expr.(type) {
	case *ast.AssignmentNode:
		return node

	case *ast.BinaryExpr:
		// Check both sides of binary expressions
		if left := r.findAssignmentInExpression(node.Left); left != nil {
			return left
		}
		if right := r.findAssignmentInExpression(node.Right); right != nil {
			return right
		}

	case *ast.ExpressionStmt:
		// Unwrap expression statements
		return r.findAssignmentInExpression(node.Expr)

	case *ast.FunctionCallNode:
		// Check function call arguments
		for _, arg := range node.Args {
			if assignment := r.findAssignmentInExpression(arg); assignment != nil {
				return assignment
			}
		}

	case *ast.PropertyFetchNode:
		// Check property fetch expressions
		if assignment := r.findAssignmentInExpression(node.Object); assignment != nil {
			return assignment
		}

	case *ast.ConcatNode:
		// Check concatenation parts
		for _, part := range node.Parts {
			if assignment := r.findAssignmentInExpression(part); assignment != nil {
				return assignment
			}
		}

	case *ast.TypeCastNode:
		// Check type cast expression
		return r.findAssignmentInExpression(node.Expr)

	case *ast.TernaryExpr:
		// Check all parts of ternary expression
		if assignment := r.findAssignmentInExpression(node.Condition); assignment != nil {
			return assignment
		}
		if assignment := r.findAssignmentInExpression(node.IfTrue); assignment != nil {
			return assignment
		}
		if assignment := r.findAssignmentInExpression(node.IfFalse); assignment != nil {
			return assignment
		}
	}

	return nil
}

func init() {
	RegisterAnalysisRule("Generic.CodeAnalysis.AssignmentInCondition", func(filename string, nodes []ast.Node) []AnalysisIssue {
		rule := &AssignmentInConditionRule{}
		return rule.CheckIssues(nodes, filename)
	})
}
