package overrides

import "testing"

func TestCompileAndIgnoreIssue(t *testing.T) {
	compiled, err := Compile(RuleOverrides{
		"PSR1.Classes.ClassDeclaration.PascalCase": {
			Classes: []string{"/^Legacy_/", "SpecialClass"},
		},
	})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	if !compiled.IgnoreIssue("PSR1.Classes.ClassDeclaration.PascalCase", "class", "Legacy_Service") {
		t.Fatal("expected Legacy_Service to be ignored")
	}
	if !compiled.IgnoreIssue("PSR1.Classes.ClassDeclaration.PascalCase", "class", "SpecialClass") {
		t.Fatal("expected SpecialClass to be ignored")
	}
	if compiled.IgnoreIssue("PSR1.Classes.ClassDeclaration.PascalCase", "class", "MySpecialClass") {
		t.Fatal("plain class override should match exact class name only")
	}
	if compiled.IgnoreIssue("PSR1.Classes.ClassDeclaration.PascalCase", "class", "ModernService") {
		t.Fatal("did not expect ModernService to be ignored")
	}
	if compiled.IgnoreIssue("PSR12.Files.EndFileNewline", "class", "Legacy_Service") {
		t.Fatal("did not expect unrelated rule code to be ignored")
	}
}

func TestCompileEmpty(t *testing.T) {
	for _, raw := range []RuleOverrides{nil, {}} {
		compiled, err := Compile(raw)
		if err != nil {
			t.Fatalf("Compile(%#v) error: %v", raw, err)
		}
		if compiled != nil {
			t.Fatalf("Compile(%#v) = %#v, want nil", raw, compiled)
		}
	}
}

