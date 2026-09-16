package syntax

import (
	"github.com/ayanozturk/go-php-parser/lexer"
	"github.com/ayanozturk/go-php-parser/token"
)

// Parser builds green trees from a Zend-faithful token stream.
type Parser struct {
	src                []byte
	tokens             []token.Token
	i                  int
	intern             *Interner
	diags              []Diagnostic
	SkipFunctionBodies bool // when true, function/method bodies stay KindTokenList blobs
}

func NewParser(src []byte) *Parser {
	return &Parser{
		src:    src,
		tokens: lexFragment(src),
		intern: NewInterner(src),
	}
}

// ParseOptions configures tiered parse cost (indexer vs open-file).
type ParseOptions struct {
	SkipFunctionBodies bool
}

// lexFragment tokenizes a code snippet in PHP mode (not HTML-at-start),
// used for name/type/attribute unit parses. Full files use LexAll via Parse.
func lexFragment(src []byte) []token.Token {
	l := lexer.NewBytes(src)
	var toks []token.Token
	for {
		tok := l.NextToken()
		toks = append(toks, tok)
		if tok.Type == token.T_EOF {
			return toks
		}
	}
}

// Parse produces a structured File with fully parsed function/method bodies.
func Parse(src []byte) *ParseResult {
	return ParseWith(src, ParseOptions{})
}

// ParseForIndex produces a structured File with function/method bodies as
// round-trip-safe KindTokenList blobs (declaration-tier for workspace indexing).
func ParseForIndex(src []byte) *ParseResult {
	return ParseWith(src, ParseOptions{SkipFunctionBodies: true})
}

// ParseWith produces a structured File using the given options.
func ParseWith(src []byte, opts ParseOptions) *ParseResult {
	p := &Parser{
		src:                src,
		tokens:             lexer.LexAll(src),
		intern:             NewInterner(src),
		SkipFunctionBodies: opts.SkipFunctionBodies,
	}
	var items []*GreenNode

	for !p.at(token.T_EOF) {
		if g := p.tryParseStructured(); g != nil {
			items = append(items, g)
			continue
		}
		items = append(items, p.bump())
	}
	// Preserve EOF trailing trivia (final newline, comments).
	items = append(items, p.bump())

	green := p.intern.Node(KindFile, items...)
	f := &File{
		Source: src,
		Lines:  token.NewLineTable(src),
		Green:  green,
		Tokens: p.tokens,
	}
	BindRed(f)
	return &ParseResult{File: f, Diagnostics: p.diags, PHPVersion: "8.3"}
}

func (p *Parser) at(tt token.TokenType) bool {
	return p.tok().Type == tt
}

func (p *Parser) tok() token.Token {
	if p.i >= len(p.tokens) {
		return token.Token{Type: token.T_EOF}
	}
	return p.tokens[p.i]
}

func (p *Parser) bump() *GreenNode {
	tok := p.tok()
	p.i++
	return p.intern.Token(tok)
}

func (p *Parser) expect(tt token.TokenType) *GreenNode {
	if p.at(tt) {
		return p.bump()
	}
	pos := p.tok().Pos.Offset
	p.diags = append(p.diags, Diagnostic{
		Message: "expected " + tt.String() + ", got " + p.tok().Type.String(),
		Span:    Span{Start: pos, End: pos},
	})
	miss := token.Token{Type: tt, Pos: p.tok().Pos, End: p.tok().Pos}
	return p.intern.Missing(miss)
}

func (p *Parser) errorf(msg string) {
	pos := p.tok().Pos.Offset
	p.diags = append(p.diags, Diagnostic{Message: msg, Span: Span{Start: pos, End: pos}})
}

func (p *Parser) tryParseStructured() *GreenNode {
	switch {
	case p.at(token.T_ATTRIBUTE):
		return p.parseAttributeList()
	case p.at(token.T_NAMESPACE):
		return p.parseNamespaceDecl()
	case p.at(token.T_USE):
		return p.parseUseDecl()
	case p.isClassLikeStart():
		return p.parseClassLikeDecl()
	case p.isFunctionStart():
		return p.parseFunctionDecl()
	case p.at(token.T_CONST):
		return p.parseConstDecl()
	case p.at(token.T_DECLARE):
		return p.parseDeclareStmt()
	case p.at(token.T_GLOBAL):
		return p.parseGlobalStmt()
	case p.at(token.T_STATIC) && p.i+1 < len(p.tokens) && p.tokens[p.i+1].Type == token.T_VARIABLE:
		return p.parseStaticVarStmt()
	case p.at(token.T_ECHO):
		return p.parseEchoStmt()
	case p.at(token.T_OPEN_TAG_WITH_ECHO):
		return p.parseShortEchoStmt()
	case p.at(token.T_RETURN):
		return p.parseReturnStmt()
	case p.at(token.T_IF):
		return p.parseIfStmt()
	case p.at(token.T_WHILE):
		return p.parseWhileStmt()
	case p.at(token.T_DO):
		return p.parseDoWhileStmt()
	case p.at(token.T_FOR):
		return p.parseForStmt()
	case p.at(token.T_FOREACH):
		return p.parseForeachStmt()
	case p.at(token.T_SWITCH):
		return p.parseSwitchStmt()
	case p.at(token.T_MATCH):
		return p.parseMatchExprStmt()
	case p.at(token.T_TRY):
		return p.parseTryStmt()
	case p.at(token.T_BREAK):
		return p.parseBreakStmt()
	case p.at(token.T_CONTINUE):
		return p.parseContinueStmt()
	case p.at(token.T_THROW):
		return p.parseThrowStmt()
	case p.at(token.T_UNSET):
		return p.parseUnsetStmt()
	case p.at(token.T_VARIABLE), p.at(token.T_INC), p.at(token.T_DEC),
		p.at(token.T_NEW), p.at(token.T_CLONE), p.at(token.T_PRINT),
		p.at(token.T_LIST), p.at(token.T_ARRAY), p.at(token.T_LBRACKET),
		p.at(token.T_LPAREN), p.at(token.T_LNUMBER), p.at(token.T_DNUMBER),
		p.at(token.T_PLUS), p.at(token.T_MINUS), p.at(token.T_NOT), p.at(token.T_TILDE),
		p.at(token.T_AT), p.at(token.T_AMPERSAND), p.at(token.T_YIELD), p.at(token.T_YIELD_FROM),
		p.at(token.T_INCLUDE), p.at(token.T_INCLUDE_ONCE), p.at(token.T_REQUIRE), p.at(token.T_REQUIRE_ONCE),
		p.at(token.T_ISSET), p.at(token.T_EMPTY), p.at(token.T_EXIT), p.at(token.T_DIE),
		p.at(token.T_TRUE), p.at(token.T_FALSE), p.at(token.T_NULL),
		p.at(token.T_FUNCTION), p.at(token.T_FN):
		return p.parseExpressionStmt()
	case p.at(token.T_ILLEGAL) && p.tok().Literal == "$":
		return p.parseExpressionStmt()
	case p.isNameStart():
		// Function/const calls and bare names as expression statements.
		return p.parseExpressionStmt()
	case p.at(token.T_CONSTANT_ENCAPSED_STRING):
		return p.parseExpressionStmt()
	case p.at(token.T_CONSTANT_STRING) && (p.tok().Literal == "\"" || p.tok().Literal == "`"):
		return p.parseExpressionStmt()
	case p.at(token.T_START_HEREDOC) || p.at(token.T_START_NOWDOC):
		return p.parseExpressionStmt()
	case p.at(token.T_SEMICOLON):
		return p.intern.Node(KindEmptyStmt, p.bump())
	default:
		return nil
	}
}

