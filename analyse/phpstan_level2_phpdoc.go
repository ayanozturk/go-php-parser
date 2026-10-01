package analyse

import (
	"fmt"
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
)

const (
	level2PHPDocClassCode            = "Level2.PHPDocClass"
	level2PHPDocGenericLessCode      = "Level2.PHPDocGenericLessTypes"
	level2PHPDocGenericMoreCode      = "Level2.PHPDocGenericMoreTypes"
	level2PHPDocGenericBoundCode     = "Level2.PHPDocGenericNotSubtype"
	level2PHPDocMethodVarianceCode   = "Level2.PHPDocMethodVariance"
	level2PHPDocTemplateVarianceCode = "Level2.PHPDocTemplateVariance"
	level2PHPDocNotGenericCode       = "Level2.PHPDocNotGeneric"
	level2PHPDocParamNameCode        = "Level2.PHPDocParamName"
	level2PHPDocParamTypeCode        = "Level2.PHPDocParamType"
	level2PHPDocPropertyTypeCode     = "Level2.PHPDocPropertyType"
	level2PHPDocReturnTypeCode       = "Level2.PHPDocReturnType"
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
			returnType = ast.TypeText(n.ReturnType)
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
	appendCallablePHPDocIssues(filename, fn, fn.Params, ast.TypeText(fn.ReturnType), fn.PHPDoc, class, ft, ctx, issues)
}

func appendCallablePHPDocIssues(filename string, declaration ast.Node, params []ast.Node, nativeReturn string, doc *ast.PHPDocNode, class *ast.ClassNode, ft FileTypeContext, ctx *AnalysisContext, issues *[]AnalysisIssue) {
	if doc == nil {
		return
	}
	for _, template := range doc.Templates {
		if template.Variance != "" {
			*issues = append(*issues, issueSpan(filename, declaration, level2PHPDocMethodVarianceCode, fmt.Sprintf(
				"Variance annotations are only allowed on class and interface template types; %s is declared on a callable.", template.Name,
			)))
		}
	}
	templates := phpDocTemplateNames(class, doc)
	var classDoc *ast.PHPDocNode
	if class != nil {
		classDoc = class.PHPDoc
	}
	aliases := phpDocTypeAliasBindings(classDoc, doc)

	for _, documented := range doc.Params {
		effectiveType := expandPHPDocTypeAliases(documented.Type, aliases)
		if indexed, ok := resolvePHPDocIndexedTypes(effectiveType, ft); ok {
			effectiveType = indexed
		}
		appendPHPDocTypeIssues(filename, declaration, effectiveType, templates, ft, ctx, issues)
		param, ok := phpDocParameter(params, documented.Name)
		if !ok {
			*issues = append(*issues, issueSpan(filename, declaration, level2PHPDocParamNameCode, fmt.Sprintf(
				"PHPDoc tag @param references unknown parameter $%s.", documented.Name,
			)))
			continue
		}
		appendTemplateVarianceIssue(filename, declaration, class, doc, effectiveType, GenericContravariant, "parameter $"+documented.Name, ft, ctx, issues)
		native := paramTypeName(param)
		if native == "" || phpDocUsesTemplate(effectiveType, templates) {
			continue
		}
		if !phpDocTypeFitsNative(effectiveType, native, ft, ctx) {
			*issues = append(*issues, issueSpan(filename, param, level2PHPDocParamTypeCode, fmt.Sprintf(
				"PHPDoc type %s for parameter $%s is not compatible with native type %s.", documented.Type, documented.Name, native,
			)))
		}
	}

	if doc.ReturnType == "" {
		return
	}
	expandedReturn := expandPHPDocTypeAliases(doc.ReturnType, aliases)
	if indexed, ok := resolvePHPDocIndexedTypes(expandedReturn, ft); ok {
		expandedReturn = indexed
	}
	appendTemplateVarianceIssue(filename, declaration, class, doc, expandedReturn, GenericCovariant, "return type", ft, ctx, issues)
	if conditional, ok := parsePHPDocConditionalType(expandedReturn); ok {
		branchUnion := conditional.thenType + "|" + conditional.elseType
		appendPHPDocTypeIssues(filename, declaration, branchUnion, templates, ft, ctx, issues)
		if nativeReturn != "" && !phpDocUsesTemplate(branchUnion, templates) && !phpDocTypeFitsNative(branchUnion, nativeReturn, ft, ctx) {
			*issues = append(*issues, issueSpan(filename, declaration, level2PHPDocReturnTypeCode, fmt.Sprintf(
				"PHPDoc return type %s is not compatible with native return type %s.", doc.ReturnType, nativeReturn,
			)))
		}
		return
	}
	effectiveReturn := collapsePHPDocConditionalType(expandedReturn, nativeReturn)
	appendPHPDocTypeIssues(filename, declaration, effectiveReturn, templates, ft, ctx, issues)
	if nativeReturn != "" && !phpDocUsesTemplate(effectiveReturn, templates) && !phpDocTypeFitsNative(effectiveReturn, nativeReturn, ft, ctx) {
		*issues = append(*issues, issueSpan(filename, declaration, level2PHPDocReturnTypeCode, fmt.Sprintf(
			"PHPDoc return type %s is not compatible with native return type %s.", doc.ReturnType, nativeReturn,
		)))
	}
}

