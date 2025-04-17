package token

type TokenType string

const (
	// Special tokens
	T_ILLEGAL       TokenType = "T_ILLEGAL"
	T_EOF           TokenType = "T_EOF"
	T_WHITESPACE    TokenType = "T_WHITESPACE"
	T_COMMENT       TokenType = "T_COMMENT"
	T_DOC_COMMENT   TokenType = "T_DOC_COMMENT"
	T_OPEN_TAG      TokenType = "T_OPEN_TAG"
	T_CLOSE_TAG     TokenType = "T_CLOSE_TAG"
	T_BAD_CHARACTER TokenType = "T_BAD_CHARACTER"

	// Variables and literals
	T_VARIABLE                 TokenType = "T_VARIABLE"
	T_STRING                   TokenType = "T_STRING"
	T_LNUMBER                  TokenType = "T_LNUMBER"
	T_DNUMBER                  TokenType = "T_DNUMBER"
	T_CONSTANT_ENCAPSED_STRING TokenType = "T_CONSTANT_ENCAPSED_STRING"
	T_CONSTANT_STRING          TokenType = "T_CONSTANT_STRING"
	T_STRING_VARNAME           TokenType = "T_STRING_VARNAME"
	T_NUM_STRING               TokenType = "T_NUM_STRING"
	T_ENCAPSED_AND_WHITESPACE  TokenType = "T_ENCAPSED_AND_WHITESPACE"
	T_INLINE_HTML              TokenType = "T_INLINE_HTML"

	// Operators
	T_PLUS             TokenType = "T_PLUS"
	T_MINUS            TokenType = "T_MINUS"
	T_MULTIPLY         TokenType = "T_MULTIPLY"
	T_DIVIDE           TokenType = "T_DIVIDE"
	T_MODULO           TokenType = "T_MODULO"
	T_AND_EQUAL        TokenType = "T_AND_EQUAL"
	T_CONCAT_EQUAL     TokenType = "T_CONCAT_EQUAL"
	T_DIV_EQUAL        TokenType = "T_DIV_EQUAL"
	T_MINUS_EQUAL      TokenType = "T_MINUS_EQUAL"
	T_MOD_EQUAL        TokenType = "T_MOD_EQUAL"
	T_MUL_EQUAL        TokenType = "T_MUL_EQUAL"
	T_PLUS_EQUAL       TokenType = "T_PLUS_EQUAL"
	T_XOR_EQUAL        TokenType = "T_XOR_EQUAL"
	T_BOOLEAN_AND      TokenType = "T_BOOLEAN_AND"
	T_BOOLEAN_OR       TokenType = "T_BOOLEAN_OR"
	T_IS_EQUAL         TokenType = "T_IS_EQUAL"
	T_IS_NOT_EQUAL     TokenType = "T_IS_NOT_EQUAL"
	T_IS_IDENTICAL     TokenType = "T_IS_IDENTICAL"
	T_IS_NOT_IDENTICAL TokenType = "T_IS_NOT_IDENTICAL"
	T_IS_SMALLER       TokenType = "T_IS_SMALLER"
	T_IS_GREATER       TokenType = "T_IS_GREATER"
	T_SPACESHIP        TokenType = "T_SPACESHIP"
	T_COALESCE         TokenType = "T_COALESCE"
	T_COALESCE_EQUAL   TokenType = "T_COALESCE_EQUAL"
	T_QUESTION         TokenType = "T_QUESTION" // ? for nullable types

	// Delimiters
	T_LPAREN                   TokenType = "T_LPAREN"
	T_RPAREN                   TokenType = "T_RPAREN"
	T_LBRACE                   TokenType = "T_LBRACE"
	T_RBRACE                   TokenType = "T_RBRACE"
	T_LBRACKET                 TokenType = "T_LBRACKET"
	T_RBRACKET                 TokenType = "T_RBRACKET"
	T_SEMICOLON                TokenType = "T_SEMICOLON"
	T_COMMA                    TokenType = "T_COMMA"
	T_ASSIGN                   TokenType = "T_ASSIGN"
	T_COLON                    TokenType = "T_COLON"
	T_DOUBLE_ARROW             TokenType = "T_DOUBLE_ARROW"
	T_BACKSLASH                TokenType = "T_BACKSLASH"
	T_DOT                      TokenType = "T_DOT"
	T_ELLIPSIS                 TokenType = "T_ELLIPSIS"
	T_AMPERSAND                TokenType = "T_AMPERSAND"
	T_OBJECT_OPERATOR          TokenType = "T_OBJECT_OPERATOR"
	T_NULLSAFE_OBJECT_OPERATOR TokenType = "T_NULLSAFE_OBJECT_OPERATOR"
	T_DOUBLE_COLON             TokenType = "T_DOUBLE_COLON"
	T_CLASS_CONST              TokenType = "T_CLASS_CONST" // Represents "::class"

	// Keywords
	T_ABSTRACT     TokenType = "T_ABSTRACT"
	T_ARRAY        TokenType = "T_ARRAY"
	T_AS           TokenType = "T_AS"
	T_BREAK        TokenType = "T_BREAK"
	T_CALLABLE     TokenType = "T_CALLABLE"
	T_CASE         TokenType = "T_CASE"
	T_CATCH        TokenType = "T_CATCH"
	T_CLASS        TokenType = "T_CLASS"
	T_CLONE        TokenType = "T_CLONE"
	T_CONST        TokenType = "T_CONST"
	T_CONTINUE     TokenType = "T_CONTINUE"
	T_DECLARE      TokenType = "T_DECLARE"
	T_DEFAULT      TokenType = "T_DEFAULT"
	T_DO           TokenType = "T_DO"
	T_ECHO         TokenType = "T_ECHO"
	T_ELSE         TokenType = "T_ELSE"
	T_ELSEIF       TokenType = "T_ELSEIF"
	T_EMPTY        TokenType = "T_EMPTY"
	T_ENDDECLARE   TokenType = "T_ENDDECLARE"
	T_ENDFOR       TokenType = "T_ENDFOR"
	T_ENDFOREACH   TokenType = "T_ENDFOREACH"
	T_ENDIF        TokenType = "T_ENDIF"
	T_ENDSWITCH    TokenType = "T_ENDSWITCH"
	T_ENDWHILE     TokenType = "T_ENDWHILE"
	T_ENUM         TokenType = "T_ENUM"
	T_EXTENDS      TokenType = "T_EXTENDS"
	T_FINAL        TokenType = "T_FINAL"
	T_FINALLY      TokenType = "T_FINALLY"
	T_FN           TokenType = "T_FN"
	T_FOR          TokenType = "T_FOR"
	T_FOREACH      TokenType = "T_FOREACH"
	T_FUNCTION     TokenType = "T_FUNCTION"
	T_GLOBAL       TokenType = "T_GLOBAL"
	T_GOTO         TokenType = "T_GOTO"
	T_IF           TokenType = "T_IF"
	T_IMPLEMENTS   TokenType = "T_IMPLEMENTS"
	T_INCLUDE      TokenType = "T_INCLUDE"
	T_INCLUDE_ONCE TokenType = "T_INCLUDE_ONCE"
	T_INSTANCEOF   TokenType = "T_INSTANCEOF"
	T_INSTEADOF    TokenType = "T_INSTEADOF"
	T_INTERFACE    TokenType = "T_INTERFACE"
	T_ISSET        TokenType = "T_ISSET"
	T_LIST         TokenType = "T_LIST"
	T_MATCH        TokenType = "T_MATCH"
	T_NAMESPACE    TokenType = "T_NAMESPACE"
	T_NEW          TokenType = "T_NEW"
	T_PRIVATE      TokenType = "T_PRIVATE"
	T_PROTECTED    TokenType = "T_PROTECTED"
	T_PUBLIC       TokenType = "T_PUBLIC"
	T_REQUIRE      TokenType = "T_REQUIRE"
	T_REQUIRE_ONCE TokenType = "T_REQUIRE_ONCE"
	T_RETURN       TokenType = "T_RETURN"
	T_STATIC       TokenType = "T_STATIC"
	T_SWITCH       TokenType = "T_SWITCH"
	T_THROW        TokenType = "T_THROW"
	T_TRAIT        TokenType = "T_TRAIT"
	T_TRY          TokenType = "T_TRY"
	T_UNSET        TokenType = "T_UNSET"
	T_USE          TokenType = "T_USE"
	T_VAR          TokenType = "T_VAR"
	T_WHILE        TokenType = "T_WHILE"
	T_YIELD        TokenType = "T_YIELD"
	T_YIELD_FROM   TokenType = "T_YIELD_FROM"

	// Magic Constants
	T_CLASS_C  TokenType = "T_CLASS_C"
	T_DIR      TokenType = "T_DIR"
	T_FILE     TokenType = "T_FILE"
	T_FUNC_C   TokenType = "T_FUNC_C"
	T_LINE     TokenType = "T_LINE"
	T_METHOD_C TokenType = "T_METHOD_C"
	T_NS_C     TokenType = "T_NS_C"
	T_TRAIT_C  TokenType = "T_TRAIT_C"

	// Special Constants
	T_TRUE  TokenType = "T_TRUE"
	T_FALSE TokenType = "T_FALSE"
	T_NULL  TokenType = "T_NULL"

	// Type casting
	T_ARRAY_CAST  TokenType = "T_ARRAY_CAST"
	T_BOOL_CAST   TokenType = "T_BOOL_CAST"
	T_DOUBLE_CAST TokenType = "T_DOUBLE_CAST"
	T_INT_CAST    TokenType = "T_INT_CAST"
	T_OBJECT_CAST TokenType = "T_OBJECT_CAST"
	T_STRING_CAST TokenType = "T_STRING_CAST"
	T_UNSET_CAST  TokenType = "T_UNSET_CAST"

	// Heredoc/Nowdoc
	T_START_HEREDOC            TokenType = "T_START_HEREDOC"
	T_END_HEREDOC              TokenType = "T_END_HEREDOC"
	T_DOLLAR_OPEN_CURLY_BRACES TokenType = "T_DOLLAR_OPEN_CURLY_BRACES"
	T_CURLY_OPEN               TokenType = "T_CURLY_OPEN"

	// Attributes (PHP 8.0+)
	T_ATTRIBUTE TokenType = "T_ATTRIBUTE"
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