func (p *Parser) isModifierToken(tt token.TokenType) bool {
	switch tt {
	case token.T_PUBLIC, token.T_PROTECTED, token.T_PRIVATE,
		token.T_STATIC, token.T_ABSTRACT, token.T_FINAL, token.T_READONLY:
		return true
	default:
		return false
	}
}

func (p *Parser) skipModifierTokens(i int) int {
	for i < len(p.tokens) {
		tt := p.tokens[i].Type
		if !p.isModifierToken(tt) {
			return i
		}
		if (tt == token.T_PUBLIC || tt == token.T_PROTECTED || tt == token.T_PRIVATE) &&
			i+3 < len(p.tokens) &&
			p.tokens[i+1].Type == token.T_LPAREN &&
			p.tokens[i+2].Type == token.T_STRING && p.tokens[i+2].Literal == "set" &&
			p.tokens[i+3].Type == token.T_RPAREN {
			i += 4
			continue
		}
		i++
	}
	return i
}

func (p *Parser) isClassLikeStart() bool {
	i := p.skipModifierTokens(p.i)
	if i >= len(p.tokens) {
		return false
	}
	switch p.tokens[i].Type {
	case token.T_CLASS, token.T_INTERFACE, token.T_TRAIT, token.T_ENUM:
		return true
	default:
		return false
	}
}

func (p *Parser) isFunctionStart() bool {
	i := p.skipModifierTokens(p.i)
	return i < len(p.tokens) && p.tokens[i].Type == token.T_FUNCTION
}

func (p *Parser) parseModifierList() *GreenNode {
	var parts []*GreenNode
	for p.isModifierToken(p.tok().Type) {
		tt := p.tok().Type
		if (tt == token.T_PUBLIC || tt == token.T_PROTECTED || tt == token.T_PRIVATE) &&
			p.i+3 < len(p.tokens) &&
			p.tokens[p.i+1].Type == token.T_LPAREN &&
			p.tokens[p.i+2].Type == token.T_STRING && p.tokens[p.i+2].Literal == "set" &&
			p.tokens[p.i+3].Type == token.T_RPAREN {
			parts = append(parts, p.bump())
			parts = append(parts, p.bump())
			parts = append(parts, p.bump())
			parts = append(parts, p.bump())
			continue
		}
		parts = append(parts, p.bump())
	}
	if len(parts) == 0 {
		return nil
	}
	return p.intern.Node(KindModifierList, parts...)
}

func (p *Parser) isNameStart() bool {
	switch p.tok().Type {
	case token.T_STRING, token.T_NS_SEPARATOR, token.T_NAMESPACE, token.T_SELF, token.T_PARENT, token.T_STATIC:
		return true
	default:
		return false
	}
}

// ParseName parses Unqualified / Qualified / FullyQualified / Relative names.
func (p *Parser) ParseName() *GreenNode {
	return p.parseName()
}

func (p *Parser) parseName() *GreenNode {
	var parts []*GreenNode
	kind := KindUnqualifiedName

	if p.at(token.T_NAMESPACE) {
		parts = append(parts, p.bump())
		if p.at(token.T_NS_SEPARATOR) {
			parts = append(parts, p.bump())
			kind = KindRelativeName
		} else {
			return p.intern.Node(KindUnqualifiedName, parts...)
		}
	} else if p.at(token.T_NS_SEPARATOR) {
		parts = append(parts, p.bump())
		kind = KindFullyQualifiedName
	}

	if !(p.at(token.T_STRING) || p.at(token.T_SELF) || p.at(token.T_PARENT) || p.at(token.T_STATIC)) {
		p.errorf("expected name")
		return p.intern.Node(kind, parts...)
	}
	parts = append(parts, p.bump())

	for p.at(token.T_NS_SEPARATOR) {
		parts = append(parts, p.bump())
		if !(p.at(token.T_STRING) || p.at(token.T_SELF) || p.at(token.T_PARENT) || p.at(token.T_STATIC)) {
			p.errorf("expected name part after \\")
			break
		}
		parts = append(parts, p.bump())
		if kind == KindUnqualifiedName {
			kind = KindQualifiedName
		}
	}
	return p.intern.Node(kind, parts...)
}

func (p *Parser) isPrimitiveType() bool {
	if p.tok().Type != token.T_STRING && p.tok().Type != token.T_ARRAY &&
		p.tok().Type != token.T_CALLABLE && p.tok().Type != token.T_MIXED &&
		p.tok().Type != token.T_NEVER && p.tok().Type != token.T_TRUE &&
		p.tok().Type != token.T_FALSE && p.tok().Type != token.T_NULL {
		return false
	}
	switch lowercase(p.tok().Literal) {
	case "int", "float", "string", "bool", "array", "object", "iterable",
		"void", "mixed", "never", "null", "false", "true", "callable", "parent", "self", "static":
		return true
	default:
		return false
	}
}

func lowercase(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

// ParseType parses a PHP type including nullable, union, intersection, DNF.
func (p *Parser) ParseType() *GreenNode {
	return p.parseType()
}

func (p *Parser) parseType() *GreenNode {
	if p.at(token.T_QUESTION) {
		q := p.bump()
		inner := p.parseTypeAtom()
		return p.intern.Node(KindNullableType, q, inner)
	}
	first := p.parseTypeAtom()
	if p.at(token.T_PIPE) {
		parts := []*GreenNode{first}
		for p.at(token.T_PIPE) {
			parts = append(parts, p.bump())
			parts = append(parts, p.parseTypeAtom())
		}
		return p.intern.Node(KindUnionType, parts...)
	}
	if p.at(token.T_AMPERSAND) {
		// Ambiguous with references in params — only treat as intersection when
		// followed by a type atom (name / ( ).
		parts := []*GreenNode{first}
		for p.at(token.T_AMPERSAND) {
			amp := p.bump()
			next := p.parseTypeAtom()
			parts = append(parts, amp, next)
		}
		return p.intern.Node(KindIntersectionType, parts...)
	}
	return first
}

func (p *Parser) parseTypeAtom() *GreenNode {
	if p.at(token.T_LPAREN) {
		open := p.bump()
		inner := p.parseType()
		close := p.expect(token.T_RPAREN)
		return p.intern.Node(KindParenthesizedType, open, inner, close)
	}
	if p.at(token.T_CALLABLE) && p.i+1 < len(p.tokens) && p.tokens[p.i+1].Type == token.T_LPAREN {
		return p.parseCallableType()
	}
	if p.isPrimitiveType() && !p.at(token.T_NS_SEPARATOR) {
		// self/parent/static/primitives as PrimitiveType; class names as NamedType.
		lit := lowercase(p.tok().Literal)
		switch lit {
		case "int", "float", "string", "bool", "array", "object", "iterable",
			"void", "mixed", "never", "null", "false", "true", "callable":
			return p.intern.Node(KindPrimitiveType, p.bump())
		}
	}
	if p.isNameStart() {
		return p.intern.Node(KindNamedType, p.parseName())
	}
	p.errorf("expected type")
	return p.intern.Node(KindError, p.bump())
}

func (p *Parser) parseCallableType() *GreenNode {
	callable := p.bump()
	open := p.expect(token.T_LPAREN)
	paramList := p.parseCallableParamList()
	close := p.expect(token.T_RPAREN)
	parts := []*GreenNode{callable, open, paramList, close}
	if p.at(token.T_COLON) {
		parts = append(parts, p.bump(), p.parseType())
	}
	return p.intern.Node(KindCallableType, parts...)
}

func (p *Parser) parseCallableParamList() *GreenNode {
	var parts []*GreenNode
	for !p.at(token.T_RPAREN) && !p.at(token.T_EOF) {
		if p.at(token.T_COMMA) {
			parts = append(parts, p.bump())
			continue
		}
		parts = append(parts, p.parseCallableParam())
	}
	return p.intern.Node(KindCallableParamList, parts...)
}

func (p *Parser) parseCallableParam() *GreenNode {
	var parts []*GreenNode
	if p.isTypeStart() && !p.at(token.T_VARIABLE) {
		parts = append(parts, p.parseType())
	}
	if p.at(token.T_VARIABLE) {
		parts = append(parts, p.bump())
	}
	return p.intern.Node(KindCallableParam, parts...)
}

func (p *Parser) parseNamespaceDecl() *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.expect(token.T_NAMESPACE))
	if p.at(token.T_STRING) || p.at(token.T_NS_SEPARATOR) {
		parts = append(parts, p.parseName())
	}
	if p.at(token.T_SEMICOLON) {
		parts = append(parts, p.bump())
	} else if p.at(token.T_LBRACE) {
		parts = append(parts, p.parseStatementList())
	}
	return p.intern.Node(KindNamespaceDecl, parts...)
}

