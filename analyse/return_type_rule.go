package analyse

import (
	"fmt"
	"go-phpcs/ast"
)

type ReturnTypeRule struct{}

type ReturnTypeError struct {
	FuncName     string
	DeclaredType string
	ActualType   string
	Pos          ast.Position
}

func (e *ReturnTypeError) Error() string {
	return fmt.Sprintf("Function %s: return type mismatch, declared: %s, actual: %s at %d:%d", e.FuncName, e.DeclaredType, e.ActualType, e.Pos.Line, e.Pos.Column)
}

func (r *ReturnTypeRule) CheckFunctionReturnType(fn *ast.FunctionNode) []error {
	if fn.ReturnType == "" {
		return nil // no declared return type, nothing to check
	}

	// Collect all actual return types
	returnTypes := map[string]int{}
	var firstMismatch *ReturnTypeError
	for _, ret := range collectReturnNodes(fn.Body) {
		actualType := inferType(ret.Expr)
		returnTypes[actualType]++
		if firstMismatch == nil && !typesCompatible(fn.ReturnType, actualType) {
			firstMismatch = &ReturnTypeError{
				FuncName:     fn.Name,
				DeclaredType: fn.ReturnType,
				ActualType:   actualType,
				Pos:          ret.GetPos(),
			}
		}
	}
	if len(returnTypes) > 0 && returnTypes[fn.ReturnType] == 0 {
		var foundTypes []string
		for t := range returnTypes {
			foundTypes = append(foundTypes, t)
		}
		return []error{&ReturnTypeError{
			FuncName:     fn.Name,
			DeclaredType: fn.ReturnType,
			ActualType:   fmt.Sprintf("%v", foundTypes),
			Pos:          fn.GetPos(),
		}}
	}
	if len(returnTypes) > 1 {
		var foundTypes []string
		for t := range returnTypes {
			foundTypes = append(foundTypes, t)
		}
		return []error{&ReturnTypeError{
			FuncName:     fn.Name,
			DeclaredType: fn.ReturnType,
			ActualType:   fmt.Sprintf("multiple: %v", foundTypes),
			Pos:          fn.GetPos(),
		}}
	}
	if firstMismatch != nil {
		return []error{firstMismatch}
	}
	return nil
}

func (r *ReturnTypeRule) CheckIssues(nodes []ast.Node, filename string) []AnalysisIssue {
	var issues []AnalysisIssue
	var checkFunc func(n ast.Node)
	checkFunc = func(n ast.Node) {
		switch fn := n.(type) {
		case *ast.FunctionNode:
			errs := r.CheckFunctionReturnType(fn)
			for _, err := range errs {
				pos := fn.GetPos()
				issues = append(issues, AnalysisIssue{
					Filename: filename,
					Line:     pos.Line,
					Column:   pos.Column,
					Code:     "A.RETURN.TYPE",
					Message:  err.Error(),
				})
			}
		case *ast.ClassNode:
			for _, m := range fn.Methods {
				checkFunc(m)
			}
		}
	}
	for _, n := range nodes {
		checkFunc(n)
	}
	return issues
}

func collectReturnNodes(nodes []ast.Node) []*ast.ReturnNode {
	var returns []*ast.ReturnNode
	for _, n := range nodes {
		switch n := n.(type) {
		case *ast.ReturnNode:
			returns = append(returns, n)
		case *ast.IfNode:
			returns = append(returns, collectReturnNodes(n.Body)...)
			for _, elseif := range n.ElseIfs {
				returns = append(returns, collectReturnNodes(elseif.Body)...)
			}
			if n.Else != nil {
				returns = append(returns, collectReturnNodes(n.Else.Body)...)
			}
		case *ast.BlockNode:
			returns = append(returns, collectReturnNodes(n.Statements)...)
		case *ast.WhileNode:
			returns = append(returns, collectReturnNodes(n.Body)...)
		case *ast.ForeachNode:
			returns = append(returns, collectReturnNodes(n.Body)...)
		}
	}
	return returns
}

func inferType(expr ast.Node) string {
	switch n := expr.(type) {
	case *ast.IntegerLiteral, *ast.IntegerNode:
		return "int"
	case *ast.FloatLiteral, *ast.FloatNode:
		return "float"
	case *ast.StringLiteral, *ast.InterpolatedStringLiteral, *ast.StringNode:
		return "string"
	case *ast.BooleanLiteral, *ast.BooleanNode:
		return "bool"
	case *ast.NullLiteral, *ast.NullNode:
		return "null"
	case *ast.VariableNode:
		// Try to infer from variable name or context if possible, else mixed
		return "mixed"
	case *ast.ExpressionStmt:
		return inferType(n.Expr)
	case *ast.TypeCastNode:
		return n.Type
	case nil:
		return "void"
	default:
		if lit, ok := n.(interface{ GetValue() interface{} }); ok {
			switch lit.GetValue().(type) {
			case int, int64:
				return "int"
			case float32, float64:
				return "float"
			case string:
				return "string"
			case bool:
				return "bool"
			case nil:
				return "null"
			}
		}
		return "mixed"
	}
}

func typesCompatible(declared, actual string) bool {
	if declared == actual {
		return true
	}
	if declared == "mixed" || actual == "mixed" {
		return true
	}
	if declared == "float" && actual == "int" {
		return true
	}
	if declared == "void" && actual == "null" {
		return true
	}
	return false
}

func init() {
	RegisterAnalysisRule("A.RETURN.TYPE", func(filename string, nodes []ast.Node) []AnalysisIssue {
		rule := &ReturnTypeRule{}
		return rule.CheckIssues(nodes, filename)
	})
}
