package analyse

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/ayanozturk/go-php-parser/ast"
)

func init() {
	// Register all AST node types with gob encoder
	registerASTTypes()
}

func registerASTTypes() {
	// Core types
	gob.Register(&ast.BlockNode{})
	gob.Register(&ast.Identifier{})
	gob.Register(&ast.VariableNode{})
	gob.Register(&ast.VariableVariableNode{})
	gob.Register(&ast.StringLiteral{})
	gob.Register(&ast.InterpolatedStringLiteral{})
	gob.Register(&ast.IntegerLiteral{})
	gob.Register(&ast.FloatLiteral{})
	gob.Register(&ast.BooleanLiteral{})
	gob.Register(&ast.NullLiteral{})
	gob.Register(&ast.AssignmentNode{})
	gob.Register(&ast.ReturnNode{})
	gob.Register(&ast.ExpressionStmt{})
	gob.Register(&ast.BinaryExpr{})
	gob.Register(&ast.IfNode{})
	gob.Register(&ast.ElseIfNode{})
	gob.Register(&ast.ElseNode{})
	gob.Register(&ast.WhileNode{})
	gob.Register(&ast.DoWhileNode{})
	gob.Register(&ast.FunctionDecl{})
	gob.Register(&ast.Variable{})
	gob.Register(&ast.FunctionCall{})
	gob.Register(&ast.IdentifierNode{})
	gob.Register(&ast.FirstClassCallableNode{})
	gob.Register(&ast.BooleanNode{})
	gob.Register(&ast.NullNode{})
	gob.Register(&ast.ConcatNode{})
	gob.Register(&ast.AttributeNode{})
	gob.Register(&ast.NamespaceNode{})
	gob.Register(&ast.UseNode{})
	gob.Register(&ast.MatchNode{})
	gob.Register(&ast.MatchArmNode{})
	gob.Register(&ast.ArrowFunctionNode{})
	gob.Register(&ast.TypeCastNode{})
	gob.Register(&ast.YieldNode{})
	gob.Register(&ast.HeredocNode{})
	gob.Register(&ast.TernaryExpr{})
	gob.Register(&ast.PropertyFetchNode{})
	gob.Register(&ast.ForeachNode{})
	gob.Register(&ast.ThrowNode{})
	gob.Register(&ast.GotoNode{})
	gob.Register(&ast.LabelNode{})
	gob.Register(&ast.Position{})
}

// ASTCacheManager handles per-file AST serialization via gob.
// Stores parsed AST nodes to disk; deserializes on cache hit.
type ASTCacheManager struct {
	cacheDir string
	mu       sync.Mutex
}

// NewASTCacheManager creates AST cache manager.
func NewASTCacheManager(cacheDir string) *ASTCacheManager {
	return &ASTCacheManager{cacheDir: filepath.Join(cacheDir, "ast")}
}

// StoreAST serializes AST nodes for a file.
func (cm *ASTCacheManager) StoreAST(filePath string, nodes []ast.Node, checksum string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if err := os.MkdirAll(cm.cacheDir, 0755); err != nil {
		return err
	}

	// Map filename to cache key (use hash to avoid filesystem issues)
	cacheKey := fmt.Sprintf("%s.gob", hashPath(filePath))
	cachePath := filepath.Join(cm.cacheDir, cacheKey)

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(nodes); err != nil {
		return err
	}

	// Store with checksum as metadata
	metadata := map[string]string{"checksum": checksum}
	if err := enc.Encode(metadata); err != nil {
		return err
	}

	return os.WriteFile(cachePath, buf.Bytes(), 0644)
}

// LoadAST deserializes AST nodes for a file if checksum matches.
func (cm *ASTCacheManager) LoadAST(filePath string, checksum string) ([]ast.Node, bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cacheKey := fmt.Sprintf("%s.gob", hashPath(filePath))
	cachePath := filepath.Join(cm.cacheDir, cacheKey)

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, false
	}

	buf := bytes.NewReader(data)
	dec := gob.NewDecoder(buf)

	var nodes []ast.Node
	if err := dec.Decode(&nodes); err != nil {
		return nil, false
	}

	var metadata map[string]string
	if err := dec.Decode(&metadata); err != nil {
		return nil, false
	}

	// Verify checksum matches
	if metadata["checksum"] != checksum {
		return nil, false
	}

	return nodes, true
}

// hashPath creates a safe filename from filepath.
func hashPath(filePath string) string {
	h := 0
	for _, c := range filePath {
		h = h*31 + int(c)
	}
	return fmt.Sprintf("%x", h)
}
