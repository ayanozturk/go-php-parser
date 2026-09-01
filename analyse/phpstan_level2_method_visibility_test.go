package analyse

import "testing"

func TestLevel2ProtectedMethodVisibility(t *testing.T) {
	files := map[string]string{
		"test.php": `<?php
class Base {
    protected function work(): void {}
}
class Child extends Base {
    public function allowed(): void {
        $this->work();
    }
}
(new Base())->work();
`,
	}

	level1Issues := runAnalysisLevelOnFiles(t, files, 1)
	if hasIssueContaining(level1Issues, level2MethodVisibilityCode, "Call to protected method") {
		t.Fatalf("level one should exclude level-two visibility diagnostics, got %#v", level1Issues)
	}

	level2Issues := runAnalysisLevelOnFiles(t, files, 2)
	if countIssueContaining(level2Issues, level2MethodVisibilityCode, "Call to protected method Base::work()") != 1 {
		t.Fatalf("expected only the external protected call at level two, got %#v", level2Issues)
	}
}
