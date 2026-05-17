package analyse

import "testing"

func hasPropertyTypeIssue(issues []AnalysisIssue) bool {
	for _, iss := range issues {
		if iss.Code == "A.PROP.TYPE" {
			return true
		}
	}
	return false
}

func TestPropertyAssignmentTypeMismatch(t *testing.T) {
	php := `<?php
    class Example {
        private int $count;

        public function run(): void {
            $this->count = "bad";
        }
    }`
	issues := analysePHP(t, php)
	if !hasPropertyTypeIssue(issues) {
		t.Fatalf("expected A.PROP.TYPE issue for string assigned to int property, got: %#v", issues)
	}
}

func TestPropertyAssignmentTypeCompatible(t *testing.T) {
	php := `<?php
    class Example {
        private float $count;

        public function run(): void {
            $this->count = 1;
        }
    }`
	issues := analysePHP(t, php)
	if hasPropertyTypeIssue(issues) {
		t.Fatalf("expected no A.PROP.TYPE issue for int assigned to float property, got: %#v", issues)
	}
}

func TestPropertyAssignmentAcceptsImplementedInterface(t *testing.T) {
	php := `<?php
	namespace Doctrine\Common\Collections;

	interface Collection {}

	class ArrayCollection implements Collection {}

	class Example {
		private Collection $users;

		public function __construct()
		{
			$this->users = new ArrayCollection();
		}
	}`
	issues := analysePHP(t, php)
	if hasPropertyTypeIssue(issues) {
		t.Fatalf("expected no A.PROP.TYPE issue for class implementing interface assignment, got: %#v", issues)
	}
}
