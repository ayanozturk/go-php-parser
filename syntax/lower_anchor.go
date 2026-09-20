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
	for _, c := range n.Children() {
		if isTokenType(c, token.T_FUNCTION) {
			fnStart, _ := nodePos(file, c)
			start = fnStart.Offset
			break
		}
	}
	return start, end
}

// ClassDeclSpanOffsets mirrors lowerClass (full decl nodePos).
func ClassDeclSpanOffsets(n *RedNode, file *File) (start, end int) {
	return ContentSpanOffsets(n, file)
}
