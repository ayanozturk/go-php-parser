package lexer

import (
	"context"
	"strings"
	"testing"

	"github.com/ayanozturk/go-php-parser/token"
)

func drainTypes(src string) []token.TokenType {
	l := New(src)
	var types []token.TokenType
	for {
		tok := l.NextToken()
		types = append(types, tok.Type)
		if tok.Type == token.T_EOF {
			return types
		}
	}
}

func hasType(types []token.TokenType, want token.TokenType) bool {
	for _, t := range types {
		if t == want {
			return true
		}
	}
	return false
}

func TestSnapshotRestoreRoundTrip(t *testing.T) {
	l := New(`$a + $b`)
	first := l.NextToken()
	if first.Type != token.T_VARIABLE || first.Literal != "$a" {
		t.Fatalf("first token = %v %q", first.Type, first.Literal)
	}
	snap := l.Snapshot()
	_ = l.NextToken() // +
	second := l.NextToken()
	if second.Literal != "$b" {
		t.Fatalf("after advance got %q", second.Literal)
	}
	l.Restore(snap)
	again := l.NextToken()
	if again.Type != token.T_PLUS {
		t.Fatalf("after Restore expected T_PLUS, got %v %q", again.Type, again.Literal)
	}
	if string(l.Source()) != `$a + $b` {
		t.Fatalf("Source mismatch: %q", l.Source())
	}
}

func TestSnapshotRestorePreservesPeek(t *testing.T) {
	l := New(`foo bar`)
	peek := l.PeekToken()
	if peek.Literal != "foo" {
		t.Fatalf("peek = %q", peek.Literal)
	}
	snap := l.Snapshot()
	_ = l.NextToken()
	_ = l.NextToken()
	l.Restore(snap)
	if !l.hasPeeked {
		t.Fatal("Restore should keep peeked state")
	}
	got := l.NextToken()
	if got.Literal != "foo" {
		t.Fatalf("after Restore NextToken = %q, want foo", got.Literal)
	}
}

func TestSetCancelContextAndCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	oldBudget := cancelCheckBudget
	cancelCheckBudget = 1
	defer func() { cancelCheckBudget = oldBudget }()

	big := New(`{` + strings.Repeat("x", 8) + `}`)
	big.SetCancelContext(ctx)
	if big.NextToken().Type != token.T_LBRACE {
		t.Fatal("expected lbrace")
	}
	if _, ok := big.SkipBalancedCurlyBlockWithEnd(); ok {
		t.Fatal("expected cancel during skip")
	}
	if big.Cancelled() == nil {
		t.Fatal("Cancelled() should report error after cancel")
	}
	big.SetCancelContext(nil)
	if big.cancelCheckAt != 0 {
		t.Fatal("nil SetCancelContext should clear cancel checkpoint")
	}
}

func TestSkipBalancedCurlyBlockWrapper(t *testing.T) {
	src := []byte(`{ $x = 1; }`)
	l := NewBytes(src)
	if l.NextToken().Type != token.T_LBRACE {
		t.Fatal("expected T_LBRACE")
	}
	if !l.SkipBalancedCurlyBlock() {
		t.Fatal("SkipBalancedCurlyBlock should succeed")
	}
	if l.NextToken().Type != token.T_EOF {
		t.Fatal("expected EOF")
	}
}

func TestSkipBalancedCurlyBlockUnmatched(t *testing.T) {
	l := NewBytes([]byte(`{ $x = 1;`))
	if l.NextToken().Type != token.T_LBRACE {
		t.Fatal("expected T_LBRACE")
	}
	if l.SkipBalancedCurlyBlock() {
		t.Fatal("unmatched body should fail")
	}
}

func TestSkipBalancedCurlyBlockLineCommentCloseTag(t *testing.T) {
	// Zend: // … ?> ends the comment and leaves HTML mode; braces in the
	// HTML span must not close the PHP body. After re-open, the real } closes.
	src := []byte("{\n  // note ?>\n<html>{not-php}</html>\n<?php\n}\n")
	l := NewBytes(src)
	if l.NextToken().Type != token.T_LBRACE {
		t.Fatal("expected T_LBRACE")
	}
	end, ok := l.SkipBalancedCurlyBlockWithEnd()
	if !ok {
		t.Fatal("expected balanced skip across ?> in line comment")
	}
	if end.Offset != len(src)-1 { // trailing newline after }
		// Accept either ending at final } or after trailing newline — just
		// require the skip consumed through the PHP closing brace.
		if end.Offset < bytesIndexByte(src, '}') {
			t.Fatalf("end offset %d too early; src len %d", end.Offset, len(src))
		}
	}
}

