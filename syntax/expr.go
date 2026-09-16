package syntax

import "github.com/ayanozturk/go-php-parser/token"

// Expression precedence (higher binds tighter). Mirrors parser/operator.go.
const (
	precLogicalOr  = 0
	precLogicalXor = 1
	precLogicalAnd = 2
	precAssign     = 3
	precTernary    = 4
	precOr         = 5
	precAnd        = 6
	precBitOr      = 7
	precBitXor     = 8
	precBitAnd     = 9
	precEquality   = 10
	precCompare    = 11
	precInstanceof = 12
	precCoalesce   = 13
	precShift      = 13
	precAdd        = 14
	precMul        = 15
	precPow        = 16
	precUnary      = 17
)

func exprPrec(tt token.TokenType) (int, bool) {
	switch tt {
	case token.T_LOGICAL_OR:
		return precLogicalOr, true
	case token.T_LOGICAL_XOR:
		return precLogicalXor, true
	case token.T_LOGICAL_AND:
		return precLogicalAnd, true
	case token.T_ASSIGN, token.T_PLUS_EQUAL, token.T_MINUS_EQUAL, token.T_MUL_EQUAL,
		token.T_DIV_EQUAL, token.T_MOD_EQUAL, token.T_AND_EQUAL, token.T_OR_EQUAL,
		token.T_CONCAT_EQUAL, token.T_XOR_EQUAL, token.T_COALESCE_EQUAL, token.T_POW_EQUAL,
		token.T_SL_EQUAL, token.T_SR_EQUAL:
		return precAssign, true
	case token.T_QUESTION:
		return precTernary, true
	case token.T_BOOLEAN_OR:
		return precOr, true
	case token.T_BOOLEAN_AND:
		return precAnd, true
	case token.T_PIPE:
		return precBitOr, true
	case token.T_CARET:
		return precBitXor, true
	case token.T_AMPERSAND:
		return precBitAnd, true
	case token.T_IS_EQUAL, token.T_IS_NOT_EQUAL, token.T_IS_IDENTICAL, token.T_IS_NOT_IDENTICAL:
		return precEquality, true
	case token.T_IS_SMALLER, token.T_IS_GREATER, token.T_IS_GREATER_OR_EQUAL,
		token.T_IS_SMALLER_OR_EQUAL, token.T_SPACESHIP:
		return precCompare, true
	case token.T_INSTANCEOF:
		return precInstanceof, true
	case token.T_COALESCE, token.T_SL, token.T_SR:
		return precCoalesce, true
	case token.T_PLUS, token.T_MINUS, token.T_DOT:
		return precAdd, true
	case token.T_MULTIPLY, token.T_DIVIDE, token.T_MODULO:
		return precMul, true
	case token.T_POW:
		return precPow, true
	default:
		return 0, false
	}
}

func exprRightAssoc(tt token.TokenType) bool {
	switch tt {
	case token.T_ASSIGN, token.T_PLUS_EQUAL, token.T_MINUS_EQUAL, token.T_MUL_EQUAL,
		token.T_DIV_EQUAL, token.T_MOD_EQUAL, token.T_AND_EQUAL, token.T_OR_EQUAL,
		token.T_CONCAT_EQUAL, token.T_XOR_EQUAL, token.T_COALESCE_EQUAL, token.T_POW_EQUAL,
		token.T_SL_EQUAL, token.T_SR_EQUAL, token.T_COALESCE, token.T_POW, token.T_QUESTION:
		return true
	default:
		return false
	}
}

func isAssignOp(tt token.TokenType) bool {
	switch tt {
	case token.T_ASSIGN, token.T_PLUS_EQUAL, token.T_MINUS_EQUAL, token.T_MUL_EQUAL,
		token.T_DIV_EQUAL, token.T_MOD_EQUAL, token.T_AND_EQUAL, token.T_OR_EQUAL,
		token.T_CONCAT_EQUAL, token.T_XOR_EQUAL, token.T_COALESCE_EQUAL, token.T_POW_EQUAL,
		token.T_SL_EQUAL, token.T_SR_EQUAL:
		return true
	default:
		return false
	}
}

func (p *Parser) peekType(n int) token.TokenType {
	i := p.i + n
	if i >= len(p.tokens) {
		return token.T_EOF
	}
	return p.tokens[i].Type
}

