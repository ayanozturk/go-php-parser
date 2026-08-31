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

func TestWorkerEnvPinsGOMAXPROCS(t *testing.T) {
	t.Setenv("GOMAXPROCS", "99")
	env := workerEnv(4)
	found := false
	for _, kv := range env {
		if kv == "GOMAXPROCS=99" {
			t.Fatal("parent GOMAXPROCS leaked into worker environment")
		}
		if kv == "GOMAXPROCS=4" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected GOMAXPROCS=4 in worker environment, got %v", env)
	}
}

func TestShouldExtendColdRunsRequiresFailingCVAndBudget(t *testing.T) {
	unstable := phaseReport{Runs: []runMetrics{{DurationMs: 100}, {DurationMs: 140}}, CoefficientOfVar: 0.12}
	stable := phaseReport{Runs: []runMetrics{{DurationMs: 100}, {DurationMs: 102}}, CoefficientOfVar: 0.02}
	if shouldExtendColdRuns(stable, nil, 0.05, 10) {
		t.Fatal("stable candidate should not extend")
	}
	if !shouldExtendColdRuns(unstable, nil, 0.05, 10) {
		t.Fatal("unstable candidate should extend while budget remains")
	}
	if shouldExtendColdRuns(unstable, nil, 0.05, 0) {
		t.Fatal("exhausted extra budget should not extend")
	}
	if shouldExtendColdRuns(unstable, nil, 0, 10) {
		t.Fatal("disabled CV gate should not extend")
	}
	if !shouldExtendColdRuns(stable, &unstable, 0.05, 1) {
		t.Fatal("unstable baseline should extend the interleaved pair")
	}
}

func TestExtraInterleavedPairContinuesBalancedOrder(t *testing.T) {
	even := extraInterleavedPair(0)
	if !reflect.DeepEqual(even, []benchmarkRunTarget{benchmarkCandidate, benchmarkBaseline}) {
		t.Fatalf("even extra pair = %v", even)
	}
	odd := extraInterleavedPair(1)
	if !reflect.DeepEqual(odd, []benchmarkRunTarget{benchmarkBaseline, benchmarkCandidate}) {
		t.Fatalf("odd extra pair = %v", odd)
	}
}

func TestSummarizeDropMaxCVIsInformationalAndDoesNotBypassGate(t *testing.T) {
	report := summarize([]runMetrics{
		{DurationMs: 100},
		{DurationMs: 102},
		{DurationMs: 101},
		{DurationMs: 180},
	})
	if report.CoefficientOfVarDropMax <= 0 || report.CoefficientOfVar <= report.CoefficientOfVarDropMax {
		t.Fatalf("expected full CV to exceed drop-max CV, got %#v", report)
	}
	if reason := validatePhaseCV("candidate", report, 0.05); reason == "" {
		t.Fatal("CV gate must still reject the full sample including the outlier")
	}
}
