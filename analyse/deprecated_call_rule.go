package analyse

import (
	"fmt"

	"github.com/ayanozturk/go-php-parser/ast"
)

func init() {
	RegisterAnalysisRuleWithLevel("A.DEPRECATED.CALL", 10, "phpstan.deprecated", func(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
		return checkDeprecatedCalls(nodes, filename, ctx)
	})
}

// checkDeprecatedCalls flags calls to methods or functions annotated with
// @deprecated in their PHPDoc, mirroring PHPStan/mago's deprecation check.
func checkDeprecatedCalls(nodes []ast.Node, filename string, ctx *AnalysisContext) []AnalysisIssue {
	var issues []AnalysisIssue
	fileCtx := analysisFileTypeContext(ctx, nodes)

	// The statement walker can invoke the observer more than once for the
	// same call expression (e.g. return statements); dedupe on node identity.
	seen := make(map[ast.Node]struct{})

	observer := func(filename string, expr ast.Node, scope *functionScope, ctx *AnalysisContext) {
		if _, dup := seen[expr]; dup {
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
			seen[expr] = struct{}{}
			issues = append(issues, deprecatedCallIssue(filename, call, "method", target, method.DeprecationMessage))
		case *ast.FunctionCallNode:
			name := functionCallName(call)
			if name == "" || ctx == nil || ctx.Resolver == nil {
				return
			}
			fn, ok := ctx.Resolver.ResolveFunction(name)
			if !ok || !fn.Deprecated {
				return
			}
			seen[expr] = struct{}{}
			issues = append(issues, deprecatedCallIssue(filename, call, "function", fn.Name, fn.DeprecationMessage))
		}
	}

	var walk func(node ast.Node, class *ast.ClassNode)
	walk = func(node ast.Node, class *ast.ClassNode) {
		switch n := node.(type) {
		case *ast.ClassNode:
			for _, methodNode := range n.Methods {
				walk(methodNode, n)
			}
		case *ast.FunctionNode:
			fnScope := analysisFunctionScope(ctx, class, n, fileCtx)
			walkStatementsForArgTypesUsing(n.Body, fnScope, ctx, filename, nil, observer)
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
