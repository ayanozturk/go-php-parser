package syntax

import (
	"strconv"
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/token"
)

// lowerExpr lowers one expression CST node to a classic AST node.
// Unsupported kinds return nil without panicking.
func lowerExpr(n *RedNode, file *File) ast.Node {
	if n == nil {
		return nil
	}
	pos, end := nodePos(file, n)
	switch n.Kind() {
	case KindVariableExpr:
		return lowerVariableExpr(n, file)
	case KindLiteralExpr:
		return lowerLiteralExpr(n, file)
	case KindUnqualifiedName, KindQualifiedName, KindFullyQualifiedName, KindRelativeName, KindName:
		return &ast.IdentifierNode{Value: NameText(n), Pos: pos, EndPos: end}
	case KindBinaryExpr:
		return lowerBinaryExpr(n, file)
	case KindAssignExpr:
		return lowerAssignExpr(n, file)
	case KindCallExpr:
		return lowerCallExpr(n, file)
	case KindMemberAccessExpr, KindNullsafeMemberAccessExpr:
		return lowerMemberAccessExpr(n, file)
	case KindParenExpr:
		return lowerParenExpr(n, file)
	case KindUnaryExpr:
		return lowerUnaryExpr(n, file)
	case KindCastExpr:
		return lowerCastExpr(n, file)
	case KindStaticMemberAccessExpr:
		return lowerStaticMemberAccessExpr(n, file)
	case KindArrayAccessExpr:
		return lowerArrayAccessExpr(n, file)
	case KindNewExpr:
		return lowerNewExpr(n, file)
	case KindArrayExpr:
		return lowerArrayExpr(n, file)
	case KindTernaryExpr:
		return lowerTernaryExpr(n, file)
	case KindThrowExpr:
		return lowerThrowExpr(n, file)
	case KindClosureExpr:
		return lowerClosureExpr(n, file)
	case KindArrowFunctionExpr:
		return lowerArrowFunctionExpr(n, file)
	case KindMatchExpr:
		return lowerMatchExpr(n, file)
	case KindCloneExpr:
		return lowerCloneExpr(n, file)
	case KindListExpr:
		return lowerListExpr(n, file)
	case KindIncludeExpr, KindPrintExpr:
		return lowerKeywordUnaryExpr(n, file)
	case KindStringLiteral:
		return lowerInterpolatedStringLiteral(n, file)
	case KindHeredoc, KindNowdoc:
		return lowerHeredoc(n, file)
	default:
		return nil
	}
}

func lowerVariableExpr(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	name := ""
	for _, c := range n.Children() {
		if isTokenType(c, token.T_VARIABLE) {
			name = stripVarDollar(tokenLiteral(c))
			break
		}
	}
	if name == "" {
		name = stripVarDollar(strings.TrimSpace(n.Text()))
	}
	if name == "" {
		return nil
	}
	return &ast.VariableNode{Name: name, Pos: pos, EndPos: end}
}

func lowerLiteralExpr(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	var tokNode *RedNode
	for _, c := range n.Children() {
		if c.Green != nil && c.Green.IsToken() {
			tokNode = c
			break
		}
		if isNameKind(c.Kind()) {
			// bare name used as literal-ish (e.g. array keyword path)
			return &ast.IdentifierNode{Value: NameText(c), Pos: pos, EndPos: end}
		}
	}
	if tokNode == nil {
		return nil
	}
	tok, ok := tokNode.Green.Token()
	if !ok {
		return nil
	}
	lit := tok.Literal
	switch tok.Type {
	case token.T_LNUMBER:
		v, _ := strconv.ParseInt(strings.ReplaceAll(lit, "_", ""), 0, 64)
		return &ast.IntegerNode{Value: v, Pos: pos, EndPos: end}
	case token.T_DNUMBER:
		v, _ := strconv.ParseFloat(strings.ReplaceAll(lit, "_", ""), 64)
		return &ast.FloatNode{Value: v, Pos: pos, EndPos: end}
	case token.T_CONSTANT_ENCAPSED_STRING, token.T_CONSTANT_STRING:
		return &ast.StringLiteral{Value: decodeLowerStringLiteral(lit), Pos: pos, EndPos: end}
	case token.T_TRUE:
		return &ast.BooleanNode{Value: true, Pos: pos, EndPos: end}
	case token.T_FALSE:
		return &ast.BooleanNode{Value: false, Pos: pos, EndPos: end}
	case token.T_NULL:
		return &ast.NullNode{Pos: pos, EndPos: end}
	default:
		trimmed := strings.TrimSpace(strings.ToLower(lit))
		switch trimmed {
		case "true":
			return &ast.BooleanNode{Value: true, Pos: pos, EndPos: end}
		case "false":
			return &ast.BooleanNode{Value: false, Pos: pos, EndPos: end}
		case "null":
			return &ast.NullNode{Pos: pos, EndPos: end}
		}
		return &ast.IdentifierNode{Value: lit, Pos: pos, EndPos: end}
	}
}

