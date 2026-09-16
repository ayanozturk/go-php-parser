package syntax

import (
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/token"
)

// lowerStatements lowers a KindStatementList (or similar list) to classic body nodes.
// Unsupported children are skipped without panicking.
func lowerStatements(list *RedNode, file *File) []ast.Node {
	if list == nil {
		return nil
	}
	var out []ast.Node
	for _, c := range list.Children() {
		if c == nil || c.Kind() == KindToken {
			continue
		}
		if stmt := lowerStmt(c, file); stmt != nil {
			out = append(out, stmt)
		}
	}
	return out
}

// lowerStmt lowers one statement CST node to a classic AST node.
func lowerStmt(n *RedNode, file *File) ast.Node {
	if n == nil {
		return nil
	}
	pos, end := nodePos(file, n)
	switch n.Kind() {
	case KindExpressionStmt:
		expr := firstExprChild(n)
		if expr == nil {
			return nil
		}
		lowered := lowerExpr(expr, file)
		if lowered == nil {
			return nil
		}
		return &ast.ExpressionStmt{Expr: lowered, Pos: pos, EndPos: end}
	case KindReturnStmt:
		ret := &ast.ReturnNode{Pos: pos, EndPos: end}
		if expr := firstExprChild(n); expr != nil {
			ret.Expr = lowerExpr(expr, file)
		}
		return ret
	case KindIfStmt:
		return lowerIfStmt(n, file)
	case KindEmptyStmt:
		return &ast.EmptyStatementNode{Pos: pos, EndPos: end}
	case KindEchoStmt:
		return lowerEchoStmt(n, file)
	case KindThrowStmt:
		return lowerThrowStmt(n, file)
	case KindWhileStmt:
		return lowerWhileStmt(n, file)
	case KindDoWhileStmt:
		return lowerDoWhileStmt(n, file)
	case KindForStmt:
		return lowerForStmt(n, file)
	case KindForeachStmt:
		return lowerForeachStmt(n, file)
	case KindBreakStmt:
		return lowerBreakStmt(n, file)
	case KindContinueStmt:
		return lowerContinueStmt(n, file)
	case KindTryStmt:
		return lowerTryStmt(n, file)
	case KindSwitchStmt:
		return lowerSwitchStmt(n, file)
	default:
		return nil
	}
}

func lowerIfStmt(n *RedNode, file *File) *ast.IfNode {
	if n == nil {
		return nil
	}
	pos, end := nodePos(file, n)
	iff := &ast.IfNode{Pos: pos, EndPos: end}
	seenCond := false
	for _, c := range n.Children() {
		switch c.Kind() {
		case KindElseIfClause:
			if ei := lowerElseIfClause(c, file); ei != nil {
				iff.ElseIfs = append(iff.ElseIfs, ei)
			}
		case KindElseClause:
			iff.Else = lowerElseClause(c, file)
		case KindToken:
			continue
		default:
			if isExprKind(c.Kind()) && !seenCond {
				iff.Condition = lowerExpr(c, file)
				seenCond = true
				continue
			}
			if c.Kind() == KindStatementList || isStmtKind(c.Kind()) {
				if iff.Body == nil && seenCond {
					iff.Body = lowerControlBody(c, file)
				}
			}
		}
	}
	return iff
}

func lowerElseIfClause(n *RedNode, file *File) *ast.ElseIfNode {
	if n == nil {
		return nil
	}
	pos, end := nodePos(file, n)
	ei := &ast.ElseIfNode{Pos: pos, EndPos: end}
	seenCond := false
	for _, c := range n.Children() {
		if c.Kind() == KindToken {
			continue
		}
		if isExprKind(c.Kind()) && !seenCond {
			ei.Condition = lowerExpr(c, file)
			seenCond = true
			continue
		}
		if c.Kind() == KindStatementList || isStmtKind(c.Kind()) {
			ei.Body = lowerControlBody(c, file)
			break
		}
	}
	return ei
}

func lowerElseClause(n *RedNode, file *File) *ast.ElseNode {
	if n == nil {
		return nil
	}
	pos, end := nodePos(file, n)
	els := &ast.ElseNode{Pos: pos, EndPos: end}
	for _, c := range n.Children() {
		if c.Kind() == KindToken {
			continue
		}
		if c.Kind() == KindStatementList || isStmtKind(c.Kind()) {
			els.Body = lowerControlBody(c, file)
			break
		}
	}
	return els
}

