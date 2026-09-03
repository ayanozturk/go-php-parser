package analyse

import (
	"fmt"
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
)

const (
	level6MissingGenericTypeCode  = "Level6.MissingGenericType"
	level6MissingIterableTypeCode = "Level6.MissingIterableValueType"
	level6MissingParameterCode    = "Level6.MissingParameterType"
	level6MissingPropertyCode     = "Level6.MissingPropertyType"
	level6MissingReturnCode       = "Level6.MissingReturnType"
)

func missingTypeIssuesForFile(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
	return ensureStructuralIssues(filename, nodes, ctx).missingTypeIssues
}

func appendMissingTypeIssuesOnNode(filename string, node ast.Node, class *ast.ClassNode, ft FileTypeContext, ctx *AnalysisContext, issues *[]AnalysisIssue) {
	switch n := node.(type) {
	case *ast.FunctionNode:
		if n.Name != "" {
			isMethod := class != nil || n.Visibility != "" || len(n.Modifiers) > 0
			appendCallableMissingTypeIssues(filename, n, n.Name, isMethod, n.Params, n.ReturnType, n.PHPDoc, class, ft, ctx, issues)
		}
	case *ast.InterfaceMethodNode:
		returnType := ""
		if n.ReturnType != nil {
			returnType = n.ReturnType.TokenLiteral()
		}
		appendCallableMissingTypeIssues(filename, n, n.Name, true, n.Params, returnType, n.PHPDoc, class, ft, ctx, issues)
	case *ast.PropertyNode:
		raw := n.TypeHint
		if n.PHPDoc != nil && n.PHPDoc.VarType != "" {
			raw = n.PHPDoc.VarType
		}
		if raw == "" {
			*issues = append(*issues, issueSpan(filename, n, level6MissingPropertyCode, fmt.Sprintf("Property $%s has no type specified.", n.Name)))
			return
		}
		appendMissingTypeIssues(filename, n, raw, ft, ctx, issues)
	}
}

func appendCallableMissingTypeIssues(filename string, declaration ast.Node, name string, isMethod bool, params []ast.Node, nativeReturn string, doc *ast.PHPDocNode, class *ast.ClassNode, ft FileTypeContext, ctx *AnalysisContext, issues *[]AnalysisIssue) {
	var inherited []ResolvedMethod
	inheritedLoaded := false
	loadInherited := func() []ResolvedMethod {
		if !inheritedLoaded {
			inherited = inheritedMethodContracts(class, name, ft, ctx)
			inheritedLoaded = true
		}
		return inherited
	}
	for paramIndex, paramNode := range params {
		param, ok := paramNode.(*ast.ParamNode)
		if !ok {
			continue
		}
		raw := paramTypeName(param)
		hasDocumentedType := false
		if doc != nil {
			if documented, found := phpDocDocumentedParameter(doc, param.Name); found {
				raw = documented.Type
				hasDocumentedType = true
			}
		}
		if !hasDocumentedType && typeHasMissingDeclaration(raw, ft, ctx) {
			for _, contract := range loadInherited() {
				if paramIndex < len(contract.Params) && typeSuppliesMissingDeclarationDetail(contract.Params[paramIndex].Type) {
					raw = contract.Params[paramIndex].Type
					break
				}
			}
		}
		if raw == "" {
			*issues = append(*issues, issueSpan(filename, param, level6MissingParameterCode, fmt.Sprintf("Parameter $%s has no type specified.", param.Name)))
			continue
		}
		appendMissingTypeIssues(filename, param, raw, ft, ctx, issues)
	}
	rawReturn := nativeReturn
	hasDocumentedReturn := doc != nil && doc.ReturnType != ""
	if hasDocumentedReturn {
		rawReturn = doc.ReturnType
	} else if typeHasMissingDeclaration(rawReturn, ft, ctx) && !(rawReturn == "" && isMethod && isReturnTypeExemptMethod(name)) {
		for _, contract := range loadInherited() {
			if typeSuppliesMissingDeclarationDetail(contract.ReturnType) {
				rawReturn = contract.ReturnType
				break
			}
		}
	}
	if rawReturn == "" && !(isMethod && isReturnTypeExemptMethod(name)) {
		*issues = append(*issues, issueSpan(filename, declaration, level6MissingReturnCode, fmt.Sprintf("Function or method %s has no return type specified.", name)))
		return
	}
	appendMissingTypeIssues(filename, declaration, rawReturn, ft, ctx, issues)
}

