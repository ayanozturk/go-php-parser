package analyse

import "strings"

const phpDocGenericRelationDepthLimit = 32

func phpDocTemplateHasVarianceConflict(raw, templateName string, declared, position GenericVariance, ft FileTypeContext, ctx *AnalysisContext) bool {
	return phpDocTemplateHasVarianceConflictAt(raw, templateName, declared, position, ft, ctx, 0)
}

func phpDocTemplateHasVarianceConflictAt(raw, templateName string, declared, position GenericVariance, ft FileTypeContext, ctx *AnalysisContext, depth int) bool {
	if depth >= phpDocGenericRelationDepthLimit {
		return false
	}
	raw = stripBalancedOuterTypeParens(strings.TrimSpace(raw))
	if raw == "" {
		return false
	}
	if strings.HasPrefix(raw, "?") {
		return phpDocTemplateHasVarianceConflictAt(strings.TrimSpace(strings.TrimPrefix(raw, "?")), templateName, declared, position, ft, ctx, depth+1)
	}
	if strings.HasSuffix(raw, "[]") {
		return phpDocTemplateHasVarianceConflictAt(strings.TrimSpace(strings.TrimSuffix(raw, "[]")), templateName, declared, position, ft, ctx, depth+1)
	}
	for _, separator := range []rune{'|', '&'} {
		if parts := splitTopLevelTypes(raw, separator); len(parts) > 1 {
			for _, part := range parts {
				if phpDocTemplateHasVarianceConflictAt(part, templateName, declared, position, ft, ctx, depth+1) {
					return true
				}
			}
			return false
		}
	}
	if params, returnType, ok := phpDocCallableSignature(raw); ok {
		for _, param := range params {
			if phpDocTemplateHasVarianceConflictAt(param, templateName, declared, composeGenericVariance(position, GenericContravariant), ft, ctx, depth+1) {
				return true
			}
		}
		return phpDocTemplateHasVarianceConflictAt(returnType, templateName, declared, composeGenericVariance(position, GenericCovariant), ft, ctx, depth+1)
	}
	if instance, ok := parseExactGenericTypeFromString(raw); ok {
		variances := phpDocGenericArgumentVariances(instance.ClassName, len(instance.TypeArguments), ft, ctx)
		for index, argument := range instance.TypeArguments {
			inner := GenericInvariant
			if index < len(variances) {
				inner = variances[index]
			}
			if phpDocTemplateHasVarianceConflictAt(argument, templateName, declared, composeGenericVariance(position, inner), ft, ctx, depth+1) {
				return true
			}
		}
		return false
	}
	if strings.EqualFold(strings.TrimSpace(raw), templateName) {
		return position != declared
	}
	return false
}

