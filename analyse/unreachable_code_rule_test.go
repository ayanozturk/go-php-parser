package analyse

import (
	"go-phpcs/lexer"
	"go-phpcs/parser"
	"testing"
)

func analyseUnreachablePHP(t *testing.T, code string) []AnalysisIssue {
	t.Helper()
	l := lexer.New(code)
	p := parser.New(l, false)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	rule := &UnreachableCodeRule{}
	return rule.CheckIssues(nodes, "test.php")
}

func countUnreachableIssues(issues []AnalysisIssue) int {
	count := 0
	for _, issue := range issues {
		if issue.Code == "Generic.CodeAnalysis.UnreachableCode" {
			count++
		}
	}
	return count
}

func TestUnreachableAfterReturnInFunction(t *testing.T) {
	php := `<?php
function foo(): int {
    return 1;
    $x = 2;
}`
	issues := analyseUnreachablePHP(t, php)
	if got := countUnreachableIssues(issues); got != 1 {
		t.Fatalf("expected 1 unreachable issue, got %d (%#v)", got, issues)
	}
}

func TestUnreachableAfterThrowInFunction(t *testing.T) {
	php := `<?php
function foo(): int {
    throw $e;
    return 1;
}`
	issues := analyseUnreachablePHP(t, php)
	if got := countUnreachableIssues(issues); got != 1 {
		t.Fatalf("expected 1 unreachable issue, got %d (%#v)", got, issues)
	}
}

func TestReachableAcrossBranchesNotReported(t *testing.T) {
	php := `<?php
function foo($x): int {
    if ($x) {
        return 1;
    }

    return 2;
}`
	issues := analyseUnreachablePHP(t, php)
	if got := countUnreachableIssues(issues); got != 0 {
		t.Fatalf("expected 0 unreachable issues, got %d (%#v)", got, issues)
	}
}

func TestUnreachableInsideIfBranch(t *testing.T) {
	php := `<?php
function foo($x): int {
    if ($x) {
        return 1;
        $a = 1;
    }

    return 2;
}`
	issues := analyseUnreachablePHP(t, php)
	if got := countUnreachableIssues(issues); got != 1 {
		t.Fatalf("expected 1 unreachable issue, got %d (%#v)", got, issues)
	}
}

func TestReachableAfterIfReturnWithNullsafeCallNotReported(t *testing.T) {
	php := `<?php
function getContactEmail(): ?string {
    if ($this->email) {
        return $this->email;
    }

    $admin = $this->getAdministrator();
    return $admin?->getEmail();
}`
	issues := analyseUnreachablePHP(t, php)
	if got := countUnreachableIssues(issues); got != 0 {
		t.Fatalf("expected 0 unreachable issues, got %d (%#v)", got, issues)
	}
}

func TestUnreachableAfterExitInFunction(t *testing.T) {
	php := `<?php
function foo(): void {
    exit();
    $x = 1;
}`
	issues := analyseUnreachablePHP(t, php)
	if got := countUnreachableIssues(issues); got != 1 {
		t.Fatalf("expected 1 unreachable issue, got %d (%#v)", got, issues)
	}
}

func TestReachableAfterConditionalExitNotReported(t *testing.T) {
	php := `<?php
function foo($redirectURL): void {
    if ($redirectURL) {
        redirect($redirectURL);
        exit();
    }

    $authenticationState = $authService->getState();
}`
	issues := analyseUnreachablePHP(t, php)
	if got := countUnreachableIssues(issues); got != 0 {
		t.Fatalf("expected 0 unreachable issues, got %d (%#v)", got, issues)
	}
}

func TestUnreachableAfterIfElseBothTerminate(t *testing.T) {
	php := `<?php
function foo($x): void {
    if ($x) {
        exit();
    } else {
        throw $e;
    }

    $x = 1;
}`
	issues := analyseUnreachablePHP(t, php)
	if got := countUnreachableIssues(issues); got != 1 {
		t.Fatalf("expected 1 unreachable issue, got %d (%#v)", got, issues)
	}
}
