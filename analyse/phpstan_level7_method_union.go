package analyse

import (
	"fmt"
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
)

const level7MethodUnionCode = "Level7.MethodUnion"

func checkLevel7MethodUnion(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
	return filterIssuesByCode(methodReceiverIssuesForFile(filename, nodes, ctx), level7MethodUnionCode)
}

func appendLevel7PartialUnionMethodIssue(filename string, call *ast.MethodCallNode, scope *functionScope, ctx *AnalysisContext, issues *[]AnalysisIssue) {
	if call == nil || strings.TrimSpace(call.Method) == "" {
		return
	}
	if receiver, ok := call.Object.(*ast.VariableNode); ok && receiver.Name == "this" {
		return
	}
	appendLevel7PartialUnionMethodFromType(filename, call, inferTypeWithFacts(filename, call.Object, scope, ctx), ctx, issues)
}

func appendLevel7PartialUnionMethodFromType(filename string, call *ast.MethodCallNode, receiverType Type, ctx *AnalysisContext, issues *[]AnalysisIssue) {
	if !someButNotAllDNFAlternativesLackMethod(receiverType, call.Method, ctx) {
		return
	}
	receiverLabel := receiverType.withoutBuiltin("null").dnfString()
	*issues = append(*issues, issueSpan(filename, call, level7MethodUnionCode, fmt.Sprintf("Call to an undefined method %s::%s().", receiverLabel, call.Method)))
}

func someButNotAllDNFAlternativesLackMethod(receiverType Type, method string, ctx *AnalysisContext) bool {
	if receiverType.IsEmpty() || ctx == nil || ctx.Resolver == nil {
		return false
	}
	receiverType = receiverType.withoutBuiltin("null")
	if len(receiverType.alternatives) < 2 {
		return false
	}

	hasProviding := false
	hasLacking := false
	for _, alternative := range receiverType.alternatives {
		provides, ok := dnfAlternativeProvidesMethod(receiverType, alternative, method, ctx)
		if !ok {
			return false
		}
		if provides {
			hasProviding = true
		} else {
			hasLacking = true
		}
	}
	return hasProviding && hasLacking
}

func dnfAlternativeProvidesMethod(receiverType Type, keys []string, method string, ctx *AnalysisContext) (provides bool, ok bool) {
	classCount := 0
	for _, key := range keys {
		atom, exists := receiverType.atoms[key]
		if !exists {
			return false, false
		}
		if atom.kind == typeKindBuiltin {
			if atom.key == "null" {
				continue
			}
			return false, false
		}
		if atom.kind != typeKindClass {
			return false, false
		}
		resolvedClass, resolved := resolveMethodReceiverClass(atom.display, ctx)
		if !resolved {
			return false, false
		}
		classCount++
		if receiverClassProvidesMethod(resolvedClass.Name, method, ctx) {
			provides = true
		}
	}
	if classCount == 0 {
		return false, false
	}
	return provides, true
}

func init() {
	RegisterAnalysisRuleWithLevel(level7MethodUnionCode, 7, "level7", checkLevel7MethodUnion)
}
