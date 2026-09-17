package syntax

import (
	"fmt"
	"hash/fnv"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
)

const defaultSyntaxBenchFiles = 128

type corpusBenchFile struct {
	path string
	size int64
	hash uint64
}

type corpusBenchInput struct {
	src []byte
	res *ParseResult
}

var (
	corpusBenchParseSink *ParseResult
	corpusBenchASTSink   []ast.Node
)

// BenchmarkSyntaxCorpus* is the fast, directional rung between the tiny
// fixture smoke test and the full process-cold corpus gate. Files are read and
// identity-checked before the timer starts, so the benchmark isolates parser
// work from filesystem noise. Use -benchtime=1x because one operation parses
// the entire selected sample.
//
// Example:
//
//	SYNTAX_BENCH_DIR="$PWD/test_projects/wordpress-develop" \
//	  go test ./syntax -run '^$' -bench '^BenchmarkSyntaxCorpus' \
//	  -benchtime=1x -count=5 -benchmem
//
// This benchmark is for rejecting weak optimization ideas quickly. Accepted
// performance claims still require the full corpus/accounting/CV protocol.
func BenchmarkSyntaxCorpusParseCST(b *testing.B) {
	inputs, totalBytes := loadCorpusBenchInputs(b, false)
	b.ReportAllocs()
	b.SetBytes(totalBytes)
	runtime.GC()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, input := range inputs {
			corpusBenchParseSink = Parse(input.src)
		}
	}
}

func BenchmarkSyntaxCorpusLowerAST(b *testing.B) {
	inputs, totalBytes := loadCorpusBenchInputs(b, true)
	b.ReportAllocs()
	b.SetBytes(totalBytes)
	runtime.GC()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, input := range inputs {
			corpusBenchASTSink = LowerAST(input.res)
		}
	}
}

func BenchmarkSyntaxCorpusParseAndLowerAST(b *testing.B) {
	inputs, totalBytes := loadCorpusBenchInputs(b, false)
	b.ReportAllocs()
	b.SetBytes(totalBytes)
	runtime.GC()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, input := range inputs {
			res := Parse(input.src)
			corpusBenchASTSink = LowerAST(res)
		}
	}
}

func loadCorpusBenchInputs(b *testing.B, keepParsed bool) ([]corpusBenchInput, int64) {
	b.Helper()
	root := strings.TrimSpace(os.Getenv("SYNTAX_BENCH_DIR"))
	if root == "" {
		b.Skip("set SYNTAX_BENCH_DIR to a PHP corpus root")
	}
	limit := defaultSyntaxBenchFiles
	if value := strings.TrimSpace(os.Getenv("SYNTAX_BENCH_FILES")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 {
			b.Fatalf("SYNTAX_BENCH_FILES must be a positive integer, got %q", value)
		}
		limit = parsed
	}

	files, err := collectCorpusBenchFiles(root)
	if err != nil {
		b.Fatal(err)
	}
	files = selectCorpusBenchFiles(files, limit)
	if len(files) == 0 {
		b.Fatalf("SYNTAX_BENCH_DIR=%q has no .php files", root)
	}

	inputs := make([]corpusBenchInput, 0, len(files))
	var totalBytes int64
	var diagnostics int
	for _, file := range files {
		src, err := os.ReadFile(file.path)
		if err != nil {
			b.Fatalf("read %s: %v", file.path, err)
		}
		res := Parse(src)
		if got := Print(res.File.Root); got != string(src) {
			b.Fatalf("identity failed before benchmark for %s", file.path)
		}
		diagnostics += len(res.Diagnostics)
		input := corpusBenchInput{src: src}
		if keepParsed {
			input.res = res
		}
		inputs = append(inputs, input)
		totalBytes += int64(len(src))
	}
	b.Logf("sample: files=%d bytes=%d diagnostics=%d", len(inputs), totalBytes, diagnostics)
	return inputs, totalBytes
}

func collectCorpusBenchFiles(root string) ([]corpusBenchFile, error) {
	var files []corpusBenchFile
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".php") {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		h := fnv.New64a()
		_, _ = h.Write([]byte(filepath.ToSlash(rel)))
		files = append(files, corpusBenchFile{path: path, size: info.Size(), hash: h.Sum64()})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk syntax benchmark corpus: %w", err)
	}
	return files, nil
}

// selectCorpusBenchFiles keeps the workload stable across machines and covers
// both the general corpus and its allocation-heavy tail. One eighth of the
// sample is reserved for the largest files; the rest is a stable path-hash
// sample. Pinned corpora therefore produce the same selection every run.
func selectCorpusBenchFiles(files []corpusBenchFile, limit int) []corpusBenchFile {
	if limit <= 0 || len(files) <= limit {
		return append([]corpusBenchFile(nil), files...)
	}

	bySize := append([]corpusBenchFile(nil), files...)
	sort.Slice(bySize, func(i, j int) bool {
		if bySize[i].size != bySize[j].size {
			return bySize[i].size > bySize[j].size
		}
		return bySize[i].path < bySize[j].path
	})
	largestCount := limit / 8
	if largestCount < 1 {
		largestCount = 1
	}
	selected := make(map[string]struct{}, limit)
	result := make([]corpusBenchFile, 0, limit)
	for _, file := range bySize[:largestCount] {
		selected[file.path] = struct{}{}
		result = append(result, file)
	}

	byHash := append([]corpusBenchFile(nil), files...)
	sort.Slice(byHash, func(i, j int) bool {
		if byHash[i].hash != byHash[j].hash {
			return byHash[i].hash < byHash[j].hash
		}
		return byHash[i].path < byHash[j].path
	})
	for _, file := range byHash {
		if len(result) == limit {
			break
		}
		if _, ok := selected[file.path]; ok {
			continue
		}
		selected[file.path] = struct{}{}
		result = append(result, file)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].path < result[j].path })
	return result
}

func TestSelectCorpusBenchFilesIsStableAndKeepsLargest(t *testing.T) {
	files := []corpusBenchFile{
		{path: "a.php", size: 10, hash: 50},
		{path: "b.php", size: 20, hash: 40},
		{path: "c.php", size: 30, hash: 30},
		{path: "d.php", size: 40, hash: 20},
		{path: "largest.php", size: 1000, hash: 100},
	}
	first := selectCorpusBenchFiles(files, 3)
	second := selectCorpusBenchFiles(files, 3)
	if fmt.Sprint(first) != fmt.Sprint(second) {
		t.Fatalf("selection is not stable: first=%v second=%v", first, second)
	}
	if len(first) != 3 {
		t.Fatalf("selected %d files, want 3", len(first))
	}
	foundLargest := false
	for _, file := range first {
		if file.path == "largest.php" {
			foundLargest = true
		}
	}
	if !foundLargest {
		t.Fatalf("largest file missing from sample: %v", first)
	}
}
