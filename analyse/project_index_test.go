package analyse

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/lexer"
	"github.com/ayanozturk/go-php-parser/parser"
	"testing"
)

func parsePHPForProjectIndex(t *testing.T, php string) []ast.Node {
	t.Helper()
	p := parser.New(lexer.New(php), false)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	return nodes
}

// TestBuildProjectIndexDuplicateClassResolutionIsDeterministic guards
// against a regression where BuildProjectIndex iterated its input map in
// Go's randomized map-iteration order. When two files declare the same
// class name (a real occurrence in large corpora, e.g. duplicated test
// fixtures or namespace collisions), which file's declaration ends up
// canonical - and which are recorded as duplicates - depended on that
// random order, so every diagnostic computed relative to the class (and
// relative to its last-registered members: methods/properties/constants
// are registered per file with "last processed wins", independent of
// addClass's "first file wins" rule for the class's own metadata) could
// vary between otherwise-identical runs over the same corpus.
// BuildProjectIndex now processes files in sorted filename order, so both
// the class-metadata winner and the member winner must be stable across
// many repeated builds regardless of the input map's iteration order.
func TestBuildProjectIndexDuplicateClassResolutionIsDeterministic(t *testing.T) {
	parsed := map[string][]ast.Node{
		"z_last.php": parsePHPForProjectIndex(t, `<?php
class Dup {
    public function shared(): FromZ {}
}
`),
		"a_first.php": parsePHPForProjectIndex(t, `<?php
class Dup {
    public function shared(): FromA {}
}
`),
		"m_middle.php": parsePHPForProjectIndex(t, `<?php
class Dup {
    public function shared(): FromM {}
}
`),
	}

	for i := 0; i < 25; i++ {
		idx := BuildProjectIndex(parsed)

		// addClass keeps the first file processed in sorted order
		// (a_first.php) as canonical and records the rest as duplicates.
		if len(idx.Duplicates) != 2 {
			t.Fatalf("iteration %d: expected exactly 2 duplicate registrations, got %#v", i, idx.Duplicates)
		}
		if idx.Duplicates[0].File != "m_middle.php" || idx.Duplicates[1].File != "z_last.php" {
			t.Fatalf("iteration %d: expected duplicates in sorted-filename order [m_middle.php, z_last.php], got %#v", i, idx.Duplicates)
		}

		// The "shared" method name is registered unconditionally for every
		// file sharing the class name, with the last file processed in
		// sorted order (z_last.php) winning the overwrite.
		method, ok := idx.Methods[indexKey("Dup")]["shared"]
		if !ok {
			t.Fatalf("iteration %d: expected method %q to be registered", i, "shared")
		}
		if method.ReturnType != "FromZ" {
			t.Fatalf("iteration %d: expected the alphabetically-last file (z_last.php) to win method registration with return type FromZ, got %q", i, method.ReturnType)
		}
	}
}
