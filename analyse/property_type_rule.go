package analyse

import (
	"fmt"
	"github.com/ayanozturk/go-php-parser/ast"
	"strings"
)

type PropertyTypeRule struct{}

const (
	assignOpInvalidCode = "A.ASSIGN.OP.INVALID"
	binaryOpInvalidCode = "A.BINARY.OP.INVALID"
)

func (r *PropertyTypeRule) CheckIssues(nodes []ast.Node, filename string, ctx *AnalysisContext) []AnalysisIssue {
	return filterIssuesByCode(assignmentTypeIssuesForFile(filename, nodes, ctx), "A.PROP.TYPE")
}

func assignmentTypeIssuesForFile(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
	if ctx == nil {
		ctx = &AnalysisContext{}
	}
	if ctx.hasAssignmentTypeIssues {
		return ctx.assignmentTypeIssues
	}
	ctx.assignmentTypeIssues = collectAssignmentTypeIssues(nodes, filename, ctx)
	ctx.hasAssignmentTypeIssues = true
	return ctx.assignmentTypeIssues
}

func collectAssignmentTypeIssues(nodes []ast.Node, filename string, ctx *AnalysisContext) []AnalysisIssue {
	var issues []AnalysisIssue
	fileCtx := analysisFileTypeContext(ctx, nodes)
	var walk func(node ast.Node, class *ast.ClassNode)

	walk = func(node ast.Node, class *ast.ClassNode) {
		switch n := node.(type) {
		case *ast.ClassNode:
			for _, methodNode := range n.Methods {
				walk(methodNode, n)
			}
		case *ast.FunctionNode:
			scope := analysisFunctionScope(ctx, class, n, fileCtx)
			walkStatementsForPropertyTypes(n.Body, scope, ctx, filename, &issues)
		case *ast.NamespaceNode:
			for _, child := range n.Body {
				walk(child, class)
			}
		}
	}

	for _, node := range nodes {
		walk(node, nil)
	}

	return issues
}

func walkStatementsForPropertyTypes(nodes []ast.Node, scope *functionScope, ctx *AnalysisContext, filename string, issues *[]AnalysisIssue) {
	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.ExpressionStmt:
			walkExprForPropertyTypes(n.Expr, scope, ctx, filename, issues)
			applyExpressionScope(scope, n.Expr, ctx)
		case *ast.AssignmentNode:
			walkAssignmentForPropertyTypes(n, scope, ctx, filename, issues)
			applyAssignmentScope(scope, n, ctx)
		case *ast.ReturnNode:
			walkExprForPropertyTypes(n.Expr, scope, ctx, filename, issues)
		case *ast.BreakNode:
			walkExprForPropertyTypes(n.Level, scope, ctx, filename, issues)
		case *ast.ContinueNode:
			walkExprForPropertyTypes(n.Level, scope, ctx, filename, issues)
		case *ast.IfNode:
			walkExprForPropertyTypes(n.Condition, scope, ctx, filename, issues)
			walkStatementsForPropertyTypes(n.Body, scopeForConditionTrue(scope, n.Condition), ctx, filename, issues)
			for _, elseif := range n.ElseIfs {
				walkExprForPropertyTypes(elseif.Condition, scope, ctx, filename, issues)
				walkStatementsForPropertyTypes(elseif.Body, scopeForConditionTrue(scope, elseif.Condition), ctx, filename, issues)
			}
			if n.Else != nil {
				walkStatementsForPropertyTypes(n.Else.Body, scopeForConditionFalse(scope, n.Condition), ctx, filename, issues)
			}
			applyTerminatingIfFalseScope(scope, n)
		case *ast.BlockNode:
			walkStatementsForPropertyTypes(n.Statements, scope.clone(), ctx, filename, issues)
		case *ast.WhileNode:
			walkExprForPropertyTypes(n.Condition, scope, ctx, filename, issues)
			walkStatementsForPropertyTypes(n.Body, scope.clone(), ctx, filename, issues)
		case *ast.DoWhileNode:
			loopScope := scope.clone()
			walkStatementsForPropertyTypes(n.Body, loopScope, ctx, filename, issues)
			walkExprForPropertyTypes(n.Condition, loopScope, ctx, filename, issues)
		case *ast.ForNode:
			for _, expression := range n.Init {
				walkExprForPropertyTypes(expression, scope, ctx, filename, issues)
				applyExpressionScope(scope, expression, ctx)
			}
			loopScope := scope.clone()
			for _, condition := range n.Conditions {
				walkExprForPropertyTypes(condition, loopScope, ctx, filename, issues)
			}
			walkStatementsForPropertyTypes(n.Body, loopScope, ctx, filename, issues)
			for _, update := range n.Updates {
				walkExprForPropertyTypes(update, loopScope, ctx, filename, issues)
			}
		case *ast.ForeachNode:
			walkExprForPropertyTypes(n.Expr, scope, ctx, filename, issues)
			walkStatementsForPropertyTypes(n.Body, scope.clone(), ctx, filename, issues)
		}
	}
}

