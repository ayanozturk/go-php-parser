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

// IndexOfWord finds the index of 'word' as a standalone word in line, or -1 if not found
func IndexOfWord(line, word string) int {
	for i := 0; i+len(word) <= len(line); i++ {
		if line[i:i+len(word)] == word {
			before := i == 0 || !IsWordChar(line[i-1])
			after := i+len(word) == len(line) || !IsWordChar(line[i+len(word)])
			if before && after {
				return i
			}
		}
	}
	return -1
}

// ContainsWord checks that word is surrounded by non-word chars
func ContainsWord(line, word string) bool {
	for i := 0; i+len(word) <= len(line); i++ {
		if line[i:i+len(word)] == word {
			before := i == 0 || !IsWordChar(line[i-1])
			after := i+len(word) == len(line) || !IsWordChar(line[i+len(word)])
			if before && after {
				return true
			}
		}
	}
	return false
}

// HasWord returns true if the word is present as a standalone word
func HasWord(line, word string) bool {
	return ContainsWord(line, word)
}

// IsWordChar returns true if the byte is a-z, A-Z, 0-9, or _
func IsWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}
