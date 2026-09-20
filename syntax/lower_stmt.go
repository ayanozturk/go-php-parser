package syntax

import (
	"strconv"
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
	list.ForEachChild(func(c *RedNode) bool {
		if c == nil || c.Kind() == KindToken {
			return true
		}
		if stmt := lowerStmt(c, file); stmt != nil {
			out = append(out, stmt)
		}
		return true
	})
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
		// Reuse outer nodePos; override from lowered expr when classic coords are set.
		if p := lowered.GetPos(); p.Line > 0 {
			pos = p
		}
		if e := lowered.GetEndPos(); e.Line > 0 {
			end = e
		}
		return &ast.ExpressionStmt{
			Expr:   lowered,
			PHPDoc: leadingDocFromNode(n),
			Pos:    pos,
			EndPos: end,
		}
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
	case KindGlobalStmt:
		return lowerGlobalStmt(n, file)
	case KindStaticVarStmt:
		return lowerStaticVarStmt(n, file)
	case KindUnsetStmt:
		return lowerUnsetStmt(n, file)
	case KindDeclareStmt:
		return lowerDeclareStmt(n, file)
	case KindGotoStmt:
		label := ""
		n.ForEachChild(func(c *RedNode) bool {
			if isTokenType(c, token.T_STRING) {
				label = strings.TrimSpace(tokenLiteral(c))
				return false
			}
			return true
		})
		if label == "" {
			return nil
		}
		return &ast.GotoNode{Label: label, Pos: pos, EndPos: end}
	case KindLabelStmt:
		name := ""
		n.ForEachChild(func(c *RedNode) bool {
			if isTokenType(c, token.T_STRING) {
				name = strings.TrimSpace(tokenLiteral(c))
				return false
			}
			return true
		})
		if name == "" {
			return nil
		}
		return &ast.LabelNode{Name: name, Pos: pos, EndPos: end}
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
	n.ForEachChild(func(c *RedNode) bool {
		switch c.Kind() {
		case KindElseIfClause:
			if ei := lowerElseIfClause(c, file); ei != nil {
				iff.ElseIfs = append(iff.ElseIfs, ei)
			}
		case KindElseClause:
			iff.Else = lowerElseClause(c, file)
		case KindToken:
			return true
		default:
			if isExprKind(c.Kind()) && !seenCond {
				iff.Condition = lowerExpr(c, file)
				seenCond = true
				return true
			}
			if c.Kind() == KindStatementList || isStmtKind(c.Kind()) {
				if iff.Body == nil && seenCond {
					iff.Body = lowerControlBody(c, file)
				}
			}
		}
		return true
	})
	return iff
}

func lowerElseIfClause(n *RedNode, file *File) *ast.ElseIfNode {
	if n == nil {
		return nil
	}
	pos, end := nodePos(file, n)
	ei := &ast.ElseIfNode{Pos: pos, EndPos: end}
	seenCond := false
	n.ForEachChild(func(c *RedNode) bool {
		if c.Kind() == KindToken {
			return true
		}
		if isExprKind(c.Kind()) && !seenCond {
			ei.Condition = lowerExpr(c, file)
			seenCond = true
			return true
		}
		if c.Kind() == KindStatementList || isStmtKind(c.Kind()) {
			ei.Body = lowerControlBody(c, file)
			return false
		}
		return true
	})
	return ei
}

func lowerElseClause(n *RedNode, file *File) *ast.ElseNode {
	if n == nil {
		return nil
	}
	pos, end := nodePos(file, n)
	els := &ast.ElseNode{Pos: pos, EndPos: end}
	n.ForEachChild(func(c *RedNode) bool {
		if c.Kind() == KindToken {
			return true
		}
		if c.Kind() == KindStatementList || isStmtKind(c.Kind()) {
			els.Body = lowerControlBody(c, file)
			return false
		}
		return true
	})
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
	var found *RedNode
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if isExprKind(green.Kind()) {
			found = &RedNode{File: n.File, Parent: n, Green: green, Offset: offset}
			return false
		}
		return true
	})
	return found
}

func lowerEchoStmt(n *RedNode, file *File) ast.Node {
	if n == nil {
		return nil
	}
	pos, end := nodePos(file, n)
	var exprs []ast.Node
	n.ForEachChild(func(c *RedNode) bool {
		if isExprKind(c.Kind()) || isNameKind(c.Kind()) {
			if e := lowerExpr(c, file); e != nil {
				exprs = append(exprs, e)
			}
		}
		return true
	})
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
	n.ForEachChild(func(c *RedNode) bool {
		if c.Kind() == KindToken {
			return true
		}
		if (isExprKind(c.Kind()) || isNameKind(c.Kind())) && !seenCond {
			w.Condition = lowerExpr(c, file)
			seenCond = true
			return true
		}
		if c.Kind() == KindStatementList || isStmtKind(c.Kind()) {
			w.Body = lowerControlBody(c, file)
			return false
		}
		return true
	})
	return w
}

