// PSR12.Classes.ClosingBraceOnOwnLine
package style

import (
	"go-phpcs/ast"
	"go-phpcs/style/helper"
	"strings"
)

const closingBraceOnOwnLineCode = "PSR12.Classes.ClosingBraceOnOwnLine"

type ClosingBraceOnOwnLineChecker struct{}

// Helper struct to track class parsing state
type classState struct {
	inClass    bool
	braceDepth int
}

// Helper: Check and report issues for a class closing brace line
func checkClassClosingBraceIssues(line string, nextLine string, lineNum int, filename string, issues *[]StyleIssue) {
	indices := findClosingBraceIndices(line)
	if len(indices) == 0 {
		return
	}
	lastIdx := indices[len(indices)-1]
	before := strings.TrimSpace(line[:lastIdx])
	after := strings.TrimSpace(line[lastIdx+1:])
	if before != "" || (after != "" && after != "?>") {
		*issues = append(*issues, StyleIssue{
			Filename: filename,
			Line:     lineNum,
			Type:     Error,
			Fixable:  true,
			Message:  "Class closing brace must be on its own line with nothing before or after",
			Code:     closingBraceOnOwnLineCode,
		})
	}
	if nextLine != "" && nextLine != "}" && nextLine != "?>" {
		*issues = append(*issues, StyleIssue{
			Filename: filename,
			Line:     lineNum + 1,
			Type:     Error,
			Fixable:  true,
			Message:  "Code must not follow class closing brace on the next line (should be blank or another closing brace)",
			Code:     closingBraceOnOwnLineCode,
		})
	}
}

// Helper: Process a line when inside a class for CheckIssues
func processInClassCheckIssues(line string, nextLine string, lineNum int, filename string, state *classState, issues *[]StyleIssue) bool {
	updateClassState(line, state)
	if state.braceDepth < 0 {
		state.inClass = false
		return true
	}
	if isClassClosingBraceLine(line, state.braceDepth) {
		checkClassClosingBraceIssues(line, nextLine, lineNum, filename, issues)
		state.inClass = false
		return true
	}
	return false
}

func (c *ClosingBraceOnOwnLineChecker) CheckIssues(lines []string, filename string) []StyleIssue {
	var issues []StyleIssue
	if len(lines) == 1 && helper.TrimWhitespace(lines[0]) == "}" {
		issues = append(issues, StyleIssue{
			Filename: filename,
			Line:     1,
			Type:     Error,
			Fixable:  false,
			Message:  "Syntax error: file contains only a closing brace",
			Code:     "Syntax.Error",
		})
		return issues
	}
	state := classState{}
	for i, line := range lines {
		trimmed := helper.TrimWhitespace(line)
		if helper.IsClassDeclaration(trimmed) {
			state.inClass = true
			state.braceDepth = 0
			continue
		}
		if state.inClass {
			nextLine := ""
			if i+1 < len(lines) {
				nextLine = helper.TrimWhitespace(lines[i+1])
			}
			if processInClassCheckIssues(line, nextLine, i+1, filename, &state, &issues) {
				continue
			}
		}
	}
	return issues
}

// Helper: Update class state with brace counts
func updateClassState(line string, state *classState) {
	openCount := strings.Count(line, "{")
	closeCount := strings.Count(line, "}")
	state.braceDepth += openCount
	state.braceDepth -= closeCount
}

// Helper: Determine if this line is the class closing brace line
func isClassClosingBraceLine(line string, braceDepth int) bool {
	closeCount := strings.Count(line, "}")
	return closeCount > 0 && braceDepth == 0
}

// Helper: Find all indices of '}' in a line
func findClosingBraceIndices(line string) []int {
	indices := make([]int, 0)
	for idx := 0; ; {
		pos := strings.Index(line[idx:], "}")
		if pos == -1 {
			break
		}
		indices = append(indices, idx+pos)
		idx += pos + 1
	}
	return indices
}

// Helper: Handle a line with a class closing brace
func handleClassClosingBraceLine(line string, out *[]string) {
	indices := findClosingBraceIndices(line)
	if len(indices) == 0 {
		*out = append(*out, line)
		return
	}
	lastIdx := indices[len(indices)-1]
	before := strings.TrimRight(line[:lastIdx], " \t")
	after := strings.TrimSpace(line[lastIdx+1:])
	if before != "" {
		*out = append(*out, before)
		*out = append(*out, "}")
	} else {
		*out = append(*out, "}")
	}
	if after != "" && after != "?>" {
		*out = append(*out, after)
	}
}

// Helper: Process a line when inside a class for FixClassClosingBraceOnOwnLine
func processInClassFix(line string, nextLine string, braceDepth *int, inClass *bool, out *[]string) bool {
	openCount := strings.Count(line, "{")
	closeCount := strings.Count(line, "}")
	*braceDepth += openCount
	*braceDepth -= closeCount
	if *braceDepth < 0 {
		*inClass = false
		*out = append(*out, line)
		return true
	}
	if closeCount > 0 && *braceDepth == 0 {
		handleClassClosingBraceLine(line, out)
		if nextLine != "" && nextLine != "}" && nextLine != "?>" {
			*out = append(*out, "")
		}
		*inClass = false
		return true
	}
	return false
}

func FixClassClosingBraceOnOwnLine(content string) string {
	lines := strings.Split(content, "\n")
	var out []string
	inClass := false
	braceDepth := 0
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := helper.TrimWhitespace(line)
		if helper.IsClassDeclaration(trimmed) {
			inClass = true
			braceDepth = 0
			out = append(out, line)
			continue
		}
		if inClass {
			nextLine := ""
			if i+1 < len(lines) {
				nextLine = helper.TrimWhitespace(lines[i+1])
			}
			if processInClassFix(line, nextLine, &braceDepth, &inClass, &out) {
				continue
			}
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// ClosingBraceOnOwnLineFixer implements StyleFixer for autofix support.
type ClosingBraceOnOwnLineFixer struct{}

func (f ClosingBraceOnOwnLineFixer) Code() string { return closingBraceOnOwnLineCode }
func (f ClosingBraceOnOwnLineFixer) Fix(content string) string {
	return FixClassClosingBraceOnOwnLine(content)
}

func init() {
	RegisterRule(closingBraceOnOwnLineCode, func(filename string, content []byte, _ []ast.Node) []StyleIssue {
		lines := strings.Split(string(content), "\n")
		checker := &ClosingBraceOnOwnLineChecker{}
		return checker.CheckIssues(lines, filename)
	})
	RegisterFixer(ClosingBraceOnOwnLineFixer{})
}
