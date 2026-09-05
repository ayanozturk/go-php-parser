package token

type TokenType uint16

const (
	T_MATCH TokenType = iota + 1
	T_FN
	T_NULLSAFE_OBJECT_OPERATOR
	T_ATTRIBUTE
	T_READONLY
	T_ENUM
	T_NEVER
	T_TRAILING_COMMA
	T_SELF
	T_PARENT
	T_ILLEGAL
	T_EOF
	T_WHITESPACE
	T_COMMENT
	T_DOC_COMMENT
	T_OPEN_TAG
	T_CLOSE_TAG
	T_BAD_CHARACTER
	T_VARIABLE
	T_STRING
	T_LNUMBER
	T_DNUMBER
	T_CONSTANT_ENCAPSED_STRING
	T_CONSTANT_STRING
	T_STRING_VARNAME
	T_NUM_STRING
	T_ENCAPSED_AND_WHITESPACE
	T_INLINE_HTML
	T_PLUS
	T_MINUS
	T_INC
	T_DEC
	T_MULTIPLY
	T_DIVIDE
	T_MODULO
	T_POW
	T_AND_EQUAL
	T_CONCAT_EQUAL
	T_DIV_EQUAL
	T_MINUS_EQUAL
	T_MOD_EQUAL
	T_MUL_EQUAL
	T_OR_EQUAL
	T_POW_EQUAL
	T_PLUS_EQUAL
	T_XOR_EQUAL
	T_BOOLEAN_AND
	T_BOOLEAN_OR
	T_LOGICAL_AND
	T_LOGICAL_OR
	T_LOGICAL_XOR
	T_CARET
	T_SL
	T_SR
	T_SL_EQUAL
	T_SR_EQUAL
	T_IS_EQUAL
	T_IS_NOT_EQUAL
	T_IS_IDENTICAL
	T_IS_NOT_IDENTICAL
	T_IS_SMALLER
	T_IS_GREATER
	T_IS_GREATER_OR_EQUAL
	T_IS_SMALLER_OR_EQUAL
	T_SPACESHIP
	T_COALESCE
	T_COALESCE_EQUAL
	T_QUESTION
	T_PIPE
	T_NOT
	T_AT
	T_TILDE
	T_LPAREN
	T_RPAREN
	T_LBRACE
	T_RBRACE
	T_LBRACKET
	T_RBRACKET
	T_SEMICOLON
	T_COMMA
	T_ASSIGN
	T_COLON
	T_DOUBLE_ARROW
	T_BACKSLASH
	T_DOT
	T_ELLIPSIS
	T_AMPERSAND
	T_OBJECT_OPERATOR
	T_DOUBLE_COLON
	T_CLASS_CONST
	T_ABSTRACT
	T_ARRAY
	T_AS
	T_BREAK
	T_CALLABLE
	T_CASE
	T_CATCH
	T_CLASS
	T_CLONE
	T_CONST
	T_CONTINUE
	T_DECLARE
	T_DEFAULT
	T_DO
	T_ECHO
	T_PRINT
	T_ELSE
	T_ELSEIF
	T_EMPTY
	T_ENDDECLARE
	T_ENDFOR
	T_ENDFOREACH
	T_ENDIF
	T_ENDSWITCH
	T_ENDWHILE
	T_EXTENDS
	T_FINAL
	T_FINALLY
	T_FOR
	T_FOREACH
	T_FUNCTION
	T_GLOBAL
	T_GOTO
	T_IF
	T_IMPLEMENTS
	T_INCLUDE
	T_INCLUDE_ONCE
	T_INSTANCEOF
	T_INSTEADOF
	T_INTERFACE
	T_ISSET
	T_LIST
	T_DIE
	T_EXIT
	T_NAMESPACE
	T_NS_SEPARATOR
	T_NEW
	T_PRIVATE
	T_PROTECTED
	T_PUBLIC
	T_REQUIRE
	T_REQUIRE_ONCE
	T_RETURN
	T_STATIC
	T_SWITCH
	T_THROW
	T_TRAIT
	T_TRY
	T_UNSET
	T_USE
	T_VAR
	T_WHILE
	T_YIELD
	T_YIELD_FROM
	T_CLASS_C
	T_DIR
	T_FILE
	T_FUNC_C
	T_LINE
	T_METHOD_C
	T_NS_C
	T_TRAIT_C
	T_TRUE
	T_FALSE
	T_NULL
	T_MIXED
	T_ARRAY_CAST
	T_BOOL_CAST
	T_DOUBLE_CAST
	T_INT_CAST
	T_OBJECT_CAST
	T_STRING_CAST
	T_UNSET_CAST
	T_START_HEREDOC
	T_END_HEREDOC
	T_START_NOWDOC
	T_END_NOWDOC
	T_DOLLAR_OPEN_CURLY_BRACES
	T_CURLY_OPEN
)

