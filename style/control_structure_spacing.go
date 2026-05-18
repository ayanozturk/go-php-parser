package style

import (
	"go-phpcs/ast"
	"go-phpcs/style/helper"
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
	nonCallKeywords map[string]struct{}
}

type stringState struct {
	inSingle     bool
	inDouble     bool
	heredocLabel string
}

// NewControlStructureSpacingChecker creates a new checker with proper initialization
func NewControlStructureSpacingChecker() *ControlStructureSpacingChecker {
	keywords := []string{"if", "else", "elseif", "for", "foreach", "while", "do", "switch", "try", "catch", "finally"}
	nonCallKeywords := map[string]struct{}{
		"array":        {},
		"clone":        {},
		"die":          {},
		"echo":         {},
		"empty":        {},
		"eval":         {},
		"exit":         {},
		"fn":           {},
		"function":     {},
		"include":      {},
		"include_once": {},
		"isset":        {},
		"list":         {},
		"match":        {},
		"new":          {},
		"print":        {},
		"require":      {},
		"require_once": {},
		"return":       {},
		"unset":        {},
		"use":          {},
	}
	return &ControlStructureSpacingChecker{controlKeywords: keywords, nonCallKeywords: nonCallKeywords}
}

func (c *ControlStructureSpacingChecker) isNonCallKeyword(name string) bool {
	_, ok := c.nonCallKeywords[name]
	return ok
}

// CheckIssues analyzes the code for control structure spacing violations.
func (c *ControlStructureSpacingChecker) CheckIssues(lines []string, filename string) []StyleIssue {
	var issues []StyleIssue
	commentState := &helper.CommentState{}
	state := stringState{}

	for i, line := range lines {
		masked, nextState := maskControlStructureNonCode(line, commentState, state)
		trimmed := strings.TrimSpace(masked)
		state = nextState

		if len(trimmed) == 0 {
			continue
		}

		issues = append(issues, c.checkControlKeywordSpacing(masked, filename, i+1)...)
		issues = append(issues, c.checkFunctionCallSpacing(masked, filename, i+1)...)
		issues = append(issues, c.checkBraceSpacing(masked, filename, i+1)...)
	}

	return issues
}

func maskControlStructureNonCode(line string, commentState *helper.CommentState, state stringState) (string, stringState) {
	masked := []byte(line)
	if state.heredocLabel != "" {
		for i := range masked {
			masked[i] = ' '
		}
		if isHeredocTerminatorLine(line, state.heredocLabel) {
			state.heredocLabel = ""
		}
		return string(masked), state
	}

	for i := 0; i < len(masked); {
		if commentState.InBlockComment {
			masked[i] = ' '
			if i+1 < len(masked) {
				masked[i+1] = ' '
			}
			j := helper.HandleBlockComment(line, i, commentState)
			if j == i {
				i++
				continue
			}
			for k := i; k < j && k < len(masked); k++ {
				masked[k] = ' '
			}
			i = j
			continue
		}

		if !state.inSingle && !state.inDouble {
			if label, ok := heredocStartAt(line, i); ok {
				for k := i; k < len(masked); k++ {
					masked[k] = ' '
				}
				state.heredocLabel = label
				break
			}
			if i+1 < len(masked) && line[i] == '/' && line[i+1] == '/' {
				for k := i; k < len(masked); k++ {
					masked[k] = ' '
				}
				break
			}
			if line[i] == '#' {
				for k := i; k < len(masked); k++ {
					masked[k] = ' '
				}
				break
			}
			if i+1 < len(masked) && line[i] == '/' && line[i+1] == '*' {
				j := helper.HandleBlockComment(line, i, commentState)
				for k := i; k < j && k < len(masked); k++ {
					masked[k] = ' '
				}
				i = j
				continue
			}
		}

		if !state.inDouble && line[i] == '\'' {
			state.inSingle = !state.inSingle
			masked[i] = ' '
			i++
			continue
		}
		if !state.inSingle && line[i] == '"' {
			state.inDouble = !state.inDouble
			masked[i] = ' '
			i++
			continue
		}
		if (state.inSingle || state.inDouble) && line[i] == '\\' {
			masked[i] = ' '
			if i+1 < len(masked) {
				masked[i+1] = ' '
			}
			i += 2
			continue
		}
		if state.inSingle || state.inDouble {
			masked[i] = ' '
			i++
			continue
		}

		i++
	}

	return string(masked), state
}

// checkControlKeywordSpacing ensures single space after control keywords.
func (c *ControlStructureSpacingChecker) checkControlKeywordSpacing(line, filename string, lineNum int) []StyleIssue {
	var issues []StyleIssue

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

			if nextChar == '=' {
				continue
			}

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

func heredocStartAt(line string, pos int) (string, bool) {
	if pos+3 > len(line) || line[pos:pos+3] != "<<<" {
		return "", false
	}

	marker := strings.TrimSpace(line[pos+3:])
	if marker == "" {
		return "", false
	}

	if marker[0] == '\'' || marker[0] == '"' {
		quote := marker[0]
		end := strings.IndexByte(marker[1:], quote)
		if end < 0 {
			return "", false
		}
		label := marker[1 : 1+end]
		if label == "" {
			return "", false
		}
		return label, true
	}

	end := 0
	for end < len(marker) {
		ch := marker[end]
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' {
			end++
			continue
		}
		break
	}
	if end == 0 {
		return "", false
	}
	return marker[:end], true
}

