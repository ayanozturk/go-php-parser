package analyse

import (
	"strings"
	"testing"
)

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
		{"FooÄ", "fooä"},
		{"ÄFoo", "äfoo"},
		{"Straße", "straße"},
	}
	for _, tc := range cases {
		if got := asciiLowerIdent(tc.in); got != tc.want {
			t.Fatalf("asciiLowerIdent(%q)=%q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestASCIILowerIdentAllocations(t *testing.T) {
	var got string
	lowerAllocs := testing.AllocsPerRun(100, func() {
		got = asciiLowerIdent("already_lower")
	})
	if lowerAllocs != 0 {
		t.Fatalf("already-lowercase ASCII allocation count = %.2f, want 0", lowerAllocs)
	}
	if got != "already_lower" {
		t.Fatalf("already-lowercase ASCII result = %q", got)
	}

	upperAllocs := testing.AllocsPerRun(100, func() {
		got = asciiLowerIdent("FooBar")
	})
	if upperAllocs != 0 {
		t.Fatalf("cached mixed-case ASCII allocation count = %.2f, want 0", upperAllocs)
	}
	if got != "foobar" {
		t.Fatalf("uppercase ASCII result = %q", got)
	}

	first := asciiLowerIdent("Vendor\\Service")
	second := asciiLowerIdent("Vendor\\Service")
	if first != "vendor\\service" || second != first {
		t.Fatalf("interned mixed-case results diverged: %q vs %q", first, second)
	}

	for _, input := range []string{"ä", "Straße"} {
		input := input
		got = asciiLowerIdent(input)
		if want := strings.ToLower(input); got != want {
			t.Fatalf("Unicode asciiLowerIdent(%q) = %q, want strings.ToLower result %q", input, got, want)
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
