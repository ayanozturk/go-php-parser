package analyse

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/syntax"
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

func TestSideEffectsAttributeBeforeClassIsDeclarationOnly(t *testing.T) {
	php := `<?php
#[AllowMockObjectsWithoutExpectations]
class MyTest {
}
`
	issues := runSideEffectsAnalysis(t, php)
	if len(issues) != 0 {
		t.Fatalf("expected no issues for attributed class declaration, got %d: %v", len(issues), issues)
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

func TestSideEffectsUseAndClassDeclarationOnly(t *testing.T) {
	php := `<?php
namespace MyNamespace;

use Foo\Bar;

final class MyClass {}
`
	issues := runSideEffectsAnalysis(t, php)
	if len(issues) != 0 {
		t.Fatalf("expected no issues for namespace/use/class declarations, got %d: %v", len(issues), issues)
	}
}

func TestSideEffectsClassMethodInstantiationNotTopLevel(t *testing.T) {
	php := `<?php
class MyClass {
    /**
     * Returns a new instance.
     */
    public function make() {
        return new OtherClass();
    }
}
`
	issues := runSideEffectsAnalysis(t, php)
	if len(issues) != 0 {
		t.Fatalf("expected no issues for instantiation inside class method, got %d: %v", len(issues), issues)
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
	nodes, diags := syntax.ParseAST([]byte(php))
	if len(diags) > 0 {
		t.Fatalf("parser errors: %v", diags)
	}

	rule := &SideEffectsRule{}
	return rule.CheckIssuesWithSource("test.php", []byte(php), nodes)
}

func TestSideEffectsCSTMatchesLoweredPath(t *testing.T) {
	fixtures := []string{
		"<?php\nclass MyClass {\n    public function method() {}\n}\n\nfunction myFunction() {}\n\nconst MY_CONSTANT = 'value';\n",
		"<?php\n#[AllowMockObjectsWithoutExpectations]\nclass MyTest {\n}\n",
		"<?php\necho \"Hello World\";\n$x = 42;\nfile_put_contents('file.txt', 'content');\n",
		"<?php\nclass MyClass {\n    public function method() {}\n}\n\necho \"Hello World\";\n",
		"<?php\nclass MyClass {}\n\nnew MyClass();\n",
		"<?php\nfunction myFunction() {}\n\necho \"test\";\n",
		"<?php",
		"<?php\n// echo \"comment\";\n/* echo \"block comment\"; */\nclass MyClass {}\n",
		"<?php\nclass MyClass {}\n\n$code = 'echo \"string\";';\n",
		"<?php\nnamespace MyNamespace;\n\nclass MyClass {}\nfunction myFunction() {}\n",
		"<?php\nnamespace MyNamespace;\n\nuse Foo\\Bar;\n\nfinal class MyClass {}\n",
		"<?php\nclass MyClass {\n    public function make() {\n        return new OtherClass();\n    }\n}\n",
		"<?php\ninterface MyInterface {}\n\necho \"test\";\n",
		"<?php\ntrait MyTrait {}\n\nprint \"test\";\n",
		"<?php\nclass MyClass {}\n\nheader('Content-Type: application/json');\nsetcookie('session', 'value');\nsession_start();\nmail('to@example.com', 'subject', 'body');\n",
		// Braced namespace form.
		"<?php\nnamespace MyNamespace {\n    class MyClass {}\n    echo \"test\";\n}\n",
		// declare(): bodyless form is never a side effect on its own.
		"<?php\ndeclare(strict_types=1);\n\nclass MyClass {}\n",
		// declare(): empty braced body is not a side effect (Body stays nil).
		"<?php\nclass MyClass {}\n\ndeclare(strict_types=1) {\n}\n",
		// declare(): non-empty braced body is unconditionally a side effect.
		"<?php\nclass MyClass {}\n\ndeclare(strict_types=1) {\n    echo \"test\";\n}\n",
	}

	rule := &SideEffectsRule{}
	for i, php := range fixtures {
		content := []byte(php)
		nodes, diags := syntax.ParseAST(content)
		if len(diags) > 0 {
			t.Fatalf("fixture %d: parser errors: %v", i, diags)
		}
		lowered := rule.CheckIssuesWithSource("test.php", content, nodes)
		cst := rule.CheckIssuesFromCST("test.php", content)
		if len(lowered) != len(cst) {
			t.Fatalf("fixture %d: issue count mismatch: lowered=%d cst=%d\nphp:\n%s", i, len(lowered), len(cst), php)
		}
		for j := range lowered {
			if lowered[j].Line != cst[j].Line || lowered[j].Column != cst[j].Column || lowered[j].Code != cst[j].Code {
				t.Fatalf("fixture %d: issue %d mismatch: lowered=%+v cst=%+v", i, j, lowered[j], cst[j])
			}
		}
	}
}