func lowerDoWhileStmt(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	d := &ast.DoWhileNode{Pos: pos, EndPos: end}
	n.ForEachChild(func(c *RedNode) bool {
		if c.Kind() == KindToken {
			return true
		}
		if d.Body == nil && (c.Kind() == KindStatementList || isStmtKind(c.Kind())) {
			d.Body = lowerControlBody(c, file)
			return true
		}
		if isExprKind(c.Kind()) || isNameKind(c.Kind()) {
			d.Condition = lowerExpr(c, file)
		}
		return true
	})
	return d
}

// walkRedChildrenFrom visits children at indices >= start. fn returns the next
// index to resume from, or -1 to continue with the following child.
func walkRedChildrenFrom(children []*RedNode, start int, fn func(c *RedNode, i int) int) int {
	for i := start; i < len(children); i++ {
		if n := fn(children[i], i); n >= 0 {
			return n
		}
	}
	return len(children)
}

func lowerForStmt(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	f := &ast.ForNode{Pos: pos, EndPos: end}
	children := n.Children()
	// Skip to '(' after for.
	i := walkRedChildrenFrom(children, 0, func(c *RedNode, idx int) int {
		if isTokenType(c, token.T_LPAREN) {
			return idx + 1
		}
		return -1
	})
	f.Init, i = collectForClause(children, i, file)
	f.Conditions, i = collectForClause(children, i, file)
	f.Updates, i = collectForClause(children, i, file)
	walkRedChildrenFrom(children, i, func(c *RedNode, idx int) int {
		if c.Kind() == KindToken {
			return -1
		}
		if c.Kind() == KindStatementList || isStmtKind(c.Kind()) {
			f.Body = lowerControlBody(c, file)
			return idx
		}
		return -1
	})
	return f
}

func collectForClause(children []*RedNode, start int, file *File) ([]ast.Node, int) {
	var exprs []ast.Node
	next := walkRedChildrenFrom(children, start, func(c *RedNode, idx int) int {
		if isTokenType(c, token.T_SEMICOLON) || isTokenType(c, token.T_RPAREN) {
			return idx + 1
		}
		if isTokenType(c, token.T_COMMA) {
			return -1
		}
		if isExprKind(c.Kind()) || isNameKind(c.Kind()) {
			if e := lowerExpr(c, file); e != nil {
				exprs = append(exprs, e)
			}
		}
		return -1
	})
	return exprs, next
}

func lowerForeachStmt(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	fe := &ast.ForeachNode{Pos: pos, EndPos: end}
	children := n.Children()
	i := walkRedChildrenFrom(children, 0, func(c *RedNode, idx int) int {
		if isTokenType(c, token.T_LPAREN) {
			return idx + 1
		}
		return -1
	})
	// Iterated expression.
	i = walkRedChildrenFrom(children, i, func(c *RedNode, idx int) int {
		if isTokenType(c, token.T_AS) {
			return idx + 1
		}
		if isExprKind(c.Kind()) || isNameKind(c.Kind()) {
			fe.Expr = lowerExpr(c, file)
		}
		return -1
	})
	refBeforeFirst := false
	i = walkRedChildrenFrom(children, i, func(c *RedNode, idx int) int {
		if isTokenType(c, token.T_AMPERSAND) {
			refBeforeFirst = true
			return idx + 1
		}
		return idx
	})
	var first *RedNode
	i = walkRedChildrenFrom(children, i, func(c *RedNode, idx int) int {
		if isExprKind(c.Kind()) || isNameKind(c.Kind()) {
			first = c
			return idx + 1
		}
		if isTokenType(c, token.T_RPAREN) {
			return idx
		}
		return -1
	})
	hasArrow := false
	i = walkRedChildrenFrom(children, i, func(c *RedNode, idx int) int {
		if isTokenType(c, token.T_DOUBLE_ARROW) {
			hasArrow = true
			return idx + 1
		}
		if isTokenType(c, token.T_RPAREN) {
			return idx
		}
		return -1
	})
	if first != nil {
		if hasArrow {
			key := lowerExpr(first, file)
			if refBeforeFirst && key != nil {
				kp, ke := nodePos(file, first)
				key = &ast.UnaryExpr{Operator: "&", Operand: key, Pos: kp, EndPos: ke}
			}
			fe.KeyVar = key
			i = walkRedChildrenFrom(children, i, func(c *RedNode, idx int) int {
				if isTokenType(c, token.T_AMPERSAND) {
					fe.ByRef = true
					return idx + 1
				}
				return idx
			})
			i = walkRedChildrenFrom(children, i, func(c *RedNode, idx int) int {
				if isExprKind(c.Kind()) || isNameKind(c.Kind()) {
					fe.ValueVar = lowerExpr(c, file)
					return idx + 1
				}
				if isTokenType(c, token.T_RPAREN) {
					return idx
				}
				return -1
			})
		} else {
			fe.ValueVar = lowerExpr(first, file)
			fe.ByRef = refBeforeFirst
		}
	}
	walkRedChildrenFrom(children, i, func(c *RedNode, idx int) int {
		if c.Kind() == KindToken {
			return -1
		}
		if c.Kind() == KindStatementList || isStmtKind(c.Kind()) {
			fe.Body = lowerControlBody(c, file)
			return idx
		}
		return -1
	})
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
	n.ForEachChild(func(c *RedNode) bool {
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
		return true
	})
	return tr
}

