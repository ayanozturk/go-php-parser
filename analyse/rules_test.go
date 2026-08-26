package analyse

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"reflect"
	"sort"
	"testing"
)

func TestListRegisteredAnalysisRuleCodes(t *testing.T) {
	ClearAnalysisRules()

	RegisterAnalysisRule("Z.TEST.RULE", func(filename string, nodes []ast.Node) []AnalysisIssue { return nil })
	RegisterAnalysisRule("A.TEST.RULE", func(filename string, nodes []ast.Node) []AnalysisIssue { return nil })

	codes := ListRegisteredAnalysisRuleCodes()
	if len(codes) != 2 {
		t.Fatalf("expected 2 codes, got %d", len(codes))
	}
	if !sort.StringsAreSorted(codes) {
		t.Errorf("codes are not sorted: %v", codes)
	}

	foundA, foundZ := false, false
	for _, c := range codes {
		if c == "A.TEST.RULE" {
			foundA = true
		}
		if c == "Z.TEST.RULE" {
			foundZ = true
		}
	}
	if !foundA || !foundZ {
		t.Errorf("expected both A.TEST.RULE and Z.TEST.RULE to be present, got %v", codes)
	}
}

func TestRunAnalysisRulesDeterministicOrder(t *testing.T) {
	ClearAnalysisRules()

	RegisterAnalysisRule("B.RULE", func(filename string, nodes []ast.Node) []AnalysisIssue {
		return []AnalysisIssue{{Filename: filename, Code: "B.RULE", Message: "B"}}
	})
	RegisterAnalysisRule("A.RULE", func(filename string, nodes []ast.Node) []AnalysisIssue {
		return []AnalysisIssue{{Filename: filename, Code: "A.RULE", Message: "A"}}
	})

	issues := RunAnalysisRules("test.php", nil)
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
	if issues[0].Code != "A.RULE" || issues[0].Message != "A" {
		t.Errorf("expected first issue from A.RULE, got %#v", issues[0])
	}
	if issues[1].Code != "B.RULE" || issues[1].Message != "B" {
		t.Errorf("expected second issue from B.RULE, got %#v", issues[1])
	}
}

func TestRunAnalysisRulesPreservesContextPHPVersion(t *testing.T) {
	ClearAnalysisRules()
	defer ClearAnalysisRules()

	RegisterAnalysisRuleWithContext("PHP.VERSION", func(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
		return []AnalysisIssue{{Filename: filename, Code: "PHP.VERSION", Message: ctx.PHPVersion}}
	})

	issues := RunAnalysisRulesWithContext("test.php", nil, &AnalysisContext{PHPVersion: "8.4"})
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].Message != "8.4" {
		t.Fatalf("expected PHP version 8.4 in context, got %q", issues[0].Message)
	}
}

func TestRunAnalysisRulesPreservesReadOnlySemanticContext(t *testing.T) {
	ClearAnalysisRules()
	defer ClearAnalysisRules()

	resolver := NewProjectIndex()
	facts := &countingFactReader{}
	flow := &stubFlowGraphReader{}
	variableFlow := singleFileVariableFlow{filename: "test.php"}
	RegisterAnalysisRuleWithContext("SEMANTIC.CONTEXT", func(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
		if ctx.Resolver != resolver {
			t.Fatal("registered rule did not receive the supplied symbol resolver")
		}
		if ctx.Facts != facts {
			t.Fatal("registered rule did not receive the supplied semantic fact reader")
		}
		if ctx.Flow != flow {
			t.Fatal("registered rule did not receive the supplied flow graph reader")
		}
		if !reflect.DeepEqual(ctx.VariableFlow, variableFlow) {
			t.Fatal("registered rule did not receive the supplied variable-flow reader")
		}
		return nil
	})

	RunAnalysisRulesWithContext("test.php", nil, &AnalysisContext{Resolver: resolver, Facts: facts, Flow: flow, VariableFlow: variableFlow})
}

type stubFlowGraphReader struct{}

func (*stubFlowGraphReader) StatementReachable(FlowStatementKey) (bool, bool) {
	return false, false
}

func (*stubFlowGraphReader) ScopeMayFallThrough(FlowScopeKey) (bool, bool) {
	return false, false
}

func (*stubFlowGraphReader) ControlFlowGraph(FlowScopeKey) (ControlFlowGraph, bool) {
	return ControlFlowGraph{}, false
}

func TestRunAnalysisRulesFiltersDisabledIssueCodes(t *testing.T) {
	ClearAnalysisRules()
	defer ClearAnalysisRules()

	RegisterAnalysisRule("GROUPED.RULE", func(filename string, nodes []ast.Node) []AnalysisIssue {
		return []AnalysisIssue{
			{Filename: filename, Code: "ENABLED", Message: "keep"},
			{Filename: filename, Code: "DISABLED", Message: "drop"},
		}
	})

	issues := RunAnalysisRulesWithContext("test.php", nil, &AnalysisContext{
		DisabledIssueCodes: map[string]bool{"DISABLED": true},
	})
	if len(issues) != 1 || issues[0].Code != "ENABLED" {
		t.Fatalf("expected only enabled issue, got %#v", issues)
	}
}

func TestClearAnalysisRules(t *testing.T) {
	ClearAnalysisRules()

	RegisterAnalysisRule("SOME.RULE", func(filename string, nodes []ast.Node) []AnalysisIssue { return nil })
	if len(ListRegisteredAnalysisRuleCodes()) != 1 {
		t.Fatalf("expected 1 rule registered")
	}

	ClearAnalysisRules()
	if len(ListRegisteredAnalysisRuleCodes()) != 0 {
		t.Fatalf("expected registry to be empty after ClearAnalysisRules")
	}
}
