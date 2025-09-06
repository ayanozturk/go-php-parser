package style

import (
	"go-phpcs/lexer"
	"go-phpcs/parser"
	"testing"
)

func TestClassConstantValidNames(t *testing.T) {
	php := `<?php
class TestClass {
    public const FOO = 'value';
    public const BAR_BAZ = 123;
    public const MY_CONSTANT = true;
    public const API_KEY = 'key';
    public const MAX_SIZE = 100;
    public const DEFAULT_VALUE = null;
}
`
	issues := runClassConstantAnalysis(t, php)
	if len(issues) != 0 {
		t.Fatalf("expected no issues for valid constant names, got %d: %v", len(issues), issues)
	}
}

func TestClassConstantInvalidNames(t *testing.T) {
	php := `<?php
class TestClass {
    public const foo = 'value';
    public const barBaz = 123;
}
`
	issues := runClassConstantAnalysis(t, php)
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues for invalid constant names, got %d: %v", len(issues), issues)
	}
	for _, issue := range issues {
		if issue.Code != psr1ClassConstantNameCode {
			t.Errorf("expected %s, got %s", psr1ClassConstantNameCode, issue.Code)
		}
	}
}

func TestClassConstantWithVisibility(t *testing.T) {
	php := `<?php
class TestClass {
    public const PUBLIC_CONST = 'public';
    protected const PROTECTED_CONST = 'protected';
    private const PRIVATE_CONST = 'private';
    const NO_VISIBILITY = 'none';
}
`
	issues := runClassConstantAnalysis(t, php)
	if len(issues) != 0 {
		t.Fatalf("expected no issues for constants with various visibility, got %d: %v", len(issues), issues)
	}
}

func TestClassConstantEdgeCases(t *testing.T) {
	php := `<?php
class TestClass {
    public const _STARTS_WITH_UNDERSCORE = 'invalid';
    public const ENDS_WITH_UNDERSCORE_ = 'invalid';
    public const DOUBLE__UNDERSCORE = 'invalid';
    public const SINGLE = 'valid';
    public const A = 'valid';
    public const A1 = 'valid';
    public const A_1 = 'valid';
}
`
	issues := runClassConstantAnalysis(t, php)
	// Should have 3 invalid constants: _STARTS_WITH_UNDERSCORE, ENDS_WITH_UNDERSCORE_, DOUBLE__UNDERSCORE
	if len(issues) != 3 {
		t.Fatalf("expected 3 issues for edge case constants, got %d: %v", len(issues), issues)
	}
}

func TestClassConstantWithTypes(t *testing.T) {
	php := `<?php
class TestClass {
    public const STRING_CONST = 'value';
    public const INT_CONST = 123;
    public const BOOL_CONST = true;
}
`
	issues := runClassConstantAnalysis(t, php)
	if len(issues) != 0 {
		t.Fatalf("expected no issues for typed constants, got %d: %v", len(issues), issues)
	}
}

func TestTraitConstants(t *testing.T) {
	php := `<?php
trait TestTrait {
    public const TRAIT_CONST = 'trait';
}
`
	issues := runClassConstantAnalysis(t, php)
	if len(issues) != 0 {
		t.Fatalf("expected no issues for valid trait constant, got %d: %v", len(issues), issues)
	}
}

func TestClassConstantComplexValues(t *testing.T) {
	php := `<?php
class TestClass {
    public const SIMPLE_STRING = 'hello';
    public const invalidName = 'should fail';
}
`
	issues := runClassConstantAnalysis(t, php)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue for invalid constant name, got %d: %v", len(issues), issues)
	}
}

func TestEmptyClass(t *testing.T) {
	php := `<?php
class EmptyClass {
}
`
	issues := runClassConstantAnalysis(t, php)
	if len(issues) != 0 {
		t.Fatalf("expected no issues for class without constants, got %d: %v", len(issues), issues)
	}
}

func runClassConstantAnalysis(t *testing.T, php string) []StyleIssue {
	t.Helper()

	// Parse the PHP code to get AST nodes
	l := lexer.New(php)
	p := parser.New(l, true) // Use debug mode for better parsing
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	return RunSelectedRules("test.php", []byte(php), nodes, []string{psr1ClassConstantNameCode})
}
