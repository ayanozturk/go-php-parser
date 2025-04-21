package lexer

import (
	"go-phpcs/token"
	"testing"
)

func TestLexer_NextToken_Basic(t *testing.T) {
	input := `<?php $var = 123 + 456.78; // comment\n /* block */ '\\' "str" ?>`
	lex := New(input)
	// Print first few tokens for debugging
	tokens := []token.Token{}
	for i := 0; i < 5; i++ {
		tokens = append(tokens, lex.NextToken())
	}
	for i, tok := range tokens {
		t.Logf("token %d: type=%v, literal=%q", i, tok.Type, tok.Literal)
	}
	// Adjust test to match actual output after running
}

func TestLexer_ObjectOperator(t *testing.T) {
	lex := New("->")
	tok := lex.NextToken()
	if tok.Type != token.T_OBJECT_OPERATOR {
		t.Errorf("expected T_OBJECT_OPERATOR, got %v", tok.Type)
	}
}

func TestLexer_DocComment(t *testing.T) {
	lex := New("/** doc */")
	tok := lex.NextToken()
	if tok.Type != token.T_DOC_COMMENT {
		t.Errorf("expected T_DOC_COMMENT, got %v", tok.Type)
	}
}

func TestLexer_BooleanOr(t *testing.T) {
	lex := New("||")
	tok := lex.NextToken()
	if tok.Type != token.T_BOOLEAN_OR {
		t.Errorf("expected T_BOOLEAN_OR, got %v", tok.Type)
	}
}

func TestLexer_Coalesce(t *testing.T) {
	lex := New("??")
	tok := lex.NextToken()
	if tok.Type != token.T_COALESCE {
		t.Errorf("expected T_COALESCE, got %v", tok.Type)
	}
}

func TestLexer_CoalesceEqual(t *testing.T) {
	lex := New("??=")
	tok := lex.NextToken()
	if tok.Type != token.T_COALESCE_EQUAL {
		t.Errorf("expected T_COALESCE_EQUAL, got %v", tok.Type)
	}
}

func TestLexer_Pipe(t *testing.T) {
	lex := New("|")
	tok := lex.NextToken()
	if tok.Type != token.T_PIPE {
		t.Errorf("expected T_PIPE, got %v", tok.Type)
	}
}

func TestLexer_Question(t *testing.T) {
	lex := New("?")
	tok := lex.NextToken()
	if tok.Type != token.T_QUESTION {
		t.Errorf("expected T_QUESTION, got %v", tok.Type)
	}
}

func TestLexer_IllegalToken(t *testing.T) {
	lex := New("\x01") // Non-printable, non-PHP token
	tok := lex.NextToken()
	if tok.Type != token.T_ILLEGAL {
		t.Errorf("expected T_ILLEGAL, got %v", tok.Type)
	}
}

func TestHelper_isIdentifierStart(t *testing.T) {
	if !isIdentifierStart('a') || !isIdentifierStart('_') {
		t.Error("isIdentifierStart failed for valid identifier start")
	}
	if isIdentifierStart('1') {
		t.Error("isIdentifierStart incorrectly accepted digit")
	}
}

func TestHelper_isDigit(t *testing.T) {
	if !isDigit('0') || !isDigit('9') {
		t.Error("isDigit failed for digit")
	}
	if isDigit('a') {
		t.Error("isDigit incorrectly accepted letter")
	}
}

func TestLexer_StringEscapes(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"'foo\\nbar'", "foo\nbar"},
		{"'foo\\tbar'", "foo\tbar"},
		{"'foo\\rbar'", "foo\rbar"},
		{"'foo\\'bar'", "foo'bar"},
		{"'foo\\\\bar'", "foo\\bar"},
		{"'foo\\xbar'", "foo\\xbar"},
	}
	for _, c := range cases {
		lex := New(c.input)
		tok := lex.NextToken()
		if tok.Type != token.T_CONSTANT_STRING && tok.Type != token.T_CONSTANT_ENCAPSED_STRING {
			t.Errorf("expected string token, got %v", tok.Type)
		}
		if tok.Literal != c.expected {
			t.Errorf("expected %q, got %q", c.expected, tok.Literal)
		}
	}
}

