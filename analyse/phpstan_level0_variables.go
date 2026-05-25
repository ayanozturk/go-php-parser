package analyse

import (
	"fmt"
	"go-phpcs/ast"
	"strings"
)

func (r *PHPStanLevel0Rule) checkUndefinedVariables(filename string, nodes []ast.Node, ctx *AnalysisContext, fileCtx fileTypeContext) []AnalysisIssue {
	var issues []AnalysisIssue
	var walkStatements func([]ast.Node, map[string]bool, *ast.ClassNode, bool) map[string]bool
	walkStatements = func(stmts []ast.Node, defined map[string]bool, class *ast.ClassNode, inFunction bool) map[string]bool {
		for _, stmt := range stmts {
			switch n := stmt.(type) {
			case *ast.FunctionNode:
				local := map[string]bool{}
				for _, param := range n.Params {
					if p, ok := param.(*ast.ParamNode); ok {
						local[p.Name] = true
					}
				}
				if class != nil && !hasModifier(n.Modifiers, "static") {
					local["this"] = true
				}
				walkStatements(n.Body, local, class, true)
			case *ast.ClassNode:
				for _, method := range n.Methods {
					if fn, ok := method.(*ast.FunctionNode); ok {
						walkStatements([]ast.Node{fn}, defined, n, false)
					}
				}
			case *ast.NamespaceNode:
				defined = walkStatements(n.Body, defined, class, inFunction)
			case *ast.StaticVarDeclNode:
				for _, entry := range n.Vars {
					if entry.Init != nil {
						checkExprVars(filename, entry.Init, defined, &issues)
					}
					defined[entry.Name] = true
				}
			case *ast.AssignmentNode:
				checkExprVars(filename, n.Right, defined, &issues)
				defineAssignmentTarget(n.Left, defined)
			case *ast.ExpressionStmt:
				if assign, ok := n.Expr.(*ast.AssignmentNode); ok {
					checkExprVars(filename, assign.Right, defined, &issues)
					defineAssignmentTarget(assign.Left, defined)
				} else {
					checkExprVars(filename, n.Expr, defined, &issues)
				}
			case *ast.ReturnNode:
				checkExprVars(filename, n.Expr, defined, &issues)
			case *ast.ThrowNode:
				checkExprVars(filename, n.Expr, defined, &issues)
			case *ast.IfNode:
				checkExprVars(filename, n.Condition, defined, &issues)
				before := cloneBoolMap(defined)
				thenDefined := walkStatements(n.Body, cloneBoolMap(defined), class, inFunction)
				branchUnion := cloneBoolMap(before)
				for k := range thenDefined {
					branchUnion[k] = true
				}
				for _, elseif := range n.ElseIfs {
					checkExprVars(filename, elseif.Condition, before, &issues)
					ed := walkStatements(elseif.Body, cloneBoolMap(before), class, inFunction)
					for k := range ed {
						branchUnion[k] = true
					}
				}
				if n.Else != nil {
					ed := walkStatements(n.Else.Body, cloneBoolMap(before), class, inFunction)
					for k := range ed {
						branchUnion[k] = true
					}
				}
				defined = branchUnion
			case *ast.ForeachNode:
				checkExprVars(filename, n.Expr, defined, &issues)
				defineAssignmentTarget(n.KeyVar, defined)
				defineAssignmentTarget(n.ValueVar, defined)
				defined = walkStatements(n.Body, defined, class, inFunction)
			case *ast.TryNode:
				defined = walkStatements(n.Body, defined, class, inFunction)
				for _, catchNode := range n.Catches {
					catchDefined := cloneBoolMap(defined)
					if catchNode.Variable != "" {
						catchDefined[strings.TrimPrefix(catchNode.Variable, "$")] = true
					}
					walkStatements(catchNode.Body, catchDefined, class, inFunction)
				}
				defined = walkStatements(n.Finally, defined, class, inFunction)
			}
		}
		return defined
	}
	defined := map[string]bool{"GLOBALS": true, "_SERVER": true, "_GET": true, "_POST": true, "_FILES": true, "_COOKIE": true, "_SESSION": true, "_REQUEST": true, "_ENV": true, "argc": true, "argv": true}
	walkStatements(nodes, defined, nil, false)
	return issues
}

func checkExprVars(filename string, node ast.Node, defined map[string]bool, issues *[]AnalysisIssue) {
	if node == nil {
		return
	}
	switch n := node.(type) {
	case *ast.VariableNode:
		if !defined[n.Name] {
			*issues = append(*issues, issue(filename, n.GetPos(), level0VariablesCode, fmt.Sprintf("Undefined variable: $%s", n.Name)))
		}
	case *ast.AssignmentNode:
		checkExprVars(filename, n.Right, defined, issues)
		defineAssignmentTarget(n.Left, defined)
	case *ast.FunctionCallNode:
		name := strings.ToLower(functionCallName(n))
		if name == "isset" || name == "empty" {
			return
		}
		if name == "compact" {
			for _, arg := range n.Args {
				if variableName, ok := stringLiteralValue(argumentValue(arg)); ok && !defined[variableName] {
					*issues = append(*issues, issue(filename, arg.GetPos(), level0VariablesCode, fmt.Sprintf("Undefined variable: $%s", variableName)))
				}
			}
			return
		}
		for _, arg := range n.Args {
			checkExprVars(filename, argumentValue(arg), defined, issues)
		}
	case *ast.MethodCallNode:
		checkExprVars(filename, n.Object, defined, issues)
		for _, arg := range n.Args {
			checkExprVars(filename, argumentValue(arg), defined, issues)
		}
	case *ast.NewNode:
		for _, arg := range n.Args {
			checkExprVars(filename, argumentValue(arg), defined, issues)
		}
	case *ast.PropertyFetchNode:
		checkExprVars(filename, n.Object, defined, issues)
	case *ast.ArrayAccessNode:
		checkExprVars(filename, n.Var, defined, issues)
		checkExprVars(filename, n.Index, defined, issues)
	case *ast.BinaryExpr:
		checkExprVars(filename, n.Left, defined, issues)
		checkExprVars(filename, n.Right, defined, issues)
	case *ast.UnaryExpr:
		checkExprVars(filename, n.Operand, defined, issues)
	case *ast.TernaryExpr:
		checkExprVars(filename, n.Condition, defined, issues)
		checkExprVars(filename, n.IfTrue, defined, issues)
		checkExprVars(filename, n.IfFalse, defined, issues)
	case *ast.ArrayNode:
		for _, element := range n.Elements {
			checkExprVars(filename, element, defined, issues)
		}
	case *ast.ArrayItemNode:
		checkExprVars(filename, n.Key, defined, issues)
		checkExprVars(filename, n.Value, defined, issues)
	case *ast.NamedArgumentNode:
		checkExprVars(filename, n.Value, defined, issues)
	case *ast.UnpackedArgumentNode:
		checkExprVars(filename, n.Expr, defined, issues)
	case *ast.ConcatNode:
		for _, part := range n.Parts {
			checkExprVars(filename, part, defined, issues)
		}
	}
}

func defineAssignmentTarget(node ast.Node, defined map[string]bool) {
	switch n := node.(type) {
	case *ast.VariableNode:
		defined[n.Name] = true
	case *ast.ArrayAccessNode:
		defineAssignmentTarget(n.Var, defined)
	case *ast.PropertyFetchNode:
		defineAssignmentTarget(n.Object, defined)
	}
}