func (p *Parser) peekLit(n int) string {
	i := p.i + n
	if i >= len(p.tokens) {
		return ""
	}
	return p.tokens[i].Literal
}

// parseExpression parses one expression green node (structured).
func (p *Parser) parseExpression() *GreenNode {
	return p.parseExprBp(0)
}

func (p *Parser) parseExprBp(minPrec int) *GreenNode {
	left := p.parsePrefixExpr()
	if left == nil {
		return nil
	}
	return p.parseInfixExpr(left, minPrec)
}

func (p *Parser) parsePrefixExpr() *GreenNode {
	switch p.tok().Type {
	case token.T_PLUS, token.T_MINUS, token.T_NOT, token.T_TILDE, token.T_AT,
		token.T_INC, token.T_DEC, token.T_AMPERSAND:
		op := p.bump()
		operand := p.parseExprBp(precUnary)
		if operand == nil {
			return p.intern.Node(KindUnaryExpr, op)
		}
		return p.intern.Node(KindUnaryExpr, op, operand)
	case token.T_CLONE:
		kw := p.bump()
		operand := p.parseExprBp(precUnary)
		if operand == nil {
			return p.intern.Node(KindCloneExpr, kw)
		}
		return p.intern.Node(KindCloneExpr, kw, operand)
	case token.T_THROW:
		kw := p.bump()
		operand := p.parseExprBp(precUnary)
		if operand == nil {
			return p.intern.Node(KindThrowExpr, kw)
		}
		return p.intern.Node(KindThrowExpr, kw, operand)
	case token.T_PRINT:
		kw := p.bump()
		// print binds just above logical and/xor/or (same as assignment band).
		operand := p.parseExprBp(precAssign)
		if operand == nil {
			return p.intern.Node(KindPrintExpr, kw)
		}
		return p.intern.Node(KindPrintExpr, kw, operand)
	case token.T_INCLUDE, token.T_INCLUDE_ONCE, token.T_REQUIRE, token.T_REQUIRE_ONCE:
		kw := p.bump()
		operand := p.parseExprBp(precAssign)
		if operand == nil {
			return p.intern.Node(KindIncludeExpr, kw)
		}
		return p.intern.Node(KindIncludeExpr, kw, operand)
	case token.T_YIELD, token.T_YIELD_FROM:
		return p.parseYieldExpr()
	case token.T_LPAREN:
		if p.isCastStart() {
			return p.parseCastExpr()
		}
	}
	left := p.parsePrimaryExpr()
	if left == nil {
		return nil
	}
	return p.parsePostfixExpr(left)
}

func (p *Parser) parseYieldExpr() *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.bump())
	if p.at(token.T_SEMICOLON) || p.at(token.T_COMMA) || p.at(token.T_RPAREN) ||
		p.at(token.T_RBRACKET) || p.at(token.T_RBRACE) || p.at(token.T_COLON) ||
		p.at(token.T_DOUBLE_ARROW) || p.at(token.T_EOF) {
		return p.intern.Node(KindYieldExpr, parts...)
	}
	// Optional key => value, or bare value.
	first := p.parseExprBp(precAssign)
	if first != nil {
		parts = append(parts, first)
		if p.at(token.T_DOUBLE_ARROW) {
			parts = append(parts, p.bump())
			if val := p.parseExprBp(precAssign); val != nil {
				parts = append(parts, val)
			}
		}
	}
	return p.intern.Node(KindYieldExpr, parts...)
}

func (p *Parser) isCastStart() bool {
	if !p.at(token.T_LPAREN) {
		return false
	}
	t1 := p.peekType(1)
	t2 := p.peekType(2)
	if t2 != token.T_RPAREN {
		return false
	}
	switch t1 {
	case token.T_ARRAY, token.T_CALLABLE, token.T_UNSET:
		return true
	case token.T_STRING:
		switch p.peekLit(1) {
		case "int", "integer", "bool", "boolean", "float", "double", "real",
			"string", "binary", "unset", "object", "void":
			return true
		}
	}
	return false
}

func (p *Parser) parseCastExpr() *GreenNode {
	open := p.bump()
	typ := p.bump()
	close := p.expect(token.T_RPAREN)
	operand := p.parseExprBp(precUnary)
	if operand == nil {
		return p.intern.Node(KindCastExpr, open, typ, close)
	}
	return p.intern.Node(KindCastExpr, open, typ, close, operand)
}

