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
	Type     IssueType // ERROR or WARNING
	Fixable  bool      // true if autofix is possible
	Message  string
	Code     string // e.g. PEAR.Commenting.FileComment.Missing
}