func (p *Parser) parseUseDecl() *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.expect(token.T_USE))
	if p.at(token.T_FUNCTION) || p.at(token.T_CONST) {
		parts = append(parts, p.bump())
	}
	for {
		parts = append(parts, p.parseUseClause())
		if p.at(token.T_COMMA) {
			parts = append(parts, p.bump())
			continue
		}
		break
	}
	parts = append(parts, p.expect(token.T_SEMICOLON))
	return p.intern.Node(KindUseDecl, parts...)
}

// parseUseClause parses either a plain `Name [as Alias]` clause or a group-use
// `Prefix\{ Item, Item as Alias, ... }`.
func (p *Parser) parseUseClause() *GreenNode {
	clauseParts := []*GreenNode{p.parseName()}
	// Group-use prefix ends with `\`; the lexer emits T_BACKSLASH (not
	// T_NS_SEPARATOR) immediately before `{`.
	if p.at(token.T_BACKSLASH) || p.at(token.T_NS_SEPARATOR) {
		if p.i+1 < len(p.tokens) && p.tokens[p.i+1].Type == token.T_LBRACE {
			clauseParts = append(clauseParts, p.bump())
			clauseParts = append(clauseParts, p.parseUseGroup())
			return p.intern.Node(KindUseClause, clauseParts...)
		}
	}
	if p.at(token.T_LBRACE) {
		clauseParts = append(clauseParts, p.parseUseGroup())
		return p.intern.Node(KindUseClause, clauseParts...)
	}
	if p.at(token.T_AS) {
		clauseParts = append(clauseParts, p.bump())
		if p.at(token.T_STRING) {
			clauseParts = append(clauseParts, p.intern.Node(KindUnqualifiedName, p.bump()))
		}
	}
	return p.intern.Node(KindUseClause, clauseParts...)
}

func (p *Parser) parseUseGroup() *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.expect(token.T_LBRACE))
	for {
		if p.at(token.T_RBRACE) {
			break
		}
		// Group items may be function/const-typed: use Foo\{ function bar, const BAZ }
		itemParts := []*GreenNode{}
		if p.at(token.T_FUNCTION) || p.at(token.T_CONST) {
			itemParts = append(itemParts, p.bump())
		}
		itemParts = append(itemParts, p.parseName())
		if p.at(token.T_AS) {
			itemParts = append(itemParts, p.bump())
			if p.at(token.T_STRING) {
				itemParts = append(itemParts, p.intern.Node(KindUnqualifiedName, p.bump()))
			}
		}
		parts = append(parts, p.intern.Node(KindUseClause, itemParts...))
		if p.at(token.T_COMMA) {
			parts = append(parts, p.bump())
			continue
		}
		break
	}
	parts = append(parts, p.expect(token.T_RBRACE))
	return p.intern.Node(KindUseGroup, parts...)
}

func (p *Parser) parseAttributeList() *GreenNode {
	var groups []*GreenNode
	for p.at(token.T_ATTRIBUTE) {
		groups = append(groups, p.parseAttributeGroup())
	}
	return p.intern.Node(KindAttributeList, groups...)
}

func (p *Parser) parseAttributeGroup() *GreenNode {
	hash := p.expect(token.T_ATTRIBUTE) // #[
	var parts []*GreenNode
	parts = append(parts, hash)
	for {
		parts = append(parts, p.parseAttribute())
		if p.at(token.T_COMMA) {
			parts = append(parts, p.bump())
			continue
		}
		break
	}
	parts = append(parts, p.expect(token.T_RBRACKET))
	return p.intern.Node(KindAttributeGroup, parts...)
}

func (p *Parser) parseAttribute() *GreenNode {
	name := p.parseName()
	children := []*GreenNode{name}
	if p.at(token.T_LPAREN) {
		children = append(children, p.parseArgList())
	}
	return p.intern.Node(KindAttribute, children...)
}

func (p *Parser) parseArgList() *GreenNode {
	open := p.expect(token.T_LPAREN)
	parts := []*GreenNode{open}
	for !p.at(token.T_RPAREN) && !p.at(token.T_EOF) {
		parts = append(parts, p.parseArg())
		if p.at(token.T_COMMA) {
			parts = append(parts, p.bump())
			continue
		}
		break
	}
	parts = append(parts, p.expect(token.T_RPAREN))
	return p.intern.Node(KindArgList, parts...)
}

func (p *Parser) parseArg() *GreenNode {
	return p.parseCallArg()
}

func (p *Parser) parseStringLiteral() *GreenNode {
	return p.intern.Node(KindStringLiteral, p.bump())
}

func (p *Parser) parseInterpolatedString() *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.bump()) // opening "
	for !p.at(token.T_EOF) {
		if p.at(token.T_CONSTANT_STRING) && p.tok().Literal == "\"" {
			parts = append(parts, p.bump())
			break
		}
		switch p.tok().Type {
		case token.T_ENCAPSED_AND_WHITESPACE:
			parts = append(parts, p.intern.Node(KindStringPart, p.bump()))
		case token.T_VARIABLE:
			parts = append(parts, p.intern.Node(KindVariablePart, p.bump()))
		case token.T_CURLY_OPEN, token.T_DOLLAR_OPEN_CURLY_BRACES:
			parts = append(parts, p.parseEncapsulatedExpr())
		default:
			parts = append(parts, p.bump())
		}
	}
	return p.intern.Node(KindStringLiteral, parts...)
}

func (p *Parser) parseEncapsulatedExpr() *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.bump())
	depth := 1
	for !p.at(token.T_EOF) && depth > 0 {
		switch p.tok().Type {
		case token.T_LBRACE, token.T_CURLY_OPEN, token.T_DOLLAR_OPEN_CURLY_BRACES:
			depth++
		case token.T_RBRACE:
			depth--
		}
		parts = append(parts, p.bump())
	}
	return p.intern.Node(KindEncapsulatedExpr, parts...)
}

func (p *Parser) parseHeredoc() *GreenNode {
	start := p.bump()
	kind := KindHeredoc
	if start.token != nil && start.token.Type == token.T_START_NOWDOC {
		kind = KindNowdoc
	}
	var parts []*GreenNode
	parts = append(parts, start)
	for !p.at(token.T_EOF) && !p.at(token.T_END_HEREDOC) && !p.at(token.T_END_NOWDOC) {
		switch p.tok().Type {
		case token.T_ENCAPSED_AND_WHITESPACE:
			parts = append(parts, p.intern.Node(KindStringPart, p.bump()))
		case token.T_VARIABLE:
			parts = append(parts, p.intern.Node(KindVariablePart, p.bump()))
		case token.T_CURLY_OPEN, token.T_DOLLAR_OPEN_CURLY_BRACES:
			parts = append(parts, p.parseEncapsulatedExpr())
		default:
			parts = append(parts, p.bump())
		}
	}
	if p.at(token.T_END_HEREDOC) || p.at(token.T_END_NOWDOC) {
		parts = append(parts, p.bump())
	}
	return p.intern.Node(kind, parts...)
}

