package lexer

import (
	"bytes"
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/ayanozturk/go-php-parser/token"
)

// nthErrCancel returns context.Canceled after n successful Err() calls.
// LexAllContext and the in-scanner budget checks only consult Err(), so this
// makes mid-body cancellation deterministic without relying on wall time.
type nthErrCancel struct {
	context.Context
	left int32
}

func (c *nthErrCancel) Err() error {
	if atomic.AddInt32(&c.left, -1) < 0 {
		return context.Canceled
	}
	return nil
}

func furthestTokenOffset(toks []token.Token) int {
	max := 0
	for _, tok := range toks {
		if tok.End.Offset > max {
			max = tok.End.Offset
		}
		if tok.Pos.Offset > max {
			max = tok.Pos.Offset
		}
		for _, tr := range tok.LeadingTrivia {
			if tr.End.Offset > max {
				max = tr.End.Offset
			}
			if tr.Pos.Offset > max {
				max = tr.Pos.Offset
			}
		}
	}
	return max
}

func TestLexAllContextCancelsMidMultiMegabyteHeredoc(t *testing.T) {
	const bodySize = 4 << 20 // 4 MiB
	var b bytes.Buffer
	b.Grow(bodySize + 64)
	b.WriteString("<?php $x = <<<EOT\n")
	b.Write(bytes.Repeat([]byte("a"), bodySize))
	b.WriteString("\nEOT;\n")
	src := b.Bytes()

	// Enough Err() successes to pass open-tag / $x / = between-token checks,
	// then cancel inside the heredoc body scanner (one NextToken).
	ctx := &nthErrCancel{Context: context.Background(), left: 6}
	toks, err := LexAllContext(ctx, src)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got err=%v toks=%d", err, len(toks))
	}
	if off := furthestTokenOffset(toks); off > bodySize/8 {
		t.Fatalf("cancellation did not bound scan progress: furthest offset %d on %d-byte body", off, bodySize)
	}
	for _, tok := range toks {
		if tok.Type == token.T_END_HEREDOC || tok.Type == token.T_END_NOWDOC {
			t.Fatalf("cancelled lex returned heredoc terminator: %#v", tok)
		}
	}
}

func TestLexAllContextCancelsMidMultiMegabyteString(t *testing.T) {
	const bodySize = 4 << 20
	var b bytes.Buffer
	b.Grow(bodySize + 32)
	b.WriteString("<?php $x = \"")
	b.Write(bytes.Repeat([]byte("b"), bodySize))
	b.WriteString("\";\n")
	src := b.Bytes()

	ctx := &nthErrCancel{Context: context.Background(), left: 6}
	toks, err := LexAllContext(ctx, src)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got err=%v toks=%d", err, len(toks))
	}
	if off := furthestTokenOffset(toks); off > bodySize/8 {
		t.Fatalf("cancellation did not bound scan progress: furthest offset %d on %d-byte body", off, bodySize)
	}
}

func TestLexAllContextCancelsMidMultiMegabyteBlockComment(t *testing.T) {
	const bodySize = 4 << 20
	var b bytes.Buffer
	b.Grow(bodySize + 32)
	b.WriteString("<?php /*")
	b.Write(bytes.Repeat([]byte("c"), bodySize))
	b.WriteString("*/ $x;\n")
	src := b.Bytes()

	ctx := &nthErrCancel{Context: context.Background(), left: 4}
	toks, err := LexAllContext(ctx, src)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got err=%v toks=%d", err, len(toks))
	}
	if off := furthestTokenOffset(toks); off > bodySize/8 {
		t.Fatalf("cancellation did not bound scan progress: furthest offset %d on %d-byte body", off, bodySize)
	}
}

func TestLexAllContextNilStillLexesLargeHeredoc(t *testing.T) {
	const bodySize = 256 << 10 // 256 KiB — identity check without multi-second CI cost
	var b bytes.Buffer
	b.Grow(bodySize + 64)
	b.WriteString("<?php $x = <<<EOT\n")
	b.Write(bytes.Repeat([]byte("d"), bodySize))
	b.WriteString("\nEOT;\n")
	src := b.Bytes()

	toks, err := LexAllContext(nil, src)
	if err != nil {
		t.Fatalf("nil context returned error: %v", err)
	}
	if got := PrintTokens(toks, src); got != string(src) {
		t.Fatalf("large heredoc identity failed: got %d bytes, want %d", len(got), len(src))
	}
}
