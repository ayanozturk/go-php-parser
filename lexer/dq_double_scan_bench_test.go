package lexer

import (
	"strings"
	"testing"

	"github.com/ayanozturk/go-php-parser/token"
)

// Microbench for plain double-quoted string lexing cost vs single-quoted
// (one-pass control) and interpolating DQ. Sizes mirror large config/template
// literals. Used to verify the fused commitConstantDoubleQuote path.

func makePHPDQ(body string) string {
	return "<?php " + body + ";"
}

func makePlainDQ(size int) string {
	// No $, no {$, no unescaped " — constant encapsed path.
	return `"` + strings.Repeat("a", size) + `"`
}

func makeInterpDQ(size int) string {
	// Dollar near the end forces classification work then encapsed tokenize.
	if size < 4 {
		size = 4
	}
	return `"` + strings.Repeat("a", size-2) + `$x"`
}

func makePlainSQ(size int) string {
	return `'` + strings.Repeat("a", size) + `'`
}

func drainLexer(src string) int {
	lex := New(src)
	n := 0
	for {
		tok := lex.NextToken()
		n++
		if tok.Type == token.T_EOF {
			return n
		}
	}
}

func BenchmarkDQPlain64KiB(b *testing.B) {
	src := makePHPDQ(makePlainDQ(64 << 10))
	b.SetBytes(int64(len(src)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		drainLexer(src)
	}
}

func BenchmarkDQInterp64KiB(b *testing.B) {
	src := makePHPDQ(makeInterpDQ(64 << 10))
	b.SetBytes(int64(len(src)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		drainLexer(src)
	}
}

func BenchmarkSQPlain64KiB(b *testing.B) {
	src := makePHPDQ(makePlainSQ(64 << 10))
	b.SetBytes(int64(len(src)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		drainLexer(src)
	}
}

func BenchmarkDQPlain1MiB(b *testing.B) {
	src := makePHPDQ(makePlainDQ(1 << 20))
	b.SetBytes(int64(len(src)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		drainLexer(src)
	}
}

func BenchmarkDQInterp1MiB(b *testing.B) {
	src := makePHPDQ(makeInterpDQ(1 << 20))
	b.SetBytes(int64(len(src)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		drainLexer(src)
	}
}

func BenchmarkSQPlain1MiB(b *testing.B) {
	src := makePHPDQ(makePlainSQ(1 << 20))
	b.SetBytes(int64(len(src)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		drainLexer(src)
	}
}

// Isolate just the string token path (skip open-tag overhead).
func BenchmarkDQPlainOnly64KiB(b *testing.B) {
	body := makePlainDQ(64 << 10)
	b.SetBytes(int64(len(body)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lex := New(body)
		_ = lex.NextToken()
	}
}

func BenchmarkSQPlainOnly64KiB(b *testing.B) {
	body := makePlainSQ(64 << 10)
	b.SetBytes(int64(len(body)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lex := New(body)
		_ = lex.NextToken()
	}
}

func BenchmarkDQVsSQRatio64KiB(b *testing.B) {
	dq := makePlainDQ(64 << 10)
	sq := makePlainSQ(64 << 10)
	b.ReportAllocs()
	b.Run("DQ", func(b *testing.B) {
		b.SetBytes(int64(len(dq)))
		for i := 0; i < b.N; i++ {
			lex := New(dq)
			_ = lex.NextToken()
		}
	})
	b.Run("SQ", func(b *testing.B) {
		b.SetBytes(int64(len(sq)))
		for i := 0; i < b.N; i++ {
			lex := New(sq)
			_ = lex.NextToken()
		}
	})
}
