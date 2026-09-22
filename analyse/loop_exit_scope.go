package analyse

import (
	"strconv"
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
)

// applyGuaranteedWhileExitAssignments carries types assigned on every path
// that can leave a definitely-entered while loop. It deliberately ignores
// later iterations: requiring the assignment on every first-iteration
// back-edge/break path is conservative and sufficient for retry loops whose
// successful path returns and whose recoverable catches record the failure.
func applyGuaranteedWhileExitAssignments(prefix []ast.Node, loop *ast.WhileNode, scope *functionScope, ctx *AnalysisContext) {
	if loop == nil || scope == nil || !whileConditionTrueAtEntry(prefix, loop.Condition, scope) {
		return
	}

	entry := loopFlowPath{scope: scopeForConditionTrue(scope, loop.Condition, ctx), assigned: map[string]struct{}{}}
	result := simulateLoopFlowStatements(loop.Body, []loopFlowPath{entry}, ctx)
	exits := append(append([]loopFlowPath(nil), result.normal...), result.continues...)
	exits = append(exits, result.breaks...)
	if len(exits) == 0 {
		return
	}

	for name := range exits[0].assigned {
		var joined Type
		for index, exit := range exits {
			if _, ok := exit.assigned[name]; !ok || exit.scope == nil {
				joined = EmptyType()
				break
			}
			typ, ok := exit.scope.variable(name)
			if !ok || typ.IsEmpty() {
				joined = EmptyType()
				break
			}
			if index == 0 {
				joined = typ
			} else {
				joined = unionInferredTypes(joined, typ)
			}
		}
		if !joined.IsEmpty() {
			scope.setVariable(name, joined)
		}
	}
}

type loopFlowPath struct {
	scope    *functionScope
	assigned map[string]struct{}
}

type loopFlowResult struct {
	normal    []loopFlowPath
	breaks    []loopFlowPath
	continues []loopFlowPath
}

func simulateLoopFlowStatements(statements []ast.Node, inputs []loopFlowPath, ctx *AnalysisContext) loopFlowResult {
	result := loopFlowResult{normal: inputs}
	for _, statement := range statements {
		if len(result.normal) == 0 {
			break
		}
		next := loopFlowResult{}
		for _, input := range result.normal {
			step := simulateLoopFlowStatement(statement, input, ctx)
			next.normal = append(next.normal, step.normal...)
			next.breaks = append(next.breaks, step.breaks...)
			next.continues = append(next.continues, step.continues...)
		}
		result.normal = next.normal
		result.breaks = append(result.breaks, next.breaks...)
		result.continues = append(result.continues, next.continues...)
	}
	return result
}

func simulateLoopFlowStatement(statement ast.Node, input loopFlowPath, ctx *AnalysisContext) loopFlowResult {
	switch node := statement.(type) {
	case *ast.AssignmentNode:
		applyAssignmentScope(input.scope, node, ctx)
		markLoopFlowAssignment(&input, node)
		return loopFlowResult{normal: []loopFlowPath{input}}
	case *ast.ExpressionStmt:
		applyExpressionStmtScope(input.scope, node, ctx)
		markLoopFlowAssignment(&input, assignmentExpr(node.Expr))
		return loopFlowResult{normal: []loopFlowPath{input}}
	case *ast.ReturnNode, *ast.ThrowNode:
		return loopFlowResult{}
	case *ast.BreakNode:
		if loopControlLevelOne(node.Level) {
			return loopFlowResult{breaks: []loopFlowPath{input}}
		}
		// A multi-level break also leaves this loop. The enclosing target may
		// be outside the simulated body, so preserve it as a possible exit.
		return loopFlowResult{breaks: []loopFlowPath{input}}
	case *ast.ContinueNode:
		if loopControlLevelOne(node.Level) {
			return loopFlowResult{continues: []loopFlowPath{input}}
		}
		// continue N transfers control to an enclosing loop and therefore
		// leaves the loop whose exit assignments are being summarized.
		return loopFlowResult{breaks: []loopFlowPath{input}}
	case *ast.BlockNode:
		return simulateLoopFlowStatements(node.Statements, []loopFlowPath{input}, ctx)
	case *ast.IfNode:
		return simulateLoopFlowIf(node, input, ctx)
	case *ast.TryNode:
		return simulateLoopFlowTry(node, input, ctx)
	default:
		return loopFlowResult{normal: []loopFlowPath{input}}
	}
}

func simulateLoopFlowIf(node *ast.IfNode, input loopFlowPath, ctx *AnalysisContext) loopFlowResult {
	branches := []loopFlowResult{
		simulateLoopFlowStatements(node.Body, []loopFlowPath{cloneLoopFlowPath(input, scopeForConditionTrue(input.scope, node.Condition, ctx))}, ctx),
	}
	for _, elseif := range node.ElseIfs {
		branches = append(branches, simulateLoopFlowStatements(elseif.Body, []loopFlowPath{cloneLoopFlowPath(input, scopeForConditionTrue(input.scope, elseif.Condition, ctx))}, ctx))
	}
	if node.Else != nil {
		branches = append(branches, simulateLoopFlowStatements(node.Else.Body, []loopFlowPath{cloneLoopFlowPath(input, scopeForConditionFalse(input.scope, node.Condition, ctx))}, ctx))
	} else {
		branches = append(branches, loopFlowResult{normal: []loopFlowPath{cloneLoopFlowPath(input, input.scope.clone())}})
	}
	return mergeLoopFlowResults(branches...)
}

