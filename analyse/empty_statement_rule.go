package analyse

import (
	"go-phpcs/ast"
	"go-phpcs/style/helper"
	"os"
	"strings"
	"unicode"
)

// EmptyStatementRule detects superfluous empty statements (standalone semicolons)
// and control structures with an immediate semicolon body (e.g., if (cond);).
// It operates primarily on source text for robustness, while remaining isolated via a helper entrypoint for tests.
type EmptyStatementRule struct{}

// CheckIssuesWithSource performs analysis given explicit source content (used by tests for isolation).
func (r *EmptyStatementRule) CheckIssuesWithSource(filename string, content []byte, _ []ast.Node) []AnalysisIssue {
	lines := strings.Split(string(content), "\n")
	var issues []AnalysisIssue

	commentState := &helper.CommentState{}

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

	for i, line := range lines {
		// Reset segment tracker at start of each new line
		hasCodeSinceBoundary = false
		qs := &helper.QuoteState{}

		// Convenience closures
		isWordBoundary := func(idx int) bool {
			if idx < 0 || idx >= len(line) {
				return true
			}
			return !unicode.IsLetter(rune(line[idx])) && !unicode.IsDigit(rune(line[idx])) && line[idx] != '_'
		}

		// Scan characters of the line
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
				if j+3 <= len(line) && strings.EqualFold(line[j:j+3], "for") && isWordBoundary(j-1) && isWordBoundary(j+3) {
					// Skip whitespace to find '('
					k := j + 3
					for k < len(line) && unicode.IsSpace(rune(line[k])) {
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
				if j+2 <= len(line) && strings.EqualFold(line[j:j+2], "if") && isWordBoundary(j-1) && isWordBoundary(j+2) {
					// find matching ')'
					k := j + 2
					for k < len(line) && unicode.IsSpace(rune(line[k])) {
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
				if j+5 <= len(line) && strings.EqualFold(line[j:j+5], "while") && isWordBoundary(j-1) && isWordBoundary(j+5) {
					k := j + 5
					for k < len(line) && unicode.IsSpace(rune(line[k])) {
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
							ctrl = pendingCtrl{keyword: "while", line: i + 1, colAfterParen: k + 1, active: true}
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
				if unicode.IsSpace(rune(ch)) {
					// keep waiting
				} else if ch == '{' {
					ctrl.active = false // has body, not empty
				} else if ch == ';' {
					// Empty control statement
					issues = append(issues, AnalysisIssue{
						Filename: filename,
						Line:     i + 1,
						Column:   j + 1,
						Code:     "Generic.CodeAnalysis.EmptyStatement",
						Message:  "Empty statement detected",
					})
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
			if !unicode.IsSpace(rune(ch)) {
				hasCodeSinceBoundary = true
			}
			j++
		}

		// Handle heredoc end state per line
		_ = helper.HandleHeredocEnd(line, commentState)
	}

	return issues
}

// CheckIssues reads the source file and delegates to CheckIssuesWithSource.
func (r *EmptyStatementRule) CheckIssues(nodes []ast.Node, filename string) []AnalysisIssue {
	content, err := os.ReadFile(filename)
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
