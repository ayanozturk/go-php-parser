package analyse

import (
	"fmt"
	"strings"

	"github.com/ayanozturk/go-php-parser/syntax"
)

// appendTypeRefUseIssuesFromCST is the CST-native analogue of
// checkTypeReferenceOnNode's *ast.UseNode arm. Class/default imports are
// intentionally no-ops (same as the AST path) and never LowerUseDeclNode.
// Only `use function` / `use const` pay existence checks.
//
// Issue spans match LowerUseDeclNode: the whole KindUseDecl span.
func appendTypeRefUseIssuesFromCST(filename string, useDecl *syntax.RedNode, ctx *AnalysisContext, guards reflectionGuards, issues *[]AnalysisIssue) {
	if useDecl == nil || useDecl.Kind() != syntax.KindUseDecl || ctx == nil || ctx.Resolver == nil {
		return
	}
	syntax.ForEachUseDeclImport(useDecl, func(imp syntax.UseDeclImport) {
		switch imp.Type {
		case "function":
			name := strings.TrimPrefix(imp.Path, `\`)
			if ctx.Resolver.FunctionExists(name) || guards.hasFunction(name) {
				return
			}
			*issues = append(*issues, issueSpanRed(filename, useDecl, level0SymbolsCode,
				fmt.Sprintf("Used function %s not found.", imp.Path)))
		case "const":
			name := strings.TrimPrefix(imp.Path, `\`)
			if ctx.Resolver.ConstantExists(name) || guards.hasConstant(name) {
				return
			}
			*issues = append(*issues, issueSpanRed(filename, useDecl, level0SymbolsCode,
				fmt.Sprintf("Used constant %s not found.", imp.Path)))
		default:
			// Class imports: no type-ref check (namespace-prefix aliases OK).
		}
	})
}
