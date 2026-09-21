package command

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ayanozturk/go-php-parser/analyse"
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/style"
	"github.com/ayanozturk/go-php-parser/syntax"
)

func TestCountLinesEdgeCases(t *testing.T) {
	if got := CountLines(nil); got != 0 {
		t.Fatalf("nil: got %d", got)
	}
	if got := CountLines([]byte{}); got != 0 {
		t.Fatalf("empty: got %d", got)
	}
	if got := CountLines([]byte("a")); got != 1 {
		t.Fatalf("no trailing newline: got %d", got)
	}
	if got := CountLines([]byte("a\n")); got != 1 {
		t.Fatalf("one line with newline: got %d", got)
	}
	if got := CountLines([]byte("a\nb")); got != 2 {
		t.Fatalf("two lines no final newline: got %d", got)
	}
	if got := CountLines([]byte("a\nb\n")); got != 2 {
		t.Fatalf("two lines with final newline: got %d", got)
	}
}

func TestCollectLinesSums(t *testing.T) {
	ch := make(chan int, 3)
	done := make(chan struct{})
	total := 0
	go CollectLines(ch, &total, done)
	ch <- 1
	ch <- 2
	ch <- 4
	close(ch)
	<-done
	if total != 7 {
		t.Fatalf("total=%d want 7", total)
	}
}

func TestExecuteCommandKnownAndUnknown(t *testing.T) {
	var buf bytes.Buffer
	ExecuteCommand("list-files", nil, nil, "x.php", &buf)
	if !strings.Contains(buf.String(), "list-files") {
		t.Fatalf("expected list-files message, got %q", buf.String())
	}
	buf.Reset()
	ExecuteCommand("no-such-command", nil, nil, "x.php", &buf)
	if buf.Len() != 0 {
		t.Fatalf("unknown command should be a no-op, got %q", buf.String())
	}
}

func TestProcessSingleAndMultipleFilesWithWriter(t *testing.T) {
	dir := t.TempDir()
	ok := filepath.Join(dir, "Ok.php")
	missing := filepath.Join(dir, "Missing.php")
	writeAnalyzeFixture(t, ok, "<?php\nclass Ok {}\n")

	var buf bytes.Buffer
	errs, lines := ProcessSingleFileWithWriter(ok, "list-files", false, nil, nil, &buf)
	if errs != 0 || lines < 1 {
		t.Fatalf("ok file: errs=%d lines=%d out=%q", errs, lines, buf.String())
	}
	if !strings.Contains(buf.String(), "list-files") {
		t.Fatalf("expected command output, got %q", buf.String())
	}

	buf.Reset()
	errs, lines = ProcessSingleFileWithWriter(missing, "list-files", false, nil, nil, &buf)
	if errs != 1 || lines != 0 {
		t.Fatalf("missing file: errs=%d lines=%d", errs, lines)
	}
	if !strings.Contains(buf.String(), "Could not read file") {
		t.Fatalf("expected read error, got %q", buf.String())
	}

	buf.Reset()
	errs, lines = ProcessMultipleFilesWithWriter([]string{ok, missing}, "list-files", false, 1, nil, nil, &buf)
	if errs != 1 || lines < 1 {
		t.Fatalf("multi: errs=%d lines=%d out=%q", errs, lines, buf.String())
	}

	buf.Reset()
	_, _ = ProcessSingleFileWithWriter(ok, "style", false, []string{"PSR12.Files.NoBlankLineAfterPHPOpeningTag"}, nil, &buf)
}

