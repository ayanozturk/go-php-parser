package analyse

import (
	"fmt"
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
)

const level8MethodNonObjectCode = "PHPStan.Level8.MethodNonObject"

func checkLevel8MethodNonObject(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
	return filterIssuesByCode(methodReceiverIssuesForFile(filename, nodes, ctx), level8MethodNonObjectCode)
}

func appendLevel8NullableMethodIssue(filename string, call *ast.MethodCallNode, scope *functionScope, ctx *AnalysisContext, issues *[]AnalysisIssue) {
	if call == nil || call.Nullsafe || strings.TrimSpace(call.Method) == "" {
		return
	}
	if receiver, ok := call.Object.(*ast.VariableNode); ok && receiver.Name == "this" {
		return
	}
	appendLevel8NullableMethodFromType(filename, call, inferTypeWithFacts(filename, call.Object, scope, ctx), ctx, issues)
}

func appendLevel8NullableMethodFromType(filename string, call *ast.MethodCallNode, receiverType Type, ctx *AnalysisContext, issues *[]AnalysisIssue) {
	if call == nil || call.Nullsafe {
		return
	}
	label, ok := nullableObjectMethodLabel(receiverType, call.Method, ctx)
	if !ok {
		return
	}
	*issues = append(*issues, issueSpan(filename, call, level8MethodNonObjectCode, fmt.Sprintf("Cannot call method %s() on %s.", call.Method, label)))
}

func nullableObjectMethodLabel(receiverType Type, method string, ctx *AnalysisContext) (string, bool) {
	if receiverType.IsEmpty() || ctx == nil || ctx.Resolver == nil {
		return "", false
	}
	if !receiverType.hasBuiltin("null") || receiverType.hasBuiltin("mixed") {
		return "", false
	}
	if _, already := methodNonObjectLabel(receiverType, method, ctx); already {
		return "", false
	}

	remaining := receiverType.withoutBuiltin("null")
	if remaining.IsEmpty() {
		return "", false
	}
	if !remainingObjectLikeProvidesMethod(remaining, method, ctx) {
		return "", false
	}
	return nullableReceiverLabel(remaining), true
}

func nullableReceiverLabel(remaining Type) string {
	label := remaining.dnfString()
	if len(remaining.alternatives) == 1 && len(remaining.alternatives[0]) > 1 {
		label = "(" + label + ")"
	}
	return label + "|null"
}

func remainingObjectLikeProvidesMethod(remaining Type, method string, ctx *AnalysisContext) bool {
	alternatives := remaining.alternatives
	if len(alternatives) == 0 {
		return false
	}
	for _, alternative := range alternatives {
		provides, ok := dnfAlternativeProvidesMethodOrObject(remaining, alternative, method, ctx)
		if !ok || !provides {
			return false
		}
	}
	return true
}

func dnfAlternativeProvidesMethodOrObject(receiverType Type, keys []string, method string, ctx *AnalysisContext) (bool, bool) {
	classKeys := make([]string, 0, len(keys))
	hasObject := false
	for _, key := range keys {
		atom, exists := receiverType.atoms[key]
		if !exists {
			return false, false
		}
		if atom.kind == typeKindBuiltin {
			if atom.key == "object" {
				hasObject = true
				continue
			}
			if atom.key == "null" {
				continue
			}
			return false, false
		}
		if atom.kind != typeKindClass {
			return false, false
		}
		classKeys = append(classKeys, key)
	}
	if len(classKeys) == 0 {
		return hasObject, hasObject
	}
	return dnfAlternativeProvidesMethod(receiverType, classKeys, method, ctx)
}

func init() {
	RegisterAnalysisRuleWithLevel(level8MethodNonObjectCode, 8, "phpstan.level8", checkLevel8MethodNonObject)
}
