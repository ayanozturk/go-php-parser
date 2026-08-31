package analyse

import (
	"fmt"
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
)

const level2MethodExistenceCode = "PHPStan.Level2.MethodExistence"

func checkLevel2MethodExistence(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
	ctx = ensureLevel0Context(filename, nodes, ctx)
	var issues []AnalysisIssue
	seen := make(map[*ast.MethodCallNode]struct{})
	check := func(filename string, expr ast.Node, scope *functionScope, ctx *AnalysisContext) {
		call, ok := expr.(*ast.MethodCallNode)
		if !ok {
			return
		}
		seen[call] = struct{}{}
		appendLevel2UnknownMethodIssue(filename, call, scope, ctx, &issues)
	}
	walkLevel2MethodExpressions(filename, nodes, ctx, check)

	walkAll(nodes, func(node ast.Node, _ *ast.ClassNode, _ *ast.FunctionNode, _ FileTypeContext) {
		call, ok := node.(*ast.MethodCallNode)
		if !ok {
			return
		}
		if _, alreadyChecked := seen[call]; alreadyChecked {
			return
		}
		appendLevel2UnknownMethodIssue(filename, call, nil, ctx, &issues)
	})

	return issues
}

func appendLevel2UnknownMethodIssue(filename string, call *ast.MethodCallNode, scope *functionScope, ctx *AnalysisContext, issues *[]AnalysisIssue) {
	if call == nil || strings.TrimSpace(call.Method) == "" {
		return
	}
	if receiver, ok := call.Object.(*ast.VariableNode); ok && receiver.Name == "this" {
		// PHPStan reports unknown methods on $this at level 0. The existing
		// symbols pass owns that diagnostic and must not be duplicated here.
		return
	}

	receiverType := inferTypeWithFacts(filename, call.Object, scope, ctx)
	if className, single := receiverType.SingleClassName(); single {
		resolvedClass, ok := resolveMethodReceiverClass(className, ctx)
		if !ok || receiverClassProvidesMethod(resolvedClass.Name, call.Method, ctx) {
			return
		}
		*issues = append(*issues, issueSpan(filename, call, level2MethodExistenceCode, fmt.Sprintf("Call to an undefined method %s::%s().", resolvedClass.Name, call.Method)))
		return
	}

	if !allReceiverClassesLackMethod(receiverType, call.Method, ctx) {
		return
	}
	receiverLabel := receiverType.withoutBuiltin("null").dnfString()
	*issues = append(*issues, issueSpan(filename, call, level2MethodExistenceCode, fmt.Sprintf("Call to an undefined method %s::%s().", receiverLabel, call.Method)))
}

func resolveMethodReceiverClass(className string, ctx *AnalysisContext) (ResolvedClass, bool) {
	if ctx == nil || ctx.Resolver == nil {
		return ResolvedClass{}, false
	}
	resolvedClass, ok := ctx.Resolver.ResolveClass(className)
	if !ok {
		// Unknown receiver classes are owned by the level-0 symbol checks.
		return ResolvedClass{}, false
	}
	return resolvedClass, true
}

func receiverClassProvidesMethod(className, method string, ctx *AnalysisContext) bool {
	if _, ok := ctx.Resolver.ResolveMethod(className, method); ok {
		return true
	}
	_, magic := ctx.Resolver.ResolveMethod(className, "__call")
	return magic
}

func allReceiverClassesLackMethod(receiverType Type, method string, ctx *AnalysisContext) bool {
	if receiverType.IsEmpty() || ctx == nil || ctx.Resolver == nil {
		return false
	}
	classCount := 0
	for _, atom := range receiverType.atoms {
		if atom.kind != typeKindClass {
			if atom.kind == typeKindBuiltin && atom.key == "null" {
				continue
			}
			// Mixed and other partly non-object receivers have separate PHPStan
			// semantics. Keep this rule conservative until those codes are gated.
			return false
		}
		resolvedClass, ok := resolveMethodReceiverClass(atom.display, ctx)
		if !ok {
			return false
		}
		if receiverClassProvidesMethod(resolvedClass.Name, method, ctx) {
			return false
		}
		classCount++
	}
	return classCount > 0
}

func walkLevel2MethodExpressions(filename string, nodes []ast.Node, ctx *AnalysisContext, observe semanticExpressionObserver) {
	fileCtx := analysisFileTypeContext(ctx, nodes)

	var walkDeclarations func([]ast.Node, *ast.ClassNode)
	walkDeclarations = func(declarations []ast.Node, class *ast.ClassNode) {
		for _, node := range declarations {
			switch n := node.(type) {
			case *ast.NamespaceNode:
				walkDeclarations(n.Body, class)
			case *ast.ClassNode:
				for _, method := range n.Methods {
					if function, ok := method.(*ast.FunctionNode); ok {
						walkDeclarations([]ast.Node{function}, n)
					}
				}
			case *ast.FunctionNode:
				scope := analysisFunctionScope(ctx, class, n, fileCtx)
				walkStatementsForArgTypesUsing(n.Body, scope, ctx, filename, nil, observe)
			}
		}
	}
	walkDeclarations(nodes, nil)

	// Preserve flow across executable file-scope statements. Declaration
	// nodes are ignored by the shared statement walker.
	fileScope := newFunctionScopeWithContext(ctx, nil, &ast.FunctionNode{}, fileCtx)
	walkStatementsForArgTypesUsing(nodes, fileScope, ctx, filename, nil, observe)
}

func init() {
	RegisterAnalysisRuleWithLevel(level2MethodExistenceCode, 2, "phpstan.level2", checkLevel2MethodExistence)
}
