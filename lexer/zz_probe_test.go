package lexer

import (
	"testing"
)

func TestProbeParityCandidates(t *testing.T) {
	cases := []string{
		`<?php $a = $b ?: $c;`,
		`<?php function gen() { yield from g(); }`,
		`<?php $f = strlen(...);`,
		`<?php $x = (int)$y;`,
		`<?php enum E: string { case A = 'a'; }`,
		`<?php class C { public const string FOO = 1; }`,
		`<?php $o->$x; $o->{$y};`,
		`<?php $a = <<<'NOW'
plain $x
NOW;`,
	}
	for _, src := range cases {
		phpOut, err := phpTokenDump(src)
		if err != nil {
			t.Logf("CANDIDATE %q: php dump error: %v\n%s", src, err, phpOut)
			continue
		}
		ours := ourTokenDump([]byte(src))
		phpLines := normalizeZendDump(parseDumpLines(phpOut), true)
		ourLines := normalizeZendDump(parseDumpLines(ours), false)
		if diff := diffDumpLines(phpLines, ourLines); diff != "" {
			t.Logf("CANDIDATE %q: DIFF\nPHP:\n%s\nOURS:\n%s\nDIFF:\n%s", src, phpOut, ours, diff)
		} else {
			t.Logf("CANDIDATE %q: EQUAL", src)
		}
	}
}
