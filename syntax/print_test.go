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

func TestIdentityPrintShortEchoHTML(t *testing.T) {
	fixtures := []string{
		`<h2>"<?= $statusCode; ?> <?= $statusText; ?>".</h2>`,
		`<?= $x ?>`,
		`<?= $this->addElementToGhost(); ?></svg>` + "\n",
		"<?php\n$f = function($x) use ($y) { return $x + $y; };\n$g = fn($a) => $a;\n$o = new class { public $z; };\n",
	}
	for _, src := range fixtures {
		res := Parse([]byte(src))
		got := Print(res.File.Root)
		if got != src {
			t.Fatalf("identity failed\nwant %q\ngot  %q", src, got)
		}
	}
}
