package syntax

import (
	"strings"
	"testing"

	"github.com/ayanozturk/go-php-parser/token"
)

// dumpGoldTree is a test-only structured tree dump (kinds + significant token
// text). Uses Kind()/TokenType()/Token() so it stays resilient if green field
// layout changes under R1.
func dumpGoldTree(n *RedNode) string {
	var b strings.Builder
	writeGold(&b, n, 0)
	return b.String()
}

func writeGold(b *strings.Builder, n *RedNode, depth int) {
	if n == nil {
		return
	}
	indent := strings.Repeat("  ", depth)
	b.WriteString(indent)
	b.WriteString(n.Kind().String())
	if n.Green != nil && n.Green.IsToken() {
		tok, ok := n.Green.Token()
		if ok {
			b.WriteByte(' ')
			b.WriteString(n.Green.TokenType().String())
			text := significantTokenText(n, tok)
			if text != "" {
				b.WriteByte(' ')
				b.WriteString(quoteGold(text))
			}
		}
	}
	b.WriteByte('\n')
	for _, c := range n.Children() {
		writeGold(b, c, depth+1)
	}
}

func significantTokenText(n *RedNode, tok token.Token) string {
	if tok.Type == token.T_EOF {
		return ""
	}
	if n.File != nil {
		src := n.File.Source
		lead := 0
		for _, tr := range tok.LeadingTrivia {
			w := tr.Width()
			if w == 0 {
				w = len(tr.Literal)
			}
			lead += w
		}
		sig := tok.Width()
		if sig == 0 {
			sig = len(tok.Literal)
		}
		start := n.Offset + lead
		end := start + sig
		if start >= 0 && end <= len(src) && start <= end {
			return string(src[start:end])
		}
	}
	if tok.Literal != "" {
		return tok.Literal
	}
	return ""
}

func quoteGold(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '\\', '"':
			b.WriteByte('\\')
			b.WriteByte(c)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteByte(c)
		}
	}
	b.WriteByte('"')
	return b.String()
}

func mustParseFragment(t *testing.T, src string, parse func(*Parser) *GreenNode) *RedNode {
	t.Helper()
	p := NewParser([]byte(src))
	g := parse(p)
	if g == nil {
		t.Fatalf("nil green for %q", src)
	}
	f := &File{Source: []byte(src), Green: g, Lines: token.NewLineTable([]byte(src))}
	BindRed(f)
	if Print(f.Root) != src {
		t.Fatalf("identity failed for %q: got %q", src, Print(f.Root))
	}
	return f.Root
}

