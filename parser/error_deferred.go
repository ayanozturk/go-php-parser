package parser

import "fmt"

// ErrorDeferred represents a deferred error message for the parser.
type ErrorDeferred struct {
	Format string
	Args   []interface{}
}

// Error implements the error interface, formatting only when needed.
func (e ErrorDeferred) Error() string {
	return fmt.Sprintf(e.Format, e.Args...)
}
