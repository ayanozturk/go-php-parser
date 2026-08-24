package analyse

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/lexer"
	"github.com/ayanozturk/go-php-parser/parser"
	"strings"
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

func TestProjectIndexAssignsStableIDsAndDeclarationSpans(t *testing.T) {
	const filename = "src/Model.php"
	source := `<?php
namespace App;

class BaseModel {
    protected string $name;
    public const KIND = 'base';
    public function label(int $id): string {}
}

class ChildModel extends BaseModel {}

function load_model(int $id): ChildModel {}

interface Contract {
    public function run(): void;
}

enum Status {
    case Ready;
}
`
	idx := BuildProjectIndex(map[string][]ast.Node{filename: parsePHPForProjectIndex(t, source)})

	base, ok := idx.ResolveClass(`App\BaseModel`)
	if !ok {
		t.Fatal("expected BaseModel class")
	}
	assertIndexedSymbol(t, source, base.ID, `class:app\basemodel`, base.Declaration, filename)

	child, ok := idx.ResolveClass(`app\childmodel`)
	if !ok {
		t.Fatal("expected ChildModel class")
	}
	assertIndexedSymbol(t, source, child.ID, `class:app\childmodel`, child.Declaration, filename)

	method, ok := idx.ResolveMethod(`App\ChildModel`, "LABEL")
	if !ok || method.DeclaringClass != `App\BaseModel` {
		t.Fatalf("expected inherited BaseModel::label, got %#v, %v", method, ok)
	}
	assertIndexedSymbol(t, source, method.ID, `method:app\basemodel:label`, method.Declaration, filename)

	property, ok := idx.ResolveProperty(`App\ChildModel`, "$name")
	if !ok || property.DeclaringClass != `App\BaseModel` {
		t.Fatalf("expected inherited BaseModel::$name, got %#v, %v", property, ok)
	}
	assertIndexedSymbol(t, source, property.ID, `property:app\basemodel:name`, property.Declaration, filename)

	constant, ok := idx.ResolveConstant(`App\ChildModel`, "KIND")
	if !ok || constant.DeclaringClass != `App\BaseModel` {
		t.Fatalf("expected inherited BaseModel::KIND, got %#v, %v", constant, ok)
	}
	assertIndexedSymbol(t, source, constant.ID, `constant:app\basemodel:kind`, constant.Declaration, filename)

	function, ok := idx.ResolveFunction(`App\load_model`)
	if !ok {
		t.Fatal("expected load_model function")
	}
	assertIndexedSymbol(t, source, function.ID, `function:app\load_model`, function.Declaration, filename)

	interfaceMethod, ok := idx.ResolveMethod(`App\Contract`, "run")
	if !ok {
		t.Fatal("expected Contract::run interface method")
	}
	assertIndexedSymbol(t, source, interfaceMethod.ID, `method:app\contract:run`, interfaceMethod.Declaration, filename)

	enumCase, ok := idx.ResolveConstant(`App\Status`, "Ready")
	if !ok {
		t.Fatal("expected Status::Ready enum case")
	}
	assertIndexedSymbol(t, source, enumCase.ID, `constant:app\status:ready`, enumCase.Declaration, filename)

	enumMethod, ok := idx.ResolveMethod(`App\Status`, "cases")
	if !ok {
		t.Fatal("expected synthetic Status::cases method")
	}
	assertIndexedSymbol(t, source, enumMethod.ID, `method:app\status:cases`, enumMethod.Declaration, filename)
}

func TestProjectIndexAssignsStableIDsToBuiltinsWithoutFakeLocations(t *testing.T) {
	idx := NewProjectIndex()

	class, ok := idx.ResolveClass("DateTimeImmutable")
	if !ok || class.ID != "class:datetimeimmutable" {
		t.Fatalf("unexpected built-in class metadata: %#v, %v", class, ok)
	}
	if class.Declaration != (SourceLocation{}) {
		t.Fatalf("built-in class should not have a fake declaration: %#v", class.Declaration)
	}

	method, ok := idx.ResolveMethod("DateTimeImmutable", "createFromFormat")
	if !ok || method.ID != "method:datetimeimmutable:createfromformat" {
		t.Fatalf("unexpected built-in method metadata: %#v, %v", method, ok)
	}
	if method.Declaration != (SourceLocation{}) {
		t.Fatalf("built-in method should not have a fake declaration: %#v", method.Declaration)
	}

	function, ok := idx.ResolveFunction("strlen")
	if !ok || function.ID != "function:strlen" {
		t.Fatalf("unexpected built-in function metadata: %#v, %v", function, ok)
	}
	if function.Declaration != (SourceLocation{}) {
		t.Fatalf("built-in function should not have a fake declaration: %#v", function.Declaration)
	}
}

func assertIndexedSymbol(t *testing.T, source string, gotID, wantID SymbolID, location SourceLocation, filename string) {
	t.Helper()
	if gotID != wantID {
		t.Fatalf("unexpected symbol ID %q, want %q", gotID, wantID)
	}
	if location.File != filename {
		t.Fatalf("unexpected declaration file %q, want %q", location.File, filename)
	}
	if location.Start.Offset < 0 || location.End.Offset <= location.Start.Offset || location.End.Offset > len(source) {
		t.Fatalf("invalid declaration span %#v for source length %d", location, len(source))
	}
	if strings.TrimSpace(source[location.Start.Offset:location.End.Offset]) == "" {
		t.Fatalf("declaration span %#v covers only whitespace", location)
	}
}
