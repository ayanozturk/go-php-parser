package parser

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// ParseError is a structured syntax/recovery diagnostic emitted during parsing.
// Coordinates are one-based rune line/column with byte Offset, matching
// token.Position and ast.Position. End* fields form a half-open [start, end)
// span; zero end fields mean "point at start only" (same convention as
// analyse.AnalysisIssue).
//
// Errors() continues to return Error() strings so PHP Strom's line N:C: parser
// remains compatible during migration to StructuredErrors().
type ParseError struct {
	Line      int
	Column    int
	Offset    int
	EndLine   int
	EndColumn int
	EndOffset int
	Code      string
	Message   string // human text without a leading "line N:C:" prefix
}

// Error implements the error interface with the legacy string form editors parse.
func (e ParseError) Error() string {
	if e.Line > 0 && e.Column > 0 {
		return fmt.Sprintf("line %d:%d: %s", e.Line, e.Column, e.Message)
	}
	return e.Message
}

// stripLeadingLineCol removes a leading "line N:C: " prefix from a formatted
// parser message. When absent, line and column are zero.
func stripLeadingLineCol(msg string) (bare string, line, col int) {
	const prefix = "line "
	if !strings.HasPrefix(msg, prefix) {
		return msg, 0, 0
	}
	rest := msg[len(prefix):]
	i := 0
	for i < len(rest) && unicode.IsDigit(rune(rest[i])) {
		i++
	}
	if i == 0 || i >= len(rest) || rest[i] != ':' {
		return msg, 0, 0
	}
	line, err1 := strconv.Atoi(rest[:i])
	rest = rest[i+1:]
	j := 0
	for j < len(rest) && unicode.IsDigit(rune(rest[j])) {
		j++
	}
	if j == 0 || j >= len(rest) || rest[j] != ':' {
		return msg, 0, 0
	}
	col, err2 := strconv.Atoi(rest[:j])
	if err1 != nil || err2 != nil {
		return msg, 0, 0
	}
	bare = strings.TrimSpace(rest[j+1:])
	return bare, line, col
}

// ClassifyParseError maps a bare diagnostic message to a stable Parser.* code.
func ClassifyParseError(msg string) string {
	return classifyParseError(msg)
}

func classifyParseError(msg string) string {
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "expected <?php"):
		return "Parser.MissingOpenTag"
	case strings.HasPrefix(msg, "Parser panic:"),
		strings.Contains(lower, "context cancelled"):
		return "Parser.Internal"
	case strings.Contains(lower, "expected"):
		return "Parser.ExpectedToken"
	default:
		return "Parser.Syntax"
	}
}