// parseFunctionDecl emits a green FunctionDecl/MethodDecl covering signature + body.
// With SkipFunctionBodies, the body is a round-trip-safe KindTokenList blob;
// otherwise the body is a structured KindStatementList.
func (p *Parser) parseFunctionDecl() *GreenNode {
	var parts []*GreenNode
	if mods := p.parseModifierList(); mods != nil {
		parts = append(parts, mods)
	}
	parts = append(parts, p.expect(token.T_FUNCTION))
	if p.at(token.T_AMPERSAND) {
		parts = append(parts, p.bump())
	}
	// Zend keeps keyword tokens (T_LIST, T_DEFAULT, …) after `function`; accept them as names.
	if name := p.parseIdentName(); name != nil {
		parts = append(parts, name)
	}
	parts = append(parts, p.expect(token.T_LPAREN))
	parts = append(parts, p.parseParamList())
	parts = append(parts, p.expect(token.T_RPAREN))
	if p.at(token.T_COLON) {
		parts = append(parts, p.bump())
		parts = append(parts, p.parseType())
	}
	kind := KindFunctionDecl
	if p.at(token.T_LBRACE) {
		if p.SkipFunctionBodies {
			parts = append(parts, p.parseBalancedBlock())
		} else {
			parts = append(parts, p.parseStatementList())
		}
	} else if p.at(token.T_SEMICOLON) {
		parts = append(parts, p.bump())
		kind = KindMethodDecl // interface/abstract style
	}
	return p.intern.Node(kind, parts...)
}

// parseIdentName consumes a T_STRING or keyword-identifier (e.g. T_LIST) as an UnqualifiedName.
// PHP allows most keywords as method/function/const names while token_get_all still emits the keyword kind.
func (p *Parser) parseIdentName() *GreenNode {
	if !p.atIdentName() {
		return nil
	}
	return p.intern.Node(KindUnqualifiedName, p.bump())
}

func (p *Parser) atIdentName() bool {
	if p.at(token.T_STRING) {
		return true
	}
	lit := p.tok().Literal
	if lit == "" {
		return false
	}
	switch p.tok().Type {
	case token.T_LPAREN, token.T_RPAREN, token.T_LBRACE, token.T_RBRACE,
		token.T_LBRACKET, token.T_RBRACKET, token.T_SEMICOLON, token.T_COMMA,
		token.T_AMPERSAND, token.T_ELLIPSIS, token.T_COLON, token.T_DOUBLE_ARROW,
		token.T_ASSIGN, token.T_EOF, token.T_VARIABLE, token.T_NS_SEPARATOR,
		token.T_OBJECT_OPERATOR, token.T_NULLSAFE_OBJECT_OPERATOR, token.T_DOUBLE_COLON,
		token.T_ATTRIBUTE, token.T_OPEN_TAG, token.T_OPEN_TAG_WITH_ECHO, token.T_CLOSE_TAG, token.T_INLINE_HTML,
		token.T_CONSTANT_STRING, token.T_CONSTANT_ENCAPSED_STRING, token.T_LNUMBER, token.T_DNUMBER,
		token.T_START_HEREDOC, token.T_START_NOWDOC, token.T_END_HEREDOC, token.T_END_NOWDOC,
		token.T_ENCAPSED_AND_WHITESPACE, token.T_CURLY_OPEN, token.T_DOLLAR_OPEN_CURLY_BRACES,
		token.T_ILLEGAL:
		return false
	default:
		r := lit[0]
		return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_'
	}
}

func (p *Parser) parseClassLikeDecl() *GreenNode {
	var parts []*GreenNode
	if mods := p.parseModifierList(); mods != nil {
		parts = append(parts, mods)
	}
	var kind Kind
	switch p.tok().Type {
	case token.T_CLASS:
		kind = KindClassDecl
	case token.T_INTERFACE:
		kind = KindInterfaceDecl
	case token.T_TRAIT:
		kind = KindTraitDecl
	case token.T_ENUM:
		kind = KindEnumDecl
	default:
		p.errorf("expected class-like declaration")
		return p.intern.Node(KindError, p.bump())
	}
	parts = append(parts, p.bump()) // class|interface|trait|enum
	if p.at(token.T_STRING) {
		parts = append(parts, p.intern.Node(KindUnqualifiedName, p.bump()))
	}
	// enum backing type: enum Suit: string
	if kind == KindEnumDecl && p.at(token.T_COLON) {
		parts = append(parts, p.bump())
		parts = append(parts, p.parseType())
	}
	if p.at(token.T_EXTENDS) {
		parts = append(parts, p.parseExtendsClause())
	}
	if p.at(token.T_IMPLEMENTS) {
		parts = append(parts, p.parseImplementsClause())
	}
	if p.at(token.T_LBRACE) {
		parts = append(parts, p.parseMemberList())
	} else if p.at(token.T_SEMICOLON) {
		parts = append(parts, p.bump())
	}
	return p.intern.Node(kind, parts...)
}

func (p *Parser) parseExtendsClause() *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.expect(token.T_EXTENDS))
	parts = append(parts, p.parseName())
	for p.at(token.T_COMMA) {
		parts = append(parts, p.bump())
		parts = append(parts, p.parseName())
	}
	return p.intern.Node(KindExtendsClause, parts...)
}

func (p *Parser) parseImplementsClause() *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.expect(token.T_IMPLEMENTS))
	parts = append(parts, p.parseName())
	for p.at(token.T_COMMA) {
		parts = append(parts, p.bump())
		parts = append(parts, p.parseName())
	}
	return p.intern.Node(KindImplementsClause, parts...)
}

func (p *Parser) parseMemberList() *GreenNode {
	return p.parseBraceDelimitedList(KindMemberList, func() *GreenNode {
		if p.at(token.T_ATTRIBUTE) {
			return p.parseAttributeList()
		}
		if member := p.tryParseMember(); member != nil {
			return member
		}
		return nil
	})
}

func (p *Parser) parseStatementList() *GreenNode {
	return p.parseBraceDelimitedList(KindStatementList, func() *GreenNode {
		if p.at(token.T_ATTRIBUTE) {
			return p.parseAttributeList()
		}
		if stmt := p.tryParseStructured(); stmt != nil {
			return stmt
		}
		return nil
	})
}

// parseBraceDelimitedList covers `{` items… `}` with bump-fallback for identity.
func (p *Parser) parseBraceDelimitedList(kind Kind, tryItem func() *GreenNode) *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.expect(token.T_LBRACE))
	for !p.at(token.T_RBRACE) && !p.at(token.T_EOF) {
		if item := tryItem(); item != nil {
			parts = append(parts, item)
			continue
		}
		// Fallback: consume one token so progress continues (identity still holds).
		parts = append(parts, p.bump())
	}
	if p.at(token.T_RBRACE) {
		parts = append(parts, p.bump())
	}
	return p.intern.Node(kind, parts...)
}

func (p *Parser) tryParseMember() *GreenNode {
	if p.at(token.T_USE) {
		return p.parseUseTraitClause()
	}
	if p.at(token.T_CASE) {
		return p.parseEnumCase()
	}
	// Look past modifiers for member kind.
	i := p.skipModifierTokens(p.i)
	if i >= len(p.tokens) {
		return nil
	}
	switch p.tokens[i].Type {
	case token.T_FUNCTION:
		return p.parseFunctionDecl()
	case token.T_CONST:
		return p.parseClassConstDecl()
	case token.T_VARIABLE:
		return p.parsePropertyDecl()
	case token.T_STRING, token.T_QUESTION, token.T_LPAREN, token.T_NS_SEPARATOR,
		token.T_ARRAY, token.T_CALLABLE, token.T_MIXED, token.T_STATIC, token.T_SELF, token.T_PARENT:
		// Typed property: [modifiers] type $var
		return p.parsePropertyDecl()
	default:
		return nil
	}
}