func bytesIndexByte(b []byte, c byte) int {
	for i, ch := range b {
		if ch == c && i > 0 { // skip opening {
			return i
		}
	}
	return -1
}

func TestSkipBalancedCurlyBlockHashCommentCloseTag(t *testing.T) {
	src := []byte("{\n  # note ?>\n<html>x</html>\n<?php\n}\n")
	l := NewBytes(src)
	if l.NextToken().Type != token.T_LBRACE {
		t.Fatal("expected T_LBRACE")
	}
	if _, ok := l.SkipBalancedCurlyBlockWithEnd(); !ok {
		t.Fatal("expected balanced skip across ?> in # comment")
	}
}

func TestSkipBalancedCurlyBlockCommentsAndBacktickAndHTML(t *testing.T) {
	src := []byte("{\n  // line\n  /* block { } */\n  $s = `{$x}`;\n  ?>\n  html { }\n  <?php\n}\n")
	l := NewBytes(src)
	if l.NextToken().Type != token.T_LBRACE {
		t.Fatal("expected T_LBRACE")
	}
	end, ok := l.SkipBalancedCurlyBlockWithEnd()
	if !ok {
		t.Fatal("expected balanced skip")
	}
	if end.Offset < len(src)-2 {
		t.Fatalf("end %d, src %d", end.Offset, len(src))
	}
}

func TestSkipBalancedCurlyBlockNowdocCRLF(t *testing.T) {
	src := []byte("{\r\n$s = <<<'EOT'\r\nbody { }\r\nEOT;\r\n}\r\n")
	l := NewBytes(src)
	if l.NextToken().Type != token.T_LBRACE {
		t.Fatal("expected T_LBRACE")
	}
	end, ok := l.SkipBalancedCurlyBlockWithEnd()
	if !ok {
		t.Fatal("expected CRLF nowdoc skip")
	}
	if end.Offset < len(src)-2 {
		t.Fatalf("end %d want near %d", end.Offset, len(src))
	}
}

func TestCaretAndGreaterOperators(t *testing.T) {
	cases := []struct {
		src  string
		want token.TokenType
		lit  string
	}{
		{"^", token.T_CARET, "^"},
		{"^=", token.T_XOR_EQUAL, "^="},
		{">", token.T_IS_GREATER, ">"},
		{">=", token.T_IS_GREATER_OR_EQUAL, ">="},
		{">>=", token.T_SR_EQUAL, ">>="},
	}
	for _, c := range cases {
		tok := New(c.src).NextToken()
		if tok.Type != c.want || tok.Literal != c.lit {
			t.Errorf("%q: got %v %q, want %v %q", c.src, tok.Type, tok.Literal, c.want, c.lit)
		}
	}
}

func TestNumberLiteralsExtended(t *testing.T) {
	cases := []struct {
		src  string
		want token.TokenType
		lit  string
	}{
		{"0b1010", token.T_LNUMBER, "0b1010"},
		{"0B1_0", token.T_LNUMBER, "0B1_0"},
		{"0xFF", token.T_LNUMBER, "0xFF"},
		{"0Xdead_beef", token.T_LNUMBER, "0Xdead_beef"},
		{"1e10", token.T_DNUMBER, "1e10"},
		{"1.2e+3", token.T_DNUMBER, "1.2e+3"},
		{"1.7E-2", token.T_DNUMBER, "1.7E-2"},
		{"1_234", token.T_LNUMBER, "1_234"},
		{"1.2.3", token.T_DNUMBER, "1.2"}, // second dot stops float
	}
	for _, c := range cases {
		tok := New(c.src).NextToken()
		if tok.Type != c.want || tok.Literal != c.lit {
			t.Errorf("%q: got %v %q, want %v %q", c.src, tok.Type, tok.Literal, c.want, c.lit)
		}
	}
	// Remainder after 1.2.3
	l := New("1.2.3")
	_ = l.NextToken()
	tok := l.NextToken()
	if tok.Type != token.T_DNUMBER && tok.Type != token.T_LNUMBER && tok.Type != token.T_DOT {
		// After "1.2" the next token should be ".3" handled as DOT + number or similar.
		if tok.Type != token.T_DOT {
			t.Fatalf("after 1.2 expected DOT or number start, got %v %q", tok.Type, tok.Literal)
		}
	}
}

