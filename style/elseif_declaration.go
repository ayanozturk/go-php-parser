package style

import (
	"go-phpcs/ast"
	"regexp"
	"strings"
)

const elseIfDeclarationCode = "PSR12.ControlStructures.ElseIfDeclaration"

// ElseIfDeclarationChecker enforces PSR12 rule that "else if" should be written as "elseif"
// PSR12 states: "The keyword elseif SHOULD be used instead of else if so that all control keywords look like single words."
type ElseIfDeclarationChecker struct {
	elseIfRegex *regexp.Regexp
}

// NewElseIfDeclarationChecker creates a new checker with proper initialization
func NewElseIfDeclarationChecker() *ElseIfDeclarationChecker {
	// Match "else" followed by whitespace and then "if"
	// This regex captures the whitespace between "else" and "if"
	elseIfRegex := regexp.MustCompile(`\belse(\s+)if\b`)
	
	return &ElseIfDeclarationChecker{
		elseIfRegex: elseIfRegex,
	}
}

// CheckIssues analyzes the code for else if declaration violations
func (c *ElseIfDeclarationChecker) CheckIssues(lines []string, filename string) []StyleIssue {
	var issues []StyleIssue
	
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Skip empty lines and comments
		if len(trimmed) == 0 || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
			continue
		}
		
		// Check for "else if" patterns
		issues = append(issues, c.checkElseIfDeclaration(line, filename, i+1)...)
	}
	
	return issues
}

// checkElseIfDeclaration finds instances of "else if" and reports them as violations
func (c *ElseIfDeclarationChecker) checkElseIfDeclaration(line, filename string, lineNum int) []StyleIssue {
	var issues []StyleIssue
	
	// Skip if line contains string literals that might contain "else if"
	if c.isInStringLiteral(line) {
		return issues
	}
	
	matches := c.elseIfRegex.FindAllStringSubmatchIndex(line, -1)
	for _, match := range matches {
		// match[0] is start of entire match, match[1] is end of entire match
		// match[2] is start of whitespace capture group, match[3] is end of whitespace capture group
		
		if len(match) >= 4 {
			elseStart := match[0]
			elseEnd := match[0] + 4 // "else" is 4 characters
			whitespaceStart := match[2]
			whitespaceEnd := match[3]
			ifStart := whitespaceEnd
			
			// Check if this match is inside a string literal
			if c.isPositionInString(line, elseStart) {
				continue
			}
			
			// Verify this is actually "else" followed by "if"
			if elseEnd <= len(line) && ifStart+2 <= len(line) {
				elseWord := line[elseStart:elseEnd]
				ifWord := line[ifStart:ifStart+2]
				whitespace := line[whitespaceStart:whitespaceEnd]
				
				if elseWord == "else" && ifWord == "if" {
					issues = append(issues, StyleIssue{
						Filename: filename,
						Line:     lineNum,
						Column:   elseStart + 1, // 1-based column
						Type:     Error,
						Fixable:  true,
						Message:  "Use 'elseif' instead of 'else if'. Found " + strings.ReplaceAll(whitespace, "\t", "\\t") + " between 'else' and 'if'",
						Code:     elseIfDeclarationCode,
					})
				}
			}
		}
	}
	
	return issues
}

// isInStringLiteral checks if the line contains string literals that might contain "else if"
func (c *ElseIfDeclarationChecker) isInStringLiteral(line string) bool {
	// Simple check for common string patterns that contain "else if"
	return strings.Contains(line, `"`) && strings.Contains(line, "else if") ||
		   strings.Contains(line, `'`) && strings.Contains(line, "else if")
}

// isPositionInString checks if a specific position in the line is inside a string literal
func (c *ElseIfDeclarationChecker) isPositionInString(line string, pos int) bool {
	inSingleQuote := false
	inDoubleQuote := false
	escaped := false
	
	for i := 0; i < pos && i < len(line); i++ {
		char := line[i]
		
		if escaped {
			escaped = false
			continue
		}
		
		if char == '\\' {
			escaped = true
			continue
		}
		
		if char == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
		} else if char == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
		}
	}
	
	return inSingleQuote || inDoubleQuote
}

// FixElseIfDeclaration fixes else if declaration issues by replacing "else if" with "elseif"
func FixElseIfDeclaration(content string) string {
	// Replace all instances of "else" followed by whitespace and "if" with "elseif"
	elseIfRegex := regexp.MustCompile(`\belse\s+if\b`)
	return elseIfRegex.ReplaceAllString(content, "elseif")
}

// ElseIfDeclarationFixer implements StyleFixer for autofix support
type ElseIfDeclarationFixer struct{}

func (f ElseIfDeclarationFixer) Code() string              { return elseIfDeclarationCode }
func (f ElseIfDeclarationFixer) Fix(content string) string { return FixElseIfDeclaration(content) }

func init() {
	RegisterRule(elseIfDeclarationCode, func(filename string, content []byte, _ []ast.Node) []StyleIssue {
		lines := strings.Split(string(content), "\n")
		checker := NewElseIfDeclarationChecker()
		return checker.CheckIssues(lines, filename)
	})
	RegisterFixer(ElseIfDeclarationFixer{})
}