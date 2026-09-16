package syntax

import (
	"strings"
	"testing"
)

func TestParsePropertyHooksStructured(t *testing.T) {
	cases := []struct {
		name       string
		src        string
		wantProps  int
		wantLists  int
		wantHooks  int
	}{
		{
			name:      "arrow get/set",
			src:       "<?php\nclass C { public string $x { get => $this->x; set => $this->x = $value; } }\n",
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
			res := Parse([]byte(tc.src))
			if got := Print(res.File.Root); got != tc.src {
				t.Fatalf("identity\nwant %q\ngot  %q", tc.src, got)
			}
			var props, lists, hooks int
			var walk func(*RedNode)
			walk = func(n *RedNode) {
				if n == nil {
					return
				}
				switch n.Kind() {
				case KindPropertyDecl:
					props++
				case KindPropertyHookList:
					lists++
				case KindPropertyHook:
					hooks++
				}
				for _, c := range n.Children() {
					walk(c)
				}
			}
			walk(res.File.Root)
			if props != tc.wantProps || lists != tc.wantLists || hooks != tc.wantHooks {
				t.Fatalf("structure props=%d lists=%d hooks=%d want %d/%d/%d\n%s",
					props, lists, hooks, tc.wantProps, tc.wantLists, tc.wantHooks, dumpGoldTree(res.File.Root))
			}
		})
	}
}

func TestGoldTreePropertyHooks(t *testing.T) {
	src := "<?php\nclass C { public string $x { get => $this->x; set => $this->x = $value; } }\n"
	res := Parse([]byte(src))
	if Print(res.File.Root) != src {
		t.Fatalf("identity failed\nwant %q\ngot  %q", src, Print(res.File.Root))
	}
	got := dumpGoldTree(res.File.Root)
	needles := []string{
		"PropertyDecl",
		"PropertyHookList",
		"PropertyHook",
		"Token T_STRING \"get\"",
		"Token T_STRING \"set\"",
		"Token T_DOUBLE_ARROW \"=>\"",
		"MemberAccessExpr",
		"AssignExpr",
	}
	for _, n := range needles {
		if !strings.Contains(got, n) {
			t.Fatalf("gold property-hooks dump missing %q\n%s", n, got)
		}
	}
	// Single PropertyDecl owns the hook list (not fragmented sibling decls).
	if strings.Count(got, "PropertyDecl\n") != 1 {
		t.Fatalf("expected exactly one PropertyDecl in dump:\n%s", got)
	}
	if !strings.Contains(got, "PropertyDecl\n") || !strings.Contains(got, "  PropertyHookList\n") {
		t.Fatalf("expected PropertyHookList nested under PropertyDecl:\n%s", got)
	}
}

func TestGoldTreeAsymmetricVisibilityModifier(t *testing.T) {
	src := "<?php\nclass C { public private(set) string $x; }\n"
	res := Parse([]byte(src))
	if Print(res.File.Root) != src {
		t.Fatalf("identity failed\nwant %q\ngot  %q", src, Print(res.File.Root))
	}
	got := dumpGoldTree(res.File.Root)
	needles := []string{
		"ModifierList",
		"Token T_PUBLIC \"public\"",
		"Token T_PRIVATE \"private\"",
		"Token T_LPAREN \"(\"",
		"Token T_STRING \"set\"",
		"Token T_RPAREN \")\"",
		"PropertyDecl",
	}
	for _, n := range needles {
		if !strings.Contains(got, n) {
			t.Fatalf("gold asymmetric-visibility dump missing %q\n%s", n, got)
		}
	}
}
