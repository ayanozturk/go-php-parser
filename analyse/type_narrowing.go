package analyse

import (
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
)

type instanceofCondition struct {
	variable string
	target   ast.Node
	negated  bool
}

// parseInstanceofCondition recognizes `$x instanceof T`, `!($x instanceof T)`,
// and PHP's unparenthesized `!$x instanceof T` (parsed as `(!$x) instanceof T`).
func parseInstanceofCondition(condition ast.Node) (instanceofCondition, bool) {
	if condition == nil {
		return instanceofCondition{}, false
	}
	switch cond := condition.(type) {
	case *ast.UnaryExpr:
		if cond.Operator != "!" {
			return instanceofCondition{}, false
		}
		inner, ok := parseInstanceofCondition(cond.Operand)
		if !ok {
			return instanceofCondition{}, false
		}
		inner.negated = !inner.negated
		return inner, true
	case *ast.BinaryExpr:
		if cond.Operator != "instanceof" {
			return instanceofCondition{}, false
		}
		variable, negatedLeft, ok := instanceofSubject(cond.Left)
		if !ok {
			return instanceofCondition{}, false
		}
		return instanceofCondition{variable: variable, target: cond.Right, negated: negatedLeft}, true
	}
	return instanceofCondition{}, false
}

func instanceofSubject(node ast.Node) (string, bool, bool) {
	switch n := node.(type) {
	case *ast.VariableNode:
		return n.Name, false, true
	case *ast.UnaryExpr:
		if n.Operator == "!" {
			if variable, _, ok := instanceofSubject(n.Operand); ok {
				return variable, true, true
			}
		}
	}
	return "", false, false
}

// detectInstanceofNarrowing walks an if condition to find instanceof checks.
// Returns the variable and narrowed type on true branch.
func detectInstanceofNarrowing(condition ast.Node) (variable string, narrowedType string, ok bool) {
	cond, parsed := parseInstanceofCondition(condition)
	if !parsed || cond.negated {
		return "", "", false
	}
	className := extractClassNameFromNode(cond.target)
	if cond.variable == "" || className == "" {
		return "", "", false
	}
	return cond.variable, className, true
}

func extractVariable(node ast.Node) string {
	if node == nil {
		return ""
	}
	if v, ok := node.(*ast.VariableNode); ok {
		return v.Name
	}
	return ""
}

func extractClassNameFromNode(node ast.Node) string {
	if node == nil {
		return ""
	}
	switch n := node.(type) {
	case *ast.IdentifierNode:
		parts := strings.Split(n.Value, "\\")
		return parts[len(parts)-1]
	case *ast.VariableNode:
		return ""
	}
	return ""
}

// insertNarrowingFacts writes instanceof narrowing facts for a file directly
// into the snapshot store. Duplicate spans keep the first fact.
func insertNarrowingFacts(store semanticFactStore, filename string, statements []ast.Node) {
	walkStatementsForNarrowing(store, filename, statements)
}

func walkStatementsForNarrowing(store semanticFactStore, filename string, statements []ast.Node) {
	for i, stmt := range statements {
		walkNodeForNarrowing(store, filename, stmt)
		ifNode, ok := stmt.(*ast.IfNode)
		if !ok {
			continue
		}
		cond, parsed := parseInstanceofCondition(ifNode.Condition)
		if !parsed || !cond.negated || !statementsExitCurrentBlock(ifNode.Body) {
			continue
		}
		className := extractClassNameFromNode(cond.target)
		if className == "" {
			continue
		}
		for _, later := range statements[i+1:] {
			walkNodeAndAddNarrowing(store, filename, cond.variable, className, later)
		}
	}
}

