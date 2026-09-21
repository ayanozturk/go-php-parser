package syntax

import (
	"strconv"
	"unsafe"

	"github.com/ayanozturk/go-php-parser/token"
)

// Kind identifies a green/red syntax node.
type Kind uint16

const (
	KindError Kind = iota
	KindToken
	KindMissing
	KindFile
	KindTokenList // flat list of tokens covering a span (tokens-only CST)

	// Names
	KindName
	KindUnqualifiedName
	KindQualifiedName
	KindFullyQualifiedName
	KindRelativeName
	KindNameList

	// Types
	KindNamedType
	KindNullableType
	KindUnionType
	KindIntersectionType
	KindParenthesizedType
	KindPrimitiveType
	KindCallableType

	// Attributes
	KindAttributeList
	KindAttributeGroup
	KindAttribute
	KindArgList
	KindArg
	KindNamedArg

	// Strings / heredoc
	KindStringLiteral
	KindHeredoc
	KindNowdoc
	KindStringPart
	KindVariablePart
	KindEncapsulatedExpr

	// Modifiers / declarations / members
	KindModifierList
	KindClassDecl
	KindInterfaceDecl
	KindTraitDecl
	KindEnumDecl
	KindFunctionDecl
	KindMethodDecl
	KindPropertyDecl
	KindClassConstDecl
	KindEnumCase
	KindParam
	KindParamList
	KindMemberList
	KindExtendsClause
	KindImplementsClause
	KindUseTraitClause
	KindNamespaceDecl
	KindUseDecl
	KindUseClause
	KindUseGroup
	KindCallableParam
	KindCallableParamList
	KindStatementList
	KindEmptyStmt
	KindConstDecl
	KindDeclareStmt
	KindGlobalStmt
	KindStaticVarStmt
	KindEchoStmt
	KindReturnStmt
	KindExpressionStmt
	KindIfStmt
	KindElseIfClause
	KindElseClause
	KindWhileStmt
	KindDoWhileStmt
	KindForStmt
	KindForeachStmt
	KindSwitchStmt
	KindCaseClause
	KindDefaultClause
	KindMatchExpr
	KindMatchArm
	KindTryStmt
	KindCatchClause
	KindFinallyClause
	KindBreakStmt
	KindContinueStmt
	KindThrowStmt
	KindUnsetStmt
	KindGotoStmt
	KindLabelStmt
	KindTraitAdaptationList
	KindTraitAdaptation

	// Expressions
	KindVariableExpr
	KindLiteralExpr
	KindBinaryExpr
	KindUnaryExpr
	KindAssignExpr
	KindTernaryExpr
	KindCallExpr
	KindMemberAccessExpr
	KindNullsafeMemberAccessExpr
	KindArrayAccessExpr
	KindStaticMemberAccessExpr
	KindNewExpr
	KindCloneExpr
	KindCastExpr
	KindParenExpr
	KindArrayExpr
	KindArrayElement
	KindListExpr
	KindPrintExpr
	KindIncludeExpr
	KindThrowExpr
	KindYieldExpr
	KindVariableVariableExpr
	KindFirstClassCallableExpr
	KindClosureExpr
	KindArrowFunctionExpr
	KindAnonymousClass
	KindClosureUseClause

	// Property hooks (PHP 8.4)
	KindPropertyHookList
	KindPropertyHook
)

