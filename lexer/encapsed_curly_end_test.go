package lexer

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/token"
)

func TestCurlyInterpRbraceEndDoesNotSwallowBody(t *testing.T) {
	cases := []string{
		`<?php echo <<<EOT
hello {$x}
EOT;
`,
		`<?php $a = "hello {$x} world";
`,
		`<?php $a = "doctrine.dbal.{$name}_connection";
`,
	}
	for _, src := range cases {
		l := NewFileBytes([]byte(src))
		var toks []token.Token
		for {
			tok := l.NextToken()
			toks = append(toks, tok)
			if tok.Type == token.T_EOF {
				break
			}
		}
		for i, tok := range toks {
			if tok.Type != token.T_RBRACE {
				continue
			}
			if tok.Literal != "}" {
				t.Fatalf("%q: RBRACE literal %q", src, tok.Literal)
			}
			if tok.Width() != 1 {
				t.Fatalf("%q: RBRACE Width=%d End=%v (token[%d]); want 1\nfull: %+v",
					src, tok.Width(), tok.End, i, toks)
			}
			// Following token must start immediately after '}'.
			if i+1 < len(toks) {
				next := toks[i+1]
				if next.Pos.Offset != tok.Pos.Offset+1 && next.Type != token.T_EOF {
					// Allow trivia-only advances via End; significant next should abut.
					if next.Pos.Offset < tok.Pos.Offset+1 {
						t.Fatalf("%q: next token overlaps RBRACE: rbrace=%v next=%v", src, tok, next)
					}
				}
			}
		}
		// Stream must cover source without overlapping widths on significant tokens.
		off := 0
		for _, tok := range toks {
			for _, tr := range tok.LeadingTrivia {
				off += tr.Width()
				if tr.Width() == 0 {
					off += len(tr.Literal)
				}
			}
			w := tok.Width()
			if tok.Type == token.T_EOF {
				for _, tr := range tok.TrailingTrivia {
					off += tr.Width()
					if tr.Width() == 0 {
						off += len(tr.Literal)
					}
				}
				continue
			}
			if tok.Pos.Offset != off {
				t.Fatalf("%q: token %s at Pos.Offset=%d but cumulative=%d", src, tok.Type, tok.Pos.Offset, off)
			}
			off += w
		}
	}
}