func TestGoldTreeAttributes(t *testing.T) {
	src := `#[\Foo\Bar(x: 1), Baz]`
	root := mustParseFragment(t, src, (*Parser).parseAttributeList)
	want := "" +
		"AttributeList\n" +
		"  AttributeGroup\n" +
		"    Token T_ATTRIBUTE \"#[\"\n" +
		"    Attribute\n" +
		"      FullyQualifiedName\n" +
		"        Token T_NS_SEPARATOR \"\\\\\"\n" +
		"        Token T_STRING \"Foo\"\n" +
		"        Token T_NS_SEPARATOR \"\\\\\"\n" +
		"        Token T_STRING \"Bar\"\n" +
		"      ArgList\n" +
		"        Token T_LPAREN \"(\"\n" +
		"        NamedArg\n" +
		"          Token T_STRING \"x\"\n" +
		"          Token T_COLON \":\"\n" +
		"          LiteralExpr\n" +
		"            Token T_LNUMBER \"1\"\n" +
		"        Token T_RPAREN \")\"\n" +
		"    Token T_COMMA \",\"\n" +
		"    Attribute\n" +
		"      UnqualifiedName\n" +
		"        Token T_STRING \"Baz\"\n" +
		"    Token T_RBRACKET \"]\"\n"
	got := dumpGoldTree(root)
	if got != want {
		t.Fatalf("gold attribute tree mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestGoldTreeDNFType(t *testing.T) {
	src := `(Foo&Bar)|null`
	root := mustParseFragment(t, src, (*Parser).parseType)
	want := "" +
		"UnionType\n" +
		"  ParenthesizedType\n" +
		"    Token T_LPAREN \"(\"\n" +
		"    IntersectionType\n" +
		"      NamedType\n" +
		"        UnqualifiedName\n" +
		"          Token T_STRING \"Foo\"\n" +
		"      Token T_AMPERSAND \"&\"\n" +
		"      NamedType\n" +
		"        UnqualifiedName\n" +
		"          Token T_STRING \"Bar\"\n" +
		"    Token T_RPAREN \")\"\n" +
		"  Token T_PIPE \"|\"\n" +
		"  PrimitiveType\n" +
		"    Token T_NULL \"null\"\n"
	got := dumpGoldTree(root)
	if got != want {
		t.Fatalf("gold DNF type tree mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestGoldTreeNames(t *testing.T) {
	cases := []struct {
		src  string
		want string
	}{
		{
			src: "Foo",
			want: "" +
				"UnqualifiedName\n" +
				"  Token T_STRING \"Foo\"\n",
		},
		{
			src: `Foo\Bar`,
			want: "" +
				"QualifiedName\n" +
				"  Token T_STRING \"Foo\"\n" +
				"  Token T_NS_SEPARATOR \"\\\\\"\n" +
				"  Token T_STRING \"Bar\"\n",
		},
		{
			src: `\Foo\Bar`,
			want: "" +
				"FullyQualifiedName\n" +
				"  Token T_NS_SEPARATOR \"\\\\\"\n" +
				"  Token T_STRING \"Foo\"\n" +
				"  Token T_NS_SEPARATOR \"\\\\\"\n" +
				"  Token T_STRING \"Bar\"\n",
		},
		{
			src: `namespace\Foo`,
			want: "" +
				"RelativeName\n" +
				"  Token T_NAMESPACE \"namespace\"\n" +
				"  Token T_NS_SEPARATOR \"\\\\\"\n" +
				"  Token T_STRING \"Foo\"\n",
		},
	}
	for _, tc := range cases {
		root := mustParseFragment(t, tc.src, (*Parser).parseName)
		got := dumpGoldTree(root)
		if got != tc.want {
			t.Fatalf("gold name %q\nwant:\n%s\ngot:\n%s", tc.src, tc.want, got)
		}
	}
}

func TestGoldTreeHeredocInterpolation(t *testing.T) {
	src := "<<<EOT\nhello $x\nEOT"
	root := mustParseFragment(t, src, (*Parser).parseHeredoc)
	want := "" +
		"Heredoc\n" +
		"  Token T_START_HEREDOC \"<<<EOT\\n\"\n" +
		"  StringPart\n" +
		"    Token T_ENCAPSED_AND_WHITESPACE \"hello \"\n" +
		"  VariablePart\n" +
		"    Token T_VARIABLE \"$x\"\n" +
		"  StringPart\n" +
		"    Token T_ENCAPSED_AND_WHITESPACE \"\\n\"\n" +
		"  Token T_END_HEREDOC \"EOT\"\n"
	got := dumpGoldTree(root)
	if got != want {
		t.Fatalf("gold heredoc tree mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestGoldTreeInterpolatedString(t *testing.T) {
	src := `"hello $x world"`
	root := mustParseFragment(t, src, (*Parser).parseInterpolatedString)
	want := "" +
		"StringLiteral\n" +
		"  Token T_CONSTANT_STRING \"\\\"\"\n" +
		"  StringPart\n" +
		"    Token T_ENCAPSED_AND_WHITESPACE \"hello \"\n" +
		"  VariablePart\n" +
		"    Token T_VARIABLE \"$x\"\n" +
		"  StringPart\n" +
		"    Token T_ENCAPSED_AND_WHITESPACE \" world\"\n" +
		"  Token T_CONSTANT_STRING \"\\\"\"\n"
	got := dumpGoldTree(root)
	if got != want {
		t.Fatalf("gold interpolated string tree mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestGoldTreeExpressions(t *testing.T) {
	cases := []struct {
		src  string
		want string
	}{
		{
			src: "$a + $b * $c",
			want: "" +
				"BinaryExpr\n" +
				"  VariableExpr\n" +
				"    Token T_VARIABLE \"$a\"\n" +
				"  Token T_PLUS \"+\"\n" +
				"  BinaryExpr\n" +
				"    VariableExpr\n" +
				"      Token T_VARIABLE \"$b\"\n" +
				"    Token T_MULTIPLY \"*\"\n" +
				"    VariableExpr\n" +
				"      Token T_VARIABLE \"$c\"\n",
		},
		{
			src: "$x = 1",
			want: "" +
				"AssignExpr\n" +
				"  VariableExpr\n" +
				"    Token T_VARIABLE \"$x\"\n" +
				"  Token T_ASSIGN \"=\"\n" +
				"  LiteralExpr\n" +
				"    Token T_LNUMBER \"1\"\n",
		},
		{
			src: "$o->m($a)",
			want: "" +
				"CallExpr\n" +
				"  MemberAccessExpr\n" +
				"    VariableExpr\n" +
				"      Token T_VARIABLE \"$o\"\n" +
				"    Token T_OBJECT_OPERATOR \"->\"\n" +
				"    Token T_STRING \"m\"\n" +
				"  ArgList\n" +
				"    Token T_LPAREN \"(\"\n" +
				"    Arg\n" +
				"      VariableExpr\n" +
				"        Token T_VARIABLE \"$a\"\n" +
				"    Token T_RPAREN \")\"\n",
		},
		{
			src: "$o?->p",
			want: "" +
				"NullsafeMemberAccessExpr\n" +
				"  VariableExpr\n" +
				"    Token T_VARIABLE \"$o\"\n" +
				"  Token T_NULLSAFE_OBJECT_OPERATOR \"?->\"\n" +
				"  Token T_STRING \"p\"\n",
		},
		{
			src: "Foo::BAR",
			want: "" +
				"StaticMemberAccessExpr\n" +
				"  UnqualifiedName\n" +
				"    Token T_STRING \"Foo\"\n" +
				"  Token T_DOUBLE_COLON \"::\"\n" +
				"  Token T_STRING \"BAR\"\n",
		},
		{
			src: "$a[0]",
			want: "" +
				"ArrayAccessExpr\n" +
				"  VariableExpr\n" +
				"    Token T_VARIABLE \"$a\"\n" +
				"  Token T_LBRACKET \"[\"\n" +
				"  LiteralExpr\n" +
				"    Token T_LNUMBER \"0\"\n" +
				"  Token T_RBRACKET \"]\"\n",
		},
		{
			src: "[1, 'k' => 2, ...$xs]",
			want: "" +
				"ArrayExpr\n" +
				"  Token T_LBRACKET \"[\"\n" +
				"  ArrayElement\n" +
				"    LiteralExpr\n" +
				"      Token T_LNUMBER \"1\"\n" +
				"  Token T_COMMA \",\"\n" +
				"  ArrayElement\n" +
				"    LiteralExpr\n" +
				"      Token T_CONSTANT_ENCAPSED_STRING \"'k'\"\n" +
				"    Token T_DOUBLE_ARROW \"=>\"\n" +
				"    LiteralExpr\n" +
				"      Token T_LNUMBER \"2\"\n" +
				"  Token T_COMMA \",\"\n" +
				"  ArrayElement\n" +
				"    Token T_ELLIPSIS \"...\"\n" +
				"    VariableExpr\n" +
				"      Token T_VARIABLE \"$xs\"\n" +
				"  Token T_RBRACKET \"]\"\n",
		},
		{
			src: "42",
			want: "" +
				"LiteralExpr\n" +
				"  Token T_LNUMBER \"42\"\n",
		},
		{
			src: "$v",
			want: "" +
				"VariableExpr\n" +
				"  Token T_VARIABLE \"$v\"\n",
		},
		{
			src: "new Foo(1)",
			want: "" +
				"NewExpr\n" +
				"  Token T_NEW \"new\"\n" +
				"  UnqualifiedName\n" +
				"    Token T_STRING \"Foo\"\n" +
				"  ArgList\n" +
				"    Token T_LPAREN \"(\"\n" +
				"    Arg\n" +
				"      LiteralExpr\n" +
				"        Token T_LNUMBER \"1\"\n" +
				"    Token T_RPAREN \")\"\n",
		},
		{
			src: "$a ? $b : $c",
			want: "" +
				"TernaryExpr\n" +
				"  VariableExpr\n" +
				"    Token T_VARIABLE \"$a\"\n" +
				"  Token T_QUESTION \"?\"\n" +
				"  VariableExpr\n" +
				"    Token T_VARIABLE \"$b\"\n" +
				"  Token T_COLON \":\"\n" +
				"  VariableExpr\n" +
				"    Token T_VARIABLE \"$c\"\n",
		},
		{
			src: "array(1, 2)",
			want: "" +
				"ArrayExpr\n" +
				"  Token T_ARRAY \"array\"\n" +
				"  Token T_LPAREN \"(\"\n" +
				"  ArrayElement\n" +
				"    LiteralExpr\n" +
				"      Token T_LNUMBER \"1\"\n" +
				"  Token T_COMMA \",\"\n" +
				"  ArrayElement\n" +
				"    LiteralExpr\n" +
				"      Token T_LNUMBER \"2\"\n" +
				"  Token T_RPAREN \")\"\n",
		},
		{
			src: "list($x, $y)",
			want: "" +
				"ListExpr\n" +
				"  Token T_LIST \"list\"\n" +
				"  Token T_LPAREN \"(\"\n" +
				"  ArrayElement\n" +
				"    VariableExpr\n" +
				"      Token T_VARIABLE \"$x\"\n" +
				"  Token T_COMMA \",\"\n" +
				"  ArrayElement\n" +
				"    VariableExpr\n" +
				"      Token T_VARIABLE \"$y\"\n" +
				"  Token T_RPAREN \")\"\n",
		},
		{
			src: "(int)$x",
			want: "" +
				"CastExpr\n" +
				"  Token T_LPAREN \"(\"\n" +
				"  Token T_STRING \"int\"\n" +
				"  Token T_RPAREN \")\"\n" +
				"  VariableExpr\n" +
				"    Token T_VARIABLE \"$x\"\n",
		},
		{
			src: "!$x",
			want: "" +
				"UnaryExpr\n" +
				"  Token T_NOT \"!\"\n" +
				"  VariableExpr\n" +
				"    Token T_VARIABLE \"$x\"\n",
		},
		{
			src: "foo(...)",
			want: "" +
				"FirstClassCallableExpr\n" +
				"  UnqualifiedName\n" +
				"    Token T_STRING \"foo\"\n" +
				"  Token T_LPAREN \"(\"\n" +
				"  Token T_ELLIPSIS \"...\"\n" +
				"  Token T_RPAREN \")\"\n",
		},
		{
			src: "isset($a)",
			want: "" +
				"CallExpr\n" +
				"  Token T_ISSET \"isset\"\n" +
				"  ArgList\n" +
				"    Token T_LPAREN \"(\"\n" +
				"    Arg\n" +
				"      VariableExpr\n" +
				"        Token T_VARIABLE \"$a\"\n" +
				"    Token T_RPAREN \")\"\n",
		},
	}
	for _, tc := range cases {
		root := mustParseFragment(t, tc.src, (*Parser).parseExpression)
		got := dumpGoldTree(root)
		if got != tc.want {
			t.Fatalf("gold expression %q\nwant:\n%s\ngot:\n%s", tc.src, tc.want, got)
		}
	}
}

func TestGoldTreeControlFlow(t *testing.T) {
	src := "<?php\nif ($a) { echo $a; } else { echo 0; }\nwhile ($i) { break; }\n"
	res := Parse([]byte(src))
	if Print(res.File.Root) != src {
		t.Fatalf("identity failed\nwant %q\ngot  %q", src, Print(res.File.Root))
	}
	got := dumpGoldTree(res.File.Root)
	// Assert structured control-flow shapes (gold excerpt), not mere kind presence.
	needles := []string{
		"IfStmt",
		"ElseClause",
		"WhileStmt",
		"BreakStmt",
		"EchoStmt",
		"Token T_IF \"if\"",
		"Token T_ELSE \"else\"",
		"Token T_WHILE \"while\"",
		"Token T_BREAK \"break\"",
	}
	for _, n := range needles {
		if !strings.Contains(got, n) {
			t.Fatalf("gold control-flow dump missing %q\n%s", n, got)
		}
	}
	// Nested shape: IfStmt contains ElseClause; WhileStmt contains BreakStmt.
	if !strings.Contains(got, "IfStmt\n") || !strings.Contains(got, "  ElseClause\n") {
		t.Fatalf("expected ElseClause nested under IfStmt:\n%s", got)
	}
	if !strings.Contains(got, "WhileStmt\n") || !strings.Contains(got, "  BreakStmt\n") && !strings.Contains(got, "    BreakStmt\n") {
		t.Fatalf("expected BreakStmt under WhileStmt:\n%s", got)
	}
}
