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
