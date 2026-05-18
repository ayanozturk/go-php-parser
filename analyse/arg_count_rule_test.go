package analyse

import "testing"

func hasArgCountIssue(issues []AnalysisIssue) bool {
	for _, iss := range issues {
		if iss.Code == argumentCountRuleCode {
			return true
		}
	}
	return false
}

func TestMethodArgumentCountMismatch(t *testing.T) {
	php := `<?php
class Example {
	public function takesTwo(string $name, int $count): void {
	}

	public function run(): void {
		$this->takesTwo("ok");
	}
}`
	issues := analysePHP(t, php)
	if !hasArgCountIssue(issues) {
		t.Fatalf("expected A.ARG.COUNT issue for missing method argument, got: %#v", issues)
	}
}

func TestConstructorArgumentCountMismatch(t *testing.T) {
	php := `<?php
class Example {
	public function __construct(string $name, int $count) {
	}

	public function build(): void {
		new self("ok", 1, 2);
	}
}`
	issues := analysePHP(t, php)
	if !hasArgCountIssue(issues) {
		t.Fatalf("expected A.ARG.COUNT issue for extra constructor argument, got: %#v", issues)
	}
}

func TestVariadicAndOptionalArgumentCountAccepted(t *testing.T) {
	php := `<?php
class Example {
	public function takesMany(string $name, ?int $count = null, string ...$extra): void {
	}

	public function run(): void {
		$this->takesMany("ok");
		$this->takesMany("ok", 1, "a", "b");
	}
}`
	issues := analysePHP(t, php)
	if hasArgCountIssue(issues) {
		t.Fatalf("expected no A.ARG.COUNT issue for optional and variadic args, got: %#v", issues)
	}
}

func TestInheritedExceptionConstructorArgumentCountAccepted(t *testing.T) {
	php := `<?php
class SubscriptionReactivationException extends Exception
{
	public static function invalidStatus(string $status): self
	{
		return new self("Subscription cannot be reactivated");
	}
}`
	issues := analysePHP(t, php)
	if hasArgCountIssue(issues) {
		t.Fatalf("expected no A.ARG.COUNT issue for inherited Exception constructor, got: %#v", issues)
	}
}

func TestInheritedExceptionConstructorTooManyArgumentsReported(t *testing.T) {
	php := `<?php
class SubscriptionReactivationException extends Exception
{
	public static function invalidStatus(string $status): self
	{
		return new self("message", 0, null, "extra");
	}
}`
	issues := analysePHP(t, php)
	if !hasArgCountIssue(issues) {
		t.Fatalf("expected A.ARG.COUNT issue for too many Exception constructor args, got: %#v", issues)
	}
}