// lowerControlBody accepts a KindStatementList or a single statement node.
func lowerControlBody(n *RedNode, file *File) []ast.Node {
	if n == nil {
		return nil
	}
	if n.Kind() == KindStatementList {
		return lowerStatements(n, file)
	}
	if stmt := lowerStmt(n, file); stmt != nil {
		return []ast.Node{stmt}
	}
	return nil
}

func firstExprChild(n *RedNode) *RedNode {
	if n == nil {
		return nil
	}
	for _, c := range n.Children() {
		if isExprKind(c.Kind()) {
			return c
		}
	}
	return nil
}

func lowerEchoStmt(n *RedNode, file *File) ast.Node {
	if n == nil {
		return nil
	}
	pos, end := nodePos(file, n)
	var exprs []ast.Node
	for _, c := range n.Children() {
		if isExprKind(c.Kind()) || isNameKind(c.Kind()) {
			if e := lowerExpr(c, file); e != nil {
				exprs = append(exprs, e)
			}
		}
	}
	if len(exprs) == 0 {
		return nil
	}
	if len(exprs) == 1 {
		return &ast.ExpressionStmt{Expr: exprs[0], Pos: pos, EndPos: end}
	}
	stmts := make([]ast.Node, 0, len(exprs))
	for _, e := range exprs {
		ep, ee := e.GetPos(), e.GetEndPos()
		stmts = append(stmts, &ast.ExpressionStmt{Expr: e, Pos: ep, EndPos: ee})
	}
	return &ast.BlockNode{Statements: stmts, Pos: pos, EndPos: end}
}

func lowerThrowStmt(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	expr := firstExprChild(n)
	if expr == nil {
		return nil
	}
	return &ast.ThrowNode{
		Expr:   lowerExpr(expr, file),
		Pos:    pos,
		EndPos: end,
	}
}

func lowerWhileStmt(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	w := &ast.WhileNode{Pos: pos, EndPos: end}
	seenCond := false
	for _, c := range n.Children() {
		if c.Kind() == KindToken {
			continue
		}
		if (isExprKind(c.Kind()) || isNameKind(c.Kind())) && !seenCond {
			w.Condition = lowerExpr(c, file)
			seenCond = true
			continue
		}
		if c.Kind() == KindStatementList || isStmtKind(c.Kind()) {
			w.Body = lowerControlBody(c, file)
			break
		}
	}
	return w
}

func lowerDoWhileStmt(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	d := &ast.DoWhileNode{Pos: pos, EndPos: end}
	for _, c := range n.Children() {
		if c.Kind() == KindToken {
			continue
		}
		if d.Body == nil && (c.Kind() == KindStatementList || isStmtKind(c.Kind())) {
			d.Body = lowerControlBody(c, file)
			continue
		}
		if isExprKind(c.Kind()) || isNameKind(c.Kind()) {
			d.Condition = lowerExpr(c, file)
		}
	}
	return d
}

func lowerForStmt(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	f := &ast.ForNode{Pos: pos, EndPos: end}
	children := n.Children()
	i := 0
	// Skip to '(' after for.
	for ; i < len(children); i++ {
		if isTokenType(children[i], token.T_LPAREN) {
			i++
			break
		}
	}
	f.Init, i = collectForClause(children, i, file)
	f.Conditions, i = collectForClause(children, i, file)
	f.Updates, i = collectForClause(children, i, file)
	for ; i < len(children); i++ {
		c := children[i]
		if c.Kind() == KindToken {
			continue
		}
		if c.Kind() == KindStatementList || isStmtKind(c.Kind()) {
			f.Body = lowerControlBody(c, file)
			break
		}
	}
	return f
}

func collectForClause(children []*RedNode, start int, file *File) ([]ast.Node, int) {
	var exprs []ast.Node
	for i := start; i < len(children); i++ {
		c := children[i]
		if isTokenType(c, token.T_SEMICOLON) || isTokenType(c, token.T_RPAREN) {
			return exprs, i + 1
		}
		if isTokenType(c, token.T_COMMA) {
			continue
		}
		if isExprKind(c.Kind()) || isNameKind(c.Kind()) {
			if e := lowerExpr(c, file); e != nil {
				exprs = append(exprs, e)
			}
		}
	}
	return exprs, len(children)
}

