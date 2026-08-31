package analyse

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// ApplyTemplateBindings substitutes template identifiers in a PHPDoc type.
// Identifiers are replaced token-by-token so T does not alter names such as
// Type or App\TEntity.
func ApplyTemplateBindings(raw string, bindings map[string]string) string {
	if raw == "" || len(bindings) == 0 {
		return raw
	}
	var out strings.Builder
	for start := 0; start < len(raw); {
		r, size := utf8.DecodeRuneInString(raw[start:])
		if !isTemplateIdentifierRune(r) {
			out.WriteString(raw[start : start+size])
			start += size
			continue
		}
		end := start + size
		for end < len(raw) {
			next, nextSize := utf8.DecodeRuneInString(raw[end:])
			if !isTemplateIdentifierRune(next) {
				break
			}
			end += nextSize
		}
		token := raw[start:end]
		if replacement, ok := bindings[token]; ok {
			out.WriteString(replacement)
		} else {
			out.WriteString(token)
		}
		start = end
	}
	return out.String()
}

func isTemplateIdentifierRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '\\'
}

func templateNames(docTemplates []string) map[string]struct{} {
	names := make(map[string]struct{}, len(docTemplates))
	for _, name := range docTemplates {
		names[name] = struct{}{}
	}
	return names
}

func normalizeTemplateAwareType(raw string, ctx FileTypeContext, templates map[string]struct{}) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	prefix := ""
	if strings.HasPrefix(raw, "?") {
		prefix = "?"
		raw = strings.TrimSpace(strings.TrimPrefix(raw, "?"))
	}
	return prefix + normalizeTemplateAwareTypeExpression(raw, ctx, templates)
}

func normalizeTemplateAwareTypeExpression(raw string, ctx FileTypeContext, templates map[string]struct{}) string {
	raw = stripBalancedOuterTypeParens(strings.TrimSpace(raw))
	if raw == "" {
		return ""
	}
	if parts := splitTopLevelTypes(raw, '|'); len(parts) > 1 {
		for idx, part := range parts {
			normalized := normalizeTemplateAwareTypeExpression(part, ctx, templates)
			if len(splitTopLevelTypes(normalized, '&')) > 1 {
				normalized = "(" + normalized + ")"
			}
			parts[idx] = normalized
		}
		return strings.Join(parts, "|")
	}
	if parts := splitTopLevelTypes(raw, '&'); len(parts) > 1 {
		for idx, part := range parts {
			parts[idx] = normalizeTemplateAwareTypeExpression(part, ctx, templates)
		}
		return strings.Join(parts, "&")
	}
	if _, ok := templates[raw]; ok {
		return raw
	}
	return normalizeTypeWithContext(raw, ctx)
}

func bindGenericParent(parent ResolvedClass, relation ResolvedGenericParent, current map[string]string) map[string]string {
	if len(parent.TemplateParams) == 0 || len(relation.TypeArguments) == 0 {
		return nil
	}
	bindings := make(map[string]string, len(parent.TemplateParams))
	for i, name := range parent.TemplateParams {
		if i >= len(relation.TypeArguments) {
			break
		}
		bindings[name] = ApplyTemplateBindings(relation.TypeArguments[i], current)
	}
	return bindings
}

func genericRelationTo(class ResolvedClass, parentName string) (ResolvedGenericParent, bool) {
	for _, relation := range class.GenericParents {
		if strings.EqualFold(strings.TrimPrefix(relation.Name, `\`), strings.TrimPrefix(parentName, `\`)) {
			return relation, true
		}
	}
	return ResolvedGenericParent{}, false
}

// parseGenericTypeFromString extracts class name and type arguments from strings like "Collection<User>"
// Returns (GenericInstance, true) if the string contains type arguments, (empty, false) otherwise.
func parseGenericTypeFromString(typeStr string) (GenericInstance, bool) {
	typeStr = strings.TrimSpace(typeStr)
	if !strings.Contains(typeStr, "<") || !strings.Contains(typeStr, ">") {
		return GenericInstance{}, false
	}

	openIdx := strings.Index(typeStr, "<")
	closeIdx := strings.LastIndex(typeStr, ">")
	if openIdx < 0 || closeIdx <= openIdx {
		return GenericInstance{}, false
	}

	className := strings.TrimSpace(typeStr[:openIdx])
	if className == "" {
		return GenericInstance{}, false
	}

	argsStr := typeStr[openIdx+1 : closeIdx]
	args := splitTopLevelTypes(argsStr, ',')
	var typeArgs []string
	for _, arg := range args {
		arg = strings.TrimSpace(arg)
		if arg != "" {
			typeArgs = append(typeArgs, arg)
		}
	}

	if len(typeArgs) == 0 {
		return GenericInstance{}, false
	}

	return GenericInstance{ClassName: className, TypeArguments: typeArgs}, true
}
