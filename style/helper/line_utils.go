package helper

// TrimWhitespace removes leading and trailing spaces and tabs from a string.
func TrimWhitespace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

// IsClassDeclaration checks if a line looks like a class/interface/trait/enum declaration.
func IsClassDeclaration(line string) bool {
	line = TrimWhitespace(line)
	if len(line) == 0 || line[0] == '/' || line[0] == '#' {
		return false
	}
	keywords := []string{"class ", "interface ", "trait ", "enum "}
	for _, k := range keywords {
		if len(line) >= len(k) && line[:len(k)] == k {
			return true
		}
	}
	return false
}
