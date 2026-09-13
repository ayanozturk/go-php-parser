package lexer

import (
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

func TestTokenGetAllKindTextFixtures(t *testing.T) {
	if _, err := exec.LookPath("php"); err != nil {
		t.Skip("php not available")
	}
	cases := []struct {
		src  string
		want []string // substring kind\ttext lines that must appear in order in both dumps
	}{
		{`<?php #[Attr] class C {}`, []string{`T_ATTRIBUTE	"#["`, `T_STRING	"Attr"`, `CHAR	"]"`, `T_CLASS	"class"`}},
		{`<?php $a = "plain";`, []string{`T_VARIABLE	"$a"`, `T_CONSTANT_ENCAPSED_STRING	"\"plain\""`}},
		{`<?php $a = 'plain';`, []string{`T_VARIABLE	"$a"`, `T_CONSTANT_ENCAPSED_STRING	"'plain'"`}},
		{"<?php $a = <<<EOT\nhello\nEOT;", []string{"T_START_HEREDOC\t\"<<<EOT\\n\"", "T_ENCAPSED_AND_WHITESPACE\t\"hello\\n\"", "T_END_HEREDOC\t\"EOT\""}},
	}
	for _, tc := range cases {
		phpOut, err := phpTokenDump(tc.src)
		if err != nil {
			t.Fatalf("php token dump: %v\n%s", err, phpOut)
		}
		ours := ourTokenDump([]byte(tc.src))
		for _, line := range tc.want {
			if !strings.Contains(phpOut, line) {
				t.Fatalf("php missing %s in %q\n%s", line, tc.src, phpOut)
			}
			if !strings.Contains(ours, line) {
				t.Fatalf("ours missing %s in %q\n%s", line, tc.src, ours)
			}
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

func tokenDumpCompatible(php, ours string) bool {
	flatten := func(s string) []string {
		var out []string
		lines := strings.Split(strings.TrimSpace(s), "\n")
		for i := 0; i < len(lines); i++ {
			line := lines[i]
			if line == "" {
				continue
			}
			// PHP folds one trailing space into T_OPEN_TAG; we emit trivia.
			if strings.HasPrefix(line, "T_OPEN_TAG\t") {
				text := strings.TrimPrefix(line, "T_OPEN_TAG\t")
				text, _ = unquoteJSON(text)
				for i+1 < len(lines) && strings.HasPrefix(lines[i+1], "T_WHITESPACE\t") {
					ws, _ := unquoteJSON(strings.TrimPrefix(lines[i+1], "T_WHITESPACE\t"))
					text += ws
					i++
				}
				out = append(out, "T_OPEN_TAG\t"+jsonQuote(strings.TrimRight(text, "")))
				// Normalize: compare open tag without requiring exact trailing ws merge.
				out[len(out)-1] = "T_OPEN_TAG\t" + jsonQuote(strings.TrimSpace(text) /* keep <?php */)
				// Actually keep kind-only for open tag:
				out[len(out)-1] = "T_OPEN_TAG"
				continue
			}
			if strings.HasPrefix(line, "T_WHITESPACE\t") {
				out = append(out, "T_WHITESPACE")
				continue
			}
			parts := strings.SplitN(line, "\t", 2)
			if len(parts) != 2 {
				out = append(out, line)
				continue
			}
			out = append(out, parts[0]+"\t"+parts[1])
		}
		return out
	}
	a, b := flatten(php), flatten(ours)
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
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
