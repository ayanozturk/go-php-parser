// Command phpstan-compat measures diagnostic compatibility with PHPStan on a
// shared source corpus. The headline compatibility percentage is the F1 score:
// the harmonic mean of exact-location diagnostic precision and recall.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ayanozturk/go-php-parser/command"
)

const schemaVersion = 1

type diagnostic struct {
	Path       string `json:"path"`
	Line       int    `json:"line"`
	Code       string `json:"code,omitempty"`
	Identifier string `json:"identifier,omitempty"`
}

type levelMetrics struct {
	Level                int     `json:"level"`
	CompatibilityPct     float64 `json:"compatibilityPct"`
	PrecisionPct         float64 `json:"precisionPct"`
	RecallPct            float64 `json:"recallPct"`
	ExactMatches         int     `json:"exactMatches"`
	EngineDiagnostics    int     `json:"engineDiagnostics"`
	PHPStanDiagnostics   int     `json:"phpstanDiagnostics"`
	EngineOnly           int     `json:"engineOnly"`
	PHPStanOnly          int     `json:"phpstanOnly"`
	UnmappedEngine       int     `json:"unmappedEngineDiagnostics"`
	FilesDiscovered      int     `json:"filesDiscovered"`
	FilesAnalyzed        int     `json:"filesAnalyzed"`
	PHPStanFilesAnalyzed int     `json:"phpstanFilesAnalyzed"`
}

type report struct {
	SchemaVersion   int               `json:"schemaVersion"`
	GeneratedAt     string            `json:"generatedAt"`
	Root            string            `json:"root"`
	Paths           []string          `json:"paths"`
	PHPStanVersion  string            `json:"phpstanVersion"`
	Matching        string            `json:"matching"`
	HeadlineMetric  string            `json:"headlineMetric"`
	CrosswalkSource []crosswalkSource `json:"crosswalkSources"`
	Levels          []levelMetrics    `json:"levels"`
}

type crosswalkSource struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type fixtureManifest struct {
	SchemaVersion int `json:"schemaVersion"`
	Reference     struct {
		Tool  string `json:"tool"`
		Level int    `json:"level"`
	} `json:"reference"`
	Cases []struct {
		ID                 string   `json:"id"`
		Capability         string   `json:"capability"`
		File               string   `json:"file"`
		EngineCodes        []string `json:"engineCodes"`
		PHPStanIdentifiers []string `json:"phpstanIdentifiers"`
	} `json:"cases"`
}

type phpstanOutput struct {
	Files map[string]struct {
		Messages []struct {
			Line       int    `json:"line"`
			Identifier string `json:"identifier"`
		} `json:"messages"`
	} `json:"files"`
	Errors []string `json:"errors"`
}

func main() {
	root := flag.String("root", ".", "project root used to normalize diagnostic paths")
	pathsFlag := flag.String("paths", "", "comma-separated first-party files/directories to report (default: root)")
	indexPathsFlag := flag.String("index-paths", "", "comma-separated extra files/directories indexed only for symbol resolution")
	levelsFlag := flag.String("levels", "0,1,2,3,5,6,7,8", "comma-separated PHPStan levels")
	phpstanBin := flag.String("phpstan-bin", "phpstan", "PHPStan executable")
	phpstanConfig := flag.String("phpstan-config", "", "PHPStan configuration file")
	crosswalkGlob := flag.String("crosswalk", "testdata/diagnostic-differential*/manifest.json", "glob of differential manifests used as the identifier crosswalk")
	workers := flag.Int("workers", runtime.NumCPU(), "engine analysis workers")
	jsonOutput := flag.Bool("json", false, "emit JSON")
	output := flag.String("output", "", "optional report output file")
	flag.Parse()

	result, err := run(*root, splitList(*pathsFlag), splitList(*indexPathsFlag), *levelsFlag, *phpstanBin, *phpstanConfig, *crosswalkGlob, *workers)
	if err != nil {
		fmt.Fprintln(os.Stderr, "phpstan-compat:", err)
		os.Exit(2)
	}

	var out io.Writer = os.Stdout
	if *output != "" {
		file, err := os.Create(*output)
		if err != nil {
			fmt.Fprintln(os.Stderr, "phpstan-compat:", err)
			os.Exit(2)
		}
		defer file.Close()
		out = file
	}
	if *jsonOutput {
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			fmt.Fprintln(os.Stderr, "phpstan-compat:", err)
			os.Exit(2)
		}
		return
	}
	printReport(out, result)
}

