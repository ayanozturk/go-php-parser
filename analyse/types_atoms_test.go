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

func TestNonEmptyLowercaseStringRemainsUnrefinedUntilCaseSemanticsExist(t *testing.T) {
	if got := ParseType("non-empty-lowercase-string").String(); got != "string" {
		t.Fatalf("normalized non-empty-lowercase-string = %q, want string", got)
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

func TestQuotedPHPDocStringLiteralNormalizationAndSubtyping(t *testing.T) {
	if got := ParseType(`'Ready'`).String(); got != `'Ready'` {
		t.Fatalf("literal normalization = %q, want exact case-preserved literal", got)
	}
	if !ParseType(`string`).Accepts(ParseType(`'Ready'`)) {
		t.Fatal("string should accept an exact string literal")
	}
	if !ParseType(`non-empty-string`).Accepts(ParseType(`'Ready'`)) {
		t.Fatal("non-empty-string should accept a non-empty exact literal")
	}
	if ParseType(`non-empty-string`).Accepts(ParseType(`''`)) {
		t.Fatal("non-empty-string should reject the empty exact literal")
	}
	if !ParseType(`'Ready'|'Busy'`).Accepts(ParseType(`'Ready'`)) {
		union, member := ParseType(`'Ready'|'Busy'`), ParseType(`'Ready'`)
		t.Fatalf("literal union %q (%#v) should accept member %q (%#v)", union.String(), union.atoms, member.String(), member.atoms)
	}
	if ParseType(`'Ready'`).Accepts(ParseType(`'ready'`)) {
		t.Fatal("literal strings should remain case-sensitive")
	}
	if ParseType(`'Ready'`).Accepts(ParseType(`string`)) {
		t.Fatal("an unrefined string should remain distinct from an exact literal in the type lattice")
	}
	if !argumentExpectedTypeAcceptsActual(ParseType(`'Ready'`), ParseType(`string`), nil, nil) {
		t.Fatal("level-5 PHPStan behavior accepts an unrefined string at a literal parameter")
	}
}

func TestIntegerRangesAndLiteralSubtyping(t *testing.T) {
	cases := []struct {
		declared, actual string
		want             bool
	}{
		{"int<1, 10>", "5", true},
		{"int<1,10>", "0", false},
		{"int<min, max>", "positive-int", true},
		{"int<0, max>", "non-negative-int", true},
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