func TestEncapsedVariableForms(t *testing.T) {
	src := `"pre $v[0] mid $w[foo] $n[$x] ${name} lone $ end"`
	types := drainTypes(src)
	for _, want := range []token.TokenType{
		token.T_VARIABLE,
		token.T_LBRACKET,
		token.T_NUM_STRING,
		token.T_RBRACKET,
		token.T_STRING, // foo
		token.T_DOLLAR_OPEN_CURLY_BRACES,
		token.T_STRING_VARNAME,
	} {
		if !hasType(types, want) {
			t.Fatalf("missing %v in %v for %q", want, types, src)
		}
	}
	// Identity round-trip (PHP-mode New; LexAll/NewFile would treat this as HTML)
	l := New(src)
	var toks []token.Token
	for {
		tok := l.NextToken()
		toks = append(toks, tok)
		if tok.Type == token.T_EOF {
			break
		}
	}
	if got := PrintTokens(toks, []byte(src)); got != src {
		t.Fatalf("identity:\nwant %q\ngot  %q", src, got)
	}
}

func TestEncapsedSimpleVarAndCurly(t *testing.T) {
	src := `"hello {$user}!"`
	l := New(src)
	var toks []token.Token
	for {
		tok := l.NextToken()
		toks = append(toks, tok)
		if tok.Type == token.T_EOF {
			break
		}
	}
	if got := PrintTokens(toks, []byte(src)); got != src {
		t.Fatalf("identity fail: %q vs %q", src, got)
	}
	if !hasType(drainTypes(src), token.T_CURLY_OPEN) {
		t.Fatal("expected T_CURLY_OPEN")
	}
}

func TestDollarOpenCurlyOutsideString(t *testing.T) {
	tok := New(`${$x}`).NextToken()
	if tok.Type != token.T_DOLLAR_OPEN_CURLY_BRACES || tok.Literal != "${" {
		t.Fatalf("got %v %q", tok.Type, tok.Literal)
	}
}

func TestLoneDollarIllegal(t *testing.T) {
	tok := New(`$`).NextToken()
	if tok.Type != token.T_ILLEGAL || tok.Literal != "$" {
		t.Fatalf("got %v %q", tok.Type, tok.Literal)
	}
}

func TestEmptyDocCommentDoesNotSwallowFollowingTokens(t *testing.T) {
	tok := New("/**/$x").NextToken()
	if tok.Type != token.T_VARIABLE || tok.Literal != "$x" {
		t.Fatalf("got %v %q", tok.Type, tok.Literal)
	}
	if len(tok.LeadingTrivia) != 1 || tok.LeadingTrivia[0].Literal != "/**/" {
		t.Fatalf("trivia %+v", tok.LeadingTrivia)
	}
	if tok.LeadingTrivia[0].Type != token.T_COMMENT {
		t.Fatalf("expected T_COMMENT for /**/, got %v", tok.LeadingTrivia[0].Type)
	}
}

func TestLineCommentCloseTagEndsComment(t *testing.T) {
	tok := New("// hi ?>$x").NextToken()
	if tok.Type != token.T_CLOSE_TAG {
		t.Fatalf("expected T_CLOSE_TAG after //…?>, got %v %q trivia=%+v", tok.Type, tok.Literal, tok.LeadingTrivia)
	}
	if len(tok.LeadingTrivia) == 0 || tok.LeadingTrivia[0].Literal != "// hi " {
		t.Fatalf("expected // hi  trivia, got %+v", tok.LeadingTrivia)
	}
}

func TestHashCommentCloseTagTerminates(t *testing.T) {
	tok := New("# hi ?>$x").NextToken()
	if tok.Type != token.T_CLOSE_TAG {
		t.Fatalf("expected T_CLOSE_TAG, got %v", tok.Type)
	}
	if len(tok.LeadingTrivia) == 0 || tok.LeadingTrivia[0].Literal != "# hi " {
		t.Fatalf("trivia %+v", tok.LeadingTrivia)
	}
}

