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
		count := s.countStatements(line, commentState)
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

func (s *DisallowMultipleStatementsSniff) countStatements(line string, commentState *helper.CommentState) int {
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
		// Heredoc start
		j2 = helper.HandleHeredocStart(line, j, commentState)
		if j2 != j {
			break
		}
		if commentState.InHeredoc {
			break
		}
		// Line comment
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
		if s.isStatementSeparator(line, j, qs, commentState) {
			count++
		}
		j++
	}
	return count
}

func (s *DisallowMultipleStatementsSniff) isStatementSeparator(line string, j int, qs *helper.QuoteState, commentState *helper.CommentState) bool {
	return !qs.InSingle && !qs.InDouble && !commentState.InBlockComment && !commentState.InHeredoc && line[j] == ';'
}