func walkExprForPropertyTypes(node ast.Node, scope *functionScope, ctx *AnalysisContext, filename string, issues *[]AnalysisIssue) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *ast.AssignmentNode:
		walkAssignmentForPropertyTypes(n, scope, ctx, filename, issues)
	case *ast.MethodCallNode:
		walkExprForPropertyTypes(n.Object, scope, ctx, filename, issues)
		for _, arg := range n.Args {
			walkExprForPropertyTypes(argumentValue(arg), scope, ctx, filename, issues)
		}
	case *ast.FunctionCallNode:
		for _, arg := range n.Args {
			walkExprForPropertyTypes(argumentValue(arg), scope, ctx, filename, issues)
		}
	case *ast.PropertyFetchNode:
		walkExprForPropertyTypes(n.Object, scope, ctx, filename, issues)
	case *ast.BinaryExpr:
		walkExprForPropertyTypes(n.Left, scope, ctx, filename, issues)
		walkExprForPropertyTypes(n.Right, scope, ctx, filename, issues)
		leftType := inferTypeWithFacts(filename, n.Left, scope, ctx)
		rightType := inferTypeWithFacts(filename, n.Right, scope, ctx)
		_, valid, known := binaryOperationResult(n.Operator, leftType, rightType)
		if known && !valid {
			*issues = append(*issues, issueSpan(filename, n, binaryOpInvalidCode, fmt.Sprintf(
				"Invalid binary operation %s between %s and %s", n.Operator, typeLabel(leftType), typeLabel(rightType),
			)))
		}
	case *ast.UnaryExpr:
		walkExprForPropertyTypes(n.Operand, scope, ctx, filename, issues)
	case *ast.ConcatNode:
		for _, part := range n.Parts {
			walkExprForPropertyTypes(part, scope, ctx, filename, issues)
		}
	case *ast.TernaryExpr:
		walkExprForPropertyTypes(n.Condition, scope, ctx, filename, issues)
		walkExprForPropertyTypes(n.IfTrue, scope, ctx, filename, issues)
		walkExprForPropertyTypes(n.IfFalse, scope, ctx, filename, issues)
	case *ast.NewNode:
		for _, arg := range n.Args {
			walkExprForPropertyTypes(argumentValue(arg), scope, ctx, filename, issues)
		}
	case *ast.NamedArgumentNode:
		walkExprForPropertyTypes(n.Value, scope, ctx, filename, issues)
	case *ast.UnpackedArgumentNode:
		walkExprForPropertyTypes(n.Expr, scope, ctx, filename, issues)
	}
}

