package token

import (
	"testing"
)

func TestTokenTypeConstants(t *testing.T) {
	if T_SELF.String() != "T_SELF" {
		t.Errorf("T_SELF = %q, want %q", T_SELF, "T_SELF")
	}
	if T_PARENT.String() != "T_PARENT" {
		t.Errorf("T_PARENT = %q, want %q", T_PARENT, "T_PARENT")
	}
	if T_FUNCTION.String() != "T_FUNCTION" {
		t.Errorf("T_FUNCTION = %q, want %q", T_FUNCTION, "T_FUNCTION")
	}
	if T_VARIABLE.String() != "T_VARIABLE" {
		t.Errorf("T_VARIABLE = %q, want %q", T_VARIABLE, "T_VARIABLE")
	}
	if T_WHITESPACE.String() != "T_WHITESPACE" {
		t.Errorf("T_WHITESPACE = %q, want %q", T_WHITESPACE, "T_WHITESPACE")
	}
}

func TestTokenTypeStringAllNamed(t *testing.T) {
	for i, name := range tokenTypeNames {
		tt := TokenType(i)
		got := tt.String()
		if name == "" {
			if got != "T_UNKNOWN" {
				t.Fatalf("TokenType(%d).String() = %q, want T_UNKNOWN", i, got)
			}
			continue
		}
		if got != name {
			t.Fatalf("TokenType(%d).String() = %q, want %q", i, got, name)
		}
	}
}

func TestTokenTypeStringUnknown(t *testing.T) {
	if got := TokenType(0).String(); got != "T_UNKNOWN" {
		t.Fatalf("zero TokenType.String() = %q, want T_UNKNOWN", got)
	}
	outOfRange := TokenType(len(tokenTypeNames) + 10)
	if got := outOfRange.String(); got != "T_UNKNOWN" {
		t.Fatalf("out-of-range String() = %q, want T_UNKNOWN", got)
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
		t.Errorf("Token.Type = %s, want %s", tok.Type, T_STRING)
	}
	if tok.Literal != "foobar" {
		t.Errorf("Token.Literal = %q, want %q", tok.Literal, "foobar")
	}
	if tok.Pos.Line != 1 || tok.Pos.Column != 2 || tok.Pos.Offset != 3 {
		t.Errorf("Token.Pos not set correctly: %+v", tok.Pos)
	}
}

func TestTokenTextUsesSourceSpan(t *testing.T) {
	src := []byte("<?php echo $x;")
	tok := Token{
		Type:    T_VARIABLE,
		Literal: "$stale",
		Pos:     Position{Offset: 11},
		End:     Position{Offset: 13},
	}
	if got := tok.Text(src); got != "$x" {
		t.Fatalf("Text() = %q, want %q", got, "$x")
	}
}

func TestTokenTextFallsBackToLiteral(t *testing.T) {
	src := []byte("short")
	cases := []struct {
		name string
		tok  Token
	}{
		{
			name: "unset end",
			tok:  Token{Literal: "fallback", Pos: Position{Offset: 1}},
		},
		{
			name: "end beyond source",
			tok: Token{
				Literal: "fallback",
				Pos:     Position{Offset: 0},
				End:     Position{Offset: 99},
			},
		},
		{
			name: "negative pos",
			tok: Token{
				Literal: "fallback",
				Pos:     Position{Offset: -1},
				End:     Position{Offset: 2},
			},
		},
		{
			name: "zero width span",
			tok: Token{
				Literal: "fallback",
				Pos:     Position{Offset: 1},
				End:     Position{Offset: 1},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.tok.Text(src); got != "fallback" {
				t.Fatalf("Text() = %q, want fallback", got)
			}
		})
	}
}

func TestTokenIsTrivia(t *testing.T) {
	trivia := []TokenType{T_WHITESPACE, T_COMMENT, T_DOC_COMMENT}
	for _, tt := range trivia {
		tok := Token{Type: tt}
		if !tok.IsTrivia() {
			t.Fatalf("%s should be trivia", tt)
		}
	}
	nonTrivia := Token{Type: T_STRING}
	if nonTrivia.IsTrivia() {
		t.Fatal("T_STRING must not be trivia")
	}
	eof := Token{Type: T_EOF}
	if eof.IsTrivia() {
		t.Fatal("T_EOF must not be trivia")
	}
}