func (p *Parser) parseUseTraitClause() *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.expect(token.T_USE))
	parts = append(parts, p.parseName())
	for p.at(token.T_COMMA) {
		parts = append(parts, p.bump())
		parts = append(parts, p.parseName())
	}
	if p.at(token.T_LBRACE) {
		parts = append(parts, p.parseTraitAdaptationList())
	} else if p.at(token.T_SEMICOLON) {
		parts = append(parts, p.bump())
	}
	return p.intern.Node(KindUseTraitClause, parts...)
}

func (p *Parser) parseTraitAdaptationList() *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.expect(token.T_LBRACE))
	for !p.at(token.T_RBRACE) && !p.at(token.T_EOF) {
		if p.at(token.T_SEMICOLON) {
			parts = append(parts, p.intern.Node(KindEmptyStmt, p.bump()))
			continue
		}
		if adapt := p.tryParseTraitAdaptation(); adapt != nil {
			parts = append(parts, adapt)
			continue
		}
		// Keep identity if adaptation shape is unexpected.
		parts = append(parts, p.bump())
	}
	if p.at(token.T_RBRACE) {
		parts = append(parts, p.bump())
	}
	return p.intern.Node(KindTraitAdaptationList, parts...)
}

func (p *Parser) tryParseTraitAdaptation() *GreenNode {
	if !(p.at(token.T_STRING) || p.at(token.T_NS_SEPARATOR) || p.isNameStart()) {
		return nil
	}
	start := p.i
	var parts []*GreenNode
	// Method reference: [Trait::]method
	name := p.parseName()
	if p.at(token.T_DOUBLE_COLON) {
		parts = append(parts, name, p.bump())
		if p.at(token.T_STRING) {
			parts = append(parts, p.intern.Node(KindUnqualifiedName, p.bump()))
		}
	} else {
		parts = append(parts, name)
	}
	switch {
	case p.at(token.T_AS):
		parts = append(parts, p.bump())
		if mods := p.parseVisibilityOnlyModifiers(); mods != nil {
			parts = append(parts, mods)
		}
		if p.at(token.T_STRING) {
			parts = append(parts, p.intern.Node(KindUnqualifiedName, p.bump()))
		}
	case p.at(token.T_INSTEADOF):
		parts = append(parts, p.bump())
		for {
			parts = append(parts, p.parseName())
			if p.at(token.T_COMMA) {
				parts = append(parts, p.bump())
				continue
			}
			break
		}
	default:
		p.i = start
		return nil
	}
	if p.at(token.T_SEMICOLON) {
		parts = append(parts, p.bump())
	}
	return p.intern.Node(KindTraitAdaptation, parts...)
}

func (p *Parser) parseVisibilityOnlyModifiers() *GreenNode {
	if p.at(token.T_PUBLIC) || p.at(token.T_PROTECTED) || p.at(token.T_PRIVATE) {
		return p.intern.Node(KindModifierList, p.bump())
	}
	// After T_AS, visibility keywords are lexed as T_STRING (semi-reserved).
	if p.at(token.T_STRING) {
		switch lowercase(p.tok().Literal) {
		case "public", "protected", "private":
			return p.intern.Node(KindModifierList, p.bump())
		}
	}
	return nil
}

func (p *Parser) parseConstDecl() *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.expect(token.T_CONST))
	for {
		if p.at(token.T_STRING) {
			parts = append(parts, p.intern.Node(KindUnqualifiedName, p.bump()))
		}
		if p.at(token.T_ASSIGN) {
			parts = append(parts, p.bump())
			if expr := p.parseExpression(); expr != nil {
				parts = append(parts, expr)
			} else {
				parts = append(parts, p.parseUntilCommaOrSemi())
			}
		}
		if p.at(token.T_COMMA) {
			parts = append(parts, p.bump())
			continue
		}
		break
	}
	if p.at(token.T_SEMICOLON) {
		parts = append(parts, p.bump())
	}
	return p.intern.Node(KindConstDecl, parts...)
}

func (p *Parser) parseDeclareStmt() *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.expect(token.T_DECLARE))
	parts = append(parts, p.expect(token.T_LPAREN))
	// Directive list as tokens until ')' — keeps identity without a full expr grammar.
	depth := 1
	var dirParts []*GreenNode
	for !p.at(token.T_EOF) && depth > 0 {
		switch p.tok().Type {
		case token.T_LPAREN:
			depth++
		case token.T_RPAREN:
			depth--
			if depth == 0 {
				break
			}
		}
		if depth > 0 {
			dirParts = append(dirParts, p.bump())
		}
	}
	if len(dirParts) > 0 {
		parts = append(parts, p.intern.Node(KindTokenList, dirParts...))
	}
	parts = append(parts, p.expect(token.T_RPAREN))
	if p.at(token.T_LBRACE) {
		parts = append(parts, p.parseStatementList())
	} else if p.at(token.T_COLON) {
		// Alternate declare syntax: declare(...): ... enddeclare;
		parts = append(parts, p.bump())
		var body []*GreenNode
		for !p.at(token.T_EOF) {
			if p.at(token.T_ENDDECLARE) {
				body = append(body, p.bump())
				if p.at(token.T_SEMICOLON) {
					body = append(body, p.bump())
				}
				break
			}
			body = append(body, p.bump())
		}
		if len(body) > 0 {
			parts = append(parts, p.intern.Node(KindTokenList, body...))
		}
	} else if p.at(token.T_SEMICOLON) {
		parts = append(parts, p.bump())
	}
	return p.intern.Node(KindDeclareStmt, parts...)
}

func (p *Parser) parseGlobalStmt() *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.expect(token.T_GLOBAL))
	for {
		if p.at(token.T_VARIABLE) {
			parts = append(parts, p.bump())
		}
		if p.at(token.T_COMMA) {
			parts = append(parts, p.bump())
			continue
		}
		break
	}
	if p.at(token.T_SEMICOLON) {
		parts = append(parts, p.bump())
	}
	return p.intern.Node(KindGlobalStmt, parts...)
}

func (p *Parser) parseStaticVarStmt() *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.expect(token.T_STATIC))
	for {
		if p.at(token.T_VARIABLE) {
			parts = append(parts, p.bump())
		}
		if p.at(token.T_ASSIGN) {
			parts = append(parts, p.bump())
			if expr := p.parseExpression(); expr != nil {
				parts = append(parts, expr)
			} else {
				parts = append(parts, p.parseUntilCommaOrSemi())
			}
		}
		if p.at(token.T_COMMA) {
			parts = append(parts, p.bump())
			continue
		}
		break
	}
	if p.at(token.T_SEMICOLON) {
		parts = append(parts, p.bump())
	}
	return p.intern.Node(KindStaticVarStmt, parts...)
}

func (p *Parser) parseEchoStmt() *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.expect(token.T_ECHO))
	parts = append(parts, p.parseExprListComma()...)
	if p.at(token.T_SEMICOLON) {
		parts = append(parts, p.bump())
	}
	return p.intern.Node(KindEchoStmt, parts...)
}

// parseShortEchoStmt parses <?= expr [, expr...] [;] as KindEchoStmt.
func (p *Parser) parseShortEchoStmt() *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.expect(token.T_OPEN_TAG_WITH_ECHO))
	parts = append(parts, p.parseExprListComma()...)
	if p.at(token.T_SEMICOLON) {
		parts = append(parts, p.bump())
	}
	return p.intern.Node(KindEchoStmt, parts...)
}

func (p *Parser) parseReturnStmt() *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.expect(token.T_RETURN))
	if !p.at(token.T_SEMICOLON) && !p.at(token.T_EOF) {
		if expr := p.parseExpression(); expr != nil {
			parts = append(parts, expr)
		}
	}
	if p.at(token.T_SEMICOLON) {
		parts = append(parts, p.bump())
	}
	return p.intern.Node(KindReturnStmt, parts...)
}