func walkAssignmentForPropertyTypes(assign *ast.AssignmentNode, scope *functionScope, ctx *AnalysisContext, filename string, issues *[]AnalysisIssue) {
	if assign == nil {
		return
	}

	walkExprForPropertyTypes(assign.Right, scope, ctx, filename, issues)

	actual := inferTypeWithFacts(filename, assign.Right, scope, ctx)
	if assign.Operator != "=" {
		leftType := inferAssignmentTargetType(assign.Left, scope, ctx, filename)
		result, valid, known := compoundAssignmentResult(assign.Operator, leftType, actual)
		if known && !valid {
			*issues = append(*issues, issueSpan(filename, assign, assignOpInvalidCode, fmt.Sprintf(
				"Invalid compound assignment %s between %s and %s", assign.Operator, typeLabel(leftType), typeLabel(actual),
			)))
			return
		}
		if !known {
			return
		}
		actual = result
	}
	if !analysisLevelAtLeast(ctx, 3) {
		return
	}

	expected, propertyName, ok := resolvePropertyAssignmentTarget(assign.Left, scope, ctx, filename)
	if !ok || expected.IsEmpty() {
		return
	}

	if expected.AcceptsWithContext(actual, scope, ctx) {
		return
	}

	actualLabel := actual.String()
	if actualLabel == "" {
		actualLabel = "mixed"
	}
	pos := assign.Right.GetPos()
	*issues = append(*issues, AnalysisIssue{
		Filename: filename,
		Line:     pos.Line,
		Column:   pos.Column,
		Code:     "A.PROP.TYPE",
		Message:  fmt.Sprintf("Property %s expects %s, got %s", propertyName, expected.String(), actualLabel),
	})
}

func resolvePropertyAssignmentTarget(left ast.Node, scope *functionScope, ctx *AnalysisContext, filename string) (Type, string, bool) {
	switch target := left.(type) {
	case *ast.PropertyFetchNode:
		return resolvePropertyTypeForAssignment(target, scope, ctx, filename)
	case *ast.ClassConstFetchNode:
		return resolveStaticPropertyTypeForAssignment(target, scope, ctx)
	default:
		return EmptyType(), "", false
	}
}

func inferAssignmentTargetType(left ast.Node, scope *functionScope, ctx *AnalysisContext, filename string) Type {
	if expected, _, ok := resolvePropertyAssignmentTarget(left, scope, ctx, filename); ok {
		return expected
	}
	return inferTypeWithFacts(filename, left, scope, ctx)
}

func compoundAssignmentResult(operator string, left, right Type) (Type, bool, bool) {
	leftName, leftOK := singleCompoundBuiltin(left)
	rightName, rightOK := singleCompoundBuiltin(right)
	if !leftOK || !rightOK {
		return EmptyType(), false, false
	}

	switch operator {
	case "+=":
		if leftName == "array" || rightName == "array" {
			if leftName == "array" && rightName == "array" {
				return ParseType("array"), true, true
			}
			return EmptyType(), false, true
		}
		return numericCompoundResult(leftName, rightName)
	case "-=", "*=", "/=", "**=":
		return numericCompoundResult(leftName, rightName)
	case "%=", "<<=", ">>=":
		if isNumericCompoundType(leftName) && isNumericCompoundType(rightName) {
			return ParseType("int"), true, true
		}
		return EmptyType(), false, true
	case ".=":
		if leftName == "array" || rightName == "array" {
			return EmptyType(), false, true
		}
		return ParseType("string"), true, true
	default:
		return EmptyType(), false, false
	}
}

// binaryOperationResult returns the result type for operators whose result is
// independent of runtime values. Invalid known operand pairs are reported by
// the same expression walk used for property and compound-assignment checks.
func binaryOperationResult(operator string, left, right Type) (Type, bool, bool) {
	switch operator {
	case "+":
		return compoundAssignmentResult("+=", left, right)
	case "-":
		return compoundAssignmentResult("-=", left, right)
	case "*":
		return compoundAssignmentResult("*=", left, right)
	case "%":
		return compoundAssignmentResult("%=", left, right)
	case "<<":
		return compoundAssignmentResult("<<=", left, right)
	case ">>":
		return compoundAssignmentResult(">>=", left, right)
	case "==", "!=", "===", "!==", "<", ">", "<=", ">=":
		return ParseType("bool"), true, true
	case "<=>":
		return ParseType("int"), true, true
	case "&&", "||", "and", "or", "xor":
		return ParseType("bool"), true, true
	default:
		return EmptyType(), false, false
	}
}

func numericCompoundResult(left, right string) (Type, bool, bool) {
	if !isNumericCompoundType(left) || !isNumericCompoundType(right) {
		if !isKnownCompoundOperand(left) || !isKnownCompoundOperand(right) {
			return EmptyType(), false, false
		}
		return EmptyType(), false, true
	}
	if left == "float" || right == "float" {
		return ParseType("float"), true, true
	}
	return ParseType("int"), true, true
}

