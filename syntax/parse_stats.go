package syntax

import "sync/atomic"

var parseInvocations atomic.Uint64

// ParseInvocationCount returns how many times ParseWithContext completed a parse
// since process start or the last ResetParseInvocationCount call.
func ParseInvocationCount() uint64 { return parseInvocations.Load() }

// ResetParseInvocationCount zeroes the parse invocation counter (tests / profiling).
func ResetParseInvocationCount() { parseInvocations.Store(0) }

func noteParseInvocation() { parseInvocations.Add(1) }
