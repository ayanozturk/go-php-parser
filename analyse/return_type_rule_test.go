package analyse

import (
	"go-phpcs/lexer"
	"go-phpcs/parser"
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
