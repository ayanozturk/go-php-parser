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

// MatchArmConditions returns the arm-condition expression RedNodes of a
// match-expression (the `default` arm contributes no expression, matching
// lowerMatchArm). Mirrors lowerMatchArm's condition detection.
func MatchArmConditions(n *RedNode) []*RedNode {
	if n == nil || n.Kind() != KindMatchExpr {
		return nil
	}
	var out []*RedNode
	for _, arm := range n.Children() {
		if arm.Kind() != KindMatchArm {
			continue
		}
		seenArrow := false
		for _, c := range arm.Children() {
			if isTokenType(c, token.T_DOUBLE_ARROW) {
				seenArrow = true
				continue
			}
			if !seenArrow && (isExprKind(c.Kind()) || isNameKind(c.Kind())) {
				out = append(out, c)
			}
		}
	}
	return out
}

// IfBody returns the raw "then" body RedNode of an if-statement (either a
// KindStatementList or a single statement), or nil. Pass to
// StatementBodyList for a normalized statement slice. Mirrors lowerIfStmt.
func IfBody(n *RedNode) *RedNode {
	if n == nil || n.Kind() != KindIfStmt {
		return nil
	}
	seenCond := false
	for _, c := range n.Children() {
		switch c.Kind() {
		case KindElseIfClause, KindElseClause, KindToken:
			continue
		default:
			if isExprKind(c.Kind()) && !seenCond {
				seenCond = true
				continue
			}
			if seenCond && (c.Kind() == KindStatementList || isStmtKind(c.Kind())) {
				return c
			}
		}
	}
	return nil
}

// IfElseIfs returns the ElseIfClause children of an if-statement.
func IfElseIfs(n *RedNode) []*RedNode {
	if n == nil || n.Kind() != KindIfStmt {
		return nil
	}
	var out []*RedNode
	for _, c := range n.Children() {
		if c.Kind() == KindElseIfClause {
			out = append(out, c)
		}
	}
	return out
}

// IfElse returns the ElseClause child of an if-statement, or nil.
func IfElse(n *RedNode) *RedNode {
	if n == nil || n.Kind() != KindIfStmt {
		return nil
	}
	for _, c := range n.Children() {
		if c.Kind() == KindElseClause {
			return c
		}
	}
	return nil
}

// ElseIfBody returns the raw body RedNode of an else-if clause. Mirrors
// lowerElseIfClause.
func ElseIfBody(n *RedNode) *RedNode {
	if n == nil || n.Kind() != KindElseIfClause {
		return nil
	}
	seenCond := false
	for _, c := range n.Children() {
		if c.Kind() == KindToken {
			continue
		}
		if isExprKind(c.Kind()) && !seenCond {
			seenCond = true
			continue
		}
		if c.Kind() == KindStatementList || isStmtKind(c.Kind()) {
			return c
		}
	}
	return nil
}

// ElseBody returns the raw body RedNode of an else clause. Mirrors
// lowerElseClause.
func ElseBody(n *RedNode) *RedNode {
	if n == nil || n.Kind() != KindElseClause {
		return nil
	}
	for _, c := range n.Children() {
		if c.Kind() == KindToken {
			continue
		}
		if c.Kind() == KindStatementList || isStmtKind(c.Kind()) {
			return c
		}
	}
	return nil
}

// WhileBody returns the raw body RedNode of a while-statement. Mirrors
// lowerWhileStmt.
func WhileBody(n *RedNode) *RedNode {
	if n == nil || n.Kind() != KindWhileStmt {
		return nil
	}
	seenCond := false
	for _, c := range n.Children() {
		if c.Kind() == KindToken {
			continue
		}
		if (isExprKind(c.Kind()) || isNameKind(c.Kind())) && !seenCond {
			seenCond = true
			continue
		}
		if c.Kind() == KindStatementList || isStmtKind(c.Kind()) {
			return c
		}
	}
	return nil
}

// DoWhileBody returns the raw body RedNode of a do-while statement. Mirrors
// lowerDoWhileStmt.
func DoWhileBody(n *RedNode) *RedNode {
	if n == nil || n.Kind() != KindDoWhileStmt {
		return nil
	}
	for _, c := range n.Children() {
		if c.Kind() == KindToken {
			continue
		}
		if c.Kind() == KindStatementList || isStmtKind(c.Kind()) {
			return c
		}
	}
	return nil
}

// ForBody returns the raw body RedNode of a for-statement. Mirrors
// lowerForStmt.
func ForBody(n *RedNode) *RedNode {
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
	_, i = forClauseChildren(children, i) // init
	_, i = forClauseChildren(children, i) // condition
	_, i = forClauseChildren(children, i) // update
	for ; i < len(children); i++ {
		c := children[i]
		if c.Kind() == KindToken {
			continue
		}
		if c.Kind() == KindStatementList || isStmtKind(c.Kind()) {
			return c
		}
	}
	return nil
}

