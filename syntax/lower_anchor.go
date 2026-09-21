package syntax

import "github.com/ayanozturk/go-php-parser/token"

// ContentSpanOffsets returns half-open byte offsets matching nodePos content start/end.
func ContentSpanOffsets(n *RedNode, file *File) (start, end int) {
	if n == nil {
		return 0, 0
	}
	s, e := nodePos(file, n)
	return s.Offset, e.Offset
}

// FunctionDeclSpanOffsets mirrors lowerFunction Pos/EndPos:
// start at T_FUNCTION token content start; end at decl content end.
func FunctionDeclSpanOffsets(n *RedNode, file *File) (start, end int) {
	if n == nil {
		return 0, 0
	}
	pos, endPos := nodePos(file, n)
	start, end = pos.Offset, endPos.Offset
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if isGreenTokenType(green, token.T_FUNCTION) {
			fnStart, _ := nodePos(file, n.bindChild(green, offset))
			start = fnStart.Offset
			return false
		}
		return true
	})
	return start, end
}

// ClassDeclSpanOffsets mirrors lowerClass (full decl nodePos).
func ClassDeclSpanOffsets(n *RedNode, file *File) (start, end int) {
	return ContentSpanOffsets(n, file)
}
