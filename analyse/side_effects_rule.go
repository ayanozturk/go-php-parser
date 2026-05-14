package analyse

import (
	"go-phpcs/ast"
)

// SideEffectsRule implements PSR1.Files.SideEffects
// Ensures files either declare symbols OR cause side-effects, but not both
type SideEffectsRule struct{}

// CheckIssuesWithSource performs analysis given explicit source content (used by tests)
func (r *SideEffectsRule) CheckIssuesWithSource(filename string, content []byte, nodes []ast.Node) []AnalysisIssue {
	var issues []AnalysisIssue

	// Analyze top-level AST nodes for side effects and declarations.
	hasSideEffects := r.hasSideEffects(nodes)
	hasDeclarations := r.hasDeclarations(nodes)

	if hasSideEffects && hasDeclarations {
		issues = append(issues, AnalysisIssue{
			Filename: filename,
			Line:     1,
			Column:   1,
			Code:     "PSR1.Files.SideEffects",
			Message:  "A file should declare new symbols or cause side-effects, but not both",
		})
	}

	return issues
}

// CheckIssues analyzes the entire file to detect both symbol declarations and side effects
func (r *SideEffectsRule) CheckIssues(nodes []ast.Node, filename string) []AnalysisIssue {
	return r.CheckIssuesWithSource(filename, nil, nodes)
}

// hasSideEffects checks whether the file has executable top-level statements.
// Declarations, namespace/import bookkeeping, and comments/docblocks are not side effects.
func (r *SideEffectsRule) hasSideEffects(nodes []ast.Node) bool {
	for _, node := range nodes {
		if r.isSideEffectNode(node) {
			return true
		}
	}
	return false
}

func (r *SideEffectsRule) isSideEffectNode(node ast.Node) bool {
	switch n := node.(type) {
	case nil:
		return false
	case *ast.NamespaceNode:
		return r.hasSideEffects(n.Body)
	case *ast.UseNode:
		return false
	case *ast.DeclareNode:
		return r.isSideEffectNode(n.Body)
	case *ast.CommentNode:
		return false
	case *ast.PHPDocNode:
		return false
	case *ast.ExpressionStmt:
		return r.isSideEffectExpr(n.Expr)
	case *ast.ClassNode, *ast.FunctionNode, *ast.InterfaceNode, *ast.TraitNode, *ast.ConstantNode, *ast.EnumNode:
		return false
	default:
		return true
	}
}

func (r *SideEffectsRule) isSideEffectExpr(node ast.Node) bool {
	switch node.(type) {
	case nil:
		return false
	case *ast.AssignmentNode:
		return false
	case *ast.IdentifierNode:
		// The current parser can surface top-level `use Foo\Bar;` as an Identifier
		// expression statement. Treat that malformed import shape as non-side-effecting.
		return false
	default:
		return true
	}
}

// hasDeclarations checks if the file contains symbol declarations
func (r *SideEffectsRule) hasDeclarations(nodes []ast.Node) bool {
	for _, node := range nodes {
		if r.isDeclaration(node) {
			return true
		}
	}
	return false
}

// isDeclaration checks if a node represents a symbol declaration
func (r *SideEffectsRule) isDeclaration(node ast.Node) bool {
	switch n := node.(type) {
	case *ast.ClassNode:
		return true
	case *ast.FunctionNode:
		return true
	case *ast.InterfaceNode:
		return true
	case *ast.TraitNode:
		return true
	case *ast.ConstantNode:
		return true
	case *ast.EnumNode:
		return true
	case *ast.NamespaceNode:
		// Check if namespace contains declarations
		for _, bodyNode := range n.Body {
			if r.isDeclaration(bodyNode) {
				return true
			}
		}
	}

	return false
}

func init() {
	RegisterAnalysisRule("PSR1.Files.SideEffects", func(filename string, nodes []ast.Node) []AnalysisIssue {
		rule := &SideEffectsRule{}
		return rule.CheckIssues(nodes, filename)
	})
}
