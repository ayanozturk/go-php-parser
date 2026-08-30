package analyse

import "testing"

func TestLevel3ThrowExpressionValidity(t *testing.T) {
	files := map[string]string{
		"test.php": `<?php
throw new Exception();
throw new DateTime();
throw new MissingThrowable();
`,
	}

	level2Issues := runPHPStanLevelOnFiles(t, files, 2)
	if hasIssueContaining(level2Issues, level3ThrowTypeCode, "to throw") {
		t.Fatalf("level two should exclude level-three throw diagnostics, got %#v", level2Issues)
	}

	level3Issues := runPHPStanLevelOnFiles(t, files, 3)
	if hasIssueContaining(level3Issues, level3ThrowTypeCode, "Invalid type Exception to throw") {
		t.Fatalf("Exception should be throwable, got %#v", level3Issues)
	}
	if !hasIssueContaining(level3Issues, level3ThrowTypeCode, "Invalid type DateTime to throw") {
		t.Fatalf("expected non-throwable throw issue at level three, got %#v", level3Issues)
	}
	if hasIssueContaining(level3Issues, level3ThrowTypeCode, "MissingThrowable") {
		t.Fatalf("unresolved throw types should remain with symbol checks, got %#v", level3Issues)
	}
}
