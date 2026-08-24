package analyse

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/lexer"
	"github.com/ayanozturk/go-php-parser/parser"
	"testing"
)

// helper to run analysis on a PHP snippet and return issues
func analysePHP(t *testing.T, code string) []AnalysisIssue {
	t.Helper()
	l := lexer.New(code)
	p := parser.New(l, false)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	return RunAnalysisRules("test.php", nodes)
}

func hasReturnTypeIssue(issues []AnalysisIssue) bool {
	for _, iss := range issues {
		if iss.Code == "A.RETURN.TYPE" {
			return true
		}
	}
	return false
}

func TestImplodeReturnsStringNoMismatch(t *testing.T) {
	php := `<?php
    function foo(): string {
        $arr = ["a", "b"];
        return implode("\n", $arr);
    }`
	issues := analysePHP(t, php)
	if hasReturnTypeIssue(issues) {
		t.Fatalf("expected no A.RETURN.TYPE issue for implode returning string, got: %#v", issues)
	}
}

func TestShortArrayLiteralReturnMatchesArrayType(t *testing.T) {
	php := `<?php
    function values(): array {
        return ['name'];
    }`
	issues := analysePHP(t, php)
	if hasReturnTypeIssue(issues) {
		t.Fatalf("expected no A.RETURN.TYPE issue for short array literal returned as array, got: %#v", issues)
	}
}

func TestMultipleCompatibleTypesNoError(t *testing.T) {
	php := `<?php
    function bar(): bool {
        if ($x) {
            return true;
        }
        return $y; // mixed
    }`
	issues := analysePHP(t, php)
	if hasReturnTypeIssue(issues) {
		t.Fatalf("expected no A.RETURN.TYPE issue when actual types are [bool,mixed] declared bool, got: %#v", issues)
	}
}

func TestAssignedVariableReturnTypeNoMismatch(t *testing.T) {
	php := `<?php
	function foo(): string {
		$value = "ok";
		return $value;
	}`
	issues := analysePHP(t, php)
	if hasReturnTypeIssue(issues) {
		t.Fatalf("expected no A.RETURN.TYPE issue for assigned local string, got: %#v", issues)
	}
}

func TestNewExpressionClassReturnTypeNoMismatch(t *testing.T) {
	php := `<?php
	class User {}

	function makeUser(): User {
		return new User();
	}`
	issues := analysePHP(t, php)
	if hasReturnTypeIssue(issues) {
		t.Fatalf("expected no A.RETURN.TYPE issue for new User return, got: %#v", issues)
	}
}

func TestThisPropertyReturnTypeNoMismatch(t *testing.T) {
	php := `<?php
	class User {}

	class UserRepository {
		private User $user;

		public function current(): User {
			return $this->user;
		}
	}`
	issues := analysePHP(t, php)
	if hasReturnTypeIssue(issues) {
		t.Fatalf("expected no A.RETURN.TYPE issue for typed property fetch, got: %#v", issues)
	}
}

func TestThisMethodReturnTypeNoMismatch(t *testing.T) {
	php := `<?php
	class User {}

	class UserRepository {
		public function current(): User {
			return $this->loadUser();
		}

		private function loadUser(): User {
			return new User();
		}
	}`
	issues := analysePHP(t, php)
	if hasReturnTypeIssue(issues) {
		t.Fatalf("expected no A.RETURN.TYPE issue for same-class method return, got: %#v", issues)
	}
}

func TestPromotedPropertyReturnTypeNoMismatch(t *testing.T) {
	php := `<?php
	class SessionStore {}

	class Session
	{
		public function __construct(private SessionStore $session)
		{
		}

		public function store(): SessionStore
		{
			return $this->session;
		}
	}`
	issues := analysePHP(t, php)
	if hasReturnTypeIssue(issues) {
		t.Fatalf("expected no A.RETURN.TYPE issue for promoted property fetch, got: %#v", issues)
	}
}

