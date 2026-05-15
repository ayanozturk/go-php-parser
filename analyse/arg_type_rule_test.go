package analyse

import "testing"

func hasArgTypeIssue(issues []AnalysisIssue) bool {
	for _, iss := range issues {
		if iss.Code == "A.ARG.TYPE" {
			return true
		}
	}
	return false
}

func TestMethodArgumentTypeMismatch(t *testing.T) {
	php := `<?php
    class Example {
        public function takesInt(int $count): void {
        }

        public function run(): void {
            $this->takesInt("bad");
        }
    }`
	issues := analysePHP(t, php)
	if !hasArgTypeIssue(issues) {
		t.Fatalf("expected A.ARG.TYPE issue for string passed to int parameter, got: %#v", issues)
	}
}

func TestMethodArgumentTypeCompatible(t *testing.T) {
	php := `<?php
    class Example {
        public function takesFloat(float $value): void {
        }

        public function run(): void {
            $this->takesFloat(1);
        }
    }`
	issues := analysePHP(t, php)
	if hasArgTypeIssue(issues) {
		t.Fatalf("expected no A.ARG.TYPE issue for int passed to float parameter, got: %#v", issues)
	}
}
