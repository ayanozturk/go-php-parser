package diag

import (
	"strings"
	"testing"

	"github.com/ayanozturk/go-php-parser/token"
)

func TestParseErrorError(t *testing.T) {
	e := ParseError{Line: 2, Column: 5, Message: "expected ';'"}
	if e.Error() != "line 2:5: expected ';'" {
		t.Fatalf("Error()=%q", e.Error())
	}
	e = ParseError{Message: "bare"}
	if e.Error() != "bare" {
		t.Fatalf("Error()=%q want bare", e.Error())
	}
	// Line>0 but Column==0 takes the bare-message path.
	e = ParseError{Line: 3, Column: 0, Message: "no column"}
	if e.Error() != "no column" {
		t.Fatalf("Error()=%q want bare message when Column==0", e.Error())
	}
}

func TestClassifyParseError(t *testing.T) {
	cases := []struct {
		msg  string
		code string
	}{
		{"expected <?php", "Parser.MissingOpenTag"},
		{"EXPECTED <?php", "Parser.MissingOpenTag"},
		{"Parser panic: boom", "Parser.Internal"},
		{"parser panic: x", "Parser.Internal"},
		{"context cancelled", "Parser.Internal"},
		{"Context Cancelled", "Parser.Internal"},
		{"expected ';', got '}'", "Parser.ExpectedToken"},
		{"line 1:1: expected ';'", "Parser.ExpectedToken"},
		{"invalid syntax", "Parser.Syntax"},
	}
	for _, c := range cases {
		if got := ClassifyParseError(c.msg); got != c.code {
			t.Fatalf("ClassifyParseError(%q)=%q want %q", c.msg, got, c.code)
		}
	}
}

func TestParseErrorFromOffsets(t *testing.T) {
	src := []byte("<?php\nfoo();\n")
	e := ParseErrorFromOffsets(src, 7, 10, "expected ';'")
	if e.Line != 2 || e.Column != 2 {
		t.Fatalf("start location: line=%d col=%d", e.Line, e.Column)
	}
	if e.EndLine != 2 || e.EndColumn != 5 {
		t.Fatalf("end location: line=%d col=%d", e.EndLine, e.EndColumn)
	}
	if e.Offset != 7 || e.EndOffset != 10 {
		t.Fatalf("offsets: start=%d end=%d", e.Offset, e.EndOffset)
	}
	if e.Code != "Parser.ExpectedToken" {
		t.Fatalf("Code=%q", e.Code)
	}
	if e.Message != "expected ';'" || strings.HasPrefix(e.Message, "line ") {
		t.Fatalf("Message=%q", e.Message)
	}
}

func TestParseErrorFromOffsetsWithLines(t *testing.T) {
	src := []byte("<?php\nfoo();\n")
	lines := token.NewLineTable(src)

	t.Run("parity with FromOffsets", func(t *testing.T) {
		a := ParseErrorFromOffsets(src, 7, 10, "expected ';'")
		b := ParseErrorFromOffsetsWithLines(src, lines, 7, 10, "expected ';'")
		if a != b {
			t.Fatalf("parity mismatch:\n  FromOffsets=%+v\n  WithLines=%+v", a, b)
		}
	})

	t.Run("end less than start", func(t *testing.T) {
		e := ParseErrorFromOffsetsWithLines(src, lines, 10, 7, "expected ';'")
		if e.Offset != 10 {
			t.Fatalf("Offset=%d want raw start 10", e.Offset)
		}
		if e.EndOffset != 10 {
			t.Fatalf("EndOffset=%d want clamped to start 10", e.EndOffset)
		}
		if e.Line != e.EndLine || e.Column != e.EndColumn {
			t.Fatalf("point span after clamp: start=%d:%d end=%d:%d", e.Line, e.Column, e.EndLine, e.EndColumn)
		}
	})

	t.Run("prefixed message stripped", func(t *testing.T) {
		e := ParseErrorFromOffsetsWithLines(src, lines, 7, 10, "line 99:9: expected ';'")
		if e.Message != "expected ';'" {
			t.Fatalf("Message=%q want bare", e.Message)
		}
		if e.Code != "Parser.ExpectedToken" {
			t.Fatalf("Code=%q want ExpectedToken from bare", e.Code)
		}
		got := e.Error()
		if got != "line 2:2: expected ';'" {
			t.Fatalf("Error()=%q want single prefix from stored coords", got)
		}
		if strings.Count(got, "line ") != 1 {
			t.Fatalf("Error() double-prefixed: %q", got)
		}
	})

	t.Run("point span", func(t *testing.T) {
		e := ParseErrorFromOffsetsWithLines(src, lines, 7, 7, "expected ';'")
		if e.Offset != 7 || e.EndOffset != 7 {
			t.Fatalf("offsets start=%d end=%d", e.Offset, e.EndOffset)
		}
		if e.Line != e.EndLine || e.Column != e.EndColumn {
			t.Fatalf("point: start=%d:%d end=%d:%d", e.Line, e.Column, e.EndLine, e.EndColumn)
		}
	})
}