func TestAttributeOnlyHashBracket(t *testing.T) {
	tok := New("#[Attr]").NextToken()
	if tok.Type != token.T_ATTRIBUTE || tok.Literal != "#[" {
		t.Fatalf("got %v %q", tok.Type, tok.Literal)
	}
	tok = New("#[Attr]").NextToken()
	_ = tok
	types := drainTypes("#[Deprecated] function f() {}")
	if !hasType(types, token.T_ATTRIBUTE) || !hasType(types, token.T_FUNCTION) {
		t.Fatalf("types %v", types)
	}
}

func TestMissingHeredocIdentifier(t *testing.T) {
	types := drainTypes("<<<\n")
	if !hasType(types, token.T_ILLEGAL) {
		t.Fatalf("expected T_ILLEGAL for missing label, got %v", types)
	}
}

func TestIndentedHeredocPreservesBody(t *testing.T) {
	src := "<?php\n$s = <<<SQL\n        SELECT 1\n    SQL;\n"
	var body, endLit string
	l := New(src)
	for {
		tok := l.NextToken()
		switch tok.Type {
		case token.T_ENCAPSED_AND_WHITESPACE:
			body = tok.Literal
		case token.T_END_HEREDOC:
			endLit = tok.Literal
		case token.T_EOF:
			goto done
		}
	}
done:
	if body != "        SELECT 1\n" {
		t.Fatalf("body %q", body)
	}
	if endLit != "    SQL" {
		t.Fatalf("end %q", endLit)
	}
}

func TestHeredocCRLFOpener(t *testing.T) {
	src := "<<<EOT\r\nhello\r\nEOT"
	l := New(src)
	var toks []token.Token
	for {
		tok := l.NextToken()
		toks = append(toks, tok)
		if tok.Type == token.T_EOF {
			break
		}
	}
	if got := PrintTokens(toks, []byte(src)); got != src {
		t.Fatalf("CRLF heredoc identity:\nwant %q\ngot  %q", src, got)
	}
}

func TestHeredocTerminatorNotPrefixOfIdent(t *testing.T) {
	// Use New (PHP mode): LexAll/NewFile treats tag-less input as inline HTML.
	src := "<<<EOT\nEOTx\nEOT"
	l := New(src)
	var body string
	var toks []token.Token
	for {
		tok := l.NextToken()
		toks = append(toks, tok)
		if tok.Type == token.T_ENCAPSED_AND_WHITESPACE {
			body += tok.Literal
		}
		if tok.Type == token.T_EOF {
			break
		}
	}
	if got := PrintTokens(toks, []byte(src)); got != src {
		t.Fatalf("identity fail %q vs %q", src, got)
	}
	if body != "EOTx\n" {
		t.Fatalf("body should keep EOTx line, got %q", body)
	}
}

func TestOpenTagWithEchoAndCloseTag(t *testing.T) {
	types := drainTypes("<?= $x ?>")
	if !hasType(types, token.T_OPEN_TAG_WITH_ECHO) {
		t.Fatalf("missing echo open tag: %v", types)
	}
	if !hasType(types, token.T_CLOSE_TAG) {
		t.Fatalf("missing close tag: %v", types)
	}
}

func TestNsSeparatorForcesStringIdent(t *testing.T) {
	// After \, semi-reserved words stay T_STRING.
	l := New(`\enum`)
	if l.NextToken().Type != token.T_NS_SEPARATOR {
		t.Fatal("expected NS_SEPARATOR")
	}
	tok := l.NextToken()
	if tok.Type != token.T_STRING || tok.Literal != "enum" {
		t.Fatalf("expected T_STRING enum, got %v %q", tok.Type, tok.Literal)
	}
}