var kindNames = [...]string{
	KindError:                      "Error",
	KindToken:                      "Token",
	KindMissing:                    "Missing",
	KindFile:                       "File",
	KindTokenList:                  "TokenList",
	KindName:                       "Name",
	KindUnqualifiedName:            "UnqualifiedName",
	KindQualifiedName:              "QualifiedName",
	KindFullyQualifiedName:         "FullyQualifiedName",
	KindRelativeName:               "RelativeName",
	KindNameList:                   "NameList",
	KindNamedType:                  "NamedType",
	KindNullableType:               "NullableType",
	KindUnionType:                  "UnionType",
	KindIntersectionType:           "IntersectionType",
	KindParenthesizedType:          "ParenthesizedType",
	KindPrimitiveType:              "PrimitiveType",
	KindCallableType:               "CallableType",
	KindAttributeList:              "AttributeList",
	KindAttributeGroup:             "AttributeGroup",
	KindAttribute:                  "Attribute",
	KindArgList:                    "ArgList",
	KindArg:                        "Arg",
	KindNamedArg:                   "NamedArg",
	KindStringLiteral:              "StringLiteral",
	KindHeredoc:                    "Heredoc",
	KindNowdoc:                     "Nowdoc",
	KindStringPart:                 "StringPart",
	KindVariablePart:               "VariablePart",
	KindEncapsulatedExpr:           "EncapsulatedExpr",
	KindModifierList:                "ModifierList",
	KindClassDecl:                  "ClassDecl",
	KindInterfaceDecl:              "InterfaceDecl",
	KindTraitDecl:                  "TraitDecl",
	KindEnumDecl:                   "EnumDecl",
	KindFunctionDecl:               "FunctionDecl",
	KindMethodDecl:                 "MethodDecl",
	KindPropertyDecl:               "PropertyDecl",
	KindClassConstDecl:             "ClassConstDecl",
	KindEnumCase:                   "EnumCase",
	KindParam:                      "Param",
	KindParamList:                  "ParamList",
	KindMemberList:                 "MemberList",
	KindExtendsClause:              "ExtendsClause",
	KindImplementsClause:           "ImplementsClause",
	KindUseTraitClause:             "UseTraitClause",
	KindNamespaceDecl:              "NamespaceDecl",
	KindUseDecl:                    "UseDecl",
	KindUseClause:                  "UseClause",
	KindUseGroup:                   "UseGroup",
	KindCallableParam:              "CallableParam",
	KindCallableParamList:          "CallableParamList",
	KindStatementList:              "StatementList",
	KindEmptyStmt:                  "EmptyStmt",
	KindConstDecl:                  "ConstDecl",
	KindDeclareStmt:                "DeclareStmt",
	KindGlobalStmt:                 "GlobalStmt",
	KindStaticVarStmt:              "StaticVarStmt",
	KindEchoStmt:                   "EchoStmt",
	KindReturnStmt:                 "ReturnStmt",
	KindExpressionStmt:             "ExpressionStmt",
	KindIfStmt:                     "IfStmt",
	KindElseIfClause:               "ElseIfClause",
	KindElseClause:                 "ElseClause",
	KindWhileStmt:                  "WhileStmt",
	KindDoWhileStmt:                "DoWhileStmt",
	KindForStmt:                    "ForStmt",
	KindForeachStmt:                "ForeachStmt",
	KindSwitchStmt:                 "SwitchStmt",
	KindCaseClause:                 "CaseClause",
	KindDefaultClause:              "DefaultClause",
	KindMatchExpr:                  "MatchExpr",
	KindMatchArm:                   "MatchArm",
	KindTryStmt:                    "TryStmt",
	KindCatchClause:                "CatchClause",
	KindFinallyClause:              "FinallyClause",
	KindBreakStmt:                  "BreakStmt",
	KindContinueStmt:               "ContinueStmt",
	KindThrowStmt:                  "ThrowStmt",
	KindUnsetStmt:                  "UnsetStmt",
	KindGotoStmt:                   "GotoStmt",
	KindLabelStmt:                  "LabelStmt",
	KindTraitAdaptationList:        "TraitAdaptationList",
	KindTraitAdaptation:            "TraitAdaptation",
	KindVariableExpr:               "VariableExpr",
	KindLiteralExpr:                "LiteralExpr",
	KindBinaryExpr:                 "BinaryExpr",
	KindUnaryExpr:                  "UnaryExpr",
	KindAssignExpr:                 "AssignExpr",
	KindTernaryExpr:                "TernaryExpr",
	KindCallExpr:                   "CallExpr",
	KindMemberAccessExpr:           "MemberAccessExpr",
	KindNullsafeMemberAccessExpr:   "NullsafeMemberAccessExpr",
	KindArrayAccessExpr:            "ArrayAccessExpr",
	KindStaticMemberAccessExpr:     "StaticMemberAccessExpr",
	KindNewExpr:                    "NewExpr",
	KindCloneExpr:                  "CloneExpr",
	KindCastExpr:                   "CastExpr",
	KindParenExpr:                  "ParenExpr",
	KindArrayExpr:                  "ArrayExpr",
	KindArrayElement:               "ArrayElement",
	KindListExpr:                   "ListExpr",
	KindPrintExpr:                  "PrintExpr",
	KindIncludeExpr:                "IncludeExpr",
	KindThrowExpr:                  "ThrowExpr",
	KindYieldExpr:                  "YieldExpr",
	KindVariableVariableExpr:       "VariableVariableExpr",
	KindFirstClassCallableExpr:     "FirstClassCallableExpr",
	KindClosureExpr:                "ClosureExpr",
	KindArrowFunctionExpr:          "ArrowFunctionExpr",
	KindAnonymousClass:             "AnonymousClass",
	KindClosureUseClause:           "ClosureUseClause",
	KindPropertyHookList:           "PropertyHookList",
	KindPropertyHook:               "PropertyHook",
}

func (k Kind) String() string {
	if int(k) < len(kindNames) && kindNames[k] != "" {
		return kindNames[k]
	}
	return "Kind(?)"
}

// Span is a half-open byte range into the source buffer.
type Span struct {
	Start int
	End   int
}

