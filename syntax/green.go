package syntax

import "github.com/ayanozturk/go-php-parser/token"

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
	KindTraitAdaptationList
	KindTraitAdaptation
)

var kindNames = [...]string{
	KindError:              "Error",
	KindToken:              "Token",
	KindMissing:            "Missing",
	KindFile:               "File",
	KindTokenList:          "TokenList",
	KindName:               "Name",
	KindUnqualifiedName:    "UnqualifiedName",
	KindQualifiedName:      "QualifiedName",
	KindFullyQualifiedName: "FullyQualifiedName",
	KindRelativeName:       "RelativeName",
	KindNameList:           "NameList",
	KindNamedType:          "NamedType",
	KindNullableType:       "NullableType",
	KindUnionType:          "UnionType",
	KindIntersectionType:   "IntersectionType",
	KindParenthesizedType:  "ParenthesizedType",
	KindPrimitiveType:      "PrimitiveType",
	KindCallableType:       "CallableType",
	KindAttributeList:      "AttributeList",
	KindAttributeGroup:     "AttributeGroup",
	KindAttribute:          "Attribute",
	KindArgList:            "ArgList",
	KindArg:                "Arg",
	KindNamedArg:           "NamedArg",
	KindStringLiteral:      "StringLiteral",
	KindHeredoc:            "Heredoc",
	KindNowdoc:             "Nowdoc",
	KindStringPart:         "StringPart",
	KindVariablePart:       "VariablePart",
	KindEncapsulatedExpr:   "EncapsulatedExpr",
	KindModifierList:       "ModifierList",
	KindClassDecl:          "ClassDecl",
	KindInterfaceDecl:      "InterfaceDecl",
	KindTraitDecl:          "TraitDecl",
	KindEnumDecl:           "EnumDecl",
	KindFunctionDecl:       "FunctionDecl",
	KindMethodDecl:         "MethodDecl",
	KindPropertyDecl:       "PropertyDecl",
	KindClassConstDecl:     "ClassConstDecl",
	KindEnumCase:           "EnumCase",
	KindParam:              "Param",
	KindParamList:          "ParamList",
	KindMemberList:         "MemberList",
	KindExtendsClause:      "ExtendsClause",
	KindImplementsClause:   "ImplementsClause",
	KindUseTraitClause:     "UseTraitClause",
	KindNamespaceDecl:      "NamespaceDecl",
	KindUseDecl:            "UseDecl",
	KindUseClause:          "UseClause",
	KindUseGroup:           "UseGroup",
	KindCallableParam:      "CallableParam",
	KindCallableParamList:  "CallableParamList",
	KindStatementList:      "StatementList",
	KindEmptyStmt:          "EmptyStmt",
	KindConstDecl:          "ConstDecl",
	KindDeclareStmt:        "DeclareStmt",
	KindGlobalStmt:         "GlobalStmt",
	KindStaticVarStmt:      "StaticVarStmt",
	KindEchoStmt:           "EchoStmt",
	KindReturnStmt:         "ReturnStmt",
	KindExpressionStmt:     "ExpressionStmt",
	KindTraitAdaptationList: "TraitAdaptationList",
	KindTraitAdaptation:    "TraitAdaptation",
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

// GreenNode is an immutable syntax node. Children are tokens or nested greens.
// Width covers every byte of the construct including trivia on tokens.
type GreenNode struct {
	Kind     Kind
	Width    int
	Token    *token.Token // set when KindToken / KindMissing
	Children []*GreenNode
}

func (g *GreenNode) IsToken() bool {
	return g != nil && (g.Kind == KindToken || g.Kind == KindMissing) && g.Token != nil
}

// Interner deduplicates identical green subtrees (simple map keyed by structure).
type Interner struct {
	nodes map[string]*GreenNode
}

func NewInterner() *Interner {
	return &Interner{nodes: make(map[string]*GreenNode)}
}

func (in *Interner) Token(tok token.Token) *GreenNode {
	w := tokenWidth(tok)
	key := "t:" + tok.Type.String() + ":" + itoa(tok.Pos.Offset) + ":" + itoa(w)
	if n, ok := in.nodes[key]; ok {
		return n
	}
	t := tok
	n := &GreenNode{Kind: KindToken, Width: w, Token: &t}
	in.nodes[key] = n
	return n
}

func (in *Interner) Missing(tok token.Token) *GreenNode {
	t := tok
	n := &GreenNode{Kind: KindMissing, Width: 0, Token: &t}
	return n
}

func (in *Interner) Node(kind Kind, children ...*GreenNode) *GreenNode {
	w := 0
	for _, c := range children {
		if c != nil {
			w += c.Width
		}
	}
	n := &GreenNode{Kind: kind, Width: w, Children: children}
	return n
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

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [20]byte
	pos := len(b)
	neg := i < 0
	if neg {
		i = -i
	}
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		b[pos] = '-'
	}
	return string(b[pos:])
}
