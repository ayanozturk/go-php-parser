package analyse

import (
	"github.com/ayanozturk/go-php-parser/syntax"
)

// CheckPropertyCallableTypeIssuesFromCST is the CST-direct analogue of
// Level0Rule's level0PropertyCallableTypeCode check
// (appendPropertyCallableTypeIssue): a typed property or promoted
// constructor parameter cannot declare "callable" in its type.
//
// Like checkLanguageOnNode, this check needs no FileTypeContext/class/
// currentFn scope, so a flat syntax.Walk is safe. PropertyDecl/Param are
// inspected via CST accessors (PropertyDeclType / ParamIsPromoted / …)
// without LowerPropertyDeclNode / LowerParamNode.
func CheckPropertyCallableTypeIssuesFromCST(filename string, content []byte) []AnalysisIssue {
	return checkPropertyCallableTypeIssuesFromParsed(filename, syntax.Parse(content))
}

func checkPropertyCallableTypeIssuesFromParsed(filename string, res *syntax.ParseResult) []AnalysisIssue {
	if res == nil || res.File == nil || res.File.Root == nil {
		return nil
	}

	var issues []AnalysisIssue
	syntax.Walk(res.File.Root, func(n *syntax.RedNode) bool {
		switch n.Kind() {
		case syntax.KindPropertyDecl:
			appendPropertyCallableTypeIssueFromCST(filename, n, &issues)
		case syntax.KindParam:
			appendPromotedParamCallableTypeIssueFromCST(filename, n, &issues)
		}
		// Keep descending: a property/param default value can contain a
		// nested closure with its own KindParam nodes.
		return true
	})
	return issues
}