func lowerCatchClause(n *RedNode, file *File) *ast.CatchNode {
	if n == nil {
		return nil
	}
	pos, end := nodePos(file, n)
	catch := &ast.CatchNode{Pos: pos, EndPos: end}
	n.ForEachChild(func(c *RedNode) bool {
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
		return true
	})
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
	var body []ast.Node
	n.ForEachChild(func(c *RedNode) bool {
		if c.Kind() == KindStatementList {
			body = lowerStatements(c, file)
			return false
		}
		return true
	})
	return body
}

func lowerSwitchStmt(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	sw := &ast.SwitchNode{Pos: pos, EndPos: end}
	seenExpr := false
	n.ForEachChild(func(c *RedNode) bool {
		if c.Kind() == KindToken {
			return true
		}
		if (isExprKind(c.Kind()) || isNameKind(c.Kind())) && !seenExpr {
			sw.Expr = lowerExpr(c, file)
			seenExpr = true
			return true
		}
		if c.Kind() == KindStatementList {
			sw.Cases = lowerSwitchCases(c, file)
			return false
		}
		return true
	})
	return sw
}

func lowerUnsetStmt(n *RedNode, file *File) ast.Node {
	if n == nil {
		return nil
	}
	pos, end := nodePos(file, n)
	var args []ast.Node
	n.ForEachChild(func(c *RedNode) bool {
		if c.Kind() == KindArgList {
			args = lowerArgList(c, file)
			return false
		}
		return true
	})
	if len(args) == 0 {
		n.ForEachChild(func(c *RedNode) bool {
			if isExprKind(c.Kind()) || isNameKind(c.Kind()) {
				if e := lowerExpr(c, file); e != nil {
					args = append(args, e)
				}
			}
			return true
		})
	}
	if len(args) == 0 {
		return nil
	}
	name := &ast.IdentifierNode{Value: "unset", Pos: pos, EndPos: end}
	call := &ast.FunctionCallNode{Name: name, Args: args, Pos: pos, EndPos: end}
	return &ast.ExpressionStmt{Expr: call, Pos: pos, EndPos: end}
}

func lowerDeclareStmt(n *RedNode, file *File) ast.Node {
	if n == nil {
		return nil
	}
	pos, end := nodePos(file, n)
	decl := &ast.DeclareNode{Directives: map[string]ast.Node{}, Pos: pos, EndPos: end}
	n.ForEachChild(func(c *RedNode) bool {
		switch c.Kind() {
		case KindTokenList:
			if len(decl.Directives) == 0 {
				decl.Directives = lowerDeclareDirectives(c, file)
			}
		case KindStatementList:
			stmts := lowerStatements(c, file)
			if len(stmts) > 0 {
				bp, be := nodePos(file, c)
				decl.Body = &ast.BlockNode{Statements: stmts, Pos: bp, EndPos: be}
			}
		}
		return true
	})
	if len(decl.Directives) == 0 {
		return nil
	}
	return decl
}

