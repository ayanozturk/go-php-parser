package helper

// PascalCase returns the PascalCase version of the input name.
// If the name is already PascalCase, it returns it unchanged.
func PascalCase(name string) string {
	if name == "" {
		return name
	}
	result := ""
	capitalizeNext := true
	for _, r := range name {
		if r == '_' {
			capitalizeNext = true
			continue
		}
		if capitalizeNext {
			result += string(ToUpper(r))
			capitalizeNext = false
		} else {
			result += string(r)
		}
	}
	return result
}

// CamelCase returns the camelCase version of the input name.
// If the name is already camelCase, it returns it unchanged.
func CamelCase(name string) string {
	if name == "" {
		return name
	}
	result := ""
	capitalizeNext := false
	for i, r := range name {
		if r == '_' {
			capitalizeNext = true
			continue
		}
		if i == 0 {
			result += string(ToLower(r))
			continue
		}
		if capitalizeNext {
			result += string(ToUpper(r))
			capitalizeNext = false
		} else {
			result += string(r)
		}
	}
	return result
}

func ToLower(r rune) rune {
	if 'A' <= r && r <= 'Z' {
		return r + ('a' - 'A')
	}
	return r
}

func ToUpper(r rune) rune {
	if 'a' <= r && r <= 'z' {
		return r - ('a' - 'A')
	}
	return r
}