func phpDocGenericArgumentVariances(className string, count int, ft FileTypeContext, ctx *AnalysisContext) []GenericVariance {
	lower := asciiLowerIdent(strings.TrimPrefix(strings.TrimSpace(className), `\`))
	if lower == "array" || lower == "list" || lower == "non-empty-array" || lower == "non-empty-list" || lower == "iterable" || lower == "class-string" || lower == "interface-string" || lower == "trait-string" {
		variances := make([]GenericVariance, count)
		for index := range variances {
			variances[index] = GenericCovariant
		}
		return variances
	}
	if ctx == nil || ctx.Resolver == nil {
		return nil
	}
	class, ok := ctx.Resolver.ResolveClass(normalizeTypeWithContext(className, ft))
	if !ok {
		return nil
	}
	variances := make([]GenericVariance, count)
	for index := range variances {
		variances[index] = GenericInvariant
		if index < len(class.TemplateVariances) && class.TemplateVariances[index] != "" {
			variances[index] = class.TemplateVariances[index]
		}
	}
	return variances
}

func composeGenericVariance(outer, inner GenericVariance) GenericVariance {
	if outer == GenericInvariant || inner == GenericInvariant {
		return GenericInvariant
	}
	if inner == GenericCovariant {
		return outer
	}
	if outer == GenericCovariant {
		return GenericContravariant
	}
	return GenericCovariant
}

func phpDocTypeIsSubtype(actual, expected string, ft FileTypeContext, ctx *AnalysisContext) bool {
	return phpDocTypeIsSubtypeAt(actual, expected, ft, ctx, 0, make(map[string]struct{}))
}

func phpDocTypeIsSubtypeAt(actual, expected string, ft FileTypeContext, ctx *AnalysisContext, depth int, seen map[string]struct{}) bool {
	if depth >= phpDocGenericRelationDepthLimit {
		return false
	}
	actual = normalizeTemplateAwareType(actual, ft, nil)
	expected = normalizeTemplateAwareType(expected, ft, nil)
	actual = stripBalancedOuterTypeParens(strings.TrimSpace(actual))
	expected = stripBalancedOuterTypeParens(strings.TrimSpace(expected))
	if actual == "" || expected == "" || strings.EqualFold(actual, expected) || strings.EqualFold(expected, "mixed") {
		return true
	}
	if strings.EqualFold(actual, "mixed") {
		return false
	}
	if actualParts := splitTopLevelTypes(actual, '|'); len(actualParts) > 1 {
		for _, part := range actualParts {
			if !phpDocTypeIsSubtypeAt(part, expected, ft, ctx, depth+1, seen) {
				return false
			}
		}
		return true
	}
	if expectedParts := splitTopLevelTypes(expected, '|'); len(expectedParts) > 1 {
		for _, part := range expectedParts {
			if phpDocTypeIsSubtypeAt(actual, part, ft, ctx, depth+1, seen) {
				return true
			}
		}
		return false
	}
	if expectedParts := splitTopLevelTypes(expected, '&'); len(expectedParts) > 1 {
		for _, part := range expectedParts {
			if !phpDocTypeIsSubtypeAt(actual, part, ft, ctx, depth+1, seen) {
				return false
			}
		}
		return true
	}
	if actualParts := splitTopLevelTypes(actual, '&'); len(actualParts) > 1 {
		for _, part := range actualParts {
			if phpDocTypeIsSubtypeAt(part, expected, ft, ctx, depth+1, seen) {
				return true
			}
		}
		return false
	}

	actualGeneric, actualIsGeneric := parseExactGenericTypeFromString(actual)
	expectedGeneric, expectedIsGeneric := parseExactGenericTypeFromString(expected)
	if expectedIsGeneric {
		if actualIsGeneric && samePHPDocClass(actualGeneric.ClassName, expectedGeneric.ClassName, ft, ctx) {
			return phpDocGenericArgumentsAreSubtypes(actualGeneric.TypeArguments, expectedGeneric.TypeArguments, expectedGeneric.ClassName, ft, ctx, depth, seen)
		}
		candidate := actualGeneric
		if !actualIsGeneric {
			candidate = GenericInstance{ClassName: actual}
		}
		if inherited, ok := inheritedPHPDocGeneric(candidate, expectedGeneric.ClassName, ft, ctx, depth, seen); ok {
			return phpDocGenericArgumentsAreSubtypes(inherited.TypeArguments, expectedGeneric.TypeArguments, expectedGeneric.ClassName, ft, ctx, depth, seen)
		}
		return false
	}
	actualBase := actual
	if actualIsGeneric {
		actualBase = actualGeneric.ClassName
	}
	expectedBase := expected
	if expectedIsGeneric {
		expectedBase = expectedGeneric.ClassName
	}
	return ParseType(normalizeTypeWithContext(expectedBase, ft)).AcceptsWithContext(ParseType(normalizeTypeWithContext(actualBase, ft)), nil, ctx)
}

func phpDocGenericArgumentsAreSubtypes(actual, expected []string, className string, ft FileTypeContext, ctx *AnalysisContext, depth int, seen map[string]struct{}) bool {
	if len(actual) != len(expected) {
		return false
	}
	variances := phpDocGenericArgumentVariances(className, len(expected), ft, ctx)
	for index := range expected {
		variance := GenericInvariant
		if index < len(variances) {
			variance = variances[index]
		}
		switch variance {
		case GenericCovariant:
			if !phpDocTypeIsSubtypeAt(actual[index], expected[index], ft, ctx, depth+1, seen) {
				return false
			}
		case GenericContravariant:
			if !phpDocTypeIsSubtypeAt(expected[index], actual[index], ft, ctx, depth+1, seen) {
				return false
			}
		default:
			if !phpDocTypeIsSubtypeAt(actual[index], expected[index], ft, ctx, depth+1, seen) || !phpDocTypeIsSubtypeAt(expected[index], actual[index], ft, ctx, depth+1, seen) {
				return false
			}
		}
	}
	return true
}

func inheritedPHPDocGeneric(actual GenericInstance, targetName string, ft FileTypeContext, ctx *AnalysisContext, depth int, seen map[string]struct{}) (GenericInstance, bool) {
	if ctx == nil || ctx.Resolver == nil || depth >= phpDocGenericRelationDepthLimit {
		return GenericInstance{}, false
	}
	class, ok := ctx.Resolver.ResolveClass(actual.ClassName)
	if !ok || len(class.GenericParents) == 0 {
		return GenericInstance{}, false
	}
	key := phpDocGenericInstanceKey(actual, ft)
	if _, exists := seen[key]; exists {
		return GenericInstance{}, false
	}
	seen[key] = struct{}{}
	defer delete(seen, key)
	bindings := make(map[string]string, len(class.TemplateParams))
	for index, name := range class.TemplateParams {
		if index < len(actual.TypeArguments) {
			bindings[name] = actual.TypeArguments[index]
		}
	}
	for _, parent := range class.GenericParents {
		parentInstance := GenericInstance{ClassName: parent.Name}
		for _, argument := range parent.TypeArguments {
			parentInstance.TypeArguments = append(parentInstance.TypeArguments, ApplyTemplateBindings(argument, bindings))
		}
		if samePHPDocClass(parentInstance.ClassName, targetName, ft, ctx) {
			return parentInstance, true
		}
		if inherited, found := inheritedPHPDocGeneric(parentInstance, targetName, ft, ctx, depth+1, seen); found {
			return inherited, true
		}
	}
	return GenericInstance{}, false
}

func samePHPDocClass(left, right string, ft FileTypeContext, ctx *AnalysisContext) bool {
	leftName := normalizeTypeWithContext(left, ft)
	rightName := normalizeTypeWithContext(right, ft)
	if strings.EqualFold(strings.TrimPrefix(leftName, `\`), strings.TrimPrefix(rightName, `\`)) {
		return true
	}
	if ctx != nil && ctx.Resolver != nil {
		leftClass, leftOK := ctx.Resolver.ResolveClass(leftName)
		rightClass, rightOK := ctx.Resolver.ResolveClass(rightName)
		return leftOK && rightOK && strings.EqualFold(leftClass.Name, rightClass.Name)
	}
	return false
}

func phpDocGenericInstanceKey(instance GenericInstance, ft FileTypeContext) string {
	return strings.ToLower(normalizeTypeWithContext(instance.ClassName, ft) + "<" + strings.Join(instance.TypeArguments, ",") + ">")
}
