package syntax

import (
	"sync"

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

	// runeColCache memoizes the last (line, byteOffsetWithinLine, runeColumn)
	// computed by positionAt, so a forward pass over many nodes on the same
	// line (e.g. lowering a huge single-line array literal) only rune-counts
	// the delta since the last call instead of re-scanning from the start of
	// the line every time. Without this, a file with N nodes packed onto one
	// very long line costs O(N^2) instead of O(N) to position (each call
	// rescans from lineStart, and lineStart never advances within the line).
	// Mutex-protected because multiple rules may query positions on the same
	// *File concurrently.
	runeColCache struct {
		mu     sync.Mutex
		valid  bool
		line   int // 0-based line the cache applies to
		offset int // byte offset within Source the cached column ends at
		col    int // 0-based rune column at `offset`
	}

	childDescCache struct {
		mu    sync.Mutex
		byKey map[redNodeKey][]redChildDesc
	}
	contentStartCache struct {
		mu    sync.Mutex
		byKey map[redNodeKey]int
	}
}

type redChildDesc struct {
	green  *GreenNode
	offset int
}

type redNodeKey struct {
	green  *GreenNode
	offset int
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

func buildChildDescs(green *GreenNode, offset int) []redChildDesc {
	if green == nil || len(green.children) == 0 {
		return nil
	}
	out := make([]redChildDesc, 0, len(green.children))
	off := offset
	for _, g := range green.children {
		if g == nil {
			continue
		}
		out = append(out, redChildDesc{green: g, offset: off})
		off += g.width
	}
	return out
}

func (f *File) childDescs(parentGreen *GreenNode, parentOff int) []redChildDesc {
	if parentGreen == nil || len(parentGreen.children) == 0 {
		return nil
	}
	key := redNodeKey{green: parentGreen, offset: parentOff}
	f.childDescCache.mu.Lock()
	if f.childDescCache.byKey == nil {
		f.childDescCache.byKey = make(map[redNodeKey][]redChildDesc)
	}
	if descs, ok := f.childDescCache.byKey[key]; ok {
		f.childDescCache.mu.Unlock()
		return descs
	}
	f.childDescCache.mu.Unlock()

	descs := buildChildDescs(parentGreen, parentOff)

	f.childDescCache.mu.Lock()
	f.childDescCache.byKey[key] = descs
	f.childDescCache.mu.Unlock()
	return descs
}

func (f *File) cachedContentStart(key redNodeKey) (int, bool) {
	f.contentStartCache.mu.Lock()
	defer f.contentStartCache.mu.Unlock()
	if f.contentStartCache.byKey == nil {
		return 0, false
	}
	v, ok := f.contentStartCache.byKey[key]
	return v, ok
}

func (f *File) storeContentStart(key redNodeKey, start int) {
	f.contentStartCache.mu.Lock()
	if f.contentStartCache.byKey == nil {
		f.contentStartCache.byKey = make(map[redNodeKey]int)
	}
	f.contentStartCache.byKey[key] = start
	f.contentStartCache.mu.Unlock()
}

func (r *RedNode) Children() []*RedNode {
	if r == nil || r.Green == nil || len(r.Green.children) == 0 {
		return nil
	}
	var descs []redChildDesc
	if r.File != nil {
		descs = r.File.childDescs(r.Green, r.Offset)
	} else {
		descs = buildChildDescs(r.Green, r.Offset)
	}
	out := make([]*RedNode, 0, len(descs))
	for _, d := range descs {
		out = append(out, &RedNode{File: r.File, Parent: r, Green: d.green, Offset: d.offset})
	}
	return out
}

// ForEachChild visits each child without allocating a []*RedNode slice.
// If fn returns false, iteration stops.
func (r *RedNode) ForEachChild(fn func(*RedNode) bool) {
	if r == nil || r.Green == nil || len(r.Green.children) == 0 || fn == nil {
		return
	}
	var descs []redChildDesc
	if r.File != nil {
		descs = r.File.childDescs(r.Green, r.Offset)
	} else {
		descs = buildChildDescs(r.Green, r.Offset)
	}
	for _, d := range descs {
		child := &RedNode{File: r.File, Parent: r, Green: d.green, Offset: d.offset}
		if !fn(child) {
			return
		}
	}
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