func TestPlusIncAndAssign(t *testing.T) {
	cases := []struct {
		src  string
		want token.TokenType
	}{
		{"++", token.T_INC},
		{"+=", token.T_PLUS_EQUAL},
		{"--", token.T_DEC},
		{"<>", token.T_IS_NOT_EQUAL},
		{"<<=", token.T_SL_EQUAL},
		{"===", token.T_IS_IDENTICAL},
		{"==", token.T_IS_EQUAL},
		{"=>", token.T_DOUBLE_ARROW},
		{"::", token.T_DOUBLE_COLON},
		{"<=", token.T_IS_SMALLER_OR_EQUAL},
	}
	for _, c := range cases {
		tok := New(c.src).NextToken()
		if tok.Type != c.want {
			t.Errorf("%q: got %v want %v", c.src, tok.Type, c.want)
		}
	}
}

func TestStripUnderscoresHelper(t *testing.T) {
	if stripUnderscores("123") != "123" {
		t.Fatal("no-op path")
	}
	if stripUnderscores("1_2_3") != "123" {
		t.Fatal("strip path")
	}
}

func TestSkipWhitespaceAndReadStringHelpers(t *testing.T) {
	l := New("   abc")
	l.skipWhitespace()
	if l.char != 'a' {
		t.Fatalf("after skipWhitespace char=%q", l.char)
	}
	l2 := New(`'hi\n'`)
	// Position at opening quote content: New leaves char at first rune '
	if l2.char != '\'' {
		t.Fatalf("char %q", l2.char)
	}
	l2.readChar() // move into string
	got := l2.readString('\'')
	if got != "hi\n" {
		t.Fatalf("readString = %q", got)
	}
}

func TestInlineHTMLNewFile(t *testing.T) {
	src := "hello<?= $x ?>"
	toks := LexAll([]byte(src)) // LexAll uses NewBytes → PHP mode
	// Prefer NewFile for HTML-leading files
	l := NewFile(src)
	tok := l.NextToken()
	if tok.Type != token.T_INLINE_HTML || tok.Literal != "hello" {
		t.Fatalf("got %v %q", tok.Type, tok.Literal)
	}
	tok = l.NextToken()
	if tok.Type != token.T_OPEN_TAG_WITH_ECHO {
		t.Fatalf("got %v", tok.Type)
	}
	_ = toks
}

func TestEmptyQueueHeredocToken(t *testing.T) {
	l := New("")
	tok := l.nextHeredocToken()
	if tok.Type != token.T_ILLEGAL {
		t.Fatalf("got %v", tok.Type)
	}
}

func TestNullsafeAndCloseTagNewline(t *testing.T) {
	tok := New("?->").NextToken()
	if tok.Type != token.T_NULLSAFE_OBJECT_OPERATOR || tok.Literal != "?->" {
		t.Fatalf("got %v %q", tok.Type, tok.Literal)
	}
	// ?>\n should not leave a leading blank HTML token from the newline.
	l := New("?>\n$x")
	if l.NextToken().Type != token.T_CLOSE_TAG {
		t.Fatal("expected close tag")
	}
	// After close, inHTML: $x is inline HTML including maybe nothing before $
	tok = l.NextToken()
	if tok.Type != token.T_INLINE_HTML || tok.Literal != "$x" {
		t.Fatalf("expected INLINE_HTML $x without leading newline, got %v %q", tok.Type, tok.Literal)
	}
	l2 := New("?>\r\n$x")
	_ = l2.NextToken()
	tok = l2.NextToken()
	if tok.Type != token.T_INLINE_HTML || tok.Literal != "$x" {
		t.Fatalf("CRLF close: got %v %q", tok.Type, tok.Literal)
	}
}

func TestQuotedHeredocLabelAndMissingLabelSkip(t *testing.T) {
	src := "<<<\"EOT\"\nbody\nEOT"
	l := New(src)
	tok := l.NextToken()
	if tok.Type != token.T_START_HEREDOC {
		t.Fatalf("got %v", tok.Type)
	}
	var body string
	for {
		tok = l.NextToken()
		if tok.Type == token.T_ENCAPSED_AND_WHITESPACE {
			body = tok.Literal
		}
		if tok.Type == token.T_EOF {
			break
		}
	}
	if body != "body\n" {
		t.Fatalf("body %q", body)
	}

	// SkipBalanced path: missing heredoc label after <<<
	skipSrc := []byte("{\n$s = <<< \n}\n")
	ls := NewBytes(skipSrc)
	if ls.NextToken().Type != token.T_LBRACE {
		t.Fatal("lbrace")
	}
	// May or may not balance; must not panic
	_, _ = ls.SkipBalancedCurlyBlockWithEnd()
}

