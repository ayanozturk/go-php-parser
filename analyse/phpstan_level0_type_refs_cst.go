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

// appendTypeRefPropertyIssuesFromCST checks property type hints without
// LowerPropertyDeclNode. Span parity: single-name decls use the whole
// property decl; multi-name decls use each variable token span (same as
// property-callable CST).
func appendTypeRefPropertyIssuesFromCST(filename string, propDecl *syntax.RedNode, ft FileTypeContext, ctx *AnalysisContext, guards reflectionGuards, issues *[]AnalysisIssue) {
	if propDecl == nil || propDecl.Kind() != syntax.KindPropertyDecl || ctx == nil {
		return
	}
	typ := syntax.PropertyDeclType(propDecl)
	if typ == nil {
		return
	}
	raw := syntax.TypeText(typ)
	if raw == "" {
		return
	}
	vars := syntax.PropertyDeclVariables(propDecl)
	if len(vars) == 0 {
		return
	}
	if len(vars) == 1 {
		name := syntax.VariableName(vars[0])
		start := propDecl.Pos()
		checkTypeReference(filename, start, "Property $"+name, raw, ft, ctx, guards, issues)
		return
	}
	for _, v := range vars {
		name := syntax.VariableName(v)
		start, _ := syntax.RawSpanPositions(v)
		checkTypeReference(filename, start, "Property $"+name, raw, ft, ctx, guards, issues)
	}
}