var tokenTypeNames = [...]string{
	T_MATCH:                    "T_MATCH",
	T_FN:                       "T_FN",
	T_NULLSAFE_OBJECT_OPERATOR: "T_NULLSAFE_OBJECT_OPERATOR",
	T_ATTRIBUTE:                "T_ATTRIBUTE",
	T_READONLY:                 "T_READONLY",
	T_ENUM:                     "T_ENUM",
	T_NEVER:                    "T_NEVER",
	T_TRAILING_COMMA:           "T_TRAILING_COMMA",
	T_SELF:                     "T_SELF",
	T_PARENT:                   "T_PARENT",
	T_ILLEGAL:                  "T_ILLEGAL",
	T_EOF:                      "T_EOF",
	T_WHITESPACE:               "T_WHITESPACE",
	T_COMMENT:                  "T_COMMENT",
	T_DOC_COMMENT:              "T_DOC_COMMENT",
	T_OPEN_TAG:                 "T_OPEN_TAG",
	T_CLOSE_TAG:                "T_CLOSE_TAG",
	T_BAD_CHARACTER:            "T_BAD_CHARACTER",
	T_VARIABLE:                 "T_VARIABLE",
	T_STRING:                   "T_STRING",
	T_LNUMBER:                  "T_LNUMBER",
	T_DNUMBER:                  "T_DNUMBER",
	T_CONSTANT_ENCAPSED_STRING: "T_CONSTANT_ENCAPSED_STRING",
	T_CONSTANT_STRING:          "T_CONSTANT_STRING",
	T_STRING_VARNAME:           "T_STRING_VARNAME",
	T_NUM_STRING:               "T_NUM_STRING",
	T_ENCAPSED_AND_WHITESPACE:  "T_ENCAPSED_AND_WHITESPACE",
	T_INLINE_HTML:              "T_INLINE_HTML",
	T_PLUS:                     "T_PLUS",
	T_MINUS:                    "T_MINUS",
	T_INC:                      "T_INC",
	T_DEC:                      "T_DEC",
	T_MULTIPLY:                 "T_MULTIPLY",
	T_DIVIDE:                   "T_DIVIDE",
	T_MODULO:                   "T_MODULO",
	T_POW:                      "T_POW",
	T_AND_EQUAL:                "T_AND_EQUAL",
	T_CONCAT_EQUAL:             "T_CONCAT_EQUAL",
	T_DIV_EQUAL:                "T_DIV_EQUAL",
	T_MINUS_EQUAL:              "T_MINUS_EQUAL",
	T_MOD_EQUAL:                "T_MOD_EQUAL",
	T_MUL_EQUAL:                "T_MUL_EQUAL",
	T_OR_EQUAL:                 "T_OR_EQUAL",
	T_POW_EQUAL:                "T_POW_EQUAL",
	T_PLUS_EQUAL:               "T_PLUS_EQUAL",
	T_XOR_EQUAL:                "T_XOR_EQUAL",
	T_BOOLEAN_AND:              "T_BOOLEAN_AND",
	T_BOOLEAN_OR:               "T_BOOLEAN_OR",
	T_LOGICAL_AND:              "T_LOGICAL_AND",
	T_LOGICAL_OR:               "T_LOGICAL_OR",
	T_LOGICAL_XOR:              "T_LOGICAL_XOR",
	T_CARET:                    "T_CARET",
	T_SL:                       "T_SL",
	T_SR:                       "T_SR",
	T_SL_EQUAL:                 "T_SL_EQUAL",
	T_SR_EQUAL:                 "T_SR_EQUAL",
	T_IS_EQUAL:                 "T_IS_EQUAL",
	T_IS_NOT_EQUAL:             "T_IS_NOT_EQUAL",
	T_IS_IDENTICAL:             "T_IS_IDENTICAL",
	T_IS_NOT_IDENTICAL:         "T_IS_NOT_IDENTICAL",
	T_IS_SMALLER:               "T_IS_SMALLER",
	T_IS_GREATER:               "T_IS_GREATER",
	T_IS_GREATER_OR_EQUAL:      "T_IS_GREATER_OR_EQUAL",
	T_IS_SMALLER_OR_EQUAL:      "T_IS_SMALLER_OR_EQUAL",
	T_SPACESHIP:                "T_SPACESHIP",
	T_COALESCE:                 "T_COALESCE",
	T_COALESCE_EQUAL:           "T_COALESCE_EQUAL",
	T_QUESTION:                 "T_QUESTION",
	T_PIPE:                     "T_PIPE",
	T_NOT:                      "T_NOT",
	T_AT:                       "T_AT",
	T_TILDE:                    "T_TILDE",
	T_LPAREN:                   "T_LPAREN",
	T_RPAREN:                   "T_RPAREN",
	T_LBRACE:                   "T_LBRACE",
	T_RBRACE:                   "T_RBRACE",
	T_LBRACKET:                 "T_LBRACKET",
	T_RBRACKET:                 "T_RBRACKET",
	T_SEMICOLON:                "T_SEMICOLON",
	T_COMMA:                    "T_COMMA",
	T_ASSIGN:                   "T_ASSIGN",
	T_COLON:                    "T_COLON",
	T_DOUBLE_ARROW:             "T_DOUBLE_ARROW",
	T_BACKSLASH:                "T_BACKSLASH",
	T_DOT:                      "T_DOT",
	T_ELLIPSIS:                 "T_ELLIPSIS",
	T_AMPERSAND:                "T_AMPERSAND",
	T_OBJECT_OPERATOR:          "T_OBJECT_OPERATOR",
	T_DOUBLE_COLON:             "T_DOUBLE_COLON",
	T_CLASS_CONST:              "T_CLASS_CONST",
	T_ABSTRACT:                 "T_ABSTRACT",
	T_ARRAY:                    "T_ARRAY",
	T_AS:                       "T_AS",
	T_BREAK:                    "T_BREAK",
	T_CALLABLE:                 "T_CALLABLE",
	T_CASE:                     "T_CASE",
	T_CATCH:                    "T_CATCH",
	T_CLASS:                    "T_CLASS",
	T_CLONE:                    "T_CLONE",
	T_CONST:                    "T_CONST",
	T_CONTINUE:                 "T_CONTINUE",
	T_DECLARE:                  "T_DECLARE",
	T_DEFAULT:                  "T_DEFAULT",
	T_DO:                       "T_DO",
	T_ECHO:                     "T_ECHO",
	T_PRINT:                    "T_PRINT",
	T_ELSE:                     "T_ELSE",
	T_ELSEIF:                   "T_ELSEIF",
	T_EMPTY:                    "T_EMPTY",
	T_ENDDECLARE:               "T_ENDDECLARE",
	T_ENDFOR:                   "T_ENDFOR",
	T_ENDFOREACH:               "T_ENDFOREACH",
	T_ENDIF:                    "T_ENDIF",
	T_ENDSWITCH:                "T_ENDSWITCH",
	T_ENDWHILE:                 "T_ENDWHILE",
	T_EXTENDS:                  "T_EXTENDS",
	T_FINAL:                    "T_FINAL",
	T_FINALLY:                  "T_FINALLY",
	T_FOR:                      "T_FOR",
	T_FOREACH:                  "T_FOREACH",
	T_FUNCTION:                 "T_FUNCTION",
	T_GLOBAL:                   "T_GLOBAL",
	T_GOTO:                     "T_GOTO",
	T_IF:                       "T_IF",
	T_IMPLEMENTS:               "T_IMPLEMENTS",
	T_INCLUDE:                  "T_INCLUDE",
	T_INCLUDE_ONCE:             "T_INCLUDE_ONCE",
	T_INSTANCEOF:               "T_INSTANCEOF",
	T_INSTEADOF:                "T_INSTEADOF",
	T_INTERFACE:                "T_INTERFACE",
	T_ISSET:                    "T_ISSET",
	T_LIST:                     "T_LIST",
	T_DIE:                      "T_DIE",
	T_EXIT:                     "T_EXIT",
	T_NAMESPACE:                "T_NAMESPACE",
	T_NS_SEPARATOR:             "T_NS_SEPARATOR",
	T_NEW:                      "T_NEW",
	T_PRIVATE:                  "T_PRIVATE",
	T_PROTECTED:                "T_PROTECTED",
	T_PUBLIC:                   "T_PUBLIC",
	T_REQUIRE:                  "T_REQUIRE",
	T_REQUIRE_ONCE:             "T_REQUIRE_ONCE",
	T_RETURN:                   "T_RETURN",
	T_STATIC:                   "T_STATIC",
	T_SWITCH:                   "T_SWITCH",
	T_THROW:                    "T_THROW",
	T_TRAIT:                    "T_TRAIT",
	T_TRY:                      "T_TRY",
	T_UNSET:                    "T_UNSET",
	T_USE:                      "T_USE",
	T_VAR:                      "T_VAR",
	T_WHILE:                    "T_WHILE",
	T_YIELD:                    "T_YIELD",
	T_YIELD_FROM:               "T_YIELD_FROM",
	T_CLASS_C:                  "T_CLASS_C",
	T_DIR:                      "T_DIR",
	T_FILE:                     "T_FILE",
	T_FUNC_C:                   "T_FUNC_C",
	T_LINE:                     "T_LINE",
	T_METHOD_C:                 "T_METHOD_C",
	T_NS_C:                     "T_NS_C",
	T_TRAIT_C:                  "T_TRAIT_C",
	T_TRUE:                     "T_TRUE",
	T_FALSE:                    "T_FALSE",
	T_NULL:                     "T_NULL",
	T_MIXED:                    "T_MIXED",
	T_ARRAY_CAST:               "T_ARRAY_CAST",
	T_BOOL_CAST:                "T_BOOL_CAST",
	T_DOUBLE_CAST:              "T_DOUBLE_CAST",
	T_INT_CAST:                 "T_INT_CAST",
	T_OBJECT_CAST:              "T_OBJECT_CAST",
	T_STRING_CAST:              "T_STRING_CAST",
	T_UNSET_CAST:               "T_UNSET_CAST",
	T_START_HEREDOC:            "T_START_HEREDOC",
	T_END_HEREDOC:              "T_END_HEREDOC",
	T_START_NOWDOC:             "T_START_NOWDOC",
	T_END_NOWDOC:               "T_END_NOWDOC",
	T_DOLLAR_OPEN_CURLY_BRACES: "T_DOLLAR_OPEN_CURLY_BRACES",
	T_CURLY_OPEN:               "T_CURLY_OPEN",
}

