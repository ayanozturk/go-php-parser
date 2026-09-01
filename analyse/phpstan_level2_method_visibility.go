package analyse

import (
	"fmt"
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
)

const level2MethodVisibilityCode = "Level2.MethodVisibility"

func checkLevel2MethodVisibility(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
	return methodVisibilityIssuesForFile(filename, nodes, ctx)
}

func appendMethodVisibilityOnNode(filename string, node ast.Node, class *ast.ClassNode, currentFn *ast.FunctionNode, ft FileTypeContext, ctx *AnalysisContext, issues *[]AnalysisIssue) {
	switch n := node.(type) {
	case *ast.FunctionCallNode:
		name := functionCallName(n)
		className, methodName, ok := strings.Cut(name, "::")
		if !ok || strings.HasPrefix(className, "$") {
			return
		}
		resolvedClass := resolveClassLikeForCall(className, class, ft, ctx)
		if isSpecialClassName(resolvedClass) {
			return
		}
		method, ok := ctx.Resolver.ResolveMethod(resolvedClass, methodName)
		if !ok {
			return
		}
		appendProtectedMethodVisibilityIssue(filename, n.GetPos(), method, resolvedClass, class, ft, ctx.Resolver, issues)
	case *ast.MethodCallNode:
		className := ""
		if receiver, ok := n.Object.(*ast.VariableNode); ok && receiver.Name == "this" {
			if isStaticMethod(currentFn) {
				return
			}
			className = currentClassName(class, ft)
		} else {
			className = methodCallClassName(n.Object, ft)
		}
		if className == "" {
			return
		}
		method, ok := ctx.Resolver.ResolveMethod(className, n.Method)
		if !ok {
			return
		}
		appendProtectedMethodVisibilityIssue(filename, n.GetPos(), method, className, class, ft, ctx.Resolver, issues)
	}
}

func appendProtectedMethodVisibilityIssue(filename string, pos ast.Position, method ResolvedMethod, className string, currentClass *ast.ClassNode, ft FileTypeContext, resolver SymbolResolver, issues *[]AnalysisIssue) {
	if method.Visibility != "protected" {
		return
	}
	declaringClass := method.DeclaringClass
	if declaringClass == "" {
		declaringClass = className
	}
	caller := callerClassName(currentClass, ft)
	if caller == "" || (!isSubclassOf(resolver, caller, declaringClass) && !classUsesTrait(resolver, caller, declaringClass)) {
		*issues = append(*issues, issue(filename, pos, level2MethodVisibilityCode, fmt.Sprintf("Call to protected method %s::%s().", declaringClass, method.Name)))
	}
}

func init() {
	RegisterAnalysisRuleWithLevel(level2MethodVisibilityCode, 2, "level2", checkLevel2MethodVisibility)
}
