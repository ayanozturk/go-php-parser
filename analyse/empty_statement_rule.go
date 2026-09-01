package analyse

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/sharedcache"
	"github.com/ayanozturk/go-php-parser/style/helper"
	"strings"
)

// EmptyStatementRule detects superfluous empty statements (standalone semicolons)
// and control structures with an immediate semicolon body (e.g., if (cond);).
// It operates primarily on source text for robustness, while remaining isolated via a helper entrypoint for tests.
type EmptyStatementRule struct{}

// CheckIssuesWithSource performs analysis given explicit source content (used by tests for isolation).
func (r *EmptyStatementRule) CheckIssuesWithSource(filename string, content []byte, _ []ast.Node) []AnalysisIssue {
	source := bytesAsString(content)
	var issues []AnalysisIssue

	commentState := &helper.CommentState{}
	qs := &helper.QuoteState{}

	// Track potential control header immediately followed by a semicolon.
	type pendingCtrl struct {
		keyword       string
		line          int
		colAfterParen int // column position right after the matching ')'
		active        bool
	}
	var ctrl pendingCtrl

	// Track whether we are inside a for-control parentheses to ignore its internal semicolons.
	inForControl := false
	forControlDepth := 0

	// Reset per-statement segment tracking at line start and after each semicolon.
	hasCodeSinceBoundary := false

	lineStart := 0
	i := 0
	for lineStart <= len(source) {
		lineEnd := lineStart
		for lineEnd < len(source) && source[lineEnd] != '\n' {
			lineEnd++
		}
		line := source[lineStart:lineEnd]
		if n := len(line); n > 0 && line[n-1] == '\r' {
			line = line[:n-1]
		}
		hasCodeSinceBoundary = false
		if !commentState.InBlockComment && !commentState.InHeredoc && !ctrl.active && !emptyStatementLineNeedsScan(line) {
			if lineEnd == len(source) {
				break
			}
			lineStart = lineEnd + 1
			i++
			continue
		}
		*qs = helper.QuoteState{}

		j := 0
		for j < len(line) {
			// Handle/skip block comments spanning lines
			j2 := helper.HandleBlockComment(line, j, commentState)
			if j2 != j {
				j = j2
				continue
			}
			if commentState.InBlockComment {
				j++
				continue
			}
			// Handle heredoc state (we do not analyze inside heredoc lines)
			j2 = helper.HandleHeredocStart(line, j, commentState)
			if j2 != j {
				// heredoc start found; skip rest of this line
				break
			}
			if commentState.InHeredoc {
				// ignore content until heredoc end is found by outer caller in next lines
				break
			}
			// Skip end-of-line comments
			if !qs.InSingle && !qs.InDouble {
				if j+1 < len(line) && line[j] == '/' && line[j+1] == '/' {
					break
				}
				if line[j] == '#' {
					break
				}
			}

			// Handle quotes
			j2 = helper.HandleQuotes(line, j, qs)
			if j2 != j {
				j = j2
				continue
			}

			if qs.InSingle || qs.InDouble {
				j++
				continue
			}

			ch := line[j]

			// Recognize control keywords if/for/while followed by '(' (word boundaries)
			if !inForControl && !ctrl.active {
				// try "for"
				if hasKeywordAtASCII(line, j, "for") {
					k := j + 3
					for k < len(line) && isASCIISpace(line[k]) {
						k++
					}
					if k < len(line) && line[k] == '(' {
						inForControl = true
						forControlDepth = 1
						j = k + 1
						hasCodeSinceBoundary = true // there's code before first ';' inside control
						continue
					}
				}
				// try "if" / "while"
				if hasKeywordAtASCII(line, j, "if") {
					k := j + 2
					for k < len(line) && isASCIISpace(line[k]) {
						k++
					}
					if k < len(line) && line[k] == '(' {
						// track until we hit ')' on same line; multi-line headers not handled here
						depth := 1
						k++
						for k < len(line) && depth > 0 {
							if line[k] == '(' {
								depth++
							} else if line[k] == ')' {
								depth--
							}
							k++
						}
						if depth == 0 {
							ctrl = pendingCtrl{keyword: "if", line: i + 1, colAfterParen: k + 1, active: true}
							j = k
							hasCodeSinceBoundary = true
							continue
						}
					}
				}
				if hasKeywordAtASCII(line, j, "while") {
					k := j + 5
					for k < len(line) && isASCIISpace(line[k]) {
						k++
					}
					if k < len(line) && line[k] == '(' {
						depth := 1
						k++
						for k < len(line) && depth > 0 {
							if line[k] == '(' {
								depth++
							} else if line[k] == ')' {
								depth--
							}
							k++
						}
						if depth == 0 {
							keyword := "while"
							if isDoWhileHeader(line, j) {
								keyword = "do-while"
							}
							ctrl = pendingCtrl{keyword: keyword, line: i + 1, colAfterParen: k + 1, active: true}
							j = k
							hasCodeSinceBoundary = true
							continue
						}
					}
				}
			}

			// Track parens depth for for-control to ignore its internal semicolons
			if inForControl {
				if ch == '(' {
					forControlDepth++
				} else if ch == ')' {
					forControlDepth--
					if forControlDepth == 0 {
						inForControl = false
						// After finishing for-header, mark pending control like others
						ctrl = pendingCtrl{keyword: "for", line: i + 1, colAfterParen: j + 2, active: true}
					}
				}
				j++
				continue
			}

			// After control paren, if we encounter non-space code before ';' or '{', cancel pending
			if ctrl.active {
				if isASCIISpace(ch) {
					// keep waiting
				} else if ch == '{' {
					ctrl.active = false // has body, not empty
				} else if ch == ';' {
					if ctrl.keyword != "do-while" {
						// Empty control statement
						issues = append(issues, AnalysisIssue{
							Filename: filename,
							Line:     i + 1,
							Column:   j + 1,
							Code:     "Generic.CodeAnalysis.EmptyStatement",
							Message:  "Empty statement detected",
						})
					}
					ctrl.active = false
				} else {
					// Some code appears; cancel pending
					ctrl.active = false
				}
			}

			// Generic empty statement: semicolon with no code since last boundary
			if ch == ';' {
				if !hasCodeSinceBoundary {
					issues = append(issues, AnalysisIssue{
						Filename: filename,
						Line:     i + 1,
						Column:   j + 1,
						Code:     "Generic.CodeAnalysis.EmptyStatement",
						Message:  "Empty statement detected",
					})
				}
				// reset for next segment
				hasCodeSinceBoundary = false
				j++
				continue
			}

			// Any other visible character counts as code (outside comments/strings)
			if !isASCIISpace(ch) {
				hasCodeSinceBoundary = true
			}
			j++
		}

		// Handle heredoc end state per line
		_ = helper.HandleHeredocEnd(line, commentState)
		if lineEnd == len(source) {
			break
		}
		lineStart = lineEnd + 1
		i++
	}

	return issues
}

