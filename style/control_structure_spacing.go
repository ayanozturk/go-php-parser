package style

import (
	"go-phpcs/ast"
	"regexp"
	"strings"
)

const controlStructureSpacingCode = "PSR12.ControlStructures.ControlStructureSpacing"

// ControlStructureSpacingChecker enforces PSR12 control structure spacing rules:
// - Single space after control structure keywords (if, for, while, etc.)
// - No space between function name and opening parenthesis
// - Single space before opening brace of control structures
type ControlStructureSpacingChecker struct {
	controlKeywords []string
}

// NewControlStructureSpacingChecker creates a new checker with proper initialization
func NewControlStructureSpacingChecker() *ControlStructureSpacingChecker {
	keywords := []string{"if", "else", "elseif", "for", "foreach", "while", "do", "switch", "try", "catch", "finally"}
	return &ControlStructureSpacingChecker{controlKeywords: keywords}
}

// CheckIssues analyzes the code for control structure spacing violations
func (c *ControlStructureSpacingChecker) CheckIssues(lines []string, filename string) []StyleIssue {
	var issues []StyleIssue

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments
		if len(trimmed) == 0 || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
			continue
		}

		// Check control structure keyword spacing
		issues = append(issues, c.checkControlKeywordSpacing(line, filename, i+1)...)

		// Check function call spacing (should have no space before parenthesis)
		issues = append(issues, c.checkFunctionCallSpacing(line, filename, i+1)...)

		// Check brace spacing for control structures
		issues = append(issues, c.checkBraceSpacing(line, filename, i+1)...)
	}

	return issues
}

// checkControlKeywordSpacing ensures single space after control keywords
func (c *ControlStructureSpacingChecker) checkControlKeywordSpacing(line, filename string, lineNum int) []StyleIssue {
	var issues []StyleIssue

	// Quick precheck: ignore lines without letters
	hasLetter := false
	for i := 0; i < len(line); i++ {
		if (line[i] >= 'a' && line[i] <= 'z') || (line[i] >= 'A' && line[i] <= 'Z') {
			hasLetter = true
			break
		}
	}
	if !hasLetter {
		return issues
	}

	// Scan for identifiers and compare to keyword list
	for i := 0; i < len(line); i++ {
		ch := line[i]
		if !(ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')) {
			continue
		}
		start := i
		for i < len(line) && (line[i] == '_' || (line[i] >= 'a' && line[i] <= 'z') || (line[i] >= 'A' && line[i] <= 'Z') || (line[i] >= '0' && line[i] <= '9')) {
			i++
		}
		end := i
		keyword := line[start:end]

		// Skip string literal positions
		if c.isInsideString(line, start) {
			continue
		}

		// Ensure boundaries
		if start > 0 && (isAlphaNumeric(line[start-1]) || line[start-1] == '_') {
			continue
		}
		if end < len(line) && (isAlphaNumeric(line[end]) || line[end] == '_') {
			continue
		}

		isKw := false
		for _, kw := range c.controlKeywords {
			if keyword == kw {
				isKw = true
				break
			}
		}
		if !isKw {
			continue
		}

		if end < len(line) {
			nextChar := line[end]

			if keyword == "else" {
				if end+2 <= len(line) && line[end:end+2] == "if" {
					continue
				}
				if nextChar == '{' {
					issues = append(issues, StyleIssue{Filename: filename, Line: lineNum, Column: end + 1, Type: Error, Fixable: true, Message: "Expected single space after '" + keyword + "' keyword", Code: controlStructureSpacingCode})
					continue
				}
			}

			if nextChar == '(' {
				issues = append(issues, StyleIssue{Filename: filename, Line: lineNum, Column: end + 1, Type: Error, Fixable: true, Message: "Expected single space after '" + keyword + "' keyword", Code: controlStructureSpacingCode})
			} else if nextChar != ' ' && nextChar != '\t' {
				if keyword != "else" || nextChar != '{' {
					issues = append(issues, StyleIssue{Filename: filename, Line: lineNum, Column: end + 1, Type: Error, Fixable: true, Message: "Expected single space after '" + keyword + "' keyword", Code: controlStructureSpacingCode})
				}
			} else {
				spaceCount := 0
				for j := end; j < len(line) && (line[j] == ' ' || line[j] == '\t'); j++ {
					spaceCount++
				}
				if spaceCount > 1 {
					issues = append(issues, StyleIssue{Filename: filename, Line: lineNum, Column: end + 1, Type: Error, Fixable: true, Message: "Expected single space after '" + keyword + "' keyword, found multiple spaces", Code: controlStructureSpacingCode})
				}
			}
		}
	}

	return issues
}

// isInsideString checks if a position is inside a string literal
func (c *ControlStructureSpacingChecker) isInsideString(line string, pos int) bool {
	inSingle := false
	inDouble := false

	for i := 0; i < pos && i < len(line); i++ {
		if !inSingle && !inDouble {
			if line[i] == '\'' {
				inSingle = true
			} else if line[i] == '"' {
				inDouble = true
			}
		} else if inSingle && line[i] == '\'' && (i == 0 || line[i-1] != '\\') {
			inSingle = false
		} else if inDouble && line[i] == '"' && (i == 0 || line[i-1] != '\\') {
			inDouble = false
		}
	}

	return inSingle || inDouble
}