func (p *Parser) parseInfixExpr(left *GreenNode, minPrec int) *GreenNode {
	for {
		if p.at(token.T_QUESTION) {
			prec := precTernary
			if minPrec > prec {
				break
			}
			left = p.parseTernaryExpr(left)
			continue
		}
		prec, ok := exprPrec(p.tok().Type)
		if !ok || prec < minPrec {
			break
		}
		opTok := p.tok().Type
		op := p.bump()
		nextMin := prec + 1
		if exprRightAssoc(opTok) {
			nextMin = prec
		}
		var right *GreenNode
		if opTok == token.T_INSTANCEOF {
			right = p.parseInstanceofRHS()
		} else {
			right = p.parseExprBp(nextMin)
		}
		kind := KindBinaryExpr
		if isAssignOp(opTok) {
			kind = KindAssignExpr
		}
		if right == nil {
			left = p.intern.Node(kind, left, op)
		} else {
			left = p.intern.Node(kind, left, op, right)
		}
	}
	return left
}

func (p *Parser) parseTernaryExpr(cond *GreenNode) *GreenNode {
	q := p.bump() // ?
	var parts []*GreenNode
	parts = append(parts, cond, q)
	// Elvis?: cond ?: else
	if p.at(token.T_COLON) {
		parts = append(parts, p.bump())
		els := p.parseExprBp(precTernary)
		if els != nil {
			parts = append(parts, els)
		}
		return p.intern.Node(KindTernaryExpr, parts...)
	}
	then := p.parseExprBp(0)
	if then != nil {
		parts = append(parts, then)
	}
	if p.at(token.T_COLON) {
		parts = append(parts, p.bump())
		els := p.parseExprBp(precTernary)
		if els != nil {
			parts = append(parts, els)
		}
	}
	return p.intern.Node(KindTernaryExpr, parts...)
}

func (p *Parser) parseInstanceofRHS() *GreenNode {
	if p.isNameStart() {
		return p.parseName()
	}
	return p.parseExprBp(precInstanceof + 1)
}

func (p *Parser) parsePostfixExpr(expr *GreenNode) *GreenNode {
	for {
		switch p.tok().Type {
		case token.T_INC, token.T_DEC:
			expr = p.intern.Node(KindUnaryExpr, expr, p.bump())
		case token.T_OBJECT_OPERATOR:
			expr = p.parseMemberAccess(expr, KindMemberAccessExpr)
		case token.T_NULLSAFE_OBJECT_OPERATOR:
			expr = p.parseMemberAccess(expr, KindNullsafeMemberAccessExpr)
		case token.T_DOUBLE_COLON:
			expr = p.parseStaticMemberAccess(expr)
		case token.T_LBRACKET:
			expr = p.parseArrayAccess(expr)
		case token.T_LBRACE:
			// Rare: $a{0} deprecated string offset — keep tokens structured as ArrayAccess-like.
			expr = p.parseBraceAccess(expr)
		case token.T_LPAREN:
			expr = p.parseCallOrFirstClass(expr)
		default:
			return expr
		}
	}
}

func (p *Parser) parseMemberAccess(expr *GreenNode, kind Kind) *GreenNode {
	op := p.bump()
	member := p.parseMemberName()
	if member == nil {
		return p.intern.Node(kind, expr, op)
	}
	return p.intern.Node(kind, expr, op, member)
}

func (p *Parser) parseMemberName() *GreenNode {
	switch p.tok().Type {
	case token.T_STRING, token.T_VARIABLE:
		return p.bump()
	case token.T_LBRACE:
		open := p.bump()
		inner := p.parseExpression()
		close := p.expect(token.T_RBRACE)
		if inner == nil {
			return p.intern.Node(KindParenExpr, open, close)
		}
		return p.intern.Node(KindParenExpr, open, inner, close)
	case token.T_CURLY_OPEN, token.T_DOLLAR_OPEN_CURLY_BRACES:
		return p.parseEncapsulatedExpr()
	default:
		if p.atIdentName() {
			return p.bump()
		}
		return nil
	}
}

