package syntax

import (
	"github.com/ayanozturk/go-php-parser/token"
)

// File is the red root for a parsed source file.
type File struct {
	Source   []byte
	Lines    token.LineTable
	Green    *GreenNode
	Tokens   []token.Token
	Root     *RedNode
}

// RedNode is a typed view over a green node with parent and absolute span.
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
	return r.Green.Kind
}

func (r *RedNode) Span() Span {
	if r == nil || r.Green == nil {
		return Span{}
	}
	return Span{Start: r.Offset, End: r.Offset + r.Green.Width}
}

func (r *RedNode) Text() string {
	s := r.Span()
	if s.Start < 0 || s.End > len(r.File.Source) || s.Start > s.End {
		return ""
	}
	return string(r.File.Source[s.Start:s.End])
}

func (r *RedNode) Children() []*RedNode {
	if r == nil || r.Green == nil || len(r.Green.Children) == 0 {
		return nil
	}
	out := make([]*RedNode, 0, len(r.Green.Children))
	off := r.Offset
	for _, g := range r.Green.Children {
		if g == nil {
			continue
		}
		child := &RedNode{File: r.File, Parent: r, Green: g, Offset: off}
		out = append(out, child)
		off += g.Width
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

func (r *RedNode) Tokens() []token.Token {
	var out []token.Token
	var walk func(*RedNode)
	walk = func(n *RedNode) {
		if n == nil || n.Green == nil {
			return
		}
		if n.Green.IsToken() {
			out = append(out, *n.Green.Token)
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
