package analyse

import (
	"fmt"

	"github.com/ayanozturk/go-php-parser/syntax"
)

// appendTypeRefCatchIssuesFromCST is the CST-native analogue of
// checkTypeReferenceOnNode's *ast.CatchNode arm. Reads catch type names
// without LowerCatchClauseNode (which would also lower the catch body).
func appendTypeRefCatchIssuesFromCST(filename string, catch *syntax.RedNode, ft FileTypeContext, ctx *AnalysisContext, guards reflectionGuards, issues *[]AnalysisIssue) {
	if catch == nil || catch.Kind() != syntax.KindCatchClause || ctx == nil || ctx.Resolver == nil {
		return
	}
	for _, catchType := range syntax.CatchClauseTypeNames(catch) {
		name := ft.resolveClassLike(catchType)
		resolved, ok := ctx.Resolver.ResolveClass(name)
		if !ok {
			if guards.hasClass(name) {
				continue
			}
			*issues = append(*issues, issueSpanRed(filename, catch, level0SymbolsCode,
				fmt.Sprintf("Caught class %s not found.", name)))
			continue
		}
		if resolved.Kind == "trait" || resolved.Kind == "enum" {
			*issues = append(*issues, issueSpanRed(filename, catch, level0ClassModelCode,
				fmt.Sprintf("Caught %s %s is not throwable.", resolved.Kind, resolved.Name)))
		}
	}
}
