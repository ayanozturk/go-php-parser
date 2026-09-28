package style

import "github.com/ayanozturk/go-php-parser/diag"

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
	// and diag.ParseError).
	EndLine     int
	EndColumn   int
	Type        IssueType // ERROR or WARNING
	Fixable     bool      // true if autofix is possible
	Message     string
	Code        string // e.g. PEAR.Commenting.FileComment.Missing
	SubjectKind string
	SubjectName string
}

// AsDiagnostic adapts a style issue to the shared representation. Existing
// rule APIs keep their rune-based coordinates; the source text is used here
// to preserve their exact UTF-8 byte range at transport boundaries.
func (i StyleIssue) AsDiagnostic(sourceText []byte, sourceName string) diag.Diagnostic {
	severity := diag.SeverityWarning
	if i.Type == Error {
		severity = diag.SeverityError
	}
	code := i.Code
	if code == "" {
		code = "Style.Unknown"
	}
	return diag.Diagnostic{
		Filename: i.Filename,
		Source:   sourceName,
		Code:     code,
		Severity: severity,
		Message:  i.Message,
		Span: diag.ByteSpanFromRunePositions(
			sourceText, i.Line, i.Column, i.EndLine, i.EndColumn,
		),
	}
}
