package analyse

import (
	"fmt"
	"sort"
	"sync"

	"github.com/ayanozturk/go-php-parser/syntax"
	"github.com/ayanozturk/go-php-parser/token"
)

var (
	callLowerMissReasonsMu sync.Mutex
	callLowerMissReasons   map[string]uint64
)

func resetCallLowerMissReasons() {
	callLowerMissReasonsMu.Lock()
	callLowerMissReasons = make(map[string]uint64)
	callLowerMissReasonsMu.Unlock()
}

func recordCallExprLowerMissReason(reason string) {
	if !syntaxLowerMemoEnabled.Load() || reason == "" {
		return
	}
	callLowerMissReasonsMu.Lock()
	if callLowerMissReasons == nil {
		callLowerMissReasons = make(map[string]uint64)
	}
	callLowerMissReasons[reason]++
	callLowerMissReasonsMu.Unlock()
}

// CallLowerMissReasonSnapshot returns a copy of KindCallExpr Lower* miss
// reason counts (only populated when EnableSyntaxLowerMemoStats is on).
func CallLowerMissReasonSnapshot() map[string]uint64 {
	callLowerMissReasonsMu.Lock()
	defer callLowerMissReasonsMu.Unlock()
	out := make(map[string]uint64, len(callLowerMissReasons))
	for k, v := range callLowerMissReasons {
		out[k] = v
	}
	return out
}

// FormatCallLowerMissReasons renders sorted reason counts for logs.
func FormatCallLowerMissReasons(m map[string]uint64) string {
	if len(m) == 0 {
		return "  (none)\n"
	}
	type kv struct {
		k string
		v uint64
	}
	rows := make([]kv, 0, len(m))
	var sum uint64
	for k, v := range m {
		rows = append(rows, kv{k, v})
		sum += v
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].v != rows[j].v {
			return rows[i].v > rows[j].v
		}
		return rows[i].k < rows[j].k
	})
	out := fmt.Sprintf("call Lower* miss reasons (n=%d):\n", sum)
	for _, r := range rows {
		pct := 100 * float64(r.v) / float64(sum)
		out += fmt.Sprintf("  %6d  %5.1f%%  %s\n", r.v, pct, r.k)
	}
	return out
}

// classifyCallExprCSTFallback explains why tryCSTCallExprForMemo returned
// ok=false for a KindCallExpr (forcing LowerExprNode). Used only for
// attribution when memo stats are enabled.
func classifyCallExprCSTFallback(n *syntax.RedNode) string {
	if n == nil || n.Kind() != syntax.KindCallExpr {
		return "not_call"
	}
	callee := syntax.CallCallee(n)
	if callee == nil {
		return classifyBuiltinCallToken(n)
	}
	switch callee.Kind() {
	case syntax.KindMemberAccessExpr, syntax.KindNullsafeMemberAccessExpr:
		if syntax.MemberAccessName(callee) == "" {
			// Normally handled as CST nil (suppress); should not Lower.
			return "dynamic_instance_unexpected_lower"
		}
		obj := syntax.MemberAccessObject(callee)
		if obj == nil {
			return "member_nil_object"
		}
		return "member_complex_object:" + obj.Kind().String()

	case syntax.KindStaticMemberAccessExpr:
		class, member, dynamic := syntax.StaticMemberAccessParts(callee)
		if dynamic {
			return "static_dynamic"
		}
		if class == "" {
			return "static_empty_class"
		}
		if member == "" {
			return "static_empty_member"
		}
		return "static_unexpected"

	case syntax.KindVariableExpr:
		return "variable_empty_name"

	case syntax.KindParenExpr:
		inner := syntax.ParenInner(callee)
		if inner == nil {
			return "paren_empty"
		}
		return "paren_complex_inner:" + inner.Kind().String()

	case syntax.KindCallExpr:
		return "callee_call_iife"

	case syntax.KindArrayAccessExpr:
		return "callee_array_dim"

	case syntax.KindFirstClassCallableExpr:
		return "callee_fcc"

	case syntax.KindNewExpr:
		return "callee_new_unexpected" // (new T)() is unusual; usually (new T)->m()

	default:
		return "callee_other:" + callee.Kind().String()
	}
}

// classifyBuiltinCallToken attributes KindCallExpr with no callee expr/name
// child (isset/empty/unset/exit/die keyword-token calls).
func classifyBuiltinCallToken(n *syntax.RedNode) string {
	if n == nil {
		return "builtin_no_callee"
	}
	var reason string
	n.ForEachChildDesc(func(green *syntax.GreenNode, _ int) bool {
		if green == nil || green.Kind() != syntax.KindToken {
			return true
		}
		switch green.TokenType() {
		case token.T_ISSET:
			reason = "builtin:isset"
			return false
		case token.T_EMPTY:
			reason = "builtin:empty"
			return false
		case token.T_UNSET:
			reason = "builtin:unset"
			return false
		case token.T_EXIT:
			reason = "builtin:exit"
			return false
		case token.T_DIE:
			reason = "builtin:die"
			return false
		default:
			return true
		}
	})
	if reason != "" {
		return reason
	}
	return "builtin_no_callee"
}
