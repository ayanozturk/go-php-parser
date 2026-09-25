package syntax

import (
	"strings"
	"testing"
)

func mustParseIdentity(t *testing.T, src string) *ParseResult {
	t.Helper()
	res := Parse([]byte(src))
	if got := Print(res.File.Root); got != src {
		t.Fatalf("identity\nwant %q\ngot  %q", src, got)
	}
	return res
}

func assertGoldNeedles(t *testing.T, got, label string, needles []string) {
	t.Helper()
	for _, n := range needles {
		if !strings.Contains(got, n) {
			t.Fatalf("gold %s dump missing %q\n%s", label, n, got)
		}
	}
}

func countKinds(root *RedNode, kinds ...Kind) map[Kind]int {
	want := make(map[Kind]int, len(kinds))
	for _, k := range kinds {
		want[k] = 0
	}
	var walk func(*RedNode)
	walk = func(n *RedNode) {
		if n == nil {
			return
		}
		if _, ok := want[n.Kind()]; ok {
			want[n.Kind()]++
		}
		for _, c := range n.Children() {
			walk(c)
		}
	}
	walk(root)
	return want
}

func TestParsePropertyHooksStructured(t *testing.T) {
	cases := []struct {
		name      string
		src       string
		wantProps int
		wantLists int
		wantHooks int
	}{
		{
			name:      "arrow get/set",
			src:       "<?php\nclass C { public string $x { get => $this->x; set => $this->x = $value; } }\n",
			wantProps: 1, wantLists: 1, wantHooks: 2,
		},
		{
			name:      "arrow get/set with default value",
			src:       "<?php\nclass C { public int $x = 321 { get => $this->x; set => $this->x = $value; } }\n",
			wantProps: 1, wantLists: 1, wantHooks: 2,
		},
		{
			name:      "abstract get",
			src:       "<?php\nabstract class C { abstract public string $bar { get; } }\n",
			wantProps: 1, wantLists: 1, wantHooks: 1,
		},
		{
			name:      "interface get/set",
			src:       "<?php\ninterface I { public string $foo { get; set; } }\n",
			wantProps: 1, wantLists: 1, wantHooks: 2,
		},
		{
			name:      "braced body with set params",
			src:       "<?php\nclass C { public string $x { get { return $this->x; } set (string $v) { $this->x = $v; } } }\n",
			wantProps: 1, wantLists: 1, wantHooks: 2,
		},
		{
			name:      "asymmetric visibility without hooks",
			src:       "<?php\nclass C { public private(set) string $x; }\n",
			wantProps: 1, wantLists: 0, wantHooks: 0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := mustParseIdentity(t, tc.src)
			got := countKinds(res.File.Root, KindPropertyDecl, KindPropertyHookList, KindPropertyHook)
			if got[KindPropertyDecl] != tc.wantProps ||
				got[KindPropertyHookList] != tc.wantLists ||
				got[KindPropertyHook] != tc.wantHooks {
				t.Fatalf("structure props=%d lists=%d hooks=%d want %d/%d/%d\n%s",
					got[KindPropertyDecl], got[KindPropertyHookList], got[KindPropertyHook],
					tc.wantProps, tc.wantLists, tc.wantHooks, dumpGoldTree(res.File.Root))
			}
		})
	}
}

func TestGoldTreePropertyHooks(t *testing.T) {
	src := "<?php\nclass C { public string $x { get => $this->x; set => $this->x = $value; } }\n"
	res := mustParseIdentity(t, src)
	got := dumpGoldTree(res.File.Root)
	assertGoldNeedles(t, got, "property-hooks", []string{
		"PropertyDecl",
		"PropertyHookList",
		"PropertyHook",
		"Token T_STRING \"get\"",
		"Token T_STRING \"set\"",
		"Token T_DOUBLE_ARROW \"=>\"",
		"MemberAccessExpr",
		"AssignExpr",
	})
	if strings.Count(got, "PropertyDecl\n") != 1 {
		t.Fatalf("expected exactly one PropertyDecl in dump:\n%s", got)
	}
	if !strings.Contains(got, "  PropertyHookList\n") {
		t.Fatalf("expected PropertyHookList nested under PropertyDecl:\n%s", got)
	}
}

func TestGoldTreeAsymmetricVisibilityModifier(t *testing.T) {
	src := "<?php\nclass C { public private(set) string $x; }\n"
	res := mustParseIdentity(t, src)
	assertGoldNeedles(t, dumpGoldTree(res.File.Root), "asymmetric-visibility", []string{
		"ModifierList",
		"Token T_PUBLIC \"public\"",
		"Token T_PRIVATE \"private\"",
		"Token T_LPAREN \"(\"",
		"Token T_STRING \"set\"",
		"Token T_RPAREN \")\"",
		"PropertyDecl",
	})
}