func (p *Parser) parseStaticMemberAccess(expr *GreenNode) *GreenNode {
	op := p.bump() // ::
	var parts []*GreenNode
	parts = append(parts, expr, op)
	switch {
	case p.at(token.T_VARIABLE), p.at(token.T_STRING), p.at(token.T_CLASS), p.at(token.T_CLASS_CONST):
		parts = append(parts, p.bump())
	case p.at(token.T_LBRACE):
		if m := p.parseMemberName(); m != nil {
			parts = append(parts, m)
		}
	case p.at(token.T_DOLLAR_OPEN_CURLY_BRACES), p.at(token.T_CURLY_OPEN):
		parts = append(parts, p.parseEncapsulatedExpr())
	default:
		if p.atIdentName() {
			parts = append(parts, p.bump())
		}
	}
	return p.intern.Node(KindStaticMemberAccessExpr, parts...)
}

func (p *Parser) parseArrayAccess(expr *GreenNode) *GreenNode {
	open := p.bump()
	var parts []*GreenNode
	parts = append(parts, expr, open)
	if !p.at(token.T_RBRACKET) {
		if idx := p.parseExpression(); idx != nil {
			parts = append(parts, idx)
		}
	}
	parts = append(parts, p.expect(token.T_RBRACKET))
	return p.intern.Node(KindArrayAccessExpr, parts...)
}

func (p *Parser) parseBraceAccess(expr *GreenNode) *GreenNode {
	open := p.bump()
	var parts []*GreenNode
	parts = append(parts, expr, open)
	if !p.at(token.T_RBRACE) {
		if idx := p.parseExpression(); idx != nil {
			parts = append(parts, idx)
		}
	}
	parts = append(parts, p.expect(token.T_RBRACE))
	return p.intern.Node(KindArrayAccessExpr, parts...)
}

func (p *Parser) parseCallOrFirstClass(expr *GreenNode) *GreenNode {
	// First-class callable: expr(...)
	if p.peekType(1) == token.T_ELLIPSIS && p.peekType(2) == token.T_RPAREN {
		open := p.bump()
		ell := p.bump()
		close := p.bump()
		return p.intern.Node(KindFirstClassCallableExpr, expr, open, ell, close)
	}
	args := p.parseCallArgList()
	return p.intern.Node(KindCallExpr, expr, args)
}

// parseCallArgList parses '(' args ')' into KindArgList with structured exprs.
func (p *Parser) parseCallArgList() *GreenNode {
	open := p.expect(token.T_LPAREN)
	parts := []*GreenNode{open}
	for !p.at(token.T_RPAREN) && !p.at(token.T_EOF) {
		start := p.i
		parts = append(parts, p.parseCallArg())
		if p.at(token.T_COMMA) {
			parts = append(parts, p.bump())
			continue
		}
		if p.i == start {
			// No progress — bump to avoid infinite loops on unexpected tokens.
			parts = append(parts, p.bump())
			continue
		}
		break
	}
	parts = append(parts, p.expect(token.T_RPAREN))
	return p.intern.Node(KindArgList, parts...)
}

func (p *Parser) parseCallArg() *GreenNode {
	// Named args accept contextual keyword identifiers (e.g. match: false).
	if p.atIdentName() && p.peekType(1) == token.T_COLON {
		name := p.bump()
		colon := p.bump()
		expr := p.parseExpression()
		if expr == nil {
			return p.intern.Node(KindNamedArg, name, colon)
		}
		return p.intern.Node(KindNamedArg, name, colon, expr)
	}
	var parts []*GreenNode
	if p.at(token.T_ELLIPSIS) {
		parts = append(parts, p.bump())
	}
	if expr := p.parseExpression(); expr != nil {
		parts = append(parts, expr)
	}
	return p.intern.Node(KindArg, parts...)
}

