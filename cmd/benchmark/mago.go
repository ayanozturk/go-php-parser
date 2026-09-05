package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type magoReport struct {
	Binary             string      `json:"binary"`
	Version            string      `json:"version"`
	Config             string      `json:"config"`
	Workspace          string      `json:"workspace"`
	Threads            int         `json:"threads"`
	SourceFiles        int         `json:"sourceFiles"`
	AnalyzerFiles      int         `json:"analyzerFiles"`
	ValidationRun      runMetrics  `json:"validationRun"`
	ColdFullAnalysis   phaseReport `json:"coldFullAnalysis"`
	SemanticComparable bool        `json:"semanticComparable"`
}

func magoVersion(binary string) string {
	cmd := exec.Command(binary, "--version")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func magoListFileCount(binary, workspace, config, command string) (int, error) {
	args := magoGlobalArgs(workspace, config, 0)
	args = append(args, "list-files")
	if command != "" {
		args = append(args, "--command", command)
	}
	cmd := exec.Command(binary, args...)
	cmd.Env = magoEnv(0)
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("mago list-files: %w", err)
	}
	count := 0
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "INFO") || strings.HasPrefix(line, "WARN") {
			continue
		}
		count++
	}
	return count, nil
}

func execMagoAnalyze(binary, workspace, config string, threads, sourceFiles, analyzerFiles int) (runMetrics, error) {
	if threads < 1 {
		threads = 1
	}
	args := magoGlobalArgs(workspace, config, threads)
	args = append(args, "analyze", "--reporting-format", "count")
	cmd := exec.Command(binary, args...)
	cmd.Env = magoEnv(threads)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	start := time.Now()
	runErr := cmd.Run()
	duration := time.Since(start)
	diagnostics, parseErr := parseMagoCountOutput(stdout.String())
	if parseErr != nil {
		if runErr != nil {
			return runMetrics{}, fmt.Errorf("mago analyze: %w\nstderr: %s", runErr, stderr.String())
		}
		return runMetrics{}, fmt.Errorf("mago analyze count output: %w\nstderr: %s", parseErr, stderr.String())
	}
	result := runMetrics{
		DurationMs:         duration.Milliseconds(),
		FilesDiscovered:    sourceFiles,
		FilesParsed:        analyzerFiles,
		FilesFailed:        0,
		DiagnosticsEmitted: diagnostics,
	}
	if rss := peakRSSBytes(cmd.ProcessState); rss > 0 {
		result.PeakRSSBytes = rss
	}
	return result, nil
}

func magoGlobalArgs(workspace, config string, threads int) []string {
	args := []string{
		"--workspace", workspace,
		"--config", config,
		"--colors", "never",
		"--no-version-check",
	}
	if threads > 0 {
		args = append(args, "--threads", strconv.Itoa(threads))
	}
	return args
}

func magoEnv(threads int) []string {
	env := os.Environ()
	out := make([]string, 0, len(env)+2)
	for _, kv := range env {
		if strings.HasPrefix(kv, "MAGO_THREADS=") || strings.HasPrefix(kv, "GOMAXPROCS=") {
			continue
		}
		out = append(out, kv)
	}
	if threads > 0 {
		out = append(out, "MAGO_THREADS="+strconv.Itoa(threads))
	}
	return out
}

func parseMagoCountOutput(output string) (int, error) {
	total := 0
	found := false
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		switch strings.TrimSpace(strings.ToLower(key)) {
		case "error", "warning", "note", "help":
			count, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return 0, fmt.Errorf("invalid mago count line %q: %w", line, err)
			}
			total += count
			found = true
		}
	}
	if !found {
		return 0, fmt.Errorf("no mago count totals in output %q", strings.TrimSpace(output))
	}
	return total, nil
}
