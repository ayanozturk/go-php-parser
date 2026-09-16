package analyse

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

const emptyStatementCode = "Generic.CodeAnalysis.EmptyStatement"

type EmptyStatementRule struct{}

func appendEmptyStatementIssue(filename string, node ast.Node, issues *[]AnalysisIssue) {
	empty, ok := node.(*ast.EmptyStatementNode)
	if !ok {
		return
	}
	*issues = append(*issues, issueSpan(filename, empty, emptyStatementCode, "Empty statement detected"))
}

func (r *EmptyStatementRule) CheckIssuesWithSource(filename string, content []byte, nodes []ast.Node) []AnalysisIssue {
	if len(nodes) == 0 && len(content) > 0 {
		nodes, _ = syntax.ParseAST(content)
	}
	var issues []AnalysisIssue
	walkAllWithoutTypeContext(nodes, func(node ast.Node) {
		appendEmptyStatementIssue(filename, node, &issues)
	})
	return issues
}

func init() {
	RegisterAnalysisRuleWithContext(emptyStatementCode, func(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
		ctx = ensureSharedFileDiagnostics(filename, nodes, ctx)
		return ctx.emptyStatementIssues
	})
}
