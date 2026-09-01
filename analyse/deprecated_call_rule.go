package analyse

import (
	"fmt"

	"github.com/ayanozturk/go-php-parser/ast"
)

func init() {
	RegisterAnalysisRuleWithLevel("A.DEPRECATED.CALL", 10, "deprecated", func(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
		return checkDeprecatedCalls(nodes, filename, ctx)
	})
}

func checkDeprecatedCalls(nodes []ast.Node, filename string, ctx *AnalysisContext) []AnalysisIssue {
	ctx = ensureArgCallDiagnostics(filename, nodes, ctx)
	return ctx.deprecatedCallIssues
}

func appendDeprecatedCallFromExpr(filename string, expr ast.Node, scope *functionScope, ctx *AnalysisContext) {
	if ctx == nil || ctx.deprecatedCallSink == nil || expr == nil {
		return
	}
	if _, dup := ctx.deprecatedCallSeen[expr]; dup {
		return
	}
	switch call := expr.(type) {
	case *ast.MethodCallNode:
		method, ok := resolveMethodForCall(call, scope, ctx, filename)
		if !ok || !method.Deprecated {
			return
		}
		target := method.Name
		if method.DeclaringClass != "" {
			target = method.DeclaringClass + "::" + method.Name
		}
		ctx.deprecatedCallSeen[expr] = struct{}{}
		*ctx.deprecatedCallSink = append(*ctx.deprecatedCallSink, deprecatedCallIssue(filename, call, "method", target, method.DeprecationMessage))
	case *ast.FunctionCallNode:
		name := functionCallName(call)
		if name == "" || ctx.Resolver == nil {
			return
		}
		fn, ok := resolveFunctionView(ctx.Resolver, name)
		if !ok || !fn.Deprecated {
			return
		}
		ctx.deprecatedCallSeen[expr] = struct{}{}
		*ctx.deprecatedCallSink = append(*ctx.deprecatedCallSink, deprecatedCallIssue(filename, call, "function", fn.Name, fn.DeprecationMessage))
	}
}

func deprecatedCallIssue(filename string, call ast.Node, kind, target, deprecationMessage string) AnalysisIssue {
	message := fmt.Sprintf("Call to deprecated %s: `%s`.", kind, target)
	iss := issueSpanWarning(filename, call, "A.DEPRECATED.CALL", message)
	iss.SubjectKind = kind
	iss.SubjectName = target
	if deprecationMessage != "" {
		iss.Message += " " + deprecationMessage
	}
	return iss
}
