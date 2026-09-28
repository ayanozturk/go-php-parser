package diag

import (
	"bytes"
	"unicode/utf16"
	"unicode/utf8"
)

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

// ByteSpanFromRunePositions converts one-based rune line and column
// coordinates to half-open UTF-8 byte offsets. Invalid starts map to the
// source start; invalid, absent, or reversed ends produce a zero-width span.
func ByteSpanFromRunePositions(source []byte, startLine, startColumn, endLine, endColumn int) ByteSpan {
	start, ok := byteOffsetFromRunePosition(source, startLine, startColumn)
	if !ok {
		return ByteSpan{}
	}
	end := start
	if endLine > 0 && endColumn > 0 {
		if candidate, valid := byteOffsetFromRunePosition(source, endLine, endColumn); valid && candidate >= start {
			end = candidate
		}
	}
	return ByteSpan{Start: start, End: end}
}

// UTF16Position is an LSP-compatible, zero-based position. It lives here as
// a protocol-neutral coordinate pair so every transport uses the same byte
// span conversion rules.
type UTF16Position struct {
	Line      uint32
	Character uint32
}

// UTF16PositionFromByteOffset converts a UTF-8 byte offset to a zero-based
// line and UTF-16 code-unit column. Out-of-range offsets are clamped; offsets
// inside a multi-byte rune are rounded down to its start.
func UTF16PositionFromByteOffset(source []byte, offset int) UTF16Position {
	if offset < 0 {
		offset = 0
	}
	if offset > len(source) {
		offset = len(source)
	}
	line, lineStart := 0, 0
	for i := 0; i < offset; i++ {
		if source[i] == '\n' {
			line++
			lineStart = i + 1
		}
	}
	column := 0
	for i := lineStart; i < offset; {
		if source[i] == '\r' && i+1 == offset && offset < len(source) && source[offset] == '\n' {
			break
		}
		r, size := utf8.DecodeRune(source[i:])
		if size < 1 || i+size > offset {
			break
		}
		column += utf16.RuneLen(r)
		i += size
	}
	return UTF16Position{Line: uint32(line), Character: uint32(column)}
}

func byteOffsetFromRunePosition(source []byte, line, column int) (int, bool) {
	if line < 1 || column < 1 {
		return 0, false
	}
	lineStart := 0
	currentLine := 1
	for currentLine < line {
		rel := bytes.IndexByte(source[lineStart:], '\n')
		if rel < 0 {
			return 0, false
		}
		lineStart += rel + 1
		currentLine++
	}
	lineEnd := len(source)
	if rel := bytes.IndexByte(source[lineStart:], '\n'); rel >= 0 {
		lineEnd = lineStart + rel
	}
	if lineEnd > lineStart && source[lineEnd-1] == '\r' {
		lineEnd--
	}
	target := column - 1
	for offset, count := lineStart, 0; ; {
		if count == target {
			return offset, true
		}
		if offset >= lineEnd {
			return 0, false
		}
		_, size := utf8.DecodeRune(source[offset:lineEnd])
		if size < 1 {
			return 0, false
		}
		offset += size
		count++
	}
}
