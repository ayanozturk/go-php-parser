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
	ctx = ensureArgCallDiagnostics(filename, nodes, ctx)
	return ctx.assignmentTypeIssues
}

func recordAssignmentTypeIssues(assign *ast.AssignmentNode, scope *functionScope, ctx *AnalysisContext, filename string) {
	if assign == nil || ctx == nil || ctx.assignmentTypeSink == nil {
		return
	}
	actual := inferTypeWithFacts(filename, assign.Right, scope, ctx)
	if assign.Operator != "=" {
		leftType := inferAssignmentTargetType(assign.Left, scope, ctx, filename)
		result, valid, known := compoundAssignmentResult(assign.Operator, leftType, actual)
		if known && !valid {
			*ctx.assignmentTypeSink = append(*ctx.assignmentTypeSink, issueSpan(filename, assign, assignOpInvalidCode, fmt.Sprintf(
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
	*ctx.assignmentTypeSink = append(*ctx.assignmentTypeSink, AnalysisIssue{
		Filename: filename,
		Line:     pos.Line,
		Column:   pos.Column,
		Code:     "A.PROP.TYPE",
		Message:  fmt.Sprintf("Property %s expects %s, got %s", propertyName, expected.String(), actualLabel),
	})
}

func recordBinaryOpIssue(expr *ast.BinaryExpr, scope *functionScope, ctx *AnalysisContext, filename string) {
	if expr == nil || ctx == nil || ctx.assignmentTypeSink == nil {
		return
	}
	leftType := inferTypeWithFacts(filename, expr.Left, scope, ctx)
	rightType := inferTypeWithFacts(filename, expr.Right, scope, ctx)
	_, valid, known := binaryOperationResult(expr.Operator, leftType, rightType)
	if known && !valid {
		*ctx.assignmentTypeSink = append(*ctx.assignmentTypeSink, issueSpan(filename, expr, binaryOpInvalidCode, fmt.Sprintf(
			"Invalid binary operation %s between %s and %s", expr.Operator, typeLabel(leftType), typeLabel(rightType),
		)))
	}
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
