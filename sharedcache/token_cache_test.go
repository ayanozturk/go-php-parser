package sharedcache

import (
	"go-phpcs/token"
	"testing"
)

func TestBatchTokenizeFilesAndGetCachedTokens(t *testing.T) {
	ClearTokenCache()
	files := map[string][]byte{
		"a.php": []byte("<?php echo 1;"),
		"b.php": []byte("<?php $x = 2;"),
	}
	BatchTokenizeFiles(files)

	toksA := GetCachedTokens("a.php")
	toksB := GetCachedTokens("b.php")
	if len(toksA) == 0 || toksA[len(toksA)-1].Type != token.T_EOF {
		t.Errorf("Tokens for a.php missing or not ending with EOF: %+v", toksA)
	}
	if len(toksB) == 0 || toksB[len(toksB)-1].Type != token.T_EOF {
		t.Errorf("Tokens for b.php missing or not ending with EOF: %+v", toksB)
	}
	// Retokenize and ensure cache is cleared
	ClearTokenCache()
	if GetCachedTokens("a.php") != nil {
		t.Errorf("Token cache not cleared for a.php")
	}
}

func TestSplitLinesCachedAndDelete(t *testing.T) {
	ClearLinesCache()

	content := []byte("<?php\n$a = 1;\n")
	lines := SplitLinesCached(content)
	if len(lines) != 3 || lines[1] != "$a = 1;" {
		t.Fatalf("unexpected split lines: %#v", lines)
	}
	if cached := SplitLinesCached(content); &cached[0] != &lines[0] {
		t.Fatalf("expected cached split lines to be reused")
	}

	DeleteCachedLines(content)
	afterDelete := SplitLinesCached(content)
	if len(afterDelete) != len(lines) || afterDelete[1] != lines[1] {
		t.Fatalf("unexpected split lines after delete: %#v", afterDelete)
	}
}
