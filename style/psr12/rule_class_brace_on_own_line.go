package psr12

import (
	"go-phpcs/style"
)

// ClassBraceOnOwnLineChecker checks that class opening braces are on their own line (PSR-12 4.1)
type ClassBraceOnOwnLineChecker struct{}

func (c *ClassBraceOnOwnLineChecker) CheckIssues(lines []string, filename string) []style.StyleIssue {
	var issues []style.StyleIssue
	for i := 0; i < len(lines); i++ {
		trimmed := lines[i]
		if isClassDeclaration(trimmed) {
			// If the same line contains a brace, it's an error
			if containsBraceOnSameLine(trimmed) {
				issues = append(issues, style.StyleIssue{
					Filename: filename,
					Line:     i + 1,
					Type:     style.Error,
					Fixable:  false,
					Message:  "Opening brace for class-like declaration must be on its own line",
					Code:     "PSR12.Classes.OpenBraceOnOwnLine",
				})
				continue
			}
			// If not last line, next line must be exactly "{"
			if i+1 < len(lines) {
				next := lines[i+1]
				if next != "{" {
					issues = append(issues, style.StyleIssue{
						Filename: filename,
						Line:     i + 2,
						Type:     style.Error,
						Fixable:  false,
						Message:  "Opening brace for class-like declaration must be on its own line",
						Code:     "PSR12.Classes.OpenBraceOnOwnLine",
					})
				}
			}
		}
	}
	return issues
}

func containsBraceOnSameLine(line string) bool {
	// Look for '{' after class/interface/trait/enum keyword
	keywords := []string{"class ", "interface ", "trait ", "enum "}
	for _, k := range keywords {
		idx := indexOf(line, k)
		if idx != -1 {
			// Check for '{' after the keyword
			braceIdx := indexOf(line[idx+len(k):], "{")
			if braceIdx != -1 {
				return true
			}
		}
	}
	return false
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}


// isClassDeclaration checks if a line looks like a class/interface/trait/enum declaration
func isClassDeclaration(line string) bool {
	line = trimWhitespace(line)
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

func trimWhitespace(s string) string {
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