func (p *Parser) parsePrimaryExpr() *GreenNode {
	switch p.tok().Type {
	case token.T_VARIABLE:
		v := p.bump()
		return p.intern.Node(KindVariableExpr, v)
	case token.T_LNUMBER, token.T_DNUMBER, token.T_CONSTANT_ENCAPSED_STRING,
		token.T_TRUE, token.T_FALSE, token.T_NULL:
		return p.intern.Node(KindLiteralExpr, p.bump())
	case token.T_CONSTANT_STRING:
		if p.tok().Literal == "\"" {
			return p.parseInterpolatedString()
		}
		if p.tok().Literal == "`" {
			return p.parseInterpolatedString() // backticks share encapsulated machine
		}
		return p.intern.Node(KindLiteralExpr, p.bump())
	case token.T_START_HEREDOC, token.T_START_NOWDOC:
		return p.parseHeredoc()
	case token.T_LBRACKET:
		return p.parseArrayExpr(false)
	case token.T_ARRAY:
		if p.peekType(1) == token.T_LPAREN {
			return p.parseArrayExpr(true)
		}
		// bare `array` as name/type-ish identifier in expr position
		return p.intern.Node(KindLiteralExpr, p.parseName())
	case token.T_LIST:
		return p.parseListExpr()
	case token.T_NEW:
		return p.parseNewExpr()
	case token.T_MATCH:
		return p.parseMatchExpr()
	case token.T_LPAREN:
		return p.parseParenExpr()
	case token.T_ISSET, token.T_EMPTY, token.T_EXIT, token.T_DIE:
		return p.parseBuiltinCall()
	case token.T_FUNCTION, token.T_FN:
		return p.parseClosureOrArrow()
	case token.T_STATIC:
		// static function(...) / static fn(...) are closures, not the name `static`.
		if p.peekType(1) == token.T_FUNCTION || p.peekType(1) == token.T_FN {
			return p.parseClosureOrArrow()
		}
		name := p.parseName()
		return name
	case token.T_ATTRIBUTE:
		// Attributes before closures/anon classes in expr position.
		attrs := p.parseAttributeList()
		inner := p.parsePrefixExpr()
		if inner == nil {
			return attrs
		}
		// Re-wrap: attribute list as leading children of the inner root when possible.
		return p.intern.Node(inner.kind, append([]*GreenNode{attrs}, inner.children...)...)
	case token.T_ILLEGAL:
		if p.tok().Literal == "$" {
			return p.parseVariableVariable()
		}
		return nil
	case token.T_DOLLAR_OPEN_CURLY_BRACES:
		return p.parseDollarCurlyVar()
	default:
		if p.isNameStart() {
			name := p.parseName()
			return name
		}
		return nil
	}
}

func (p *Parser) parseVariableVariable() *GreenNode {
	// $$var — lexer emits T_ILLEGAL("$") then the inner variable/expr.
	dollar := p.bump()
	inner := p.parsePrefixExpr()
	if inner == nil {
		return p.intern.Node(KindVariableVariableExpr, dollar)
	}
	return p.intern.Node(KindVariableVariableExpr, dollar, inner)
}

func (p *Parser) parseDollarCurlyVar() *GreenNode {
	// ${expr}
	open := p.bump()
	inner := p.parseExpression()
	close := p.expect(token.T_RBRACE)
	if inner == nil {
		return p.intern.Node(KindVariableVariableExpr, open, close)
	}
	return p.intern.Node(KindVariableVariableExpr, open, inner, close)
}

func (p *Parser) parseParenExpr() *GreenNode {
	open := p.bump()
	inner := p.parseExpression()
	close := p.expect(token.T_RPAREN)
	if inner == nil {
		return p.intern.Node(KindParenExpr, open, close)
	}
	return p.intern.Node(KindParenExpr, open, inner, close)
}

func (p *Parser) parseBuiltinCall() *GreenNode {
	name := p.bump()
	if p.at(token.T_LPAREN) {
		args := p.parseCallArgList()
		return p.intern.Node(KindCallExpr, name, args)
	}
	// exit/die may take an unparenthesized expression.
	if expr := p.parseExprBp(precUnary); expr != nil {
		return p.intern.Node(KindCallExpr, name, expr)
	}
	return p.intern.Node(KindCallExpr, name)
}

func (p *Parser) parseNewExpr() *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.bump()) // new
	if p.at(token.T_ATTRIBUTE) {
		parts = append(parts, p.parseAttributeList())
	}
	if p.at(token.T_CLASS) {
		parts = append(parts, p.parseAnonymousClass())
		return p.intern.Node(KindNewExpr, parts...)
	}
	if p.at(token.T_LPAREN) {
		// new (expr)(...)
		parts = append(parts, p.parseParenExpr())
	} else if p.isNameStart() {
		parts = append(parts, p.parseName())
	} else if p.at(token.T_VARIABLE) {
		parts = append(parts, p.intern.Node(KindVariableExpr, p.bump()))
	}
	if p.at(token.T_LPAREN) {
		parts = append(parts, p.parseCallArgList())
	}
	return p.intern.Node(KindNewExpr, parts...)
}

