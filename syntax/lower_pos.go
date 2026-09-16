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
	runeCol := utf8.RuneCount(file.Source[lineStart:offset])
	return ast.Position{Line: line + 1, Column: runeCol + 1, Offset: offset}
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
	if n.Green != nil && n.Green.IsToken() {
		if tok, ok := n.Green.Token(); ok {
			return sp.Start + leadingTriviaWidth(tok)
		}
		return sp.Start
	}
	if t := firstTokenDescendant(n); t != nil {
		return contentStartOffset(t)
	}
	return sp.Start
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

func firstTokenDescendant(n *RedNode) *RedNode {
	if n == nil {
		return nil
	}
	if n.Green != nil && n.Green.IsToken() {
		return n
	}
	for _, c := range n.Children() {
		if t := firstTokenDescendant(c); t != nil {
			return t
		}
	}
	return nil
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