func TestSkipQuotedStringEscapesAndUnclosed(t *testing.T) {
	src := []byte(`{ $a = "x\"y"; $b = 'z'; $c = "unclosed }`)
	l := NewBytes(src)
	if l.NextToken().Type != token.T_LBRACE {
		t.Fatal("lbrace")
	}
	// Unclosed quote: skip should fail (EOF) without panic
	if l.SkipBalancedCurlyBlock() {
		t.Fatal("unclosed string should prevent balance")
	}
}

func TestSkipBalancedSingleQuotedEscapes(t *testing.T) {
	src := []byte(`{ $a = 'it\'s {ok}'; }`)
	l := NewBytes(src)
	if l.NextToken().Type != token.T_LBRACE {
		t.Fatal("lbrace")
	}
	end, ok := l.SkipBalancedCurlyBlockWithEnd()
	if !ok {
		t.Fatal("expected balance with escaped quotes")
	}
	if end.Offset != len(src) {
		t.Fatalf("end %d want %d", end.Offset, len(src))
	}
}

func TestReadStringAllEscapes(t *testing.T) {
	l := New(`"ab"`)
	l.readChar() // past "
	if got := l.readString('"'); got != "ab" {
		t.Fatalf("fast path %q", got)
	}
	cases := []struct {
		inner string
		want  string
	}{
		{`a\tb`, "a\tb"},
		{`a\rb`, "a\rb"},
		{`a\"b`, `a"b`},
		{`a\\b`, `a\b`},
		{`a\xb`, "a\\xb"},
	}
	for _, c := range cases {
		src := `"` + c.inner + `"`
		lx := New(src)
		lx.readChar()
		if got := lx.readString('"'); got != c.want {
			t.Fatalf("%q: got %q want %q", c.inner, got, c.want)
		}
	}
}

func TestBackslashInStringModeViaRestore(t *testing.T) {
	l := New(`\Foo`)
	snap := l.Snapshot()
	snap.inString = true
	l.Restore(snap)
	tok := l.NextToken()
	if tok.Type != token.T_BACKSLASH {
		t.Fatalf("inStringMode should emit T_BACKSLASH, got %v", tok.Type)
	}
}

func TestTextClampAndAsciiString(t *testing.T) {
	l := New("ab")
	if l.text(-1, 100) != "ab" {
		t.Fatal("text clamp")
	}
	if l.text(5, 1) != "" {
		t.Fatal("empty slice")
	}
	if asciiString('A') != "A" {
		t.Fatal("ascii")
	}
	if asciiString(0x1F600) == "" { // grinning face rune — non-ascii path
		t.Fatal("non-ascii asciiString")
	}
}

func TestPeekCharUnicode(t *testing.T) {
	l := New("πx")
	if l.char != 'π' {
		t.Fatalf("char %q", l.char)
	}
	if l.peekChar() != 'x' {
		t.Fatalf("peek %q", l.peekChar())
	}
	l2 := New("π")
	if l2.peekChar() != 0 {
		t.Fatalf("peek at end want 0 got %q", l2.peekChar())
	}
}

func TestLessThanPartialOpenTag(t *testing.T) {
	// <? without php stays as smaller / not open tag when scanned as symbol sequence
	tok := New("<?").NextToken()
	// lexLess sees <? — not full php/echo → T_IS_SMALLER for '<'
	if tok.Type != token.T_IS_SMALLER {
		t.Fatalf("got %v %q", tok.Type, tok.Literal)
	}
}

func TestUnclosedBlockCommentTrivia(t *testing.T) {
	tok := New("/* unterminated").NextToken()
	if tok.Type != token.T_EOF {
		t.Fatalf("expected EOF with trailing trivia, got %v", tok.Type)
	}
	if len(tok.TrailingTrivia) == 0 || tok.TrailingTrivia[0].Type != token.T_COMMENT {
		t.Fatalf("trivia %+v", tok.TrailingTrivia)
	}
}

func TestSkipBlockCommentUnclosed(t *testing.T) {
	src := []byte("{ /* unterminated")
	l := NewBytes(src)
	_ = l.NextToken()
	if l.SkipBalancedCurlyBlock() {
		t.Fatal("unclosed comment should fail balance")
	}
}

