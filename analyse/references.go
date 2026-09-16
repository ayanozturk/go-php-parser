package analyse

import (
	"github.com/ayanozturk/go-php-parser/syntax"
)

// IndexSyntaxFileForReferences binds a full syntax ParseResult into g for
// project-wide rename/refs. Callers must pass a reference-complete parse
// (syntax.Parse), not ParseForIndex — skip-bodies trees omit use sites (R3).
//
// This is the dual-lex cutover vertical slice: one parse → BindSyntaxResult →
// ProjectUsageGraph. Classic AST ProjectIndex is unchanged.
func IndexSyntaxFileForReferences(g *ProjectUsageGraph, uri string, res *syntax.ParseResult) {
	if g == nil {
		return
	}
	graph := BindSyntaxResult(uri, res)
	g.PutFile(uri, graph.Uses)
}

// ReferencesAt returns project-wide matching uses for the symbol under
// byteOffset in uri. The active file is re-bound from res (no re-parse of other
// files); other URIs must already be present in g via IndexSyntaxFileForReferences.
//
// When no NameUse covers offset, or the needle has no Resolved identity, the
// result is nil (not an error) — same contract as FindMatching on empty input.
func ReferencesAt(g *ProjectUsageGraph, uri string, res *syntax.ParseResult, byteOffset int) []NameUse {
	if g == nil || res == nil {
		return nil
	}
	// Refresh the active file from the caller's parse so overlay edits are visible
	// without requiring a separate PutFile from the editor host.
	IndexSyntaxFileForReferences(g, uri, res)
	needle, ok := UseAtOffset(g.UsesForURI(uri), byteOffset)
	if !ok {
		return nil
	}
	return g.FindMatching(needle)
}

// RenameTargetsAt is the rename query surface: identical hits to ReferencesAt.
// Workspace-edit construction stays in the editor host (PHP Strom) until that
// adapter is wired; the parser exposes symbol identity + spans only.
func RenameTargetsAt(g *ProjectUsageGraph, uri string, res *syntax.ParseResult, byteOffset int) []NameUse {
	return ReferencesAt(g, uri, res, byteOffset)
}