func TestIgnoreIssueNilReceiverAndEmptySubject(t *testing.T) {
	var nilCompiled *Compiled
	if nilCompiled.IgnoreIssue("any.rule", "class", "Foo") {
		t.Fatal("nil Compiled should not ignore")
	}

	compiled, err := Compile(RuleOverrides{
		"rule.A": {Classes: []string{"Foo"}},
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if compiled.IgnoreIssue("rule.A", "class", "") {
		t.Fatal("empty subjectName should not match")
	}
}

func TestIgnoreIssueNonClassKinds(t *testing.T) {
	compiled, err := Compile(RuleOverrides{
		"rule.A": {Classes: []string{"SpecialClass"}},
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	for _, kind := range []string{"method", "function", ""} {
		if compiled.IgnoreIssue("rule.A", kind, "SpecialClass") {
			t.Fatalf("subjectKind %q should not ignore even when class name matches", kind)
		}
	}
}

func TestExactNameCaseInsensitive(t *testing.T) {
	compiled, err := Compile(RuleOverrides{
		"rule.A": {Classes: []string{"SpecialClass"}},
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	for _, name := range []string{"SpecialClass", "specialclass", "SPECIALCLASS", "sPeCiAlClAsS"} {
		if !compiled.IgnoreIssue("rule.A", "class", name) {
			t.Fatalf("exact override should match %q case-insensitively", name)
		}
	}
	if compiled.IgnoreIssue("rule.A", "class", "MySpecialClass") {
		t.Fatal("exact SpecialClass must not match MySpecialClass")
	}
}

func TestExactNameQuotesMetacharacters(t *testing.T) {
	tests := []struct {
		pattern string
		match   string
		nomatch string
	}{
		{pattern: "Foo.Bar", match: "Foo.Bar", nomatch: "FooXBar"},
		{pattern: "A+B", match: "A+B", nomatch: "AB"},
		{pattern: "A+B", match: "a+b", nomatch: "aab"},
	}

	for _, tc := range tests {
		compiled, err := Compile(RuleOverrides{
			"rule.A": {Classes: []string{tc.pattern}},
		})
		if err != nil {
			t.Fatalf("Compile(%q): %v", tc.pattern, err)
		}
		if !compiled.IgnoreIssue("rule.A", "class", tc.match) {
			t.Fatalf("pattern %q should literally match %q", tc.pattern, tc.match)
		}
		if compiled.IgnoreIssue("rule.A", "class", tc.nomatch) {
			t.Fatalf("pattern %q must not treat metacharacters as regex (matched %q)", tc.pattern, tc.nomatch)
		}
	}
}

func TestExactNameWhitespaceTrimmed(t *testing.T) {
	compiled, err := Compile(RuleOverrides{
		"rule.A": {Classes: []string{"  SpecialClass  "}},
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if !compiled.IgnoreIssue("rule.A", "class", "SpecialClass") {
		t.Fatal("whitespace around exact names should be trimmed")
	}
	if !compiled.IgnoreIssue("rule.A", "class", "specialclass") {
		t.Fatal("trimmed exact names remain case-insensitive")
	}
}

func TestSingleSlashIsExactNotEmptyRegex(t *testing.T) {
	compiled, err := Compile(RuleOverrides{
		"rule.A": {Classes: []string{"/"}},
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if compiled == nil {
		t.Fatal("single-slash pattern should compile as an exact name, not be skipped")
	}
	if !compiled.IgnoreIssue("rule.A", "class", "/") {
		t.Fatal("exact `/` should match subject `/`")
	}
	if compiled.IgnoreIssue("rule.A", "class", "Anything") {
		t.Fatal("exact `/` must not become an empty regex that matches everything")
	}
}

func TestSlashRegexRemainsUnanchoredUnlessAuthored(t *testing.T) {
	compiled, err := Compile(RuleOverrides{
		"rule.A": {Classes: []string{"/Legacy_/", "/^Legacy_/"}},
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	// Unanchored /Legacy_/ matches as a substring.
	if !compiled.IgnoreIssue("rule.A", "class", "XLegacy_Y") {
		t.Fatal("unanchored /Legacy_/ should match XLegacy_Y")
	}
	// Author-anchored /^Legacy_/ still requires the prefix.
	if !compiled.IgnoreIssue("rule.A", "class", "Legacy_Service") {
		t.Fatal("anchored /^Legacy_/ should match Legacy_Service")
	}
}

func TestCompileMultipleRuleCodes(t *testing.T) {
	compiled, err := Compile(RuleOverrides{
		"rule.A": {Classes: []string{"Alpha"}},
		"rule.B": {Classes: []string{"Beta"}},
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	if !compiled.IgnoreIssue("rule.A", "class", "Alpha") {
		t.Fatal("rule.A should ignore Alpha")
	}
	if compiled.IgnoreIssue("rule.A", "class", "Beta") {
		t.Fatal("rule.A should not ignore Beta")
	}
	if !compiled.IgnoreIssue("rule.B", "class", "beta") {
		t.Fatal("rule.B should ignore Beta case-insensitively")
	}
	if compiled.IgnoreIssue("rule.B", "class", "Alpha") {
		t.Fatal("rule.B should not ignore Alpha")
	}
}

func TestSlashRegexCaseSensitivityLeftToAuthor(t *testing.T) {
	compiled, err := Compile(RuleOverrides{
		"rule.A": {Classes: []string{"/Legacy_/"}},
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if !compiled.IgnoreIssue("rule.A", "class", "Legacy_Service") {
		t.Fatal("expected Legacy_Service to match /Legacy_/")
	}
	// Slash bodies are not auto-(?i); case-sensitive by default.
	if compiled.IgnoreIssue("rule.A", "class", "legacy_service") {
		t.Fatal("slash-regex must not auto-add (?i); legacy_service should not match /Legacy_/")
	}
}

func TestCompileRejectsInvalidRegex(t *testing.T) {
	_, err := Compile(RuleOverrides{
		"PSR1.Classes.ClassDeclaration.PascalCase": {
			Classes: []string{"/[unterminated/"},
		},
	})
	if err == nil {
		t.Fatal("expected invalid regex error")
	}
}

func TestCompileIgnoresBlankPatterns(t *testing.T) {
	compiled, err := Compile(RuleOverrides{
		"PSR1.Classes.ClassDeclaration.PascalCase": {
			Classes: []string{"", "   ", "//", "/^Legacy_/"},
		},
	})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	if !compiled.IgnoreIssue("PSR1.Classes.ClassDeclaration.PascalCase", "class", "Legacy_Service") {
		t.Fatal("expected non-blank Legacy_ pattern to be applied")
	}
	if compiled.IgnoreIssue("PSR1.Classes.ClassDeclaration.PascalCase", "class", "ModernService") {
		t.Fatal("blank override patterns should not match every class")
	}
}

func TestCompileReturnsNilWhenAllPatternsAreBlank(t *testing.T) {
	compiled, err := Compile(RuleOverrides{
		"PSR1.Classes.ClassDeclaration.PascalCase": {
			Classes: []string{"", "   ", "//"},
		},
	})
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	if compiled != nil {
		t.Fatalf("expected nil matcher when all override patterns are blank, got %#v", compiled)
	}
}
