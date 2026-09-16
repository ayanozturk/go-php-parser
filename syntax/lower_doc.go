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
// KindAttributeList children at each level.
func firstSignificantToken(n *RedNode) *RedNode {
	if n == nil {
		return nil
	}
	if n.Green != nil && n.Green.IsToken() {
		return n
	}
	for _, c := range n.Children() {
		if c.Kind() == KindAttributeList {
			continue
		}
		if found := firstSignificantToken(c); found != nil {
			return found
		}
	}
	return nil
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
