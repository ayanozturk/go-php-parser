package analyse

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ayanozturk/go-php-parser/syntax"
)

// languageCSTGoto holds a goto target collected without LowerStmtNode.
// Positions are snapshotted at walk time because syntax.Walk reuses RedNode.
type languageCSTGoto struct {
	label     string
	line      int
	column    int
	endLine   int
	endColumn int
}

// appendLanguageArrayIssuesFromCST is the CST-native analogue of
// checkLanguageOnNode's *ast.ArrayNode arm (duplicate literal keys).
// Avoids LowerExprNode of the whole array (including values).
func appendLanguageArrayIssuesFromCST(filename string, arr *syntax.RedNode, issues *[]AnalysisIssue) {
	if arr == nil || arr.Kind() != syntax.KindArrayExpr {
		return
	}
	seen := map[string]struct{}{}
	for _, el := range syntax.ArrayElements(arr) {
		key := syntax.ArrayElementKey(el)
		if key == nil {
			continue
		}
		lit, ok := literalKeyFromCST(key)
		if !ok {
			continue
		}
		if _, exists := seen[lit]; exists {
			*issues = append(*issues, issueSpanRed(filename, el, level0LanguageCode,
				fmt.Sprintf("Array has %s duplicate key.", lit)))
			continue
		}
		seen[lit] = struct{}{}
	}
}

// appendLanguageUnaryIssuesFromCST handles KindUnaryExpr ++/-- writable checks.
func appendLanguageUnaryIssuesFromCST(filename string, n *syntax.RedNode, issues *[]AnalysisIssue) {
	if n == nil || n.Kind() != syntax.KindUnaryExpr {
		return
	}
	op := syntax.UnaryOperator(n)
	switch op {
	case "++", "--":
		operand := syntax.UnaryOperand(n)
		if operand == nil || syntax.IsWritableExprKind(operand) {
			return
		}
		*issues = append(*issues, issueSpanRed(filename, n, level0LanguageCode,
			fmt.Sprintf("Cannot use %s on non-variable expression.", op)))
	}
}

// appendLanguageIncludeIssuesFromCST handles KindIncludeExpr path existence
// (include/require/include_once/require_once) without LowerExprNode.
func appendLanguageIncludeIssuesFromCST(filename string, n *syntax.RedNode, issues *[]AnalysisIssue) {
	if n == nil || n.Kind() != syntax.KindIncludeExpr {
		return
	}
	op := syntax.KeywordUnaryOperator(n)
	switch asciiLowerIdent(op) {
	case "include", "include_once", "require", "require_once":
	default:
		return
	}
	operand := syntax.UnaryOperand(n)
	if operand == nil {
		return
	}
	path, ok := syntax.LiteralStringValue(operand)
	if !ok {
		return
	}
	if _, err := os.Stat(resolveIncludePath(filename, path)); err != nil {
		*issues = append(*issues, issueSpanRed(filename, n, level0LanguageCode,
			fmt.Sprintf("Path in %s() \"%s\" is not a file or it does not exist.", op, path)))
	}
}

// appendLanguageCastIssuesFromCST handles KindCastExpr void/unset bans.
func appendLanguageCastIssuesFromCST(filename string, n *syntax.RedNode, issues *[]AnalysisIssue) {
	if n == nil || n.Kind() != syntax.KindCastExpr {
		return
	}
	typ := syntax.CastTypeName(n)
	if !strings.EqualFold(typ, "unset") && !strings.EqualFold(typ, "void") {
		return
	}
	*issues = append(*issues, issueSpanRed(filename, n, level0LanguageCode,
		fmt.Sprintf("Cannot cast to %s.", typ)))
}

// collectLanguageLabelGotoFromCST records labels / gotos without LowerStmtNode.
func collectLanguageLabelGotoFromCST(n *syntax.RedNode, labels map[string]struct{}, gotos *[]languageCSTGoto) {
	if n == nil {
		return
	}
	name := syntax.LabelOrGotoName(n)
	if name == "" {
		return
	}
	switch n.Kind() {
	case syntax.KindLabelStmt:
		labels[name] = struct{}{}
	case syntax.KindGotoStmt:
		start, end := n.Pos(), n.EndPos()
		*gotos = append(*gotos, languageCSTGoto{
			label:     name,
			line:      start.Line,
			column:    start.Column,
			endLine:   end.Line,
			endColumn: end.Column,
		})
	}
}

func appendUndefinedGotoIssuesFromCST(filename string, labels map[string]struct{}, gotos []languageCSTGoto, issues *[]AnalysisIssue) {
	for _, g := range gotos {
		if _, ok := labels[g.label]; ok {
			continue
		}
		*issues = append(*issues, AnalysisIssue{
			Filename:  filename,
			Line:      g.line,
			Column:    g.column,
			EndLine:   g.endLine,
			EndColumn: g.endColumn,
			Code:      level0LanguageCode,
			Message:   fmt.Sprintf("Goto to undefined label %s.", g.label),
		})
	}
}

func literalKeyFromCST(n *syntax.RedNode) (string, bool) {
	if n == nil {
		return "", false
	}
	if s, ok := syntax.LiteralStringValue(n); ok {
		return strconv.Quote(s), true
	}
	if v, ok := syntax.LiteralIntValue(n); ok {
		return strconv.FormatInt(v, 10), true
	}
	return "", false
}

// appendLanguageNonCallIssuesFromCST dispatches remaining flat-language kinds
// that previously paid LowerExprNode / LowerStmtNode.
func appendLanguageNonCallIssuesFromCST(filename string, n *syntax.RedNode, labels map[string]struct{}, gotos *[]languageCSTGoto, issues *[]AnalysisIssue) {
	if n == nil {
		return
	}
	switch n.Kind() {
	case syntax.KindLabelStmt, syntax.KindGotoStmt:
		collectLanguageLabelGotoFromCST(n, labels, gotos)
	case syntax.KindArrayExpr:
		appendLanguageArrayIssuesFromCST(filename, n, issues)
	case syntax.KindUnaryExpr:
		appendLanguageUnaryIssuesFromCST(filename, n, issues)
	case syntax.KindIncludeExpr:
		appendLanguageIncludeIssuesFromCST(filename, n, issues)
	case syntax.KindCastExpr:
		appendLanguageCastIssuesFromCST(filename, n, issues)
	}
}