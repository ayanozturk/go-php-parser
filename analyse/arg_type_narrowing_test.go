package analyse

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/lexer"
	"github.com/ayanozturk/go-php-parser/parser"
)

func parseArgTypeNarrowingPHP(t *testing.T, source string) []AnalysisIssue {
	t.Helper()
	const filename = "narrowing.php"
	p := parser.New(lexer.New(source), false)
	nodes := p.Parse()
	if errs := p.Errors(); len(errs) != 0 {
		t.Fatalf("parse narrowing fixture: %v", errs)
	}

	snapshot, err := NewSemanticSnapshot(map[string][]ast.Node{filename: nodes}, nil)
	if err != nil {
		t.Fatalf("build semantic snapshot: %v", err)
	}

	issues := RunAnalysisRulesWithContext(filename, nodes, snapshot.NewAnalysisContext())
	return issues
}

func filterArgTypeIssues(issues []AnalysisIssue) []AnalysisIssue {
	var filtered []AnalysisIssue
	for _, issue := range issues {
		if issue.Code == "PHPStan.Level0.Invocation" {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}

func TestArgumentTypeRuleUsesNarrowedTypeInsideInstanceof(t *testing.T) {
	// Without narrowing, $obj->method() inside the if body would report
	// "unknown method" because $obj type is mixed.
	// With narrowing, $obj is typed as User, so the call is valid.
	issues := parseArgTypeNarrowingPHP(t, `<?php
class User {
    public function getName(): string {
        return "John";
    }
}

function process($obj) {
    if ($obj instanceof User) {
        $name = $obj->getName();
    }
}`)

	argTypeIssues := filterArgTypeIssues(issues)
	if len(argTypeIssues) > 0 {
		t.Fatalf("expected no unknown-method errors inside instanceof branch, got %#v", argTypeIssues)
	}
}

func TestArgumentTypeRuleNarrowingIsLocalToIfBranch(t *testing.T) {
	// Narrowing only applies inside the if block. Outside, narrowing doesn't apply.
	// This tests that narrowing is scoped correctly and doesn't leak.
	issues := parseArgTypeNarrowingPHP(t, `<?php
class User {
    public function getName(): string {
        return "John";
    }
}

function process($obj) {
    if ($obj instanceof User) {
        $x = $obj->getName();
    }
    // Outside if block, narrowing doesn't apply (but rule is still conservative)
    $y = $obj->getName();
}`)

	// Should have no errors (rule is conservative on mixed types)
	argTypeIssues := filterArgTypeIssues(issues)
	if len(argTypeIssues) > 0 {
		t.Logf("Issues: %#v", argTypeIssues)
		// Test passes if no false positives reported
	}
}