// parseClosureOrArrow parses function(){} / fn()=> / static variants as structured greens.
func (p *Parser) parseClosureOrArrow() *GreenNode {
	var parts []*GreenNode
	if p.at(token.T_STATIC) {
		parts = append(parts, p.bump())
	}
	isArrow := p.at(token.T_FN)
	if p.at(token.T_FUNCTION) || p.at(token.T_FN) {
		parts = append(parts, p.bump())
	} else {
		p.errorf("expected function or fn")
		return p.intern.Node(KindError, parts...)
	}
	if p.at(token.T_AMPERSAND) {
		parts = append(parts, p.bump())
	}
	parts = append(parts, p.expect(token.T_LPAREN))
	parts = append(parts, p.parseParamList())
	parts = append(parts, p.expect(token.T_RPAREN))
	if !isArrow && p.at(token.T_USE) {
		parts = append(parts, p.parseClosureUseClause())
	}
	if p.at(token.T_COLON) {
		parts = append(parts, p.bump())
		parts = append(parts, p.parseType())
	}
	if isArrow {
		parts = append(parts, p.expect(token.T_DOUBLE_ARROW))
		if expr := p.parseExpression(); expr != nil {
			parts = append(parts, expr)
		}
		return p.intern.Node(KindArrowFunctionExpr, parts...)
	}
	if p.at(token.T_LBRACE) {
		if p.SkipFunctionBodies {
			parts = append(parts, p.parseBalancedBlock())
		} else {
			parts = append(parts, p.parseStatementList())
		}
	}
	return p.intern.Node(KindClosureExpr, parts...)
}

func (p *Parser) parseClosureUseClause() *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.expect(token.T_USE))
	parts = append(parts, p.expect(token.T_LPAREN))
	for !p.at(token.T_RPAREN) && !p.at(token.T_EOF) {
		if p.at(token.T_COMMA) {
			parts = append(parts, p.bump())
			continue
		}
		if p.at(token.T_AMPERSAND) {
			parts = append(parts, p.bump())
		}
		if p.at(token.T_VARIABLE) {
			parts = append(parts, p.bump())
			continue
		}
		parts = append(parts, p.bump())
	}
	parts = append(parts, p.expect(token.T_RPAREN))
	return p.intern.Node(KindClosureUseClause, parts...)
}

// parseAnonymousClass parses `class [(args)] [extends …] [implements …] { members }`.
func (p *Parser) parseAnonymousClass() *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.expect(token.T_CLASS))
	if p.at(token.T_LPAREN) {
		parts = append(parts, p.parseCallArgList())
	}
	if p.at(token.T_EXTENDS) {
		parts = append(parts, p.parseExtendsClause())
	}
	if p.at(token.T_IMPLEMENTS) {
		parts = append(parts, p.parseImplementsClause())
	}
	if p.at(token.T_LBRACE) {
		if p.SkipFunctionBodies {
			parts = append(parts, p.parseBalancedBlock())
		} else {
			parts = append(parts, p.parseMemberList())
		}
	}
	return p.intern.Node(KindAnonymousClass, parts...)
}

func (p *Parser) parseArrayExpr(useArrayKeyword bool) *GreenNode {
	var parts []*GreenNode
	var closeType token.TokenType
	if useArrayKeyword {
		parts = append(parts, p.bump()) // array
		parts = append(parts, p.expect(token.T_LPAREN))
		closeType = token.T_RPAREN
	} else {
		parts = append(parts, p.bump()) // [
		closeType = token.T_RBRACKET
	}
	for !p.at(closeType) && !p.at(token.T_EOF) {
		parts = append(parts, p.parseArrayElement())
		if p.at(token.T_COMMA) {
			parts = append(parts, p.bump())
			continue
		}
		break
	}
	parts = append(parts, p.expect(closeType))
	return p.intern.Node(KindArrayExpr, parts...)
}

