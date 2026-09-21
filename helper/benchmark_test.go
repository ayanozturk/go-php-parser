package helper

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func sampleHistory() BenchmarkHistory {
	return BenchmarkHistory{Results: []BenchmarkResult{
		{
			Timestamp:  time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
			FilesCount: 10,
			LinesCount: 1000,
			ColdTime:   2.0,
			WarmTime:   1.0,
			PeakMemory: 10 * 1024 * 1024,
			CacheHit:   false,
			Commit:     "aaa",
		},
		{
			Timestamp:  time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC),
			FilesCount: 12,
			LinesCount: 1200,
			ColdTime:   4.0,
			WarmTime:   2.0,
			PeakMemory: 20 * 1024 * 1024,
			CacheHit:   true,
			Commit:     "bbb",
		},
	}}
}

func TestLoadBenchmarkHistoryMissing(t *testing.T) {
	h, err := LoadBenchmarkHistory(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("expected nil error for missing file, got %v", err)
	}
	if len(h.Results) != 0 {
		t.Fatalf("expected empty history, got %d results", len(h.Results))
	}
}

func TestLoadBenchmarkHistoryReadError(t *testing.T) {
	dir := t.TempDir()
	// Reading a directory yields a non-IsNotExist error.
	if _, err := LoadBenchmarkHistory(dir); err == nil {
		t.Fatal("expected read error for directory path")
	}
}

func TestSaveBenchmarkHistoryMkdirFailure(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocked")
	if err := os.WriteFile(blocker, []byte("not a dir"), 0644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	path := filepath.Join(blocker, "nested", "history.json")
	if err := SaveBenchmarkHistory(path, sampleHistory()); err == nil {
		t.Fatal("expected MkdirAll failure when parent is a file")
	}
}

func TestGenerateBenchmarkReportMkdirFailure(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocked")
	if err := os.WriteFile(blocker, []byte("not a dir"), 0644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	path := filepath.Join(blocker, "nested", "report.html")
	if err := GenerateBenchmarkReport(sampleHistory(), path); err == nil {
		t.Fatal("expected MkdirAll failure when parent is a file")
	}
}


func TestSaveAndLoadBenchmarkHistoryRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "history.json")
	want := sampleHistory()
	if err := SaveBenchmarkHistory(path, want); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := LoadBenchmarkHistory(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(got.Results) != len(want.Results) {
		t.Fatalf("result count: got %d want %d", len(got.Results), len(want.Results))
	}
	if got.Results[1].CacheHit != true || got.Results[1].WarmTime != 2.0 {
		t.Fatalf("unexpected second result: %+v", got.Results[1])
	}
}

func TestSaveBenchmarkHistoryFlatPath(t *testing.T) {
	// filepath.Dir("history.json") == "." — MkdirAll must be skipped.
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getcwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	if err := SaveBenchmarkHistory("history.json", sampleHistory()); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := os.Stat("history.json"); err != nil {
		t.Fatalf("expected history.json: %v", err)
	}
}

func TestGenerateBenchmarkReportEmpty(t *testing.T) {
	out := filepath.Join(t.TempDir(), "reports", "empty.html")
	if err := GenerateBenchmarkReport(BenchmarkHistory{}, out); err != nil {
		t.Fatalf("generate: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got := string(data); got != "<h1>No benchmarks yet</h1>" {
		t.Fatalf("unexpected empty report: %q", got)
	}
}

func TestGenerateBenchmarkReportWithResults(t *testing.T) {
	out := filepath.Join(t.TempDir(), "dash", "report.html")
	if err := GenerateBenchmarkReport(sampleHistory(), out); err != nil {
		t.Fatalf("generate: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	html := string(data)
	for _, want := range []string{
		"go-phpcs Performance Dashboard",
		"Latest Cold Time",
		"Avg Warm Time",
		"✓",
		"—",
		"1.50s",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("report missing %q; got:\n%s", want, html)
		}
	}
}

func TestAvgMinMaxField(t *testing.T) {
	h := sampleHistory().Results
	if got := avgField(nil, "warm"); got != 0 {
		t.Fatalf("avg empty: got %v", got)
	}
	if got := avgField(h, "warm"); got != 1.5 {
		t.Fatalf("avg warm: got %v want 1.5", got)
	}
	if got := avgField(h, "cold"); got != 3.0 {
		t.Fatalf("avg cold: got %v want 3.0", got)
	}
	if got := avgField(h, "other"); got != 0 {
		t.Fatalf("avg unknown field: got %v", got)
	}
	if got := minField(nil, "warm"); got != 0 {
		t.Fatalf("min empty: got %v", got)
	}
	if got := minField(h, "warm"); got != 1.0 {
		t.Fatalf("min warm: got %v want 1.0", got)
	}
	if got := minField(h, "cold"); got != 2.0 {
		t.Fatalf("min cold: got %v want 2.0", got)
	}
	if got := minField(h, "other"); got != 0 {
		t.Fatalf("min unknown: got %v", got)
	}
	if got := maxField(nil, "warm"); got != 0 {
		t.Fatalf("max empty: got %v", got)
	}
	if got := maxField(h, "warm"); got != 2.0 {
		t.Fatalf("max warm: got %v want 2.0", got)
	}
	if got := maxField(h, "cold"); got != 4.0 {
		t.Fatalf("max cold: got %v want 4.0", got)
	}
	if got := maxField(h, "other"); got != 0 {
		t.Fatalf("max unknown: got %v", got)
	}
}

func TestTrackMemory(t *testing.T) {
	start, done := TrackMemory()
	if start == 0 {
		// Alloc can theoretically be 0; still require a closer.
		t.Log("start alloc was 0")
	}
	// Force some heap growth so the closer observes Alloc >= start.
	sink := make([]byte, 256*1024)
	for i := range sink {
		sink[i] = byte(i)
	}
	peak := done()
	if peak < start {
		t.Fatalf("peak %d should be >= start %d", peak, start)
	}
	_ = sink
}
