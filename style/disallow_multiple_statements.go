package style

import (
	"go-phpcs/ast"
	"go-phpcs/style/helper"
	"strings"
)

const disallowMultipleStatementsCode = "Generic.Formatting.DisallowMultipleStatements"

type DisallowMultipleStatementsSniff struct{}

func (s *DisallowMultipleStatementsSniff) CheckIssues(lines []string, filename string) []StyleIssue {
	var issues []StyleIssue
	commentState := &helper.CommentState{}
	for i, line := range lines {
		if helper.SkipLineComment(line) {
			continue
		}
		if helper.HandleHeredocEnd(line, commentState) {
			continue
		}
		// Fast pre-check: skip lines with 0 or 1 semicolon
		if strings.Count(line, ";") <= 1 {
			continue
		}
		count := s.countStatements(line, commentState)
		if count > 1 {
			issues = append(issues, StyleIssue{
				Filename: filename,
				Line:     i + 1,
				Type:     Error,
				Fixable:  false,
				Message:  "Multiple statements detected on the same line",
				Code:     disallowMultipleStatementsCode,
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
		if s.skipOrHandleBlockComment(line, &j, commentState) {
			continue
		}
		if s.skipOrHandleHeredoc(line, &j, commentState) {
			break
		}
		if s.skipLineComment(line, j, qs) {
			break
		}
		if s.handleQuotes(line, &j, qs) {
			continue
		}
		if s.isStatementSeparator(line, j, qs, commentState) {
			count++
		}
		j++
	}
	return count
}

func (s *DisallowMultipleStatementsSniff) skipOrHandleBlockComment(line string, j *int, commentState *helper.CommentState) bool {
	j2 := helper.HandleBlockComment(line, *j, commentState)
	if j2 != *j {
		*j = j2
		return true
	}
	if commentState.InBlockComment {
		*j++
		return true
	}
	return false
}

func (s *DisallowMultipleStatementsSniff) skipOrHandleHeredoc(line string, j *int, commentState *helper.CommentState) bool {
	j2 := helper.HandleHeredocStart(line, *j, commentState)
	if j2 != *j {
		return true
	}
	if commentState.InHeredoc {
		return true
	}
	return false
}

func (s *DisallowMultipleStatementsSniff) skipLineComment(line string, j int, qs *helper.QuoteState) bool {
	if !qs.InSingle && !qs.InDouble {
		if j+1 < len(line) && line[j] == '/' && line[j+1] == '/' {
			return true
		}
		if line[j] == '#' {
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

func (s *DisallowMultipleStatementsSniff) isStatementSeparator(line string, j int, qs *helper.QuoteState, commentState *helper.CommentState) bool {
	return !qs.InSingle && !qs.InDouble && !commentState.InBlockComment && !commentState.InHeredoc && line[j] == ';'
}

func init() {
	RegisterRule(disallowMultipleStatementsCode, func(filename string, content []byte, _ []ast.Node) []StyleIssue {
		lines := strings.Split(string(content), "\n")
		sniff := &DisallowMultipleStatementsSniff{}
		return sniff.CheckIssues(lines, filename)
	})
}
