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

func TestMethodArgumentTypeRefinedInsideInstanceofAndBranch(t *testing.T) {
	php := `<?php
    namespace App;

    class UploadedFile {
    }

    class Example {
        public function getDocumentFile(): ?UploadedFile {
            return new UploadedFile();
        }

        public function uploadDocument(UploadedFile $documentFile): void {
        }

        public function run(string $documentSelection): void {
            $documentFile = $this->getDocumentFile();
            if ($documentSelection === "upload" && $documentFile instanceof UploadedFile) {
                $this->uploadDocument($documentFile);
            }
        }
    }`
	issues := analysePHP(t, php)
	if hasArgTypeIssue(issues) {
		t.Fatalf("expected no A.ARG.TYPE issue for variable refined by instanceof in true branch, got: %#v", issues)
	}
}

func TestMethodArgumentTypeNotRefinedWithoutInstanceofBranch(t *testing.T) {
	php := `<?php
    namespace App;

    class UploadedFile {
    }

    class Example {
        public function getDocumentFile(): ?UploadedFile {
            return new UploadedFile();
        }

        public function uploadDocument(UploadedFile $documentFile): void {
        }

        public function run(string $documentSelection): void {
            $documentFile = $this->getDocumentFile();
            if ($documentSelection === "upload") {
                $this->uploadDocument($documentFile);
            }
        }
    }`
	issues := analysePHP(t, php)
	if !hasArgTypeIssue(issues) {
		t.Fatalf("expected A.ARG.TYPE issue without instanceof refinement, got: %#v", issues)
	}
}

func TestMethodArgumentTypeRefinedInsideIsStringAndBranch(t *testing.T) {
	php := `<?php
    class Company {
    }

    class RequestQuery {
        public function get(string $name): bool|float|int|null|string {
            return "abc";
        }
    }

    class Request {
        public RequestQuery $query;
    }

    class Example {
        private Request $request;

        public function getCompany(): Company {
            return new Company();
        }

        public function getDocumentForCompany(string $documentId, Company $company): void {
        }

        public function run(): void {
            $company = $this->getCompany();
            $documentId = $this->request->query->get("documentId");
            if (is_string($documentId) && $documentId !== "") {
                $this->getDocumentForCompany($documentId, $company);
            }
        }
    }`
	issues := analysePHP(t, php)
	if hasArgTypeIssue(issues) {
		t.Fatalf("expected no A.ARG.TYPE issue for variable refined by is_string in true branch, got: %#v", issues)
	}
}

func TestMethodArgumentTypeMatchesNamedArgumentsByParameterName(t *testing.T) {
	php := `<?php
class Example {
    public function save(string $title, int $count): void {
    }

    public function run(): void {
        $this->save(count: 2, title: "ok");
    }
}`
	issues := analysePHP(t, php)
	if hasArgTypeIssue(issues) {
		t.Fatalf("expected no A.ARG.TYPE issue for correctly typed named arguments, got: %#v", issues)
	}
}

func TestConstructorArgumentTypeMismatchDetectedForNamedArguments(t *testing.T) {
	php := `<?php
class Response {
    public function __construct(
        int $status = 200,
        array $headers = [],
        ?string $body = null
    ) {
    }

    public function make(): void {
        new self(body: []);
    }
}
`
	issues := analysePHP(t, php)
	if !hasArgTypeIssue(issues) {
		t.Fatalf("expected A.ARG.TYPE issue for constructor named arg type mismatch, got: %#v", issues)
	}
}
