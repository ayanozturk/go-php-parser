package token

type TokenType string

const (
	T_OPEN_TAG                 TokenType = "T_OPEN_TAG"
	T_VARIABLE                 TokenType = "T_VARIABLE"
	T_FUNCTION                 TokenType = "T_FUNCTION"
	T_IDENTIFIER               TokenType = "T_IDENTIFIER"
	T_LPAREN                   TokenType = "T_LPAREN"
	T_RPAREN                   TokenType = "T_RPAREN"
	T_LBRACE                   TokenType = "T_LBRACE"
	T_RBRACE                   TokenType = "T_RBRACE"
	T_SEMICOLON                TokenType = "T_SEMICOLON"
	T_ASSIGN                   TokenType = "T_ASSIGN"
	T_IS_EQUAL                 TokenType = "T_IS_EQUAL"
	T_CONSTANT_STRING          TokenType = "T_CONSTANT_STRING"
	T_CONSTANT_ENCAPSED_STRING TokenType = "T_CONSTANT_ENCAPSED_STRING"
	T_AMPERSAND                TokenType = "T_AMPERSAND"
	T_ELLIPSIS                 TokenType = "T_ELLIPSIS"
	T_COMMA                    TokenType = "T_COMMA"
	T_ARRAY                    TokenType = "T_ARRAY"
	T_STRING                   TokenType = "T_STRING"
	T_CALLABLE                 TokenType = "T_CALLABLE"
	T_EOF                      TokenType = "T_EOF"
	T_COMMENT                  TokenType = "T_COMMENT"
	T_DOC_COMMENT              TokenType = "T_DOC_COMMENT"
	T_LBRACKET                 TokenType = "T_LBRACKET"
	T_RBRACKET                 TokenType = "T_RBRACKET"
	T_DOUBLE_ARROW             TokenType = "T_DOUBLE_ARROW"

	// Control structures
	T_IF     TokenType = "T_IF"
	T_ELSE   TokenType = "T_ELSE"
	T_ELSEIF TokenType = "T_ELSEIF"
	T_ENDIF  TokenType = "T_ENDIF"

	// Literal types
	T_LNUMBER TokenType = "T_LNUMBER" // Integer literal
	T_DNUMBER TokenType = "T_DNUMBER" // Float literal
	T_TRUE    TokenType = "T_TRUE"    // Boolean true
	T_FALSE   TokenType = "T_FALSE"   // Boolean false
	T_NULL    TokenType = "T_NULL"    // Null value

	// Class related tokens
	T_CLASS      TokenType = "T_CLASS"      // class keyword
	T_EXTENDS    TokenType = "T_EXTENDS"    // extends keyword
	T_INTERFACE  TokenType = "T_INTERFACE"  // interface keyword
	T_IMPLEMENTS TokenType = "T_IMPLEMENTS" // implements keyword
	T_NEW        TokenType = "T_NEW"        // new keyword
	T_PUBLIC     TokenType = "T_PUBLIC"     // public keyword
	T_PRIVATE    TokenType = "T_PRIVATE"    // private keyword
	T_PROTECTED  TokenType = "T_PROTECTED"  // protected keyword
	T_OBJECT_OP  TokenType = "T_OBJECT_OP"  // -> operator
	T_COLON      TokenType = "T_COLON"      // : operator (for return types)

	// Output tokens
	T_ECHO TokenType = "T_ECHO" // echo keyword

	// New token types
	T_INT    TokenType = "T_INT"    // integer literal
	T_FLOAT  TokenType = "T_FLOAT"  // float literal
	T_RETURN TokenType = "T_RETURN" // return keyword
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
