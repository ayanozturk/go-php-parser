package analyse

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

// SideEffectsRule implements PSR1.Files.SideEffects
// Ensures files either declare symbols OR cause side-effects, but not both
type SideEffectsRule struct{}

// CheckIssuesFromCST performs the same PSR1.Files.SideEffects analysis
// directly against the syntax CST (see syntax.Parse/syntax.NamespaceBody),
// without lowering to []ast.Node first. Mirrors CheckIssuesWithSource's
// isSideEffectNode/isDeclaration logic node-kind-by-node-kind.
func (r *SideEffectsRule) CheckIssuesFromCST(filename string, content []byte) []AnalysisIssue {
	return r.checkIssuesFromParsedCST(filename, syntax.Parse(content))
}

func (r *SideEffectsRule) checkIssuesFromParsedCST(filename string, res *syntax.ParseResult) []AnalysisIssue {
	if res == nil || res.File == nil || res.File.Root == nil {
		return nil
	}
	top := flattenTopLevelForSideEffects(res.File.Root)

	hasSideEffects := false
	hasDeclarations := false
	for _, n := range top {
		if isSideEffectCSTNode(n) {
			hasSideEffects = true
		}
		if isDeclarationCSTNode(n) {
			hasDeclarations = true
		}
	}

	if hasSideEffects && hasDeclarations {
		return []AnalysisIssue{{
			Filename: filename,
			Line:     1,
			Column:   1,
			Code:     "PSR1.Files.SideEffects",
			Message:  "A file should declare new symbols or cause side-effects, but not both",
		}}
	}
	return nil
}

// flattenTopLevelForSideEffects returns the file's top-level CST nodes,
// unwrapping namespace declarations into their body nodes (braced or
// unbraced) to mirror how ast.NamespaceNode.Body is treated by
// isSideEffectNode/isDeclaration.
func flattenTopLevelForSideEffects(root *syntax.RedNode) []*syntax.RedNode {
	if root == nil {
		return nil
	}
	children := root.Children()
	var out []*syntax.RedNode
	for i := 0; i < len(children); i++ {
		c := children[i]
		if c.Kind() == syntax.KindNamespaceDecl {
			body, consumed := syntax.NamespaceBody(c, children, i)
			out = append(out, body...)
			i += consumed
			continue
		}
		out = append(out, c)
	}
	return out
}

// isSideEffectCSTNode mirrors isSideEffectNode's ast.Node type switch.
func isSideEffectCSTNode(n *syntax.RedNode) bool {
	if n == nil {
		return false
	}
	switch n.Kind() {
	case syntax.KindToken, syntax.KindTokenList, syntax.KindError, syntax.KindMissing:
		return false
	case syntax.KindUseDecl, syntax.KindAttributeList,
		syntax.KindClassDecl, syntax.KindFunctionDecl, syntax.KindInterfaceDecl,
		syntax.KindTraitDecl, syntax.KindConstDecl, syntax.KindEnumDecl:
		return false
	case syntax.KindDeclareStmt:
		// Mirrors the *ast.DeclareNode case: a braced body with at least one
		// lowerable statement is unconditionally a side effect (the original
		// code never inspects the body's contents); a bodyless
		// `declare(...);` or an empty `declare(...) {}` is not.
		bodyList := n.FirstChildOfKind(syntax.KindStatementList)
		return bodyList != nil && len(syntax.StatementBodyList(bodyList)) > 0
	case syntax.KindExpressionStmt:
		return isSideEffectExprCST(n)
	default:
		return true
	}
}

// isSideEffectExprCST mirrors isSideEffectExpr's ast.Node type switch.
func isSideEffectExprCST(stmt *syntax.RedNode) bool {
	expr := syntax.ExpressionStmtExpr(stmt)
	if expr == nil {
		return false
	}
	switch expr.Kind() {
	case syntax.KindAssignExpr:
		return false
	case syntax.KindUnqualifiedName, syntax.KindQualifiedName,
		syntax.KindFullyQualifiedName, syntax.KindRelativeName, syntax.KindName:
		// The current parser can surface top-level `use Foo\Bar;` as a bare
		// name expression statement. Treat that malformed import shape as
		// non-side-effecting, matching the ast.IdentifierNode case.
		return false
	default:
		return true
	}
}

// isDeclarationCSTNode mirrors isDeclaration's ast.Node type switch (namespace
// unwrapping already happened in flattenTopLevelForSideEffects).
func isDeclarationCSTNode(n *syntax.RedNode) bool {
	if n == nil {
		return false
	}
	switch n.Kind() {
	case syntax.KindClassDecl, syntax.KindFunctionDecl, syntax.KindInterfaceDecl,
		syntax.KindTraitDecl, syntax.KindConstDecl, syntax.KindEnumDecl:
		return true
	}
	return false
}

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
	case *ast.AttributeNode:
		return false
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

// runRegisteredSideEffectsRule is the body of the registered
// "PSR1.Files.SideEffects" callback, split out so tests can invoke exactly
// what production runs without depending on global registry state.
func runRegisteredSideEffectsRule(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
	rule := &SideEffectsRule{}
	if len(ctx.Content) > 0 {
		res := sharedParseResult(ctx, ctx.Content)
		return rule.checkIssuesFromParsedCST(filename, res)
	}
	return rule.CheckIssues(nodes, filename)
}

func init() {
	RegisterAnalysisRuleWithContext("PSR1.Files.SideEffects", runRegisteredSideEffectsRule)
}