func (s Span) Len() int { return s.End - s.Start }

// GreenNode is an immutable, position-independent syntax node.
// Absolute source offsets live on red views only (R1).
// Fields are unexported; construct only via Interner.
type GreenNode struct {
	kind     Kind
	width    int
	token    *token.Token // KindToken / KindMissing; Pos/End cleared
	children []*GreenNode
	// Byte offsets relative to this green's start: content begins after leading
	// trivia on the first token descendant and ends before trailing trivia on
	// token greens; composites span the full width (endRel == width).
	contentStartRel int
	contentEndRel   int
}

// Kind returns the green node kind.
func (g *GreenNode) Kind() Kind {
	if g == nil {
		return KindError
	}
	return g.kind
}

// Width returns the byte width covered by this green (including trivia on tokens).
func (g *GreenNode) Width() int {
	if g == nil {
		return 0
	}
	return g.width
}

// Children returns the immutable child list (do not mutate the slice).
func (g *GreenNode) Children() []*GreenNode {
	if g == nil {
		return nil
	}
	return g.children
}

// TokenType returns the token type for token/missing greens.
func (g *GreenNode) TokenType() token.TokenType {
	if g == nil || g.token == nil {
		return token.T_EOF
	}
	return g.token.Type
}

// Token returns a copy of the position-independent token, if any.
func (g *GreenNode) Token() (token.Token, bool) {
	if g == nil || g.token == nil {
		return token.Token{}, false
	}
	return *g.token, true
}

func (g *GreenNode) IsToken() bool {
	return g != nil && (g.kind == KindToken || g.kind == KindMissing) && g.token != nil
}

// Interner deduplicates identical position-independent green subtrees.
// src is the owned file buffer used to fingerprint token text when Literal is empty.
type Interner struct {
	src    []byte
	nodes  map[string]*GreenNode
	keyBuf []byte // reused for Token/Node map keys; lookup via string(buf) avoids hit alloc
}

func NewInterner(src []byte) *Interner {
	return &Interner{src: src, nodes: make(map[string]*GreenNode)}
}

func (in *Interner) Token(tok token.Token) *GreenNode {
	tok = in.materializeLiterals(tok)
	w := tokenWidth(tok)
	in.buildTokenKey(tok, w)
	if n, ok := in.nodes[string(in.keyBuf)]; ok {
		return n
	}
	key := string(in.keyBuf)
	t := positionIndependentToken(tok)
	startRel, endRel := greenTokenContentBounds(w, t)
	n := &GreenNode{kind: KindToken, width: w, token: &t, contentStartRel: startRel, contentEndRel: endRel}
	in.nodes[key] = n
	return n
}

func (in *Interner) Missing(tok token.Token) *GreenNode {
	tok = in.materializeLiterals(tok)
	t := positionIndependentToken(tok)
	// Missing tokens are not shared — each recovery site is distinct.
	startRel, endRel := greenTokenContentBounds(0, t)
	return &GreenNode{kind: KindMissing, width: 0, token: &t, contentStartRel: startRel, contentEndRel: endRel}
}

func (in *Interner) Node(kind Kind, children ...*GreenNode) *GreenNode {
	w := 0
	ks := kind.String()
	b := in.keyBuf[:0]
	need := 2 + len(ks) + len(children)*18
	if cap(b) < need {
		b = make([]byte, 0, need)
	}
	b = append(b, 'n', ':')
	b = append(b, ks...)
	for _, c := range children {
		b = append(b, ':')
		if c == nil {
			b = append(b, '0')
			continue
		}
		w += c.width
		b = strconv.AppendUint(b, uint64(uintptr(unsafe.Pointer(c))), 16)
	}
	in.keyBuf = b
	if n, ok := in.nodes[string(b)]; ok {
		return n
	}
	key := string(b)
	ch := append([]*GreenNode(nil), children...)
	startRel := greenCompositeContentStartRel(ch)
	n := &GreenNode{kind: kind, width: w, children: ch, contentStartRel: startRel, contentEndRel: w}
	in.nodes[key] = n
	return n
}

func greenTokenContentBounds(width int, tok token.Token) (startRel, endRel int) {
	startRel = leadingTriviaWidth(tok)
	endRel = width - trailingTriviaWidth(tok)
	return startRel, endRel
}

func greenCompositeContentStartRel(children []*GreenNode) int {
	childOff := 0
	for _, c := range children {
		if c == nil {
			continue
		}
		return childOff + c.contentStartRel
	}
	return 0
}

// Len reports how many distinct greens are currently interned (tests/metrics).
func (in *Interner) Len() int {
	if in == nil {
		return 0
	}
	return len(in.nodes)
}

