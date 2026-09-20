package analyse

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
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

		case *ast.DoWhileNode:
			for _, bodyNode := range node.Body {
				checkFunc(bodyNode)
			}
			addAssignmentIssues(node.Condition)

		case *ast.ForNode:
			for _, condition := range node.Conditions {
				addAssignmentIssues(condition)
			}
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

// CheckIssuesWithSource is a CST-direct test-isolation entry point (see
// empty_statement_rule.go's CheckIssuesWithSource for the established
// pattern). It parses raw source and walks *syntax.RedNode directly instead
// of lowering to []ast.Node first. Only statement-body containers (if/
// elseif/else/while/do-while/for/function/class) are recursed into; a
// condition subtree is only ever visited once, via the bounded
// findAssignmentsInCST walk, to avoid double-counting nested constructs
// (e.g. a match-expression embedded inside another statement's condition).
// Wired into the registered rule when ctx.Content is present; otherwise the
// registered rule falls back to CheckIssues (the ast.Node path).
func (r *AssignmentInConditionRule) CheckIssuesWithSource(filename string, content []byte) []AnalysisIssue {
	return r.checkIssuesFromParsedCST(filename, syntax.Parse(content))
}

func (r *AssignmentInConditionRule) checkIssuesFromParsedCST(filename string, res *syntax.ParseResult) []AnalysisIssue {
	if res == nil || res.File == nil || res.File.Root == nil {
		return nil
	}
	var issues []AnalysisIssue
	addCond := func(cond *syntax.RedNode) {
		for _, assign := range findAssignmentsInCST(cond) {
			pos := assign.Pos()
			issues = append(issues, AnalysisIssue{
				Filename: filename,
				Line:     pos.Line,
				Column:   pos.Column,
				Code:     "Generic.CodeAnalysis.AssignmentInCondition",
				Message:  "Assignment in condition",
			})
		}
	}
	var walkStmt func(n *syntax.RedNode)
	walkBody := func(body *syntax.RedNode) {
		for _, stmt := range syntax.StatementBodyList(body) {
			walkStmt(stmt)
		}
	}
	walkStmt = func(n *syntax.RedNode) {
		if n == nil {
			return
		}
		switch n.Kind() {
		case syntax.KindStatementList:
			walkBody(n)
		case syntax.KindIfStmt:
			addCond(syntax.IfCondition(n))
			walkBody(syntax.IfBody(n))
			for _, elseif := range syntax.IfElseIfs(n) {
				addCond(syntax.IfCondition(elseif))
				walkBody(syntax.ElseIfBody(elseif))
			}
			if els := syntax.IfElse(n); els != nil {
				walkBody(syntax.ElseBody(els))
			}
		case syntax.KindWhileStmt:
			addCond(syntax.WhileCondition(n))
			walkBody(syntax.WhileBody(n))
		case syntax.KindDoWhileStmt:
			walkBody(syntax.DoWhileBody(n))
			addCond(syntax.DoWhileCondition(n))
		case syntax.KindForStmt:
			for _, cond := range syntax.ForConditions(n) {
				addCond(cond)
			}
			walkBody(syntax.ForBody(n))
		case syntax.KindMatchExpr:
			addCond(syntax.MatchCondition(n))
			for _, cond := range syntax.MatchArmConditions(n) {
				addCond(cond)
			}
		case syntax.KindFunctionDecl, syntax.KindMethodDecl:
			walkBody(syntax.FunctionBody(n))
		case syntax.KindClassDecl:
			for _, method := range syntax.ClassMethods(n) {
				walkStmt(method)
			}
		}
	}
	for _, top := range res.File.Root.Children() {
		walkStmt(top)
	}
	return issues
}

// findAssignmentsInCST is the CST analogue of findAssignmentsInExpression:
// it recurses only into the same expression shapes the ast.Node version
// does (assign/binary/ternary/cast operands, unwrapped expression
// statements, parenthesized expressions, property-fetch objects, and
// plain-function-call arguments -- explicitly NOT method-call args, matching
// the absence of an *ast.MethodCallNode case in the original switch).
func findAssignmentsInCST(expr *syntax.RedNode) []*syntax.RedNode {
	if expr == nil {
		return nil
	}
	var found []*syntax.RedNode
	var visit func(n *syntax.RedNode) bool
	visit = func(n *syntax.RedNode) bool {
		switch n.Kind() {
		case syntax.KindAssignExpr:
			found = append(found, n)
			return true
		case syntax.KindBinaryExpr, syntax.KindTernaryExpr, syntax.KindCastExpr,
			syntax.KindExpressionStmt, syntax.KindParenExpr,
			syntax.KindMemberAccessExpr, syntax.KindNullsafeMemberAccessExpr,
			syntax.KindArgList, syntax.KindArg, syntax.KindNamedArg:
			return true
		case syntax.KindCallExpr:
			if !syntax.CallIsMethodLike(n) {
				if args := syntax.CallArgList(n); args != nil {
					syntax.Walk(args, visit)
				}
			}
			return false
		default:
			return false
		}
	}
	syntax.Walk(expr, visit)
	return found
}

// runRegisteredAssignmentInConditionRule is the body of the registered
// "Generic.CodeAnalysis.AssignmentInCondition" callback, split out so tests
// can invoke exactly what production runs without depending on global
// registry state.
func runRegisteredAssignmentInConditionRule(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
	rule := &AssignmentInConditionRule{}
	if len(ctx.Content) > 0 {
		res := sharedParseResult(ctx, ctx.Content)
		return rule.checkIssuesFromParsedCST(filename, res)
	}
	return rule.CheckIssues(nodes, filename)
}

func init() {
	RegisterAnalysisRuleWithContext("Generic.CodeAnalysis.AssignmentInCondition", runRegisteredAssignmentInConditionRule)
}
