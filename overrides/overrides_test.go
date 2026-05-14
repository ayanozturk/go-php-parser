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
	if compiled.IgnoreIssue("PSR1.Classes.ClassDeclaration.PascalCase", "class", "ModernService") {
		t.Fatal("did not expect ModernService to be ignored")
	}
	if compiled.IgnoreIssue("PSR12.Files.EndFileNewline", "class", "Legacy_Service") {
		t.Fatal("did not expect unrelated rule code to be ignored")
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