func appendTemplateVarianceIssue(filename string, declaration ast.Node, class *ast.ClassNode, doc *ast.PHPDocNode, raw string, position GenericVariance, description string, ft FileTypeContext, ctx *AnalysisContext, issues *[]AnalysisIssue) {
	if class == nil || class.PHPDoc == nil {
		return
	}
	for _, template := range class.PHPDoc.Templates {
		shadowed := false
		if doc != nil {
			for _, local := range doc.Templates {
				if asciiLowerIdent(local.Name) == asciiLowerIdent(template.Name) {
					shadowed = true
					break
				}
			}
		}
		if shadowed {
			continue
		}
		variance := GenericInvariant
		switch template.Variance {
		case string(GenericCovariant):
			variance = GenericCovariant
		case string(GenericContravariant):
			variance = GenericContravariant
		default:
			continue
		}
		if !phpDocTemplateHasVarianceConflict(raw, template.Name, variance, position, ft, ctx) {
			continue
		}
		*issues = append(*issues, issueSpan(filename, declaration, level2PHPDocTemplateVarianceCode, fmt.Sprintf(
			"Template type %s is declared %s but occurs in %s.", template.Name, variance, description,
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
	var classDoc *ast.PHPDocNode
	if class != nil {
		classDoc = class.PHPDoc
	}
	effectiveType := expandPHPDocTypeAliases(documented, phpDocTypeAliasBindings(classDoc, property.PHPDoc))
	appendPHPDocTypeIssues(filename, property, effectiveType, templates, ft, ctx, issues)
	nativeHint := ast.TypeText(property.TypeHint)
	if nativeHint == "" || phpDocUsesTemplate(effectiveType, templates) || phpDocTypeFitsNative(effectiveType, nativeHint, ft, ctx) {
		return
	}
	*issues = append(*issues, issueSpan(filename, property, level2PHPDocPropertyTypeCode, fmt.Sprintf(
		"PHPDoc type %s for property $%s is not compatible with native type %s.", documented, property.Name, nativeHint,
	)))
}

func appendPHPDocTypeIssues(filename string, declaration ast.Node, raw string, templates map[string]struct{}, ft FileTypeContext, ctx *AnalysisContext, issues *[]AnalysisIssue) {
	if ctx == nil || ctx.Resolver == nil {
		return
	}
	raw = stripBalancedOuterTypeParens(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), "?")))
	if raw == "" {
		return
	}
	if parts := splitTopLevelTypes(raw, '|'); len(parts) > 1 {
		for _, part := range parts {
			appendPHPDocTypeIssues(filename, declaration, part, templates, ft, ctx, issues)
		}
		return
	}
	if parts := splitTopLevelTypes(raw, '&'); len(parts) > 1 {
		for _, part := range parts {
			appendPHPDocTypeIssues(filename, declaration, part, templates, ft, ctx, issues)
		}
		return
	}
	if params, returnType, ok := phpDocCallableSignature(raw); ok {
		base := strings.TrimSpace(raw[:strings.Index(raw, "(")])
		if !strings.EqualFold(base, "callable") {
			appendUnknownPHPDocClass(filename, declaration, ft.resolveClassLike(base), templates, ctx, issues)
		}
		for _, param := range params {
			appendPHPDocTypeIssues(filename, declaration, param, templates, ft, ctx, issues)
		}
		appendPHPDocTypeIssues(filename, declaration, returnType, templates, ft, ctx, issues)
		return
	}
	if instance, ok := parseExactGenericTypeFromString(raw); ok && len(instance.TypeArguments) == 1 {
		base := asciiLowerIdent(strings.TrimSpace(instance.ClassName))
		if base == "key-of" || base == "value-of" {
			appendPHPDocTypeIssues(filename, declaration, instance.TypeArguments[0], templates, ft, ctx, issues)
			return
		}
	}
	if instance, ok := parseExactGenericTypeFromString(raw); ok {
		appendPHPDocGenericBaseIssues(filename, declaration, instance, templates, ft, ctx, issues)
		for _, argument := range instance.TypeArguments {
			appendPHPDocTypeIssues(filename, declaration, argument, templates, ft, ctx, issues)
		}
		return
	}
	if body, ok := arrayShapeBody(raw); ok {
		for _, entry := range splitTopLevelTypes(body, ',') {
			_, value, valid := splitArrayShapeEntry(entry)
			if valid {
				appendPHPDocTypeIssues(filename, declaration, value, templates, ft, ctx, issues)
			}
		}
		return
	}
	for _, name := range referencedClassTypes(raw, ft) {
		appendUnknownPHPDocClass(filename, declaration, name, templates, ctx, issues)
	}
}

func appendPHPDocGenericBaseIssues(filename string, declaration ast.Node, instance GenericInstance, templates map[string]struct{}, ft FileTypeContext, ctx *AnalysisContext, issues *[]AnalysisIssue) {
	baseType := ParseType(instance.ClassName)
	if baseType.IsEmpty() || !baseType.hasClassAtom() {
		return
	}
	name := ft.resolveClassLike(instance.ClassName)
	if !appendUnknownPHPDocClass(filename, declaration, name, templates, ctx, issues) {
		return
	}
	resolved, ok := ctx.Resolver.ResolveClass(name)
	if !ok {
		return
	}
	want, got := len(resolved.TemplateParams), len(instance.TypeArguments)
	switch {
	case want == 0:
		*issues = append(*issues, issueSpan(filename, declaration, level2PHPDocNotGenericCode, fmt.Sprintf("Class %s is not generic.", name)))
	case got < want:
		*issues = append(*issues, issueSpan(filename, declaration, level2PHPDocGenericLessCode, fmt.Sprintf("Generic class %s requires %d type arguments, %d given.", name, want, got)))
	case got > want:
		*issues = append(*issues, issueSpan(filename, declaration, level2PHPDocGenericMoreCode, fmt.Sprintf("Generic class %s requires %d type arguments, %d given.", name, want, got)))
	default:
		for index, argument := range instance.TypeArguments {
			if index >= len(resolved.TemplateBounds) || resolved.TemplateBounds[index] == "" || phpDocUsesTemplate(argument, templates) {
				continue
			}
			bound := ParseType(resolved.TemplateBounds[index])
			if !bound.IsEmpty() && !phpDocTypeIsSubtype(argument, resolved.TemplateBounds[index], ft, ctx) {
				*issues = append(*issues, issueSpan(filename, declaration, level2PHPDocGenericBoundCode, fmt.Sprintf(
					"Type argument %s is not a subtype of template bound %s for %s.", argument, resolved.TemplateBounds[index], name,
				)))
			}
		}
	}
}

func phpDocCallableSignature(raw string) ([]string, string, bool) {
	raw = strings.TrimSpace(raw)
	open := strings.Index(raw, "(")
	if open < 0 {
		return nil, "", false
	}
	base := strings.TrimSpace(raw[:open])
	if !strings.EqualFold(base, "callable") && !strings.EqualFold(strings.TrimPrefix(base, `\`), "Closure") {
		return nil, "", false
	}
	depth, closeIndex := 0, -1
	for index, char := range raw[open:] {
		switch char {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				closeIndex = open + index
			}
		}
		if closeIndex >= 0 {
			break
		}
	}
	if closeIndex < 0 {
		return nil, "", false
	}
	suffix := strings.TrimSpace(raw[closeIndex+1:])
	if !strings.HasPrefix(suffix, ":") {
		return nil, "", false
	}
	returnType := strings.TrimSpace(strings.TrimPrefix(suffix, ":"))
	if returnType == "" {
		return nil, "", false
	}
	body := strings.TrimSpace(raw[open+1 : closeIndex])
	if body == "" {
		return nil, returnType, true
	}
	parts := splitTopLevelTypes(body, ',')
	params := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		part = strings.TrimPrefix(part, "...")
		part = strings.TrimPrefix(part, "&")
		if variable := strings.Index(part, "$"); variable >= 0 {
			part = strings.TrimSpace(part[:variable])
		}
		part = strings.TrimSpace(strings.TrimSuffix(part, "..."))
		part = strings.TrimSpace(strings.TrimSuffix(part, "&"))
		part = strings.TrimSpace(strings.TrimSuffix(part, "="))
		if part != "" {
			params = append(params, part)
		}
	}
	return params, returnType, true
}

func resolvedCallableParamTypes(raw string, typeCtx FileTypeContext, templates map[string]struct{}) []ResolvedParam {
	paramTypes, _, ok := phpDocCallableSignature(raw)
	if !ok || len(paramTypes) == 0 {
		return nil
	}
	params := make([]ResolvedParam, 0, len(paramTypes))
	for index, paramType := range paramTypes {
		params = append(params, ResolvedParam{
			Name: fmt.Sprintf("arg%d", index+1),
			Type: normalizeTemplateAwareType(paramType, typeCtx, templates),
		})
	}
	return params
}

// appendUnknownPHPDocClass returns true when the class is known (or is a
// special/template name), allowing callers to perform additional checks.
func appendUnknownPHPDocClass(filename string, declaration ast.Node, name string, templates map[string]struct{}, ctx *AnalysisContext, issues *[]AnalysisIssue) bool {
	if isSpecialClassName(name) {
		return true
	}
	templateName := strings.TrimPrefix(name, `\`)
	if separator := strings.LastIndex(templateName, `\`); separator >= 0 {
		templateName = templateName[separator+1:]
	}
	if _, template := templates[asciiLowerIdent(templateName)]; template {
		return true
	}
	if _, ok := ctx.Resolver.ResolveClass(name); ok {
		return true
	}
	*issues = append(*issues, issueSpan(filename, declaration, level2PHPDocClassCode, fmt.Sprintf("PHPDoc references unknown class %s.", name)))
	return false
}

func phpDocTypeFitsNative(documented, native string, ft FileTypeContext, ctx *AnalysisContext) bool {
	documentedType := ParseType(normalizeTypeWithContext(erasePHPDocGenericArguments(documented), ft))
	nativeType := ParseType(normalizeTypeWithContext(native, ft))
	if documentedType.IsEmpty() || nativeType.IsEmpty() {
		return true
	}
	return nativeType.AcceptsWithContext(documentedType, nil, ctx)
}

func erasePHPDocGenericArguments(raw string) string {
	raw = stripBalancedOuterTypeParens(strings.TrimSpace(raw))
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "?") {
		return "?" + erasePHPDocGenericArguments(strings.TrimSpace(strings.TrimPrefix(raw, "?")))
	}
	if parts := splitTopLevelTypes(raw, '|'); len(parts) > 1 {
		for index, part := range parts {
			parts[index] = erasePHPDocGenericArguments(part)
		}
		return strings.Join(parts, "|")
	}
	if parts := splitTopLevelTypes(raw, '&'); len(parts) > 1 {
		for index, part := range parts {
			parts[index] = erasePHPDocGenericArguments(part)
		}
		return strings.Join(parts, "&")
	}
	if _, _, ok := phpDocCallableSignature(raw); ok {
		// Native compatibility concerns the callable object itself; signature
		// parameter and return types are validated separately above.
		return strings.TrimSpace(raw[:strings.Index(raw, "(")])
	}
	if instance, ok := parseExactGenericTypeFromString(raw); ok {
		return instance.ClassName
	}
	return raw
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

func collectPHPDocTypeAliases(nodes []ast.Node) map[string]struct{} {
	var aliases map[string]struct{}
	walkAllWithoutTypeContext(nodes, func(node ast.Node) {
		collectPHPDocAliasNamesOnNode(node, &aliases)
	})
	return aliases
}

func collectPHPDocAliasNamesOnNode(node ast.Node, aliases *map[string]struct{}) {
	switch n := node.(type) {
	case *ast.ClassNode:
		collectPHPDocAliasNames(n.PHPDoc, aliases)
	case *ast.FunctionNode:
		collectPHPDocAliasNames(n.PHPDoc, aliases)
	case *ast.PropertyNode:
		collectPHPDocAliasNames(n.PHPDoc, aliases)
	}
}

func collectPHPDocAliasNames(doc *ast.PHPDocNode, aliases *map[string]struct{}) {
	if doc == nil {
		return
	}
	for _, alias := range doc.TypeAliases {
		if strings.TrimSpace(alias.Name) == "" {
			continue
		}
		if *aliases == nil {
			*aliases = make(map[string]struct{})
		}
		(*aliases)[asciiLowerIdent(alias.Name)] = struct{}{}
	}
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
		{level2PHPDocGenericLessCode},
		{level2PHPDocGenericMoreCode},
		{level2PHPDocGenericBoundCode},
		{level2PHPDocMethodVarianceCode},
		{level2PHPDocTemplateVarianceCode},
		{level2PHPDocNotGenericCode},
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
