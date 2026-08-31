package analyse

import "testing"

func TestASCIILowerIdent(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"already_lower", "already_lower"},
		{"FooBar", "foobar"},
		{`Vendor\Service`, `vendor\service`},
		{"Ä", "ä"},
	}
	for _, tc := range cases {
		if got := asciiLowerIdent(tc.in); got != tc.want {
			t.Fatalf("asciiLowerIdent(%q)=%q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestIndexKeyASCIILower(t *testing.T) {
	t.Parallel()
	if got := indexKey(` \Vendor\Service `); got != `vendor\service` {
		t.Fatalf("indexKey=%q", got)
	}
	in := "already\\lower"
	if got := indexKey(in); got != in {
		t.Fatalf("lowercase indexKey should keep the input string, got %q", got)
	}
}