func lowerBinaryExpr(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	children := n.Children()
	var left, right *RedNode
	var op string
	for _, c := range children {
		if c.Green != nil && c.Green.IsToken() {
			op = strings.TrimSpace(tokenLiteral(c))
			continue
		}
		if isExprKind(c.Kind()) || isNameKind(c.Kind()) {
			if left == nil {
				left = c
			} else {
				right = c
			}
		}
	}
	if left == nil || op == "" {
		return nil
	}
	return &ast.BinaryExpr{
		Left:     lowerExpr(left, file),
		Operator: op,
		Right:    lowerExpr(right, file),
		Pos:      pos,
		EndPos:   end,
	}
}

func lowerAssignExpr(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	children := n.Children()
	var left, right *RedNode
	var op string
	for _, c := range children {
		if c.Green != nil && c.Green.IsToken() {
			op = strings.TrimSpace(tokenLiteral(c))
			continue
		}
		if isExprKind(c.Kind()) || isNameKind(c.Kind()) {
			if left == nil {
				left = c
			} else {
				right = c
			}
		}
	}
	if left == nil || op == "" {
		return nil
	}
	leftNode := lowerExpr(left, file)
	rightNode := lowerExpr(right, file)
	if leftNode != nil {
		pos = leftNode.GetPos()
	}
	if rightNode != nil {
		end = rightNode.GetEndPos()
	}
	return &ast.AssignmentNode{
		Left:     leftNode,
		Operator: op,
		Right:    rightNode,
		Pos:      pos,
		EndPos:   end,
	}
}

func lowerCallExpr(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	var callee, args *RedNode
	for _, c := range n.Children() {
		switch {
		case c.Kind() == KindArgList:
			args = c
		case isExprKind(c.Kind()) || isNameKind(c.Kind()):
			if callee == nil {
				callee = c
			}
		}
	}
	if callee == nil {
		return nil
	}
	argNodes := lowerArgList(args, file)

	switch callee.Kind() {
	case KindMemberAccessExpr, KindNullsafeMemberAccessExpr:
		obj, method := splitMemberAccess(callee, file)
		if method == "" {
			return nil
		}
		return &ast.MethodCallNode{
			Object:   obj,
			Method:   method,
			Args:     argNodes,
			Pos:      pos,
			EndPos:   end,
			Nullsafe: callee.Kind() == KindNullsafeMemberAccessExpr,
		}
	case KindStaticMemberAccessExpr:
		// Classic emits FunctionCallNode{Name: IdentifierNode{"Class::method"}}.
		class, member := splitStaticMemberAccess(callee, file)
		if class == "" || member == "" {
			return nil
		}
		return &ast.FunctionCallNode{
			Name: &ast.IdentifierNode{
				Value:  class + "::" + member,
				Pos:    pos,
				EndPos: end,
			},
			Args:   argNodes,
			Pos:    pos,
			EndPos: end,
		}
	default:
		name := lowerExpr(callee, file)
		callPos, callEnd := pos, end
		if name != nil {
			callPos = name.GetPos()
		}
		return &ast.FunctionCallNode{
			Name:   name,
			Args:   argNodes,
			Pos:    callPos,
			EndPos: callEnd,
		}
	}
}

func lowerArgList(list *RedNode, file *File) []ast.Node {
	if list == nil {
		return nil
	}
	var out []ast.Node
	for _, c := range list.Children() {
		switch c.Kind() {
		case KindArg:
			if a := lowerArg(c, file); a != nil {
				out = append(out, a)
			}
		case KindNamedArg:
			if a := lowerNamedArg(c, file); a != nil {
				out = append(out, a)
			}
		}
	}
	return out
}