func lowerForeachStmt(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	fe := &ast.ForeachNode{Pos: pos, EndPos: end}
	children := n.Children()
	i := 0
	for ; i < len(children); i++ {
		if isTokenType(children[i], token.T_LPAREN) {
			i++
			break
		}
	}
	// Iterated expression.
	for ; i < len(children); i++ {
		c := children[i]
		if isTokenType(c, token.T_AS) {
			i++
			break
		}
		if isExprKind(c.Kind()) || isNameKind(c.Kind()) {
			fe.Expr = lowerExpr(c, file)
		}
	}
	refBeforeFirst := false
	if i < len(children) && isTokenType(children[i], token.T_AMPERSAND) {
		refBeforeFirst = true
		i++
	}
	var first *RedNode
	for ; i < len(children); i++ {
		c := children[i]
		if isExprKind(c.Kind()) || isNameKind(c.Kind()) {
			first = c
			i++
			break
		}
		if isTokenType(c, token.T_RPAREN) {
			break
		}
	}
	hasArrow := false
	for j := i; j < len(children); j++ {
		if isTokenType(children[j], token.T_DOUBLE_ARROW) {
			hasArrow = true
			i = j + 1
			break
		}
		if isTokenType(children[j], token.T_RPAREN) {
			break
		}
	}
	if first != nil {
		if hasArrow {
			key := lowerExpr(first, file)
			if refBeforeFirst && key != nil {
				kp, ke := nodePos(file, first)
				key = &ast.UnaryExpr{Operator: "&", Operand: key, Pos: kp, EndPos: ke}
			}
			fe.KeyVar = key
			if i < len(children) && isTokenType(children[i], token.T_AMPERSAND) {
				fe.ByRef = true
				i++
			}
			for ; i < len(children); i++ {
				c := children[i]
				if isExprKind(c.Kind()) || isNameKind(c.Kind()) {
					fe.ValueVar = lowerExpr(c, file)
					i++
					break
				}
				if isTokenType(c, token.T_RPAREN) {
					break
				}
			}
		} else {
			fe.ValueVar = lowerExpr(first, file)
			fe.ByRef = refBeforeFirst
		}
	}
	for ; i < len(children); i++ {
		c := children[i]
		if c.Kind() == KindToken {
			continue
		}
		if c.Kind() == KindStatementList || isStmtKind(c.Kind()) {
			fe.Body = lowerControlBody(c, file)
			break
		}
	}
	return fe
}

func lowerBreakStmt(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	var level ast.Node
	if expr := firstExprChild(n); expr != nil {
		level = lowerExpr(expr, file)
	}
	return &ast.BreakNode{Level: level, Pos: pos, EndPos: end}
}

func lowerContinueStmt(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	var level ast.Node
	if expr := firstExprChild(n); expr != nil {
		level = lowerExpr(expr, file)
	}
	return &ast.ContinueNode{Level: level, Pos: pos, EndPos: end}
}

func lowerTryStmt(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	tr := &ast.TryNode{Pos: pos, EndPos: end}
	for _, c := range n.Children() {
		switch c.Kind() {
		case KindStatementList:
			if tr.Body == nil {
				tr.Body = lowerStatements(c, file)
			}
		case KindCatchClause:
			if catch := lowerCatchClause(c, file); catch != nil {
				tr.Catches = append(tr.Catches, catch)
			}
		case KindFinallyClause:
			tr.Finally = lowerFinallyClause(c, file)
		}
	}
	return tr
}

func lowerCatchClause(n *RedNode, file *File) *ast.CatchNode {
	if n == nil {
		return nil
	}
	pos, end := nodePos(file, n)
	catch := &ast.CatchNode{Pos: pos, EndPos: end}
	for _, c := range n.Children() {
		switch {
		case isTypeKind(c.Kind()):
			catch.Types = catchTypeNames(c, file)
		case isTokenType(c, token.T_VARIABLE):
			catch.Variable = stripVarDollar(tokenLiteral(c))
		case c.Kind() == KindStatementList:
			catch.Body = lowerStatements(c, file)
		case c.Kind() == KindTokenList:
			// Recovery fallback: concatenate token text as a single type.
			if text := strings.TrimSpace(c.Text()); text != "" && len(catch.Types) == 0 {
				catch.Types = []string{text}
			}
		}
	}
	return catch
}

