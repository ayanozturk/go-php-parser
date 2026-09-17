package analyse

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

// helper to run analysis on a PHP snippet and return issues
func analysePHPCode(t *testing.T, code string) []AnalysisIssue {
	t.Helper()
	nodes, diags := syntax.ParseAST([]byte(code))
	if len(diags) > 0 {
		t.Fatalf("parser errors: %v", diags)
	}
	return RunAnalysisRules("test.php", nodes)
}

// debug helper to print parsed AST
func debugPHPCode(t *testing.T, code string) {
	t.Helper()
	nodes, diags := syntax.ParseAST([]byte(code))
	if len(diags) > 0 {
		t.Fatalf("parser errors: %v", diags)
	}
	for i, node := range nodes {
		t.Logf("Node %d: %s", i, node.String())
		if ifNode, ok := node.(*ast.IfNode); ok {
			t.Logf("  If condition: %s", ifNode.Condition.String())
			t.Logf("  ElseIfs count: %d", len(ifNode.ElseIfs))
			for j, elseif := range ifNode.ElseIfs {
				t.Logf("    ElseIf %d condition: %s", j, elseif.Condition.String())
			}
		}
	}
}

func hasAssignmentInConditionIssue(issues []AnalysisIssue) bool {
	for _, iss := range issues {
		if iss.Code == "Generic.CodeAnalysis.AssignmentInCondition" {
			return true
		}
	}
	return false
}

func countAssignmentInConditionIssues(issues []AnalysisIssue) int {
	count := 0
	for _, iss := range issues {
		if iss.Code == "Generic.CodeAnalysis.AssignmentInCondition" {
			count++
		}
	}
	return count
}

func TestAssignmentInIfCondition(t *testing.T) {
	php := `<?php
if ($result = doSomething()) {
    echo "Success";
}`
	issues := analysePHPCode(t, php)
	if !hasAssignmentInConditionIssue(issues) {
		t.Fatalf("expected Generic.CodeAnalysis.AssignmentInCondition issue for assignment in if condition, got: %#v", issues)
	}
	if countAssignmentInConditionIssues(issues) != 1 {
		t.Fatalf("expected 1 assignment in condition issue, got %d", countAssignmentInConditionIssues(issues))
	}
}

func TestAssignmentInElseIfCondition(t *testing.T) {
	php := `<?php
if ($x > 0) {
    echo "positive";
} elseif ($result = doSomething()) {
    echo "Success";
}
`
	issues := analysePHPCode(t, php)
	if !hasAssignmentInConditionIssue(issues) {
		t.Fatalf("expected Generic.CodeAnalysis.AssignmentInCondition issue for assignment in elseif condition, got: %#v", issues)
	}
}

func TestMultipleAssignmentsInConditions(t *testing.T) {
	php := `<?php
if ($a = func1()) {
    if ($b = func2()) {
        echo "both";
    }
}`
	issues := analysePHPCode(t, php)
	if countAssignmentInConditionIssues(issues) != 2 {
		t.Fatalf("expected 2 assignment in condition issues, got %d", countAssignmentInConditionIssues(issues))
	}
}

func TestNestedAssignmentInComplexExpression(t *testing.T) {
	php := `<?php
if (($result = doSomething()) && $result > 0) {
    echo "Success";
}`
	issues := analysePHPCode(t, php)
	if !hasAssignmentInConditionIssue(issues) {
		t.Fatalf("expected Generic.CodeAnalysis.AssignmentInCondition issue for nested assignment in complex expression, got: %#v", issues)
	}
}

func TestMultipleAssignmentsInSingleCondition(t *testing.T) {
	php := `<?php
if (($first = getFirst()) && ($second = getSecond())) {
    echo "both";
}`
	issues := analysePHPCode(t, php)
	if countAssignmentInConditionIssues(issues) != 2 {
		t.Fatalf("expected 2 assignment in condition issues, got %d", countAssignmentInConditionIssues(issues))
	}
}