func lowerArg(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	unpacked := false
	var expr *RedNode
	for _, c := range n.Children() {
		if isTokenType(c, token.T_ELLIPSIS) {
			unpacked = true
			continue
		}
		if isExprKind(c.Kind()) || isNameKind(c.Kind()) {
			expr = c
		}
	}
	if expr == nil {
		return nil
	}
	lowered := lowerExpr(expr, file)
	if lowered == nil {
		return nil
	}
	if unpacked {
		return &ast.UnpackedArgumentNode{Expr: lowered, Pos: pos, EndPos: end}
	}
	return lowered
}

func lowerNamedArg(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	name := ""
	var expr *RedNode
	for _, c := range n.Children() {
		if c.Green != nil && c.Green.IsToken() {
			tt, ok := c.Green.Token()
			if !ok {
				continue
			}
			if tt.Type == token.T_STRING || tt.Type == token.T_CLASS {
				if name == "" {
					name = tt.Literal
				}
			}
			continue
		}
		if isExprKind(c.Kind()) || isNameKind(c.Kind()) {
			expr = c
		}
	}
	if name == "" || expr == nil {
		return nil
	}
	return &ast.NamedArgumentNode{
		Name:   name,
		Value:  lowerExpr(expr, file),
		Pos:    pos,
		EndPos: end,
	}
}

func lowerMemberAccessExpr(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	obj, prop := splitMemberAccess(n, file)
	// Empty property is valid for incomplete `$obj->` (completion / recovery).
	if obj == nil && prop == "" {
		return nil
	}
	return &ast.PropertyFetchNode{
		Object:   obj,
		Property: prop,
		Pos:      pos,
		EndPos:   end,
	}
}

func splitMemberAccess(n *RedNode, file *File) (object ast.Node, member string) {
	var objNode *RedNode
	for _, c := range n.Children() {
		if c.Kind() == KindToken {
			tt, ok := tokenOf(c)
			if !ok {
				continue
			}
			switch tt.Type {
			case token.T_OBJECT_OPERATOR, token.T_NULLSAFE_OBJECT_OPERATOR:
				continue
			case token.T_STRING, token.T_VARIABLE:
				member = strings.TrimSpace(tt.Literal)
				if tt.Type == token.T_VARIABLE {
					member = stripVarDollar(member)
				}
			default:
				if isKeywordMemberName(tt.Type) {
					member = strings.TrimSpace(tt.Literal)
				}
			}
			continue
		}
		if isExprKind(c.Kind()) || isNameKind(c.Kind()) {
			if objNode == nil {
				objNode = c
			}
		}
	}
	if objNode != nil {
		object = lowerExpr(objNode, file)
	}
	return object, member
}

func lowerParenExpr(n *RedNode, file *File) ast.Node {
	for _, c := range n.Children() {
		if isExprKind(c.Kind()) || isNameKind(c.Kind()) {
			return lowerExpr(c, file)
		}
	}
	return nil
}

func lowerUnaryExpr(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	var op string
	var operand *RedNode
	for _, c := range n.Children() {
		if c.Green != nil && c.Green.IsToken() {
			op = strings.TrimSpace(tokenLiteral(c))
			continue
		}
		if isExprKind(c.Kind()) || isNameKind(c.Kind()) {
			operand = c
		}
	}
	if op == "" || operand == nil {
		return nil
	}
	return &ast.UnaryExpr{
		Operator: op,
		Operand:  lowerExpr(operand, file),
		Pos:      pos,
		EndPos:   end,
	}
}

func lowerCastExpr(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	castType := ""
	var operand *RedNode
	seenOpen := false
	for _, c := range n.Children() {
		if c.Green != nil && c.Green.IsToken() {
			tt, ok := tokenOf(c)
			if !ok {
				continue
			}
			switch tt.Type {
			case token.T_LPAREN:
				seenOpen = true
			case token.T_RPAREN:
				seenOpen = false
			default:
				if seenOpen && castType == "" {
					castType = strings.TrimSpace(tt.Literal)
				}
			}
			continue
		}
		if isExprKind(c.Kind()) || isNameKind(c.Kind()) {
			operand = c
		}
	}
	if castType == "" || operand == nil {
		return nil
	}
	return &ast.TypeCastNode{
		Type:   castType,
		Expr:   lowerExpr(operand, file),
		Pos:    pos,
		EndPos: end,
	}
}

