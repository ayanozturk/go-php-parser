package style

// import "io"

// IssueCollector implements io.Writer and collects style issues written as PHPCS lines.
type IssueCollector struct {
	Issues *[]StyleIssue
}

func (c *IssueCollector) Write(p []byte) (n int, err error) {
	// This is a stub: the actual implementation should parse PHPCS lines to StyleIssue, but
	// in our code, we call PrintPHPCSStyleIssueToWriter, so instead, we should just append manually.
	return len(p), nil
}

// Append implements a custom method for adding issues directly.
func (c *IssueCollector) Append(issue StyleIssue) {
	if c.Issues != nil {
		*c.Issues = append(*c.Issues, issue)
	}
}
