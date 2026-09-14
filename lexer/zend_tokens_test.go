package lexer

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/ayanozturk/go-php-parser/token"
)

func TestTokenStreamIdentityRoundTrip(t *testing.T) {
	fixtures := []string{
		"<?php\n$a = 1;\n",
		"<?php\n#[Attr]\nclass C {}\n",
		"<?php\n#[\\Foo\\Bar(x: 1)]\nfunction f(string $a) {}\n",
		"<?php\n/** doc */\nFunction f() {}\n",
		"<?php\n$a = \"plain\";\n$b = 'x';\n",
		"<?php\n$a = \"hello $x world\";\n",
		"<?php\n$a = <<<EOT\nhello $x\nEOT;\n",
		"<?php\n$a = <<<'EOT'\nhello $x\nEOT;\n",
		"<?php\n$a = <<<EOT\n    hi\n    EOT;\n",
		"<?php\n$n = 1_000;\n",
	}
	for _, src := range fixtures {
		toks := LexAll([]byte(src))
		got := PrintTokens(toks, []byte(src))
		if got != src {
			t.Fatalf("identity failed\nwant: %q\ngot:  %q", src, got)
		}
	}
}

func TestAttributeIsOnlyHashBracket(t *testing.T) {
	src := "<?php\n#[\\SensitiveParameter]\nfunction f() {}\n"
	l := NewFile(src)
	var attr token.Token
	for {
		tok := l.NextToken()
		if tok.Type == token.T_ATTRIBUTE {
			attr = tok
			break
		}
		if tok.Type == token.T_EOF {
			t.Fatal("missing T_ATTRIBUTE")
		}
	}
	if attr.Literal != "#[" {
		t.Fatalf("T_ATTRIBUTE should be #[, got %q", attr.Literal)
	}
}

func TestKeywordKeepsOriginalSpelling(t *testing.T) {
	l := New("Function")
	tok := l.NextToken()
	if tok.Type != token.T_FUNCTION {
		t.Fatalf("expected T_FUNCTION, got %s", tok.Type)
	}
	if tok.Literal != "Function" {
		t.Fatalf("expected spelling Function, got %q", tok.Literal)
	}
}

// TestTokenGetAllFullOrderedEquality asserts full ordered kind+text equality vs
// PHP token_get_all (R5). Known Zend/token_get_all surface differences are
// normalized only where the plan intentionally diverges (trivia-on-token open
// tag fold, atomic names vs T_NAME_*, NOWDOC kinds, single-char CHAR folding).
func TestTokenGetAllFullOrderedEquality(t *testing.T) {
	if _, err := exec.LookPath("php"); err != nil {
		t.Skip("php not available")
	}
	cases := []string{
		`<?php #[Attr] class C {}`,
		`<?php $a = "plain";`,
		`<?php $a = 'plain';`,
		"<?php $a = <<<EOT\nhello\nEOT;",
		`<?php $a = "hello $x world";`,
		"<?php\n/** doc */\nFunction f() {}\n",
		"<?php\n#[\\Foo\\Bar(x: 1)]\nfunction f(string $a) {}\n",
		"<?php\n$a = <<<'EOT'\nhello $x\nEOT;\n",
		"<?php\n$a = <<<EOT\n    hi\n    EOT;\n",
		"<?php\n$n = 1_000;\n",
		"<?php\n$a = 1;\n",
		`<?php $a = 1 + 2 * 3;`,
		`<?php $a = Foo\Bar;`,
		`<?php $a = \Foo\Bar;`,
		`<?php $a = namespace\Foo;`,
		`<?php $o->m();`,
		`<?php $o?->m();`,
	}
	for _, src := range cases {
		phpOut, err := phpTokenDump(src)
		if err != nil {
			t.Fatalf("php token dump for %q: %v\n%s", src, err, phpOut)
		}
		ours := ourTokenDump([]byte(src))
		phpLines := normalizeZendDump(parseDumpLines(phpOut), true)
		ourLines := normalizeZendDump(parseDumpLines(ours), false)
		if diff := diffDumpLines(phpLines, ourLines); diff != "" {
			t.Fatalf("full ordered token equality failed for %q\nPHP:\n%s\nOURS:\n%s\nDIFF:\n%s",
				src, phpOut, ours, diff)
		}
	}
}

