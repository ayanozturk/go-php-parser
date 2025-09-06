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
	keywordRegex    *regexp.Regexp
	functionRegex   *regexp.Regexp
}

// NewControlStructureSpacingChecker creates a new checker with proper initialization
func NewControlStructureSpacingChecker() *ControlStructureSpacingChecker {
	keywords := []string{"if", "else", "elseif", "for", "foreach", "while", "do", "switch", "try", "catch", "finally"}
	
	// Create regex for control keywords - must be word boundaries
	keywordPattern := `\b(` + strings.Join(keywords, "|") + `)\b`
	keywordRegex := regexp.MustCompile(keywordPattern)
	
	// Create regex for function calls - function name followed by space(s) and parenthesis
	functionRegex := regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_]*)\s+\(`)
	
	return &ControlStructureSpacingChecker{
		controlKeywords: keywords,
		keywordRegex:    keywordRegex,
		functionRegex:   functionRegex,
	}
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
	
	// Skip if line is inside a string (simple check for quotes)
	if strings.Contains(line, `"`) || strings.Contains(line, `'`) {
		// More sophisticated string detection would be needed for production
		inString := false
		quote := byte(0)
		for i := 0; i < len(line); i++ {
			if !inString && (line[i] == '"' || line[i] == '\'') {
				inString = true
				quote = line[i]
			} else if inString && line[i] == quote && (i == 0 || line[i-1] != '\\') {
				inString = false
				quote = 0
			}
		}
		if inString {
			return issues // Skip lines that are inside strings
		}
	}
	
	matches := c.keywordRegex.FindAllStringIndex(line, -1)
	for _, match := range matches {
		start := match[0]
		end := match[1]
		keyword := line[start:end]
		
		// Skip if inside a string literal
		if c.isInsideString(line, start) {
			continue
		}
		
		// Skip if this is part of a larger word (shouldn't happen with word boundaries, but safety check)
		if start > 0 && (isAlphaNumeric(line[start-1]) || line[start-1] == '_') {
			continue
		}
		if end < len(line) && (isAlphaNumeric(line[end]) || line[end] == '_') {
			continue
		}
		
		// Check what follows the keyword
		if end < len(line) {
			nextChar := line[end]
			
			// Special case for 'else' - it might be followed by 'if' or '{'
			if keyword == "else" {
				if end+2 <= len(line) && line[end:end+2] == "if" {
					// This is 'elseif' - should be handled by elseif pattern
					continue
				}
				if nextChar == '{' {
					issues = append(issues, StyleIssue{
						Filename: filename,
						Line:     lineNum,
						Column:   end + 1,
						Type:     Error,
						Fixable:  true,
						Message:  "Expected single space after '" + keyword + "' keyword",
						Code:     controlStructureSpacingCode,
					})
					continue
				}
			}
			
			// For other keywords, expect space before '('
			if nextChar == '(' {
				issues = append(issues, StyleIssue{
					Filename: filename,
					Line:     lineNum,
					Column:   end + 1,
					Type:     Error,
					Fixable:  true,
					Message:  "Expected single space after '" + keyword + "' keyword",
					Code:     controlStructureSpacingCode,
				})
			} else if nextChar != ' ' && nextChar != '\t' {
				// Not a space - this might be valid for some cases like 'else{' but generally wrong
				if keyword != "else" || nextChar != '{' {
					issues = append(issues, StyleIssue{
						Filename: filename,
						Line:     lineNum,
						Column:   end + 1,
						Type:     Error,
						Fixable:  true,
						Message:  "Expected single space after '" + keyword + "' keyword",
						Code:     controlStructureSpacingCode,
					})
				}
			} else {
				// Check for multiple spaces
				spaceCount := 0
				for j := end; j < len(line) && (line[j] == ' ' || line[j] == '\t'); j++ {
					spaceCount++
				}
				if spaceCount > 1 {
					issues = append(issues, StyleIssue{
						Filename: filename,
						Line:     lineNum,
						Column:   end + 1,
						Type:     Error,
						Fixable:  true,
						Message:  "Expected single space after '" + keyword + "' keyword, found multiple spaces",
						Code:     controlStructureSpacingCode,
					})
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
	
	matches := c.functionRegex.FindAllStringSubmatch(line, -1)
	matchIndices := c.functionRegex.FindAllStringSubmatchIndex(line, -1)
	
	for i, match := range matches {
		if len(match) < 2 {
			continue
		}
		
		functionName := match[1]
		indices := matchIndices[i]
		
		// Skip if this looks like a control structure
		isControlKeyword := false
		for _, keyword := range c.controlKeywords {
			if functionName == keyword {
				isControlKeyword = true
				break
			}
		}
		if isControlKeyword {
			continue
		}
		
		// Since our regex now only matches when there ARE spaces, we found a violation
		nameEnd := indices[2] // end of function name capture group
		
		issues = append(issues, StyleIssue{
			Filename: filename,
			Line:     lineNum,
			Column:   nameEnd + 1,
			Type:     Error,
			Fixable:  true,
			Message:  "No space allowed between function name and opening parenthesis",
			Code:     controlStructureSpacingCode,
		})
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
		
		// Fix function call spacing (remove spaces before parenthesis)
		funcPattern := regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_]*)\s+\(`)
		line = funcPattern.ReplaceAllStringFunc(line, func(match string) string {
			parts := strings.Split(match, "(")
			funcName := strings.TrimSpace(parts[0])
			// Skip control keywords
			for _, keyword := range checker.controlKeywords {
				if funcName == keyword {
					return match // Don't fix control structures
				}
			}
			return funcName + "("
		})
		
		// Fix brace spacing
		line = regexp.MustCompile(`\)\s*\{`).ReplaceAllString(line, ") {")
		line = regexp.MustCompile(`\)\s{2,}\{`).ReplaceAllString(line, ") {")
		
		lines[i] = line
	}
	
	return strings.Join(lines, "\n")
}

// ControlStructureSpacingFixer implements StyleFixer for autofix support
type ControlStructureSpacingFixer struct{}

func (f ControlStructureSpacingFixer) Code() string              { return controlStructureSpacingCode }
func (f ControlStructureSpacingFixer) Fix(content string) string { return FixControlStructureSpacing(content) }

func init() {
	RegisterRule(controlStructureSpacingCode, func(filename string, content []byte, _ []ast.Node) []StyleIssue {
		lines := strings.Split(string(content), "\n")
		checker := NewControlStructureSpacingChecker()
		return checker.CheckIssues(lines, filename)
	})
	RegisterFixer(ControlStructureSpacingFixer{})
}