package syntax

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/token"
)

// leadingDocFromNode mirrors classic harvestTriviaDoc / consumeCurrentDoc for a
// declaration red node: find the first significant token (skipping
// KindAttributeList), take the last T_DOC_COMMENT in its LeadingTrivia, and
// parse it via ExtractPHPDocFromComment.
func leadingDocFromNode(n *RedNode) *ast.PHPDocNode {
	tokNode := firstSignificantToken(n)
	if tokNode == nil || tokNode.Green == nil {
		return nil
	}
	tok, ok := tokNode.Green.Token()
	if !ok {
		return nil
	}
	literal := lastDocCommentLiteral(tok.LeadingTrivia)
	if literal == "" {
		return nil
	}
	doc := ast.ExtractPHPDocFromComment(literal)
	if doc == nil {
		return nil
	}
	doc.Pos = spanStart(tokNode.File, tokNode.Span())
	return doc
}

// firstSignificantToken returns the first token leaf under n, skipping
// KindAttributeList children at each level. Walks green children without
// allocating intermediate RedNode wrappers; binds only the winning token
// (or returns n when n itself is the token).
func firstSignificantToken(n *RedNode) *RedNode {
	if n == nil || n.Green == nil {
		return nil
	}
	if n.Green.IsToken() {
		return n
	}
	var foundGreen *GreenNode
	var foundOff int
	var search func(green *GreenNode, offset int) bool
	search = func(green *GreenNode, offset int) bool {
		if green == nil {
			return false
		}
		if green.IsToken() {
			foundGreen, foundOff = green, offset
			return true
		}
		off := offset
		for _, g := range green.children {
			if g == nil {
				continue
			}
			if g.Kind() == KindAttributeList {
				off += g.width
				continue
			}
			if search(g, off) {
				return true
			}
			off += g.width
		}
		return false
	}
	if !search(n.Green, n.Offset) {
		return nil
	}
	return &RedNode{File: n.File, Green: foundGreen, Offset: foundOff}
}

func lastDocCommentLiteral(trivia []token.Token) string {
	literal := ""
	for _, tr := range trivia {
		if tr.Type == token.T_DOC_COMMENT {
			literal = tr.Literal
		}
	}
	return literal
}
