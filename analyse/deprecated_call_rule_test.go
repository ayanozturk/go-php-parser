package analyse

import (
	"strings"
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

func TestDeprecatedCallReportsFunctionAndMethod(t *testing.T) {
	php := `<?php
/** @deprecated use neu() instead */
function alt(): void {}

class Legacy {
    /** @deprecated prefer neu */
    public function old(): void {}
}

function run(): void {
    alt();
    (new Legacy())->old();
}
`
	nodes, diags := syntax.ParseAST([]byte(php))
	if len(diags) > 0 {
		t.Fatalf("parser diagnostics: %v", diags)
	}
	project := BuildProjectIndex(map[string][]ast.Node{"test.php": nodes})
	ctx := &AnalysisContext{Resolver: project}
	issues := checkDeprecatedCalls(nodes, "test.php", ctx)

	var sawFunc, sawMethod bool
	for _, iss := range issues {
		if iss.Code != "A.DEPRECATED.CALL" {
			continue
		}
		if iss.Severity != "warning" {
			t.Fatalf("deprecated call should be a warning, got %#v", iss)
		}
		if iss.SubjectKind == "function" && strings.Contains(iss.Message, "`alt`") {
			sawFunc = true
			if !strings.Contains(iss.Message, "use neu() instead") {
				t.Fatalf("expected deprecation message appended, got %q", iss.Message)
			}
		}
		if iss.SubjectKind == "method" && strings.Contains(iss.SubjectName, "old") {
			sawMethod = true
			if !strings.Contains(iss.Message, "prefer neu") {
				t.Fatalf("expected method deprecation message, got %q", iss.Message)
			}
		}
	}
	if !sawFunc || !sawMethod {
		t.Fatalf("expected deprecated function and method issues, got %#v", issues)
	}
}

func TestDeprecatedCallIssueFormatsMessage(t *testing.T) {
	call := &ast.FunctionCallNode{Name: &ast.IdentifierNode{Value: "alt"}}
	iss := deprecatedCallIssue("f.php", call, "function", "alt", "gone")
	if iss.Code != "A.DEPRECATED.CALL" || iss.Severity != "warning" {
		t.Fatalf("unexpected issue %#v", iss)
	}
	if iss.SubjectKind != "function" || iss.SubjectName != "alt" {
		t.Fatalf("unexpected subject %#v", iss)
	}
	if !strings.Contains(iss.Message, "Call to deprecated function: `alt`.") || !strings.HasSuffix(iss.Message, "gone") {
		t.Fatalf("unexpected message %q", iss.Message)
	}
}
