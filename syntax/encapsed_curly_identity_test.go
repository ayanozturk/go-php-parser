package syntax

import "testing"

func TestEncapsedCurlyIdentity(t *testing.T) {
	cases := []string{
		"<?php\necho <<<EOT\nhello {$x}\nEOT;\n",
		"<?php\n$a = \"hello {$x} world\";\n",
		"<?php\n$a = \"doctrine.dbal.{$name}_connection\";\n",
		"<?php\necho <<<EOTXT\n        comment: |\n                  {$mainRepo}\n                  We're looking forward to your PR there!\n\n        EOTXT;\n",
	}
	for _, src := range cases {
		res := Parse([]byte(src))
		got := Print(res.File.Root)
		if got != src {
			t.Fatalf("identity\nwant %q\ngot  %q", src, got)
		}
	}
}