func lowerDeclareDirectives(list *RedNode, file *File) map[string]ast.Node {
	if list == nil {
		return nil
	}
	var tokens []*RedNode
	list.ForEachChild(func(c *RedNode) bool {
		if c.Kind() == KindToken {
			tokens = append(tokens, c)
		}
		return true
	})
	out := make(map[string]ast.Node)
	for i := 0; i < len(tokens); i++ {
		if !isTokenType(tokens[i], token.T_STRING) {
			continue
		}
		name := strings.TrimSpace(tokenLiteral(tokens[i]))
		if name == "" {
			continue
		}
		i++
		for i < len(tokens) && isTokenType(tokens[i], token.T_COMMA) {
			i++
		}
		if i >= len(tokens) || !isTokenType(tokens[i], token.T_ASSIGN) {
			continue
		}
		i++
		if i >= len(tokens) {
			break
		}
		if val := lowerDeclareDirectiveValue(tokens[i], file); val != nil {
			out[name] = val
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func lowerDeclareDirectiveValue(n *RedNode, file *File) ast.Node {
	if n == nil {
		return nil
	}
	pos, end := nodePos(file, n)
	lit := tokenLiteral(n)
	switch {
	case isTokenType(n, token.T_LNUMBER):
		v, _ := strconv.ParseInt(strings.ReplaceAll(lit, "_", ""), 0, 64)
		return &ast.IntegerNode{Value: v, Pos: pos, EndPos: end}
	case isTokenType(n, token.T_DNUMBER):
		v, _ := strconv.ParseFloat(strings.ReplaceAll(lit, "_", ""), 64)
		return &ast.FloatNode{Value: v, Pos: pos, EndPos: end}
	case isTokenType(n, token.T_CONSTANT_ENCAPSED_STRING):
		return &ast.StringLiteral{Value: decodeLowerStringLiteral(lit), Pos: pos, EndPos: end}
	case isTokenType(n, token.T_STRING):
		return &ast.IdentifierNode{Value: lit, Pos: pos, EndPos: end}
	default:
		return nil
	}
}

func lowerGlobalStmt(n *RedNode, file *File) ast.Node {
	if n == nil {
		return nil
	}
	pos, end := nodePos(file, n)
	var vars []ast.GlobalVarEntry
	n.ForEachChild(func(c *RedNode) bool {
		if !isTokenType(c, token.T_VARIABLE) {
			return true
		}
		vp, ve := nodePos(file, c)
		vars = append(vars, ast.GlobalVarEntry{
			Name:   stripVarDollar(tokenLiteral(c)),
			Pos:    vp,
			EndPos: ve,
		})
		return true
	})
	if len(vars) == 0 {
		return nil
	}
	return &ast.GlobalVarDeclNode{Vars: vars, Pos: pos, EndPos: end}
}

func lowerStaticVarStmt(n *RedNode, file *File) ast.Node {
	if n == nil {
		return nil
	}
	pos, end := nodePos(file, n)
	var vars []ast.StaticVarEntry
	var curName string
	var curPos, curEnd ast.Position
	var curInit ast.Node
	flush := func() {
		if curName == "" {
			return
		}
		vars = append(vars, ast.StaticVarEntry{
			Name:   curName,
			Init:   curInit,
			Pos:    curPos,
			EndPos: curEnd,
		})
		curName = ""
		curInit = nil
	}
	n.ForEachChild(func(c *RedNode) bool {
		switch {
		case isTokenType(c, token.T_VARIABLE):
			flush()
			curPos, curEnd = nodePos(file, c)
			curName = stripVarDollar(tokenLiteral(c))
		case isTokenType(c, token.T_COMMA), isTokenType(c, token.T_SEMICOLON):
			flush()
		case isTokenType(c, token.T_ASSIGN):
			return true
		default:
			if curName != "" && (isExprKind(c.Kind()) || isNameKind(c.Kind())) {
				curInit = lowerExpr(c, file)
			}
		}
		return true
	})
	flush()
	if len(vars) == 0 {
		return nil
	}
	return &ast.StaticVarDeclNode{Vars: vars, Pos: pos, EndPos: end}
}

func lowerSwitchCases(block *RedNode, file *File) []*ast.SwitchCaseNode {
	if block == nil {
		return nil
	}
	var cases []*ast.SwitchCaseNode
	block.ForEachChild(func(c *RedNode) bool {
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
		return true
	})
	return cases
}

func lowerCaseClause(n *RedNode, file *File, isDefault bool) *ast.SwitchCaseNode {
	if n == nil {
		return nil
	}
	pos, end := nodePos(file, n)
	sc := &ast.SwitchCaseNode{IsDefault: isDefault, Pos: pos, EndPos: end}
	n.ForEachChild(func(c *RedNode) bool {
		if c.Kind() == KindToken {
			return true
		}
		if !isDefault && sc.Expr == nil && (isExprKind(c.Kind()) || isNameKind(c.Kind())) {
			sc.Expr = lowerExpr(c, file)
			return true
		}
		if c.Kind() == KindStatementList {
			sc.Body = lowerStatements(c, file)
		}
		return true
	})
	return sc
}

func isStmtKind(k Kind) bool {
	switch k {
	case KindExpressionStmt, KindReturnStmt, KindIfStmt, KindEmptyStmt,
		KindEchoStmt, KindBreakStmt, KindContinueStmt, KindThrowStmt,
		KindUnsetStmt, KindGotoStmt, KindLabelStmt, KindWhileStmt, KindDoWhileStmt, KindForStmt,
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
