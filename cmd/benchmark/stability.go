package main

import (
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

type hostEnvironment struct {
	GoMaxProcs         int     `json:"goMaxProcs"`
	ProcessColdWarmups int     `json:"processColdWarmups"`
	SettleMs           int     `json:"settleMs"`
	ExtraColdBudget    int     `json:"extraColdRunBudget"`
	LoadAverage1       float64 `json:"loadAverage1,omitempty"`
	LoadAverage5       float64 `json:"loadAverage5,omitempty"`
	LoadAverage15      float64 `json:"loadAverage15,omitempty"`
}

func workerEnv(workers int) []string {
	if workers < 1 {
		workers = 1
	}
	env := os.Environ()
	out := make([]string, 0, len(env)+1)
	for _, kv := range env {
		if strings.HasPrefix(kv, "GOMAXPROCS=") {
			continue
		}
		out = append(out, kv)
	}
	return append(out, "GOMAXPROCS="+strconv.Itoa(workers))
}

func extraInterleavedPair(completedPerEngine int) []benchmarkRunTarget {
	if completedPerEngine%2 == 0 {
		return []benchmarkRunTarget{benchmarkCandidate, benchmarkBaseline}
	}
	return []benchmarkRunTarget{benchmarkBaseline, benchmarkCandidate}
}

func shouldExtendColdRuns(candidate phaseReport, baseline *phaseReport, maxCV float64, extraRemaining int) bool {
	if extraRemaining <= 0 || maxCV <= 0 {
		return false
	}
	if validatePhaseCV("candidate", candidate, maxCV) != "" {
		return true
	}
	if baseline != nil && validatePhaseCV("baseline", *baseline, maxCV) != "" {
		return true
	}
	return false
}

func settle(ms int) {
	if ms > 0 {
		time.Sleep(time.Duration(ms) * time.Millisecond)
	}
}

func dropOneMax(durations []float64) []float64 {
	if len(durations) < 2 {
		return append([]float64(nil), durations...)
	}
	maxIdx := 0
	for i, duration := range durations {
		if duration > durations[maxIdx] {
			maxIdx = i
		}
	}
	out := make([]float64, 0, len(durations)-1)
	out = append(out, durations[:maxIdx]...)
	out = append(out, durations[maxIdx+1:]...)
	return out
}

func coefficientOfVariation(durations []float64) float64 {
	if len(durations) < 2 {
		return 0
	}
	sum := 0.0
	for _, duration := range durations {
		sum += duration
	}
	mean := sum / float64(len(durations))
	if mean == 0 {
		return 0
	}
	variance := 0.0
	for _, duration := range durations {
		diff := duration - mean
		variance += diff * diff
	}
	variance /= float64(len(durations))
	return math.Sqrt(variance) / mean
}
