package analyse

import (
	"github.com/ayanozturk/go-php-parser/syntax"
)

// CheckLanguageIssuesFromCST is the CST-direct analogue of Level0Rule's
// language checks (level0LanguageCode/level0InvocationCode): undefined goto
// labels, duplicate array literal keys, include/require path existence,
// ++/-- on non-writable expressions, void/unset casts, invalid regex
// patterns, and printf/sprintf placeholder counts.
//
// checkLanguageOnNode ignores its FileTypeContext parameter, so unlike
// walkAllWithFileContext-driven rules this check needs no namespace/class/
// function scope at all - it walks the CST flatly with syntax.Walk.
// KindCallExpr uses a CST-native leaf (appendLanguageCallIssuesFromCST);
// remaining matched kinds use CST-native leaves in
// appendLanguageNonCallIssuesFromCST (no LowerStmtNode/LowerExprNode).
//
// walkAllConfigured (the shared ast.Node dispatcher) has no case for
// *ast.SwitchNode/*ast.SwitchCaseNode, so switch statement bodies are
// invisible to every rule it drives, including this one - a pre-existing
// gap, not a deliberate design choice. This CST-direct walk skips
// KindSwitchStmt subtrees entirely to match that gap; see
// /memories/repo/cst-direct-migration.md for the full writeup.
func CheckLanguageIssuesFromCST(filename string, content []byte) []AnalysisIssue {
	return checkLanguageIssuesFromParsed(filename, syntax.Parse(content))
}

func checkLanguageIssuesFromParsed(filename string, res *syntax.ParseResult) []AnalysisIssue {
	if res == nil || res.File == nil || res.File.Root == nil {
		return nil
	}

	var issues []AnalysisIssue
	labels := map[string]struct{}{}
	var gotos []languageCSTGoto
	syntax.Walk(res.File.Root, func(n *syntax.RedNode) bool {
		switch n.Kind() {
		case syntax.KindSwitchStmt:
			// walkAllConfigured (the ast.Node-based dispatcher) has no case
			// for *ast.SwitchNode/*ast.SwitchCaseNode, so switch bodies (case
			// conditions and case bodies alike) are invisible to every rule
			// driven by it - including this one. Preserve that gap here so
			// this stays a faithful, swappable replacement; see
			// /memories/repo/cst-direct-migration.md for the full writeup.
			return false
		case syntax.KindLabelStmt, syntax.KindGotoStmt,
			syntax.KindArrayExpr, syntax.KindUnaryExpr, syntax.KindIncludeExpr, syntax.KindCastExpr:
			appendLanguageNonCallIssuesFromCST(filename, n, labels, &gotos, &issues)
		case syntax.KindCallExpr:
			appendLanguageCallIssuesFromCST(filename, n, &issues)
		}
		return true
	})
	appendUndefinedGotoIssuesFromCST(filename, labels, gotos, &issues)
	return issues
}