func TestLazyInitPropertyReturnTypeNoMismatch(t *testing.T) {
	// Lazy-init pattern: $this->prop is ?Type, assigned inside
	// `if (null === $this->prop)`, then returned as Type.
	php := `<?php
class MemberService {}
class Example {
    private ?MemberService $memberService = null;

    public function getMemberService(): MemberService
    {
        if (null === $this->memberService) {
            $this->memberService = new MemberService();
        }
        return $this->memberService;
    }
}`
	issues := analysePHP(t, php)
	if hasReturnTypeIssue(issues) {
		t.Fatalf("expected no A.RETURN.TYPE issue for lazy-init property, got: %#v", issues)
	}
}

func TestReturnTypeRuleUsesSnapshotInferredTypeFact(t *testing.T) {
	const filename = "src/Answer.php"
	nodes, expression := parseReturnFactFixture(t, `<?php
function answer(): string {
    return 42;
}
`)
	rule := &ReturnTypeRule{}
	if issues := rule.CheckIssues(nodes, filename, &AnalysisContext{}); !hasReturnTypeIssue(issues) {
		t.Fatalf("expected ordinary inference to report int returned as string, got %#v", issues)
	}

	key := inferredTypeFactKey(filename, expression)
	snapshot, err := NewSemanticSnapshot(map[string][]ast.Node{filename: nodes}, []SemanticFact{{Key: key, Type: "string"}})
	if err != nil {
		t.Fatalf("build semantic snapshot: %v", err)
	}
	if issues := rule.CheckIssues(nodes, filename, snapshot.NewAnalysisContext()); hasReturnTypeIssue(issues) {
		t.Fatalf("expected shared inferred-type fact to satisfy declared return type, got %#v", issues)
	}
}

func TestReturnTypeRuleIgnoresMismatchedAndEmptyInferredTypeFacts(t *testing.T) {
	const filename = "src/Answer.php"
	nodes, expression := parseReturnFactFixture(t, `<?php
function answer(): string {
    return 42;
}
`)
	key := inferredTypeFactKey(filename, expression)
	mismatched := key
	mismatched.StartOffset++
	reader := &countingFactReader{facts: map[SemanticFactKey]SemanticFact{
		mismatched: {Key: mismatched, Type: "string"},
	}}

	issues := (&ReturnTypeRule{}).CheckIssues(nodes, filename, &AnalysisContext{Facts: reader})
	if !hasReturnTypeIssue(issues) {
		t.Fatalf("expected nonmatching fact span to fall back to ordinary inference, got %#v", issues)
	}
	if reader.lookups != 1 {
		t.Fatalf("expected one exact fact lookup, got %d", reader.lookups)
	}

	reader.facts[key] = SemanticFact{Key: key}
	issues = (&ReturnTypeRule{}).CheckIssues(nodes, filename, &AnalysisContext{Facts: reader})
	if !hasReturnTypeIssue(issues) {
		t.Fatalf("expected empty inferred type fact to fall back to ordinary inference, got %#v", issues)
	}
}

type countingFactReader struct {
	facts   map[SemanticFactKey]SemanticFact
	lookups int
}

func (r *countingFactReader) Fact(key SemanticFactKey) (SemanticFact, bool) {
	r.lookups++
	fact, ok := r.facts[key]
	return fact, ok
}

func (r *countingFactReader) FactsForFile(string) []SemanticFact {
	return nil
}

func parseReturnFactFixture(t *testing.T, source string) ([]ast.Node, ast.Node) {
	t.Helper()
	p := parser.New(lexer.NewFile(source), false)
	nodes := p.Parse()
	if len(p.Errors()) != 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	fn, ok := nodes[0].(*ast.FunctionNode)
	if !ok || len(fn.Body) != 1 {
		t.Fatalf("expected one function with one statement, got %#v", nodes)
	}
	ret, ok := fn.Body[0].(*ast.ReturnNode)
	if !ok || ret.Expr == nil {
		t.Fatalf("expected return expression, got %#v", fn.Body[0])
	}
	return nodes, ret.Expr
}

func inferredTypeFactKey(filename string, expression ast.Node) SemanticFactKey {
	return SemanticFactKey{
		File:        filename,
		StartOffset: expression.GetPos().Offset,
		EndOffset:   expression.GetEndPos().Offset,
		Kind:        FactKindInferredType,
	}
}
