package analyse

import (
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
)

// detectInstanceofNarrowing walks an if condition to find instanceof checks.
// Returns the variable and narrowed type on true branch.
func detectInstanceofNarrowing(condition ast.Node) (variable string, narrowedType string, ok bool) {
	if condition == nil {
		return "", "", false
	}

	switch cond := condition.(type) {
	case *ast.BinaryExpr:
		if cond.Operator == "instanceof" {
			varName := extractVariable(cond.Left)
			className := extractClassNameFromNode(cond.Right)
			if varName != "" && className != "" {
				return varName, className, true
			}
		}
	case *ast.UnaryExpr:
		if cond.Operator == "!" {
			return detectInstanceofNarrowing(cond.Operand)
		}
	}
	return "", "", false
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

// collectNarrowingFacts generates narrowing facts for a file's statements.
func collectNarrowingFacts(filename string, statements []ast.Node) []SemanticFact {
	var facts []SemanticFact
	walkStatementsForNarrowing(filename, statements, &facts)
	return facts
}

func walkStatementsForNarrowing(filename string, statements []ast.Node, facts *[]SemanticFact) {
	for _, stmt := range statements {
		walkNodeForNarrowing(filename, stmt, facts)
	}
}

func walkNodeForNarrowing(filename string, node ast.Node, facts *[]SemanticFact) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *ast.IfNode:
		if varName, className, ok := detectInstanceofNarrowing(n.Condition); ok {
			addNarrowingToBody(filename, varName, className, n.Body, facts)
		}
		walkStatementsForNarrowing(filename, n.Body, facts)
		for _, elseif := range n.ElseIfs {
			if varName, className, ok := detectInstanceofNarrowing(elseif.Condition); ok {
				addNarrowingToBody(filename, varName, className, elseif.Body, facts)
			}
			walkStatementsForNarrowing(filename, elseif.Body, facts)
		}
		if n.Else != nil {
			walkStatementsForNarrowing(filename, n.Else.Body, facts)
		}

	case *ast.BlockNode:
		walkStatementsForNarrowing(filename, n.Statements, facts)
	case *ast.WhileNode:
		walkStatementsForNarrowing(filename, n.Body, facts)
	case *ast.ForNode:
		walkStatementsForNarrowing(filename, n.Body, facts)
	case *ast.ForeachNode:
		walkStatementsForNarrowing(filename, n.Body, facts)
	case *ast.DoWhileNode:
		walkStatementsForNarrowing(filename, n.Body, facts)
	case *ast.SwitchNode:
		for _, caseNode := range n.Cases {
			walkStatementsForNarrowing(filename, caseNode.Body, facts)
		}
	case *ast.TryNode:
		walkStatementsForNarrowing(filename, n.Body, facts)
		for _, catchNode := range n.Catches {
			walkStatementsForNarrowing(filename, catchNode.Body, facts)
		}
		walkStatementsForNarrowing(filename, n.Finally, facts)
	case *ast.FunctionNode:
		walkStatementsForNarrowing(filename, n.Body, facts)
	case *ast.ClassNode:
		for _, method := range n.Methods {
			walkNodeForNarrowing(filename, method, facts)
		}
	}
}

func addNarrowingToBody(filename, varName, className string, body []ast.Node, facts *[]SemanticFact) {
	for _, stmt := range body {
		walkNodeAndAddNarrowing(filename, varName, className, stmt, facts)
	}
}

func walkNodeAndAddNarrowing(filename, varName, className string, node ast.Node, facts *[]SemanticFact) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *ast.VariableNode:
		if n.Name == varName {
			fact := SemanticFact{
				Key: SemanticFactKey{
					File:        filename,
					StartOffset: n.GetPos().Offset,
					EndOffset:   n.GetEndPos().Offset,
					Kind:        FactKindNarrowed,
				},
				Subject: SymbolID(className),
				Type:    className,
				Value:   "instanceof",
			}
			*facts = append(*facts, fact)
		}

	case *ast.BlockNode:
		for _, stmt := range n.Statements {
			walkNodeAndAddNarrowing(filename, varName, className, stmt, facts)
		}
	case *ast.ExpressionStmt:
		walkNodeAndAddNarrowing(filename, varName, className, n.Expr, facts)
	case *ast.BinaryExpr:
		walkNodeAndAddNarrowing(filename, varName, className, n.Left, facts)
		walkNodeAndAddNarrowing(filename, varName, className, n.Right, facts)
	case *ast.UnaryExpr:
		walkNodeAndAddNarrowing(filename, varName, className, n.Operand, facts)
	case *ast.FunctionCallNode:
		walkNodeAndAddNarrowing(filename, varName, className, n.Name, facts)
		for _, arg := range n.Args {
			walkNodeAndAddNarrowing(filename, varName, className, arg, facts)
		}
	case *ast.MethodCallNode:
		walkNodeAndAddNarrowing(filename, varName, className, n.Object, facts)
		for _, arg := range n.Args {
			walkNodeAndAddNarrowing(filename, varName, className, arg, facts)
		}
	case *ast.PropertyFetchNode:
		walkNodeAndAddNarrowing(filename, varName, className, n.Object, facts)
	case *ast.ArrayAccessNode:
		walkNodeAndAddNarrowing(filename, varName, className, n.Var, facts)
		if n.Index != nil {
			walkNodeAndAddNarrowing(filename, varName, className, n.Index, facts)
		}
	case *ast.IfNode:
		walkStatementsForNarrowing(filename, n.Body, facts)
	}
}
