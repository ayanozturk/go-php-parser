package analyse

import (
	"fmt"
	"go-phpcs/ast"
	"strconv"
	"strings"
)

var allowedEnumMagicMethods = map[string]struct{}{
	"__call":       {},
	"__callstatic": {},
	"__invoke":     {},
}

func checkEnumLegality(filename, enumName string, enum *ast.EnumNode, issues *[]AnalysisIssue) {
	backed := enum.BackedBy != ""
	if backed && !isValidEnumBackingType(enum.BackedBy) {
		*issues = append(*issues, issue(filename, enum.GetPos(), level0ClassModelCode, fmt.Sprintf("Backed enum %s can have only \"int\" or \"string\" type.", enumName)))
	}
	for _, implemented := range enum.Implements {
		if strings.EqualFold(strings.TrimPrefix(implemented, `\`), "Serializable") {
			*issues = append(*issues, issue(filename, enum.GetPos(), level0ClassModelCode, fmt.Sprintf("Enum %s cannot implement Serializable.", enumName)))
		}
	}

	duplicateValues := map[string][]string{}
	for _, enumCase := range enum.Cases {
		if !backed && enumCase.Value != nil {
			*issues = append(*issues, issue(filename, enumCase.GetPos(), level0ClassModelCode, fmt.Sprintf(
				"Enum %s is not backed, but case %s has value %s.",
				enumName,
				enumCase.Name,
				describeEnumCaseValue(enumCase.Value),
			)))
			continue
		}
		if !backed {
			continue
		}
		if enumCase.Value == nil {
			*issues = append(*issues, issue(filename, enumCase.GetPos(), level0ClassModelCode, fmt.Sprintf(
				"Enum case %s::%s does not have a value but the enum is backed with the \"%s\" type.",
				enumName,
				enumCase.Name,
				enum.BackedBy,
			)))
			continue
		}
		if !enumCaseValueMatchesBacking(enum.BackedBy, enumCase.Value) {
			*issues = append(*issues, issue(filename, enumCase.GetPos(), level0ClassModelCode, fmt.Sprintf(
				"Enum case %s::%s value %s does not match the \"%s\" type.",
				enumName,
				enumCase.Name,
				describeEnumCaseValue(enumCase.Value),
				strings.ToLower(enum.BackedBy),
			)))
			continue
		}
		if scalar, ok := enumCaseScalarValue(enumCase.Value); ok {
			key := fmt.Sprintf("%v", scalar)
			duplicateValues[key] = append(duplicateValues[key], enumCase.Name)
		}
	}

	for value, caseNames := range duplicateValues {
		if len(caseNames) <= 1 {
			continue
		}
		*issues = append(*issues, issue(filename, enum.GetPos(), level0ClassModelCode, fmt.Sprintf(
			"Enum %s has duplicate value %s for cases %s.",
			enumName,
			value,
			strings.Join(caseNames, ", "),
		)))
	}

	for _, methodNode := range enum.Methods {
		method, ok := methodNode.(*ast.FunctionNode)
		if !ok {
			continue
		}
		lowerName := strings.ToLower(method.Name)
		if isMagicMethodName(method.Name) {
			switch lowerName {
			case "__construct":
				*issues = append(*issues, issue(filename, method.GetPos(), level0ClassModelCode, fmt.Sprintf("Enum %s contains constructor.", enumName)))
			case "__destruct":
				*issues = append(*issues, issue(filename, method.GetPos(), level0ClassModelCode, fmt.Sprintf("Enum %s contains destructor.", enumName)))
			default:
				if _, allowed := allowedEnumMagicMethods[lowerName]; !allowed {
					*issues = append(*issues, issue(filename, method.GetPos(), level0ClassModelCode, fmt.Sprintf("Enum %s contains magic method %s().", enumName, method.Name)))
				}
			}
		}
		if lowerName == "cases" || (backed && (lowerName == "from" || lowerName == "tryfrom")) {
			*issues = append(*issues, issue(filename, method.GetPos(), level0ClassModelCode, fmt.Sprintf("Enum %s cannot redeclare native method %s().", enumName, method.Name)))
		}
	}
}

func isValidEnumBackingType(backing string) bool {
	switch strings.ToLower(strings.TrimSpace(backing)) {
	case "int", "string":
		return true
	default:
		return false
	}
}

func isMagicMethodName(name string) bool {
	return strings.HasPrefix(name, "__")
}

func enumCaseScalarValue(node ast.Node) (interface{}, bool) {
	switch n := node.(type) {
	case *ast.IntegerLiteral:
		return n.Value, true
	case *ast.IntegerNode:
		return n.Value, true
	case *ast.StringLiteral:
		return n.Value, true
	case *ast.StringNode:
		return n.Value, true
	}
	return nil, false
}

func enumCaseValueMatchesBacking(backing string, node ast.Node) bool {
	value, ok := enumCaseScalarValue(node)
	if !ok {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(backing)) {
	case "int":
		_, ok := value.(int64)
		return ok
	case "string":
		_, ok := value.(string)
		return ok
	default:
		return false
	}
}

func describeEnumCaseValue(node ast.Node) string {
	if value, ok := enumCaseScalarValue(node); ok {
		switch v := value.(type) {
		case int64:
			return strconv.FormatInt(v, 10)
		case string:
			return strconv.Quote(v)
		}
	}
	return "value"
}
