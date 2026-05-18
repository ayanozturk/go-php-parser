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

func TestMethodArgumentTypeRefinedAfterNullGuardThrow(t *testing.T) {
	php := `<?php
    class DocumentPolicy {
    }

    class Example {
        public function findPolicy(): ?DocumentPolicy {
            return new DocumentPolicy();
        }

        public function getVersionHistory(DocumentPolicy $policy): void {
        }

        public function belongsToWrongCompany(): bool {
            return false;
        }

        public function run(): void {
            $policy = $this->findPolicy();
            if (!$policy || $this->belongsToWrongCompany()) {
                throw $e;
            }

            $this->getVersionHistory($policy);
        }
    }`
	issues := analysePHP(t, php)
	if hasArgTypeIssue(issues) {
		t.Fatalf("expected no A.ARG.TYPE issue after terminating null guard, got: %#v", issues)
	}
}

func TestMethodArgumentTypeRefinedAfterExplicitNullGuardReturn(t *testing.T) {
	php := `<?php
    class DocumentPolicy {
    }

    class Example {
        public function findPolicy(): ?DocumentPolicy {
            return new DocumentPolicy();
        }

        public function getVersionHistory(DocumentPolicy $policy): void {
        }

        public function run(): void {
            $policy = $this->findPolicy();
            if ($policy === null) {
                return;
            }

            $this->getVersionHistory($policy);
        }
    }`
	issues := analysePHP(t, php)
	if hasArgTypeIssue(issues) {
		t.Fatalf("expected no A.ARG.TYPE issue after explicit null guard, got: %#v", issues)
	}
}

func TestMethodArgumentTypeNotRefinedAfterNonTerminatingNullCheck(t *testing.T) {
	php := `<?php
    class DocumentPolicy {
    }

    class Example {
        public function findPolicy(): ?DocumentPolicy {
            return new DocumentPolicy();
        }

        public function getVersionHistory(DocumentPolicy $policy): void {
        }

        public function run(): void {
            $policy = $this->findPolicy();
            if (!$policy) {
                $fallback = true;
            }

            $this->getVersionHistory($policy);
        }
    }`
	issues := analysePHP(t, php)
	if !hasArgTypeIssue(issues) {
		t.Fatalf("expected A.ARG.TYPE issue when null check does not terminate, got: %#v", issues)
	}
}