func (p *Parser) parseExpressionStmt() *GreenNode {
	var parts []*GreenNode
	if expr := p.parseExpression(); expr != nil {
		parts = append(parts, expr)
	} else {
		parts = append(parts, p.parseUntilStmtEnd())
	}
	if p.at(token.T_SEMICOLON) {
		parts = append(parts, p.bump())
	}
	return p.intern.Node(KindExpressionStmt, parts...)
}

func (p *Parser) parseBreakStmt() *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.expect(token.T_BREAK))
	if !p.at(token.T_SEMICOLON) && !p.at(token.T_EOF) {
		if expr := p.parseExpression(); expr != nil {
			parts = append(parts, expr)
		}
	}
	if p.at(token.T_SEMICOLON) {
		parts = append(parts, p.bump())
	}
	return p.intern.Node(KindBreakStmt, parts...)
}

func (p *Parser) parseContinueStmt() *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.expect(token.T_CONTINUE))
	if !p.at(token.T_SEMICOLON) && !p.at(token.T_EOF) {
		if expr := p.parseExpression(); expr != nil {
			parts = append(parts, expr)
		}
	}
	if p.at(token.T_SEMICOLON) {
		parts = append(parts, p.bump())
	}
	return p.intern.Node(KindContinueStmt, parts...)
}

func (p *Parser) parseThrowStmt() *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.expect(token.T_THROW))
	if !p.at(token.T_SEMICOLON) && !p.at(token.T_EOF) {
		if expr := p.parseExpression(); expr != nil {
			parts = append(parts, expr)
		}
	}
	if p.at(token.T_SEMICOLON) {
		parts = append(parts, p.bump())
	}
	return p.intern.Node(KindThrowStmt, parts...)
}

func (p *Parser) parseUnsetStmt() *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.expect(token.T_UNSET))
	parts = p.appendUnsetParenArgs(parts)
	if p.at(token.T_SEMICOLON) {
		parts = append(parts, p.bump())
	}
	return p.intern.Node(KindUnsetStmt, parts...)
}

// parseControlBody parses a braced block, alternate colon body until stop, or a single statement.
func (p *Parser) parseControlBody(stop ...token.TokenType) []*GreenNode {
	if p.at(token.T_LBRACE) {
		return []*GreenNode{p.parseStatementList()}
	}
	if p.at(token.T_COLON) {
		var parts []*GreenNode
		parts = append(parts, p.bump())
		parts = append(parts, p.parseColonStatementList(stop...))
		return parts
	}
	if stmt := p.tryParseStructured(); stmt != nil {
		return []*GreenNode{stmt}
	}
	// Fallback: consume one statement-ish span as tokens until ';' or block enders.
	var parts []*GreenNode
	parts = append(parts, p.parseUntilStmtEnd())
	if p.at(token.T_SEMICOLON) {
		parts = append(parts, p.bump())
	}
	return []*GreenNode{p.intern.Node(KindExpressionStmt, parts...)}
}

func (p *Parser) atAny(types ...token.TokenType) bool {
	for _, tt := range types {
		if p.at(tt) {
			return true
		}
	}
	return false
}

func (p *Parser) parseColonStatementList(stop ...token.TokenType) *GreenNode {
	var parts []*GreenNode
	for !p.at(token.T_EOF) && !p.atAny(stop...) {
		if stmt := p.tryParseStructured(); stmt != nil {
			parts = append(parts, stmt)
			continue
		}
		parts = append(parts, p.bump())
	}
	return p.intern.Node(KindStatementList, parts...)
}

func (p *Parser) parseIfStmt() *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.expect(token.T_IF))
	parts = p.appendSimpleParenExpr(parts)
	alt := p.at(token.T_COLON)
	parts = append(parts, p.parseControlBody(token.T_ELSEIF, token.T_ELSE, token.T_ENDIF)...)
	for p.at(token.T_ELSEIF) {
		parts = append(parts, p.parseElseIfClause(alt))
	}
	if p.at(token.T_ELSE) {
		parts = append(parts, p.parseElseClause(alt))
	}
	if alt {
		if p.at(token.T_ENDIF) {
			parts = append(parts, p.bump())
		}
		if p.at(token.T_SEMICOLON) {
			parts = append(parts, p.bump())
		}
	}
	return p.intern.Node(KindIfStmt, parts...)
}

func (p *Parser) parseElseIfClause(alt bool) *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.expect(token.T_ELSEIF))
	parts = p.appendSimpleParenExpr(parts)
	if alt {
		parts = append(parts, p.parseControlBody(token.T_ELSEIF, token.T_ELSE, token.T_ENDIF)...)
	} else {
		parts = append(parts, p.parseControlBody()...)
	}
	return p.intern.Node(KindElseIfClause, parts...)
}

func (p *Parser) parseElseClause(alt bool) *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.expect(token.T_ELSE))
	if alt {
		parts = append(parts, p.parseControlBody(token.T_ENDIF)...)
	} else {
		parts = append(parts, p.parseControlBody()...)
	}
	return p.intern.Node(KindElseClause, parts...)
}

func (p *Parser) parseWhileStmt() *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.expect(token.T_WHILE))
	parts = p.appendSimpleParenExpr(parts)
	alt := p.at(token.T_COLON)
	parts = append(parts, p.parseControlBody(token.T_ENDWHILE)...)
	if alt {
		if p.at(token.T_ENDWHILE) {
			parts = append(parts, p.bump())
		}
		if p.at(token.T_SEMICOLON) {
			parts = append(parts, p.bump())
		}
	}
	return p.intern.Node(KindWhileStmt, parts...)
}

func (p *Parser) parseDoWhileStmt() *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.expect(token.T_DO))
	parts = append(parts, p.parseControlBody()...)
	parts = append(parts, p.expect(token.T_WHILE))
	parts = p.appendSimpleParenExpr(parts)
	if p.at(token.T_SEMICOLON) {
		parts = append(parts, p.bump())
	}
	return p.intern.Node(KindDoWhileStmt, parts...)
}

func (p *Parser) parseForStmt() *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.expect(token.T_FOR))
	parts = p.appendForParenHeader(parts)
	alt := p.at(token.T_COLON)
	parts = append(parts, p.parseControlBody(token.T_ENDFOR)...)
	if alt {
		if p.at(token.T_ENDFOR) {
			parts = append(parts, p.bump())
		}
		if p.at(token.T_SEMICOLON) {
			parts = append(parts, p.bump())
		}
	}
	return p.intern.Node(KindForStmt, parts...)
}

func (p *Parser) parseForeachStmt() *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.expect(token.T_FOREACH))
	parts = p.appendForeachParenHeader(parts)
	alt := p.at(token.T_COLON)
	parts = append(parts, p.parseControlBody(token.T_ENDFOREACH)...)
	if alt {
		if p.at(token.T_ENDFOREACH) {
			parts = append(parts, p.bump())
		}
		if p.at(token.T_SEMICOLON) {
			parts = append(parts, p.bump())
		}
	}
	return p.intern.Node(KindForeachStmt, parts...)
}

func (p *Parser) parseSwitchStmt() *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.expect(token.T_SWITCH))
	parts = p.appendSimpleParenExpr(parts)
	if p.at(token.T_LBRACE) {
		parts = append(parts, p.parseSwitchBlock(false))
	} else if p.at(token.T_COLON) {
		parts = append(parts, p.bump())
		parts = append(parts, p.parseSwitchBlock(true))
		if p.at(token.T_ENDSWITCH) {
			parts = append(parts, p.bump())
		}
		if p.at(token.T_SEMICOLON) {
			parts = append(parts, p.bump())
		}
	}
	return p.intern.Node(KindSwitchStmt, parts...)
}

