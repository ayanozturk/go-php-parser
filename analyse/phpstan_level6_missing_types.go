package analyse

import (
	"fmt"
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
)

const (
	level6MissingGenericTypeCode  = "Level6.MissingGenericType"
	level6MissingIterableTypeCode = "Level6.MissingIterableValueType"
)

func missingTypeIssuesForFile(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
	return ensureStructuralIssues(filename, nodes, ctx).missingTypeIssues
}

func appendMissingTypeIssuesOnNode(filename string, node ast.Node, ft FileTypeContext, ctx *AnalysisContext, issues *[]AnalysisIssue) {
	switch n := node.(type) {
	case *ast.FunctionNode:
		appendCallableMissingTypeIssues(filename, n, n.Params, n.ReturnType, n.PHPDoc, ft, ctx, issues)
	case *ast.InterfaceMethodNode:
		returnType := ""
		if n.ReturnType != nil {
			returnType = n.ReturnType.TokenLiteral()
		}
		appendCallableMissingTypeIssues(filename, n, n.Params, returnType, n.PHPDoc, ft, ctx, issues)
	case *ast.PropertyNode:
		raw := n.TypeHint
		if n.PHPDoc != nil && n.PHPDoc.VarType != "" {
			raw = n.PHPDoc.VarType
		}
		appendMissingTypeIssues(filename, n, raw, ft, ctx, issues)
	}
}

func appendCallableMissingTypeIssues(filename string, declaration ast.Node, params []ast.Node, nativeReturn string, doc *ast.PHPDocNode, ft FileTypeContext, ctx *AnalysisContext, issues *[]AnalysisIssue) {
	for _, paramNode := range params {
		param, ok := paramNode.(*ast.ParamNode)
		if !ok {
			continue
		}
		raw := paramTypeName(param)
		if doc != nil {
			if documented, found := phpDocDocumentedParameter(doc, param.Name); found {
				raw = documented.Type
			}
		}
		appendMissingTypeIssues(filename, param, raw, ft, ctx, issues)
	}
	rawReturn := nativeReturn
	if doc != nil && doc.ReturnType != "" {
		rawReturn = doc.ReturnType
	}
	appendMissingTypeIssues(filename, declaration, rawReturn, ft, ctx, issues)
}

func phpDocDocumentedParameter(doc *ast.PHPDocNode, name string) (ast.PHPDocParam, bool) {
	if doc == nil {
		return ast.PHPDocParam{}, false
	}
	for _, param := range doc.Params {
		if param.Name == name {
			return param, true
		}
	}
	return ast.PHPDocParam{}, false
}

func appendMissingTypeIssues(filename string, declaration ast.Node, raw string, ft FileTypeContext, ctx *AnalysisContext, issues *[]AnalysisIssue) {
	if ctx == nil || ctx.Resolver == nil {
		return
	}
	raw = stripBalancedOuterTypeParens(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), "?")))
	if raw == "" {
		return
	}
	if parts := splitTopLevelTypes(raw, '|'); len(parts) > 1 {
		for _, part := range parts {
			appendMissingTypeIssues(filename, declaration, part, ft, ctx, issues)
		}
		return
	}
	if parts := splitTopLevelTypes(raw, '&'); len(parts) > 1 {
		for _, part := range parts {
			appendMissingTypeIssues(filename, declaration, part, ft, ctx, issues)
		}
		return
	}
	if instance, ok := parseExactGenericTypeFromString(raw); ok {
		for _, argument := range instance.TypeArguments {
			appendMissingTypeIssues(filename, declaration, argument, ft, ctx, issues)
		}
		return
	}
	if params, returnType, ok := phpDocCallableSignature(raw); ok {
		for _, param := range params {
			appendMissingTypeIssues(filename, declaration, param, ft, ctx, issues)
		}
		appendMissingTypeIssues(filename, declaration, returnType, ft, ctx, issues)
		return
	}
	if body, ok := arrayShapeBody(raw); ok {
		for _, entry := range splitTopLevelTypes(body, ',') {
			_, value, valid := splitArrayShapeEntry(entry)
			if valid {
				appendMissingTypeIssues(filename, declaration, value, ft, ctx, issues)
			}
		}
		return
	}
	if strings.HasSuffix(raw, "[]") {
		appendMissingTypeIssues(filename, declaration, strings.TrimSpace(strings.TrimSuffix(raw, "[]")), ft, ctx, issues)
		return
	}

	switch asciiLowerIdent(raw) {
	case "array", "iterable", "list", "non-empty-array", "non-empty-list":
		*issues = append(*issues, issueSpan(filename, declaration, level6MissingIterableTypeCode, fmt.Sprintf("Iterable type %s does not specify its value type.", raw)))
		return
	}

	name := ft.resolveClassLike(raw)
	if isSpecialClassName(name) {
		return
	}
	resolved, ok := ctx.Resolver.ResolveClass(name)
	if ok && len(resolved.TemplateParams) > 0 {
		*issues = append(*issues, issueSpan(filename, declaration, level6MissingGenericTypeCode, fmt.Sprintf("Generic class %s does not specify its template types.", name)))
	}
}

func init() {
	for _, code := range []string{level6MissingGenericTypeCode, level6MissingIterableTypeCode} {
		code := code
		RegisterAnalysisRuleWithLevel(code, 6, "level6", func(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
			return filterIssuesByCode(missingTypeIssuesForFile(filename, nodes, ctx), code)
		})
	}
}
