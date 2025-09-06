package analyse

import (
	"go-phpcs/ast"
	"go-phpcs/sharedcache"
	"strings"
)

// SideEffectsRule implements PSR1.Files.SideEffects
// Ensures files either declare symbols OR cause side-effects, but not both
type SideEffectsRule struct{}

// CheckIssuesWithSource performs analysis given explicit source content (used by tests)
func (r *SideEffectsRule) CheckIssuesWithSource(filename string, content []byte, nodes []ast.Node) []AnalysisIssue {
	var issues []AnalysisIssue

	// Analyze the file content for side effects and declarations
	hasSideEffects := r.hasSideEffects(string(content))
	hasDeclarations := r.hasDeclarations(nodes)

	if hasSideEffects && hasDeclarations {
		issues = append(issues, AnalysisIssue{
			Filename: filename,
			Line:     1,
			Column:   1,
			Code:     "PSR1.Files.SideEffects",
			Message:  "A file should declare new symbols or cause side-effects, but not both",
		})
	}

	return issues
}

// CheckIssues analyzes the entire file to detect both symbol declarations and side effects
func (r *SideEffectsRule) CheckIssues(nodes []ast.Node, filename string) []AnalysisIssue {
	content, err := sharedcache.GetCachedFileContent(filename)
	if err != nil {
		return nil
	}

	return r.CheckIssuesWithSource(filename, content, nodes)
}

// hasSideEffects checks if the file content contains side effect operations
func (r *SideEffectsRule) hasSideEffects(content string) bool {
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "/*") {
			continue
		}

		// Check for common side effect patterns
		if r.containsSideEffect(line) {
			return true
		}
	}

	return false
}

// containsSideEffect checks if a line contains side effect operations
func (r *SideEffectsRule) containsSideEffect(line string) bool {
	// Common side effect patterns (case insensitive)
	patterns := []string{
		"echo ", "print ", "printf(", "sprintf(",
		"file_get_contents(", "file_put_contents(", "fopen(",
		"header(", "setcookie(", "session_start(",
		"mail(", "curl_exec(", "exec(", "system(",
		"require ", "include ", "require_once ", "include_once ",
		"new ", // instantiation can be considered a side effect in PSR1 context
	}

	lineLower := strings.ToLower(line)

	for _, pattern := range patterns {
		if strings.Contains(lineLower, strings.ToLower(pattern)) {
			// Additional checks to avoid false positives
			if r.isLikelySideEffect(line, pattern) {
				return true
			}
		}
	}

	return false
}

// isLikelySideEffect performs additional validation to avoid false positives
func (r *SideEffectsRule) isLikelySideEffect(line, pattern string) bool {
	// Skip if this looks like a function declaration parameter
	if strings.Contains(line, "function ") && strings.Contains(line, "$") {
		return false
	}

	// Skip comments and strings - check if the pattern appears within quotes or comments
	if r.isInCommentOrString(line, pattern) {
		return false
	}

	// Skip if this is within a function/class definition context
	// This is a simplified check - in practice, we'd need more sophisticated parsing
	if strings.Contains(line, "function ") || strings.Contains(line, "class ") {
		return false
	}

	return true
}

// isInCommentOrString checks if a pattern appears within comments or string literals
func (r *SideEffectsRule) isInCommentOrString(line, pattern string) bool {
	// Simple check: if the line contains comment markers or quotes before the pattern
	lineLower := strings.ToLower(line)
	patternLower := strings.ToLower(pattern)

	// Find where the pattern appears
	patternIndex := strings.Index(lineLower, patternLower)
	if patternIndex == -1 {
		return false
	}

	// Check if there's a comment marker or quote before the pattern
	beforePattern := lineLower[:patternIndex]

	// Check for comment markers
	if strings.Contains(beforePattern, "//") ||
		strings.Contains(beforePattern, "/*") ||
		strings.Contains(beforePattern, "#") {
		return true
	}

	// Check for quotes (simple heuristic)
	singleQuoteCount := strings.Count(beforePattern, "'")
	doubleQuoteCount := strings.Count(beforePattern, "\"")

	// If we have an odd number of quotes before the pattern, it's likely inside a string
	if singleQuoteCount%2 == 1 || doubleQuoteCount%2 == 1 {
		return true
	}

	return false
}

// hasDeclarations checks if the file contains symbol declarations
func (r *SideEffectsRule) hasDeclarations(nodes []ast.Node) bool {
	for _, node := range nodes {
		if r.isDeclaration(node) {
			return true
		}
	}
	return false
}

// isDeclaration checks if a node represents a symbol declaration
func (r *SideEffectsRule) isDeclaration(node ast.Node) bool {
	switch n := node.(type) {
	case *ast.ClassNode:
		return true
	case *ast.FunctionNode:
		return true
	case *ast.InterfaceNode:
		return true
	case *ast.TraitNode:
		return true
	case *ast.ConstantNode:
		return true
	case *ast.NamespaceNode:
		// Check if namespace contains declarations
		for _, bodyNode := range n.Body {
			if r.isDeclaration(bodyNode) {
				return true
			}
		}
	}

	return false
}

func init() {
	RegisterAnalysisRule("PSR1.Files.SideEffects", func(filename string, nodes []ast.Node) []AnalysisIssue {
		rule := &SideEffectsRule{}
		return rule.CheckIssues(nodes, filename)
	})
}
