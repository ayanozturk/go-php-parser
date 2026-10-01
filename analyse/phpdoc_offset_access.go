package analyse

import (
	"sort"
	"strconv"
	"strings"
)

func resolvePHPDocIndexedTypes(raw string, typeCtx FileTypeContext) (string, bool) {
	if projected, ok := resolvePHPDocKeyValueProjection(raw, typeCtx); ok {
		return projected, true
	}
	return resolvePHPDocOffsetAccess(raw, typeCtx)
}

// resolvePHPDocOffsetAccess resolves PHPDoc offset access after aliases and
// template bindings have been applied. Unresolved template expressions are
// left intact so a later call-site substitution can resolve them.
func resolvePHPDocOffsetAccess(raw string, typeCtx FileTypeContext) (string, bool) {
	raw = stripBalancedOuterTypeParens(strings.TrimSpace(raw))
	parts := splitTopLevelTypes(raw, '|')
	if len(parts) > 1 {
		resolved := make([]string, 0, len(parts))
		for _, part := range parts {
			value, ok := resolvePHPDocIndexedTypes(part, typeCtx)
			if !ok {
				return "", false
			}
			resolved = append(resolved, value)
		}
		return ParseType(strings.Join(resolved, "| ")).String(), true
	}

	base, key, ok := splitPHPDocOffsetAccess(raw)
	if !ok {
		return "", false
	}
	resolvedBase := base
	if projected, ok := resolvePHPDocIndexedTypes(base, typeCtx); ok {
		resolvedBase = projected
	}
	if fields := parseArrayShapeFields(resolvedBase, typeCtx); len(fields) > 0 {
		keys := splitTopLevelTypes(key, '|')
		var result []string
		for _, candidate := range keys {
			field, exists := fields[normalizeArrayShapeKey(candidate)]
			if !exists {
				return "", false
			}
			fieldType := arrayShapeFieldType(field)
			if fieldType == "" {
				return "", false
			}
			result = append(result, fieldType)
		}
		if len(result) > 0 {
			return strings.Join(result, "|"), true
		}
		return "", false
	}

	instance, generic := parseExactGenericTypeFromString(resolvedBase)
	if generic {
		if _, value, iterable := iterableTypesFromGeneric(instance); iterable && !value.IsEmpty() {
			return value.String(), true
		}
	}
	return "", false
}

func arrayShapeFieldType(field arrayShapeField) string {
	if !field.typ.IsEmpty() {
		return field.typ.String()
	}
	if len(field.nested) > 0 {
		keys := make([]string, 0, len(field.nested))
		for key := range field.nested {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		entries := make([]string, 0, len(keys))
		for _, key := range keys {
			value := arrayShapeFieldType(field.nested[key])
			if value == "" {
				return ""
			}
			shapeKey := key
			if _, err := strconv.ParseInt(key, 10, 64); err != nil {
				shapeKey = "'" + strings.ReplaceAll(strings.ReplaceAll(key, `\\`, `\\\\`), "'", `\\'`) + "'"
			}
			entries = append(entries, shapeKey+": "+value)
		}
		return "array{" + strings.Join(entries, ", ") + "}"
	}
	if !field.callable.IsEmpty() {
		return "callable"
	}
	return ""
}

// resolvePHPDocKeyValueProjection resolves key-of<T> and value-of<T> for
// concrete array shapes and built-in generic iterables. Template expressions
// remain unchanged until their bindings are available.
func resolvePHPDocKeyValueProjection(raw string, typeCtx FileTypeContext) (string, bool) {
	instance, ok := parseExactGenericTypeFromString(strings.TrimSpace(raw))
	if !ok || len(instance.TypeArguments) != 1 {
		return "", false
	}
	projection := asciiLowerIdent(strings.TrimSpace(instance.ClassName))
	if projection != "key-of" && projection != "value-of" {
		return "", false
	}
	base := strings.TrimSpace(instance.TypeArguments[0])
	base = stripBalancedOuterTypeParens(base)
	parts := splitTopLevelTypes(base, '|')
	if len(parts) > 1 {
		var projected []string
		for _, part := range parts {
			value, resolved := resolvePHPDocKeyValueProjection(projection+"<"+strings.TrimSpace(part)+">", typeCtx)
			if !resolved {
				return "", false
			}
			projected = append(projected, value)
		}
		return ParseType(strings.Join(projected, "| ")).String(), true
	}
	if fields := parseArrayShapeFields(base, typeCtx); len(fields) > 0 {
		if projection == "key-of" {
			var keys Type
			for key := range fields {
				if isNumericArrayKey(key) {
					keys = unionInferredTypes(keys, ParseType(key))
				} else {
					keys = unionInferredTypes(keys, ParseType(quotePHPDocStringLiteral(key)))
				}
			}
			if !keys.IsEmpty() {
				return keys.String(), true
			}
			return "never", true
		}
		var values Type
		for _, field := range fields {
			valueType := arrayShapeFieldType(field)
			if valueType == "" {
				return "", false
			}
			values = unionInferredTypes(values, ParseType(valueType))
		}
		if !values.IsEmpty() {
			return values.String(), true
		}
		return "", false
	}
	iterable, ok := parseExactGenericTypeFromString(base)
	if !ok || !isBuiltinArrayGeneric(iterable.ClassName) {
		return "", false
	}
	key, value, ok := iterableTypesFromGeneric(iterable)
	if !ok {
		return "", false
	}
	if projection == "key-of" {
		return key.String(), !key.IsEmpty()
	}
	return value.String(), !value.IsEmpty()
}

// splitPHPDocOffsetAccess returns the base type and key from a trailing
// PHPDoc offset-access expression (for example array{foo: int}['foo']).
func splitPHPDocOffsetAccess(raw string) (base, key string, ok bool) {
	raw = strings.TrimSpace(raw)
	if len(raw) < 3 || raw[len(raw)-1] != ']' {
		return "", "", false
	}
	angle, paren, brace, square := 0, 0, 0, 0
	quote := byte(0)
	start := -1
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		if quote != 0 {
			if c == '\\' {
				i++
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '\'' || c == '"' {
			quote = c
			continue
		}
		switch c {
		case '<':
			angle++
		case '>':
			if angle > 0 {
				angle--
			}
		case '(':
			paren++
		case ')':
			if paren > 0 {
				paren--
			}
		case '{':
			brace++
		case '}':
			if brace > 0 {
				brace--
			}
		case '[':
			if angle == 0 && paren == 0 && brace == 0 && square == 0 {
				start = i
			}
			square++
		case ']':
			if square > 0 {
				square--
			}
		}
	}
	if start <= 0 || square != 0 {
		return "", "", false
	}
	base = strings.TrimSpace(raw[:start])
	key = strings.TrimSpace(raw[start+1 : len(raw)-1])
	return base, key, base != "" && key != ""
}