func isKnownCompoundOperand(name string) bool {
	switch name {
	case "array", "float", "int", "string":
		return true
	default:
		return false
	}
}

func isNumericCompoundType(name string) bool {
	return name == "int" || name == "float"
}

func singleCompoundBuiltin(typ Type) (string, bool) {
	if len(typ.atoms) != 1 {
		return "", false
	}
	for _, atom := range typ.atoms {
		if atom.kind != typeKindBuiltin || atom.key == "mixed" {
			return "", false
		}
		return atom.key, true
	}
	return "", false
}

func typeLabel(typ Type) string {
	if label := typ.String(); label != "" {
		return label
	}
	return "mixed"
}

func resolveStaticPropertyTypeForAssignment(fetch *ast.ClassConstFetchNode, scope *functionScope, ctx *AnalysisContext) (Type, string, bool) {
	if fetch == nil || fetch.ConstExpr != nil || !strings.HasPrefix(fetch.Const, "$") {
		return EmptyType(), "", false
	}

	propertyName := strings.TrimPrefix(fetch.Const, "$")
	className := strings.TrimPrefix(strings.TrimSpace(fetch.Class), `\`)
	if className == "" || propertyName == "" {
		return EmptyType(), "", false
	}

	if scope != nil {
		switch strings.ToLower(className) {
		case "self", "static":
			className = scope.className
		case "parent":
			if class, ok := scope.typeCtx.resolveClass(scope.className); ok && len(class.Extends) > 0 {
				className = class.Extends[0]
			} else {
				return EmptyType(), "", false
			}
		default:
			className = scope.typeCtx.resolveClassLike(className)
		}
	}

	if ctx != nil && ctx.Resolver != nil {
		if property, ok := ctx.Resolver.ResolveProperty(className, propertyName); ok {
			if !property.IsStatic {
				return EmptyType(), "", false
			}
			return ParseType(property.Type), className + "::$" + property.Name, true
		}
	}
	if scope != nil && strings.EqualFold(className, scope.className) {
		if propertyType, ok := resolveSameClassPropertyType(scope, propertyName); ok {
			return propertyType, className + "::$" + propertyName, true
		}
	}

	return EmptyType(), "", false
}

func resolvePropertyTypeForAssignment(fetch *ast.PropertyFetchNode, scope *functionScope, ctx *AnalysisContext, filename string) (Type, string, bool) {
	if fetch == nil {
		return EmptyType(), "", false
	}

	if object, ok := fetch.Object.(*ast.VariableNode); ok && object.Name == "this" {
		if propertyType, ok := resolveSameClassPropertyType(scope, fetch.Property); ok {
			return propertyType, "$this->" + fetch.Property, true
		}
	}

	objectType := inferTypeWithFacts(filename, fetch.Object, scope, ctx)
	className, ok := objectType.SingleClassName()
	if !ok {
		return EmptyType(), "", false
	}
	if scope != nil && scope.className != "" && strings.EqualFold(className, scope.className) {
		if propertyType, ok := resolveSameClassPropertyType(scope, fetch.Property); ok {
			return propertyType, className + "::$" + fetch.Property, true
		}
	}
	if ctx != nil && ctx.Resolver != nil {
		if property, ok := ctx.Resolver.ResolveProperty(className, fetch.Property); ok {
			return ParseType(property.Type), className + "::$" + property.Name, true
		}
	}

	return EmptyType(), "", false
}

func init() {
	RegisterAnalysisRuleWithLevel(assignOpInvalidCode, 2, "level2", func(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
		return filterIssuesByCode(assignmentTypeIssuesForFile(filename, nodes, ctx), assignOpInvalidCode)
	})
	RegisterAnalysisRuleWithLevel(binaryOpInvalidCode, 2, "level2", func(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
		return filterIssuesByCode(assignmentTypeIssuesForFile(filename, nodes, ctx), binaryOpInvalidCode)
	})
	RegisterAnalysisRuleWithLevel("A.PROP.TYPE", 3, "level3", func(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
		return filterIssuesByCode(assignmentTypeIssuesForFile(filename, nodes, ctx), "A.PROP.TYPE")
	})
}
