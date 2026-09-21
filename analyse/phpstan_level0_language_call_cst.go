package analyse

import (
	"fmt"
	"regexp"

	"github.com/ayanozturk/go-php-parser/syntax"
)

// appendLanguageCallIssuesFromCST is the CST-native analogue of
// checkLanguageOnNode's *ast.FunctionCallNode arm (preg_match regex validity
// and printf/sprintf placeholder counts). Method-like calls are skipped —
// they lower to MethodCallNode and never enter that arm.
//
// Avoids syntax.LowerExprNode for KindCallExpr on the fused/flat language path.
func appendLanguageCallIssuesFromCST(filename string, call *syntax.RedNode, issues *[]AnalysisIssue) {
	if call == nil || call.Kind() != syntax.KindCallExpr {
		return
	}
	if syntax.CallIsMethodLike(call) {
		return
	}
	callee := syntax.CallCallee(call)
	if callee == nil || !syntax.IsNameKind(callee.Kind()) {
		return
	}
	name := asciiLowerIdent(trimFunctionCallNamePrefix(syntax.NameText(callee)))
	if name != "preg_match" && name != "printf" && name != "sprintf" {
		return
	}
	args := syntax.CallArgs(call)
	if len(args) == 0 {
		return
	}
	first := syntax.ArgExpr(args[0])
	if first == nil {
		return
	}
	lit, ok := syntax.LiteralStringValue(first)
	if !ok {
		return
	}
	if name == "preg_match" {
		if _, err := regexp.Compile(extractRegexpBody(lit)); err != nil {
			*issues = append(*issues, issueSpanRed(filename, call, level0LanguageCode,
				fmt.Sprintf("Regex pattern is invalid: %s", err.Error())))
		}
		return
	}
	// printf / sprintf
	required := countPrintfPlaceholders(lit)
	if required > len(args)-1 {
		*issues = append(*issues, issueSpanRed(filename, call, level0InvocationCode,
			fmt.Sprintf("Call to function %s contains %d placeholders, %d values given.", name, required, len(args)-1)))
	}
}
