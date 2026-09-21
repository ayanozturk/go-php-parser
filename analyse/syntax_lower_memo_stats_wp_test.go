package analyse

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

// TestSyntaxLowerMemoStatsWPSample ranks memoLower entrypoints on a WP slice.
// Skips when test_projects/wordpress-develop is absent.
func TestSyntaxLowerMemoStatsWPSample(t *testing.T) {
	root := filepath.Join("..", "test_projects", "wordpress-develop")
	if st, err := os.Stat(root); err != nil || !st.IsDir() {
		t.Skip("wordpress-develop corpus not present")
	}

	EnableSyntaxLowerMemoStats(true)
	defer EnableSyntaxLowerMemoStats(false)
	ResetSyntaxLowerMemoStats()

	var files []string
	_ = filepath.Walk(filepath.Join(root, "src"), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Ext(path) != ".php" {
			return nil
		}
		files = append(files, path)
		if len(files) >= 200 {
			return filepath.SkipAll
		}
		return nil
	})
	if len(files) == 0 {
		t.Fatal("no php files under wordpress-develop/src")
	}

	level := 6
	parsed := map[string][]ast.Node{}
	contents := map[string][]byte{}
	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		nodes, _ := syntax.ParseAST(src)
		rel, _ := filepath.Rel(root, path)
		parsed[rel] = nodes
		contents[rel] = src
	}
	project := BuildProjectIndex(parsed)
	for rel, src := range contents {
		ctx := &AnalysisContext{
			Content:       src,
			AnalysisLevel: &level,
			Resolver:      project,
		}
		ensureSharedFileDiagnosticsFromCST(rel, src, parsed[rel], ctx)
	}

	snap := SnapshotSyntaxLowerMemoStats()
	t.Log("\n" + FormatSyntaxLowerMemoStats(snap))
}