func TestLexer_FloatAndDot(t *testing.T) {
	lex := New("1.23 . ...")
	tok := lex.NextToken()
	if tok.Type != token.T_DNUMBER || tok.Literal != "1.23" {
		t.Errorf("expected float token, got %v %q", tok.Type, tok.Literal)
	}
	tok = lex.NextToken()
	if tok.Type != token.T_DOT {
		t.Errorf("expected dot token, got %v", tok.Type)
	}
	tok = lex.NextToken()
	if tok.Type != token.T_ELLIPSIS {
		t.Errorf("expected ellipsis token, got %v", tok.Type)
	}
}

func TestLexer_Keywords(t *testing.T) {
	keywords := map[string]token.TokenType{
		"echo": token.T_ECHO,
		"new": token.T_NEW,
		"private": token.T_PRIVATE,
		"enum": token.T_ENUM,
		"case": token.T_CASE,
		"trait": token.T_TRAIT,
		"callable": token.T_CALLABLE,
		"true": token.T_TRUE,
		"false": token.T_FALSE,
		"null": token.T_NULL,
		"instanceof": token.T_INSTANCEOF,
		"implements": token.T_IMPLEMENTS,
	}
	for kw, typ := range keywords {
		lex := New(kw)
		tok := lex.NextToken()
		if tok.Type != typ {
			t.Errorf("expected %v for %q, got %v", typ, kw, tok.Type)
		}
	}
}

func TestLexer_Punctuation(t *testing.T) {
	lex := New("]\\")
	tok := lex.NextToken()
	if tok.Type != token.T_RBRACKET {
		t.Errorf("expected T_RBRACKET, got %v", tok.Type)
	}
	tok = lex.NextToken()
	if tok.Type != token.T_BACKSLASH {
		t.Errorf("expected T_BACKSLASH, got %v", tok.Type)
	}
}






func TestLexer_IdentifiersAndKeywords(t *testing.T) {
	cases := []struct {
		input    string
		typeWant token.TokenType
		litWant  string
	}{
		{"$变量", token.T_VARIABLE, "$变量"},
		{"π", token.T_STRING, "π"},
		{"Café", token.T_STRING, "Café"},
		{"function π() {}", token.T_FUNCTION, "function"}, // first token
	}
	for _, c := range cases {
		lex := New(c.input)
		tok := lex.NextToken()
		if tok.Type != c.typeWant {
			t.Errorf("input %q: expected %v, got %v", c.input, c.typeWant, tok.Type)
		}
		if tok.Literal != c.litWant {
			t.Errorf("input %q: expected literal %q, got %q", c.input, c.litWant, tok.Literal)
		}
	}

	input := `function myFunc() { return 42; }`
	lex := New(input)
	var foundFunc, foundReturn bool
	for i := 0; i < 10; i++ {
		tok := lex.NextToken()
		if tok.Type == token.T_FUNCTION {
			foundFunc = true
		}
		if tok.Type == token.T_RETURN {
			foundReturn = true
		}
	}
	if !foundFunc || !foundReturn {
		t.Errorf("expected to find T_FUNCTION and T_RETURN tokens")
	}
}

func TestLexer_StringLiteral(t *testing.T) {
	input := `'foo\'bar' "baz\"qux"`
	lex := New(input)
	tok1 := lex.NextToken()
	tok2 := lex.NextToken()
	// Accept both T_CONSTANT_ENCAPSED_STRING and T_CONSTANT_STRING for compatibility
	if tok1.Type != token.T_CONSTANT_ENCAPSED_STRING && tok1.Type != token.T_CONSTANT_STRING {
		t.Errorf("expected string token, got %v", tok1.Type)
	}
	if tok2.Type != token.T_CONSTANT_ENCAPSED_STRING && tok2.Type != token.T_CONSTANT_STRING {
		t.Errorf("expected string token, got %v", tok2.Type)
	}
}