func TestAssignmentInFunctionCallWithinCondition(t *testing.T) {
	php := `<?php
if (process($result = getData())) {
    echo "processed";
}`
	issues := analysePHPCode(t, php)
	if !hasAssignmentInConditionIssue(issues) {
		t.Fatalf("expected Generic.CodeAnalysis.AssignmentInCondition issue for assignment in function call within condition, got: %#v", issues)
	}
}

func TestValidComparisonInCondition(t *testing.T) {
	php := `<?php
$result = doSomething();
if ($result == 'success') {
    echo "Success";
}`
	issues := analysePHPCode(t, php)
	if hasAssignmentInConditionIssue(issues) {
		t.Fatalf("expected no Generic.CodeAnalysis.AssignmentInCondition issue for valid comparison, got: %#v", issues)
	}
}

func TestValidAssignmentOutsideCondition(t *testing.T) {
	php := `<?php
$result = doSomething();
if ($result) {
    echo "Success";
}`
	issues := analysePHPCode(t, php)
	if hasAssignmentInConditionIssue(issues) {
		t.Fatalf("expected no Generic.CodeAnalysis.AssignmentInCondition issue for assignment outside condition, got: %#v", issues)
	}
}

func TestCompoundAssignmentInCondition(t *testing.T) {
	php := `<?php
$x = 5;
if ($x += 10) {
    echo "x is now 15";
}`
	issues := analysePHPCode(t, php)
	if !hasAssignmentInConditionIssue(issues) {
		t.Fatalf("expected Generic.CodeAnalysis.AssignmentInCondition issue for compound assignment in condition, got: %#v", issues)
	}
}

func TestPropertyAssignmentInCondition(t *testing.T) {
	php := `<?php
class Test {
    public $prop;
}

$obj = new Test();
if ($obj->prop = 'value') {
    echo "assigned";
}`
	issues := analysePHPCode(t, php)
	if !hasAssignmentInConditionIssue(issues) {
		t.Fatalf("expected Generic.CodeAnalysis.AssignmentInCondition issue for property assignment in condition, got: %#v", issues)
	}
}

func TestAssignmentInConditionWithMultipleStatements(t *testing.T) {
	php := `<?php
function test() {
    if ($result = doSomething()) {
        echo "Success";
        return $result;
    }
    return null;
}`
	issues := analysePHPCode(t, php)
	if !hasAssignmentInConditionIssue(issues) {
		t.Fatalf("expected Generic.CodeAnalysis.AssignmentInCondition issue for assignment in condition within function, got: %#v", issues)
	}
}

func TestAssignmentInNestedIfConditions(t *testing.T) {
	php := `<?php
if ($outer = getOuter()) {
    if ($inner = getInner()) {
        echo "both assigned";
    }
}`
	issues := analysePHPCode(t, php)
	if countAssignmentInConditionIssues(issues) != 2 {
		t.Fatalf("expected 2 assignment in condition issues for nested if statements, got %d", countAssignmentInConditionIssues(issues))
	}
}

func TestValidTernaryWithoutAssignment(t *testing.T) {
	php := `<?php
$x = 5;
$result = $x > 0 ? 'positive' : 'negative';`
	issues := analysePHPCode(t, php)
	if hasAssignmentInConditionIssue(issues) {
		t.Fatalf("expected no Generic.CodeAnalysis.AssignmentInCondition issue for valid ternary without assignment, got: %#v", issues)
	}
}

func TestValidComplexConditionWithoutAssignment(t *testing.T) {
	php := `<?php
$a = 1;
$b = 2;
if ($a > 0 && $b < 10 || ($a + $b) == 3) {
    echo "complex condition";
}`
	issues := analysePHPCode(t, php)
	if hasAssignmentInConditionIssue(issues) {
		t.Fatalf("expected no Generic.CodeAnalysis.AssignmentInCondition issue for complex condition without assignment, got: %#v", issues)
	}
}

