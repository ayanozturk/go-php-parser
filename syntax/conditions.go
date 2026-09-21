package syntax

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/token"
)

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
			cond = &RedNode{File: n.File, Green: green, Offset: offset}
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
			exprs = append(exprs, &RedNode{File: n.File, Green: green, Offset: offset})
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
		arm := &RedNode{File: n.File, Green: green, Offset: offset}
		seenArrow := false
		arm.ForEachChildDesc(func(green *GreenNode, offset int) bool {
			if isGreenTokenType(green, token.T_DOUBLE_ARROW) {
				seenArrow = true
				return true
			}
			k := green.Kind()
			if !seenArrow && (isExprKind(k) || isNameKind(k)) {
				out = append(out, &RedNode{File: arm.File, Green: green, Offset: offset})
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
				body = &RedNode{File: n.File, Green: green, Offset: offset}
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
			out = append(out, &RedNode{File: n.File, Green: green, Offset: offset})
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
			els = &RedNode{File: n.File, Green: green, Offset: offset}
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
			body = &RedNode{File: n.File, Green: green, Offset: offset}
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
			body = &RedNode{File: n.File, Green: green, Offset: offset}
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
			body = &RedNode{File: n.File, Green: green, Offset: offset}
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
			body = &RedNode{File: n.File, Green: green, Offset: offset}
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
			body = &RedNode{File: n.File, Green: green, Offset: offset}
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
			out = append(out, &RedNode{File: n.File, Green: green, Offset: offset})
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
			body = &RedNode{File: n.File, Green: green, Offset: offset}
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
			members = &RedNode{File: n.File, Green: green, Offset: offset}
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
			out = append(out, &RedNode{File: members.File, Green: green, Offset: offset})
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

// CallCallee returns the callee expression/name child of a KindCallExpr
// (first expr/name child that is not inside KindArgList), or nil for
// builtin-token calls (isset/empty/…) that have no separate callee node.
func CallCallee(n *RedNode) *RedNode {
	if n == nil || n.Kind() != KindCallExpr {
		return nil
	}
	var callee *RedNode
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		if k == KindArgList || k == KindToken {
			return true
		}
		if isExprKind(k) || isNameKind(k) {
			callee = &RedNode{File: n.File, Green: green, Offset: offset}
			return false
		}
		return true
	})
	return callee
}

// CallArgs returns KindArg / KindNamedArg children of a call's ArgList, in
// source order (excludes punctuation tokens).
func CallArgs(n *RedNode) []*RedNode {
	list := CallArgList(n)
	if list == nil {
		return nil
	}
	var out []*RedNode
	list.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		switch green.Kind() {
		case KindArg, KindNamedArg:
			out = append(out, &RedNode{File: list.File, Green: green, Offset: offset})
		}
		return true
	})
	return out
}

// ArgExpr returns the value expression of a KindArg or KindNamedArg (the
// expr/name child; for named args this is the value after `:`). Returns nil
// when the arg is missing or only an unpack ellipsis without an expression.
func ArgExpr(n *RedNode) *RedNode {
	if n == nil {
		return nil
	}
	switch n.Kind() {
	case KindArg, KindNamedArg:
	default:
		return nil
	}
	var expr *RedNode
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		if isExprKind(k) || isNameKind(k) {
			expr = &RedNode{File: n.File, Green: green, Offset: offset}
			// NamedArg may have a name identifier before the value; keep
			// scanning so the last expr/name wins (value side).
			if n.Kind() == KindNamedArg {
				return true
			}
			return false
		}
		return true
	})
	return expr
}

// LiteralStringValue returns the decoded contents of a constant string
// literal expression (KindLiteralExpr with a quoted string token), matching
// lowering → *ast.StringLiteral.Value. Returns false for non-string literals
// and interpolated KindStringLiteral forms.
func LiteralStringValue(n *RedNode) (string, bool) {
	if n == nil {
		return "", false
	}
	switch n.Kind() {
	case KindLiteralExpr:
		var lit string
		ok := false
		n.ForEachChildDesc(func(green *GreenNode, _ int) bool {
			if !green.IsToken() {
				return true
			}
			tok, tokOK := green.Token()
			if !tokOK {
				return true
			}
			switch tok.Type {
			case token.T_CONSTANT_ENCAPSED_STRING, token.T_CONSTANT_STRING:
				lit = tok.Literal
				if lit == "" {
					lit = greenTokenLiteral(green)
				}
				ok = true
				return false
			}
			return true
		})
		if !ok {
			return "", false
		}
		return decodeLowerStringLiteral(lit), true
	default:
		return "", false
	}
}

