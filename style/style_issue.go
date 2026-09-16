package style

type IssueType string

const (
	Error   IssueType = "ERROR"
	Warning IssueType = "WARNING"
)

type StyleIssue struct {
	Filename string
	Line     int
	Column   int
	// EndLine and EndColumn optionally carry the end of the offending
	// source span, forming a half-open [Line:Column, EndLine:EndColumn)
	// range for editor squiggly-underline diagnostics. Zero end fields
	// mean "point at start only" (same convention as analyse.AnalysisIssue
	// and parser.ParseError).
	EndLine   int
	EndColumn int
	Type      IssueType // ERROR or WARNING
	Fixable     bool      // true if autofix is possible
	Message     string
	Code        string // e.g. PEAR.Commenting.FileComment.Missing
	SubjectKind string
	SubjectName string
}
