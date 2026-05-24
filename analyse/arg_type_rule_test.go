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

func TestMethodArgumentObjectParameterAcceptsClassInstance(t *testing.T) {
	php := `<?php
    namespace Symfony\Component\PropertyAccess\Tests;

    class UninitializedPrivateProperty {
    }

    class PropertyAccessor {
        public function getValue(array|object $objectOrArray, string $propertyPath): mixed {
        }
    }

    class Example {
        private PropertyAccessor $propertyAccessor;

        public function run(): void {
            $this->propertyAccessor->getValue(new UninitializedPrivateProperty(), "uninitialized");
        }
    }`
	issues := analysePHP(t, php)
	if hasArgTypeIssue(issues) {
		t.Fatalf("expected no A.ARG.TYPE issue for class instance passed to object parameter, got: %#v", issues)
	}
}

func TestMethodArgumentTypeNormalizesPhpDocAliasesAndGenerics(t *testing.T) {
	php := `<?php
    class CollectionConsumer {
        /**
         * @param list<string> $names
         * @param class-string<Foo> $className
         * @param boolean $enabled
         * @param integer $count
         */
        public function consume($names, $className, $enabled, $count): void {
        }

        public function run(): void {
            $this->consume(["a"], Foo::class, true, 1);
        }
    }

    class Foo {
    }`
	issues := analysePHP(t, php)
	if hasArgTypeIssue(issues) {
		t.Fatalf("expected no A.ARG.TYPE issue for normalized PHPDoc aliases/generics, got: %#v", issues)
	}
}

func TestInheritedMethodSignatureUsedForArgumentTypes(t *testing.T) {
	php := `<?php
    class BaseAccessor {
        public function getValue(array|object $objectOrArray, string $propertyPath): mixed {
        }
    }

    class PropertyAccessor extends BaseAccessor {
    }

    class Fixture {
    }

    class Example {
        private PropertyAccessor $propertyAccessor;

        public function run(): void {
            $this->propertyAccessor->getValue(new Fixture(), "value");
        }
    }`
	issues := analysePHP(t, php)
	if hasArgTypeIssue(issues) {
		t.Fatalf("expected no A.ARG.TYPE issue using inherited method signature, got: %#v", issues)
	}
}

