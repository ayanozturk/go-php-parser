package syntax

import (
	"strings"
	"testing"
)

// TestHotspotIncompleteRecoveryControls locks recovery for incomplete
// editor inputs across the step-3 high-risk hotspots (Elvis, yield,
// dynamic/braced members, first-class callables, casts, enum cases,
// class constants, property hooks, nowdoc, mixed-case keywords).
//
// Each control asserts: source identity holds, diagnostics stay bounded,
// the hotspot node kind survives recovery, and (where the grammar emits
// one) a KindMissing close token is present. These are characterization
// controls, not grammar changes: two lenient shapes (bare `?:` with no
// else, operand-less cast) intentionally produce no diagnostic today and
// are locked as such so any future strictness is a visible diff.
func TestHotspotIncompleteRecoveryControls(t *testing.T) {
	cases := []struct {
		name        string
		src         string
		wantKinds   []Kind
		wantDiag    string // "" means expect zero diagnostics
		wantMissing bool
	}{
		{
			name:        "elvis missing else",
			src:         "<?php $a = $b ?:",
			wantKinds:   []Kind{KindTernaryExpr},
			wantDiag:    "",
			wantMissing: false,
		},
		{
			name:        "yield from truncated",
			src:         "<?php function gen() { yield from",
			wantKinds:   []Kind{KindFunctionDecl, KindYieldExpr},
			wantDiag:    "T_RBRACE",
			wantMissing: true,
		},
		{
			name:        "braced member unclosed",
			src:         "<?php $o->{$y",
			wantKinds:   []Kind{KindMemberAccessExpr, KindParenExpr},
			wantDiag:    "T_RBRACE",
			wantMissing: true,
		},
		{
			name:        "first-class callable unclosed",
			src:         "<?php $f = strlen(...",
			wantKinds:   []Kind{KindCallExpr, KindArgList},
			wantDiag:    "T_RPAREN",
			wantMissing: true,
		},
		{
			name:        "cast missing operand",
			src:         "<?php $x = (int)",
			wantKinds:   []Kind{KindCastExpr},
			wantDiag:    "",
			wantMissing: false,
		},
		{
			name:        "enum case missing value",
			src:         "<?php enum E: string { case A =",
			wantKinds:   []Kind{KindEnumDecl, KindEnumCase},
			wantDiag:    "T_RBRACE",
			wantMissing: true,
		},
		{
			name:        "class const trailing comma",
			src:         "<?php class C { public const string FOO = 1,",
			wantKinds:   []Kind{KindClassDecl, KindClassConstDecl},
			wantDiag:    "T_RBRACE",
			wantMissing: true,
		},
		{
			name:        "property hook truncated",
			src:         "<?php class C { public string $x { get",
			wantKinds:   []Kind{KindPropertyDecl, KindPropertyHookList, KindPropertyHook},
			wantDiag:    "T_RBRACE",
			wantMissing: true,
		},
		{
			name:        "nowdoc unterminated",
			src:         "<?php $a = <<<'NOW'\nplain $x\n",
			wantKinds:   []Kind{KindNowdoc},
			wantDiag:    "",
			wantMissing: false,
		},
		{
			name:        "mixed-case if unclosed",
			src:         "<?php IF ($a) {",
			wantKinds:   []Kind{KindIfStmt},
			wantDiag:    "T_RBRACE",
			wantMissing: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := Parse([]byte(tc.src))
			if got := Print(res.File.Root); got != tc.src {
				t.Fatalf("recovery identity\nwant %q\ngot  %q", tc.src, got)
			}
			for _, k := range tc.wantKinds {
				if !containsKind(res.File.Root, k) {
					t.Fatalf("expected %s in tree:\n%s", k, dumpGoldTree(res.File.Root))
				}
			}
			if tc.wantDiag == "" {
				if len(res.Diagnostics) != 0 {
					t.Fatalf("expected zero diagnostics, got %#v", res.Diagnostics)
				}
			} else {
				if len(res.Diagnostics) == 0 {
					t.Fatalf("expected %q diagnostic, got none", tc.wantDiag)
				}
				if len(res.Diagnostics) > 4 {
					t.Fatalf("diagnostics unbounded (%d): %#v", len(res.Diagnostics), res.Diagnostics)
				}
				found := false
				for _, d := range res.Diagnostics {
					if strings.Contains(d.Message, tc.wantDiag) {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("diag want substring %q, got %#v", tc.wantDiag, res.Diagnostics)
				}
			}
			if tc.wantMissing && !containsKind(res.File.Root, KindMissing) {
				t.Fatalf("expected KindMissing in tree:\n%s", dumpGoldTree(res.File.Root))
			}
		})
	}
}