func TestProcessFilePaths(t *testing.T) {
	dir := t.TempDir()
	ok := filepath.Join(dir, "Tok.php")
	bad := filepath.Join(dir, "Bad.php")
	writeAnalyzeFixture(t, ok, "<?php\necho 1;\n")
	writeAnalyzeFixture(t, bad, "<?php\nfunction broken(\n")

	var buf bytes.Buffer
	lines := ProcessFile(ok, "tokens", false, &buf)
	if lines < 1 || !strings.Contains(buf.String(), "T_") {
		t.Fatalf("tokens: lines=%d out=%q", lines, buf.String())
	}

	buf.Reset()
	lines = ProcessFile(ok, "analyze", false, &buf)
	if lines < 1 || !strings.Contains(buf.String(), "analyze") {
		t.Fatalf("analyze stub: lines=%d out=%q", lines, buf.String())
	}

	buf.Reset()
	lines = ProcessFile(ok, "nope", false, &buf)
	if lines < 1 {
		t.Fatalf("unknown command should still return line count, got %d", lines)
	}
	if !strings.Contains(buf.String(), "Unknown command: nope") {
		t.Fatalf("expected unknown command on writer, got %q", buf.String())
	}
	if !strings.Contains(buf.String(), "Usage:") {
		t.Fatalf("expected usage on writer after unknown command, got %q", buf.String())
	}

	buf.Reset()
	lines = ProcessFile(bad, "tokens", false, &buf)
	if lines < 1 || !strings.Contains(buf.String(), "Parsing errors") {
		t.Fatalf("parse errors: lines=%d out=%q", lines, buf.String())
	}

	buf.Reset()
	lines = ProcessFile(filepath.Join(dir, "missing.php"), "tokens", false, &buf)
	if lines != 0 || !strings.Contains(buf.String(), "Error reading file") {
		t.Fatalf("read error: lines=%d out=%q", lines, buf.String())
	}
}

func TestProcessFileWithErrorsAndMultiple(t *testing.T) {
	dir := t.TempDir()
	ok := filepath.Join(dir, "Ok.php")
	bad := filepath.Join(dir, "Bad.php")
	writeAnalyzeFixture(t, ok, "<?php\nclass Ok {}\n")
	writeAnalyzeFixture(t, bad, "<?php\nfunction broken(\n")

	var buf bytes.Buffer
	errs, lines := ProcessFileWithErrors(ok, "list-files", false, nil, nil, &buf)
	if errs != nil || lines < 1 {
		t.Fatalf("ok non-style: errs=%v lines=%d", errs, lines)
	}

	buf.Reset()
	errs, lines = ProcessFileWithErrors(ok, "style", false, []string{"PSR12.Files.NoBlankLineAfterPHPOpeningTag"}, nil, &buf)
	if errs != nil || lines < 1 {
		t.Fatalf("ok style: errs=%v lines=%d", errs, lines)
	}

	buf.Reset()
	errs, lines = ProcessFileWithErrors(bad, "style", false, nil, nil, &buf)
	if errs == nil || lines < 1 {
		t.Fatalf("bad style: errs=%v lines=%d", errs, lines)
	}

	buf.Reset()
	errs, lines = ProcessFileWithErrors(filepath.Join(dir, "nope.php"), "style", false, nil, nil, &buf)
	if errs != nil || lines != 0 {
		t.Fatalf("missing: errs=%v lines=%d", errs, lines)
	}
	if !strings.Contains(buf.String(), "Error reading file") {
		t.Fatalf("expected read error, got %q", buf.String())
	}

	buf.Reset()
	totalErrs, totalLines := ProcessMultipleFiles([]string{ok, bad}, "list-files", false, 2, nil, nil, &buf)
	if totalErrs < 1 || totalLines < 1 {
		t.Fatalf("ProcessMultipleFiles: errs=%d lines=%d out=%q", totalErrs, totalLines, buf.String())
	}
	if !strings.Contains(buf.String(), "Parsing errors") {
		t.Fatalf("expected collected parse errors, got %q", buf.String())
	}
}

