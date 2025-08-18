package analyse

import (
	"go-phpcs/lexer"
	"go-phpcs/parser"
	"testing"
)

// helper to run analysis on a PHP snippet and return issues
func analysePHP(t *testing.T, code string) []AnalysisIssue {
	t.Helper()
	l := lexer.New(code)
	p := parser.New(l, false)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	return RunAnalysisRules("test.php", nodes)
}

func hasReturnTypeIssue(issues []AnalysisIssue) bool {
	for _, iss := range issues {
		if iss.Code == "A.RETURN.TYPE" {
			return true
		}
	}
	return false
}

func TestImplodeReturnsStringNoMismatch(t *testing.T) {
	php := `<?php
    function foo(): string {
        $arr = ["a", "b"];
        return implode("\n", $arr);
    }`
	issues := analysePHP(t, php)
	if hasReturnTypeIssue(issues) {
		t.Fatalf("expected no A.RETURN.TYPE issue for implode returning string, got: %#v", issues)
	}
}

func TestMultipleCompatibleTypesNoError(t *testing.T) {
	php := `<?php
    function bar(): bool {
        if ($x) {
            return true;
        }
        return $y; // mixed
    }`
	issues := analysePHP(t, php)
	if hasReturnTypeIssue(issues) {
		t.Fatalf("expected no A.RETURN.TYPE issue when actual types are [bool,mixed] declared bool, got: %#v", issues)
	}
}
