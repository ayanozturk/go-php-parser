package syntax

import "testing"

func TestIdentityPrintTokensOnly(t *testing.T) {
	fixtures := []string{
		"<?php\n$a = 1;\n",
		"<?php\n#[Attr]\nclass C {}\n",
		"<?php\n/** doc */\nFunction f() {}\n",
		"<?php\n$a = \"hello $x\";\n",
		"<?php\n$a = <<<EOT\nhi $x\nEOT;\n",
		"<?php\n$a = <<<'EOT'\nhi\nEOT;\n",
	}
	for _, src := range fixtures {
		f := ParseTokens([]byte(src))
		got := Print(f.Root)
		if got != src {
			t.Fatalf("print(parse(src)) != src\nwant %q\ngot  %q", src, got)
		}
	}
}
