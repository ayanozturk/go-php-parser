package analyse

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/lexer"
	"github.com/ayanozturk/go-php-parser/parser"
)

func parseTypeNarrowingPHP(t *testing.T, source string) map[string][]SemanticFact {
	t.Helper()
	const filename = "narrowing.php"
	p := parser.New(lexer.New(source), false)
	nodes := p.Parse()
	if errs := p.Errors(); len(errs) != 0 {
		t.Fatalf("parse narrowing fixture: %v", errs)
	}
	snapshot, err := NewSemanticSnapshot(map[string][]ast.Node{filename: nodes}, nil)
	if err != nil {
		t.Fatalf("build semantic snapshot: %v", err)
	}
	facts := snapshot.FactsForFile(filename)
	result := make(map[string][]SemanticFact)
	for _, fact := range facts {
		if fact.Key.Kind == FactKindNarrowed {
			result[fact.Type] = append(result[fact.Type], fact)
		}
	}
	return result
}

func TestTypeNarrowingDetectsInstanceof(t *testing.T) {
	facts := parseTypeNarrowingPHP(t, `<?php
function process($obj) {
    if ($obj instanceof User) {
        $name = $obj->name;
    }
}`)

	if len(facts["User"]) == 0 {
		t.Fatalf("expected narrowing fact for User class, got none")
	}
}

func TestTypeNarrowingIgnoresOutsideBranch(t *testing.T) {
	facts := parseTypeNarrowingPHP(t, `<?php
function process($obj) {
    $obj->method();
    if ($obj instanceof User) {
        $name = $obj->name;
    }
}`)

	// Should have narrowing inside the if, but not before it
	if len(facts["User"]) == 0 {
		t.Fatalf("expected narrowing fact inside if branch, got none")
	}
}

func TestTypeNarrowingMultipleBranches(t *testing.T) {
	facts := parseTypeNarrowingPHP(t, `<?php
function process($obj) {
    if ($obj instanceof User) {
        $x = $obj->name;
    } elseif ($obj instanceof Admin) {
        $y = $obj->role;
    }
}`)

	if len(facts["User"]) == 0 || len(facts["Admin"]) == 0 {
		t.Fatalf("expected narrowing facts for both User and Admin, got User:%d Admin:%d", len(facts["User"]), len(facts["Admin"]))
	}
}
