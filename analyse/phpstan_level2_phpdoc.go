package analyse

import (
	"fmt"
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
)

const (
	level2PHPDocClassCode        = "Level2.PHPDocClass"
	level2PHPDocParamNameCode    = "Level2.PHPDocParamName"
	level2PHPDocParamTypeCode    = "Level2.PHPDocParamType"
	level2PHPDocPropertyTypeCode = "Level2.PHPDocPropertyType"
	level2PHPDocReturnTypeCode   = "Level2.PHPDocReturnType"
)

func phpDocIssuesForFile(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
	return ensureStructuralIssues(filename, nodes, ctx).phpDocIssues
}

func appendPHPDocIssuesOnNode(filename string, node ast.Node, class *ast.ClassNode, ft FileTypeContext, ctx *AnalysisContext, issues *[]AnalysisIssue) {
	switch n := node.(type) {
	case *ast.FunctionNode:
		appendFunctionPHPDocIssues(filename, n, class, ft, ctx, issues)
	case *ast.InterfaceMethodNode:
		returnType := ""
		if n.ReturnType != nil {
			returnType = n.ReturnType.TokenLiteral()
		}
		appendCallablePHPDocIssues(filename, n, n.Params, returnType, n.PHPDoc, class, ft, ctx, issues)
	case *ast.PropertyNode:
		appendPropertyPHPDocIssues(filename, n, class, ft, ctx, issues)
	}
}

func appendFunctionPHPDocIssues(filename string, fn *ast.FunctionNode, class *ast.ClassNode, ft FileTypeContext, ctx *AnalysisContext, issues *[]AnalysisIssue) {
	if fn == nil {
		return
	}
	appendCallablePHPDocIssues(filename, fn, fn.Params, fn.ReturnType, fn.PHPDoc, class, ft, ctx, issues)
}

func appendCallablePHPDocIssues(filename string, declaration ast.Node, params []ast.Node, nativeReturn string, doc *ast.PHPDocNode, class *ast.ClassNode, ft FileTypeContext, ctx *AnalysisContext, issues *[]AnalysisIssue) {
	if doc == nil {
		return
	}
	templates := phpDocTemplateNames(class, doc)

	for _, documented := range doc.Params {
		appendUnknownPHPDocClasses(filename, declaration, documented.Type, templates, ft, ctx, issues)
		param, ok := phpDocParameter(params, documented.Name)
		if !ok {
			*issues = append(*issues, issueSpan(filename, declaration, level2PHPDocParamNameCode, fmt.Sprintf(
				"PHPDoc tag @param references unknown parameter $%s.", documented.Name,
			)))
			continue
		}
		native := paramTypeName(param)
		if native == "" || phpDocUsesTemplate(documented.Type, templates) {
			continue
		}
		if !phpDocTypeFitsNative(documented.Type, native, ft, ctx) {
			*issues = append(*issues, issueSpan(filename, param, level2PHPDocParamTypeCode, fmt.Sprintf(
				"PHPDoc type %s for parameter $%s is not compatible with native type %s.", documented.Type, documented.Name, native,
			)))
		}
	}

	if doc.ReturnType == "" {
		return
	}
	appendUnknownPHPDocClasses(filename, declaration, doc.ReturnType, templates, ft, ctx, issues)
	if nativeReturn != "" && !phpDocUsesTemplate(doc.ReturnType, templates) && !phpDocTypeFitsNative(doc.ReturnType, nativeReturn, ft, ctx) {
		*issues = append(*issues, issueSpan(filename, declaration, level2PHPDocReturnTypeCode, fmt.Sprintf(
			"PHPDoc return type %s is not compatible with native return type %s.", doc.ReturnType, nativeReturn,
		)))
	}
}

func phpDocParameter(params []ast.Node, name string) (*ast.ParamNode, bool) {
	for _, paramNode := range params {
		if param, ok := paramNode.(*ast.ParamNode); ok && param.Name == name {
			return param, true
		}
	}
	return nil, false
}