func phpTokenDump(src string) (string, error) {
	cmd := exec.Command("php", "-r", `foreach (token_get_all(stream_get_contents(STDIN)) as $t) { if (is_array($t)) { echo token_name($t[0]), "\t", json_encode($t[1]), "\n"; } else { echo "CHAR\t", json_encode($t), "\n"; } }`)
	cmd.Stdin = strings.NewReader(src)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func ourTokenDump(src []byte) string {
	l := NewFileBytes(src)
	var b strings.Builder
	emit := func(typ, text string) {
		b.WriteString(typ)
		b.WriteByte('\t')
		b.WriteString(jsonQuote(text))
		b.WriteByte('\n')
	}
	for {
		tok := l.NextToken()
		for _, tr := range tok.LeadingTrivia {
			emit(tr.Type.String(), tr.Text(src))
		}
		if tok.Type == token.T_EOF {
			for _, tr := range tok.TrailingTrivia {
				emit(tr.Type.String(), tr.Text(src))
			}
			return b.String()
		}
		text := tok.Text(src)
		switch {
		case tok.Type == token.T_CONSTANT_STRING && text == "\"":
			emit("CHAR", text)
		case isCharLike(tok):
			emit("CHAR", text)
		default:
			emit(mapOurType(tok.Type), text)
		}
	}
}

func isCharLike(tok token.Token) bool {
	switch tok.Type {
	case token.T_LPAREN, token.T_RPAREN, token.T_LBRACE, token.T_RBRACE,
		token.T_LBRACKET, token.T_RBRACKET, token.T_SEMICOLON, token.T_COMMA,
		token.T_ASSIGN, token.T_COLON:
		return true
	default:
		return false
	}
}

func mapOurType(t token.TokenType) string {
	switch t {
	case token.T_CONSTANT_STRING:
		return "T_CONSTANT_ENCAPSED_STRING"
	default:
		return t.String()
	}
}

type dumpLine struct {
	kind string
	text string // unquoted source text
}

func parseDumpLines(s string) []dumpLine {
	var out []dumpLine
	for _, raw := range strings.Split(strings.TrimRight(s, "\n"), "\n") {
		if raw == "" {
			continue
		}
		parts := strings.SplitN(raw, "\t", 2)
		if len(parts) != 2 {
			out = append(out, dumpLine{kind: raw})
			continue
		}
		text, _ := unquoteJSON(parts[1])
		out = append(out, dumpLine{kind: parts[0], text: text})
	}
	return out
}

// normalizeZendDump produces a comparable ordered stream.
// expandNames applies only to PHP dumps (token_get_all T_NAME_* compounds).
func normalizeZendDump(in []dumpLine, expandNames bool) []dumpLine {
	var out []dumpLine
	for i := 0; i < len(in); i++ {
		l := in[i]
		if l.kind == "T_OPEN_TAG" {
			text := l.text
			for i+1 < len(in) && in[i+1].kind == "T_WHITESPACE" {
				text += in[i+1].text
				i++
			}
			out = append(out, dumpLine{kind: "T_OPEN_TAG", text: text})
			continue
		}
		if expandNames {
			switch l.kind {
			case "T_NAME_QUALIFIED", "T_NAME_FULLY_QUALIFIED", "T_NAME_RELATIVE":
				out = append(out, expandPHPName(l)...)
				continue
			}
		}
		out = append(out, dumpLine{kind: zendComparableKind(l.kind, l.text), text: l.text})
	}
	return out
}

func expandPHPName(l dumpLine) []dumpLine {
	text := l.text
	var out []dumpLine
	i := 0
	if strings.HasPrefix(text, `namespace\`) && l.kind == "T_NAME_RELATIVE" {
		out = append(out, dumpLine{kind: "T_NAMESPACE", text: "namespace"})
		out = append(out, dumpLine{kind: "T_NS_SEPARATOR", text: `\`})
		i = len("namespace\\")
	} else if strings.HasPrefix(text, `\`) && l.kind == "T_NAME_FULLY_QUALIFIED" {
		out = append(out, dumpLine{kind: "T_NS_SEPARATOR", text: `\`})
		i = 1
	}
	for i < len(text) {
		if text[i] == '\\' {
			out = append(out, dumpLine{kind: "T_NS_SEPARATOR", text: `\`})
			i++
			continue
		}
		j := i
		for j < len(text) && text[j] != '\\' {
			j++
		}
		out = append(out, dumpLine{kind: "T_STRING", text: text[i:j]})
		i = j
	}
	return out
}

func zendComparableKind(kind, text string) string {
	switch kind {
	case "T_START_NOWDOC":
		return "T_START_HEREDOC"
	case "T_END_NOWDOC":
		return "T_END_HEREDOC"
	}
	if len(text) == 1 {
		switch text[0] {
		case '(', ')', '{', '}', '[', ']', ';', ',', '=', ':',
			'+', '-', '*', '/', '%', '.', '!', '?', '|', '&', '^', '~', '@', '<', '>':
			return "CHAR"
		}
	}
	return kind
}

func diffDumpLines(php, ours []dumpLine) string {
	var sb strings.Builder
	n := len(php)
	if len(ours) > n {
		n = len(ours)
	}
	if len(php) != len(ours) {
		sb.WriteString(fmt.Sprintf("len php=%d ours=%d\n", len(php), len(ours)))
	}
	for i := 0; i < n; i++ {
		var a, b dumpLine
		if i < len(php) {
			a = php[i]
		}
		if i < len(ours) {
			b = ours[i]
		}
		if a.kind != b.kind || a.text != b.text {
			sb.WriteString(fmt.Sprintf("[%d] php=%s %q | ours=%s %q\n", i, a.kind, a.text, b.kind, b.text))
		}
	}
	return sb.String()
}

func jsonQuote(s string) string {
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

func unquoteJSON(s string) (string, error) {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		var b strings.Builder
		for i := 1; i < len(s)-1; i++ {
			if s[i] == '\\' && i+1 < len(s)-1 {
				i++
				switch s[i] {
				case 'n':
					b.WriteByte('\n')
				case 'r':
					b.WriteByte('\r')
				case 't':
					b.WriteByte('\t')
				case '/', '"', '\\':
					b.WriteByte(s[i])
				default:
					b.WriteByte(s[i])
				}
				continue
			}
			b.WriteByte(s[i])
		}
		return b.String(), nil
	}
	return s, nil
}
