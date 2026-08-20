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

func normalizeTemplateAwareType(raw string, ctx fileTypeContext, templates map[string]struct{}) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	prefix := ""
	if strings.HasPrefix(raw, "?") {
		prefix = "?"
		raw = strings.TrimSpace(strings.TrimPrefix(raw, "?"))
	}
	parts := splitTopLevelTypes(raw, '|')
	for i, part := range parts {
		intersections := splitTopLevelTypes(part, '&')
		for j, atom := range intersections {
			atom = strings.TrimSpace(atom)
			if _, ok := templates[atom]; ok {
				intersections[j] = atom
				continue
			}
			intersections[j] = normalizeTypeWithContext(atom, ctx)
		}
		parts[i] = strings.Join(intersections, "&")
	}
	return prefix + strings.Join(parts, "|")
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
