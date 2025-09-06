package analyse

import (
	"go-phpcs/lexer"
	"go-phpcs/parser"
	"testing"
)

func TestSideEffectsOnlyDeclarations(t *testing.T) {
	php := `<?php
class MyClass {
    public function method() {}
}

function myFunction() {}

const MY_CONSTANT = 'value';
`
	issues := runSideEffectsAnalysis(t, php)
	if len(issues) != 0 {
		t.Fatalf("expected no issues for declarations-only file, got %d: %v", len(issues), issues)
	}
}

func TestSideEffectsOnlySideEffects(t *testing.T) {
	php := `<?php
echo "Hello World";
$x = 42;
file_put_contents('file.txt', 'content');
`
	issues := runSideEffectsAnalysis(t, php)
	if len(issues) != 0 {
		t.Fatalf("expected no issues for side-effects-only file, got %d: %v", len(issues), issues)
	}
}

func TestSideEffectsBothDeclarationsAndSideEffects(t *testing.T) {
	php := `<?php
class MyClass {
    public function method() {}
}

echo "Hello World";
`
	issues := runSideEffectsAnalysis(t, php)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue for mixed file, got %d: %v", len(issues), issues)
	}
	if issues[0].Code != "PSR1.Files.SideEffects" {
		t.Fatalf("expected PSR1.Files.SideEffects issue, got %s", issues[0].Code)
	}
}

func TestSideEffectsClassWithInstantiation(t *testing.T) {
	php := `<?php
class MyClass {}

new MyClass();
`
	issues := runSideEffectsAnalysis(t, php)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue for class with instantiation, got %d: %v", len(issues), issues)
	}
}

func TestSideEffectsFunctionWithEcho(t *testing.T) {
	php := `<?php
function myFunction() {}

echo "test";
`
	issues := runSideEffectsAnalysis(t, php)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue for function with echo, got %d: %v", len(issues), issues)
	}
}

func TestSideEffectsEmptyFile(t *testing.T) {
	php := `<?php`
	issues := runSideEffectsAnalysis(t, php)
	if len(issues) != 0 {
		t.Fatalf("expected no issues for empty file, got %d: %v", len(issues), issues)
	}
}

func TestSideEffectsCommentsIgnored(t *testing.T) {
	php := `<?php
// echo "comment";
/* echo "block comment"; */
class MyClass {}
`
	issues := runSideEffectsAnalysis(t, php)
	if len(issues) != 0 {
		t.Fatalf("expected no issues when side effects are in comments, got %d: %v", len(issues), issues)
	}
}

func TestSideEffectsStringsIgnored(t *testing.T) {
	php := `<?php
class MyClass {}

$code = 'echo "string";';
`
	issues := runSideEffectsAnalysis(t, php)
	if len(issues) != 0 {
		t.Fatalf("expected no issues when side effects are in strings, got %d: %v", len(issues), issues)
	}
}

func TestSideEffectsNamespaceWithDeclarations(t *testing.T) {
	php := `<?php
namespace MyNamespace;

class MyClass {}
function myFunction() {}
`
	issues := runSideEffectsAnalysis(t, php)
	if len(issues) != 0 {
		t.Fatalf("expected no issues for namespace with declarations, got %d: %v", len(issues), issues)
	}
}

func TestSideEffectsInterfaceDeclaration(t *testing.T) {
	php := `<?php
interface MyInterface {}

echo "test";
`
	issues := runSideEffectsAnalysis(t, php)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue for interface with side effect, got %d: %v", len(issues), issues)
	}
}

func TestSideEffectsTraitDeclaration(t *testing.T) {
	php := `<?php
trait MyTrait {}

print "test";
`
	issues := runSideEffectsAnalysis(t, php)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue for trait with side effect, got %d: %v", len(issues), issues)
	}
}

func TestSideEffectsComplexSideEffects(t *testing.T) {
	php := `<?php
class MyClass {}

header('Content-Type: application/json');
setcookie('session', 'value');
session_start();
mail('to@example.com', 'subject', 'body');
`
	issues := runSideEffectsAnalysis(t, php)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue for complex side effects, got %d: %v", len(issues), issues)
	}
}

func runSideEffectsAnalysis(t *testing.T, php string) []AnalysisIssue {
	t.Helper()

	// Parse the PHP code to get AST nodes
	l := lexer.New(php)
	p := parser.New(l, false)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	rule := &SideEffectsRule{}
	return rule.CheckIssuesWithSource("test.php", []byte(php), nodes)
}
