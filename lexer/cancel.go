package lexer

import "context"

// cancelCheckBudget is how many input bytes may be scanned between cooperative
// cancellation checks inside a single NextToken call. LexAllContext still
// checks between tokens; this bound covers megabyte heredoc/string/comment
// bodies that otherwise never return to that loop.
var cancelCheckBudget = 64 << 10 // 64 KiB

// SetCancelContext enables cooperative cancellation for streaming NextToken /
// SkipBalancedCurlyBlock scans. A nil context disables checks.
func (l *Lexer) SetCancelContext(ctx context.Context) {
	l.setCancelContext(ctx)
}

func (l *Lexer) setCancelContext(ctx context.Context) {
	l.ctx = ctx
	l.cancelErr = nil
	if ctx == nil {
		l.cancelCheckAt = 0
		return
	}
	l.cancelCheckAt = l.pos + cancelCheckBudget
}

// Cancelled reports a cooperative cancellation observed during scanning.
func (l *Lexer) Cancelled() error {
	return l.cancelled()
}

// cancelled reports a cooperative cancellation observed during scanning.
func (l *Lexer) cancelled() error {
	return l.cancelErr
}

// checkCancel consults ctx.Err when the lexer has advanced cancelCheckBudget
// bytes since the last check. It is cheap on the hot path: a single integer
// compare until the budget is exhausted. Returns true when cancelled.
func (l *Lexer) checkCancel() bool {
	return l.noteCancelProgress(l.pos)
}

// noteCancelProgress is like checkCancel but for scanners that walk a local
// offset without advancing l.pos (e.g. mid-string classification probes).
func (l *Lexer) noteCancelProgress(offset int) bool {
	if l.ctx == nil || l.cancelErr != nil {
		return l.cancelErr != nil
	}
	if offset < l.cancelCheckAt {
		return false
	}
	// Advance the checkpoint even when still under the file length so a
	// cancelled context is observed within one budget window of progress.
	l.cancelCheckAt = offset + cancelCheckBudget
	if err := l.ctx.Err(); err != nil {
		l.cancelErr = err
		return true
	}
	return false
}
