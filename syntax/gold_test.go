package syntax

import (
	"strings"
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
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
		{
			src: "function($x) use ($y) { return $x; }",
			want: "" +
				"ClosureExpr\n" +
				"  Token T_FUNCTION \"function\"\n" +
				"  Token T_LPAREN \"(\"\n" +
				"  ParamList\n" +
				"    Param\n" +
				"      Token T_VARIABLE \"$x\"\n" +
				"  Token T_RPAREN \")\"\n" +
				"  ClosureUseClause\n" +
				"    Token T_USE \"use\"\n" +
				"    Token T_LPAREN \"(\"\n" +
				"    Token T_VARIABLE \"$y\"\n" +
				"    Token T_RPAREN \")\"\n" +
				"  StatementList\n" +
				"    Token T_LBRACE \"{\"\n" +
				"    ReturnStmt\n" +
				"      Token T_RETURN \"return\"\n" +
				"      VariableExpr\n" +
				"        Token T_VARIABLE \"$x\"\n" +
				"      Token T_SEMICOLON \";\"\n" +
				"    Token T_RBRACE \"}\"\n",
		},
		{
			src: "fn($x): int => $x + 1",
			want: "" +
				"ArrowFunctionExpr\n" +
				"  Token T_FN \"fn\"\n" +
				"  Token T_LPAREN \"(\"\n" +
				"  ParamList\n" +
				"    Param\n" +
				"      Token T_VARIABLE \"$x\"\n" +
				"  Token T_RPAREN \")\"\n" +
				"  Token T_COLON \":\"\n" +
				"  PrimitiveType\n" +
				"    Token T_STRING \"int\"\n" +
				"  Token T_DOUBLE_ARROW \"=>\"\n" +
				"  BinaryExpr\n" +
				"    VariableExpr\n" +
				"      Token T_VARIABLE \"$x\"\n" +
				"    Token T_PLUS \"+\"\n" +
				"    LiteralExpr\n" +
				"      Token T_LNUMBER \"1\"\n",
		},
		{
			src: "new class($a) extends Base { public $x; }",
			want: "" +
				"NewExpr\n" +
				"  Token T_NEW \"new\"\n" +
				"  AnonymousClass\n" +
				"    Token T_CLASS \"class\"\n" +
				"    ArgList\n" +
				"      Token T_LPAREN \"(\"\n" +
				"      Arg\n" +
				"        VariableExpr\n" +
				"          Token T_VARIABLE \"$a\"\n" +
				"      Token T_RPAREN \")\"\n" +
				"    ExtendsClause\n" +
				"      Token T_EXTENDS \"extends\"\n" +
				"      UnqualifiedName\n" +
				"        Token T_STRING \"Base\"\n" +
				"    MemberList\n" +
				"      Token T_LBRACE \"{\"\n" +
				"      PropertyDecl\n" +
				"        ModifierList\n" +
				"          Token T_PUBLIC \"public\"\n" +
				"        Token T_VARIABLE \"$x\"\n" +
				"        Token T_SEMICOLON \";\"\n" +
				"      Token T_RBRACE \"}\"\n",
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

// TestGoldTreeElvisTernary locks the short-ternary shape: the middle operand
// is absent (no second expr child between ? and :), unlike the full ternary
// covered in TestGoldTreeExpressions.
func TestGoldTreeElvisTernary(t *testing.T) {
	src := "<?php $a = $b ?: $c;\n"
	res := Parse([]byte(src))
	if Print(res.File.Root) != src {
		t.Fatalf("identity failed\nwant %q\ngot  %q", src, Print(res.File.Root))
	}
	got := dumpGoldTree(res.File.Root)
	want := "" +
		"File\n" +
		"  Token T_OPEN_TAG \"<?php\"\n" +
		"  ExpressionStmt\n" +
		"    AssignExpr\n" +
		"      VariableExpr\n" +
		"        Token T_VARIABLE \"$a\"\n" +
		"      Token T_ASSIGN \"=\"\n" +
		"      TernaryExpr\n" +
		"        VariableExpr\n" +
		"          Token T_VARIABLE \"$b\"\n" +
		"        Token T_QUESTION \"?\"\n" +
		"        Token T_COLON \":\"\n" +
		"        VariableExpr\n" +
		"          Token T_VARIABLE \"$c\"\n" +
		"    Token T_SEMICOLON \";\"\n" +
		"  Token T_EOF\n"
	if got != want {
		t.Fatalf("gold elvis tree mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

// TestGoldTreeYieldForms locks every yield variant: bare, value, key=>value,
// and delegated `yield from` (T_YIELD + T_STRING("from") + value).
func TestGoldTreeYieldForms(t *testing.T) {
	src := "<?php function gen() { yield; yield $v; yield $k => $v; yield from gen(); }\n"
	res := Parse([]byte(src))
	if Print(res.File.Root) != src {
		t.Fatalf("identity failed\nwant %q\ngot  %q", src, Print(res.File.Root))
	}
	got := dumpGoldTree(res.File.Root)
	want := "" +
		"File\n" +
		"  Token T_OPEN_TAG \"<?php\"\n" +
		"  FunctionDecl\n" +
		"    Token T_FUNCTION \"function\"\n" +
		"    UnqualifiedName\n" +
		"      Token T_STRING \"gen\"\n" +
		"    Token T_LPAREN \"(\"\n" +
		"    ParamList\n" +
		"    Token T_RPAREN \")\"\n" +
		"    StatementList\n" +
		"      Token T_LBRACE \"{\"\n" +
		"      ExpressionStmt\n" +
		"        YieldExpr\n" +
		"          Token T_YIELD \"yield\"\n" +
		"        Token T_SEMICOLON \";\"\n" +
		"      ExpressionStmt\n" +
		"        YieldExpr\n" +
		"          Token T_YIELD \"yield\"\n" +
		"          VariableExpr\n" +
		"            Token T_VARIABLE \"$v\"\n" +
		"        Token T_SEMICOLON \";\"\n" +
		"      ExpressionStmt\n" +
		"        YieldExpr\n" +
		"          Token T_YIELD \"yield\"\n" +
		"          VariableExpr\n" +
		"            Token T_VARIABLE \"$k\"\n" +
		"          Token T_DOUBLE_ARROW \"=>\"\n" +
		"          VariableExpr\n" +
		"            Token T_VARIABLE \"$v\"\n" +
		"        Token T_SEMICOLON \";\"\n" +
		"      ExpressionStmt\n" +
		"        YieldExpr\n" +
		"          Token T_YIELD \"yield\"\n" +
		"          Token T_STRING \"from\"\n" +
		"          CallExpr\n" +
		"            UnqualifiedName\n" +
		"              Token T_STRING \"gen\"\n" +
		"            ArgList\n" +
		"              Token T_LPAREN \"(\"\n" +
		"              Token T_RPAREN \")\"\n" +
		"        Token T_SEMICOLON \";\"\n" +
		"      Token T_RBRACE \"}\"\n" +
		"  Token T_EOF\n"
	if got != want {
		t.Fatalf("gold yield tree mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

// TestGoldTreeDynamicBracedMembers locks dynamic/braced member shapes:
// $o->$x (bare variable), $o->{$y} and Foo::{$z} (ParenExpr braces).
func TestGoldTreeDynamicBracedMembers(t *testing.T) {
	src := "<?php $o->$x; $o->{$y}; Foo::{$z};\n"
	res := Parse([]byte(src))
	if Print(res.File.Root) != src {
		t.Fatalf("identity failed\nwant %q\ngot  %q", src, Print(res.File.Root))
	}
	got := dumpGoldTree(res.File.Root)
	want := "" +
		"File\n" +
		"  Token T_OPEN_TAG \"<?php\"\n" +
		"  ExpressionStmt\n" +
		"    MemberAccessExpr\n" +
		"      VariableExpr\n" +
		"        Token T_VARIABLE \"$o\"\n" +
		"      Token T_OBJECT_OPERATOR \"->\"\n" +
		"      Token T_VARIABLE \"$x\"\n" +
		"    Token T_SEMICOLON \";\"\n" +
		"  ExpressionStmt\n" +
		"    MemberAccessExpr\n" +
		"      VariableExpr\n" +
		"        Token T_VARIABLE \"$o\"\n" +
		"      Token T_OBJECT_OPERATOR \"->\"\n" +
		"      ParenExpr\n" +
		"        Token T_LBRACE \"{\"\n" +
		"        VariableExpr\n" +
		"          Token T_VARIABLE \"$y\"\n" +
		"        Token T_RBRACE \"}\"\n" +
		"    Token T_SEMICOLON \";\"\n" +
		"  ExpressionStmt\n" +
		"    StaticMemberAccessExpr\n" +
		"      UnqualifiedName\n" +
		"        Token T_STRING \"Foo\"\n" +
		"      Token T_DOUBLE_COLON \"::\"\n" +
		"      ParenExpr\n" +
		"        Token T_LBRACE \"{\"\n" +
		"        VariableExpr\n" +
		"          Token T_VARIABLE \"$z\"\n" +
		"        Token T_RBRACE \"}\"\n" +
		"    Token T_SEMICOLON \";\"\n" +
		"  Token T_EOF\n"
	if got != want {
		t.Fatalf("gold dynamic-member tree mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

// TestGoldTreeFirstClassCallables locks `...` callables on plain, member,
// and static receivers.
func TestGoldTreeFirstClassCallables(t *testing.T) {
	src := "<?php $f = strlen(...); $g = $o->m(...); $h = Foo::m(...);\n"
	res := Parse([]byte(src))
	if Print(res.File.Root) != src {
		t.Fatalf("identity failed\nwant %q\ngot  %q", src, Print(res.File.Root))
	}
	got := dumpGoldTree(res.File.Root)
	want := "" +
		"File\n" +
		"  Token T_OPEN_TAG \"<?php\"\n" +
		"  ExpressionStmt\n" +
		"    AssignExpr\n" +
		"      VariableExpr\n" +
		"        Token T_VARIABLE \"$f\"\n" +
		"      Token T_ASSIGN \"=\"\n" +
		"      FirstClassCallableExpr\n" +
		"        UnqualifiedName\n" +
		"          Token T_STRING \"strlen\"\n" +
		"        Token T_LPAREN \"(\"\n" +
		"        Token T_ELLIPSIS \"...\"\n" +
		"        Token T_RPAREN \")\"\n" +
		"    Token T_SEMICOLON \";\"\n" +
		"  ExpressionStmt\n" +
		"    AssignExpr\n" +
		"      VariableExpr\n" +
		"        Token T_VARIABLE \"$g\"\n" +
		"      Token T_ASSIGN \"=\"\n" +
		"      FirstClassCallableExpr\n" +
		"        MemberAccessExpr\n" +
		"          VariableExpr\n" +
		"            Token T_VARIABLE \"$o\"\n" +
		"          Token T_OBJECT_OPERATOR \"->\"\n" +
		"          Token T_STRING \"m\"\n" +
		"        Token T_LPAREN \"(\"\n" +
		"        Token T_ELLIPSIS \"...\"\n" +
		"        Token T_RPAREN \")\"\n" +
		"    Token T_SEMICOLON \";\"\n" +
		"  ExpressionStmt\n" +
		"    AssignExpr\n" +
		"      VariableExpr\n" +
		"        Token T_VARIABLE \"$h\"\n" +
		"      Token T_ASSIGN \"=\"\n" +
		"      FirstClassCallableExpr\n" +
		"        StaticMemberAccessExpr\n" +
		"          UnqualifiedName\n" +
		"            Token T_STRING \"Foo\"\n" +
		"          Token T_DOUBLE_COLON \"::\"\n" +
		"          Token T_STRING \"m\"\n" +
		"        Token T_LPAREN \"(\"\n" +
		"        Token T_ELLIPSIS \"...\"\n" +
		"        Token T_RPAREN \")\"\n" +
		"    Token T_SEMICOLON \";\"\n" +
		"  Token T_EOF\n"
	if got != want {
		t.Fatalf("gold first-class-callable tree mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

// TestGoldTreeCasts locks the CastExpr shape (open, type token, close, operand).
func TestGoldTreeCasts(t *testing.T) {
	src := "<?php $x = (int)$y;\n"
	res := Parse([]byte(src))
	if Print(res.File.Root) != src {
		t.Fatalf("identity failed\nwant %q\ngot  %q", src, Print(res.File.Root))
	}
	got := dumpGoldTree(res.File.Root)
	want := "" +
		"File\n" +
		"  Token T_OPEN_TAG \"<?php\"\n" +
		"  ExpressionStmt\n" +
		"    AssignExpr\n" +
		"      VariableExpr\n" +
		"        Token T_VARIABLE \"$x\"\n" +
		"      Token T_ASSIGN \"=\"\n" +
		"      CastExpr\n" +
		"        Token T_LPAREN \"(\"\n" +
		"        Token T_STRING \"int\"\n" +
		"        Token T_RPAREN \")\"\n" +
		"        VariableExpr\n" +
		"          Token T_VARIABLE \"$y\"\n" +
		"    Token T_SEMICOLON \";\"\n" +
		"  Token T_EOF\n"
	if got != want {
		t.Fatalf("gold cast tree mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

// TestGoldTreeEnumCases locks attributed enum cases (valued + unit) and the
// backing-type header.
func TestGoldTreeEnumCases(t *testing.T) {
	src := "<?php enum E: string { #[Attr] case A = 'a'; case B; }\n"
	res := Parse([]byte(src))
	if Print(res.File.Root) != src {
		t.Fatalf("identity failed\nwant %q\ngot  %q", src, Print(res.File.Root))
	}
	got := dumpGoldTree(res.File.Root)
	want := "" +
		"File\n" +
		"  Token T_OPEN_TAG \"<?php\"\n" +
		"  EnumDecl\n" +
		"    Token T_ENUM \"enum\"\n" +
		"    UnqualifiedName\n" +
		"      Token T_STRING \"E\"\n" +
		"    Token T_COLON \":\"\n" +
		"    PrimitiveType\n" +
		"      Token T_STRING \"string\"\n" +
		"    MemberList\n" +
		"      Token T_LBRACE \"{\"\n" +
		"      AttributeList\n" +
		"        AttributeGroup\n" +
		"          Token T_ATTRIBUTE \"#[\"\n" +
		"          Attribute\n" +
		"            UnqualifiedName\n" +
		"              Token T_STRING \"Attr\"\n" +
		"          Token T_RBRACKET \"]\"\n" +
		"      EnumCase\n" +
		"        Token T_CASE \"case\"\n" +
		"        UnqualifiedName\n" +
		"          Token T_STRING \"A\"\n" +
		"        Token T_ASSIGN \"=\"\n" +
		"        LiteralExpr\n" +
		"          Token T_CONSTANT_ENCAPSED_STRING \"'a'\"\n" +
		"        Token T_SEMICOLON \";\"\n" +
		"      EnumCase\n" +
		"        Token T_CASE \"case\"\n" +
		"        UnqualifiedName\n" +
		"          Token T_STRING \"B\"\n" +
		"        Token T_SEMICOLON \";\"\n" +
		"      Token T_RBRACE \"}\"\n" +
		"  Token T_EOF\n"
	if got != want {
		t.Fatalf("gold enum-case tree mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

// TestGoldTreeClassConstAttributes locks attributes + typed multi-name class
// constants (the class-constant attribute/value hotspot).
func TestGoldTreeClassConstAttributes(t *testing.T) {
	src := "<?php class C { #[Attr] public const string FOO = 1, BAR = 2; }\n"
	res := Parse([]byte(src))
	if Print(res.File.Root) != src {
		t.Fatalf("identity failed\nwant %q\ngot  %q", src, Print(res.File.Root))
	}
	got := dumpGoldTree(res.File.Root)
	want := "" +
		"File\n" +
		"  Token T_OPEN_TAG \"<?php\"\n" +
		"  ClassDecl\n" +
		"    Token T_CLASS \"class\"\n" +
		"    UnqualifiedName\n" +
		"      Token T_STRING \"C\"\n" +
		"    MemberList\n" +
		"      Token T_LBRACE \"{\"\n" +
		"      AttributeList\n" +
		"        AttributeGroup\n" +
		"          Token T_ATTRIBUTE \"#[\"\n" +
		"          Attribute\n" +
		"            UnqualifiedName\n" +
		"              Token T_STRING \"Attr\"\n" +
		"          Token T_RBRACKET \"]\"\n" +
		"      ClassConstDecl\n" +
		"        ModifierList\n" +
		"          Token T_PUBLIC \"public\"\n" +
		"        Token T_CONST \"const\"\n" +
		"        PrimitiveType\n" +
		"          Token T_STRING \"string\"\n" +
		"        UnqualifiedName\n" +
		"          Token T_STRING \"FOO\"\n" +
		"        Token T_ASSIGN \"=\"\n" +
		"        LiteralExpr\n" +
		"          Token T_LNUMBER \"1\"\n" +
		"        Token T_COMMA \",\"\n" +
		"        UnqualifiedName\n" +
		"          Token T_STRING \"BAR\"\n" +
		"        Token T_ASSIGN \"=\"\n" +
		"        LiteralExpr\n" +
		"          Token T_LNUMBER \"2\"\n" +
		"        Token T_SEMICOLON \";\"\n" +
		"      Token T_RBRACE \"}\"\n" +
		"  Token T_EOF\n"
	if got != want {
		t.Fatalf("gold class-const tree mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

// TestGoldTreeNowdoc locks the Nowdoc shape: interpolated-looking `$x` stays
// a single StringPart (no VariablePart), unlike heredoc.
func TestGoldTreeNowdoc(t *testing.T) {
	src := "<?php\n$a = <<<'NOW'\nplain $x\nNOW;\n"
	res := Parse([]byte(src))
	if Print(res.File.Root) != src {
		t.Fatalf("identity failed\nwant %q\ngot  %q", src, Print(res.File.Root))
	}
	got := dumpGoldTree(res.File.Root)
	want := "" +
		"File\n" +
		"  Token T_OPEN_TAG \"<?php\"\n" +
		"  ExpressionStmt\n" +
		"    AssignExpr\n" +
		"      VariableExpr\n" +
		"        Token T_VARIABLE \"$a\"\n" +
		"      Token T_ASSIGN \"=\"\n" +
		"      Nowdoc\n" +
		"        Token T_START_NOWDOC \"<<<'NOW'\\n\"\n" +
		"        StringPart\n" +
		"          Token T_ENCAPSED_AND_WHITESPACE \"plain $x\\n\"\n" +
		"        Token T_END_NOWDOC \"NOW\"\n" +
		"    Token T_SEMICOLON \";\"\n" +
		"  Token T_EOF\n"
	if got != want {
		t.Fatalf("gold nowdoc tree mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

// TestGoldTreeMixedCaseKeywords locks that keyword tokens keep their original
// spelling while the tree shape stays canonical.
func TestGoldTreeMixedCaseKeywords(t *testing.T) {
	src := "<?php\nFunction FOO() {}\nIF ($a) { ECHO 1; }\n"
	res := Parse([]byte(src))
	if Print(res.File.Root) != src {
		t.Fatalf("identity failed\nwant %q\ngot  %q", src, Print(res.File.Root))
	}
	got := dumpGoldTree(res.File.Root)
	want := "" +
		"File\n" +
		"  Token T_OPEN_TAG \"<?php\"\n" +
		"  FunctionDecl\n" +
		"    Token T_FUNCTION \"Function\"\n" +
		"    UnqualifiedName\n" +
		"      Token T_STRING \"FOO\"\n" +
		"    Token T_LPAREN \"(\"\n" +
		"    ParamList\n" +
		"    Token T_RPAREN \")\"\n" +
		"    StatementList\n" +
		"      Token T_LBRACE \"{\"\n" +
		"      Token T_RBRACE \"}\"\n" +
		"  IfStmt\n" +
		"    Token T_IF \"IF\"\n" +
		"    Token T_LPAREN \"(\"\n" +
		"    VariableExpr\n" +
		"      Token T_VARIABLE \"$a\"\n" +
		"    Token T_RPAREN \")\"\n" +
		"    StatementList\n" +
		"      Token T_LBRACE \"{\"\n" +
		"      EchoStmt\n" +
		"        Token T_ECHO \"ECHO\"\n" +
		"        LiteralExpr\n" +
		"          Token T_LNUMBER \"1\"\n" +
		"        Token T_SEMICOLON \";\"\n" +
		"      Token T_RBRACE \"}\"\n" +
		"  Token T_EOF\n"
	if got != want {
		t.Fatalf("gold mixed-case tree mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

// TestGoldTreeDeclTriviaAttachment locks doc-comment trivia attachment on a
// declaration: the comment rides as LeadingTrivia on the declaration's first
// significant token and lowers to the declaration's PHPDoc.
func TestGoldTreeDeclTriviaAttachment(t *testing.T) {
	src := "<?php\n/** prop doc */\nclass C {\n/** prop doc */\npublic int $x;\n}\n"
	res := Parse([]byte(src))
	if Print(res.File.Root) != src {
		t.Fatalf("identity failed\nwant %q\ngot  %q", src, Print(res.File.Root))
	}
	var prop *RedNode
	Walk(res.File.Root, func(n *RedNode) bool {
		if n.Kind() == KindPropertyDecl {
			prop = copyRed(n)
			return false
		}
		return true
	})
	if prop == nil {
		t.Fatal("expected PropertyDecl")
	}
	first := firstSignificantToken(prop)
	if first == nil || first.Green == nil || !first.Green.IsToken() {
		t.Fatal("expected significant token under PropertyDecl")
	}
	tok, ok := first.Green.Token()
	if !ok {
		t.Fatal("expected token")
	}
	found := false
	for _, tr := range tok.LeadingTrivia {
		if tr.Type.String() == "T_DOC_COMMENT" && strings.Contains(tr.Literal, "prop doc") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected /** prop doc */ in LeadingTrivia, got %#v", tok.LeadingTrivia)
	}
	nodes := LowerAST(res)
	if len(nodes) == 0 {
		t.Fatal("expected lowered nodes")
	}
	cls, ok := nodes[0].(*ast.ClassNode)
	if !ok {
		t.Fatalf("expected *ast.ClassNode, got %T", nodes[0])
	}
	if len(cls.Properties) == 0 {
		t.Fatal("expected class properties")
	}
	propNode, ok := cls.Properties[0].(*ast.PropertyNode)
	if !ok {
		t.Fatalf("expected *ast.PropertyNode, got %T", cls.Properties[0])
	}
	if propNode.PHPDoc == nil {
		t.Fatalf("expected PHPDoc attached to property:\n%s", dumpGoldTree(res.File.Root))
	}
}
