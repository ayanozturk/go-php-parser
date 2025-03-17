package token

type TokenType string

const (
	T_OPEN_TAG        TokenType = "T_OPEN_TAG"
	T_VARIABLE        TokenType = "T_VARIABLE"
	T_FUNCTION        TokenType = "T_FUNCTION"
	T_IDENTIFIER      TokenType = "T_IDENTIFIER"
	T_LPAREN          TokenType = "T_LPAREN"
	T_RPAREN          TokenType = "T_RPAREN"
	T_LBRACE          TokenType = "T_LBRACE"
	T_RBRACE          TokenType = "T_RBRACE"
	T_SEMICOLON       TokenType = "T_SEMICOLON"
	T_ASSIGN          TokenType = "T_ASSIGN"
	T_IS_EQUAL        TokenType = "T_IS_EQUAL"
	T_CONSTANT_STRING TokenType = "T_CONSTANT_STRING"
	T_AMPERSAND       TokenType = "T_AMPERSAND"
	T_ELLIPSIS        TokenType = "T_ELLIPSIS"
	T_COMMA           TokenType = "T_COMMA"
	T_ARRAY           TokenType = "T_ARRAY"
	T_STRING          TokenType = "T_STRING"
	T_CALLABLE        TokenType = "T_CALLABLE"
	T_EOF             TokenType = "T_EOF"
)

type Position struct {
	Line   int
	Column int
	Offset int
}

type Token struct {
	Type    TokenType
	Literal string
	Pos     Position
}
