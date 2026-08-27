package helper

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// BenchmarkResult tracks a single analysis run.
type BenchmarkResult struct {
	Timestamp   time.Time `json:"timestamp"`
	FilesCount  int       `json:"files_count"`
	LinesCount  int       `json:"lines_count"`
	ColdTime    float64   `json:"cold_time_seconds"`
	WarmTime    float64   `json:"warm_time_seconds"`
	PeakMemory  uint64    `json:"peak_memory_bytes"`
	Diagnostics int       `json:"diagnostics"`
	CacheHit    bool      `json:"cache_hit"`
	Commit      string    `json:"commit"`
}

// BenchmarkHistory stores results over time.
type BenchmarkHistory struct {
	Results []BenchmarkResult `json:"results"`
}

// LoadBenchmarkHistory reads historical results.
func LoadBenchmarkHistory(path string) (BenchmarkHistory, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return BenchmarkHistory{Results: []BenchmarkResult{}}, nil
	}
	if err != nil {
		return BenchmarkHistory{}, err
	}
	var h BenchmarkHistory
	err = json.Unmarshal(data, &h)
	return h, err
}

// SaveBenchmarkHistory writes results to disk.
func SaveBenchmarkHistory(path string, h BenchmarkHistory) error {
	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		os.MkdirAll(dir, 0755)
	}
	return os.WriteFile(path, data, 0644)
}

// GenerateBenchmarkReport creates HTML report from history.
func GenerateBenchmarkReport(history BenchmarkHistory, outPath string) error {
	html := generateBenchmarkHTML(history)
	return os.WriteFile(outPath, []byte(html), 0644)
}

func generateBenchmarkHTML(h BenchmarkHistory) string {
	if len(h.Results) == 0 {
		return "<h1>No benchmarks yet</h1>"
	}

	// Calculate stats
	latest := h.Results[len(h.Results)-1]
	avgWarm := avgField(h.Results, "warm")

	rows := ""
	for i, r := range h.Results {
		cacheSym := "—"
		if r.CacheHit {
			cacheSym = "✓"
		}
		rows += fmt.Sprintf(
			"<tr><td>%d</td><td>%s</td><td>%d</td><td>%d</td><td>%.2fs</td><td>%.2fs</td><td>%.1fMB</td><td>%s</td></tr>\n",
			i+1, r.Timestamp.Format("2006-01-02 15:04"), r.FilesCount, r.LinesCount,
			r.ColdTime, r.WarmTime, float64(r.PeakMemory)/(1024*1024), cacheSym,
		)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<title>go-phpcs Performance Dashboard</title>
	<style>
		body { font-family: Arial, sans-serif; margin: 20px; background: #f5f5f5; }
		h1 { color: #333; }
		.stats { display: grid; grid-template-columns: repeat(4, 1fr); gap: 20px; margin-bottom: 30px; }
		.stat { background: white; padding: 15px; border-radius: 5px; box-shadow: 0 2px 4px #0003; }
		.stat-label { font-size: 12px; color: #666; }
		.stat-value { font-size: 24px; font-weight: bold; color: #0066cc; }
		table { width: 100%%; background: white; border-collapse: collapse; box-shadow: 0 2px 4px #0003; }
		th, td { padding: 10px; text-align: left; border-bottom: 1px solid #eee; }
		th { background: #f9f9f9; font-weight: bold; }
		tr:hover { background: #f9f9f9; }
	</style>
</head>
<body>
	<h1>go-phpcs Performance Dashboard</h1>
	<p>Last benchmark: %s</p>

	<div class="stats">
		<div class="stat">
			<div class="stat-label">Latest Cold Time</div>
			<div class="stat-value">%.2fs</div>
		</div>
		<div class="stat">
			<div class="stat-label">Avg Warm Time</div>
			<div class="stat-value">%.2fs</div>
		</div>
		<div class="stat">
			<div class="stat-label">Peak Memory</div>
			<div class="stat-value">%.1fMB</div>
		</div>
		<div class="stat">
			<div class="stat-label">Runs</div>
			<div class="stat-value">%d</div>
		</div>
	</div>

	<table>
		<thead>
			<tr>
				<th>#</th>
				<th>Timestamp</th>
				<th>Files</th>
				<th>Lines</th>
				<th>Cold Time</th>
				<th>Warm Time</th>
				<th>Peak Memory</th>
				<th>Cache</th>
			</tr>
		</thead>
		<tbody>
			%s
		</tbody>
	</table>

	<p style="margin-top: 20px; font-size: 12px; color: #666;">
		Generated at %s
	</p>
</body>
</html>`,
		latest.Timestamp.Format("2006-01-02 15:04:05"),
		latest.ColdTime,
		avgWarm,
		float64(latest.PeakMemory)/(1024*1024),
		len(h.Results),
		rows,
		time.Now().Format("2006-01-02 15:04:05"),
	)
}

func avgField(results []BenchmarkResult, field string) float64 {
	if len(results) == 0 {
		return 0
	}
	sum := 0.0
	for _, r := range results {
		if field == "cold" {
			sum += r.ColdTime
		} else if field == "warm" {
			sum += r.WarmTime
		}
	}
	return sum / float64(len(results))
}

func minField(results []BenchmarkResult, field string) float64 {
	if len(results) == 0 {
		return 0
	}
	min := float64(^uint64(0))
	for _, r := range results {
		var val float64
		if field == "cold" {
			val = r.ColdTime
		} else if field == "warm" {
			val = r.WarmTime
		}
		if val < min {
			min = val
		}
	}
	return min
}

func maxField(results []BenchmarkResult, field string) float64 {
	if len(results) == 0 {
		return 0
	}
	max := 0.0
	for _, r := range results {
		var val float64
		if field == "cold" {
			val = r.ColdTime
		} else if field == "warm" {
			val = r.WarmTime
		}
		if val > max {
			max = val
		}
	}
	return max
}

// TrackMemory records memory usage during operation.
func TrackMemory() (uint64, func() uint64) {
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	start := m.Alloc

	return start, func() uint64 {
		runtime.GC()
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		if m.Alloc > start {
			return m.Alloc
		}
		return start
	}
}