func catchTypeNames(n *RedNode, file *File) []string {
	t := lowerType(n, file)
	if t == nil {
		return nil
	}
	if u, ok := t.(*ast.UnionTypeNode); ok {
		out := make([]string, 0, len(u.Types))
		for _, part := range u.Types {
			if s := ast.TypeText(part); s != "" {
				out = append(out, s)
			}
		}
		return out
	}
	if s := ast.TypeText(t); s != "" {
		return []string{s}
	}
	return nil
}

func lowerFinallyClause(n *RedNode, file *File) []ast.Node {
	if n == nil {
		return nil
	}
	for _, c := range n.Children() {
		if c.Kind() == KindStatementList {
			return lowerStatements(c, file)
		}
	}
	return nil
}

func lowerSwitchStmt(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	sw := &ast.SwitchNode{Pos: pos, EndPos: end}
	seenExpr := false
	for _, c := range n.Children() {
		if c.Kind() == KindToken {
			continue
		}
		if (isExprKind(c.Kind()) || isNameKind(c.Kind())) && !seenExpr {
			sw.Expr = lowerExpr(c, file)
			seenExpr = true
			continue
		}
		if c.Kind() == KindStatementList {
			sw.Cases = lowerSwitchCases(c, file)
			break
		}
	}
	return sw
}

func lowerSwitchCases(block *RedNode, file *File) []*ast.SwitchCaseNode {
	if block == nil {
		return nil
	}
	var cases []*ast.SwitchCaseNode
	for _, c := range block.Children() {
		switch c.Kind() {
		case KindCaseClause:
			if sc := lowerCaseClause(c, file, false); sc != nil {
				cases = append(cases, sc)
			}
		case KindDefaultClause:
			if sc := lowerCaseClause(c, file, true); sc != nil {
				cases = append(cases, sc)
			}
		}
	}
	return cases
}

func lowerCaseClause(n *RedNode, file *File, isDefault bool) *ast.SwitchCaseNode {
	if n == nil {
		return nil
	}
	pos, end := nodePos(file, n)
	sc := &ast.SwitchCaseNode{IsDefault: isDefault, Pos: pos, EndPos: end}
	for _, c := range n.Children() {
		if c.Kind() == KindToken {
			continue
		}
		if !isDefault && sc.Expr == nil && (isExprKind(c.Kind()) || isNameKind(c.Kind())) {
			sc.Expr = lowerExpr(c, file)
			continue
		}
		if c.Kind() == KindStatementList {
			sc.Body = lowerStatements(c, file)
		}
	}
	return sc
}

func isStmtKind(k Kind) bool {
	switch k {
	case KindExpressionStmt, KindReturnStmt, KindIfStmt, KindEmptyStmt,
		KindEchoStmt, KindBreakStmt, KindContinueStmt, KindThrowStmt,
		KindUnsetStmt, KindWhileStmt, KindDoWhileStmt, KindForStmt,
		KindForeachStmt, KindSwitchStmt, KindTryStmt, KindGlobalStmt,
		KindStaticVarStmt, KindDeclareStmt:
		return true
	default:
		return false
	}
}

func isExprKind(k Kind) bool {
	switch k {
	case KindVariableExpr, KindLiteralExpr, KindBinaryExpr, KindUnaryExpr,
		KindAssignExpr, KindTernaryExpr, KindCallExpr, KindMemberAccessExpr,
		KindNullsafeMemberAccessExpr, KindArrayAccessExpr, KindStaticMemberAccessExpr,
		KindNewExpr, KindCloneExpr, KindCastExpr, KindParenExpr,
		KindClosureExpr, KindArrowFunctionExpr, KindMatchExpr, KindYieldExpr,
		KindIncludeExpr, KindPrintExpr, KindThrowExpr, KindListExpr,
		KindArrayExpr, KindHeredoc, KindNowdoc, KindStringLiteral,
		KindVariableVariableExpr, KindFirstClassCallableExpr,
		KindUnqualifiedName, KindQualifiedName, KindFullyQualifiedName,
		KindRelativeName, KindName:
		return true
	default:
		return false
	}
}