func (in *Interner) materializeLiterals(tok token.Token) token.Token {
	if in == nil || len(in.src) == 0 {
		return tok
	}
	if tok.Literal == "" && tok.End.Offset > tok.Pos.Offset &&
		tok.Pos.Offset >= 0 && tok.End.Offset <= len(in.src) {
		tok.Literal = string(in.src[tok.Pos.Offset:tok.End.Offset])
	}
	if len(tok.LeadingTrivia) > 0 {
		tok.LeadingTrivia = materializeTriviaLiterals(in.src, tok.LeadingTrivia)
	}
	if len(tok.TrailingTrivia) > 0 {
		tok.TrailingTrivia = materializeTriviaLiterals(in.src, tok.TrailingTrivia)
	}
	return tok
}

func materializeTriviaLiterals(src []byte, triv []token.Token) []token.Token {
	// Lexer paths already fill trivia Literal. Skip the slice copy when there
	// is nothing to materialize — hot Interner.Token paid ~8% WP alloc here.
	needCopy := false
	for i := range triv {
		if triv[i].Literal == "" && triv[i].End.Offset > triv[i].Pos.Offset &&
			triv[i].Pos.Offset >= 0 && triv[i].End.Offset <= len(src) {
			needCopy = true
			break
		}
	}
	if !needCopy {
		return triv
	}
	out := make([]token.Token, len(triv))
	copy(out, triv)
	for i := range out {
		if out[i].Literal == "" && out[i].End.Offset > out[i].Pos.Offset &&
			out[i].Pos.Offset >= 0 && out[i].End.Offset <= len(src) {
			out[i].Literal = string(src[out[i].Pos.Offset:out[i].End.Offset])
		}
	}
	return out
}

func positionIndependentToken(tok token.Token) token.Token {
	t := tok
	t.Pos = token.Position{}
	t.End = token.Position{}
	t.LeadingTrivia = stripTriviaPositions(tok.LeadingTrivia)
	t.TrailingTrivia = stripTriviaPositions(tok.TrailingTrivia)
	return t
}

func stripTriviaPositions(triv []token.Token) []token.Token {
	if len(triv) == 0 {
		return nil
	}
	out := make([]token.Token, len(triv))
	for i, tr := range triv {
		out[i] = tr
		out[i].Pos = token.Position{}
		out[i].End = token.Position{}
		// Preserve Literal so identity/fingerprint and Text() fallback stay exact
		// when offsets are cleared.
		if out[i].Literal == "" && tr.Width() > 0 {
			// Width-only trivia without literal cannot be reprinted from green alone;
			// callers must print via red absolute offsets into the file source.
		}
		out[i].LeadingTrivia = nil
		out[i].TrailingTrivia = nil
	}
	return out
}

func (in *Interner) buildTokenKey(tok token.Token, w int) {
	ts := tok.Type.String()
	b := in.keyBuf[:0]
	// Heuristic: type name + width + trivia literals + token literal.
	need := 8 + len(ts) + len(tok.Literal) + len(tok.LeadingTrivia)*24 + len(tok.TrailingTrivia)*24
	if cap(b) < need {
		b = make([]byte, 0, need)
	}
	b = append(b, 't', ':')
	b = append(b, ts...)
	b = append(b, ':')
	b = strconv.AppendInt(b, int64(w), 10)
	b = append(b, ':', 'L')
	b = appendTriviaKeyBytes(b, tok.LeadingTrivia)
	b = append(b, ':', 'T')
	b = appendTriviaKeyBytes(b, tok.TrailingTrivia)
	b = append(b, ':')
	if lit := tok.Literal; lit != "" {
		b = append(b, lit...)
	} else if tok.End.Offset > tok.Pos.Offset {
		b = strconv.AppendInt(b, int64(tok.Width()), 10)
	}
	in.keyBuf = b
}

func appendTriviaKeyBytes(b []byte, triv []token.Token) []byte {
	for _, tr := range triv {
		b = append(b, ',')
		b = append(b, tr.Type.String()...)
		b = append(b, '/')
		b = strconv.AppendInt(b, int64(tokenWidth(tr)), 10)
		b = append(b, '/')
		if tr.Literal != "" {
			b = append(b, tr.Literal...)
		} else {
			b = strconv.AppendInt(b, int64(tr.Width()), 10)
		}
	}
	return b
}

func tokenWidth(tok token.Token) int {
	w := 0
	for _, tr := range tok.LeadingTrivia {
		w += tr.Width()
		if tr.Width() == 0 {
			w += len(tr.Literal)
		}
	}
	w += tok.Width()
	if tok.Width() == 0 && tok.Type != token.T_EOF {
		w += len(tok.Literal)
	}
	for _, tr := range tok.TrailingTrivia {
		w += tr.Width()
		if tr.Width() == 0 {
			w += len(tr.Literal)
		}
	}
	return w
}
