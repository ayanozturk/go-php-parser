package token

import (
	"testing"
)

func TestTokenTypeConstants(t *testing.T) {
	// Spot check a few token types
	if T_SELF != "T_SELF" {
		t.Errorf("T_SELF = %q, want %q", T_SELF, "T_SELF")
	}
	if T_PARENT != "T_PARENT" {
		t.Errorf("T_PARENT = %q, want %q", T_PARENT, "T_PARENT")
	}
	if T_FUNCTION != "T_FUNCTION" {
		t.Errorf("T_FUNCTION = %q, want %q", T_FUNCTION, "T_FUNCTION")
	}
	if T_VARIABLE != "T_VARIABLE" {
		t.Errorf("T_VARIABLE = %q, want %q", T_VARIABLE, "T_VARIABLE")
	}
	if T_WHITESPACE != "T_WHITESPACE" {
		t.Errorf("T_WHITESPACE = %q, want %q", T_WHITESPACE, "T_WHITESPACE")
	}
}

func TestPositionFields(t *testing.T) {
	pos := Position{Line: 3, Column: 5, Offset: 42}
	if pos.Line != 3 || pos.Column != 5 || pos.Offset != 42 {
		t.Errorf("Position fields not set correctly: %+v", pos)
	}
}

func TestTokenFields(t *testing.T) {
	tok := Token{Type: T_STRING, Literal: "foobar", Pos: Position{Line: 1, Column: 2, Offset: 3}}
	if tok.Type != T_STRING {
		t.Errorf("Token.Type = %q, want %q", tok.Type, T_STRING)
	}
	if tok.Literal != "foobar" {
		t.Errorf("Token.Literal = %q, want %q", tok.Literal, "foobar")
	}
	if tok.Pos.Line != 1 || tok.Pos.Column != 2 || tok.Pos.Offset != 3 {
		t.Errorf("Token.Pos not set correctly: %+v", tok.Pos)
	}
}

func TestTokenEndPosSingleLine(t *testing.T) {
	tok := Token{Type: T_STRING, Literal: "foobar", Pos: Position{Line: 1, Column: 2, Offset: 3}}
	end := tok.EndPos()
	if end.Line != 1 || end.Column != 8 || end.Offset != 9 {
		t.Errorf("EndPos() = %+v, want {Line:1 Column:8 Offset:9}", end)
	}
}

func TestTokenEndPosMultiLine(t *testing.T) {
	// A heredoc-like literal spanning two lines: "ab\ncd" starting at
	// line 1, column 5. After consuming it the end position should be on
	// line 2, right after "cd".
	tok := Token{Type: T_ENCAPSED_AND_WHITESPACE, Literal: "ab\ncd", Pos: Position{Line: 1, Column: 5, Offset: 10}}
	end := tok.EndPos()
	if end.Line != 2 {
		t.Errorf("EndPos().Line = %d, want 2", end.Line)
	}
	if end.Column != 3 {
		t.Errorf("EndPos().Column = %d, want 3 (after 'cd', 1-based)", end.Column)
	}
	if end.Offset != 15 {
		t.Errorf("EndPos().Offset = %d, want 15 (10 + len(\"ab\\ncd\"))", end.Offset)
	}
}
