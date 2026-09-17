package syntax

import "github.com/ayanozturk/go-php-parser/token"

// IfCondition returns the condition expression RedNode of an if-statement
// (or an else-if clause), or nil. Mirrors lowerIfStmt's condition detection
// without lowering to ast.Node.
func IfCondition(n *RedNode) *RedNode {
	if n == nil || (n.Kind() != KindIfStmt && n.Kind() != KindElseIfClause) {
		return nil
	}
	return firstExprChild(n)
}

// WhileCondition returns the condition expression RedNode of a while-statement,
// or nil. Mirrors lowerWhileStmt's condition detection.
func WhileCondition(n *RedNode) *RedNode {
	if n == nil || n.Kind() != KindWhileStmt {
		return nil
	}
	return firstExprOrNameChild(n)
}

// DoWhileCondition returns the condition expression RedNode of a do-while
// statement, or nil. Mirrors lowerDoWhileStmt's condition detection.
func DoWhileCondition(n *RedNode) *RedNode {
	if n == nil || n.Kind() != KindDoWhileStmt {
		return nil
	}
	return firstExprOrNameChild(n)
}

// MatchCondition returns the subject expression RedNode of a match-expression,
// or nil. Mirrors lowerMatchExpr's condition detection.
func MatchCondition(n *RedNode) *RedNode {
	if n == nil || n.Kind() != KindMatchExpr {
		return nil
	}
	seenLParen := false
	for _, c := range n.Children() {
		if isTokenType(c, token.T_LPAREN) {
			seenLParen = true
			continue
		}
		if seenLParen && (isExprKind(c.Kind()) || isNameKind(c.Kind())) {
			return c
		}
	}
	return nil
}

// ForConditions returns the (possibly multiple, comma-separated) condition
// clause expressions of a for-statement, or nil. Mirrors collectForClause's
// second clause without lowering to ast.Node.
func ForConditions(n *RedNode) []*RedNode {
	if n == nil || n.Kind() != KindForStmt {
		return nil
	}
	children := n.Children()
	i := 0
	for ; i < len(children); i++ {
		if isTokenType(children[i], token.T_LPAREN) {
			i++
			break
		}
	}
	_, i = forClauseChildren(children, i) // skip init clause
	conds, _ := forClauseChildren(children, i)
	return conds
}

func forClauseChildren(children []*RedNode, start int) ([]*RedNode, int) {
	var exprs []*RedNode
	for i := start; i < len(children); i++ {
		c := children[i]
		if isTokenType(c, token.T_SEMICOLON) || isTokenType(c, token.T_RPAREN) {
			return exprs, i + 1
		}
		if isTokenType(c, token.T_COMMA) {
			continue
		}
		if isExprKind(c.Kind()) || isNameKind(c.Kind()) {
			exprs = append(exprs, c)
		}
	}
	return exprs, len(children)
}

func firstExprOrNameChild(n *RedNode) *RedNode {
	for _, c := range n.Children() {
		if isExprKind(c.Kind()) || isNameKind(c.Kind()) {
			return c
		}
	}
	return nil
}
