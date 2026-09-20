package lexer

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ayanozturk/go-php-parser/token"
)

// TestInterpolatedHeredocDoesNotEagerQueueBody asserts that returning
// T_START_HEREDOC leaves encapsed mode active with an empty pending queue —
// the body is stepped on subsequent NextToken calls, not materialised up front.
func TestInterpolatedHeredocDoesNotEagerQueueBody(t *testing.T) {
	const lines = 2000
	var b strings.Builder
	b.WriteString("<?php $x = <<<EOT\n")
	for i := 0; i < lines; i++ {
		b.WriteString("line $i\n")
	}
	b.WriteString("EOT;\n")
	src := b.String()

	lex := New(src)
	var startSeen bool
	maxPending := 0
	for {
		tok := lex.NextToken()
		if n := len(lex.heredocTokens); n > maxPending {
			maxPending = n
		}
		if tok.Type == token.T_START_HEREDOC {
			startSeen = true
			if n := len(lex.heredocTokens); n != 0 {
				t.Fatalf("after T_START_HEREDOC pending queue=%d; want 0 (incremental emit)", n)
			}
			if lex.encapsed != encapsedHeredoc {
				t.Fatalf("after T_START_HEREDOC encapsed=%v; want encapsedHeredoc", lex.encapsed)
			}
		}
		if tok.Type == token.T_EOF {
			break
		}
	}
	if !startSeen {
		t.Fatal("expected T_START_HEREDOC")
	}
	// $i / $var[…] bursts are small; never the whole 2000-line body.
	if maxPending > 8 {
		t.Fatalf("pending queue peaked at %d; expected bounded burst only", maxPending)
	}
}

func TestInterpolatedHeredocPrintIdentity(t *testing.T) {
	src := "<?php $x = <<<EOT\nhello $name\nworld {$obj->prop}\nEOT;\n"
	toks, err := LexAllContext(nil, []byte(src))
	if err != nil {
		t.Fatalf("LexAllContext: %v", err)
	}
	if got := PrintTokens(toks, []byte(src)); got != src {
		t.Fatalf("identity failed:\nwant %q\ngot  %q", src, got)
	}
}

func TestLexAllContextCancelsBetweenInterpolatedHeredocTokens(t *testing.T) {
	// Many small interpolations so cancel can fire between NextToken steps
	// rather than only mid-chunk inside one eager scan.
	const vars = 512
	var b bytes.Buffer
	b.WriteString("<?php $x = <<<EOT\n")
	for i := 0; i < vars; i++ {
		b.WriteString("$a")
	}
	b.WriteString("\nEOT;\n")
	src := b.Bytes()

	ctx := &nthErrCancel{Context: context.Background(), left: 20}
	toks, err := LexAllContext(ctx, src)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got err=%v toks=%d", err, len(toks))
	}
	for _, tok := range toks {
		if tok.Type == token.T_END_HEREDOC {
			t.Fatalf("cancelled lex returned heredoc terminator with %d tokens emitted", len(toks))
		}
	}
	if len(toks) >= vars {
		t.Fatalf("expected cancel before draining all interpolations; got %d tokens", len(toks))
	}
}
