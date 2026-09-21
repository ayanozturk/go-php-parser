package analyse

import "testing"

func TestAtomsCompatibleBuiltinPromotions(t *testing.T) {
	cases := []struct {
		declared, actual string
		want             bool
	}{
		{"int", "int", true},
		{"float", "int", true},
		{"int", "float", false},
		{"void", "null", true},
		{"bool", "true", true},
		{"bool", "false", true},
		{"iterable", "array", true},
		{"object", "stdClass", true},
		{"string", "int", false},
	}
	for _, tc := range cases {
		declared, ok := normalizeTypeAtom(tc.declared)
		if !ok {
			t.Fatalf("parse declared %q", tc.declared)
		}
		actual, ok := normalizeTypeAtom(tc.actual)
		if !ok {
			t.Fatalf("parse actual %q", tc.actual)
		}
		if got := atomsCompatible(declared, actual); got != tc.want {
			t.Fatalf("%s Accepts %s: got %v want %v", tc.declared, tc.actual, got, tc.want)
		}
	}
}

func TestTypeAcceptsUsesAtomCompatibility(t *testing.T) {
	if !ParseType("float").Accepts(ParseType("int")) {
		t.Fatal("float should accept int")
	}
	if ParseType("int").Accepts(ParseType("float")) {
		t.Fatal("int should not accept float")
	}
}
