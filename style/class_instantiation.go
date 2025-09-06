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
	lines := SplitLinesCached(content)
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

	skipSpacesAndInlineComments := func(line string, idx int) int {
		i := idx
		for i < len(line) {
			// skip spaces/tabs
			for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
				i++
			}
			// inline block comment /* ... */
			if i+1 < len(line) && line[i] == '/' && line[i+1] == '*' {
				end := i + 2
				for end+1 < len(line) && !(line[end] == '*' && line[end+1] == '/') {
					end++
				}
				if end+1 >= len(line) {
					return len(line)
				}
				i = end + 2
				continue
			}
			break
		}
		return i
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
				pos := skipSpacesAndInlineComments(line, j)
				if pos >= len(line) {
					break
				}
				if line[pos] == '(' {
					p = pending{}
					j = pos + 1
					continue
				}
				if line[pos] == '/' && pos+1 < len(line) && line[pos+1] == '/' {
					break
				}
				// any other significant token -> issue
				issues = append(issues, StyleIssue{Filename: filename, Line: p.line, Column: p.col, Type: Error, Fixable: true, Message: "Missing parentheses for class instantiation", Code: psr1ClassInstantiationCode})
				p = pending{}
				j = pos + 1
				continue
			}

			// Detect "new ClassName" without immediate parentheses
			if j+3 < len(line) && line[j:j+3] == "new" && (j == 0 || isWordBoundary(line[j-1])) {
				k := j + 3
				for k < len(line) && (line[k] == ' ' || line[k] == '\t') {
					k++
				}
				// Anonymous class: new class ... -> ignore
				if strings.HasPrefix(line[k:], "class") && (k+5 >= len(line) || isWordBoundary(line[k+5])) {
					j = k + 5
					continue
				}
				startName := k
				for k < len(line) && (unicode.IsLetter(rune(line[k])) || unicode.IsDigit(rune(line[k])) || line[k] == '_' || line[k] == '\\') {
					k++
				}
				if startName < k {
					// Found a likely class name; now check for whitespace/comments then '('
					n := skipSpacesAndInlineComments(line, k)
					if n >= len(line) {
						// defer check to next line
						p = pending{active: true, line: i + 1, col: j + 1}
						break
					}
					if line[n] != '(' {
						issues = append(issues, StyleIssue{Filename: filename, Line: i + 1, Column: j + 1, Type: Error, Fixable: true, Message: "Missing parentheses for class instantiation", Code: psr1ClassInstantiationCode})
					}
				}
				j = k
				continue
			}

			j++
		}
	}

	return issues
}

func init() {
	RegisterRule(psr1ClassInstantiationCode, func(filename string, content []byte, _ []ast.Node) []StyleIssue {
		sniff := &ClassInstantiationSniff{}
		return sniff.CheckIssues(content, filename)
	})
}