func walkNodeForNarrowing(store semanticFactStore, filename string, node ast.Node) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *ast.IfNode:
		if varName, className, ok := detectInstanceofNarrowing(n.Condition); ok {
			addNarrowingToBody(store, filename, varName, className, n.Body)
		}
		walkStatementsForNarrowing(store, filename, n.Body)
		for _, elseif := range n.ElseIfs {
			if varName, className, ok := detectInstanceofNarrowing(elseif.Condition); ok {
				addNarrowingToBody(store, filename, varName, className, elseif.Body)
			}
			walkStatementsForNarrowing(store, filename, elseif.Body)
		}
		if n.Else != nil {
			walkStatementsForNarrowing(store, filename, n.Else.Body)
		}

	case *ast.BlockNode:
		walkStatementsForNarrowing(store, filename, n.Statements)
	case *ast.WhileNode:
		walkStatementsForNarrowing(store, filename, n.Body)
	case *ast.ForNode:
		walkStatementsForNarrowing(store, filename, n.Body)
	case *ast.ForeachNode:
		walkStatementsForNarrowing(store, filename, n.Body)
	case *ast.DoWhileNode:
		walkStatementsForNarrowing(store, filename, n.Body)
	case *ast.SwitchNode:
		for _, caseNode := range n.Cases {
			walkStatementsForNarrowing(store, filename, caseNode.Body)
		}
	case *ast.TryNode:
		walkStatementsForNarrowing(store, filename, n.Body)
		for _, catchNode := range n.Catches {
			walkStatementsForNarrowing(store, filename, catchNode.Body)
		}
		walkStatementsForNarrowing(store, filename, n.Finally)
	case *ast.FunctionNode:
		walkStatementsForNarrowing(store, filename, n.Body)
	case *ast.ClassNode:
		for _, method := range n.Methods {
			walkNodeForNarrowing(store, filename, method)
		}
	}
}

func addNarrowingToBody(store semanticFactStore, filename, varName, className string, body []ast.Node) {
	for _, stmt := range body {
		walkNodeAndAddNarrowing(store, filename, varName, className, stmt)
	}
}

func walkNodeAndAddNarrowing(store semanticFactStore, filename, varName, className string, node ast.Node) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *ast.VariableNode:
		if n.Name == varName {
			start, end := n.GetPos(), n.GetEndPos()
			if end.Offset <= start.Offset {
				return
			}
			store.putParts(SemanticFactKey{
				File:        filename,
				StartOffset: start.Offset,
				EndOffset:   end.Offset,
				Kind:        FactKindNarrowed,
			}, SymbolID(className), className, "instanceof")
		}

	case *ast.BlockNode:
		for _, stmt := range n.Statements {
			walkNodeAndAddNarrowing(store, filename, varName, className, stmt)
		}
	case *ast.ExpressionStmt:
		walkNodeAndAddNarrowing(store, filename, varName, className, n.Expr)
	case *ast.BinaryExpr:
		walkNodeAndAddNarrowing(store, filename, varName, className, n.Left)
		walkNodeAndAddNarrowing(store, filename, varName, className, n.Right)
	case *ast.UnaryExpr:
		walkNodeAndAddNarrowing(store, filename, varName, className, n.Operand)
	case *ast.FunctionCallNode:
		walkNodeAndAddNarrowing(store, filename, varName, className, n.Name)
		for _, arg := range n.Args {
			walkNodeAndAddNarrowing(store, filename, varName, className, arg)
		}
	case *ast.MethodCallNode:
		walkNodeAndAddNarrowing(store, filename, varName, className, n.Object)
		for _, arg := range n.Args {
			walkNodeAndAddNarrowing(store, filename, varName, className, arg)
		}
	case *ast.PropertyFetchNode:
		walkNodeAndAddNarrowing(store, filename, varName, className, n.Object)
	case *ast.ArrayAccessNode:
		walkNodeAndAddNarrowing(store, filename, varName, className, n.Var)
		if n.Index != nil {
			walkNodeAndAddNarrowing(store, filename, varName, className, n.Index)
		}
	case *ast.AssignmentNode:
		walkNodeAndAddNarrowing(store, filename, varName, className, n.Left)
		walkNodeAndAddNarrowing(store, filename, varName, className, n.Right)
	case *ast.IfNode:
		walkStatementsForNarrowing(store, filename, n.Body)
	}
}