func (p *Parser) parseArrayElement() *GreenNode {
	var parts []*GreenNode
	if p.at(token.T_ELLIPSIS) {
		parts = append(parts, p.bump())
		if expr := p.parseExpression(); expr != nil {
			parts = append(parts, expr)
		}
		return p.intern.Node(KindArrayElement, parts...)
	}
	if p.at(token.T_AMPERSAND) {
		parts = append(parts, p.bump())
	}
	first := p.parseExpression()
	if first != nil {
		parts = append(parts, first)
	}
	if p.at(token.T_DOUBLE_ARROW) {
		parts = append(parts, p.bump())
		if p.at(token.T_AMPERSAND) {
			parts = append(parts, p.bump())
		}
		if val := p.parseExpression(); val != nil {
			parts = append(parts, val)
		}
	}
	return p.intern.Node(KindArrayElement, parts...)
}

func (p *Parser) parseListExpr() *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.bump()) // list
	parts = append(parts, p.expect(token.T_LPAREN))
	for !p.at(token.T_RPAREN) && !p.at(token.T_EOF) {
		if p.at(token.T_COMMA) {
			parts = append(parts, p.bump())
			continue
		}
		parts = append(parts, p.parseArrayElement())
		if p.at(token.T_COMMA) {
			parts = append(parts, p.bump())
			continue
		}
		break
	}
	parts = append(parts, p.expect(token.T_RPAREN))
	return p.intern.Node(KindListExpr, parts...)
}

// parseExprListComma parses one or more expressions separated by commas until stop.
func (p *Parser) parseExprListComma() []*GreenNode {
	var parts []*GreenNode
	for {
		if expr := p.parseExpression(); expr != nil {
			parts = append(parts, expr)
		}
		if p.at(token.T_COMMA) {
			parts = append(parts, p.bump())
			continue
		}
		break
	}
	return parts
}

// appendSimpleParenExpr appends '(' expr ')' with a structured expression.
func (p *Parser) appendSimpleParenExpr(parts []*GreenNode) []*GreenNode {
	parts = append(parts, p.expect(token.T_LPAREN))
	if !p.at(token.T_RPAREN) && !p.at(token.T_EOF) {
		if expr := p.parseExpression(); expr != nil {
			parts = append(parts, expr)
		}
	}
	parts = append(parts, p.expect(token.T_RPAREN))
	return parts
}

// appendForParenHeader parses for( init ; cond ; loop ) with structured exprs.
func (p *Parser) appendForParenHeader(parts []*GreenNode) []*GreenNode {
	parts = append(parts, p.expect(token.T_LPAREN))
	// init
	if !p.at(token.T_SEMICOLON) && !p.at(token.T_RPAREN) {
		parts = append(parts, p.parseExprListComma()...)
	}
	if p.at(token.T_SEMICOLON) {
		parts = append(parts, p.bump())
	}
	// cond
	if !p.at(token.T_SEMICOLON) && !p.at(token.T_RPAREN) {
		parts = append(parts, p.parseExprListComma()...)
	}
	if p.at(token.T_SEMICOLON) {
		parts = append(parts, p.bump())
	}
	// loop
	if !p.at(token.T_RPAREN) && !p.at(token.T_EOF) {
		parts = append(parts, p.parseExprListComma()...)
	}
	parts = append(parts, p.expect(token.T_RPAREN))
	return parts
}

// appendForeachParenHeader parses foreach( expr as [&] $v [| $k => $v] ).
func (p *Parser) appendForeachParenHeader(parts []*GreenNode) []*GreenNode {
	parts = append(parts, p.expect(token.T_LPAREN))
	if expr := p.parseExpression(); expr != nil {
		parts = append(parts, expr)
	}
	if p.at(token.T_AS) {
		parts = append(parts, p.bump())
		if p.at(token.T_AMPERSAND) {
			parts = append(parts, p.bump())
		}
		if first := p.parseExpression(); first != nil {
			parts = append(parts, first)
		}
		if p.at(token.T_DOUBLE_ARROW) {
			parts = append(parts, p.bump())
			if p.at(token.T_AMPERSAND) {
				parts = append(parts, p.bump())
			}
			if second := p.parseExpression(); second != nil {
				parts = append(parts, second)
			}
		}
	}
	parts = append(parts, p.expect(token.T_RPAREN))
	return parts
}

// appendUnsetParenArgs parses unset( expr, ... ).
func (p *Parser) appendUnsetParenArgs(parts []*GreenNode) []*GreenNode {
	parts = append(parts, p.expect(token.T_LPAREN))
	if !p.at(token.T_RPAREN) {
		parts = append(parts, p.parseExprListComma()...)
	}
	parts = append(parts, p.expect(token.T_RPAREN))
	return parts
}
