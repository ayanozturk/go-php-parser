package analyse

import (
	"fmt"

	"github.com/ayanozturk/go-php-parser/ast"
)

const level3ThrowTypeCode = "PHPStan.Level3.ThrowType"

func checkLevel3ThrowTypes(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
	return throwTypeIssuesForFile(filename, nodes, ctx)
}

func appendThrowTypeOnNode(filename string, node ast.Node, ft FileTypeContext, ctx *AnalysisContext, issues *[]AnalysisIssue) {
	throwNode, ok := node.(*ast.ThrowNode)
	if !ok {
		return
	}

	className := resolveThrownClassName(throwNode.Expr, ft)
	if className == "" || isSpecialClassName(className) {
		return
	}
	resolved, ok := ctx.Resolver.ResolveClass(className)
	if !ok {
		return
	}
	if resolved.Kind == "trait" || resolved.Kind == "enum" {
		*issues = append(*issues, issueSpan(filename, throwNode, level3ThrowTypeCode, fmt.Sprintf("Cannot throw %s %s.", resolved.Kind, resolved.Name)))
		return
	}
	if !isThrowableClass(className, ctx.Resolver) {
		*issues = append(*issues, issueSpan(filename, throwNode, level3ThrowTypeCode, fmt.Sprintf("Invalid type %s to throw.", resolved.Name)))
	}
}

func init() {
	RegisterAnalysisRuleWithLevel(level3ThrowTypeCode, 3, "phpstan.level3", checkLevel3ThrowTypes)
}