func inheritedMethodContracts(class *ast.ClassNode, methodName string, ft FileTypeContext, ctx *AnalysisContext) []ResolvedMethod {
	if class == nil || ctx == nil || ctx.Resolver == nil || methodName == "" {
		return nil
	}
	resolved, ok := ctx.Resolver.ResolveClass(ft.resolveClassLike(class.Name))
	if !ok {
		return nil
	}
	queue := append(append([]string(nil), resolved.Extends...), resolved.Implements...)
	seen := make(map[string]struct{}, len(queue))
	var contracts []ResolvedMethod
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		key := indexKey(name)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		if method, found := ctx.Resolver.ResolveMethod(name, methodName); found {
			contracts = append(contracts, method)
		}
		if parent, found := ctx.Resolver.ResolveClass(name); found {
			queue = append(queue, parent.Extends...)
			queue = append(queue, parent.Implements...)
		}
	}
	return contracts
}

func typeSuppliesMissingDeclarationDetail(raw string) bool {
	raw = stripBalancedOuterTypeParens(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), "?")))
	if raw == "" {
		return false
	}
	if parts := splitTopLevelTypes(raw, '|'); len(parts) > 1 {
		for _, part := range parts {
			if typeSuppliesMissingDeclarationDetail(part) {
				return true
			}
		}
		return false
	}
	if parts := splitTopLevelTypes(raw, '&'); len(parts) > 1 {
		for _, part := range parts {
			if typeSuppliesMissingDeclarationDetail(part) {
				return true
			}
		}
		return false
	}
	if _, ok := parseExactGenericTypeFromString(raw); ok {
		return true
	}
	if _, ok := arrayShapeBody(raw); ok {
		return true
	}
	return strings.HasSuffix(raw, "[]")
}

func typeHasMissingDeclaration(raw string, ft FileTypeContext, ctx *AnalysisContext) bool {
	raw = stripBalancedOuterTypeParens(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), "?")))
	if raw == "" {
		return true
	}
	for _, delimiter := range []rune{'|', '&'} {
		if parts := splitTopLevelTypes(raw, delimiter); len(parts) > 1 {
			for _, part := range parts {
				if typeHasMissingDeclaration(part, ft, ctx) {
					return true
				}
			}
			return false
		}
	}
	if instance, ok := parseExactGenericTypeFromString(raw); ok {
		for _, argument := range instance.TypeArguments {
			if typeHasMissingDeclaration(argument, ft, ctx) {
				return true
			}
		}
		return false
	}
	if params, returnType, ok := phpDocCallableSignature(raw); ok {
		for _, param := range params {
			if typeHasMissingDeclaration(param, ft, ctx) {
				return true
			}
		}
		return typeHasMissingDeclaration(returnType, ft, ctx)
	}
	if body, ok := arrayShapeBody(raw); ok {
		for _, entry := range splitTopLevelTypes(body, ',') {
			_, value, valid := splitArrayShapeEntry(entry)
			if valid && typeHasMissingDeclaration(value, ft, ctx) {
				return true
			}
		}
		return false
	}
	if strings.HasSuffix(raw, "[]") {
		return typeHasMissingDeclaration(strings.TrimSpace(strings.TrimSuffix(raw, "[]")), ft, ctx)
	}
	switch asciiLowerIdent(raw) {
	case "array", "iterable", "list", "non-empty-array", "non-empty-list":
		return true
	}
	if ctx == nil || ctx.Resolver == nil {
		return false
	}
	resolved, ok := ctx.Resolver.ResolveClass(ft.resolveClassLike(raw))
	return ok && len(resolved.TemplateParams) > 0
}

func isReturnTypeExemptMethod(name string) bool {
	return strings.EqualFold(name, "__construct") || strings.EqualFold(name, "__destruct")
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
	for _, code := range []string{
		level6MissingGenericTypeCode,
		level6MissingIterableTypeCode,
		level6MissingParameterCode,
		level6MissingPropertyCode,
		level6MissingReturnCode,
	} {
		code := code
		RegisterAnalysisRuleWithLevel(code, 6, "level6", func(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
			return filterIssuesByCode(missingTypeIssuesForFile(filename, nodes, ctx), code)
		})
	}
}