// TestAssignmentInConditionCSTMatchesLoweredPath guards the CST-direct
// CheckIssuesWithSource path against drifting from the ast.Node-based
// CheckIssues path (still the only one wired into the registered rule).
func TestAssignmentInConditionCSTMatchesLoweredPath(t *testing.T) {
	cases := []string{
		"<?php\nif ($result = doSomething()) {\n    echo $result;\n}\n",
		"<?php\nif ($a > 1) {\n    echo 1;\n} elseif ($b = compute()) {\n    echo $b;\n} else {\n    echo 2;\n}\n",
		"<?php\nwhile ($line = readLine()) {\n    echo $line;\n}\n",
		"<?php\ndo {\n    echo 1;\n} while ($x = next());\n",
		"<?php\nfor ($i = 0; $i < 10, $j = compute(); $i++) {\n    echo $i;\n}\n",
		"<?php\n$r = match ($x = compute()) {\n    1 => 'a',\n    default => 'c',\n};\n",
		"<?php\nif ($a = ($b = 1)) {\n    echo $a;\n}\n",
		"<?php\nif (($a = 1)) {\n    echo $a;\n}\n",
		"<?php\nif (foo($a = 1)) {\n    echo $a;\n}\n",
		"<?php\nif ($obj->method($a = 1)) {\n    echo 1;\n}\n",
		"<?php\nif ([$a = 1]) {\n    echo 1;\n}\n",
		"<?php\nif ((int)($a = 1)) {\n    echo $a;\n}\n",
		"<?php\nif ($a ? ($b = 1) : ($c = 2)) {\n    echo 1;\n}\n",
		"<?php\nclass Foo {\n    public function bar() {\n        if ($x = 1) {\n            return $x;\n        }\n    }\n}\n",
		"<?php\n$a = 1;\n$b = 2;\nif ($a > 0 && $b < 10 || ($a + $b) == 3) {\n    echo 1;\n}\n",
	}
	r := &AssignmentInConditionRule{}
	for _, php := range cases {
		content := []byte(php)
		cstIssues := r.CheckIssuesWithSource("test.php", content)
		nodes, diags := syntax.ParseAST(content)
		if len(diags) > 0 {
			t.Fatalf("parser errors for %q: %v", php, diags)
		}
		loweredIssues := r.CheckIssues(nodes, "test.php")
		if len(cstIssues) != len(loweredIssues) {
			t.Fatalf("issue count mismatch for %q: cst=%d lowered=%d (cst=%+v lowered=%+v)", php, len(cstIssues), len(loweredIssues), cstIssues, loweredIssues)
		}
		for i := range cstIssues {
			if cstIssues[i].Line != loweredIssues[i].Line || cstIssues[i].Column != loweredIssues[i].Column {
				t.Fatalf("position mismatch for %q at issue %d: cst=%+v lowered=%+v", php, i, cstIssues[i], loweredIssues[i])
			}
		}
	}
}

// TestAssignmentInConditionRuleHonorsContentContext verifies that
// runRegisteredAssignmentInConditionRule -- the exact function wired into
// the rule registry via RegisterAnalysisRuleWithContext, which is what
// production calls -- branches on ctx.Content: with Content set it must
// match CheckIssuesWithSource's direct output, and without Content it must
// match CheckIssues' ast.Node output.
func TestAssignmentInConditionRuleHonorsContentContext(t *testing.T) {
	php := "<?php\nif ($x = foo()) {\n    bar();\n}\n"
	content := []byte(php)
	nodes, diags := syntax.ParseAST(content)
	if len(diags) > 0 {
		t.Fatalf("parser errors: %v", diags)
	}

	rule := &AssignmentInConditionRule{}
	wantWithContent := rule.CheckIssuesWithSource("test.php", content)
	wantWithoutContent := rule.CheckIssues(nodes, "test.php")

	gotWithContent := runRegisteredAssignmentInConditionRule("test.php", nodes, &AnalysisContext{Content: content})
	gotWithoutContent := runRegisteredAssignmentInConditionRule("test.php", nodes, &AnalysisContext{})

	if len(gotWithContent) != len(wantWithContent) {
		t.Fatalf("with Content: got %d issues via registered rule, want %d (CheckIssuesWithSource)", len(gotWithContent), len(wantWithContent))
	}
	if len(gotWithoutContent) != len(wantWithoutContent) {
		t.Fatalf("without Content: got %d issues via registered rule, want %d (CheckIssues)", len(gotWithoutContent), len(wantWithoutContent))
	}
}
