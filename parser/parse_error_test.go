package parser

import (
	"strings"
	"testing"

	"github.com/ayanozturk/go-php-parser/lexer"
)

func TestStructuredErrorsMissingOpenTag(t *testing.T) {
	p := New(lexer.New("not php"), false)
	_ = p.Parse()
	errs := p.StructuredErrors()
	if len(errs) == 0 {
		t.Fatal("expected structured parse error")
	}
	e := errs[0]
	if e.Code != "Parser.MissingOpenTag" {
		t.Fatalf("Code=%q want Parser.MissingOpenTag", e.Code)
	}
	if e.Line < 1 || e.Column < 1 {
		t.Fatalf("expected located span, got line=%d col=%d", e.Line, e.Column)
	}
	if strings.HasPrefix(e.Message, "line ") {
		t.Fatalf("Message must not include line prefix: %q", e.Message)
	}
	legacy := p.Errors()
	if len(legacy) != 1 || !strings.HasPrefix(legacy[0], "line ") {
		t.Fatalf("Errors() must keep legacy line prefix form, got %v", legacy)
	}
	if legacy[0] != e.Error() {
		t.Fatalf("Errors()[0]=%q Error()=%q", legacy[0], e.Error())
	}
}

func TestStructuredErrorsExpectedTokenSpan(t *testing.T) {
	// Malformed: missing closing paren on function call.
	src := "<?php\nfoo(1;\n"
	p := New(lexer.NewFile(src), false)
	_ = p.Parse()
	errs := p.StructuredErrors()
	if len(errs) == 0 {
		t.Fatal("expected at least one structured error")
	}
	found := false
	for _, e := range errs {
		if e.Code != "Parser.ExpectedToken" && e.Code != "Parser.Syntax" {
			continue
		}
		if e.Line < 1 {
			t.Fatalf("unlocated error: %+v", e)
		}
		if e.Message == "" || strings.HasPrefix(e.Message, "line ") {
			t.Fatalf("bad Message: %+v", e)
		}
		found = true
		break
	}
	if !found {
		t.Fatalf("expected ExpectedToken/Syntax diagnostic, got %+v", errs)
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
