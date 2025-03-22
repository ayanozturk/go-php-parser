package token

type TokenType string

const (
	T_ILLEGAL                  TokenType = "T_ILLEGAL"
	T_EOF                      TokenType = "T_EOF"
	T_WHITESPACE               TokenType = "T_WHITESPACE"
	T_COMMENT                  TokenType = "T_COMMENT"
	T_DOC_COMMENT              TokenType = "T_DOC_COMMENT"
	T_OPEN_TAG                 TokenType = "T_OPEN_TAG"
	T_CLOSE_TAG                TokenType = "T_CLOSE_TAG"
	T_VARIABLE                 TokenType = "T_VARIABLE"
	T_STRING                   TokenType = "T_STRING"
	T_LNUMBER                  TokenType = "T_LNUMBER"
	T_DNUMBER                  TokenType = "T_DNUMBER"
	T_CONSTANT_ENCAPSED_STRING TokenType = "T_CONSTANT_ENCAPSED_STRING"
	T_CONSTANT_STRING          TokenType = "T_CONSTANT_STRING"
	T_IDENTIFIER               TokenType = "T_IDENTIFIER"
	T_LPAREN                   TokenType = "T_LPAREN"
	T_RPAREN                   TokenType = "T_RPAREN"
	T_LBRACE                   TokenType = "T_LBRACE"
	T_RBRACE                   TokenType = "T_RBRACE"
	T_LBRACKET                 TokenType = "T_LBRACKET"
	T_RBRACKET                 TokenType = "T_RBRACKET"
	T_SEMICOLON                TokenType = "T_SEMICOLON"
	T_COMMA                    TokenType = "T_COMMA"
	T_ASSIGN                   TokenType = "T_ASSIGN"
	T_PLUS                     TokenType = "T_PLUS"
	T_MINUS                    TokenType = "T_MINUS"
	T_MULTIPLY                 TokenType = "T_MULTIPLY"
	T_DIVIDE                   TokenType = "T_DIVIDE"
	T_MODULO                   TokenType = "T_MODULO"
	T_COLON                    TokenType = "T_COLON"
	T_DOUBLE_ARROW             TokenType = "T_DOUBLE_ARROW"
	T_DOT                      TokenType = "T_DOT"
	T_ELLIPSIS                 TokenType = "T_ELLIPSIS"
	T_AMPERSAND                TokenType = "T_AMPERSAND"
	T_OBJECT_OP                TokenType = "T_OBJECT_OP"
	T_ARRAY                    TokenType = "T_ARRAY"
	T_CALLABLE                 TokenType = "T_CALLABLE"
	T_FUNCTION                 TokenType = "T_FUNCTION"
	T_PUBLIC                   TokenType = "T_PUBLIC"
	T_PRIVATE                  TokenType = "T_PRIVATE"
	T_PROTECTED                TokenType = "T_PROTECTED"
	T_RETURN                   TokenType = "T_RETURN"
	T_IF                       TokenType = "T_IF"
	T_ELSE                     TokenType = "T_ELSE"
	T_ELSEIF                   TokenType = "T_ELSEIF"
	T_ENDIF                    TokenType = "T_ENDIF"
	T_FOR                      TokenType = "T_FOR"
	T_WHILE                    TokenType = "T_WHILE"
	T_DO                       TokenType = "T_DO"
	T_BREAK                    TokenType = "T_BREAK"
	T_CONTINUE                 TokenType = "T_CONTINUE"
	T_CLASS                    TokenType = "T_CLASS"
	T_INTERFACE                TokenType = "T_INTERFACE"
	T_EXTENDS                  TokenType = "T_EXTENDS"
	T_IMPLEMENTS               TokenType = "T_IMPLEMENTS"
	T_NEW                      TokenType = "T_NEW"
	T_ECHO                     TokenType = "T_ECHO"
	T_TRUE                     TokenType = "T_TRUE"
	T_FALSE                    TokenType = "T_FALSE"
	T_NULL                     TokenType = "T_NULL"
	T_ENUM                     TokenType = "T_ENUM"
	T_CASE                     TokenType = "T_CASE"
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
