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
		if s.handleBlockComment(line, &j, commentState) {
			continue
		}
		if commentState.InBlockComment {
			j++
			continue
		}
		if s.handleHeredocStart(line, &j, commentState) {
			break
		}
		if commentState.InHeredoc {
			break
		}
		if s.handleLineComment(line, &j, qs) {
			break
		}
		if s.handleQuotes(line, &j, qs) {
			continue
		}
		if !qs.InSingle && !qs.InDouble && !commentState.InBlockComment && !commentState.InHeredoc && line[j] == ';' {
			count++
		}
		j++
	}
	return count
}

func (s *DisallowMultipleStatementsSniff) handleBlockComment(line string, j *int, commentState *helper.CommentState) bool {
	j2 := helper.HandleBlockComment(line, *j, commentState)
	if j2 != *j {
		*j = j2
		return true
	}
	return false
}

func (s *DisallowMultipleStatementsSniff) handleHeredocStart(line string, j *int, commentState *helper.CommentState) bool {
	j2 := helper.HandleHeredocStart(line, *j, commentState)
	if j2 != *j {
		return true
	}
	return false
}

func (s *DisallowMultipleStatementsSniff) handleLineComment(line string, j *int, qs *helper.QuoteState) bool {
	if !qs.InSingle && !qs.InDouble {
		if *j+1 < len(line) && line[*j] == '/' && line[*j+1] == '/' {
			return true
		}
		if line[*j] == '#' {
			return true
		}
	}
	return false
}

func (s *DisallowMultipleStatementsSniff) handleQuotes(line string, j *int, qs *helper.QuoteState) bool {
	j2 := helper.HandleQuotes(line, *j, qs)
	if j2 != *j {
		*j = j2
		return true
	}
	return false
}
