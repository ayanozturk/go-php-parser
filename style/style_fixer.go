package style

// StyleFixer defines the interface for autofixable style rules.
type StyleFixer interface {
	Code() string
	Fix(content string) string
}

var fixerRegistry = map[string]StyleFixer{}

// RegisterFixer registers a StyleFixer for a rule code.
func RegisterFixer(fixer StyleFixer) {
	fixerRegistry[fixer.Code()] = fixer
}

// GetFixer returns the StyleFixer for a rule code, or nil if not found.
func GetFixer(code string) StyleFixer {
	return fixerRegistry[code]
}
