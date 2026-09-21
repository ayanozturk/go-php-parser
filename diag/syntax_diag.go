package diag

import (
	"unicode/utf8"

	"github.com/ayanozturk/go-php-parser/token"
)

// ParseErrorFromOffsets builds a structured ParseError from a byte span and
// bare message. Line/column use 1-based rune coordinates matching the lexer.
//
// Building a fresh LineTable is an O(file size) scan; a caller converting
// several diagnostics for the same file should use ParseErrorFromOffsetsWithLines
// instead and build the table once, not once per diagnostic.
func ParseErrorFromOffsets(src []byte, start, end int, message string) ParseError {
	return ParseErrorFromOffsetsWithLines(src, token.NewLineTable(src), start, end, message)
}

// ParseErrorFromOffsetsWithLines is ParseErrorFromOffsets for a caller that
// already has (or is converting multiple diagnostics against) the same
// file's LineTable, avoiding an O(file size) rebuild per diagnostic.
func ParseErrorFromOffsetsWithLines(src []byte, lines token.LineTable, start, end int, message string) ParseError {
	if end < start {
		end = start
	}
	bare, _, _ := stripLeadingLineCol(message)
	startLine, startCol := offsetRuneLineCol(src, lines, start)
	endLine, endCol := offsetRuneLineCol(src, lines, end)
	return ParseError{
		Line:      startLine,
		Column:    startCol,
		Offset:    start,
		EndLine:   endLine,
		EndColumn: endCol,
		EndOffset: end,
		Code:      ClassifyParseError(bare),
		Message:   bare,
	}
}

func offsetRuneLineCol(src []byte, lines token.LineTable, offset int) (line, col int) {
	if offset < 0 {
		offset = 0
	}
	if offset > len(src) {
		offset = len(src)
	}
	line0 := 0
	if len(lines) > 0 {
		lo, hi := 0, len(lines)-1
		for lo <= hi {
			mid := (lo + hi) / 2
			if lines[mid] <= offset {
				lo = mid + 1
			} else {
				hi = mid - 1
			}
		}
		line0 = hi
		if line0 < 0 {
			line0 = 0
		}
	}
	lineStart := 0
	if line0 < len(lines) {
		lineStart = lines[line0]
	}
	if lineStart > offset {
		lineStart = offset
	}
	runeCol := utf8.RuneCount(src[lineStart:offset])
	return line0 + 1, runeCol + 1
}
