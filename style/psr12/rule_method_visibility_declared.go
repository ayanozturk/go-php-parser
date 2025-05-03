package psr12

import (
	"go-phpcs/style"
)

// MethodVisibilityDeclaredChecker checks that all class methods declare visibility (PSR-12 4.2)
type MethodVisibilityDeclaredChecker struct{}

func (c *MethodVisibilityDeclaredChecker) CheckIssues(lines []string, filename string) []style.StyleIssue {
	var issues []style.StyleIssue
	inClass := false
	braceDepth := 0
	for i, line := range lines {
		trimmed := trimWhitespace(line)
		// Skip PHPDoc and comment lines
		if len(trimmed) > 0 && (trimmed[0] == '/' || trimmed[0] == '*') {
			continue
		}
		if isClassDeclaration(trimmed) {
			inClass = true
			continue
		}
		if inClass {
			if trimmed == "{" {
				braceDepth++
				continue
			}
			if trimmed == "}" {
				braceDepth--
				if braceDepth <= 0 {
					inClass = false
				}
				continue
			}
			if isMethodDeclaration(trimmed) && !hasVisibility(trimmed) {
				issues = append(issues, style.StyleIssue{
					Filename: filename,
					Line:     i + 1,
					Type:     style.Error,
					Fixable:  false,
					Message:  "Visibility must be declared on all class methods",
					Code:     "PSR12.Methods.VisibilityDeclared",
				})
			}
		}
	}
	return issues
}

func isMethodDeclaration(line string) bool {
	// Look for 'function', but not anonymous (closures):
	idx := indexOfWord(line, "function")
	if idx == -1 {
		return false
	}
	// Find what's after 'function'
	n := idx + len("function")
	for n < len(line) && (line[n] == ' ' || line[n] == '\t') {
		n++
	}
	if n < len(line) && line[n] == '(' {
		// anonymous function: function (...) { ... }
		return false
	}
	return true // named function
}

// indexOfWord finds the index of 'word' as a standalone word in line, or -1 if not found
func indexOfWord(line, word string) int {
	for i := 0; i+len(word) <= len(line); i++ {
		if line[i:i+len(word)] == word {
			before := i == 0 || !isWordChar(line[i-1])
			after := i+len(word) == len(line) || !isWordChar(line[i+len(word)])
			if before && after {
				return i
			}
		}
	}
	return -1
}

func hasVisibility(line string) bool {
	return hasWord(line, "public") || hasWord(line, "protected") || hasWord(line, "private")
}

func containsWord(line, word string) bool {
	// Check that word is surrounded by non-word chars
	for i := 0; i+len(word) <= len(line); i++ {
		if line[i:i+len(word)] == word {
			before := i == 0 || !isWordChar(line[i-1])
			after := i+len(word) == len(line) || !isWordChar(line[i+len(word)])
			if before && after {
				return true
			}
		}
	}
	return false
}

func hasWord(line, word string) bool {
	return containsWord(line, word)
}

func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}
