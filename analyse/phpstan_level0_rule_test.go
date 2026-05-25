package analyse

import (
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"go-phpcs/parser"
	"strings"
	"testing"
)

func parsePHPForLevel0(t *testing.T, php string) []ast.Node {
	t.Helper()
	p := parser.New(lexer.New(php), false)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	return nodes
}

func runLevel0OnFiles(t *testing.T, files map[string]string) []AnalysisIssue {
	t.Helper()
	parsed := make(map[string][]ast.Node, len(files))
	for filename, php := range files {
		parsed[filename] = parsePHPForLevel0(t, php)
	}
	project := BuildProjectIndex(parsed)
	level := 0
	var issues []AnalysisIssue
	for filename, nodes := range parsed {
		ctx := &AnalysisContext{Resolver: project, Project: project, AnalysisLevel: &level}
		issues = append(issues, RunAnalysisRulesWithContext(filename, nodes, ctx)...)
	}
	return issues
}

func hasIssueContaining(issues []AnalysisIssue, code, needle string) bool {
	for _, issue := range issues {
		if issue.Code == code && strings.Contains(issue.Message, needle) {
			return true
		}
	}
	return false
}

func TestLevel0UnknownSymbolsAndFunctionArguments(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
known(tooMany: 1, 2);
missing_function();
new MissingClass();

function known($a) {}
`,
	})

	if !hasIssueContaining(issues, level0SymbolsCode, "Function missing_function not found") {
		t.Fatalf("expected unknown function issue, got %#v", issues)
	}
	if !hasIssueContaining(issues, level0SymbolsCode, "Instantiated class MissingClass not found") {
		t.Fatalf("expected unknown class issue, got %#v", issues)
	}
	if !hasIssueContaining(issues, level0InvocationCode, "Named argument cannot be followed by a positional argument") {
		t.Fatalf("expected named argument order issue, got %#v", issues)
	}
	if !hasIssueContaining(issues, level0InvocationCode, "Unknown parameter $tooMany") {
		t.Fatalf("expected unknown named parameter issue, got %#v", issues)
	}
}

func TestLevel0ClassModelAndMethodChecks(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
final class Base {}
class Child extends Base {}
class UsesMissing implements MissingInterface {}
class Calls {
    public function run() {
        $this->missing();
    }
}
`,
	})

	if !hasIssueContaining(issues, level0ClassModelCode, "extends final class Base") {
		t.Fatalf("expected final class extension issue, got %#v", issues)
	}
	if !hasIssueContaining(issues, level0ClassModelCode, "implements unknown interface MissingInterface") {
		t.Fatalf("expected unknown interface issue, got %#v", issues)
	}
	if !hasIssueContaining(issues, level0SymbolsCode, "Call to an undefined method Calls::missing") {
		t.Fatalf("expected undefined $this method issue, got %#v", issues)
	}
}

func TestLevel0CrossFileIndexResolvesKnownSymbols(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"a.php": `<?php
namespace App;
class Service {
    public function work($a) {}
}
function helper($a) {}
`,
		"b.php": `<?php
namespace App;
$s = new Service();
$s->work(1);
helper(1);
`,
	})

	for _, issue := range issues {
		if strings.Contains(issue.Message, "Service not found") || strings.Contains(issue.Message, "helper not found") {
			t.Fatalf("expected cross-file symbols to resolve, got %#v", issues)
		}
	}
}

func TestLevel0UndefinedVariablesAndLanguageChecks(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
echo $missing;
goto nowhere;
$a = ['x' => 1, 'x' => 2];
`,
	})

	if !hasIssueContaining(issues, level0VariablesCode, "Undefined variable: $missing") {
		t.Fatalf("expected undefined variable issue, got %#v", issues)
	}
	if !hasIssueContaining(issues, level0LanguageCode, "Goto to undefined label nowhere") {
		t.Fatalf("expected undefined label issue, got %#v", issues)
	}
	if !hasIssueContaining(issues, level0LanguageCode, "duplicate key") {
		t.Fatalf("expected duplicate array key issue, got %#v", issues)
	}
}

func TestAnalysisLevel0DoesNotRunHigherLevelRules(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
function bad(): int {
    return "x";
}
`,
	})

	for _, issue := range issues {
		if issue.Code == "A.RETURN.TYPE" || issue.Code == "A.PROP.TYPE" || issue.Code == "A.ARG.TYPE" || issue.Code == "Generic.CodeAnalysis.UnreachableCode" {
			t.Fatalf("higher-level issue emitted at level 0: %#v", issues)
		}
	}
}
