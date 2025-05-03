package psr12

import "go-phpcs/style"

// EndFileNewlineChecker checks that files end with a single blank line (PSR-12 2.2)
type EndFileNewlineChecker struct{}

func (c *EndFileNewlineChecker) CheckIssues(lines []string, filename string) []style.StyleIssue {
	var issues []style.StyleIssue
	if len(lines) == 0 {
		return issues
	}

	// Check for a single blank line at the end
	lastIdx := len(lines) - 1
	secondLastIdx := len(lines) - 2
	last := lines[lastIdx]
	secondLast := ""
	if secondLastIdx >= 0 {
		secondLast = lines[secondLastIdx]
	}

	// PSR-12: Files must end with a single blank line
	if last != "" || secondLast == "" || (secondLastIdx > 0 && lines[secondLastIdx-1] == "") {
		issues = append(issues, style.StyleIssue{
			Filename: filename,
			Line:     len(lines),
			Type:     style.Error,
			Fixable:  true,
			Message:  "File must end with a single blank line",
			Code:     "PSR12.Files.EndFileNewline",
		})
	}
	return issues
}

func init() {
	RegisterPSR12Rule("PSR12.Files.EndFileNewline", func(filename string, content []byte) []style.StyleIssue {
		if len(content) == 0 || content[len(content)-1] != '\n' {
			return []style.StyleIssue{{
				Filename: filename,
				Line:     0,
				Type:     style.Error,
				Fixable:  true,
				Message:  "File must end with a single blank line (newline)",
				Code:     "PSR12.Files.EndFileNewline",
			}}
		}
		return nil
	})
}