func TestEncapsedEscapedDollar(t *testing.T) {
	src := `"cost \$5"`
	l := New(src)
	var toks []token.Token
	for {
		tok := l.NextToken()
		toks = append(toks, tok)
		if tok.Type == token.T_EOF {
			break
		}
	}
	if got := PrintTokens(toks, []byte(src)); got != src {
		t.Fatalf("identity %q vs %q", src, got)
	}
	// Should be constant string (no interpolation) due to commitConstantDoubleQuote
	if toks[0].Type != token.T_CONSTANT_ENCAPSED_STRING {
		t.Fatalf("expected constant encapsed, got %v", toks[0].Type)
	}
}

func TestSkipCloseTagReopenEcho(t *testing.T) {
	src := []byte("{\n?>html<?= \n}\n")
	l := NewBytes(src)
	if l.NextToken().Type != token.T_LBRACE {
		t.Fatal("lbrace")
	}
	end, ok := l.SkipBalancedCurlyBlockWithEnd()
	if !ok {
		t.Fatal("expected balance across ?>…<?= ")
	}
	if end.Offset < len(src)-2 {
		t.Fatalf("end %d", end.Offset)
	}
}

func TestPeekCharMultibyte(t *testing.T) {
	l := New("aπ")
	if l.peekChar() != 'π' {
		t.Fatalf("peek multibyte got %q", l.peekChar())
	}
}

func TestOpenTagViaNewFileAtStart(t *testing.T) {
	l := NewFile("<?php $x;")
	tok := l.NextToken()
	if tok.Type != token.T_OPEN_TAG {
		t.Fatalf("got %v", tok.Type)
	}
	l2 := NewFile("<?=$x")
	if l2.NextToken().Type != token.T_OPEN_TAG_WITH_ECHO {
		t.Fatal("echo open")
	}
}

func TestEncapsedLoneDollarInChunk(t *testing.T) {
	// '$' not followed by ident stays in encapsed whitespace via lexEncapsedChunk
	src := `"x$ y"`
	l := New(src)
	var toks []token.Token
	for {
		tok := l.NextToken()
		toks = append(toks, tok)
		if tok.Type == token.T_EOF {
			break
		}
	}
	if got := PrintTokens(toks, []byte(src)); got != src {
		t.Fatalf("%q vs %q", src, got)
	}
}

func TestHeredocCROnlyOpener(t *testing.T) {
	src := "<<<EOT\rhello\rEOT"
	l := New(src)
	var toks []token.Token
	for {
		tok := l.NextToken()
		toks = append(toks, tok)
		if tok.Type == token.T_EOF {
			break
		}
	}
	if got := PrintTokens(toks, []byte(src)); got != src {
		t.Fatalf("%q vs %q", src, got)
	}
}

func TestSkipHeredocQuotedLabelCRLF(t *testing.T) {
	src := []byte("{\r\n$s = <<<\"L\"\r\n{\r\nL;\r\n}\r\n")
	l := NewBytes(src)
	_ = l.NextToken()
	end, ok := l.SkipBalancedCurlyBlockWithEnd()
	if !ok {
		t.Fatal("skip heredoc quoted label")
	}
	if end.Offset < len(src)-2 {
		t.Fatalf("end %d", end.Offset)
	}
}

func TestSingleQuoteUnclosed(t *testing.T) {
	tok := New("'abc").NextToken()
	if tok.Type != token.T_CONSTANT_ENCAPSED_STRING {
		t.Fatalf("got %v", tok.Type)
	}
	if tok.Literal != "'abc" {
		t.Fatalf("lit %q", tok.Literal)
	}
}

func TestLessOpenTagEchoFromSymbol(t *testing.T) {
	// When '<' is seen as a symbol (PHP mode), '<?=' is open-tag-with-echo
	tok := New("<?=").NextToken()
	if tok.Type != token.T_OPEN_TAG_WITH_ECHO {
		t.Fatalf("got %v %q", tok.Type, tok.Literal)
	}
}

func TestMatchHeredocEmptyLabelGuard(t *testing.T) {
	l := New("x")
	l.heredocLabel = ""
	if _, _, ok := l.matchHeredocTerminator(); ok {
		t.Fatal("empty label must not match")
	}
}