func run(root string, reportPaths, indexPaths []string, levelsText, phpstanBin, phpstanConfig, crosswalkGlob string, workers int) (report, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return report{}, err
	}
	if len(reportPaths) == 0 {
		reportPaths = []string{"."}
	}
	levels, err := parseLevels(levelsText)
	if err != nil {
		return report{}, err
	}
	if workers < 1 {
		workers = 1
	}
	reportable, err := collectPHPFiles(absPaths(absRoot, reportPaths))
	if err != nil {
		return report{}, err
	}
	if len(reportable) == 0 {
		return report{}, errors.New("no PHP files selected")
	}
	indexed, err := collectPHPFiles(append(append([]string{}, reportable...), absPaths(absRoot, indexPaths)...))
	if err != nil {
		return report{}, err
	}
	crosswalk, sources, err := loadCrosswalk(crosswalkGlob)
	if err != nil {
		return report{}, err
	}
	phpstanBin = resolveExecutable(absRoot, phpstanBin)
	version, err := phpstanVersion(absRoot, phpstanBin)
	if err != nil {
		return report{}, err
	}

	result := report{SchemaVersion: schemaVersion, GeneratedAt: time.Now().UTC().Format(time.RFC3339), Root: filepath.ToSlash(absRoot), Paths: reportPaths, PHPStanVersion: version, Matching: "exact normalized path + start line + compatible identifier", HeadlineMetric: "F1 (harmonic mean of diagnostic precision and recall)", CrosswalkSource: sources}
	for _, level := range levels {
		engine, _, err := runEngine(absRoot, indexed, reportable, level, workers)
		if err != nil {
			return report{}, fmt.Errorf("level %d engine: %w", level, err)
		}
		reference, referenceFiles, err := runPHPStan(absRoot, phpstanBin, phpstanConfig, reportPaths, reportable, level)
		if err != nil {
			return report{}, fmt.Errorf("level %d PHPStan: %w", level, err)
		}
		metrics := score(level, engine, reference, crosswalk)
		// The compatibility denominator is the first-party report manifest.
		// Extra index-only dependency files must not inflate its accounting.
		metrics.FilesDiscovered = len(reportable)
		metrics.FilesAnalyzed = len(reportable)
		metrics.PHPStanFilesAnalyzed = referenceFiles
		result.Levels = append(result.Levels, metrics)
	}
	return result, nil
}

func runEngine(root string, indexed, reportable []string, level, workers int) ([]diagnostic, command.AnalyzeResult, error) {
	cacheDir, err := os.MkdirTemp("", "go-php-parser-compat-cache-")
	if err != nil {
		return nil, command.AnalyzeResult{}, fmt.Errorf("create temporary cache: %w", err)
	}
	defer os.RemoveAll(cacheDir)

	result := command.AnalyzeFilesIncremental(indexed, &level, nil, workers, cacheDir)
	if len(result.ReadErrors) > 0 || len(result.ParseErrors) > 0 {
		return nil, result, fmt.Errorf("incomplete accounting: %d read errors, %d parse-error files", len(result.ReadErrors), len(result.ParseErrors))
	}
	reportSet := make(map[string]struct{}, len(reportable))
	for _, path := range reportable {
		reportSet[filepath.Clean(path)] = struct{}{}
	}
	diagnostics := make([]diagnostic, 0, len(result.Issues))
	for _, issue := range result.Issues {
		if _, ok := reportSet[filepath.Clean(issue.Filename)]; !ok {
			continue
		}
		diagnostics = append(diagnostics, diagnostic{Path: normalizePath(root, issue.Filename), Line: issue.Line, Code: issue.Code})
	}
	sortDiagnostics(diagnostics)
	return diagnostics, result, nil
}

