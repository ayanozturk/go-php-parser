package diag

// Severity is the normalized severity of a diagnostic, independent of any
// output format such as CLI text or LSP.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
	SeverityHint    Severity = "hint"
)

// ByteSpan is a half-open source range measured in UTF-8 byte offsets.
type ByteSpan struct {
	Start int
	End   int
}

// Diagnostic is the shared, transport-neutral diagnostic representation.
// Coordinates remain byte based until an output adapter converts them for its
// protocol (for example, LSP's zero-based UTF-16 positions).
type Diagnostic struct {
	Filename string
	Source   string
	Code     string
	Severity Severity
	Message  string
	Span     ByteSpan
}