func appendPropertyPHPDocIssues(filename string, property *ast.PropertyNode, class *ast.ClassNode, ft FileTypeContext, ctx *AnalysisContext, issues *[]AnalysisIssue) {
	if property == nil || property.PHPDoc == nil || property.PHPDoc.VarType == "" {
		return
	}
	templates := phpDocTemplateNames(class, property.PHPDoc)
	documented := property.PHPDoc.VarType
	appendUnknownPHPDocClasses(filename, property, documented, templates, ft, ctx, issues)
	if property.TypeHint == "" || phpDocUsesTemplate(documented, templates) || phpDocTypeFitsNative(documented, property.TypeHint, ft, ctx) {
		return
	}
	*issues = append(*issues, issueSpan(filename, property, level2PHPDocPropertyTypeCode, fmt.Sprintf(
		"PHPDoc type %s for property $%s is not compatible with native type %s.", documented, property.Name, property.TypeHint,
	)))
}

func appendUnknownPHPDocClasses(filename string, declaration ast.Node, raw string, templates map[string]struct{}, ft FileTypeContext, ctx *AnalysisContext, issues *[]AnalysisIssue) {
	if ctx == nil || ctx.Resolver == nil {
		return
	}
	for _, name := range referencedClassTypes(raw, ft) {
		if isSpecialClassName(name) {
			continue
		}
		if _, template := templates[asciiLowerIdent(strings.TrimPrefix(name, `\`))]; template {
			continue
		}
		if _, ok := ctx.Resolver.ResolveClass(name); ok {
			continue
		}
		*issues = append(*issues, issueSpan(filename, declaration, level2PHPDocClassCode, fmt.Sprintf("PHPDoc references unknown class %s.", name)))
	}
}

func phpDocTypeFitsNative(documented, native string, ft FileTypeContext, ctx *AnalysisContext) bool {
	documentedType := ParseType(normalizeTypeWithContext(documented, ft))
	nativeType := ParseType(normalizeTypeWithContext(native, ft))
	if documentedType.IsEmpty() || nativeType.IsEmpty() {
		return true
	}
	return nativeType.AcceptsWithContext(documentedType, nil, ctx)
}

func phpDocTemplateNames(class *ast.ClassNode, doc *ast.PHPDocNode) map[string]struct{} {
	var templates map[string]struct{}
	add := func(candidate *ast.PHPDocNode) {
		if candidate == nil {
			return
		}
		for _, template := range candidate.Templates {
			if templates == nil {
				templates = make(map[string]struct{}, len(candidate.Templates))
			}
			templates[asciiLowerIdent(template.Name)] = struct{}{}
		}
	}
	if class != nil {
		add(class.PHPDoc)
	}
	add(doc)
	return templates
}

func phpDocUsesTemplate(raw string, templates map[string]struct{}) bool {
	if len(templates) == 0 {
		return false
	}
	start := -1
	for index, char := range raw {
		if char == '_' || char == '\\' || char >= '0' && char <= '9' || char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z' {
			if start < 0 {
				start = index
			}
			continue
		}
		if start >= 0 {
			if _, ok := templates[asciiLowerIdent(raw[start:index])]; ok {
				return true
			}
			start = -1
		}
	}
	if start >= 0 {
		_, ok := templates[asciiLowerIdent(raw[start:])]
		return ok
	}
	return false
}

func init() {
	for _, rule := range []struct {
		code string
	}{
		{level2PHPDocClassCode},
		{level2PHPDocParamNameCode},
		{level2PHPDocParamTypeCode},
		{level2PHPDocPropertyTypeCode},
		{level2PHPDocReturnTypeCode},
	} {
		code := rule.code
		RegisterAnalysisRuleWithLevel(code, 2, "level2", func(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
			return filterIssuesByCode(phpDocIssuesForFile(filename, nodes, ctx), code)
		})
	}
}
