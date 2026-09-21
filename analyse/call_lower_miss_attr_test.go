package analyse

import (
	"strings"
	"testing"

	"github.com/ayanozturk/go-php-parser/syntax"
)

func TestClassifyCallExprCSTFallbackBuiltins(t *testing.T) {
	cases := []struct {
		src    string
		want   string
		substr string // when set, reason must contain this
	}{
		{src: `<?php isset($a);`, want: "builtin:isset"},
		{src: `<?php empty($a);`, want: "builtin:empty"},
		{src: `<?php exit(1);`, want: "builtin:exit"},
		{src: `<?php die();`, want: "builtin:die"},
		{src: `<?php foo()->bar();`, substr: "member_complex_object:CallExpr"},
		{src: `<?php $a['f']();`, want: "callee_array_dim"},
		// unset($a) is KindUnsetStmt, not KindCallExpr — not in residual.
	}
	for _, tc := range cases {
		res := syntax.Parse([]byte(tc.src))
		if res == nil || res.File == nil || res.File.Root == nil {
			t.Fatalf("parse %q", tc.src)
		}
		var call *syntax.RedNode
		syntax.Walk(res.File.Root, func(n *syntax.RedNode) bool {
			if n.Kind() == syntax.KindCallExpr {
				// Walk reuses a scratch node; copy identity before continuing.
				cp := *n
				call = &cp
				return false
			}
			return true
		})
		if call == nil {
			t.Fatalf("no KindCallExpr in %q", tc.src)
		}
		got := classifyCallExprCSTFallback(call)
		if tc.want != "" && got != tc.want {
			t.Fatalf("%q: got %q want %q", tc.src, got, tc.want)
		}
		if tc.substr != "" && !strings.Contains(got, tc.substr) {
			t.Fatalf("%q: got %q want contain %q", tc.src, got, tc.substr)
		}
	}
}

func TestFormatCallLowerMissReasons(t *testing.T) {
	s := FormatCallLowerMissReasons(map[string]uint64{
		"builtin:isset":               100,
		"member_complex_object:CallExpr": 10,
	})
	if !strings.Contains(s, "builtin:isset") || !strings.Contains(s, "n=110") {
		t.Fatalf("format:\n%s", s)
	}
}
