package functions

import (
	"go-phpcs/style"
	"go-phpcs/style/helper"
)

// DisallowMultipleStatementsSniff checks that there is at most one statement per line (Generic.Formatting.DisallowMultipleStatements)
type DisallowMultipleStatementsSniff struct{}

func (s *DisallowMultipleStatementsSniff) CheckIssues(lines []string, filename string) []style.StyleIssue {
	var issues []style.StyleIssue
	commentState := &helper.CommentState{}
	for i, line := range lines {
		if helper.SkipLineComment(line) {
			continue
		}
		if helper.HandleHeredocEnd(line, commentState) {
			continue
		}
		count := 0
		qs := &helper.QuoteState{}
		j := 0
		for j < len(line) {
			// Block comment
			j2 := helper.HandleBlockComment(line, j, commentState)
			if j2 != j {
				j = j2
				continue
			}
			if commentState.InBlockComment {
				j++
				continue
			}
			// Heredoc/Nowdoc start
			j2 = helper.HandleHeredocStart(line, j, commentState)
			if j2 != j {
				break // rest of line is part of heredoc start
			}
			if commentState.InHeredoc {
				break // skip the rest of this line
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
			j2 = helper.HandleQuotes(line, j, qs)
			if j2 != j {
				j = j2
				continue
			}
			if !qs.InSingle && !qs.InDouble && !commentState.InBlockComment && !commentState.InHeredoc && line[j] == ';' {
				count++
			}
			j++
		}
		if count > 1 {
			issues = append(issues, style.StyleIssue{
				Filename: filename,
				Line:     i + 1,
				Type:     style.Error,
				Fixable:  false,
				Message:  "Multiple statements detected on the same line",
				Code:     "Generic.Formatting.DisallowMultipleStatements",
			})
		}
	}
	return issues
}
