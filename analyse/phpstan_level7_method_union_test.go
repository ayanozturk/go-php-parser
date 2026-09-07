package analyse

import "testing"

func TestLevel7ReportsPartialUnionAndDNFMethods(t *testing.T) {
	files := map[string]string{
		"test.php": `<?php
class FirstChoice {}
class SecondChoice {
    public function optional(): void {}
}
interface AvailableContract {
    public function execute(): void;
}
interface MarkerContract {}
class AlternativeChoice {}
interface HasMethod { public function available(): void; }
interface FirstTag {}
interface SecondTag {}
class MissingAlternative {}
class MagicService {
    public function __call(string $name, array $arguments): mixed { return null; }
}
class PlainService {}

function run(
    FirstChoice|SecondChoice $choice,
    (AvailableContract&MarkerContract)|AlternativeChoice $dnf,
    (HasMethod&FirstTag)|MissingAlternative $partiallyAvailable,
    (HasMethod&FirstTag)|(HasMethod&SecondTag) $availableEverywhere,
    HasMethod&FirstTag $intersection,
    MagicService|PlainService $magicUnion,
    mixed $mixed
): void {
    $choice->optional();
    $dnf->execute();
    $partiallyAvailable->available();
    $availableEverywhere->available();
    $intersection->available();
    $intersection->missing();
    $magicUnion->dynamic();
    $mixed->unknown();
}
`,
	}

	level2Issues := runAnalysisLevelOnFiles(t, files, 2)
	for _, unexpected := range []string{"optional()", "execute()", "available()", "dynamic()", "unknown()"} {
		if hasIssueContaining(level2Issues, level7MethodUnionCode, unexpected) {
			t.Fatalf("level two should exclude level-seven union diagnostics containing %q, got %#v", unexpected, level2Issues)
		}
	}
	if hasIssueContaining(level2Issues, level2MethodExistenceCode, "optional()") ||
		hasIssueContaining(level2Issues, level2MethodExistenceCode, "execute()") ||
		hasIssueContaining(level2Issues, level2MethodExistenceCode, "available()") {
		t.Fatalf("level two should stay silent for partial unions, got %#v", level2Issues)
	}
	if countIssueContaining(level2Issues, level2MethodExistenceCode, "FirstTag&HasMethod::missing()") != 1 {
		t.Fatalf("all-missing intersections remain a level-two diagnostic, got %#v", level2Issues)
	}

	level7Issues := runAnalysisLevelOnFiles(t, files, 7)
	for _, expected := range []string{
		"FirstChoice|SecondChoice::optional()",
		"(AvailableContract&MarkerContract)|AlternativeChoice::execute()",
		"(FirstTag&HasMethod)|MissingAlternative::available()",
		"MagicService|PlainService::dynamic()",
	} {
		if countIssueContaining(level7Issues, level7MethodUnionCode, expected) != 1 {
			t.Fatalf("expected one %s diagnostic, got %#v", expected, level7Issues)
		}
	}
	if hasIssueContaining(level7Issues, level7MethodUnionCode, "(FirstTag&HasMethod)|(HasMethod&SecondTag)::available()") ||
		hasIssueContaining(level7Issues, level7MethodUnionCode, "FirstTag&HasMethod::available()") ||
		hasIssueContaining(level7Issues, level7MethodUnionCode, "FirstTag&HasMethod::missing()") ||
		hasIssueContaining(level7Issues, level7MethodUnionCode, "unknown()") {
		t.Fatalf("level seven should not duplicate all-missing, known, or mixed receivers, got %#v", level7Issues)
	}
}

func TestLevel7AllowsPhpunitMockIntersectionAndCreateMock(t *testing.T) {
	files := map[string]string{
		"mock.php": `<?php

namespace PHPUnit\Framework\MockObject;

interface MockObject
{
    public function method(string $name): MockObject;
}

namespace PHPUnit\Framework;

use PHPUnit\Framework\MockObject\MockObject;

abstract class TestCase
{
    /**
     * @template RealInstanceType of object
     * @param class-string<RealInstanceType> $type
     * @return MockObject&RealInstanceType
     */
    final protected function createMock(string $type): MockObject
    {
        throw new \RuntimeException('stub');
    }
}

namespace {

use PHPUnit\Framework\MockObject\MockObject;
use PHPUnit\Framework\TestCase;

class User
{
    public function id(): string
    {
        return '';
    }
}

final class ExampleTest extends TestCase
{
    private function mockUser(): MockObject&User
    {
        throw new \RuntimeException('stub');
    }

    public function testMocks(): void
    {
        $user = $this->mockUser();
        $user->method('id');
        $user->id();

        $this->createMock(User::class)->method('id');
        $this->createMock(User::class)->id();
    }
}

}
`,
		"partial.php": `<?php
class FirstChoice {}

class SecondChoice
{
    public function optional(): void {}
}

function runPartialUnion(FirstChoice|SecondChoice $choice): void
{
    $choice->optional();
}
`,
	}

	level7Issues := runAnalysisLevelOnFiles(t, files, 7)
	for _, unexpected := range []string{
		"MockObject&User::method()",
		"MockObject&User::id()",
		"User&MockObject::method()",
		"User&MockObject::id()",
		"method('id')",
		"::id()",
	} {
		if hasIssueContaining(level7Issues, level7MethodUnionCode, unexpected) {
			t.Fatalf("level seven should not report mock intersection or createMock calls as %s containing %q, got %#v", level7MethodUnionCode, unexpected, level7Issues)
		}
	}
	if countIssueContaining(level7Issues, level7MethodUnionCode, "FirstChoice|SecondChoice::optional()") != 1 {
		t.Fatalf("expected one partial union %s diagnostic, got %#v", level7MethodUnionCode, level7Issues)
	}
	level7MethodUnionCount := 0
	for _, issue := range level7Issues {
		if issue.Code == level7MethodUnionCode {
			level7MethodUnionCount++
		}
	}
	if level7MethodUnionCount != 1 {
		t.Fatalf("expected exactly one %s diagnostic, got %#v", level7MethodUnionCode, level7Issues)
	}
}