func lowerStaticMemberAccessExpr(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	class, constName := splitStaticMemberAccess(n, file)
	if class == "" || constName == "" {
		return nil
	}
	return &ast.ClassConstFetchNode{
		Class:  class,
		Const:  constName,
		Pos:    pos,
		EndPos: end,
	}
}

func splitStaticMemberAccess(n *RedNode, file *File) (class, member string) {
	if n == nil {
		return "", ""
	}
	for _, c := range n.Children() {
		if isNameKind(c.Kind()) && class == "" {
			class = NameText(c)
			continue
		}
		if isExprKind(c.Kind()) && class == "" {
			// Dynamic class expr — classic ClassConstFetch only stores string;
			// use TokenLiteral of lowered node when possible.
			if e := lowerExpr(c, file); e != nil {
				class = e.TokenLiteral()
			}
			continue
		}
		if c.Green != nil && c.Green.IsToken() {
			tt, ok := tokenOf(c)
			if !ok || tt.Type == token.T_DOUBLE_COLON {
				continue
			}
			member = strings.TrimSpace(tt.Literal)
			if tt.Type == token.T_VARIABLE {
				member = stripVarDollar(member)
			}
		}
	}
	return class, member
}

func lowerArrayAccessExpr(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	var arr, idx *RedNode
	for _, c := range n.Children() {
		if isExprKind(c.Kind()) || isNameKind(c.Kind()) {
			if arr == nil {
				arr = c
			} else {
				idx = c
			}
		}
	}
	if arr == nil {
		return nil
	}
	var index ast.Node
	if idx != nil {
		index = lowerExpr(idx, file)
	}
	return &ast.ArrayAccessNode{
		Var:    lowerExpr(arr, file),
		Index:  index,
		Pos:    pos,
		EndPos: end,
	}
}

func lowerNewExpr(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	var classTarget, args *RedNode
	for _, c := range n.Children() {
		switch c.Kind() {
		case KindArgList:
			args = c
		case KindAttributeList:
			continue
		case KindAnonymousClass:
			cls, ctorArgs := lowerAnonymousClass(c, file)
			if cls == nil {
				return nil
			}
			return &ast.NewNode{
				ClassExpr: cls,
				Args:      ctorArgs,
				Pos:       pos,
				EndPos:    end,
			}
		default:
			if (isExprKind(c.Kind()) || isNameKind(c.Kind())) && classTarget == nil {
				classTarget = c
			}
		}
	}
	if classTarget == nil {
		return nil
	}
	argNodes := lowerArgList(args, file)
	switch classTarget.Kind() {
	case KindUnqualifiedName, KindQualifiedName, KindFullyQualifiedName, KindRelativeName, KindName:
		return &ast.NewNode{
			ClassName: NameText(classTarget),
			Args:      argNodes,
			Pos:       pos,
			EndPos:    end,
		}
	case KindVariableExpr:
		v, ok := lowerExpr(classTarget, file).(*ast.VariableNode)
		if !ok || v == nil {
			return &ast.NewNode{
				ClassExpr: lowerExpr(classTarget, file),
				Args:      argNodes,
				Pos:       pos,
				EndPos:    end,
			}
		}
		// Classic stores bare `new $var` as ClassName="$name".
		return &ast.NewNode{
			ClassName: "$" + v.Name,
			Args:      argNodes,
			Pos:       pos,
			EndPos:    end,
		}
	default:
		return &ast.NewNode{
			ClassExpr: lowerExpr(classTarget, file),
			Args:      argNodes,
			Pos:       pos,
			EndPos:    end,
		}
	}
}

func lowerArrayExpr(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	var elements []ast.Node
	for _, c := range n.Children() {
		if c.Kind() != KindArrayElement {
			continue
		}
		if item := lowerArrayElement(c, file); item != nil {
			elements = append(elements, item)
		}
	}
	return &ast.ArrayNode{Elements: elements, Pos: pos, EndPos: end}
}

