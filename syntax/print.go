package syntax

import (
	"strings"

	"github.com/ayanozturk/go-php-parser/lexer"
	"github.com/ayanozturk/go-php-parser/token"
)

// Print returns the exact source text represented by a green/red tree by
// concatenating trivia + token text in order.
func Print(n *RedNode) string {
	if n == nil || n.File == nil {
		return ""
	}
	return PrintGreen(n.Green, n.File.Source)
}

func PrintGreen(g *GreenNode, src []byte) string {
	if g == nil {
		return ""
	}
	var b strings.Builder
	b.Grow(g.Width)
	writeGreen(&b, g, src)
	return b.String()
}

func writeGreen(b *strings.Builder, g *GreenNode, src []byte) {
	if g == nil {
		return
	}
	if g.IsToken() {
		writeToken(b, *g.Token, src)
		return
	}
	for _, c := range g.Children {
		writeGreen(b, c, src)
	}
}

func writeToken(b *strings.Builder, tok token.Token, src []byte) {
	for _, tr := range tok.LeadingTrivia {
		b.WriteString(tr.Text(src))
	}
	if tok.Type != token.T_EOF {
		b.WriteString(tok.Text(src))
	}
	for _, tr := range tok.TrailingTrivia {
		b.WriteString(tr.Text(src))
	}
}

// ParseTokens builds a tokens-only green File (no structural parse).
// Identity: Print(BindRed(ParseTokens(src))) == string(src).
func ParseTokens(src []byte) *File {
	toks := lexer.LexAll(src)
	in := NewInterner()
	children := make([]*GreenNode, 0, len(toks))
	for _, tok := range toks {
		if tok.Type == token.T_EOF {
			// Keep EOF so trailing trivia is preserved.
			children = append(children, in.Token(tok))
			continue
		}
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