func runPHPStan(root, binary, configuration string, reportPaths, reportable []string, level int) ([]diagnostic, int, error) {
	args := []string{"analyse", "--debug", "--no-progress", "--error-format=json", "--level=" + strconv.Itoa(level)}
	if configuration != "" {
		args = append(args, "--configuration="+configuration)
	}
	args = append(args, reportPaths...)
	command := exec.Command(binary, args...)
	command.Dir = root
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	if err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) || stdout.Len() == 0 {
			return nil, 0, fmt.Errorf("run: %w", err)
		}
	}
	jsonStart := phpstanJSONStart(stdout.Bytes())
	if jsonStart < 0 {
		return nil, 0, errors.New("PHPStan output did not contain a JSON report")
	}
	var decoded phpstanOutput
	if err := json.Unmarshal(stdout.Bytes()[jsonStart:], &decoded); err != nil {
		return nil, 0, fmt.Errorf("decode JSON: %w", err)
	}
	if len(decoded.Errors) > 0 {
		return nil, 0, fmt.Errorf("analysis errors: %s", strings.Join(decoded.Errors, "; "))
	}
	analyzedFiles := phpstanAnalyzedFiles(root, stdout.String()[:jsonStart]+"\n"+stderr.String())
	expectedFiles := normalizedPathSet(root, reportable)
	if missing, extra := setDifference(expectedFiles, analyzedFiles), setDifference(analyzedFiles, expectedFiles); len(missing) > 0 || len(extra) > 0 {
		return nil, 0, fmt.Errorf("file accounting mismatch: PHPStan omitted %d selected files and analyzed %d extra files (omitted sample: %s; extra sample: %s)", len(missing), len(extra), firstValue(missing), firstValue(extra))
	}

	diagnostics := make([]diagnostic, 0)
	for path, file := range decoded.Files {
		normalized := normalizePath(root, path)
		if !expectedFiles[normalized] {
			return nil, 0, fmt.Errorf("diagnostic outside selected manifest: %s", normalized)
		}
		for _, message := range file.Messages {
			if message.Identifier == "" {
				return nil, 0, fmt.Errorf("diagnostic at %s:%d has no identifier", path, message.Line)
			}
			diagnostics = append(diagnostics, diagnostic{Path: normalized, Line: message.Line, Identifier: message.Identifier})
		}
	}
	sortDiagnostics(diagnostics)
	return diagnostics, len(analyzedFiles), nil
}

func score(level int, engine, reference []diagnostic, crosswalk map[int]map[string]map[string]bool) levelMetrics {
	metrics := levelMetrics{Level: level, EngineDiagnostics: len(engine), PHPStanDiagnostics: len(reference)}
	adjacency := make([][]int, len(engine))
	for candidateIndex, candidate := range engine {
		allowed := identifiersAtLevel(crosswalk, level, candidate.Code)
		if len(allowed) == 0 {
			metrics.UnmappedEngine++
			continue
		}
		for i, expected := range reference {
			if candidate.Path == expected.Path && candidate.Line == expected.Line && allowed[expected.Identifier] {
				adjacency[candidateIndex] = append(adjacency[candidateIndex], i)
			}
		}
	}
	metrics.ExactMatches = maximumMatches(adjacency, len(reference))
	metrics.EngineOnly = len(engine) - metrics.ExactMatches
	metrics.PHPStanOnly = len(reference) - metrics.ExactMatches
	metrics.PrecisionPct = percentage(metrics.ExactMatches, len(engine))
	metrics.RecallPct = percentage(metrics.ExactMatches, len(reference))
	if len(engine)+len(reference) == 0 {
		metrics.CompatibilityPct = 100
	} else {
		metrics.CompatibilityPct = round(float64(2*metrics.ExactMatches) * 100 / float64(len(engine)+len(reference)))
	}
	return metrics
}

