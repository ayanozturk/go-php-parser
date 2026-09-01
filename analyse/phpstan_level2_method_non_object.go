package analyse

import (
	"fmt"
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
)

const level2MethodNonObjectCode = "Level2.MethodNonObject"

func checkLevel2MethodNonObject(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
	return filterIssuesByCode(methodReceiverIssuesForFile(filename, nodes, ctx), level2MethodNonObjectCode)
}

func appendLevel2NonObjectMethodIssue(filename string, call *ast.MethodCallNode, scope *functionScope, ctx *AnalysisContext, issues *[]AnalysisIssue) {
	if call == nil || strings.TrimSpace(call.Method) == "" {
		return
	}
	if receiver, ok := call.Object.(*ast.VariableNode); ok && receiver.Name == "this" {
		return
	}
	appendLevel2NonObjectMethodFromType(filename, call, inferTypeWithFacts(filename, call.Object, scope, ctx), ctx, issues)
}

func appendLevel2NonObjectMethodFromType(filename string, call *ast.MethodCallNode, receiverType Type, ctx *AnalysisContext, issues *[]AnalysisIssue) {
	label, ok := methodNonObjectLabel(receiverType, call.Method, ctx)
	if !ok {
		return
	}
	*issues = append(*issues, issueSpan(filename, call, level2MethodNonObjectCode, fmt.Sprintf("Cannot call method %s() on %s.", call.Method, label)))
}

func methodNonObjectLabel(receiverType Type, method string, ctx *AnalysisContext) (string, bool) {
	if receiverType.IsEmpty() || receiverType.hasBuiltin("mixed") {
		return "", false
	}

	hasClass := false
	hasNonObject := false
	classProvides := false
	classLacks := false
	onlyNull := true

	for _, atom := range receiverType.atoms {
		switch {
		case atom.kind == typeKindClass:
			onlyNull = false
			hasClass = true
			resolvedClass, ok := resolveMethodReceiverClass(atom.display, ctx)
			if !ok {
				return "", false
			}
			if receiverClassProvidesMethod(resolvedClass.Name, method, ctx) {
				classProvides = true
			} else {
				classLacks = true
			}
		case atom.kind == typeKindBuiltin && atom.key == "object":
			onlyNull = false
		case atom.kind == typeKindBuiltin && atom.key == "null":
			continue
		case atom.kind == typeKindBuiltin:
			onlyNull = false
			hasNonObject = true
		default:
			return "", false
		}
	}

	if onlyNull && receiverType.hasBuiltin("null") {
		return "null", true
	}
	if !hasNonObject {
		return "", false
	}
	if hasClass && classProvides && !classLacks {
		return "", false
	}

	labelType := receiverType.withoutBuiltin("null")
	if labelType.IsEmpty() {
		return "null", true
	}
	return labelType.dnfString(), true
}

func init() {
	RegisterAnalysisRuleWithLevel(level2MethodNonObjectCode, 2, "level2", checkLevel2MethodNonObject)
}
