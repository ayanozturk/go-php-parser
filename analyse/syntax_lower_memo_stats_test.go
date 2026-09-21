package analyse

import (
	"strings"
	"testing"

	"github.com/ayanozturk/go-php-parser/syntax"
)

func TestFormatSyntaxLowerMemoStatsAndCallLowerMissSnapshot(t *testing.T) {
	EnableSyntaxLowerMemoStats(true)
	defer EnableSyntaxLowerMemoStats(false)
	ResetSyntaxLowerMemoStats()

	recordMemoLower(MemoBucketExpr, MemoOutcomeHit)
	recordMemoLower(MemoBucketExpr, MemoOutcomeMiss)
	recordMemoLowerExprMiss(syntax.KindCallExpr)
	recordCallExprLowerMissReason("callee_call_iife")
	recordCallExprLowerMissReason("callee_call_iife")
	recordCallExprLowerMissReason("builtin:isset")

	snap := SnapshotSyntaxLowerMemoStats()
	formatted := FormatSyntaxLowerMemoStats(snap)
	if !strings.Contains(formatted, "memoLower hit/bridge/cst/miss by entrypoint:") {
		t.Fatalf("missing header: %s", formatted)
	}
	if !strings.Contains(formatted, "expr") || !strings.Contains(formatted, "call=1") {
		t.Fatalf("expected expr miss kind in format output:\n%s", formatted)
	}

	reasons := CallLowerMissReasonSnapshot()
	if reasons["callee_call_iife"] != 2 || reasons["builtin:isset"] != 1 {
		t.Fatalf("unexpected miss reasons %#v", reasons)
	}
	reasons["callee_call_iife"] = 99
	if CallLowerMissReasonSnapshot()["callee_call_iife"] != 2 {
		t.Fatal("snapshot must copy the map")
	}

	rendered := FormatCallLowerMissReasons(reasons)
	if !strings.Contains(rendered, "call Lower* miss reasons") || !strings.Contains(rendered, "callee_call_iife") {
		t.Fatalf("unexpected format:\n%s", rendered)
	}
	if FormatCallLowerMissReasons(nil) != "  (none)\n" {
		t.Fatalf("empty reasons should render (none)")
	}

	// Disabled path is a no-op.
	EnableSyntaxLowerMemoStats(false)
	before := SnapshotSyntaxLowerMemoStats()
	recordMemoLower(MemoBucketExpr, MemoOutcomeHit)
	recordCallExprLowerMissReason("should_not_count")
	after := SnapshotSyntaxLowerMemoStats()
	if before.Rows[MemoBucketExpr].Hit != after.Rows[MemoBucketExpr].Hit {
		t.Fatal("disabled stats should not record hits")
	}
	if CallLowerMissReasonSnapshot()["should_not_count"] != 0 {
		t.Fatal("disabled stats should not record call-lower miss reasons")
	}
}

func TestClassifyCallExprCSTFallbackBasics(t *testing.T) {
	if got := classifyCallExprCSTFallback(nil); got != "not_call" {
		t.Fatalf("nil => not_call, got %q", got)
	}
	if got := classifyBuiltinCallToken(nil); got != "builtin_no_callee" {
		t.Fatalf("nil builtin => builtin_no_callee, got %q", got)
	}
}