func lowerArrayElement(n *RedNode, file *File) ast.Node {
	if n == nil {
		return nil
	}
	pos, end := nodePos(file, n)
	children := n.Children()
	if len(children) == 0 {
		return nil
	}
	i := 0
	if isTokenType(children[0], token.T_ELLIPSIS) {
		var val *RedNode
		for _, c := range children[1:] {
			if isExprKind(c.Kind()) || isNameKind(c.Kind()) {
				val = c
				break
			}
		}
		if val == nil {
			return nil
		}
		return &ast.ArrayItemNode{
			Value:  lowerExpr(val, file),
			Unpack: true,
			Pos:    pos,
			EndPos: end,
		}
	}

	byRefBeforeFirst := false
	if isTokenType(children[i], token.T_AMPERSAND) {
		byRefBeforeFirst = true
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
	}
	if first == nil {
		return nil
	}

	// Look for => after first expr.
	hasArrow := false
	for j := i; j < len(children); j++ {
		if isTokenType(children[j], token.T_DOUBLE_ARROW) {
			hasArrow = true
			i = j + 1
			break
		}
	}
	if !hasArrow {
		return &ast.ArrayItemNode{
			Value:  lowerExpr(first, file),
			ByRef:  byRefBeforeFirst,
			Pos:    pos,
			EndPos: end,
		}
	}

	key := lowerExpr(first, file)
	if byRefBeforeFirst && key != nil {
		kp, ke := nodePos(file, first)
		key = &ast.UnaryExpr{Operator: "&", Operand: key, Pos: kp, EndPos: ke}
	}
	byRefValue := false
	if i < len(children) && isTokenType(children[i], token.T_AMPERSAND) {
		byRefValue = true
		i++
	}
	var val *RedNode
	for ; i < len(children); i++ {
		c := children[i]
		if isExprKind(c.Kind()) || isNameKind(c.Kind()) {
			val = c
			break
		}
	}
	var value ast.Node
	if val != nil {
		value = lowerExpr(val, file)
	}
	return &ast.ArrayItemNode{
		Key:    key,
		Value:  value,
		ByRef:  byRefValue,
		Pos:    pos,
		EndPos: end,
	}
}

func lowerTernaryExpr(n *RedNode, file *File) ast.Node {
	pos, end := nodePos(file, n)
	children := n.Children()
	var cond *RedNode
	seenQ := false
	seenColon := false
	var thenChild, elseChild *RedNode
	for _, c := range children {
		if c.Green != nil && c.Green.IsToken() {
			tt, ok := tokenOf(c)
			if !ok {
				continue
			}
			switch tt.Type {
			case token.T_QUESTION:
				seenQ = true
			case token.T_COLON:
				seenColon = true
			}
			continue
		}
		if !(isExprKind(c.Kind()) || isNameKind(c.Kind())) {
			continue
		}
		if cond == nil {
			cond = c
			continue
		}
		if seenQ && !seenColon && thenChild == nil {
			thenChild = c
			continue
		}
		if seenColon && elseChild == nil {
			elseChild = c
		}
	}
	if cond == nil {
		return nil
	}
	condNode := lowerExpr(cond, file)
	var ifTrue ast.Node
	if thenChild == nil {
		// Elvis / short ternary: classic sets IfTrue = Condition.
		ifTrue = condNode
	} else {
		ifTrue = lowerExpr(thenChild, file)
	}
	var ifFalse ast.Node
	if elseChild != nil {
		ifFalse = lowerExpr(elseChild, file)
	}
	return &ast.TernaryExpr{
		Condition: condNode,
		IfTrue:    ifTrue,
		IfFalse:   ifFalse,
		Pos:       pos,
		EndPos:    end,
	}
}

func lowerThrowExpr(n *RedNode, file *File) ast.Node {
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

func tokenOf(n *RedNode) (token.Token, bool) {
	if n == nil || n.Green == nil || !n.Green.IsToken() {
		return token.Token{}, false
	}
	return n.Green.Token()
}

func decodeLowerStringLiteral(lit string) string {
	if len(lit) >= 2 {
		q := lit[0]
		if (q == '"' || q == '\'') && lit[len(lit)-1] == q {
			return lit[1 : len(lit)-1]
		}
	}
	return lit
}