func (p *Parser) parseSwitchBlock(alt bool) *GreenNode {
	var parts []*GreenNode
	if !alt {
		parts = append(parts, p.expect(token.T_LBRACE))
	}
	for !p.at(token.T_EOF) {
		if !alt && p.at(token.T_RBRACE) {
			break
		}
		if alt && p.at(token.T_ENDSWITCH) {
			break
		}
		if p.at(token.T_CASE) {
			parts = append(parts, p.parseCaseClause(alt))
			continue
		}
		if p.at(token.T_DEFAULT) {
			parts = append(parts, p.parseDefaultClause(alt))
			continue
		}
		if stmt := p.tryParseStructured(); stmt != nil {
			parts = append(parts, stmt)
			continue
		}
		parts = append(parts, p.bump())
	}
	if !alt && p.at(token.T_RBRACE) {
		parts = append(parts, p.bump())
	}
	return p.intern.Node(KindStatementList, parts...)
}

func (p *Parser) parseCaseClause(alt bool) *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.expect(token.T_CASE))
	if expr := p.parseExpression(); expr != nil {
		parts = append(parts, expr)
	} else {
		parts = append(parts, p.parseUntilCaseSep())
	}
	if p.at(token.T_COLON) || p.at(token.T_SEMICOLON) {
		parts = append(parts, p.bump())
	}
	parts = append(parts, p.parseCaseBody(alt)...)
	return p.intern.Node(KindCaseClause, parts...)
}

func (p *Parser) parseDefaultClause(alt bool) *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.expect(token.T_DEFAULT))
	if p.at(token.T_COLON) || p.at(token.T_SEMICOLON) {
		parts = append(parts, p.bump())
	}
	parts = append(parts, p.parseCaseBody(alt)...)
	return p.intern.Node(KindDefaultClause, parts...)
}

func (p *Parser) parseUntilCaseSep() *GreenNode {
	var parts []*GreenNode
	depth := 0
	for !p.at(token.T_EOF) {
		if depth == 0 && (p.at(token.T_COLON) || p.at(token.T_SEMICOLON)) {
			break
		}
		switch p.tok().Type {
		case token.T_LPAREN, token.T_LBRACKET, token.T_LBRACE:
			depth++
		case token.T_RPAREN, token.T_RBRACKET, token.T_RBRACE:
			if depth > 0 {
				depth--
			}
		}
		parts = append(parts, p.bump())
	}
	return p.intern.Node(KindTokenList, parts...)
}

func (p *Parser) parseCaseBody(alt bool) []*GreenNode {
	var parts []*GreenNode
	for !p.at(token.T_EOF) {
		if p.at(token.T_CASE) || p.at(token.T_DEFAULT) {
			break
		}
		if !alt && p.at(token.T_RBRACE) {
			break
		}
		if alt && p.at(token.T_ENDSWITCH) {
			break
		}
		if stmt := p.tryParseStructured(); stmt != nil {
			parts = append(parts, stmt)
			continue
		}
		parts = append(parts, p.bump())
	}
	if len(parts) == 0 {
		return nil
	}
	return []*GreenNode{p.intern.Node(KindStatementList, parts...)}
}

func (p *Parser) parseMatchExprStmt() *GreenNode {
	match := p.parseMatchExpr()
	if p.at(token.T_SEMICOLON) {
		return p.intern.Node(KindExpressionStmt, match, p.bump())
	}
	return p.intern.Node(KindExpressionStmt, match)
}

func (p *Parser) parseMatchExpr() *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.expect(token.T_MATCH))
	parts = p.appendSimpleParenExpr(parts)
	parts = append(parts, p.expect(token.T_LBRACE))
	for !p.at(token.T_RBRACE) && !p.at(token.T_EOF) {
		if p.at(token.T_COMMA) {
			parts = append(parts, p.bump())
			continue
		}
		parts = append(parts, p.parseMatchArm())
	}
	if p.at(token.T_RBRACE) {
		parts = append(parts, p.bump())
	}
	return p.intern.Node(KindMatchExpr, parts...)
}

func (p *Parser) parseMatchArm() *GreenNode {
	var parts []*GreenNode
	if p.at(token.T_DEFAULT) {
		parts = append(parts, p.bump())
	} else {
		parts = append(parts, p.parseExprListComma()...)
	}
	if p.at(token.T_DOUBLE_ARROW) {
		parts = append(parts, p.bump())
		if expr := p.parseExpression(); expr != nil {
			parts = append(parts, expr)
		} else {
			parts = append(parts, p.parseUntilMatchArmEnd())
		}
	}
	return p.intern.Node(KindMatchArm, parts...)
}

func (p *Parser) parseUntilMatchArmEnd() *GreenNode {
	var parts []*GreenNode
	depth := 0
	for !p.at(token.T_EOF) {
		if depth == 0 && (p.at(token.T_COMMA) || p.at(token.T_RBRACE)) {
			break
		}
		switch p.tok().Type {
		case token.T_LPAREN, token.T_LBRACKET, token.T_LBRACE:
			depth++
		case token.T_RPAREN, token.T_RBRACKET, token.T_RBRACE:
			if depth > 0 {
				depth--
			} else if p.at(token.T_RBRACE) {
				break
			}
		}
		parts = append(parts, p.bump())
	}
	return p.intern.Node(KindTokenList, parts...)
}

func (p *Parser) parseTryStmt() *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.expect(token.T_TRY))
	if p.at(token.T_LBRACE) {
		parts = append(parts, p.parseStatementList())
	}
	for p.at(token.T_CATCH) {
		parts = append(parts, p.parseCatchClause())
	}
	if p.at(token.T_FINALLY) {
		parts = append(parts, p.parseFinallyClause())
	}
	return p.intern.Node(KindTryStmt, parts...)
}

func (p *Parser) parseCatchClause() *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.expect(token.T_CATCH))
	parts = append(parts, p.expect(token.T_LPAREN))
	if p.isTypeStart() {
		parts = append(parts, p.parseType())
	} else {
		// Fallback token coverage for unusual catch types.
		var inner []*GreenNode
		for !p.at(token.T_EOF) && !p.at(token.T_VARIABLE) && !p.at(token.T_RPAREN) {
			inner = append(inner, p.bump())
		}
		if len(inner) > 0 {
			parts = append(parts, p.intern.Node(KindTokenList, inner...))
		}
	}
	if p.at(token.T_VARIABLE) {
		parts = append(parts, p.bump())
	}
	parts = append(parts, p.expect(token.T_RPAREN))
	if p.at(token.T_LBRACE) {
		parts = append(parts, p.parseStatementList())
	}
	return p.intern.Node(KindCatchClause, parts...)
}

func (p *Parser) parseFinallyClause() *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.expect(token.T_FINALLY))
	if p.at(token.T_LBRACE) {
		parts = append(parts, p.parseStatementList())
	}
	return p.intern.Node(KindFinallyClause, parts...)
}

func (p *Parser) parseEnumCase() *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.expect(token.T_CASE))
	if p.at(token.T_STRING) {
		parts = append(parts, p.intern.Node(KindUnqualifiedName, p.bump()))
	}
	if p.at(token.T_ASSIGN) {
		parts = append(parts, p.bump())
		if expr := p.parseExpression(); expr != nil {
			parts = append(parts, expr)
		} else {
			parts = append(parts, p.parseUntilStmtEnd())
		}
	}
	if p.at(token.T_SEMICOLON) {
		parts = append(parts, p.bump())
	}
	return p.intern.Node(KindEnumCase, parts...)
}

