package analyse

import "strings"

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
			value, ok := resolvePHPDocOffsetAccess(part, typeCtx)
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
	if fields := parseArrayShapeFields(base, typeCtx); len(fields) > 0 {
		keys := splitTopLevelTypes(key, '|')
		var result Type
		for _, candidate := range keys {
			field, exists := fields[normalizeArrayShapeKey(candidate)]
			if !exists || field.typ.IsEmpty() {
				return "", false
			}
			result = unionInferredTypes(result, field.typ)
		}
		if !result.IsEmpty() {
			return result.String(), true
		}
		return "", false
	}

	instance, generic := parseExactGenericTypeFromString(base)
	if generic {
		if _, value, iterable := iterableTypesFromGeneric(instance); iterable && !value.IsEmpty() {
			return value.String(), true
		}
	}
	return "", false
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
	if fields := parseArrayShapeFields(base, typeCtx); len(fields) > 0 {
		if projection == "key-of" {
			stringKey, intKey := false, false
			for key := range fields {
				if isNumericArrayKey(key) {
					intKey = true
				} else {
					stringKey = true
				}
			}
			if stringKey && intKey {
				return "int|string", true
			}
			if intKey {
				return "int", true
			}
			if stringKey {
				return "string", true
			}
			return "never", true
		}
		var values Type
		for _, field := range fields {
			if field.typ.IsEmpty() {
				return "", false
			}
			values = unionInferredTypes(values, field.typ)
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
