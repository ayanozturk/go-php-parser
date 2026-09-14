package syntax

import (
	"strings"

	"github.com/ayanozturk/go-php-parser/lexer"
	"github.com/ayanozturk/go-php-parser/token"
)

// Print returns the exact source text represented by a red tree by slicing
// the file source with absolute red offsets + green widths (R1).
func Print(n *RedNode) string {
	if n == nil || n.File == nil {
		return ""
	}
	return PrintGreenAt(n.Green, n.File.Source, n.Offset)
}

// PrintGreen reprints a green subtree that starts at byte offset 0 in src
// (full-file greens only). Prefer Print on a RedNode for subtrees.
func PrintGreen(g *GreenNode, src []byte) string {
	return PrintGreenAt(g, src, 0)
}

// PrintGreenAt reprints green covering src[start:start+g.Width] by walking
// widths — green tokens carry no absolute source offsets.
func PrintGreenAt(g *GreenNode, src []byte, start int) string {
	if g == nil {
		return ""
	}
	var b strings.Builder
	b.Grow(g.width)
	off := start
	writeGreen(&b, g, src, &off)
	return b.String()
}

func writeGreen(b *strings.Builder, g *GreenNode, src []byte, off *int) {
	if g == nil {
		return
	}
	if g.IsToken() {
		end := *off + g.width
		if g.width > 0 && *off >= 0 && end <= len(src) {
			b.Write(src[*off:end])
		} else if g.token != nil {
			// Fallback for hand-built tokens without a backing source span.
			writeTokenFallback(b, *g.token, src)
		}
		*off = end
		return
	}
	for _, c := range g.children {
		writeGreen(b, c, src, off)
	}
}

func writeTokenFallback(b *strings.Builder, tok token.Token, src []byte) {
	for _, tr := range tok.LeadingTrivia {
		if t := tr.Text(src); t != "" {
			b.WriteString(t)
		} else {
			b.WriteString(tr.Literal)
		}
	}
	if tok.Type != token.T_EOF {
		if t := tok.Text(src); t != "" {
			b.WriteString(t)
		} else {
			b.WriteString(tok.Literal)
		}
	}
	for _, tr := range tok.TrailingTrivia {
		if t := tr.Text(src); t != "" {
			b.WriteString(t)
		} else {
			b.WriteString(tr.Literal)
		}
	}
}

// ParseTokens builds a tokens-only green File (no structural parse).
// Identity: Print(BindRed(ParseTokens(src))) == string(src).
func ParseTokens(src []byte) *File {
	toks := lexer.LexAll(src)
	in := NewInterner(src)
	children := make([]*GreenNode, 0, len(toks))
	for _, tok := range toks {
		children = append(children, in.Token(tok))
	}
	green := in.Node(KindFile, in.Node(KindTokenList, children...))
	f := &File{
		Source: src,
		Lines:  token.NewLineTable(src),
		Green:  green,
		Tokens: toks,
	}
	BindRed(f)
	return f
}
