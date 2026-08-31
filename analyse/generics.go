package analyse

import (
	"strconv"
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

func parseExactGenericTypeFromString(typeStr string) (GenericInstance, bool) {
	typeStr = strings.TrimSpace(typeStr)
	instance, ok := parseGenericTypeFromString(typeStr)
	if !ok {
		return GenericInstance{}, false
	}
	closeIdx := strings.LastIndex(typeStr, ">")
	if closeIdx < 0 || strings.TrimSpace(typeStr[closeIdx+1:]) != "" {
		return GenericInstance{}, false
	}
	return instance, true
}

type arrayShapeField struct {
	callable Type
	nested   map[string]arrayShapeField
	typ      Type
}

func (f arrayShapeField) empty() bool {
	return f.callable.IsEmpty() && len(f.nested) == 0 && f.typ.IsEmpty()
}

func arrayShapeCallableReturns(raw string, typeCtx FileTypeContext) map[string]Type {
	fields := flattenArrayShapeCallables(parseArrayShapeFields(raw, typeCtx))
	if len(fields) == 0 {
		return nil
	}
	return fields
}

func flattenArrayShapeCallables(fields map[string]arrayShapeField) map[string]Type {
	if len(fields) == 0 {
		return nil
	}
	flat := make(map[string]Type, len(fields))
	for key, field := range fields {
		if !field.callable.IsEmpty() {
			flat[key] = field.callable
		}
	}
	if len(flat) == 0 {
		return nil
	}
	return flat
}

func parseArrayShapeFields(raw string, typeCtx FileTypeContext) map[string]arrayShapeField {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	body, ok := arrayShapeBody(raw)
	if !ok {
		return nil
	}
	fields := make(map[string]arrayShapeField)
	nextIndex := 0
	for _, entry := range splitTopLevelTypes(body, ',') {
		key, value, ok := splitArrayShapeEntry(entry)
		if !ok {
			continue
		}
		if key == "" {
			key = strconv.Itoa(nextIndex)
			nextIndex++
		}
		field := arrayShapeField{callable: callableReturnType(value, typeCtx)}
		if nested := parseArrayShapeFields(value, typeCtx); len(nested) > 0 {
			field.nested = nested
		}
		if field.callable.IsEmpty() && len(field.nested) == 0 {
			if parsed := ParseType(normalizeTypeWithContext(value, typeCtx)); parsed.hasClassAtom() {
				field.typ = parsed
			}
		}
		if field.empty() {
			continue
		}
		fields[key] = field
	}
	if len(fields) == 0 {
		return nil
	}
	return fields
}

func arrayShapeBody(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "?")
	lower := strings.ToLower(raw)
	start := -1
	for _, prefix := range []string{"non-empty-array{", "non-empty-list{", "array{", "list{"} {
		if idx := strings.Index(lower, prefix); idx >= 0 {
			start = idx + len(prefix) - 1
			break
		}
	}
	if start < 0 || start >= len(raw) || raw[start] != '{' {
		return "", false
	}
	depth := 0
	for idx := start; idx < len(raw); idx++ {
		switch raw[idx] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return raw[start+1 : idx], true
			}
		}
	}
	return "", false
}

func splitArrayShapeEntry(entry string) (string, string, bool) {
	entry = strings.TrimSpace(entry)
	if entry == "" || entry == "..." {
		return "", "", false
	}
	depthAngle, depthParen, depthBrace := 0, 0, 0
	for idx, r := range entry {
		switch r {
		case '<':
			depthAngle++
		case '>':
			if depthAngle > 0 {
				depthAngle--
			}
		case '(':
			depthParen++
		case ')':
			if depthParen > 0 {
				depthParen--
			}
		case '{':
			depthBrace++
		case '}':
			if depthBrace > 0 {
				depthBrace--
			}
		case ':':
			if depthAngle == 0 && depthParen == 0 && depthBrace == 0 {
				key := normalizeArrayShapeKey(entry[:idx])
				value := strings.TrimSpace(entry[idx+1:])
				if key == "" || strings.ContainsAny(key, "()<>") {
					return "", entry, true
				}
				return key, value, value != ""
			}
		}
	}
	return "", entry, true
}

func normalizeArrayShapeKey(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimSuffix(raw, "?")
	raw = strings.TrimSpace(raw)
	if len(raw) >= 2 {
		if (raw[0] == '\'' && raw[len(raw)-1] == '\'') || (raw[0] == '"' && raw[len(raw)-1] == '"') {
			return raw[1 : len(raw)-1]
		}
	}
	return raw
}

func callableReturnType(raw string, typeCtx FileTypeContext) Type {
	raw = strings.TrimSpace(raw)
	open := strings.Index(raw, "(")
	if open < 0 || !strings.EqualFold(strings.TrimSpace(raw[:open]), "callable") {
		return EmptyType()
	}
	depth := 0
	closeIdx := -1
	for idx, r := range raw[open:] {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				closeIdx = open + idx
			}
		}
		if closeIdx >= 0 {
			break
		}
	}
	if closeIdx < 0 {
		return EmptyType()
	}
	suffix := strings.TrimSpace(raw[closeIdx+1:])
	if !strings.HasPrefix(suffix, ":") {
		return EmptyType()
	}
	returnType := strings.TrimSpace(strings.TrimPrefix(suffix, ":"))
	if returnType == "" {
		return EmptyType()
	}
	return ParseType(normalizeTypeWithContext(returnType, typeCtx))
}
