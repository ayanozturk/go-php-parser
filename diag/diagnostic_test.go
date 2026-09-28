package diag

import "testing"

func TestByteSpanFromRunePositionsUsesUTF8OffsetsAndCRLF(t *testing.T) {
	source := []byte("a🙂é\r\nz")
	got := ByteSpanFromRunePositions(source, 1, 2, 1, 3)
	if got != (ByteSpan{Start: 1, End: 5}) {
		t.Fatalf("expected emoji byte span [1,5), got %+v", got)
	}

	got = ByteSpanFromRunePositions(source, 2, 1, 2, 2)
	if got != (ByteSpan{Start: 9, End: 10}) {
		t.Fatalf("expected second-line byte span [9,10), got %+v", got)
	}
}

func TestByteSpanFromRunePositionsMalformedAndZeroWidth(t *testing.T) {
	source := []byte("abc\r\ndef")
	tests := []struct {
		name                   string
		startLine, startColumn int
		endLine, endColumn     int
		want                   ByteSpan
	}{
		{name: "zero width", startLine: 1, startColumn: 2, endLine: 1, endColumn: 2, want: ByteSpan{Start: 1, End: 1}},
		{name: "missing end", startLine: 1, startColumn: 2, want: ByteSpan{Start: 1, End: 1}},
		{name: "invalid start", startLine: 8, startColumn: 1, endLine: 2, endColumn: 2, want: ByteSpan{}},
		{name: "invalid end", startLine: 1, startColumn: 2, endLine: 8, endColumn: 1, want: ByteSpan{Start: 1, End: 1}},
		{name: "reversed", startLine: 2, startColumn: 2, endLine: 2, endColumn: 1, want: ByteSpan{Start: 6, End: 6}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ByteSpanFromRunePositions(source, tt.startLine, tt.startColumn, tt.endLine, tt.endColumn); got != tt.want {
				t.Fatalf("expected %+v, got %+v", tt.want, got)
			}
		})
	}
}

func TestUTF16PositionFromByteOffset(t *testing.T) {
	source := []byte("a🙂é\r\nz")
	tests := []struct {
		name   string
		offset int
		want   UTF16Position
	}{
		{name: "before emoji", offset: 1, want: UTF16Position{Line: 0, Character: 1}},
		{name: "emoji end", offset: 5, want: UTF16Position{Line: 0, Character: 3}},
		{name: "CRLF line start", offset: 9, want: UTF16Position{Line: 1, Character: 0}},
		{name: "negative clamps", offset: -1, want: UTF16Position{}},
		{name: "past end clamps", offset: 100, want: UTF16Position{Line: 1, Character: 1}},
		{name: "middle of emoji rounds down", offset: 3, want: UTF16Position{Line: 0, Character: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UTF16PositionFromByteOffset(source, tt.offset); got != tt.want {
				t.Fatalf("expected %+v, got %+v", tt.want, got)
			}
		})
	}
}

func TestSortOrdersByFilenameSpanAndStableTieBreakers(t *testing.T) {
	values := []Diagnostic{
		{Filename: "b.php", Span: ByteSpan{Start: 0}, Code: "A"},
		{Filename: "a.php", Span: ByteSpan{Start: 4}, Code: "A"},
		{Filename: "a.php", Span: ByteSpan{Start: 1}, Code: "B"},
		{Filename: "a.php", Span: ByteSpan{Start: 1}, Code: "A", Message: "z"},
		{Filename: "a.php", Span: ByteSpan{Start: 1}, Code: "A", Message: "a"},
	}
	Sort(values)
	want := []struct {
		file, code, message string
	}{
		{file: "a.php", code: "A", message: "a"},
		{file: "a.php", code: "A", message: "z"},
		{file: "a.php", code: "B", message: ""},
		{file: "a.php", code: "A", message: ""},
		{file: "b.php", code: "A", message: ""},
	}
	for i, expected := range want {
		if values[i].Filename != expected.file || values[i].Code != expected.code || values[i].Message != expected.message {
			t.Fatalf("diagnostic %d: expected %+v, got %#v", i, expected, values[i])
		}
	}
}