func TestStripLeadingLineCol(t *testing.T) {
	cases := []struct {
		name     string
		msg      string
		wantBare string
		wantLine int
		wantCol  int
	}{
		{
			name:     "well formed",
			msg:      "line 3:12: expected ';', got '}'",
			wantBare: "expected ';', got '}'",
			wantLine: 3,
			wantCol:  12,
		},
		{
			name:     "unprefixed",
			msg:      "Parser panic: boom",
			wantBare: "Parser panic: boom",
		},
		{
			name:     "just line prefix",
			msg:      "line ",
			wantBare: "line ",
		},
		{
			name:     "non digit after line",
			msg:      "line abc:",
			wantBare: "line abc:",
		},
		{
			name:     "missing column",
			msg:      "line 1:",
			wantBare: "line 1:",
		},
		{
			name:     "non digit column",
			msg:      "line 1:x:",
			wantBare: "line 1:x:",
		},
		{
			name:     "overflow digits atoi fail",
			msg:      "line " + strings.Repeat("9", 40) + ":1: expected",
			wantBare: "line " + strings.Repeat("9", 40) + ":1: expected",
		},
		{
			name:     "overflow column digits",
			msg:      "line 1:" + strings.Repeat("9", 40) + ": expected",
			wantBare: "line 1:" + strings.Repeat("9", 40) + ": expected",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			bare, line, col := stripLeadingLineCol(c.msg)
			if bare != c.wantBare || line != c.wantLine || col != c.wantCol {
				t.Fatalf("got bare=%q line=%d col=%d want bare=%q line=%d col=%d",
					bare, line, col, c.wantBare, c.wantLine, c.wantCol)
			}
		})
	}
}

func TestOffsetRuneLineCol(t *testing.T) {
	src := []byte("<?php\nfoo();\n")
	lines := token.NewLineTable(src)

	cases := []struct {
		name       string
		src        []byte
		lines      token.LineTable
		start, end int
		wantSL     int
		wantSC     int
		wantEL     int
		wantEC     int
		wantOff    int
		wantEndOff int
	}{
		{
			name:       "negative start and end",
			src:        src,
			lines:      lines,
			start:      -5,
			end:        -1,
			wantSL:     1,
			wantSC:     1,
			wantEL:     1,
			wantEC:     1,
			wantOff:    -5,
			wantEndOff: -1, // end > start numerically; offsets stored raw, line/col clamp to 0
		},
		{
			name:       "negative end less than start",
			src:        src,
			lines:      lines,
			start:      -1,
			end:        -5,
			wantSL:     1,
			wantSC:     1,
			wantEL:     1,
			wantEC:     1,
			wantOff:    -1,
			wantEndOff: -1, // end < start → EndOffset clamped to start
		},
		{
			name:       "start and end beyond len",
			src:        src,
			lines:      lines,
			start:      100,
			end:        200,
			wantSL:     3,
			wantSC:     1, // EOF after final newline is start of line 3
			wantEL:     3,
			wantEC:     1,
			wantOff:    100,
			wantEndOff: 200,
		},
		{
			name:       "empty src",
			src:        nil,
			lines:      token.NewLineTable(nil),
			start:      0,
			end:        0,
			wantSL:     1,
			wantSC:     1,
			wantEL:     1,
			wantEC:     1,
			wantOff:    0,
			wantEndOff: 0,
		},
		{
			name:       "nil line table",
			src:        src,
			lines:      nil,
			start:      7,
			end:        10,
			wantSL:     1,
			wantSC:     8,
			wantEL:     1,
			wantEC:     11,
			wantOff:    7,
			wantEndOff: 10,
		},
		{
			name:       "empty line table",
			src:        src,
			lines:      token.LineTable{},
			start:      7,
			end:        10,
			wantSL:     1,
			wantSC:     8,
			wantEL:     1,
			wantEC:     11,
			wantOff:    7,
			wantEndOff: 10,
		},
		{
			name:       "stale line table lines[0] > offset",
			src:        src,
			lines:      token.LineTable{10},
			start:      5,
			end:        5,
			wantSL:     1,
			wantSC:     1, // lineStart clamped to offset → runeCol 0 → col 1
			wantEL:     1,
			wantEC:     1,
			wantOff:    5,
			wantEndOff: 5,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := ParseErrorFromOffsetsWithLines(c.src, c.lines, c.start, c.end, "x")
			if e.Line != c.wantSL || e.Column != c.wantSC {
				t.Fatalf("start line/col=%d:%d want %d:%d", e.Line, e.Column, c.wantSL, c.wantSC)
			}
			if e.EndLine != c.wantEL || e.EndColumn != c.wantEC {
				t.Fatalf("end line/col=%d:%d want %d:%d", e.EndLine, e.EndColumn, c.wantEL, c.wantEC)
			}
			if e.Offset != c.wantOff || e.EndOffset != c.wantEndOff {
				t.Fatalf("offsets=%d/%d want %d/%d", e.Offset, e.EndOffset, c.wantOff, c.wantEndOff)
			}
		})
	}
}

func TestUTF8RuneColumns(t *testing.T) {
	// "<?php\n$ä=1;" — ä is 2 bytes (C3 A4); column after $ä should count as one rune.
	src := []byte("<?php\n$ä=1;")
	// Offsets: 0..5 = "<?php\n", 6='$', 7..8=ä, 9='=', 10='1', 11=';'
	e := ParseErrorFromOffsets(src, 9, 11, "expected ';'")
	if e.Line != 2 {
		t.Fatalf("Line=%d want 2", e.Line)
	}
	// After $ (col1) and ä (col2), '=' is column 3.
	if e.Column != 3 {
		t.Fatalf("Column=%d want 3 (rune column after multi-byte ä)", e.Column)
	}
	// End at ';' — after $=ä=1 → column 5.
	if e.EndColumn != 5 {
		t.Fatalf("EndColumn=%d want 5", e.EndColumn)
	}
}
