# Callable semantics: matched engine resources

Date: 2026-10-09. Baseline: `4b0bde77`; candidate: the commit containing this
report. macOS arm64, Go 1.23.12, `GOWORK=off`, `GOMAXPROCS=3`, three analysis
workers, level 8. The same pinned PSL selection contains 1,775 reporting source
files and 25 Revolt index dependencies (139,123 reporting lines), with zero read
or parse errors. Each measured sample starts a fresh process with fresh in-memory
caches; filesystem caches are warm. Builds and correctness runs are excluded.

The final candidate batch alternates before/after order across ten pairs. `/usr/bin/time -l`
measures peak RSS; Python `perf_counter` measures parent-observed wall time.
Diagnostics are sorted for canonical result hashes; duplicate entries remain.
Every sample has complete matching file accounting, 1,283 baseline diagnostics
and 1,203 candidate diagnostics, and one stable result hash per variant.
The intended 80-entry diagnostic reduction is reviewed separately in
[corpus quality](../reviewed-corpus-quality.md).

| Variant | Mean seconds | CV | Mean peak RSS (MiB) |
| --- | ---: | ---: | ---: |
| Before | 0.610 | 10.70% | 77.49 |
| After | 0.601 | 1.15% | 80.17 |

**This comparison is not an accepted baseline:** the before batch exceeds the
required CV <=5%, even though the candidate is stable. Two earlier exploratory
batches also exceeded the stability threshold; no individual samples were removed.
No performance improvement or proven absence of regression is claimed, and no
existing best accepted baseline is replaced. The host had unrelated background
Docker activity; that activity was left running. Repeat on a stable host before
closing the resource gate. Raw samples and canonical hashes are retained in
[the JSON report](2026-10-09-callable-semantics.json).

The split-lines cache now keeps source backing arrays alive until deletion or
eviction. These complete analysis runs include that change; they do not establish
long-lived language-server memory behavior or the separate CLI resource gate.

## Reproduction

Build this neutral workload helper once from each revision with
`GOWORK=off GOTOOLCHAIN=go1.23.12 go build -o /tmp/callable-engine-<variant>
/tmp/psl-engine.go`. Run both executables from the same corpus checkout, with
`GOMAXPROCS=3`, alternating order for ten fresh-process pairs and retaining
file/error counts, diagnostics and peak RSS for every sample. Require CV <=5%
without selecting or discarding individual samples.

```go
package main

import (
	"encoding/json"
	"github.com/ayanozturk/go-php-parser/command"
	"os"
	"path/filepath"
)

func main() {
	var files, targets []string
	roots, _ := filepath.Glob("test_projects/psl/packages/*/src")
	for _, root := range append(roots, "test_projects/psl/vendor/revolt") {
		filepath.WalkDir(root, func(p string, d os.DirEntry, e error) error {
			if e == nil && !d.IsDir() && filepath.Ext(p) == ".php" {
				files = append(files, p)
				if root != "test_projects/psl/vendor/revolt" {
					targets = append(targets, p)
				}
			}
			return e
		})
	}
	level := 8
	r := command.AnalyzeFilesIncrementalScoped(files, targets, &level, nil, 3, "")
	json.NewEncoder(os.Stdout).Encode(r)
}
```