// checkFunctionCallSpacing ensures no space between function name and parenthesis
func (c *ControlStructureSpacingChecker) checkFunctionCallSpacing(line, filename string, lineNum int) []StyleIssue {
	var issues []StyleIssue

	for i := 0; i < len(line); i++ {
		ch := line[i]
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_') {
			continue
		}
		start := i
		for i < len(line) && ((line[i] >= 'a' && line[i] <= 'z') || (line[i] >= 'A' && line[i] <= 'Z') || (line[i] >= '0' && line[i] <= '9') || line[i] == '_') {
			i++
		}
		end := i
		name := line[start:end]

		isControl := false
		for _, kw := range c.controlKeywords {
			if name == kw {
				isControl = true
				break
			}
		}
		if isControl {
			continue
		}

		j := end
		spaces := 0
		for j < len(line) && (line[j] == ' ' || line[j] == '\t') {
			spaces++
			j++
		}
		if spaces > 0 && j < len(line) && line[j] == '(' {
			issues = append(issues, StyleIssue{Filename: filename, Line: lineNum, Column: end + 1, Type: Error, Fixable: true, Message: "No space allowed between function name and opening parenthesis", Code: controlStructureSpacingCode})
		}
	}

	return issues
}

// checkBraceSpacing ensures single space before opening brace in control structures
func (c *ControlStructureSpacingChecker) checkBraceSpacing(line, filename string, lineNum int) []StyleIssue {
	var issues []StyleIssue

	// Look for patterns like ") {" or "){" in control structures
	for i := 0; i < len(line)-1; i++ {
		if line[i] == ')' && i+1 < len(line) {
			nextChar := line[i+1]
			if nextChar == '{' {
				// No space before brace
				issues = append(issues, StyleIssue{
					Filename: filename,
					Line:     lineNum,
					Column:   i + 2,
					Type:     Error,
					Fixable:  true,
					Message:  "Expected single space before opening brace",
					Code:     controlStructureSpacingCode,
				})
			} else if nextChar == ' ' || nextChar == '\t' {
				// Check for correct spacing
				spaceCount := 0
				j := i + 1
				for j < len(line) && (line[j] == ' ' || line[j] == '\t') {
					spaceCount++
					j++
				}
				if j < len(line) && line[j] == '{' {
					if spaceCount != 1 {
						issues = append(issues, StyleIssue{
							Filename: filename,
							Line:     lineNum,
							Column:   i + 2,
							Type:     Error,
							Fixable:  true,
							Message:  "Expected single space before opening brace, found multiple spaces",
							Code:     controlStructureSpacingCode,
						})
					}
				}
			}
		}
	}

	return issues
}

// isAlphaNumeric checks if a character is alphanumeric
func isAlphaNumeric(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// FixControlStructureSpacing fixes control structure spacing issues
func FixControlStructureSpacing(content string) string {
	lines := strings.Split(content, "\n")
	checker := NewControlStructureSpacingChecker()

	for i, line := range lines {
		// Fix control keyword spacing
		for _, keyword := range checker.controlKeywords {
			// Fix missing space after keyword before (
			pattern := regexp.MustCompile(`\b` + keyword + `\(`)
			line = pattern.ReplaceAllString(line, keyword+" (")

			// Fix multiple spaces after keyword
			pattern = regexp.MustCompile(`\b` + keyword + `\s{2,}`)
			line = pattern.ReplaceAllString(line, keyword+" ")
		}

		// Fix function call spacing (remove spaces before parenthesis) without regex
		line = fixFuncCallSpacingNoRegex(line, checker.controlKeywords)

		// Fix brace spacing
		line = regexp.MustCompile(`\)\s*\{`).ReplaceAllString(line, ") {")
		line = regexp.MustCompile(`\)\s{2,}\{`).ReplaceAllString(line, ") {")

		lines[i] = line
	}

	return strings.Join(lines, "\n")
}

// fixFuncCallSpacingNoRegex removes spaces between function name and '(' using a linear scan
func fixFuncCallSpacingNoRegex(line string, controlKeywords []string) string {
	var out strings.Builder
	for i := 0; i < len(line); {
		if (line[i] >= 'a' && line[i] <= 'z') || (line[i] >= 'A' && line[i] <= 'Z') || line[i] == '_' {
			start := i
			for i < len(line) && ((line[i] >= 'a' && line[i] <= 'z') || (line[i] >= 'A' && line[i] <= 'Z') || (line[i] >= '0' && line[i] <= '9') || line[i] == '_') {
				i++
			}
			name := line[start:i]
			out.WriteString(name)
			j := i
			spaceCount := 0
			for j < len(line) && (line[j] == ' ' || line[j] == '\t') {
				spaceCount++
				j++
			}
			if j < len(line) && line[j] == '(' {
				isControl := false
				for _, kw := range controlKeywords {
					if name == kw {
						isControl = true
						break
					}
				}
				if !isControl && spaceCount > 0 {
					out.WriteByte('(')
					i = j + 1
					continue
				}
			}
			for k := 0; k < spaceCount; k++ {
				out.WriteByte(' ')
			}
			if j < len(line) {
				out.WriteByte(line[j])
				i = j + 1
			} else {
				i = j
			}
		} else {
			out.WriteByte(line[i])
			i++
		}
	}
	return out.String()
}

// ControlStructureSpacingFixer implements StyleFixer for autofix support
type ControlStructureSpacingFixer struct{}

func (f ControlStructureSpacingFixer) Code() string { return controlStructureSpacingCode }
func (f ControlStructureSpacingFixer) Fix(content string) string {
	return FixControlStructureSpacing(content)
}

func init() {
	RegisterRule(controlStructureSpacingCode, func(filename string, content []byte, _ []ast.Node) []StyleIssue {
		lines := strings.Split(string(content), "\n")
		checker := NewControlStructureSpacingChecker()
		return checker.CheckIssues(lines, filename)
	})
	RegisterFixer(ControlStructureSpacingFixer{})
}