func TestTokenWidth(t *testing.T) {
	withEnd := Token{
		Literal: "ignored",
		Pos:     Position{Offset: 10},
		End:     Position{Offset: 15},
	}
	if got := withEnd.Width(); got != 5 {
		t.Fatalf("Width() with End = %d, want 5", got)
	}
	fromLiteral := Token{Literal: "abc", Pos: Position{Offset: 3}}
	if got := fromLiteral.Width(); got != 3 {
		t.Fatalf("Width() from Literal = %d, want 3", got)
	}
	zeroWidth := Token{
		Literal: "x",
		Pos:     Position{Offset: 4},
		End:     Position{Offset: 4},
	}
	if got := zeroWidth.Width(); got != 1 {
		t.Fatalf("Width() zero-width End falls back to Literal len = %d, want 1", got)
	}
}

func TestTokenEndPosSingleLine(t *testing.T) {
	tok := Token{Type: T_STRING, Literal: "foobar", Pos: Position{Line: 1, Column: 2, Offset: 3}}
	end := tok.EndPos()
	if end.Line != 1 || end.Column != 8 || end.Offset != 9 {
		t.Errorf("EndPos() = %+v, want {Line:1 Column:8 Offset:9}", end)
	}
}

func TestLineTableLineBytes(t *testing.T) {
	src := []byte("one\n two\r\nthree")
	table := NewLineTable(src)
	if got := string(table.LineBytes(src, 1)); got != "one" {
		t.Errorf("line 1 = %q, want %q", got, "one")
	}
	if got := string(table.LineBytes(src, 2)); got != " two" {
		t.Errorf("line 2 = %q, want %q", got, " two")
	}
	if got := string(table.LineBytes(src, 3)); got != "three" {
		t.Errorf("line 3 = %q, want %q", got, "three")
	}
}

func TestLineTableLineBytesEdges(t *testing.T) {
	src := []byte("a\n")
	table := NewLineTable(src)
	if len(table) != 2 {
		t.Fatalf("NewLineTable(%q) len = %d, want 2", src, len(table))
	}
	if got := string(table.LineBytes(src, 1)); got != "a" {
		t.Fatalf("line 1 = %q, want %q", got, "a")
	}
	if got := table.LineBytes(src, 2); got == nil || len(got) != 0 {
		t.Fatalf("trailing empty line = %q, want empty non-nil", got)
	}
	if got := table.LineBytes(src, 0); got != nil {
		t.Fatalf("line 0 = %q, want nil", got)
	}
	if got := table.LineBytes(src, -1); got != nil {
		t.Fatalf("line -1 = %q, want nil", got)
	}
	if got := table.LineBytes(src, 3); got != nil {
		t.Fatalf("line 3 = %q, want nil", got)
	}

	crOnly := []byte("row\r")
	crTable := NewLineTable(crOnly)
	if got := string(crTable.LineBytes(crOnly, 1)); got != "row" {
		t.Fatalf("bare CR terminator stripped: got %q, want %q", got, "row")
	}

	empty := NewLineTable(nil)
	if len(empty) != 1 || empty[0] != 0 {
		t.Fatalf("empty source LineTable = %#v, want [0]", []int(empty))
	}
	if got := empty.LineBytes(nil, 1); got == nil || len(got) != 0 {
		t.Fatalf("empty source line 1 = %q, want empty", got)
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

func TestTokenEndPosTrailingNewline(t *testing.T) {
	tok := Token{
		Type:    T_WHITESPACE,
		Literal: "x\n",
		Pos:     Position{Line: 4, Column: 2, Offset: 20},
	}
	end := tok.EndPos()
	if end.Line != 5 || end.Column != 1 || end.Offset != 22 {
		t.Fatalf("EndPos() = %+v, want {Line:5 Column:1 Offset:22}", end)
	}
}

func TestTokenEndPosEmptyLiteral(t *testing.T) {
	tok := Token{Type: T_EOF, Pos: Position{Line: 1, Column: 1, Offset: 0}}
	end := tok.EndPos()
	if end != tok.Pos {
		t.Fatalf("empty Literal EndPos() = %+v, want Pos %+v", end, tok.Pos)
	}
}

func TestTokenEndPosUsesStoredEnd(t *testing.T) {
	tok := Token{
		Type:    T_STRING,
		Literal: "foobar\nwith\nnewlines",
		Pos:     Position{Line: 1, Column: 1, Offset: 0},
		End:     Position{Line: 4, Column: 9, Offset: 42},
	}
	end := tok.EndPos()
	if end != tok.End {
		t.Errorf("EndPos() = %+v, want stored End %+v", end, tok.End)
	}
}
