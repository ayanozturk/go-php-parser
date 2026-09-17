package syntax

import (
	"bytes"
	"testing"
)

const maxSyntaxFuzzInput = 64 << 10

func FuzzSyntaxParserMalformedPHP(f *testing.F) {
	addSyntaxFuzzSeeds(f)
	f.Fuzz(func(t *testing.T, src []byte) {
		if len(src) > maxSyntaxFuzzInput {
			return
		}
		assertFuzzParseResult(t, src, Parse(src))
	})
}

func FuzzSyntaxIndexParserMalformedPHP(f *testing.F) {
	addSyntaxFuzzSeeds(f)
	f.Fuzz(func(t *testing.T, src []byte) {
		if len(src) > maxSyntaxFuzzInput {
			return
		}
		assertFuzzParseResult(t, src, ParseForIndex(src))
	})
}

func addSyntaxFuzzSeeds(f *testing.F) {
	f.Helper()
	for _, seed := range [][]byte{
		{},
		[]byte("<?php"),
		[]byte("<?php #[Attr("),
		[]byte("<?php function f() { return match($x) {"),
		[]byte("<?php function f(match: false) {}"),
		[]byte("<?php switch ($x) { case 1: { echo 1; }"),
		[]byte("<?php class C { function f() { $x = \"{$value}\"; }"),
		[]byte("<?phpA[0(00"),
		{0xff, 0xfe, 0xfd, 0x00},
	} {
		f.Add(seed)
	}
}

func assertFuzzParseResult(t *testing.T, src []byte, res *ParseResult) {
	t.Helper()
	if res == nil || res.File == nil || res.File.Root == nil {
		t.Fatal("parser returned an incomplete result")
	}
	if got := []byte(Print(res.File.Root)); !bytes.Equal(got, src) {
		t.Fatalf("lossless identity failed\nwant %q\ngot  %q", src, got)
	}
	for _, diag := range res.Diagnostics {
		if diag.Span.Start < 0 || diag.Span.End < diag.Span.Start || diag.Span.End > len(src) {
			t.Fatalf("diagnostic span %+v outside source length %d", diag.Span, len(src))
		}
	}
}
