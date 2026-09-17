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

// CheckIssuesWithSource detects empty statements (bare `;`). When raw source
// is available it walks the syntax CST directly (see syntax.Walk) rather than
// lowering to []ast.Node first, avoiding the extra conversion pass for this
// self-contained check. When pre-lowered nodes are supplied (fused walk from
// ensureSharedFileDiagnostics), it dispatches on ast.Node as before.
func (r *EmptyStatementRule) CheckIssuesWithSource(filename string, content []byte, nodes []ast.Node) []AnalysisIssue {
	var issues []AnalysisIssue
	if len(nodes) == 0 && len(content) > 0 {
		res := syntax.Parse(content)
		syntax.Walk(res.File.Root, func(n *syntax.RedNode) bool {
			if n.Kind() == syntax.KindEmptyStmt {
				issues = append(issues, issueSpanRed(filename, n, emptyStatementCode, "Empty statement detected"))
			}
			return true
		})
		return issues
	}
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
