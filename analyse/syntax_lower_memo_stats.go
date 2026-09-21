package analyse

import (
	"fmt"
	"sync/atomic"

	"github.com/ayanozturk/go-php-parser/syntax"
)

// SyntaxLowerMemoBucket identifies a memoLower* entrypoint for hit/miss stats.
type SyntaxLowerMemoBucket uint8

const (
	MemoBucketClassLike SyntaxLowerMemoBucket = iota
	MemoBucketFnDecl
	MemoBucketFnLike
	MemoBucketPropertyDecl
	MemoBucketInterfaceMethod
	MemoBucketExpr
	memoBucketCount
)

// SyntaxLowerMemoOutcome classifies a memoLower* lookup result.
type SyntaxLowerMemoOutcome uint8

const (
	MemoOutcomeHit SyntaxLowerMemoOutcome = iota
	MemoOutcomeBridge
	MemoOutcomeCST // lightweight CST build (no LowerExprNode)
	MemoOutcomeMiss
	memoOutcomeCount
)

// Expr miss kind buckets (only recorded on MemoBucketExpr + MemoOutcomeMiss).
const (
	MemoExprKindCall = iota
	MemoExprKindNew
	MemoExprKindMember // member / nullsafe / static member access
	MemoExprKindClosure
	MemoExprKindOther
	memoExprKindCount
)

var (
	syntaxLowerMemoStats    [memoBucketCount][memoOutcomeCount]atomic.Uint64
	syntaxLowerMemoExprMiss [memoExprKindCount]atomic.Uint64
	syntaxLowerMemoEnabled  atomic.Bool
)

// EnableSyntaxLowerMemoStats turns package-level memoLower hit/miss counters
// on or off. Cheap no-op when disabled; intended for one-worker WP / fused
// bench measurement of idea C (H1b).
func EnableSyntaxLowerMemoStats(on bool) {
	syntaxLowerMemoEnabled.Store(on)
}

// ResetSyntaxLowerMemoStats zeroes all memoLower counters.
func ResetSyntaxLowerMemoStats() {
	for b := SyntaxLowerMemoBucket(0); b < memoBucketCount; b++ {
		for o := SyntaxLowerMemoOutcome(0); o < memoOutcomeCount; o++ {
			syntaxLowerMemoStats[b][o].Store(0)
		}
	}
	for k := 0; k < memoExprKindCount; k++ {
		syntaxLowerMemoExprMiss[k].Store(0)
	}
	resetCallLowerMissReasons()
}

// SyntaxLowerMemoStatRow is one entrypoint's hit/bridge/miss totals.
type SyntaxLowerMemoStatRow struct {
	Bucket                      string
	Hit, Bridge, CST, Miss, Sum uint64
}

// SyntaxLowerMemoSnapshot is a point-in-time dump of memoLower counters.
type SyntaxLowerMemoSnapshot struct {
	Rows     []SyntaxLowerMemoStatRow
	ExprMiss map[string]uint64 // call/new/member/closure/other — LowerExprNode only
}

// SnapshotSyntaxLowerMemoStats returns current memoLower counters.
func SnapshotSyntaxLowerMemoStats() SyntaxLowerMemoSnapshot {
	names := [...]string{
		MemoBucketClassLike:       "classLike",
		MemoBucketFnDecl:          "fnDecl",
		MemoBucketFnLike:          "fnLike",
		MemoBucketPropertyDecl:    "propertyDecl",
		MemoBucketInterfaceMethod: "interfaceMethod",
		MemoBucketExpr:            "expr",
	}
	rows := make([]SyntaxLowerMemoStatRow, 0, memoBucketCount)
	for b := SyntaxLowerMemoBucket(0); b < memoBucketCount; b++ {
		hit := syntaxLowerMemoStats[b][MemoOutcomeHit].Load()
		bridge := syntaxLowerMemoStats[b][MemoOutcomeBridge].Load()
		cst := syntaxLowerMemoStats[b][MemoOutcomeCST].Load()
		miss := syntaxLowerMemoStats[b][MemoOutcomeMiss].Load()
		rows = append(rows, SyntaxLowerMemoStatRow{
			Bucket: names[b],
			Hit:    hit,
			Bridge: bridge,
			CST:    cst,
			Miss:   miss,
			Sum:    hit + bridge + cst + miss,
		})
	}
	exprMiss := map[string]uint64{
		"call":    syntaxLowerMemoExprMiss[MemoExprKindCall].Load(),
		"new":     syntaxLowerMemoExprMiss[MemoExprKindNew].Load(),
		"member":  syntaxLowerMemoExprMiss[MemoExprKindMember].Load(),
		"closure": syntaxLowerMemoExprMiss[MemoExprKindClosure].Load(),
		"other":   syntaxLowerMemoExprMiss[MemoExprKindOther].Load(),
	}
	return SyntaxLowerMemoSnapshot{Rows: rows, ExprMiss: exprMiss}
}

// FormatSyntaxLowerMemoStats renders a compact counter table for logs.
func FormatSyntaxLowerMemoStats(s SyntaxLowerMemoSnapshot) string {
	out := "memoLower hit/bridge/cst/miss by entrypoint:\n"
	for _, r := range s.Rows {
		out += fmt.Sprintf("  %-16s hit=%8d bridge=%8d cst=%8d miss=%8d sum=%8d\n",
			r.Bucket, r.Hit, r.Bridge, r.CST, r.Miss, r.Sum)
	}
	out += "  expr Lower* miss by kind:"
	for _, k := range []string{"call", "new", "member", "closure", "other"} {
		out += fmt.Sprintf(" %s=%d", k, s.ExprMiss[k])
	}
	out += "\n"
	return out
}

func recordMemoLower(bucket SyntaxLowerMemoBucket, outcome SyntaxLowerMemoOutcome) {
	if !syntaxLowerMemoEnabled.Load() {
		return
	}
	syntaxLowerMemoStats[bucket][outcome].Add(1)
}

func recordMemoLowerExprMiss(kind syntax.Kind) {
	if !syntaxLowerMemoEnabled.Load() {
		return
	}
	switch kind {
	case syntax.KindCallExpr:
		syntaxLowerMemoExprMiss[MemoExprKindCall].Add(1)
	case syntax.KindNewExpr:
		syntaxLowerMemoExprMiss[MemoExprKindNew].Add(1)
	case syntax.KindMemberAccessExpr, syntax.KindNullsafeMemberAccessExpr, syntax.KindStaticMemberAccessExpr:
		syntaxLowerMemoExprMiss[MemoExprKindMember].Add(1)
	case syntax.KindClosureExpr:
		syntaxLowerMemoExprMiss[MemoExprKindClosure].Add(1)
	default:
		syntaxLowerMemoExprMiss[MemoExprKindOther].Add(1)
	}
}
