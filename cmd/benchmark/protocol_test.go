package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestInterleavedRunOrderIsDeterministicAndBalancesOrder(t *testing.T) {
	tests := []struct {
		count int
		want  []benchmarkRunTarget
	}{
		{count: -1, want: nil},
		{count: 0, want: nil},
		{count: 1, want: []benchmarkRunTarget{benchmarkCandidate, benchmarkBaseline}},
		{count: 2, want: []benchmarkRunTarget{benchmarkCandidate, benchmarkBaseline, benchmarkBaseline, benchmarkCandidate}},
		{count: 3, want: []benchmarkRunTarget{benchmarkCandidate, benchmarkBaseline, benchmarkBaseline, benchmarkCandidate, benchmarkCandidate, benchmarkBaseline}},
	}

	for _, test := range tests {
		got := interleavedRunOrder(test.count)
		if !reflect.DeepEqual(got, test.want) {
			t.Fatalf("interleavedRunOrder(%d) = %v, want %v", test.count, got, test.want)
		}
		if again := interleavedRunOrder(test.count); !reflect.DeepEqual(again, got) {
			t.Fatalf("interleavedRunOrder(%d) is not deterministic: first %v, second %v", test.count, got, again)
		}
	}
}

func TestValidatePhaseAccountingRejectsMismatchedRuns(t *testing.T) {
	reference := runMetrics{
		FilesDiscovered:    12,
		FilesParsed:        10,
		FilesFailed:        2,
		TotalLOC:           345,
		TotalBytes:         6789,
		DiagnosticsEmitted: 23,
	}
	matching := reference
	if err := validatePhaseAccounting(reference, []runMetrics{matching}, true); err != nil {
		t.Fatalf("matching phase accounting rejected: %v", err)
	}

	fields := []struct {
		name   string
		mutate func(*runMetrics)
	}{
		{name: "discovered files", mutate: func(run *runMetrics) { run.FilesDiscovered++ }},
		{name: "parsed files", mutate: func(run *runMetrics) { run.FilesParsed++ }},
		{name: "failed files", mutate: func(run *runMetrics) { run.FilesFailed++ }},
		{name: "lines of code", mutate: func(run *runMetrics) { run.TotalLOC++ }},
		{name: "bytes", mutate: func(run *runMetrics) { run.TotalBytes++ }},
		{name: "diagnostics", mutate: func(run *runMetrics) { run.DiagnosticsEmitted++ }},
	}
	for _, field := range fields {
		run := reference
		field.mutate(&run)
		if err := validatePhaseAccounting(reference, []runMetrics{run}, true); err == nil {
			t.Errorf("%s mismatch was accepted", field.name)
		}
	}
}

func TestValidatePhaseAccountingCanIgnoreDiagnosticDifferencesForValidation(t *testing.T) {
	reference := runMetrics{
		FilesDiscovered:    10,
		FilesParsed:        10,
		TotalLOC:           100,
		TotalBytes:         2000,
		DiagnosticsEmitted: 4,
	}
	run := reference
	run.DiagnosticsEmitted = 9
	if err := validatePhaseAccounting(reference, []runMetrics{run}, false); err != nil {
		t.Fatalf("file-accounting-only validation rejected diagnostic-only difference: %v", err)
	}
}

func TestValidatePhaseCVRejectsEitherComparedSideAboveThreshold(t *testing.T) {
	for _, label := range []string{"candidate", "baseline"} {
		reason := validatePhaseCV(label, phaseReport{
			Runs:             []runMetrics{{DurationMs: 100}, {DurationMs: 140}},
			CoefficientOfVar: 0.12,
		}, 0.05)
		if reason == "" {
			t.Fatalf("%s CV above threshold was accepted", label)
		}
		if !strings.Contains(reason, label) || !strings.Contains(reason, "0.1200") {
			t.Fatalf("%s CV rejection omitted useful context: %q", label, reason)
		}
	}
}

func TestValidatePhaseCVAcceptsThresholdAndSingleRunCases(t *testing.T) {
	cases := []struct {
		name  string
		phase phaseReport
		maxCV float64
	}{
		{
			name: "below threshold",
			phase: phaseReport{
				Runs:             []runMetrics{{DurationMs: 100}, {DurationMs: 110}},
				CoefficientOfVar: 0.05,
			},
			maxCV: 0.06,
		},
		{
			name: "single run has no CV gate",
			phase: phaseReport{
				Runs:             []runMetrics{{DurationMs: 100}},
				CoefficientOfVar: 1,
			},
			maxCV: 0.05,
		},
		{
			name: "disabled threshold",
			phase: phaseReport{
				Runs:             []runMetrics{{DurationMs: 100}, {DurationMs: 200}},
				CoefficientOfVar: 1,
			},
			maxCV: 0,
		},
	}
	for _, test := range cases {
		if reason := validatePhaseCV(test.name, test.phase, test.maxCV); reason != "" {
			t.Errorf("%s was rejected: %s", test.name, reason)
		}
	}
}