func TestInheritedMethodSignatureDetectsArgumentTypeMismatch(t *testing.T) {
	php := `<?php
    class BaseAccessor {
        public function setCount(int $count): void {
        }
    }

    class Accessor extends BaseAccessor {
    }

    class Example {
        private Accessor $accessor;

        public function run(): void {
            $this->accessor->setCount("bad");
        }
    }`
	issues := analysePHP(t, php)
	if !hasArgTypeIssue(issues) {
		t.Fatalf("expected A.ARG.TYPE issue using inherited method signature, got: %#v", issues)
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

func TestMethodArgumentTypeRefinedInsideNotNullBranch(t *testing.T) {
	php := `<?php
    class DOMElement {
    }

    class Example {
        public function parseBooleanAttribute(DOMElement $element, string $name, bool $default): bool {
        }

        public function run(?DOMElement $element): void {
            if ($element !== null) {
                $this->parseBooleanAttribute($element, "includeGitInformation", false);
            }
        }
    }`
	issues := analysePHP(t, php)
	if hasArgTypeIssue(issues) {
		t.Fatalf("expected no A.ARG.TYPE issue for non-null variable inside !== null branch, got: %#v", issues)
	}
}

func TestMethodArgumentTypeRefinedInsideNotIsNullBranch(t *testing.T) {
	php := `<?php
    class DOMElement {
    }

    class Example {
        public function parseBooleanAttribute(DOMElement $element): bool {
        }

        public function run(?DOMElement $element): void {
            if (!is_null($element)) {
                $this->parseBooleanAttribute($element);
            }
        }
    }`
	issues := analysePHP(t, php)
	if hasArgTypeIssue(issues) {
		t.Fatalf("expected no A.ARG.TYPE issue for non-null variable inside !is_null branch, got: %#v", issues)
	}
}

func TestMethodArgumentTypeRefinedInsideTruthyObjectBranch(t *testing.T) {
	php := `<?php
    class DOMElement {
    }

    class Example {
        public function parseBooleanAttribute(DOMElement $element): bool {
        }

        public function run(?DOMElement $element): void {
            if ($element) {
                $this->parseBooleanAttribute($element);
            }
        }
    }`
	issues := analysePHP(t, php)
	if hasArgTypeIssue(issues) {
		t.Fatalf("expected no A.ARG.TYPE issue for non-null variable inside truthy branch, got: %#v", issues)
	}
}

func TestMethodArgumentTypeRefinedAfterLoopBreakNullGuard(t *testing.T) {
	php := `<?php
    class Rule {
    }

    class Example {
        public function propagate(): ?Rule {
            return new Rule();
        }

        public function analyze(Rule $rule): void {
        }

        public function run(): void {
            while (true) {
                $rule = $this->propagate();
                if ($rule === null) {
                    break;
                }

                $this->analyze($rule);
            }
        }
    }`
	issues := analysePHP(t, php)
	if hasArgTypeIssue(issues) {
		t.Fatalf("expected no A.ARG.TYPE issue after break null guard in loop body, got: %#v", issues)
	}
}

func TestMethodArgumentTypeRefinedAfterAssertNotNull(t *testing.T) {
	php := `<?php
    class Example {
        public function revert(int $level): void {
        }

        public function run(?int $lastLevel): void {
            assert($lastLevel !== null);

            $this->revert($lastLevel);
        }
    }`
	issues := analysePHP(t, php)
	if hasArgTypeIssue(issues) {
		t.Fatalf("expected no A.ARG.TYPE issue after assert non-null guard, got: %#v", issues)
	}
}

func TestMethodArgumentTypeRefinedAfterAssertNotNullFromImpreciseAssignment(t *testing.T) {
	php := `<?php
    class Example {
        public function revert(int $level): void {
        }

        public function run(): void {
            $lastLevel = null;
            assert($lastLevel !== null);

            $this->revert($lastLevel);
        }
    }`
	issues := analysePHP(t, php)
	if hasArgTypeIssue(issues) {
		t.Fatalf("expected no A.ARG.TYPE issue after assert proves imprecise null assignment non-null, got: %#v", issues)
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

func TestMethodArgumentTypeAllowsArrayPassedToByReferenceArrayParameter(t *testing.T) {
	php := `<?php
namespace App\Service\Profile;

class Example {
    public function buildSummary(): void {
        $missingFields = [];
        $totalFields = 0;
        $completedFields = 0;

        $this->trackField(true, "Profile photo", $missingFields, $totalFields, $completedFields);
    }

    /**
     * @param list<string> $missingFields
     */
    private function trackField(
        bool $isComplete,
        string $label,
        array &$missingFields,
        int &$totalFields,
        int &$completedFields,
    ): void {
    }
}`
	issues := analysePHP(t, php)
	if hasArgTypeIssue(issues) {
		t.Fatalf("expected no A.ARG.TYPE issue for array passed to by-reference array parameter, got: %#v", issues)
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

func TestNoFalsePositiveAfterNegatedInstanceofGuard(t *testing.T) {
	// `if (!($x instanceof Foo)) { return; }` narrows $x to Foo afterwards.
	// Passing $x to a method expecting Foo should produce no A.ARG.TYPE issue.
	php := `<?php
class Token {}
class Example {
    public function takesToken(Token $t): void {}

    public function run(?Token $token): void {
        if (!($token instanceof Token)) {
            return;
        }
        $this->takesToken($token);
    }
}
`
	issues := analysePHP(t, php)
	if hasArgTypeIssue(issues) {
		t.Fatalf("expected no A.ARG.TYPE issue after negated instanceof guard, got: %#v", issues)
	}
}
