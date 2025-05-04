package style

import (
	"go-phpcs/ast"
	"sort"
	"testing"
)

func TestListRegisteredRuleCodes(t *testing.T) {
	// Register some fake rules for testing
	RegisterRule("Z.TEST.RULE", func(string, []byte, []ast.Node) []StyleIssue { return nil })
	RegisterRule("A.TEST.RULE", func(string, []byte, []ast.Node) []StyleIssue { return nil })

	codes := ListRegisteredRuleCodes()
	if len(codes) < 2 {
		t.Fatalf("expected at least 2 codes, got %d", len(codes))
	}
	// Check that the codes are sorted
	if !sort.StringsAreSorted(codes) {
		t.Errorf("codes are not sorted: %v", codes)
	}
	// Check that our fake codes are present
	foundA, foundZ := false, false
	for _, c := range codes {
		if c == "A.TEST.RULE" {
			foundA = true
		}
		if c == "Z.TEST.RULE" {
			foundZ = true
		}
	}
	if !foundA || !foundZ {
		t.Errorf("expected both fake codes in list, got %v", codes)
	}
}