func simulateLoopFlowTry(node *ast.TryNode, input loopFlowPath, ctx *AnalysisContext) loopFlowResult {
	if len(node.Finally) > 0 {
		// Finally control transfers need resume-aware handling. Retaining an
		// unmodified path prevents narrowing until that is modeled explicitly.
		return loopFlowResult{normal: []loopFlowPath{input}}
	}
	branches := []loopFlowResult{
		simulateLoopFlowStatements(node.Body, []loopFlowPath{cloneLoopFlowPath(input, input.scope.clone())}, ctx),
	}
	for _, catch := range node.Catches {
		catchScope := input.scope.clone()
		if catchScope != nil && catch.Variable != "" {
			var types []string
			for _, raw := range catch.Types {
				types = append(types, catchScope.typeCtx.resolveClassLike(raw))
			}
			if typ := ParseType(strings.Join(types, "|")); !typ.IsEmpty() {
				catchScope.setVariable(strings.TrimPrefix(catch.Variable, "$"), typ)
			}
		}
		branches = append(branches, simulateLoopFlowStatements(catch.Body, []loopFlowPath{cloneLoopFlowPath(input, catchScope)}, ctx))
	}
	return mergeLoopFlowResults(branches...)
}

func mergeLoopFlowResults(results ...loopFlowResult) loopFlowResult {
	var merged loopFlowResult
	for _, result := range results {
		merged.normal = append(merged.normal, result.normal...)
		merged.breaks = append(merged.breaks, result.breaks...)
		merged.continues = append(merged.continues, result.continues...)
	}
	return merged
}

func cloneLoopFlowPath(path loopFlowPath, scope *functionScope) loopFlowPath {
	assigned := make(map[string]struct{}, len(path.assigned))
	for name := range path.assigned {
		assigned[name] = struct{}{}
	}
	return loopFlowPath{scope: scope, assigned: assigned}
}

func markLoopFlowAssignment(path *loopFlowPath, assignment *ast.AssignmentNode) {
	if path == nil || assignment == nil {
		return
	}
	if variable, ok := assignment.Left.(*ast.VariableNode); ok && variable.Name != "" {
		path.assigned[variable.Name] = struct{}{}
	}
}

func whileConditionTrueAtEntry(prefix []ast.Node, condition ast.Node, scope *functionScope) bool {
	binary, ok := condition.(*ast.BinaryExpr)
	if !ok {
		return false
	}
	variable, ok := binary.Left.(*ast.VariableNode)
	if !ok || variable.Name == "" {
		return false
	}
	left, ok := precedingIntegerAssignment(prefix, variable.Name)
	if !ok {
		return false
	}
	right, ok := loopIntegerValue(binary.Right, scope)
	if !ok {
		return false
	}
	switch binary.Operator {
	case "<":
		return left < right
	case "<=":
		return left <= right
	case ">":
		return left > right
	case ">=":
		return left >= right
	case "==", "===":
		return left == right
	case "!=", "!==":
		return left != right
	default:
		return false
	}
}

func precedingIntegerAssignment(prefix []ast.Node, name string) (int64, bool) {
	for index := len(prefix) - 1; index >= 0; index-- {
		assignment := directStatementAssignment(prefix[index])
		if assignment == nil {
			return 0, false
		}
		variable, ok := assignment.Left.(*ast.VariableNode)
		if !ok {
			return 0, false
		}
		if variable.Name != name {
			if !independentLiteralAssignment(assignment) {
				return 0, false
			}
			continue
		}
		return loopIntegerValue(assignment.Right, nil)
	}
	return 0, false
}

func directStatementAssignment(node ast.Node) *ast.AssignmentNode {
	if assignment, ok := node.(*ast.AssignmentNode); ok {
		return assignment
	}
	if statement, ok := node.(*ast.ExpressionStmt); ok {
		return assignmentExpr(statement.Expr)
	}
	return nil
}

func independentLiteralAssignment(assignment *ast.AssignmentNode) bool {
	if assignment == nil {
		return false
	}
	switch assignment.Right.(type) {
	case *ast.IntegerLiteral, *ast.IntegerNode, *ast.FloatLiteral, *ast.FloatNode,
		*ast.StringLiteral, *ast.StringNode, *ast.BooleanLiteral, *ast.BooleanNode,
		*ast.NullLiteral, *ast.NullNode:
		return true
	default:
		return false
	}
}

func loopIntegerValue(node ast.Node, scope *functionScope) (int64, bool) {
	switch value := node.(type) {
	case *ast.IntegerLiteral:
		return value.Value, true
	case *ast.IntegerNode:
		return value.Value, true
	case *ast.ClassConstFetchNode:
		values := classConstantValuesFor(value.Class, scope)
		raw, ok := values[value.Const]
		if !ok {
			return 0, false
		}
		parsed, err := strconv.ParseInt(raw, 10, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}
