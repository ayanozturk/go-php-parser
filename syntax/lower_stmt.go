package syntax

import (
	"strconv"
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/token"
)

// lowerStmtAt lowers one statement child from green+offset without bindChild.
func lowerStmtAt(file *File, green *GreenNode, offset int) ast.Node {
	if file == nil || green == nil || green.Kind() == KindToken {
		return nil
	}
	n := RedNode{File: file, Green: green, Offset: offset}
	return lowerStmt(&n, file)
}

// lowerControlBodyAt lowers a statement-list or single statement child without bindChild.
func lowerControlBodyAt(file *File, green *GreenNode, offset int) []ast.Node {
	if file == nil || green == nil || green.Kind() == KindToken {
		return nil
	}
	k := green.Kind()
	if k == KindStatementList {
		n := RedNode{File: file, Green: green, Offset: offset}
		return lowerStatements(&n, file)
	}
	if isStmtKind(k) {
		if stmt := lowerStmtAt(file, green, offset); stmt != nil {
			return []ast.Node{stmt}
		}
	}
	return nil
}

// lowerStatements lowers a KindStatementList (or similar list) to classic body nodes.
// Unsupported children are skipped without panicking.
func lowerStatements(list *RedNode, file *File) []ast.Node {
	if list == nil {
		return nil
	}
	var out []ast.Node
	list.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if green.Kind() == KindToken {
			return true
		}
		if stmt := lowerStmtAt(file, green, offset); stmt != nil {
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
		n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
			if !isGreenTokenType(green, token.T_STRING) {
				return true
			}
			label = strings.TrimSpace(greenTokenLiteral(green))
			return false
		})
		if label == "" {
			return nil
		}
		return &ast.GotoNode{Label: label, Pos: pos, EndPos: end}
	case KindLabelStmt:
		name := ""
		n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
			if !isGreenTokenType(green, token.T_STRING) {
				return true
			}
			name = strings.TrimSpace(greenTokenLiteral(green))
			return false
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
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		if k == KindToken {
			return true
		}
		switch k {
		case KindElseIfClause:
			clause := RedNode{File: n.File, Green: green, Offset: offset}
			if ei := lowerElseIfClause(&clause, file); ei != nil {
				iff.ElseIfs = append(iff.ElseIfs, ei)
			}
		case KindElseClause:
			clause := RedNode{File: n.File, Green: green, Offset: offset}
			iff.Else = lowerElseClause(&clause, file)
		default:
			if isExprKind(k) && !seenCond {
				iff.Condition = lowerExprAt(file, green, offset)
				seenCond = true
				return true
			}
			if k == KindStatementList || isStmtKind(k) {
				if iff.Body == nil && seenCond {
					iff.Body = lowerControlBodyAt(file, green, offset)
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
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		if k == KindToken {
			return true
		}
		if isExprKind(k) && !seenCond {
			ei.Condition = lowerExprAt(file, green, offset)
			seenCond = true
			return true
		}
		if k == KindStatementList || isStmtKind(k) {
			ei.Body = lowerControlBodyAt(file, green, offset)
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
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		if k == KindToken {
			return true
		}
		if k == KindStatementList || isStmtKind(k) {
			els.Body = lowerControlBodyAt(file, green, offset)
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
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		if !isExprKind(k) && !isNameKind(k) {
			return true
		}
		if e := lowerExprAt(file, green, offset); e != nil {
			exprs = append(exprs, e)
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
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		if k == KindToken {
			return true
		}
		if (isExprKind(k) || isNameKind(k)) && !seenCond {
			w.Condition = lowerExprAt(file, green, offset)
			seenCond = true
			return true
		}
		if k == KindStatementList || isStmtKind(k) {
			w.Body = lowerControlBodyAt(file, green, offset)
			return false
		}
		return true
	})
	return w
}

func lowerDoWhileStmt(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	d := &ast.DoWhileNode{Pos: pos, EndPos: end}
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		if k == KindToken {
			return true
		}
		if d.Body == nil && (k == KindStatementList || isStmtKind(k)) {
			d.Body = lowerControlBodyAt(file, green, offset)
			return true
		}
		if isExprKind(k) || isNameKind(k) {
			d.Condition = lowerExprAt(file, green, offset)
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

// walkRedNodeChildrenFrom is like walkRedChildrenFrom but walks n's direct
// children via ForEachChildDesc without bindChild per child.
func walkRedNodeChildrenFrom(n *RedNode, start int, fn func(green *GreenNode, offset int, i int) int) int {
	if n == nil {
		return start
	}
	i := 0
	next := -1
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if i < start {
			i++
			return true
		}
		if ni := fn(green, offset, i); ni >= 0 {
			next = ni
			return false
		}
		i++
		return true
	})
	if next >= 0 {
		return next
	}
	return i
}

func lowerExprRedNodes(nodes []*RedNode, file *File) []ast.Node {
	if len(nodes) == 0 {
		return nil
	}
	out := make([]ast.Node, 0, len(nodes))
	for _, c := range nodes {
		if e := lowerExpr(c, file); e != nil {
			out = append(out, e)
		}
	}
	return out
}

func lowerForStmt(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	f := &ast.ForNode{Pos: pos, EndPos: end}
	start := forStmtIndexAfterLParen(n)
	init, start := forClauseChildrenAt(n, start)
	f.Init = lowerExprRedNodes(init, file)
	conds, start := forClauseChildrenAt(n, start)
	f.Conditions = lowerExprRedNodes(conds, file)
	updates, start := forClauseChildrenAt(n, start)
	f.Updates = lowerExprRedNodes(updates, file)
	walkRedNodeChildrenFrom(n, start, func(green *GreenNode, offset int, idx int) int {
		k := green.Kind()
		if k == KindToken {
			return -1
		}
		if k == KindStatementList || isStmtKind(k) {
			f.Body = lowerControlBodyAt(file, green, offset)
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
	i := forStmtIndexAfterLParen(n)
	// Iterated expression.
	i = walkRedNodeChildrenFrom(n, i, func(green *GreenNode, offset int, idx int) int {
		if isGreenTokenType(green, token.T_AS) {
			return idx + 1
		}
		k := green.Kind()
		if isExprKind(k) || isNameKind(k) {
			fe.Expr = lowerExprAt(file, green, offset)
		}
		return -1
	})
	refBeforeFirst := false
	i = walkRedNodeChildrenFrom(n, i, func(green *GreenNode, offset int, idx int) int {
		if isGreenTokenType(green, token.T_AMPERSAND) {
			refBeforeFirst = true
			return idx + 1
		}
		return idx
	})
	var firstGreen *GreenNode
	var firstOff int
	i = walkRedNodeChildrenFrom(n, i, func(green *GreenNode, offset int, idx int) int {
		k := green.Kind()
		if isExprKind(k) || isNameKind(k) {
			firstGreen, firstOff = green, offset
			return idx + 1
		}
		if isGreenTokenType(green, token.T_RPAREN) {
			return idx
		}
		return -1
	})
	hasArrow := false
	i = walkRedNodeChildrenFrom(n, i, func(green *GreenNode, offset int, idx int) int {
		if isGreenTokenType(green, token.T_DOUBLE_ARROW) {
			hasArrow = true
			return idx + 1
		}
		if isGreenTokenType(green, token.T_RPAREN) {
			return idx
		}
		return -1
	})
	if firstGreen != nil {
		if hasArrow {
			key := lowerExprAt(file, firstGreen, firstOff)
			if refBeforeFirst && key != nil {
				kp, ke := nodePosGreen(file, firstGreen, firstOff)
				key = &ast.UnaryExpr{Operator: "&", Operand: key, Pos: kp, EndPos: ke}
			}
			fe.KeyVar = key
			i = walkRedNodeChildrenFrom(n, i, func(green *GreenNode, offset int, idx int) int {
				if isGreenTokenType(green, token.T_AMPERSAND) {
					fe.ByRef = true
					return idx + 1
				}
				return idx
			})
			i = walkRedNodeChildrenFrom(n, i, func(green *GreenNode, offset int, idx int) int {
				k := green.Kind()
				if isExprKind(k) || isNameKind(k) {
					fe.ValueVar = lowerExprAt(file, green, offset)
					return idx + 1
				}
				if isGreenTokenType(green, token.T_RPAREN) {
					return idx
				}
				return -1
			})
		} else {
			fe.ValueVar = lowerExprAt(file, firstGreen, firstOff)
			fe.ByRef = refBeforeFirst
		}
	}
	walkRedNodeChildrenFrom(n, i, func(green *GreenNode, offset int, idx int) int {
		k := green.Kind()
		if k == KindToken {
			return -1
		}
		if k == KindStatementList || isStmtKind(k) {
			fe.Body = lowerControlBodyAt(file, green, offset)
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
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		if k == KindToken {
			return true
		}
		child := RedNode{File: n.File, Green: green, Offset: offset}
		switch k {
		case KindStatementList:
			if tr.Body == nil {
				tr.Body = lowerStatements(&child, file)
			}
		case KindCatchClause:
			if catch := lowerCatchClause(&child, file); catch != nil {
				tr.Catches = append(tr.Catches, catch)
			}
		case KindFinallyClause:
			tr.Finally = lowerFinallyClause(&child, file)
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
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		switch {
		case isTypeKind(k):
			typeNode := RedNode{File: n.File, Green: green, Offset: offset}
			catch.Types = catchTypeNames(&typeNode, file)
		case k == KindToken:
			if isGreenTokenType(green, token.T_VARIABLE) {
				catch.Variable = stripVarDollar(greenTokenLiteral(green))
			}
		case k == KindStatementList:
			body := RedNode{File: n.File, Green: green, Offset: offset}
			catch.Body = lowerStatements(&body, file)
		case k == KindTokenList:
			tokenList := RedNode{File: n.File, Green: green, Offset: offset}
			// Recovery fallback: concatenate token text as a single type.
			if text := strings.TrimSpace(tokenList.Text()); text != "" && len(catch.Types) == 0 {
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
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if green.Kind() != KindStatementList {
			return true
		}
		list := RedNode{File: n.File, Green: green, Offset: offset}
		body = lowerStatements(&list, file)
		return false
	})
	return body
}

func lowerSwitchStmt(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	sw := &ast.SwitchNode{Pos: pos, EndPos: end}
	seenExpr := false
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		if k == KindToken {
			return true
		}
		if (isExprKind(k) || isNameKind(k)) && !seenExpr {
			sw.Expr = lowerExprAt(file, green, offset)
			seenExpr = true
			return true
		}
		if k == KindStatementList {
			block := RedNode{File: n.File, Green: green, Offset: offset}
			sw.Cases = lowerSwitchCases(&block, file)
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
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if green.Kind() == KindArgList {
			argList := RedNode{File: n.File, Green: green, Offset: offset}
			args = lowerArgList(&argList, file)
			return false
		}
		return true
	})
	if len(args) == 0 {
		n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
			k := green.Kind()
			if isExprKind(k) || isNameKind(k) {
				if e := lowerExprAt(file, green, offset); e != nil {
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
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		switch green.Kind() {
		case KindTokenList:
			if len(decl.Directives) == 0 {
				tl := RedNode{File: n.File, Green: green, Offset: offset}
				decl.Directives = lowerDeclareDirectives(&tl, file)
			}
		case KindStatementList:
			bodyList := RedNode{File: n.File, Green: green, Offset: offset}
			stmts := lowerStatements(&bodyList, file)
			if len(stmts) > 0 {
				bp, be := nodePosGreen(file, green, offset)
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
	var tokens []struct {
		green  *GreenNode
		offset int
	}
	list.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if green.Kind() != KindToken {
			return true
		}
		tokens = append(tokens, struct {
			green  *GreenNode
			offset int
		}{green, offset})
		return true
	})
	out := make(map[string]ast.Node)
	for i := 0; i < len(tokens); i++ {
		if !isGreenTokenType(tokens[i].green, token.T_STRING) {
			continue
		}
		name := strings.TrimSpace(greenTokenLiteral(tokens[i].green))
		if name == "" {
			continue
		}
		i++
		for i < len(tokens) && isGreenTokenType(tokens[i].green, token.T_COMMA) {
			i++
		}
		if i >= len(tokens) || !isGreenTokenType(tokens[i].green, token.T_ASSIGN) {
			continue
		}
		i++
		if i >= len(tokens) {
			break
		}
		valNode := RedNode{File: file, Green: tokens[i].green, Offset: tokens[i].offset}
		if val := lowerDeclareDirectiveValue(&valNode, file); val != nil {
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
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if green.Kind() != KindToken || green.TokenType() != token.T_VARIABLE {
			return true
		}
		vp, ve := nodePosGreen(file, green, offset)
		vars = append(vars, ast.GlobalVarEntry{
			Name:   stripVarDollar(greenTokenLiteral(green)),
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
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		if k == KindToken {
			switch green.TokenType() {
			case token.T_VARIABLE:
				flush()
				curPos, curEnd = nodePosGreen(file, green, offset)
				curName = stripVarDollar(greenTokenLiteral(green))
			case token.T_COMMA, token.T_SEMICOLON:
				flush()
			case token.T_ASSIGN:
				return true
			}
			return true
		}
		if curName != "" && (isExprKind(k) || isNameKind(k)) {
			curInit = lowerExprAt(file, green, offset)
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
	block.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		switch green.Kind() {
		case KindCaseClause:
			clause := RedNode{File: block.File, Green: green, Offset: offset}
			if sc := lowerCaseClause(&clause, file, false); sc != nil {
				cases = append(cases, sc)
			}
		case KindDefaultClause:
			clause := RedNode{File: block.File, Green: green, Offset: offset}
			if sc := lowerCaseClause(&clause, file, true); sc != nil {
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
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		if k == KindToken {
			return true
		}
		if !isDefault && sc.Expr == nil && (isExprKind(k) || isNameKind(k)) {
			sc.Expr = lowerExprAt(file, green, offset)
			return true
		}
		if k == KindStatementList {
			body := RedNode{File: n.File, Green: green, Offset: offset}
			sc.Body = lowerStatements(&body, file)
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
