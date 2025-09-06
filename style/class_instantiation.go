package style

import (
	"go-phpcs/ast"
	"go-phpcs/style/helper"
	"strings"
	"unicode"
)

const psr1ClassInstantiationCode = "PSR1.Classes.ClassInstantiation"

// ClassInstantiationSniff ensures `new ClassName()` uses parentheses per PSR recommendations.
// It scans source text to catch cases the parser would reject or normalize.
type ClassInstantiationSniff struct{}

func (s *ClassInstantiationSniff) CheckIssues(content []byte, filename string) []StyleIssue {
	lines := strings.Split(string(content), "\n")
	var issues []StyleIssue

	commentState := &helper.CommentState{}
	qs := &helper.QuoteState{}

	// State for multi-line "new Foo" followed by parentheses on next line
	type pending struct {
		active bool
		line   int
		col    int // column of 'n' in new for reporting
	}
	p := pending{}

	isWordBoundary := func(b byte) bool {
		return !(unicode.IsLetter(rune(b)) || unicode.IsDigit(rune(b)) || b == '_')
	}

	for i, line := range lines {
		// reset quote state each line; block comments/heredoc persist via commentState
		qs = &helper.QuoteState{}

		j := 0
		for j < len(line) {
			// Handle/skip block comments
			j2 := helper.HandleBlockComment(line, j, commentState)
			if j2 != j {
				j = j2
				continue
			}
			if commentState.InBlockComment {
				j++
				continue
			}
			// Handle heredoc start
			j2 = helper.HandleHeredocStart(line, j, commentState)
			if j2 != j {
				break
			}
			if commentState.InHeredoc {
				break
			}
			// Line comments
			if !qs.InSingle && !qs.InDouble {
				if j+1 < len(line) && line[j] == '/' && line[j+1] == '/' {
					break
				}
				if line[j] == '#' {
					break
				}
			}
			// Quotes
			j2 = helper.HandleQuotes(line, j, qs)
			if j2 != j {
				j = j2
				continue
			}
			if qs.InSingle || qs.InDouble {
				j++
				continue
			}

			// If we had a pending new ... ClassName and now see a significant char
			if p.active {
				if unicode.IsSpace(rune(line[j])) {
					// wait for next significant char
					j++
					continue
				}
				if line[j] == '(' {
					// parentheses present across lines; ok
					p.active = false
					j++
					continue
				}
				// If next is ';' or any other non-'(' significant char, this is a violation
				issues = append(issues, StyleIssue{
					Filename: filename,
					Line:     p.line,
					Column:   p.col,
					Type:     Error,
					Fixable:  false,
					Message:  "Class instantiation must use parentheses",
					Code:     psr1ClassInstantiationCode,
				})
				p.active = false
				// continue scanning current position without consuming
			}

			// Detect "new" keyword at word boundaries
			if j+3 <= len(line) && strings.EqualFold(line[j:j+3], "new") && (j == 0 || isWordBoundary(line[j-1])) && (j+3 == len(line) || isWordBoundary(line[j+3])) {
				// Remember start for reporting
				startCol := j + 1
				k := j + 3
				// skip whitespace
				for k < len(line) && unicode.IsSpace(rune(line[k])) {
					k++
				}
				// If next word is "class", treat as anonymous class and skip
				if k+5 <= len(line) && strings.EqualFold(line[k:k+5], "class") && (k == 0 || isWordBoundary(line[k-1])) && (k+5 == len(line) || isWordBoundary(line[k+5])) {
					j = k + 5
					continue
				}
				// parse FQCN: leading \? and segments of [A-Za-z0-9_]
				if k < len(line) && line[k] == '\\' {
					k++
				}
				segStart := k
				for k < len(line) {
					c := line[k]
					if c == '\\' {
						// require at least one char in segment
						if k == segStart {
							break
						}
						k++
						segStart = k
						continue
					}
					if unicode.IsLetter(rune(c)) || unicode.IsDigit(rune(c)) || c == '_' {
						k++
						continue
					}
					break
				}
				// No class name parsed
				if k == segStart {
					j++
					continue
				}
				// skip whitespace/comments between name and next token
				j = k
				for j < len(line) {
					// skip spaces
					if unicode.IsSpace(rune(line[j])) {
						j++
						continue
					}
					// skip inline block comments if any
					j3 := helper.HandleBlockComment(line, j, commentState)
					if j3 != j {
						j = j3
						continue
					}
					break
				}
				if j < len(line) && line[j] == '(' {
					// ok
					j++
					continue
				}
				if j >= len(line) {
					// Possibly parentheses on next line; set pending
					p = pending{active: true, line: i + 1, col: startCol}
					continue
				}
				// Not followed by '(' => violation
				issues = append(issues, StyleIssue{
					Filename: filename,
					Line:     i + 1,
					Column:   startCol,
					Type:     Error,
					Fixable:  false,
					Message:  "Class instantiation must use parentheses",
					Code:     psr1ClassInstantiationCode,
				})
				continue
			}

			j++
		}
		// handle heredoc end transitions
		_ = helper.HandleHeredocEnd(line, commentState)
	}

	return issues
}

func init() {
	RegisterRule(psr1ClassInstantiationCode, func(filename string, content []byte, _ []ast.Node) []StyleIssue {
		sniff := &ClassInstantiationSniff{}
		return sniff.CheckIssues(content, filename)
	})
}