func (t TokenType) String() string {
	if int(t) < len(tokenTypeNames) {
		if name := tokenTypeNames[t]; name != "" {
			return name
		}
	}
	return "T_UNKNOWN"
}

type Position struct {
	Line   int
	Column int
	Offset int
}

type Token struct {
	Type    TokenType
	Literal string
	Pos     Position
	End     Position
}

// EndPos returns the position immediately after this token's literal text,
// i.e. a half-open [Pos, EndPos) span. Lexer-produced tokens store End
// directly; tokens constructed in tests without End fall back to scanning
// the literal for newlines.
func (t Token) EndPos() Position {
	if t.End.Line != 0 || t.End.Column != 0 || t.End.Offset != 0 {
		return t.End
	}
	end := t.Pos
	lastNewline := -1
	for i := 0; i < len(t.Literal); i++ {
		if t.Literal[i] == '\n' {
			end.Line++
			lastNewline = i
		}
	}
	end.Offset += len(t.Literal)
	if lastNewline >= 0 {
		end.Column = len(t.Literal) - lastNewline
	} else {
		end.Column += len(t.Literal)
	}
	return end
}

// LineTable stores the starting byte offset of each source line.
// The first line always starts at offset 0.
type LineTable []int

func NewLineTable(src []byte) LineTable {
	starts := LineTable{0}
	for i, b := range src {
		if b == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

// LineBytes returns the 1-indexed line without its terminating newline.
func (t LineTable) LineBytes(src []byte, line int) []byte {
	if line < 1 || line > len(t) {
		return nil
	}
	start := t[line-1]
	end := len(src)
	if line < len(t) {
		end = t[line]
	}
	if end > start && src[end-1] == '\n' {
		end--
	}
	if end > start && src[end-1] == '\r' {
		end--
	}
	return src[start:end]
}
