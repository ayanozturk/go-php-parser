package psr12

import (
	"go-phpcs/style"
)

// NoMultipleStatementsPerLineChecker checks that there is at most one statement per line (PSR-12 2.3)
type NoMultipleStatementsPerLineChecker struct{}

func (c *NoMultipleStatementsPerLineChecker) CheckIssues(lines []string, filename string) []style.StyleIssue {
	var issues []style.StyleIssue
	inBlockComment := false
	inHeredoc := false
	heredocEnd := ""
	for i, line := range lines {
		trimmed := line
		if len(trimmed) == 0 {
			continue // skip empty lines
		}
		// Only skip if the line is a pure comment
		ltrim := len(trimmed)
		if ltrim > 1 && (trimmed[0:2] == "//" || trimmed[0:1] == "#") {
			continue
		}
		count := 0
		inSingle := false
		inDouble := false
		j := 0
		for j < len(line) {
			if inBlockComment {
				if j+1 < len(line) && line[j] == '*' && line[j+1] == '/' {
					inBlockComment = false
					j += 2
					continue
				}
				j++
				continue
			}
			if inHeredoc {
				// Heredoc/nowdoc ends only if the line (after optional whitespace) matches the marker
				k := 0
				for k < len(line) && (line[k] == ' ' || line[k] == '\t') { k++ }
				if heredocEnd != "" && line[k:] == heredocEnd {
					inHeredoc = false
					heredocEnd = ""
				}
				break // skip the rest of this line
			}
			if !inSingle && !inDouble {
				// Line comments
				if j+1 < len(line) && line[j] == '/' && line[j+1] == '/' {
					break
				}
				if line[j] == '#' {
					break
				}
				// Block comment start
				if j+1 < len(line) && line[j] == '/' && line[j+1] == '*' {
					inBlockComment = true
					j += 2
					continue
				}
				// Heredoc/Nowdoc start
				if j+2 < len(line) && line[j] == '<' && line[j+1] == '<' && line[j+2] == '<' {
					k := j + 3
					for k < len(line) && (line[k] == ' ' || line[k] == '\t') { k++ }
					start := k
					for k < len(line) && ((line[k] >= 'a' && line[k] <= 'z') || (line[k] >= 'A' && line[k] <= 'Z') || (line[k] >= '0' && line[k] <= '9') || line[k] == '_' || line[k] == '\'' || line[k] == '"') { k++ }
					heredocEnd = line[start:k]
					if heredocEnd != "" {
						inHeredoc = true
						j = k
						break // rest of line is part of heredoc start
					}
				}
			}
			if !inDouble && line[j] == '\'' {
				inSingle = !inSingle
				j++
				continue
			}
			if !inSingle && line[j] == '"' {
				inDouble = !inDouble
				j++
				continue
			}
			if line[j] == '\\' { // skip escaped quotes
				j += 2
				continue
			}
			if !inSingle && !inDouble && !inBlockComment && !inHeredoc && line[j] == ';' {
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
				Code:     "PSR12.Files.NoMultipleStatementsPerLine",
			})
		}
	}
	return issues
}