func firstExprOrNameChild(n *RedNode) *RedNode {
	if n == nil {
		return nil
	}
	var found *RedNode
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		if isExprKind(k) || isNameKind(k) {
			found = &RedNode{File: n.File, Green: green, Offset: offset}
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
			stmtList = &RedNode{File: n.File, Green: green, Offset: offset}
			return false
		}
		return true
	})
	if stmtList != nil {
		stmtList.ForEachChildDesc(func(green *GreenNode, offset int) bool {
			if green == nil || green.Kind() == KindToken {
				return true
			}
			body = append(body, &RedNode{File: stmtList.File, Green: green, Offset: offset})
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

// IsTypeKind reports whether k is a type-node kind (named, primitive,
// nullable, union, intersection, parenthesized, or callable-signature).
func IsTypeKind(k Kind) bool {
	return isTypeKind(k)
}

// PropertyDeclType returns the shared type child of a KindPropertyDecl, or nil.
func PropertyDeclType(n *RedNode) *RedNode {
	if n == nil || n.Kind() != KindPropertyDecl {
		return nil
	}
	var typ *RedNode
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if isTypeKind(green.Kind()) {
			typ = &RedNode{File: n.File, Green: green, Offset: offset}
			return false
		}
		return true
	})
	return typ
}

// PropertyDeclVariables returns each T_VARIABLE child of a KindPropertyDecl
// as a RedNode (source order). Multi-property decls share one type but each
// variable gets its own span for diagnostics.
func PropertyDeclVariables(n *RedNode) []*RedNode {
	if n == nil || n.Kind() != KindPropertyDecl {
		return nil
	}
	var vars []*RedNode
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if isGreenTokenType(green, token.T_VARIABLE) {
			vars = append(vars, &RedNode{File: n.File, Green: green, Offset: offset})
		}
		return true
	})
	return vars
}

// ParamType returns the type child of a KindParam, or nil.
func ParamType(n *RedNode) *RedNode {
	if n == nil || n.Kind() != KindParam {
		return nil
	}
	var typ *RedNode
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if isTypeKind(green.Kind()) {
			typ = &RedNode{File: n.File, Green: green, Offset: offset}
			return false
		}
		return true
	})
	return typ
}

// ParamVariable returns the T_VARIABLE child of a KindParam, or nil.
func ParamVariable(n *RedNode) *RedNode {
	if n == nil || n.Kind() != KindParam {
		return nil
	}
	var v *RedNode
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if isGreenTokenType(green, token.T_VARIABLE) {
			v = &RedNode{File: n.File, Green: green, Offset: offset}
			return false
		}
		return true
	})
	return v
}

// ParamIsPromoted reports whether a KindParam has a visibility modifier
// (public/protected/private), matching lowerParam's IsPromoted rule.
func ParamIsPromoted(n *RedNode) bool {
	if n == nil || n.Kind() != KindParam {
		return false
	}
	list := n.FirstChildOfKind(KindModifierList)
	if list == nil {
		return false
	}
	promoted := false
	list.ForEachChildDesc(func(green *GreenNode, _ int) bool {
		if green == nil || !green.IsToken() {
			return true
		}
		tt := green.TokenType()
		if tt == token.T_PUBLIC || tt == token.T_PROTECTED || tt == token.T_PRIVATE {
			promoted = true
			return false
		}
		return true
	})
	return promoted
}

// VariableName returns the identifier text of a T_VARIABLE token RedNode
// without the leading '$'. Empty string if n is not a variable token.
func VariableName(n *RedNode) string {
	if n == nil || n.Green == nil || !n.Green.IsToken() || n.Green.TokenType() != token.T_VARIABLE {
		return ""
	}
	return stripVarDollar(greenTokenLiteral(n.Green))
}

// RawSpanPositions returns classic positions for n's full green byte span
// (including leading trivia on the first token). Matches lowerProperties'
// multi-name property Pos/EndPos via spanStart/spanEnd — distinct from
// RedNode.Pos()/EndPos(), which skip leading trivia.
func RawSpanPositions(n *RedNode) (ast.Position, ast.Position) {
	if n == nil || n.File == nil {
		return ast.Position{}, ast.Position{}
	}
	sp := n.Span()
	return spanStart(n.File, sp), spanEnd(n.File, sp)
}