// StatementBodyList normalizes a body RedNode (either a KindStatementList or
// a single statement, as returned by IfBody/WhileBody/etc.) into a flat
// slice of statement RedNodes. Mirrors lowerControlBody.
func StatementBodyList(n *RedNode) []*RedNode {
	if n == nil {
		return nil
	}
	if n.Kind() == KindStatementList {
		var out []*RedNode
		for _, c := range n.Children() {
			if c == nil || c.Kind() == KindToken {
				continue
			}
			out = append(out, c)
		}
		return out
	}
	return []*RedNode{n}
}

// FunctionBody returns the KindStatementList body of a function/method
// declaration, or nil (e.g. abstract methods, or index-mode KindTokenList).
func FunctionBody(n *RedNode) *RedNode {
	if n == nil || (n.Kind() != KindFunctionDecl && n.Kind() != KindMethodDecl) {
		return nil
	}
	for _, c := range n.Children() {
		if c.Kind() == KindStatementList {
			return c
		}
	}
	return nil
}

// ClassMethods returns the KindFunctionDecl/KindMethodDecl member children of
// a class declaration.
func ClassMethods(n *RedNode) []*RedNode {
	if n == nil || n.Kind() != KindClassDecl {
		return nil
	}
	var members *RedNode
	for _, c := range n.Children() {
		if c.Kind() == KindMemberList {
			members = c
			break
		}
	}
	if members == nil {
		return nil
	}
	var out []*RedNode
	for _, m := range members.Children() {
		if m.Kind() == KindFunctionDecl || m.Kind() == KindMethodDecl {
			out = append(out, m)
		}
	}
	return out
}

// CallIsMethodLike reports whether a call-expression's callee is a member
// access (instance, nullsafe, or static), meaning it lowers to
// ast.MethodCallNode rather than ast.FunctionCallNode. Mirrors the callee
// classification in lowerCallExpr.
func CallIsMethodLike(n *RedNode) bool {
	if n == nil || n.Kind() != KindCallExpr {
		return false
	}
	for _, c := range n.Children() {
		switch c.Kind() {
		case KindArgList, KindToken:
			continue
		case KindMemberAccessExpr, KindNullsafeMemberAccessExpr, KindStaticMemberAccessExpr:
			return true
		default:
			if isExprKind(c.Kind()) || isNameKind(c.Kind()) {
				return false
			}
		}
	}
	return false
}

// CallArgList returns the KindArgList child of a call-expression, or nil.
func CallArgList(n *RedNode) *RedNode {
	if n == nil {
		return nil
	}
	return n.FirstChildOfKind(KindArgList)
}

func firstExprOrNameChild(n *RedNode) *RedNode {
	for _, c := range n.Children() {
		if isExprKind(c.Kind()) || isNameKind(c.Kind()) {
			return c
		}
	}
	return nil
}

// ExpressionStmtExpr returns the inner expression RedNode of an expression
// statement, or nil. Mirrors lowerStmt's KindExpressionStmt handling
// (firstExprChild).
func ExpressionStmtExpr(n *RedNode) *RedNode {
	if n == nil || n.Kind() != KindExpressionStmt {
		return nil
	}
	return firstExprChild(n)
}

// NamespaceBody returns a namespace declaration's body nodes, handling both
// the braced form (`namespace Foo { ... }`, a KindStatementList child) and
// the unbraced form (`namespace Foo;`, which folds the following top-level
// siblings into the namespace until the next KindNamespaceDecl). Mirrors
// lowerNamespace's exact child-classification logic. siblings/idx identify
// n's position among its own siblings (e.g. a KindFile's children); consumed
// reports how many following siblings the unbraced form absorbed.
func NamespaceBody(n *RedNode, siblings []*RedNode, idx int) (body []*RedNode, consumed int) {
	if n == nil {
		return nil, 0
	}
	for _, c := range n.Children() {
		if c.Kind() == KindStatementList {
			return c.Children(), 0
		}
	}
	for j := idx + 1; j < len(siblings); j++ {
		sib := siblings[j]
		if sib.Kind() == KindNamespaceDecl {
			break
		}
		consumed++
		body = append(body, sib)
	}
	return body, consumed
}

// IsNameKind reports whether k is one of the name-node kinds (unqualified,
// qualified, fully-qualified, relative, or the generic KindName).
func IsNameKind(k Kind) bool {
	return isNameKind(k)
}

// ClauseNames returns the name text of every name-kind child of clause (e.g.
// a KindExtendsClause or KindImplementsClause), in source order. Mirrors the
// lowering path's clauseNames used by lowerClass/lowerInterface.
func ClauseNames(clause *RedNode) []string {
	return clauseNames(clause)
}

// UnqualifiedTail returns the last backslash-separated segment of path, e.g.
// "Foo" for both "Foo" and "Ns\Foo". Mirrors the lowering path's
// unqualifiedTail helper used for class/interface declaration names.
func UnqualifiedTail(path string) string {
	return unqualifiedTail(path)
}