func loadCrosswalk(pattern string) (map[int]map[string]map[string]bool, []crosswalkSource, error) {
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return nil, nil, err
	}
	if len(paths) == 0 {
		return nil, nil, fmt.Errorf("crosswalk glob %q matched no manifests", pattern)
	}
	sort.Strings(paths)
	result := make(map[int]map[string]map[string]bool)
	sources := make([]crosswalkSource, 0, len(paths))
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, err
		}
		var manifest fixtureManifest
		decoder := json.NewDecoder(bytes.NewReader(content))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&manifest); err != nil {
			return nil, nil, fmt.Errorf("decode %s: %w", path, err)
		}
		if manifest.SchemaVersion != schemaVersion || manifest.Reference.Tool != "PHPStan" {
			return nil, nil, fmt.Errorf("unsupported crosswalk manifest %s", path)
		}
		level := manifest.Reference.Level
		if level < 0 || level > 10 {
			return nil, nil, fmt.Errorf("crosswalk manifest %s has invalid PHPStan level %d", path, level)
		}
		if result[level] == nil {
			result[level] = make(map[string]map[string]bool)
		}
		for _, fixture := range manifest.Cases {
			for _, code := range fixture.EngineCodes {
				if result[level][code] == nil {
					result[level][code] = make(map[string]bool)
				}
				for _, identifier := range fixture.PHPStanIdentifiers {
					result[level][code][identifier] = true
				}
			}
		}
		sources = append(sources, crosswalkSource{Path: filepath.ToSlash(path), SHA256: fmt.Sprintf("%x", sha256.Sum256(content))})
	}
	return result, sources, nil
}

