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

func TestNonEmptyLowercaseStringRemainsAnExactRefinement(t *testing.T) {
	if got := ParseType("non-empty-lowercase-string").String(); got != "non-empty-lowercase-string" {
		t.Fatalf("normalized non-empty-lowercase-string = %q", got)
	}
}

func TestStringRefinementLiteralCompatibility(t *testing.T) {
	cases := []struct {
		declared string
		actual   string
		want     bool
	}{
		{"numeric-string", `'1.25e+2'`, true},
		{"numeric-string", `'12px'`, false},
		{"non-empty-numeric-string", `''`, false},
		{"lowercase-string", `'ready'`, true},
		{"lowercase-string", `'Ready'`, false},
		{"non-empty-lowercase-string", `''`, false},
		{"non-empty-lowercase-string", `'ready'`, true},
		{"uppercase-string", `'READY'`, true},
		{"non-empty-uppercase-string", `'ready'`, false},
		{"non-falsy-string", `'0'`, false},
		{"truthy-string", `'ready'`, true},
		{"string", "non-empty-numeric-string", true},
		{"non-empty-string", "non-empty-lowercase-string", true},
		{"lowercase-string", "non-empty-lowercase-string", true},
		{"numeric-string", "non-empty-numeric-string", true},
	}
	for _, tc := range cases {
		if got := ParseType(tc.declared).Accepts(ParseType(tc.actual)); got != tc.want {
			t.Errorf("%s accepts %s = %v, want %v", tc.declared, tc.actual, got, tc.want)
		}
	}
}

func TestNonEmptyStringPreservesItsSubtypeBoundary(t *testing.T) {
	nonEmpty := ParseType("non-empty-string")
	empty := ParseType("empty-string")
	if !ParseType("string").Accepts(nonEmpty) {
		t.Fatal("string should accept non-empty-string")
	}
	if !ParseType("string").Accepts(empty) {
		t.Fatal("string should accept empty-string")
	}
	if !nonEmpty.Accepts(nonEmpty) {
		t.Fatal("non-empty-string should accept itself")
	}
	if nonEmpty.Accepts(empty) {
		t.Fatal("non-empty-string must reject empty-string")
	}
	if nonEmpty.Accepts(ParseType("string")) {
		t.Fatal("non-empty-string must not accept an unrefined string")
	}
}

func TestPHPDocStringLiteralSubtypingAndEscaping(t *testing.T) {
	allowed := ParseType(`'red'|'blue'`)
	if !allowed.Accepts(ParseType(`'red'`)) {
		t.Fatal("a string literal union should accept one of its members")
	}
	if allowed.Accepts(ParseType(`'green'`)) {
		t.Fatal("a string literal union should reject an unlisted member")
	}
	if !ParseType("string").Accepts(ParseType(`'red'`)) {
		t.Fatal("string should accept a string literal")
	}
	if !ParseType("empty-string").Accepts(ParseType(`''`)) || ParseType("non-empty-string").Accepts(ParseType(`''`)) {
		t.Fatal("empty-string refinements should remain compatible with exact empty literals")
	}
	for _, literal := range []string{`'it\'s'`, `'a\\b'`, `'red|blue'`, `'pair,with,commas'`} {
		if got := ParseType(literal).String(); got != literal {
			t.Errorf("ParseType(%q).String() = %q", literal, got)
		}
	}
}

func TestIntegerRangesAndLiteralSubtyping(t *testing.T) {
	cases := []struct {
		declared, actual string
		want             bool
	}{
		{"int<1, 10>", "5", true},
		{"int<1, 10>", "1", true},
		{"int<1, 10>", "10", true},
		{"int<1,10>", "0", false},
		{"int<1,10>", "11", false},
		{"int<min, max>", "positive-int", true},
		{"int<0, max>", "non-negative-int", true},
		{"int<min, -1>", "-1", true},
		{"int<0, max>", "negative-int", false},
		{"int<1,10>", "positive-int", false},
		{"int", "int<1,10>", true},
		{"float", "int<1,10>", true},
	}
	for _, tc := range cases {
		if got := ParseType(tc.declared).Accepts(ParseType(tc.actual)); got != tc.want {
			t.Errorf("%s Accepts %s = %v, want %v", tc.declared, tc.actual, got, tc.want)
		}
	}
}
