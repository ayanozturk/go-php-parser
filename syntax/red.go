package syntax

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/token"
)

// File is the red root for a parsed source file.
type File struct {
	Source []byte
	Lines  token.LineTable
	Green  *GreenNode
	Tokens []token.Token
	Root   *RedNode
}

// RedNode is a typed view over a green node with parent and absolute span.
// Absolute byte offsets live here only — never on green (R1).
type RedNode struct {
	File   *File
	Parent *RedNode
	Green  *GreenNode
	Offset int // byte offset of this node's start in File.Source
}

func (r *RedNode) Kind() Kind {
	if r == nil || r.Green == nil {
		return KindError
	}
	return r.Green.kind
}

func (r *RedNode) Span() Span {
	if r == nil || r.Green == nil {
		return Span{}
	}
	return Span{Start: r.Offset, End: r.Offset + r.Green.width}
}

func (r *RedNode) Text() string {
	s := r.Span()
	if r.File == nil || s.Start < 0 || s.End > len(r.File.Source) || s.Start > s.End {
		return ""
	}
	return string(r.File.Source[s.Start:s.End])
}

func (r *RedNode) Children() []*RedNode {
	if r == nil || r.Green == nil || len(r.Green.children) == 0 {
		return nil
	}
	out := make([]*RedNode, 0, len(r.Green.children))
	off := r.Offset
	for _, g := range r.Green.children {
		if g == nil {
			continue
		}
		child := &RedNode{File: r.File, Parent: r, Green: g, Offset: off}
		out = append(out, child)
		off += g.width
	}
	return out
}

func (r *RedNode) Child(i int) *RedNode {
	ch := r.Children()
	if i < 0 || i >= len(ch) {
		return nil
	}
	return ch[i]
}

// Pos returns the 1-based start position of the node's content, skipping
// leading trivia so it matches classic ast.Node.GetPos() coordinates.
func (r *RedNode) Pos() ast.Position {
	start, _ := nodePos(r.File, r)
	return start
}

// EndPos returns the 1-based end position of the node's content, excluding
// trailing trivia so it matches classic ast.Node.GetEndPos() coordinates.
func (r *RedNode) EndPos() ast.Position {
	_, end := nodePos(r.File, r)
	return end
}

// Tokens returns lexer tokens under this red node with absolute positions
// reconstructed from the red offset walk (green tokens are position-independent).
func (r *RedNode) Tokens() []token.Token {
	var out []token.Token
	var walk func(*RedNode)
	walk = func(n *RedNode) {
		if n == nil || n.Green == nil {
			return
		}
		if n.Green.IsToken() {
			tok, ok := n.Green.Token()
			if !ok {
				return
			}
			// Place the significant token after leading trivia within this green width.
			lead := 0
			for _, tr := range tok.LeadingTrivia {
				w := tr.Width()
				if w == 0 {
					w = len(tr.Literal)
				}
				lead += w
			}
			sig := tok.Width()
			if sig == 0 && tok.Type != token.T_EOF {
				sig = len(tok.Literal)
			}
			tok.Pos = token.Position{Offset: n.Offset + lead}
			tok.End = token.Position{Offset: n.Offset + lead + sig}
			out = append(out, tok)
			return
		}
		for _, c := range n.Children() {
			walk(c)
		}
	}
	walk(r)
	return out
}

// BindRed builds the red root for a green file node.
func BindRed(f *File) *RedNode {
	if f == nil || f.Green == nil {
		return nil
	}
	f.Root = &RedNode{File: f, Green: f.Green, Offset: 0}
	return f.Root
}
