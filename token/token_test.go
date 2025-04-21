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