func (p *Parser) parseClassConstDecl() *GreenNode {
	var parts []*GreenNode
	if mods := p.parseModifierList(); mods != nil {
		parts = append(parts, mods)
	}
	parts = append(parts, p.expect(token.T_CONST))
	if p.isTypeStart() && !(p.at(token.T_STRING) && p.i+1 < len(p.tokens) && p.tokens[p.i+1].Type == token.T_ASSIGN) {
		// Speculative typed const: const Type NAME = ...
		save := p.i
		typ := p.parseType()
		if p.at(token.T_STRING) {
			parts = append(parts, typ)
		} else {
			p.i = save
		}
	}
	for {
		if p.at(token.T_STRING) {
			parts = append(parts, p.intern.Node(KindUnqualifiedName, p.bump()))
		}
		if p.at(token.T_ASSIGN) {
			parts = append(parts, p.bump())
			if expr := p.parseExpression(); expr != nil {
				parts = append(parts, expr)
			} else {
				parts = append(parts, p.parseUntilCommaOrSemi())
			}
		}
		if p.at(token.T_COMMA) {
			parts = append(parts, p.bump())
			continue
		}
		break
	}
	if p.at(token.T_SEMICOLON) {
		parts = append(parts, p.bump())
	}
	return p.intern.Node(KindClassConstDecl, parts...)
}

func (p *Parser) parsePropertyDecl() *GreenNode {
	var parts []*GreenNode
	if mods := p.parseModifierList(); mods != nil {
		parts = append(parts, mods)
	}
	if p.isTypeStart() && !p.at(token.T_VARIABLE) {
		parts = append(parts, p.parseType())
	}
	for {
		if p.at(token.T_VARIABLE) {
			parts = append(parts, p.bump())
		}
		if p.at(token.T_ASSIGN) {
			parts = append(parts, p.bump())
			if expr := p.parseExpression(); expr != nil {
				parts = append(parts, expr)
			} else {
				parts = append(parts, p.parseUntilCommaOrSemi())
			}
		}
		// PHP 8.4 property hooks: `$prop { get => …; set { … } }` terminates the
		// property (no trailing `;`, and no further comma-separated names).
		if p.at(token.T_LBRACE) {
			parts = append(parts, p.parsePropertyHookList())
			return p.intern.Node(KindPropertyDecl, parts...)
		}
		if p.at(token.T_COMMA) {
			parts = append(parts, p.bump())
			continue
		}
		break
	}
	if p.at(token.T_SEMICOLON) {
		parts = append(parts, p.bump())
	}
	return p.intern.Node(KindPropertyDecl, parts...)
}

// parsePropertyHookList covers `{` hook… `}` attached to a property.
func (p *Parser) parsePropertyHookList() *GreenNode {
	return p.parseBraceDelimitedList(KindPropertyHookList, p.parsePropertyHook)
}

// parsePropertyHook covers one get/set hook:
//
//	[&] name [( params )] ( ; | => expr ; | { stmts } )
func (p *Parser) parsePropertyHook() *GreenNode {
	i := p.i
	if i < len(p.tokens) && p.tokens[i].Type == token.T_AMPERSAND {
		i++
	}
	if i >= len(p.tokens) || p.tokens[i].Type != token.T_STRING {
		return nil
	}
	var parts []*GreenNode
	if p.at(token.T_AMPERSAND) {
		parts = append(parts, p.bump())
	}
	parts = append(parts, p.bump()) // get / set
	if p.at(token.T_LPAREN) {
		parts = append(parts, p.expect(token.T_LPAREN))
		parts = append(parts, p.parseParamList())
		parts = append(parts, p.expect(token.T_RPAREN))
	}
	switch {
	case p.at(token.T_SEMICOLON):
		parts = append(parts, p.bump())
	case p.at(token.T_DOUBLE_ARROW):
		parts = append(parts, p.bump())
		if expr := p.parseExpression(); expr != nil {
			parts = append(parts, expr)
		}
		if p.at(token.T_SEMICOLON) {
			parts = append(parts, p.bump())
		}
	case p.at(token.T_LBRACE):
		parts = append(parts, p.parseStatementList())
	default:
		// Leave remaining tokens to the hook-list fallback bump so identity holds.
	}
	return p.intern.Node(KindPropertyHook, parts...)
}

func (p *Parser) parseUntilCommaOrSemi() *GreenNode {
	var parts []*GreenNode
	depth := 0
	for !p.at(token.T_EOF) {
		if depth == 0 && (p.at(token.T_COMMA) || p.at(token.T_SEMICOLON)) {
			break
		}
		switch p.tok().Type {
		case token.T_LPAREN, token.T_LBRACKET, token.T_LBRACE:
			depth++
		case token.T_RPAREN, token.T_RBRACKET, token.T_RBRACE:
			if depth > 0 {
				depth--
			}
		}
		parts = append(parts, p.bump())
	}
	return p.intern.Node(KindTokenList, parts...)
}

func (p *Parser) parseUntilStmtEnd() *GreenNode {
	return p.parseUntilCommaOrSemi()
}

func (p *Parser) parseParamList() *GreenNode {
	var parts []*GreenNode
	for !p.at(token.T_RPAREN) && !p.at(token.T_EOF) {
		start := p.i
		if p.at(token.T_COMMA) {
			parts = append(parts, p.bump())
			continue
		}
		parts = append(parts, p.parseParam())
		if p.i == start {
			// No progress (e.g. unexpected keyword) — bump to avoid infinite loops.
			parts = append(parts, p.bump())
		}
	}
	return p.intern.Node(KindParamList, parts...)
}

func (p *Parser) parseParam() *GreenNode {
	var parts []*GreenNode
	if p.at(token.T_ATTRIBUTE) {
		parts = append(parts, p.parseAttributeList())
	}
	if mods := p.parseModifierList(); mods != nil {
		parts = append(parts, mods)
	}
	if p.isTypeStart() {
		parts = append(parts, p.parseType())
	}
	if p.at(token.T_AMPERSAND) {
		parts = append(parts, p.bump())
	}
	if p.at(token.T_ELLIPSIS) {
		parts = append(parts, p.bump())
	}
	if p.at(token.T_VARIABLE) {
		parts = append(parts, p.bump())
	}
	if p.at(token.T_ASSIGN) {
		parts = append(parts, p.bump())
		if expr := p.parseExpression(); expr != nil {
			parts = append(parts, expr)
		} else {
			parts = append(parts, p.parseUntilParamEnd())
		}
	}
	return p.intern.Node(KindParam, parts...)
}

func (p *Parser) isTypeStart() bool {
	if p.at(token.T_QUESTION) || p.at(token.T_LPAREN) || p.at(token.T_NS_SEPARATOR) ||
		p.at(token.T_ARRAY) || p.at(token.T_CALLABLE) || p.at(token.T_MIXED) ||
		p.at(token.T_STATIC) || p.at(token.T_SELF) || p.at(token.T_PARENT) {
		return true
	}
	return p.isPrimitiveType() || p.at(token.T_STRING)
}

func (p *Parser) parseUntilParamEnd() *GreenNode {
	var parts []*GreenNode
	depth := 0
	for !p.at(token.T_EOF) {
		if depth == 0 && (p.at(token.T_COMMA) || p.at(token.T_RPAREN)) {
			break
		}
		if p.at(token.T_LPAREN) {
			depth++
		} else if p.at(token.T_RPAREN) {
			depth--
		}
		parts = append(parts, p.bump())
	}
	return p.intern.Node(KindTokenList, parts...)
}

func (p *Parser) parseBalancedBlock() *GreenNode {
	var parts []*GreenNode
	parts = append(parts, p.expect(token.T_LBRACE))
	depth := 1
	for !p.at(token.T_EOF) && depth > 0 {
		if p.at(token.T_LBRACE) {
			depth++
		} else if p.at(token.T_RBRACE) {
			depth--
		}
		parts = append(parts, p.bump())
	}
	return p.intern.Node(KindTokenList, parts...)
}