func emptyStatementLineNeedsScan(line string) bool {
	if strings.IndexByte(line, ';') >= 0 {
		return true
	}
	if strings.Contains(line, "/*") || strings.Contains(line, "<<<") {
		return true
	}
	return containsEmptyStatementControlCandidate(line)
}

func containsEmptyStatementControlCandidate(line string) bool {
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case 'f', 'F':
			if hasKeywordAtASCII(line, i, "for") {
				return true
			}
		case 'i', 'I':
			if hasKeywordAtASCII(line, i, "if") {
				return true
			}
		case 'w', 'W':
			if hasKeywordAtASCII(line, i, "while") {
				return true
			}
		}
	}
	return false
}

func isDoWhileHeader(line string, whilePos int) bool {
	i := whilePos - 1
	for i >= 0 && isASCIISpace(line[i]) {
		i--
	}
	return i >= 0 && line[i] == '}'
}

func hasKeywordAtASCII(line string, start int, keyword string) bool {
	end := start + len(keyword)
	if end > len(line) {
		return false
	}
	if start > 0 && isEmptyStatementIdentByte(line[start-1]) {
		return false
	}
	if end < len(line) && isEmptyStatementIdentByte(line[end]) {
		return false
	}
	for i := 0; i < len(keyword); i++ {
		ch := line[start+i]
		if ch >= 'A' && ch <= 'Z' {
			ch += 'a' - 'A'
		}
		if ch != keyword[i] {
			return false
		}
	}
	return true
}

func isASCIISpace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == '\v' || ch == '\f'
}

func isEmptyStatementIdentByte(ch byte) bool {
	return ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')
}

// CheckIssues reads the source file and delegates to CheckIssuesWithSource.
func (r *EmptyStatementRule) CheckIssues(nodes []ast.Node, filename string) []AnalysisIssue {
	content, err := sharedcache.GetCachedFileContent(filename)
	if err != nil {
		// Fail closed: if we cannot read file, do not report issues for this rule
		return nil
	}
	return r.CheckIssuesWithSource(filename, content, nodes)
}

func init() {
	RegisterAnalysisRule("Generic.CodeAnalysis.EmptyStatement", func(filename string, nodes []ast.Node) []AnalysisIssue {
		rule := &EmptyStatementRule{}
		return rule.CheckIssues(nodes, filename)
	})
}
