package analyse

import (
	"os"
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
)

// retainHostASTFromEnv reports whether PHP_PARSER_RETAIN_HOST_AST requests
// keeping host ingest AST through snapshot→rules (idea-A A/B). Unset/false
// keeps the default R2.5c releasing constructor.
func retainHostASTFromEnv() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("PHP_PARSER_RETAIN_HOST_AST")))
	return v == "1" || v == "true" || v == "yes"
}

// NewSemanticSnapshotWithIndexMaybeReleasing is NewSemanticSnapshotWithIndex
// when PHP_PARSER_RETAIN_HOST_AST is truthy (1/true/yes); otherwise it uses
// NewSemanticSnapshotWithIndexReleasingParsed (production default).
func NewSemanticSnapshotWithIndexMaybeReleasing(idx *ProjectIndex, parsed map[string][]ast.Node, facts []SemanticFact, targets []string) (*SemanticSnapshot, error) {
	if retainHostASTFromEnv() {
		return NewSemanticSnapshotWithIndex(idx, parsed, facts, targets)
	}
	return NewSemanticSnapshotWithIndexReleasingParsed(idx, parsed, facts, targets)
}
