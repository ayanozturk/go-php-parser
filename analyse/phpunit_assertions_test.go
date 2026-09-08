package analyse

import "testing"

func TestAssertSameDoesNotOverwriteActualWithNullExpected(t *testing.T) {
	files := map[string]string{
		"test.php": `<?php
class Team {
	public function getCreatedDate(): \DateTimeInterface { return new \DateTimeImmutable(); }
}
class ExampleTest extends \PHPUnit\Framework\TestCase {
	public function testCreate(): void {
		$capturedTeam = null;
		$team = new Team();
		$this->assertSame($capturedTeam, $team);
		$team->getCreatedDate();
	}
}
`,
	}
	issues := runAnalysisLevelOnFiles(t, files, 2)
	if hasIssueContaining(issues, level2MethodNonObjectCode, "getCreatedDate") {
		t.Fatalf("assertSame with null-inferred expected must not wipe Team actual, got %#v", issues)
	}
}

func TestAssertSameConcreteExpectedStillNarrows(t *testing.T) {
	files := map[string]string{
		"test.php": `<?php
class Provider { public function sync(): void {} }
class ExampleTest extends \PHPUnit\Framework\TestCase {
	/** @var Provider|null */
	public $provider;
	public function testSame(): void {
		$this->assertSame(new Provider(), $this->provider);
		$this->provider->sync();
	}
}
`,
	}
	issues := runAnalysisLevelOnFiles(t, files, 8)
	if hasIssueContaining(issues, level8MethodNonObjectCode, "sync") {
		t.Fatalf("assertSame with Provider expected should keep provider non-null, got %#v", issues)
	}
}