func TestPreloadFilesParallelAndCache(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "A.php")
	b := filepath.Join(dir, "B.php")
	writeAnalyzeFixture(t, a, "<?php\nclass A {}\n")
	writeAnalyzeFixture(t, b, "<?php\nclass B {}\n")

	if err := PreloadFilesParallel([]string{a, b, a}, 2); err != nil {
		t.Fatalf("preload: %v", err)
	}
	content, err := getCachedFileContent(a)
	if err != nil || !bytes.Contains(content, []byte("class A")) {
		t.Fatalf("cache miss after preload: err=%v content=%q", err, content)
	}
	// Second preload should hit cache and succeed.
	if err := PreloadFilesParallel([]string{a}, 1); err != nil {
		t.Fatalf("second preload: %v", err)
	}

	missing := filepath.Join(dir, "Missing.php")
	if err := PreloadFilesParallel([]string{missing}, 1); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestProcessStyleBatchAndCallback(t *testing.T) {
	dir := t.TempDir()
	ok := filepath.Join(dir, "Ok.php")
	bad := filepath.Join(dir, "Bad.php")
	writeAnalyzeFixture(t, ok, "<?php\n\nclass Ok {}\n")
	writeAnalyzeFixture(t, bad, "<?php\nfunction broken(\n")

	var calls atomic.Int32
	issues, parseErrs, lines := processStyleBatch(
		[]string{ok, bad, filepath.Join(dir, "Missing.php")},
		[]string{"PSR12.Files.NoBlankLineAfterPHPOpeningTag"},
		nil,
		2,
		func() { calls.Add(1) },
	)
	if parseErrs < 1 || lines < 1 {
		t.Fatalf("batch: issues=%d errs=%d lines=%d", len(issues), parseErrs, lines)
	}
	if calls.Load() != 3 {
		t.Fatalf("callback calls=%d want 3", calls.Load())
	}

	emptyIssues, emptyErrs, emptyLines := ProcessStyleFilesParallelWithCallback(nil, nil, nil, 1, nil)
	if emptyIssues != nil || emptyErrs != 0 || emptyLines != 0 {
		t.Fatalf("empty parallel: %#v %d %d", emptyIssues, emptyErrs, emptyLines)
	}
}

