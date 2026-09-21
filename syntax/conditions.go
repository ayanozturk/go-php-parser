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
	var cond *RedNode
	seenLParen := false
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if isGreenTokenType(green, token.T_LPAREN) {
			seenLParen = true
			return true
		}
		k := green.Kind()
		if seenLParen && cond == nil && (isExprKind(k) || isNameKind(k)) {
			cond = n.bindChild(green, offset)
			return false
		}
		return true
	})
	return cond
}

// ForConditions returns the (possibly multiple, comma-separated) condition
// clause expressions of a for-statement, or nil. Mirrors collectForClause's
// second clause without lowering to ast.Node.
func ForConditions(n *RedNode) []*RedNode {
	if n == nil || n.Kind() != KindForStmt {
		return nil
	}
	start := forStmtIndexAfterLParen(n)
	_, start = forClauseChildrenAt(n, start) // skip init clause
	conds, _ := forClauseChildrenAt(n, start)
	return conds
}

func forClauseChildrenAt(n *RedNode, start int) ([]*RedNode, int) {
	var exprs []*RedNode
	i := 0
	next := start
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if i < start {
			i++
			return true
		}
		if isGreenTokenType(green, token.T_SEMICOLON) || isGreenTokenType(green, token.T_RPAREN) {
			next = i + 1
			return false
		}
		if isGreenTokenType(green, token.T_COMMA) {
			i++
			return true
		}
		k := green.Kind()
		if isExprKind(k) || isNameKind(k) {
			exprs = append(exprs, n.bindChild(green, offset))
		}
		i++
		return true
	})
	if next == start && i > start {
		next = i
	}
	return exprs, next
}

func forStmtIndexAfterLParen(n *RedNode) int {
	i := 0
	after := 0
	found := false
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if isGreenTokenType(green, token.T_LPAREN) {
			after = i + 1
			found = true
			return false
		}
		i++
		return true
	})
	if found {
		return after
	}
	return i
}

// MatchArmConditions returns the arm-condition expression RedNodes of a
// match-expression (the `default` arm contributes no expression, matching
// lowerMatchArm). Mirrors lowerMatchArm's condition detection.
func MatchArmConditions(n *RedNode) []*RedNode {
	if n == nil || n.Kind() != KindMatchExpr {
		return nil
	}
	var out []*RedNode
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if green.Kind() != KindMatchArm {
			return true
		}
		arm := n.bindChild(green, offset)
		seenArrow := false
		arm.ForEachChildDesc(func(green *GreenNode, offset int) bool {
			if isGreenTokenType(green, token.T_DOUBLE_ARROW) {
				seenArrow = true
				return true
			}
			k := green.Kind()
			if !seenArrow && (isExprKind(k) || isNameKind(k)) {
				out = append(out, arm.bindChild(green, offset))
			}
			return true
		})
		return true
	})
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
	var body *RedNode
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		switch k {
		case KindElseIfClause, KindElseClause, KindToken:
			return true
		default:
			if isExprKind(k) && !seenCond {
				seenCond = true
				return true
			}
			if seenCond && body == nil && (k == KindStatementList || isStmtKind(k)) {
				body = n.bindChild(green, offset)
				return false
			}
		}
		return true
	})
	return body
}

// IfElseIfs returns the ElseIfClause children of an if-statement.
func IfElseIfs(n *RedNode) []*RedNode {
	if n == nil || n.Kind() != KindIfStmt {
		return nil
	}
	var out []*RedNode
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if green.Kind() == KindElseIfClause {
			out = append(out, n.bindChild(green, offset))
		}
		return true
	})
	return out
}

// IfElse returns the ElseClause child of an if-statement, or nil.
func IfElse(n *RedNode) *RedNode {
	if n == nil || n.Kind() != KindIfStmt {
		return nil
	}
	var els *RedNode
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if green.Kind() == KindElseClause {
			els = n.bindChild(green, offset)
			return false
		}
		return true
	})
	return els
}

// ElseIfBody returns the raw body RedNode of an else-if clause. Mirrors
// lowerElseIfClause.
func ElseIfBody(n *RedNode) *RedNode {
	if n == nil || n.Kind() != KindElseIfClause {
		return nil
	}
	seenCond := false
	var body *RedNode
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		if k == KindToken {
			return true
		}
		if isExprKind(k) && !seenCond {
			seenCond = true
			return true
		}
		if body == nil && (k == KindStatementList || isStmtKind(k)) {
			body = n.bindChild(green, offset)
			return false
		}
		return true
	})
	return body
}

