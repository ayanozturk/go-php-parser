package analyse

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
)

func TestASTCacheStoreLoadRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	cm := NewASTCacheManager(tmpDir)
	if cm.cacheDir != filepath.Join(tmpDir, "ast") {
		t.Fatalf("cache dir = %q, want .../ast", cm.cacheDir)
	}

	nodes := []ast.Node{
		&ast.ExpressionStmt{
			Expr: &ast.IntegerLiteral{Value: 1},
		},
	}
	const path = "src/example.php"
	const checksum = "abc123"

	if err := cm.StoreAST(path, nodes, checksum); err != nil {
		t.Fatalf("StoreAST: %v", err)
	}
	cachePath := filepath.Join(cm.cacheDir, hashPath(path)+".gob")
	if _, err := os.Stat(cachePath); err != nil {
		t.Fatalf("expected cache file at %s: %v", cachePath, err)
	}

	loaded, ok := cm.LoadAST(path, checksum)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if len(loaded) != 1 {
		t.Fatalf("loaded %d nodes, want 1", len(loaded))
	}
	stmt, ok := loaded[0].(*ast.ExpressionStmt)
	if !ok {
		t.Fatalf("unexpected node type %T", loaded[0])
	}
	lit, ok := stmt.Expr.(*ast.IntegerLiteral)
	if !ok || lit.Value != 1 {
		t.Fatalf("unexpected loaded expr %#v", stmt.Expr)
	}

	if _, ok := cm.LoadAST(path, "other"); ok {
		t.Fatal("checksum mismatch should miss")
	}
	if _, ok := cm.LoadAST("missing.php", checksum); ok {
		t.Fatal("missing path should miss")
	}
}

func TestHashPathDeterministicAndSafe(t *testing.T) {
	a := hashPath("App\\Service.php")
	b := hashPath("App\\Service.php")
	if a == "" || a != b {
		t.Fatalf("hashPath not deterministic: %q vs %q", a, b)
	}
	if a == hashPath("App/Service.php") {
		t.Fatal("distinct paths should not collide for this fixture")
	}
}
