package syntax

import (
	"unicode/utf8"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/token"
)

// positionAt converts a byte offset to a classic 1-based line/column Position
// using rune columns (matching the shared lexer).
func positionAt(file *File, offset int) ast.Position {
	if file == nil {
		return ast.Position{Offset: offset}
	}
	if offset < 0 {
		offset = 0
	}
	if offset > len(file.Source) {
		offset = len(file.Source)
	}
	line, _ := offsetLineCol(file.Lines, offset)
	lineStart := 0
	if line < len(file.Lines) {
		lineStart = file.Lines[line]
	}
	if lineStart > offset {
		lineStart = offset
	}
	runeCol := runeColumnAt(file, line, lineStart, offset)
	return ast.Position{Line: line + 1, Column: runeCol + 1, Offset: offset}
}

// runeColumnAt returns the 0-based rune column at offset (relative to
// lineStart) on the given 0-based line, exploiting the common case where
// callers query positions in roughly increasing offset order on the same
// line (e.g. lowering array elements left to right on one very long line).
// A cache hit rune-counts only the delta since the last call instead of
// rescanning from lineStart every time - the difference between O(N) and
// O(N^2) total cost across N calls on the same long line.
func runeColumnAt(file *File, line, lineStart, offset int) int {
	c := &file.runeColCache
	c.mu.Lock()
	defer c.mu.Unlock()

	// Splitting a rune-count into forward/backward deltas is only valid when
	// both the cached offset and the new offset sit on rune boundaries. Real
	// callers always pass lexer token-boundary offsets, which is guaranteed
	// for valid UTF-8 source; but PHP source is frequently not valid UTF-8
	// (legacy Latin-1/Windows-1252 string literals are common), and for such
	// files a byte can look like a UTF-8 continuation byte without actually
	// being part of a rune the lexer treated specially. Falling back to a
	// direct scan whenever either boundary isn't unambiguously safe keeps the
	// fast path exact instead of merely fast.
	if c.valid && c.line == line && isRuneBoundary(file.Source, c.offset) && isRuneBoundary(file.Source, offset) {
		if offset >= c.offset {
			delta := utf8.RuneCount(file.Source[c.offset:offset])
			col := c.col + delta
			c.offset, c.col = offset, col
			return col
		}
		// Lowering a nested container computes its own (start, end) before
		// descending into children, then each child's (start, end) before its
		// own children - so offset routinely jumps backward onto a position
		// closer to lineStart than to the cache. Scan whichever gap is
		// smaller instead of always rescanning from lineStart.
		if offset-lineStart < c.offset-offset {
			col := utf8.RuneCount(file.Source[lineStart:offset])
			c.offset, c.col = offset, col
			return col
		}
		delta := utf8.RuneCount(file.Source[offset:c.offset])
		col := c.col - delta
		c.offset, c.col = offset, col
		return col
	}

	col := utf8.RuneCount(file.Source[lineStart:offset])
	c.valid, c.line, c.offset, c.col = true, line, offset, col
	return col
}

// isRuneBoundary reports whether pos is not in the middle of a UTF-8 encoded
// rune - true for the start of a rune, EOF, and (importantly) any byte that
// isn't valid UTF-8 at all, since a non-UTF-8 byte is decoded as its own
// single-byte RuneError rather than a continuation.
func isRuneBoundary(src []byte, pos int) bool {
	if pos <= 0 || pos >= len(src) {
		return true
	}
	return utf8.RuneStart(src[pos])
}

func spanStart(file *File, s Span) ast.Position {
	return positionAt(file, s.Start)
}

func spanEnd(file *File, s Span) ast.Position {
	return positionAt(file, s.End)
}

func nodePos(file *File, n *RedNode) (start, end ast.Position) {
	if n == nil {
		return ast.Position{}, ast.Position{}
	}
	sp := n.Span()
	startOff := contentStartOffset(n)
	endOff := contentEndOffset(n)
	if startOff < sp.Start {
		startOff = sp.Start
	}
	if endOff > sp.End || endOff < startOff {
		endOff = sp.End
	}
	return positionAt(file, startOff), positionAt(file, endOff)
}

// contentStartOffset skips leading trivia on the first token so Pos matches
// classic lexer coordinates (T_FUNCTION starts at `f`, not the preceding `\n`).
func contentStartOffset(n *RedNode) int {
	if n == nil {
		return 0
	}
	sp := n.Span()
	var start int
	if n.Green != nil && n.Green.IsToken() {
		if tok, ok := n.Green.Token(); ok {
			start = sp.Start + leadingTriviaWidth(tok)
		} else {
			start = sp.Start
		}
	} else if tg, toff := firstTokenGreenDescendant(n.Green, n.Offset); tg != nil {
		start = contentStartOffsetGreen(tg, toff)
	} else {
		start = sp.Start
	}
	return start
}

func contentStartOffsetGreen(g *GreenNode, off int) int {
	if g == nil {
		return off
	}
	if !g.IsToken() {
		if tg, toff := firstTokenGreenDescendant(g, off); tg != nil {
			return contentStartOffsetGreen(tg, toff)
		}
		return off
	}
	tok, ok := g.Token()
	if !ok {
		return off
	}
	return off + leadingTriviaWidth(tok)
}

func contentEndOffset(n *RedNode) int {
	if n == nil {
		return 0
	}
	sp := n.Span()
	if n.Green != nil && n.Green.IsToken() {
		if tok, ok := n.Green.Token(); ok {
			end := sp.End - trailingTriviaWidth(tok)
			if end < sp.Start {
				return sp.End
			}
			return end
		}
	}
	return sp.End
}

func leadingTriviaWidth(tok token.Token) int {
	w := 0
	for _, tr := range tok.LeadingTrivia {
		w += tr.Width()
		if tr.Width() == 0 {
			w += len(tr.Literal)
		}
	}
	return w
}

func trailingTriviaWidth(tok token.Token) int {
	w := 0
	for _, tr := range tok.TrailingTrivia {
		w += tr.Width()
		if tr.Width() == 0 {
			w += len(tr.Literal)
		}
	}
	return w
}

// firstTokenGreenDescendant returns the first token green under g at off and its
// absolute offset, without allocating RedNode wrappers.
func firstTokenGreenDescendant(g *GreenNode, off int) (*GreenNode, int) {
	if g == nil {
		return nil, 0
	}
	if g.IsToken() {
		return g, off
	}
	childOff := off
	for _, c := range g.children {
		if c == nil {
			continue
		}
		if tg, toff := firstTokenGreenDescendant(c, childOff); tg != nil {
			return tg, toff
		}
		childOff += c.width
	}
	return nil, 0
}

// offsetLineCol returns 0-based line and byte column for a byte offset.
func offsetLineCol(lines token.LineTable, offset int) (line, col int) {
	if len(lines) == 0 {
		return 0, offset
	}
	if offset < 0 {
		offset = 0
	}
	lo, hi := 0, len(lines)-1
	for lo <= hi {
		mid := (lo + hi) / 2
		if lines[mid] <= offset {
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	idx := hi
	if idx < 0 {
		idx = 0
	}
	return idx, offset - lines[idx]
}
