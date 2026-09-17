package analyse

import (
	"github.com/ayanozturk/go-php-parser/syntax"
)

// CheckPropertyCallableTypeIssuesFromCST is the CST-direct analogue of
// Level0Rule's level0PropertyCallableTypeCode check
// (appendPropertyCallableTypeIssue): a typed property or promoted
// constructor parameter cannot declare "callable" in its type.
//
// Like checkLanguageOnNode, appendPropertyCallableTypeIssue needs no
// FileTypeContext/class/currentFn scope, so a flat syntax.Walk is safe.
// KindPropertyDecl/KindParam are lowered in isolation (via
// syntax.LowerPropertyDeclNode/syntax.LowerParamNode) and fed unchanged into
// appendPropertyCallableTypeIssue.
func CheckPropertyCallableTypeIssuesFromCST(filename string, content []byte) []AnalysisIssue {
	res := syntax.Parse(content)
	if res.File == nil || res.File.Root == nil {
		return nil
	}

	var issues []AnalysisIssue
	syntax.Walk(res.File.Root, func(n *syntax.RedNode) bool {
		switch n.Kind() {
		case syntax.KindPropertyDecl:
			for _, prop := range syntax.LowerPropertyDeclNode(n, res.File) {
				appendPropertyCallableTypeIssue(filename, prop, &issues)
			}
		case syntax.KindParam:
			if param := syntax.LowerParamNode(n, res.File); param != nil {
				appendPropertyCallableTypeIssue(filename, param, &issues)
			}
		}
		// Keep descending: a property/param default value can contain a
		// nested closure with its own KindParam nodes.
		return true
	})
	return issues
}
