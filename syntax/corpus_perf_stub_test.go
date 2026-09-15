package syntax

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ayanozturk/go-php-parser/lexer"
)

// TestSyntaxCorpusIdentityGate is a lightweight corpus stub for R5.
// Full Symfony/WordPress corpora are heavy; when SYNTAX_CORPUS_DIR is set,
// every *.php file must identity-print. Without the env var, a small checked-in
// fixture set still exercises the gate so phase completion is not vacuously claimed.
//
// Pin-fetch (if needed):
//
//	go run ./cmd/fetch-test-projects --only symfony,wordpress-develop
//
// Full identity gate via test:
//
//	SYNTAX_CORPUS_DIR=test_projects/symfony go test ./syntax -run TestSyntaxCorpusIdentityGate -count=1 -timeout 30m
//	SYNTAX_CORPUS_DIR=test_projects/wordpress-develop go test ./syntax -run TestSyntaxCorpusIdentityGate -count=1 -timeout 30m
//
// Metrics JSON report:
//
//	go run ./cmd/syntax-metrics --root test_projects/symfony --json
//	go run ./cmd/syntax-metrics --root test_projects/wordpress-develop --json
func TestSyntaxCorpusIdentityGate(t *testing.T) {
	dir := os.Getenv("SYNTAX_CORPUS_DIR")
	var files []string
	if dir != "" {
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if filepath.Ext(path) == ".php" {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk corpus: %v", err)
		}
		if len(files) == 0 {
			t.Fatalf("SYNTAX_CORPUS_DIR=%q has no .php files", dir)
		}
	} else {
		// Built-in mini-corpus covering gold surfaces (attrs, DNF, heredoc, control-flow, expressions).
		tmp := t.TempDir()
		fixtures := map[string]string{
			"attr.php":        "<?php\n#[Attr]\nclass C {}\n",
			"dnf.php":         "<?php\nfunction f((Foo&Bar)|null $x): void {}\n",
			"heredoc.php":     "<?php\n$a = <<<EOT\nhello $x\nEOT;\n",
			"control.php":     "<?php\nif ($a) { echo $a; } while ($i) { break; }\n",
			"names.php":       "<?php\nuse Foo\\Bar as Baz;\nnamespace App;\n",
			"expressions.php": "<?php\n$a = 1 + 2 * 3;\n$b = $obj->prop->meth($x, $y)[0];\n$c = ['k' => 1, 2];\n$d = -$a ?? true;\n$e = ($a && $b) || !$c;\n",
			"closures.php":    "<?php\n$f = function($x) use ($y) { return $x; };\n$g = static fn($a): int => $a + 1;\n$o = new class($x) { public function m() {}\n};\n",
			"short_echo.php":  `<p><?= $name; ?></p>` + "\n",
		}
		for name, src := range fixtures {
			path := filepath.Join(tmp, name)
			if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}
			files = append(files, path)
		}
	}

	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		res := Parse(src)
		got := Print(res.File.Root)
		if got != string(src) {
			t.Fatalf("corpus identity failed for %s\nwant %q\ngot  %q", path, src, got)
		}
	}
}

// TestSyntaxParsePerfSmoke is a harness stub for R5 performance gates.
// It records allocs/op, ns/op, tokens/KB, and bytes/node on a fixed fixture;
// full corpus gates belong in cmd/syntax-metrics runs. Fails only on pathological
// blow-ups so CI stays useful without claiming Symfony-scale budgets here.
//
// Pin-fetch (if needed):
//
//	go run ./cmd/fetch-test-projects --only symfony,wordpress-develop
//
// Full identity gate via test:
//
//	SYNTAX_CORPUS_DIR=test_projects/symfony go test ./syntax -run TestSyntaxCorpusIdentityGate -count=1 -timeout 30m
//	SYNTAX_CORPUS_DIR=test_projects/wordpress-develop go test ./syntax -run TestSyntaxCorpusIdentityGate -count=1 -timeout 30m
//
// Metrics JSON report:
//
//	go run ./cmd/syntax-metrics --root test_projects/symfony --json
//	go run ./cmd/syntax-metrics --root test_projects/wordpress-develop --json
func TestSyntaxParsePerfSmoke(t *testing.T) {
	src := []byte(`<?php
#[\App\Attr(x: 1)]
function f((Foo&Bar)|null $a): int|false {
  $s = "hello $a";
  $h = <<<EOT
hi $a
EOT;
  if ($a) { return 1; }
  foreach ([1, 2] as $x) { echo $x; }
  return false;
}
`)
	toks := lexer.LexAll(src)
	const iters = 200
	start := time.Now()
	var identityFail bool
	var lastNodes int
	allocs := testing.AllocsPerRun(iters, func() {
		res := Parse(src)
		if Print(res.File.Root) != string(src) {
			identityFail = true
		}
		lastNodes = countRedNodes(res.File.Root)
	})
	elapsed := time.Since(start)
	if identityFail {
		t.Fatal("identity failed inside perf smoke")
	}
	tokensPerKB := float64(len(toks)) / (float64(len(src)) / 1024.0)
	bytesPerNode := float64(len(src)) / float64(lastNodes)
	t.Logf("syntax parse+print: allocs/op=%.0f elapsed=%s for %d iters (%d bytes, %d tokens, %.1f tokens/KB, %.1f bytes/node)",
		allocs, elapsed, iters, len(src), len(toks), tokensPerKB, bytesPerNode)
	// Pathological guard only — not a Symfony/WordPress budget.
	if allocs > 50000 {
		t.Fatalf("pathological allocs/op=%.0f (stub gate)", allocs)
	}
	if elapsed > 30*time.Second {
		t.Fatalf("pathological elapsed=%s (stub gate)", elapsed)
	}
}

func countRedNodes(n *RedNode) int {
	if n == nil {
		return 0
	}
	count := 1
	for _, c := range n.Children() {
		count += countRedNodes(c)
	}
	return count
}
