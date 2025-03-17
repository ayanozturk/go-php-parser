package token

type TokenType string

const (
	T_OPEN_TAG   TokenType = "T_OPEN_TAG"
	T_VARIABLE   TokenType = "T_VARIABLE"
	T_FUNCTION   TokenType = "T_FUNCTION"
	T_IDENTIFIER TokenType = "T_IDENTIFIER"
	T_LPAREN     TokenType = "T_LPAREN"
	T_RPAREN     TokenType = "T_RPAREN"
	T_SEMICOLON  TokenType = "T_SEMICOLON"
	T_EOF        TokenType = "T_EOF"
)

type Token struct {
	Type    TokenType
	Literal string
	Pos     int
}
