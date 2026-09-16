package style

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/syntax"
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
	public function set_name() {}
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
	public function GetData();  // valid under PHPCS PSR-1
    public function set_name(); // invalid
}
`
	issues := runMethodCamelCaseAnalysis(t, php)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue for invalid interface methods, got %d: %v", len(issues), issues)
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
	if len(issues) != 0 {
		t.Fatalf("expected no issues for PascalCase method name, got %d: %v", len(issues), issues)
	}
}

func TestMethodCamelCaseGetInstance(t *testing.T) {
	php := `<?php
class Session {
	public static function GetInstance() {}
}
`
	issues := runMethodCamelCaseAnalysis(t, php)
	if len(issues) != 0 {
		t.Fatalf("expected no issues for GetInstance, got %d: %v", len(issues), issues)
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

func TestMethodCamelCaseInvalidNameSpan(t *testing.T) {
	php := `<?php
class TestClass {
	public function set_name() {}
}
`
	issues := runMethodCamelCaseAnalysis(t, php)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d: %v", len(issues), issues)
	}
	issue := issues[0]
	if issue.Line != 3 || issue.Column != 18 || issue.EndLine != 3 || issue.EndColumn != 26 {
		t.Errorf("expected Line:3 Column:18 EndLine:3 EndColumn:26, got Line:%d Column:%d EndLine:%d EndColumn:%d",
			issue.Line, issue.Column, issue.EndLine, issue.EndColumn)
	}
}

func TestMethodCamelCaseInvalidNameSpanSameLine(t *testing.T) {
	php := `<?php
class C { public function set_name() {} }
`
	issues := runMethodCamelCaseAnalysis(t, php)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d: %v", len(issues), issues)
	}
	issue := issues[0]
	if issue.Line != 2 || issue.Column != 27 || issue.EndLine != 2 || issue.EndColumn != 35 {
		t.Errorf("expected Line:2 Column:27 EndLine:2 EndColumn:35, got Line:%d Column:%d EndLine:%d EndColumn:%d",
			issue.Line, issue.Column, issue.EndLine, issue.EndColumn)
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

	nodes, diags := syntax.ParseAST([]byte(php))
	if len(diags) != 0 {
		t.Fatalf("parser errors: %v", diags)
	}

	return RunSelectedRules("test.php", []byte(php), nodes, []string{psr1MethodCamelCaseCode})
}
