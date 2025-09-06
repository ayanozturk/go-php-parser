package style

import (
	"go-phpcs/lexer"
	"go-phpcs/parser"
	"testing"
)

func TestMethodCamelCaseValidNames(t *testing.T) {
	php := `<?php
class TestClass {
    public function getUser() {}
    public function setName() {}
    public function calculateTotal() {}
    public function isValid() {}
    public function hasPermission() {}
    public function findById() {}
    public function saveData() {}
}
`
	issues := runMethodCamelCaseAnalysis(t, php)
	if len(issues) != 0 {
		t.Fatalf("expected no issues for valid camelCase methods, got %d: %v", len(issues), issues)
	}
}

func TestMethodCamelCaseInvalidNames(t *testing.T) {
	php := `<?php
class TestClass {
    public function GetUser() {}
}
`
	issues := runMethodCamelCaseAnalysis(t, php)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue for invalid method name, got %d: %v", len(issues), issues)
	}
	for _, issue := range issues {
		if issue.Code != psr1MethodCamelCaseCode {
			t.Errorf("expected %s, got %s", psr1MethodCamelCaseCode, issue.Code)
		}
	}
}

func TestMethodCamelCaseMagicMethods(t *testing.T) {
	php := `<?php
class TestClass {
    public function __construct() {}
    public function __destruct() {}
    public function __call() {}
    public function __get() {}
    public function __set() {}
    public function __toString() {}
}
`
	issues := runMethodCamelCaseAnalysis(t, php)
	if len(issues) != 0 {
		t.Fatalf("expected no issues for magic methods, got %d: %v", len(issues), issues)
	}
}

func TestMethodCamelCaseInterfaceMethods(t *testing.T) {
	php := `<?php
interface TestInterface {
    public function getData();
    public function GetData();  // invalid
    public function set_name(); // invalid
}
`
	issues := runMethodCamelCaseAnalysis(t, php)
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues for invalid interface methods, got %d: %v", len(issues), issues)
	}
}

func TestMethodCamelCaseTraitMethods(t *testing.T) {
	php := `<?php
trait TestTrait {
    public function Process_data() {}
}
`
	issues := runMethodCamelCaseAnalysis(t, php)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue for invalid trait method, got %d: %v", len(issues), issues)
	}
}

func TestMethodCamelCasePascalCase(t *testing.T) {
	php := `<?php
class TestClass {
    public function GetUser() {}
}
`
	issues := runMethodCamelCaseAnalysis(t, php)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue for PascalCase method name, got %d: %v", len(issues), issues)
	}
}

func TestMethodCamelCaseWithNumbers(t *testing.T) {
	php := `<?php
class TestClass {
    public function getUser2() {}
}
`
	issues := runMethodCamelCaseAnalysis(t, php)
	if len(issues) != 0 {
		t.Fatalf("expected no issues for valid camelCase method with number, got %d: %v", len(issues), issues)
	}
}

func TestMethodCamelCaseEmptyClass(t *testing.T) {
	php := `<?php
class TestClass {
}
`
	issues := runMethodCamelCaseAnalysis(t, php)
	if len(issues) != 0 {
		t.Fatalf("expected no issues for class without methods, got %d: %v", len(issues), issues)
	}
}

func TestMethodCamelCaseSnakeCase(t *testing.T) {
	php := `<?php
class TestClass {
    public function set_name() {}
}
`
	issues := runMethodCamelCaseAnalysis(t, php)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue for snake_case method name, got %d: %v", len(issues), issues)
	}
	if issues[0].Code != psr1MethodCamelCaseCode {
		t.Errorf("expected %s, got %s", psr1MethodCamelCaseCode, issues[0].Code)
	}
}

func runMethodCamelCaseAnalysis(t *testing.T, php string) []StyleIssue {
	t.Helper()

	// Parse the PHP code to get AST nodes
	l := lexer.New(php)
	p := parser.New(l, true) // Use debug mode for better trait parsing
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	return RunSelectedRules("test.php", []byte(php), nodes, []string{psr1MethodCamelCaseCode})
}
