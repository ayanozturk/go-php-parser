package diag

import (
	"strings"
	"testing"
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
}

func TestClassifyParseError(t *testing.T) {
	cases := []struct {
		msg  string
		code string
	}{
		{"expected <?php", "Parser.MissingOpenTag"},
		{"Parser panic: boom", "Parser.Internal"},
		{"context cancelled", "Parser.Internal"},
		{"expected ';', got '}'", "Parser.ExpectedToken"},
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

func TestStripLeadingLineCol(t *testing.T) {
	bare, line, col := stripLeadingLineCol("line 3:12: expected ';', got '}'")
	if bare != "expected ';', got '}'" || line != 3 || col != 12 {
		t.Fatalf("got bare=%q line=%d col=%d", bare, line, col)
	}
	bare, line, col = stripLeadingLineCol("Parser panic: boom")
	if bare != "Parser panic: boom" || line != 0 || col != 0 {
		t.Fatalf("unprefixed: bare=%q line=%d col=%d", bare, line, col)
	}
}
