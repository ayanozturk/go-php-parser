package syntax

import (
	"strings"
	"testing"
)

// TestMalformedRecoveryRoundTrip covers R5 malformed-input recovery: missing
// tokens still round-trip and diagnostics are structured.
func TestMalformedRecoveryRoundTrip(t *testing.T) {
	cases := []struct {
		name       string
		src        string
		parse      func(*Parser) *GreenNode
		wantDiag   string
		wantMissing bool
	}{
		{
			name:        "attr missing close bracket",
			src:         "#[Foo",
			parse:       (*Parser).parseAttributeList,
			wantDiag:    "expected T_RBRACKET",
			wantMissing: true,
		},
		{
			name:        "attr missing paren and bracket",
			src:         "#[Foo(",
			parse:       (*Parser).parseAttributeList,
			wantDiag:    "expected T_RPAREN",
			wantMissing: true,
		},
		{
			name:        "dnf missing close paren",
			src:         "(Foo&Bar",
			parse:       (*Parser).parseType,
			wantDiag:    "expected T_RPAREN",
			wantMissing: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := NewParser([]byte(tc.src))
			g := tc.parse(p)
			f := &File{Source: []byte(tc.src), Green: g}
			BindRed(f)
			got := Print(f.Root)
			if got != tc.src {
				t.Fatalf("recovery round-trip failed\nwant %q\ngot  %q", tc.src, got)
			}
			if len(p.diags) == 0 {
				t.Fatal("expected structured diagnostics")
			}
			found := false
			for _, d := range p.diags {
				if strings.Contains(d.Message, tc.wantDiag) {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("diag want substring %q, got %#v", tc.wantDiag, p.diags)
			}
			if tc.wantMissing && !containsKind(f.Root, KindMissing) {
				t.Fatalf("expected KindMissing in tree:\n%s", dumpGoldTree(f.Root))
			}
		})
	}
}

func TestMalformedFileAttributeRecovery(t *testing.T) {
	src := "<?php\n#[Attr\nclass C {}\n"
	res := Parse([]byte(src))
	if Print(res.File.Root) != src {
		t.Fatalf("file recovery identity\nwant %q\ngot  %q", src, Print(res.File.Root))
	}
	if len(res.Diagnostics) == 0 {
		t.Fatal("expected diagnostics for missing ]")
	}
	if !containsKind(res.File.Root, KindMissing) {
		t.Fatalf("expected KindMissing:\n%s", dumpGoldTree(res.File.Root))
	}
	if !containsKind(res.File.Root, KindClassDecl) {
		t.Fatal("expected ClassDecl after recovered attribute")
	}
	found := false
	for _, d := range res.Diagnostics {
		if strings.Contains(d.Message, "T_RBRACKET") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected T_RBRACKET diagnostic, got %#v", res.Diagnostics)
	}
}

func TestUnclosedClassFunctionBodyAtEOF(t *testing.T) {
	src := "<?php\nclass Foo {\n\tpublic function bar() {\n\t\t// missing closing brace\n"
	res := Parse([]byte(src))
	if Print(res.File.Root) != src {
		t.Fatalf("identity\nwant %q\ngot  %q", src, Print(res.File.Root))
	}
	if len(res.Diagnostics) < 2 {
		t.Fatalf("expected >=2 T_RBRACE diags for nested unclosed braces, got %#v", res.Diagnostics)
	}
	rbrace := 0
	for _, d := range res.Diagnostics {
		if strings.Contains(d.Message, "T_RBRACE") {
			rbrace++
		}
	}
	if rbrace < 2 {
		t.Fatalf("expected >=2 T_RBRACE diagnostics, got %#v", res.Diagnostics)
	}
	if !containsKind(res.File.Root, KindMissing) {
		t.Fatalf("expected KindMissing close braces:\n%s", dumpGoldTree(res.File.Root))
	}
}

func containsKind(n *RedNode, k Kind) bool {
	if n == nil {
		return false
	}
	if n.Kind() == k {
		return true
	}
	for _, c := range n.Children() {
		if containsKind(c, k) {
			return true
		}
	}
	return false
}