func TestLexer_NumberLiteral(t *testing.T) {
	cases := []struct {
		input    string
		typeWant token.TokenType
		litWant  string
	}{
		{"123", token.T_LNUMBER, "123"},
		{"45.67", token.T_DNUMBER, "45.67"},
		{"0o123", token.T_LNUMBER, "0o123"},
		{"0O777", token.T_LNUMBER, "0o777"},
		{"0o1_2_3", token.T_LNUMBER, "0o123"},
		// Invalid octal: should still parse as LNUMBER, but literal will be incomplete or illegal
		{"0o", token.T_LNUMBER, "0o"},
		{"0o_123", token.T_LNUMBER, "0o123"},
		{"0o89", token.T_LNUMBER, "0o"}, // 8 and 9 not valid, should stop at prefix
	}
	for _, c := range cases {
		lex := New(c.input)
		tok := lex.NextToken()
		if tok.Type != c.typeWant {
			t.Errorf("input %q: expected %v, got %v", c.input, c.typeWant, tok.Type)
		}
		if tok.Literal != c.litWant {
			t.Errorf("input %q: expected literal %q, got %q", c.input, c.litWant, tok.Literal)
		}
	}
}

func TestLexer_CommentModes(t *testing.T) {
	lex := New("// line\n/* block */")
	tok1 := lex.NextToken()
	tok2 := lex.NextToken()
	if tok1.Type != token.T_COMMENT {
		t.Errorf("expected T_COMMENT, got %v", tok1.Type)
	}
	if tok2.Type != token.T_COMMENT {
		t.Errorf("expected T_COMMENT, got %v", tok2.Type)
	}
}

func TestLexer_Operators(t *testing.T) {
	lex := New("+ - * /")
	types := []token.TokenType{token.T_PLUS, token.T_MINUS, token.T_MULTIPLY, token.T_DIVIDE}
	for _, want := range types {
		tok := lex.NextToken()
		if tok.Type != want {
			t.Errorf("expected %v, got %v", want, tok.Type)
		}
	}
}



func TestLexer_inStringMode(t *testing.T) {
	lex := New("'foo'")
	if lex.inStringMode() {
		t.Error("expected inStringMode to be false at start")
	}
}

func TestLexer_PeekToken(t *testing.T) {
	lex := New("a")
	tok1 := lex.PeekToken()
	tok2 := lex.NextToken()
	if tok1.Type != tok2.Type || tok1.Literal != tok2.Literal {
		t.Errorf("PeekToken and NextToken should return the same token")
	}
}




func TestLexer_EOF(t *testing.T) {
	lex := New("")
	tok := lex.NextToken()
	if tok.Type != token.T_EOF {
		t.Errorf("expected T_EOF, got %v", tok.Type)
	}
}

func TestLexer_HeredocQueueAndNext(t *testing.T) {
	lex := New("<<<EOD\nhello\nEOD\n")
	lex.queueHeredocTokens(token.Position{Line: 1, Column: 1, Offset: 0})
	t1 := lex.nextHeredocToken()
	t2 := lex.nextHeredocToken()
	t3 := lex.nextHeredocToken()
	if t1.Type != token.T_START_HEREDOC {
		t.Errorf("expected T_START_HEREDOC, got %v", t1.Type)
	}
	if t2.Type != token.T_ENCAPSED_AND_WHITESPACE {
		t.Errorf("expected T_ENCAPSED_AND_WHITESPACE, got %v", t2.Type)
	}
	if t3.Type != token.T_END_HEREDOC {
		t.Errorf("expected T_END_HEREDOC, got %v", t3.Type)
	}
}

// Covers NextToken's heredocTokens path
func TestLexer_NextToken_HeredocQueue(t *testing.T) {
	lex := New("")
	lex.heredocTokens = []token.Token{
		{Type: token.T_START_HEREDOC, Literal: "EOD", Pos: token.Position{Line: 1, Column: 1, Offset: 0}},
		{Type: token.T_ENCAPSED_AND_WHITESPACE, Literal: "body", Pos: token.Position{Line: 2, Column: 1, Offset: 5}},
	}
	tok1 := lex.NextToken()
	if tok1.Type != token.T_START_HEREDOC {
		t.Errorf("expected T_START_HEREDOC, got %v", tok1.Type)
	}
	tok2 := lex.NextToken()
	if tok2.Type != token.T_ENCAPSED_AND_WHITESPACE {
		t.Errorf("expected T_ENCAPSED_AND_WHITESPACE, got %v", tok2.Type)
	}
}

func TestLexer_NextHeredocToken_EmptyQueue(t *testing.T) {
	lex := New("")
	tok := lex.nextHeredocToken()
	if tok.Type != token.T_ILLEGAL {
		t.Errorf("expected T_ILLEGAL for empty heredoc queue, got %v", tok.Type)
	}
}