// ElseBody returns the raw body RedNode of an else clause. Mirrors
// lowerElseClause.
func ElseBody(n *RedNode) *RedNode {
	if n == nil || n.Kind() != KindElseClause {
		return nil
	}
	var body *RedNode
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		if k == KindToken {
			return true
		}
		if body == nil && (k == KindStatementList || isStmtKind(k)) {
			body = n.bindChild(green, offset)
			return false
		}
		return true
	})
	return body
}

// WhileBody returns the raw body RedNode of a while-statement. Mirrors
// lowerWhileStmt.
func WhileBody(n *RedNode) *RedNode {
	if n == nil || n.Kind() != KindWhileStmt {
		return nil
	}
	seenCond := false
	var body *RedNode
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		if k == KindToken {
			return true
		}
		if (isExprKind(k) || isNameKind(k)) && !seenCond {
			seenCond = true
			return true
		}
		if body == nil && (k == KindStatementList || isStmtKind(k)) {
			body = n.bindChild(green, offset)
			return false
		}
		return true
	})
	return body
}

// DoWhileBody returns the raw body RedNode of a do-while statement. Mirrors
// lowerDoWhileStmt.
func DoWhileBody(n *RedNode) *RedNode {
	if n == nil || n.Kind() != KindDoWhileStmt {
		return nil
	}
	var body *RedNode
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		if k == KindToken {
			return true
		}
		if body == nil && (k == KindStatementList || isStmtKind(k)) {
			body = n.bindChild(green, offset)
			return false
		}
		return true
	})
	return body
}

// ForBody returns the raw body RedNode of a for-statement. Mirrors
// lowerForStmt.
func ForBody(n *RedNode) *RedNode {
	if n == nil || n.Kind() != KindForStmt {
		return nil
	}
	start := forStmtIndexAfterLParen(n)
	_, start = forClauseChildrenAt(n, start) // init
	_, start = forClauseChildrenAt(n, start) // condition
	_, start = forClauseChildrenAt(n, start) // update
	var body *RedNode
	i := 0
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if i < start {
			i++
			return true
		}
		k := green.Kind()
		if k == KindToken {
			i++
			return true
		}
		if body == nil && (k == KindStatementList || isStmtKind(k)) {
			body = n.bindChild(green, offset)
			return false
		}
		i++
		return true
	})
	return body
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
		n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
			if green == nil || green.Kind() == KindToken {
				return true
			}
			out = append(out, n.bindChild(green, offset))
			return true
		})
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
	var body *RedNode
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if green.Kind() == KindStatementList {
			body = n.bindChild(green, offset)
			return false
		}
		return true
	})
	return body
}

// ClassMethods returns the KindFunctionDecl/KindMethodDecl member children of
// a class declaration.
func ClassMethods(n *RedNode) []*RedNode {
	if n == nil || n.Kind() != KindClassDecl {
		return nil
	}
	var members *RedNode
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if green.Kind() == KindMemberList {
			members = n.bindChild(green, offset)
			return false
		}
		return true
	})
	if members == nil {
		return nil
	}
	var out []*RedNode
	members.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		if k == KindFunctionDecl || k == KindMethodDecl {
			out = append(out, members.bindChild(green, offset))
		}
		return true
	})
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
	methodLike := false
	decided := false
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		switch k {
		case KindArgList, KindToken:
			return true
		case KindMemberAccessExpr, KindNullsafeMemberAccessExpr, KindStaticMemberAccessExpr:
			methodLike = true
			decided = true
			return false
		default:
			if isExprKind(k) || isNameKind(k) {
				methodLike = false
				decided = true
				return false
			}
		}
		return true
	})
	if decided {
		return methodLike
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
	if n == nil {
		return nil
	}
	var found *RedNode
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		if isExprKind(k) || isNameKind(k) {
			found = n.bindChild(green, offset)
			return false
		}
		return true
	})
	return found
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
	var stmtList *RedNode
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if green.Kind() == KindStatementList {
			stmtList = n.bindChild(green, offset)
			return false
		}
		return true
	})
	if stmtList != nil {
		stmtList.ForEachChildDesc(func(green *GreenNode, offset int) bool {
			if green == nil || green.Kind() == KindToken {
				return true
			}
			body = append(body, stmtList.bindChild(green, offset))
			return true
		})
		return body, 0
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