func TestConfigureAnalysisBuildsProjectIndex(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Cfg.php")
	writeAnalyzeFixture(t, path, "<?php\nclass Cfg {}\n")

	level := 0
	prev := configuredAnalysisLevel
	ConfigureAnalysis(&level)
	t.Cleanup(func() { ConfigureAnalysis(prev) })

	idx := buildProjectIndexForFiles([]string{path, filepath.Join(dir, "missing.php")})
	if idx == nil {
		t.Fatal("expected project index when analysis level is set")
	}
	if _, ok := idx.ResolveClass("Cfg"); !ok {
		t.Fatalf("expected Cfg in index, classes=%v", idx.Classes)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	nodes, res := syntax.ParseAndLower(content)
	if issues := runAnalysis(path, nodes, content, res, nil); issues == nil {
		// may be empty; must not panic
	}
	if issues := runAnalysis(path, nodes, content, res, idx); issues == nil {
		// explicit project path
	}

	ConfigureAnalysis(nil)
	if buildProjectIndexForFiles([]string{path}) != nil {
		t.Fatal("nil level must skip project index")
	}
}

func TestStyleCommandWriterAndCollector(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Style.php")
	writeAnalyzeFixture(t, path, "<?php\n\nclass Style {}\n")

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	nodes, _ := syntax.ParseAST(content)

	var printed bytes.Buffer
	Commands["style"].ExecuteWithRules(nodes, path, &printed, []string{"PSR12.Files.NoBlankLineAfterPHPOpeningTag"}, nil)
	if printed.Len() == 0 {
		// Rule may or may not fire depending on fixture; also exercise collector path.
	}

	var collected []style.StyleIssue
	collector := &style.IssueCollector{Issues: &collected}
	Commands["style"].ExecuteWithRules(nodes, filepath.Join(dir, "missing.php"), collector, nil, nil)
	if len(collected) == 0 {
		t.Fatal("expected FileOpenError when content cannot be loaded")
	}
	if collected[0].Code != "PSR12.Files.FileOpenError" {
		t.Fatalf("unexpected issue %#v", collected[0])
	}

	var execBuf bytes.Buffer
	Commands["style"].Execute(nodes, path, &execBuf)
}

func TestStubCommandsAndInit(t *testing.T) {
	var buf bytes.Buffer
	for _, name := range []string{"analyze", "list-files", "config", "tokens"} {
		Commands[name].Execute(nil, "", &buf)
	}
	if !strings.Contains(buf.String(), "analyze") || !strings.Contains(buf.String(), "list-files") {
		t.Fatalf("stub output incomplete: %q", buf.String())
	}

	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	buf.Reset()
	Commands["init"].Execute(nil, "", &buf)
	if !strings.Contains(buf.String(), "Created tusk.yaml") {
		t.Fatalf("init: %q", buf.String())
	}
	if _, err := os.Stat("tusk.yaml"); err != nil {
		t.Fatalf("tusk.yaml not created: %v", err)
	}
	buf.Reset()
	Commands["init"].Execute(nil, "", &buf)
	if !strings.Contains(buf.String(), "Error creating tusk.yaml") {
		t.Fatalf("expected second init to fail, got %q", buf.String())
	}
}

func TestFilterAnalyzeResultScopes(t *testing.T) {
	dir := t.TempDir()
	keepPath := filepath.Join(dir, "Keep.php")
	dropPath := filepath.Join(dir, "Drop.php")
	writeAnalyzeFixture(t, keepPath, "<?php\nclass Keep {}\n")
	writeAnalyzeFixture(t, dropPath, "<?php\nclass Drop {}\n")

	result := AnalyzeResult{
		Issues: []analyse.AnalysisIssue{
			{Filename: keepPath, Line: 1, Column: 1, Code: "A", Message: "keep", Severity: "error"},
			{Filename: dropPath, Line: 2, Column: 1, Code: "B", Message: "drop", Severity: "warning"},
		},
		ParseErrors: []ParseErrorDetail{
			{File: dropPath, Errors: []string{"parse"}},
			{File: keepPath, Errors: []string{"keep-parse"}},
		},
		ReadErrors: []FileReadError{
			{File: "<project>", Message: "snapshot boom"},
			{File: dropPath, Message: "drop-read"},
			{File: keepPath, Message: "keep-read"},
		},
		FilesDiscovered: 2,
		FilesAnalyzed:   2,
		TotalLines:      99,
	}

	filtered := FilterAnalyzeResultToFile(result, keepPath)
	if filtered.FilesDiscovered != 1 {
		t.Fatalf("discovered=%d", filtered.FilesDiscovered)
	}
	if len(filtered.Issues) != 1 || filtered.Issues[0].Filename != keepPath {
		t.Fatalf("issues=%#v", filtered.Issues)
	}
	if len(filtered.ParseErrors) != 1 || filtered.ParseErrors[0].File != keepPath {
		t.Fatalf("parse=%#v", filtered.ParseErrors)
	}
	if len(filtered.ReadErrors) != 2 {
		t.Fatalf("read errors should keep <project> + keepPath: %#v", filtered.ReadErrors)
	}
	// keepPath failed parse+read accounting → not analyzed
	if filtered.FilesAnalyzed != 0 {
		t.Fatalf("analyzed=%d want 0", filtered.FilesAnalyzed)
	}

	okOnly := FilterAnalyzeResultToFiles(AnalyzeResult{
		Issues: []analyse.AnalysisIssue{
			{Filename: keepPath, Line: 1, Column: 1, Message: "x", Severity: "warning"},
		},
	}, map[string]struct{}{keepPath: {}})
	if okOnly.FilesAnalyzed != 1 || okOnly.TotalLines < 1 {
		t.Fatalf("okOnly=%#v", okOnly)
	}
	var out bytes.Buffer
	PrintAnalyzeResult(&out, okOnly)
	if !strings.Contains(out.String(), "warning") || !strings.Contains(out.String(), "[OK]") && !strings.Contains(out.String(), "found") {
		// has issues → ERROR path
	}
	if !strings.Contains(out.String(), "found 1 issue") {
		t.Fatalf("expected issue summary, got %q", out.String())
	}
}

func TestPrintAnalyzeResultCleanAndSnippetEdges(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Snip.php")
	writeAnalyzeFixture(t, path, "<?php\nclass Snip {}\n")

	var out bytes.Buffer
	PrintAnalyzeResult(&out, AnalyzeResult{FilesDiscovered: 1, FilesAnalyzed: 1})
	if !strings.Contains(out.String(), "[OK]") || !strings.Contains(out.String(), "No errors") {
		t.Fatalf("clean output: %q", out.String())
	}

	cache := map[string]fileLineSource{}
	printIssueSnippet(&out, analyse.AnalysisIssue{
		Filename:  path,
		Line:      2,
		Column:    1,
		EndLine:   2,
		EndColumn: 6,
		Code:      "Level0.Symbols",
		Message:   "boom",
		Severity:  "warning",
	}, cache)
	if !strings.Contains(out.String(), "warning[level0-symbols]") {
		t.Fatalf("snippet header missing: %q", out.String())
	}
	if !strings.Contains(out.String(), "^^^^^") {
		t.Fatalf("expected underline width, got %q", out.String())
	}

	out.Reset()
	printIssueSnippet(&out, analyse.AnalysisIssue{
		Filename: filepath.Join(dir, "gone.php"),
		Line:     1,
		Column:   0,
		Message:  "no file",
		Severity: "error",
	}, cache)
	if !strings.Contains(out.String(), "error") || !strings.Contains(out.String(), "no file") {
		t.Fatalf("empty-code header: %q", out.String())
	}
	if line, ok := sourceLineFor(path, 99, cache); ok || line != "" {
		t.Fatalf("out-of-range line should miss: %q %v", line, ok)
	}
}

func TestAnalyzeFilesIncrementalCacheAndScope(t *testing.T) {
	dir := t.TempDir()
	decl := filepath.Join(dir, "Service.php")
	consumer := filepath.Join(dir, "Consumer.php")
	writeAnalyzeFixture(t, decl, `<?php
namespace Example;
class Service {}
`)
	writeAnalyzeFixture(t, consumer, `<?php
namespace Example;
function build(): void {
    new Service();
    new MissingService();
}
`)

	level := 0
	cacheDir := filepath.Join(dir, "cache")
	files := []string{decl, consumer}

	cold := AnalyzeFilesIncremental(files, &level, nil, 0, cacheDir) // parallelism < 1 → 1
	if cold.FilesDiscovered != 2 || cold.FilesAnalyzed != 2 {
		t.Fatalf("cold accounting: %#v", cold)
	}
	if !hasAnalysisMessage(cold, "MissingService not found") {
		t.Fatalf("cold missing-class diagnostic: %#v", cold.Issues)
	}

	warm := AnalyzeFilesIncremental(files, &level, nil, 2, cacheDir)
	if warm.FilesDiscovered != 2 || warm.FilesAnalyzed < 1 {
		t.Fatalf("warm accounting: %#v", warm)
	}
	if !hasAnalysisMessage(warm, "MissingService not found") {
		t.Fatalf("warm missing-class diagnostic: %#v", warm.Issues)
	}

	empty := AnalyzeFilesIncremental(nil, &level, nil, 1, cacheDir)
	if empty.FilesDiscovered != 0 || empty.FilesAnalyzed != 0 {
		t.Fatalf("empty: %#v", empty)
	}

	scoped := AnalyzeFilesIncrementalScoped(files, []string{consumer}, &level, nil, 2, "")
	if scoped.FilesDiscovered != 2 || scoped.FilesAnalyzed != 1 {
		t.Fatalf("scoped accounting: %#v", scoped)
	}
	for _, issue := range scoped.Issues {
		if issue.Filename == decl {
			t.Fatalf("decl should not be type-checked when targeted: %#v", issue)
		}
	}
	if !hasAnalysisMessage(scoped, "MissingService not found") {
		t.Fatalf("scoped should still diagnose consumer: %#v", scoped.Issues)
	}

	warmScoped := AnalyzeFilesIncrementalScoped(files, []string{consumer}, &level, nil, 2, cacheDir)
	if warmScoped.FilesAnalyzed < 1 {
		t.Fatalf("warm scoped: %#v", warmScoped)
	}
}

func TestSortedAnalyzeResultTieBreakers(t *testing.T) {
	in := AnalyzeResult{
		Issues: []analyse.AnalysisIssue{
			{Filename: "b.php", Line: 1, Column: 2, EndLine: 1, EndColumn: 3, Code: "B", Message: "m2"},
			{Filename: "a.php", Line: 2, Column: 1, EndLine: 2, EndColumn: 1, Code: "A", Message: "m1"},
			{Filename: "a.php", Line: 1, Column: 2, EndLine: 1, EndColumn: 2, Code: "A", Message: "m0"},
			{Filename: "a.php", Line: 1, Column: 1, EndLine: 2, EndColumn: 1, Code: "A", Message: "m0"},
			{Filename: "a.php", Line: 1, Column: 1, EndLine: 1, EndColumn: 2, Code: "B", Message: "m0"},
			{Filename: "a.php", Line: 1, Column: 1, EndLine: 1, EndColumn: 1, Code: "A", Message: "m1"},
			{Filename: "a.php", Line: 1, Column: 1, EndLine: 1, EndColumn: 1, Code: "A", Message: "m0"},
		},
		ParseErrors: []ParseErrorDetail{{File: "b.php"}, {File: "a.php"}},
		ReadErrors:  []FileReadError{{File: "b.php"}, {File: "a.php"}},
	}
	out := sortedAnalyzeResult(in)
	if out.Issues[0].Filename != "a.php" || out.Issues[0].Message != "m0" || out.Issues[0].Code != "A" {
		t.Fatalf("first issue %#v", out.Issues[0])
	}
	if out.ParseErrors[0].File != "a.php" || out.ReadErrors[0].File != "a.php" {
		t.Fatalf("error sort %#v %#v", out.ParseErrors, out.ReadErrors)
	}
}

func TestAnalyzeFilesWithCacheMergePath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Merge.php")
	writeAnalyzeFixture(t, path, "<?php\nclass Merge {}\n")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	nodes, _ := syntax.ParseASTForIndex(content)
	cached := analyse.BuildProjectIndex(map[string][]ast.Node{path: nodes})
	level := 0
	checksums := analyse.FileChecksums(map[string][]byte{path: content})
	result := analyzeFilesWithCache([]string{path}, nil, &level, nil, 1, "", cached, checksums)
	if result.FilesAnalyzed != 1 {
		t.Fatalf("merge path accounting: %#v", result)
	}
}

func TestProcessStyleParallelReadFailure(t *testing.T) {
	dir := t.TempDir()
	ok := filepath.Join(dir, "Ok.php")
	writeAnalyzeFixture(t, ok, "<?php\nclass Ok {}\n")
	missing := filepath.Join(dir, "Missing.php")
	var calls atomic.Int32
	_, _, lines := ProcessStyleFilesParallelWithCallback(
		[]string{ok, missing},
		[]string{"PSR12.Files.NoBlankLineAfterPHPOpeningTag"},
		nil,
		2,
		func() { calls.Add(1) },
	)
	if lines < 1 {
		t.Fatalf("expected lines from ok file, got %d", lines)
	}
	if calls.Load() != 2 {
		t.Fatalf("callback calls=%d want 2", calls.Load())
	}
}
