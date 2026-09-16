package syntax

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/token"
)

// positionAt converts a byte offset to a classic 1-based line/column Position.
func positionAt(file *File, offset int) ast.Position {
	if file == nil {
		return ast.Position{Offset: offset}
	}
	line, col := offsetLineCol(file.Lines, offset)
	return ast.Position{Line: line + 1, Column: col + 1, Offset: offset}
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
	s := n.Span()
	return spanStart(file, s), spanEnd(file, s)
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