func isHeredocTerminatorLine(line, label string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	if !strings.HasPrefix(trimmed, label) {
		return false
	}
	if len(trimmed) == len(label) {
		return true
	}
	next := trimmed[len(label)]
	return next == ';' || next == ',' || next == ')' || next == ']' || next == '}'
}

// checkFunctionCallSpacing ensures no space between function name and parenthesis.
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
		if c.isNonCallKeyword(name) {
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

// checkBraceSpacing ensures single space before opening brace in control structures.
func (c *ControlStructureSpacingChecker) checkBraceSpacing(line, filename string, lineNum int) []StyleIssue {
	var issues []StyleIssue

	for i := 0; i < len(line)-1; i++ {
		if line[i] == ')' && i+1 < len(line) {
			nextChar := line[i+1]
			if nextChar == '{' {
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
				spaceCount := 0
				j := i + 1
				for j < len(line) && (line[j] == ' ' || line[j] == '\t') {
					spaceCount++
					j++
				}
				if j < len(line) && line[j] == '{' && spaceCount != 1 {
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

	return issues
}

func isAlphaNumeric(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// FixControlStructureSpacing fixes control structure spacing issues.
func FixControlStructureSpacing(content string) string {
	lines := strings.Split(content, "\n")
	checker := NewControlStructureSpacingChecker()
	commentState := &helper.CommentState{}
	state := stringState{}

	for i, line := range lines {
		initialCommentState := *commentState
		initialState := state
		masked, nextState := maskControlStructureNonCode(line, commentState, state)
		state = nextState

		for _, keyword := range checker.controlKeywords {
			pattern := regexp.MustCompile(`\b` + keyword + `\(`)
			line = replaceMatchesOutsideMask(line, masked, pattern, keyword+" (")

			pattern = regexp.MustCompile(`\b` + keyword + `\s{2,}`)
			line = replaceMatchesOutsideMask(line, masked, pattern, keyword+" ")
		}
		masked = remaskControlStructureLine(line, &initialCommentState, initialState)

		line = fixFuncCallSpacingNoRegex(line, masked, checker.controlKeywords, checker.nonCallKeywords)
		masked = remaskControlStructureLine(line, &initialCommentState, initialState)

		line = replaceMatchesOutsideMask(line, masked, regexp.MustCompile(`\)\s*\{`), ") {")

		lines[i] = line
	}

	return strings.Join(lines, "\n")
}

func remaskControlStructureLine(line string, commentState *helper.CommentState, state stringState) string {
	if commentState == nil {
		masked, _ := maskControlStructureNonCode(line, &helper.CommentState{}, state)
		return masked
	}
	clone := *commentState
	masked, _ := maskControlStructureNonCode(line, &clone, state)
	return masked
}

func fixFuncCallSpacingNoRegex(line, masked string, controlKeywords []string, nonCallKeywords map[string]struct{}) string {
	var out strings.Builder
	for i := 0; i < len(line); {
		if i >= len(masked) {
			out.WriteString(line[i:])
			break
		}
		if masked[i] != line[i] {
			out.WriteByte(line[i])
			i++
			continue
		}
		if (line[i] >= 'a' && line[i] <= 'z') || (line[i] >= 'A' && line[i] <= 'Z') || line[i] == '_' {
			start := i
			for i < len(line) && i < len(masked) && masked[i] == line[i] && ((line[i] >= 'a' && line[i] <= 'z') || (line[i] >= 'A' && line[i] <= 'Z') || (line[i] >= '0' && line[i] <= '9') || line[i] == '_') {
				i++
			}
			name := line[start:i]
			out.WriteString(name)
			j := i
			spaceCount := 0
			for j < len(line) && j < len(masked) && masked[j] == line[j] && (line[j] == ' ' || line[j] == '\t') {
				spaceCount++
				j++
			}
			if j < len(line) && j < len(masked) && masked[j] == line[j] && line[j] == '(' {
				isControl := false
				for _, kw := range controlKeywords {
					if name == kw {
						isControl = true
						break
					}
				}
				if !isControl && spaceCount > 0 {
					if _, ok := nonCallKeywords[name]; ok {
						out.WriteByte(' ')
						out.WriteByte('(')
						i = j + 1
						continue
					}
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

func replaceMatchesOutsideMask(line, masked string, pattern *regexp.Regexp, replacement string) string {
	matches := pattern.FindAllStringIndex(masked, -1)
	if len(matches) == 0 {
		return line
	}

	var out strings.Builder
	last := 0
	for _, match := range matches {
		start, end := match[0], match[1]
		if start < last {
			start = last
		}
		if start > len(line) {
			break
		}
		if end > len(line) {
			end = len(line)
		}
		if start >= end {
			continue
		}
		out.WriteString(line[last:start])
		out.WriteString(replacement)
		last = end
	}
	if last < len(line) {
		out.WriteString(line[last:])
	}
	return out.String()
}

// ControlStructureSpacingFixer implements StyleFixer for autofix support.
type ControlStructureSpacingFixer struct{}

func (f ControlStructureSpacingFixer) Code() string { return controlStructureSpacingCode }
func (f ControlStructureSpacingFixer) Fix(content string) string {
	return FixControlStructureSpacing(content)
}

func init() {
	RegisterRule(controlStructureSpacingCode, func(filename string, content []byte, _ []ast.Node) []StyleIssue {
		lines := SplitLinesCached(content)
		checker := NewControlStructureSpacingChecker()
		return checker.CheckIssues(lines, filename)
	})
	RegisterFixer(ControlStructureSpacingFixer{})
}