func collectPHPFiles(paths []string) ([]string, error) {
	seen := make(map[string]struct{})
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			if strings.EqualFold(filepath.Ext(path), ".php") {
				seen[filepath.Clean(path)] = struct{}{}
			}
			continue
		}
		if err := filepath.WalkDir(path, func(candidate string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() && candidate != path && (entry.Name() == ".git" || entry.Name() == ".cache") {
				return filepath.SkipDir
			}
			if !entry.IsDir() && strings.EqualFold(filepath.Ext(candidate), ".php") {
				seen[filepath.Clean(candidate)] = struct{}{}
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}
	files := make([]string, 0, len(seen))
	for path := range seen {
		files = append(files, path)
	}
	sort.Strings(files)
	return files, nil
}

func parseLevels(value string) ([]int, error) {
	seen := make(map[int]bool)
	var levels []int
	for _, part := range splitList(value) {
		level, err := strconv.Atoi(part)
		if err != nil || level < 0 || level > 10 {
			return nil, fmt.Errorf("invalid level %q", part)
		}
		if !seen[level] {
			seen[level] = true
			levels = append(levels, level)
		}
	}
	if len(levels) == 0 {
		return nil, errors.New("at least one level is required")
	}
	sort.Ints(levels)
	return levels, nil
}

func phpstanVersion(root, binary string) (string, error) {
	command := exec.Command(binary, "--version")
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s --version: %w: %s", binary, err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

func resolveExecutable(root, binary string) string {
	if filepath.IsAbs(binary) || !strings.ContainsRune(binary, filepath.Separator) {
		return binary
	}
	return filepath.Join(root, binary)
}

func identifiersAtLevel(crosswalk map[int]map[string]map[string]bool, level int, code string) map[string]bool {
	result := make(map[string]bool)
	for reviewedLevel, codes := range crosswalk {
		if reviewedLevel > level {
			continue
		}
		for identifier := range codes[code] {
			result[identifier] = true
		}
	}
	return result
}

func maximumMatches(adjacency [][]int, referenceCount int) int {
	matchedEngine := make([]int, referenceCount)
	for i := range matchedEngine {
		matchedEngine[i] = -1
	}
	matches := 0
	for engineIndex := range adjacency {
		seen := make([]bool, referenceCount)
		var augment func(int) bool
		augment = func(candidate int) bool {
			for _, referenceIndex := range adjacency[candidate] {
				if seen[referenceIndex] {
					continue
				}
				seen[referenceIndex] = true
				if matchedEngine[referenceIndex] == -1 || augment(matchedEngine[referenceIndex]) {
					matchedEngine[referenceIndex] = candidate
					return true
				}
			}
			return false
		}
		if augment(engineIndex) {
			matches++
		}
	}
	return matches
}

func phpstanAnalyzedFiles(root, debugOutput string) map[string]bool {
	result := make(map[string]bool)
	for _, line := range strings.Split(debugOutput, "\n") {
		path := strings.TrimSpace(line)
		if !strings.EqualFold(filepath.Ext(path), ".php") {
			continue
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			result[normalizePath(root, path)] = true
		}
	}
	return result
}

func phpstanJSONStart(output []byte) int {
	const marker = "{\"totals\":"
	if bytes.HasPrefix(output, []byte(marker)) {
		return 0
	}
	index := bytes.LastIndex(output, []byte("\n"+marker))
	if index < 0 {
		return -1
	}
	return index + 1
}

func normalizedPathSet(root string, paths []string) map[string]bool {
	result := make(map[string]bool, len(paths))
	for _, path := range paths {
		result[normalizePath(root, path)] = true
	}
	return result
}

func setDifference(left, right map[string]bool) []string {
	var result []string
	for value := range left {
		if !right[value] {
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

func firstValue(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return values[0]
}

func printReport(out io.Writer, result report) {
	fmt.Fprintf(out, "PHPStan compatibility (%s)\n", result.PHPStanVersion)
	fmt.Fprintln(out, "Compatibility is F1 over exact path + line + identifier matches.")
	for _, level := range result.Levels {
		fmt.Fprintf(out, "Level %d: %.2f%% PHPStan compatible (precision %.2f%%, recall %.2f%%, exact %d, engine-only %d, PHPStan-only %d, unmapped %d)\n", level.Level, level.CompatibilityPct, level.PrecisionPct, level.RecallPct, level.ExactMatches, level.EngineOnly, level.PHPStanOnly, level.UnmappedEngine)
	}
}

func splitList(value string) []string {
	var result []string
	for _, part := range strings.Split(value, ",") {
		if part = strings.TrimSpace(part); part != "" {
			result = append(result, part)
		}
	}
	return result
}

func absPaths(root string, paths []string) []string {
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		result = append(result, filepath.Clean(path))
	}
	return result
}

func normalizePath(root, path string) string {
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	rel, err := filepath.Rel(root, filepath.Clean(path))
	if err != nil {
		return filepath.ToSlash(filepath.Clean(path))
	}
	return filepath.ToSlash(rel)
}

func sortDiagnostics(values []diagnostic) {
	sort.Slice(values, func(i, j int) bool {
		if values[i].Path != values[j].Path {
			return values[i].Path < values[j].Path
		}
		if values[i].Line != values[j].Line {
			return values[i].Line < values[j].Line
		}
		if values[i].Code != values[j].Code {
			return values[i].Code < values[j].Code
		}
		return values[i].Identifier < values[j].Identifier
	})
}

func percentage(numerator, denominator int) float64 {
	if denominator == 0 {
		if numerator == 0 {
			return 100
		}
		return 0
	}
	return round(float64(numerator) * 100 / float64(denominator))
}

func round(value float64) float64 {
	parsed, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", value), 64)
	return parsed
}
