package analyse

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/lexer"
	"github.com/ayanozturk/go-php-parser/parser"
	"reflect"
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

	enumName, ok := idx.ResolveProperty(`App\Status`, "name")
	if !ok || enumName.Type != "string" {
		t.Fatalf("expected synthetic Status::$name, got %#v, %v", enumName, ok)
	}
	assertIndexedSymbol(t, source, enumName.ID, `property:app\status:name`, enumName.Declaration, filename)

	if _, ok := idx.ResolveProperty(`App\Status`, "value"); ok {
		t.Fatal("did not expect unit enum Status to expose $value")
	}

	enumMethod, ok := idx.ResolveMethod(`App\Status`, "cases")
	if !ok {
		t.Fatal("expected synthetic Status::cases method")
	}
	assertIndexedSymbol(t, source, enumMethod.ID, `method:app\status:cases`, enumMethod.Declaration, filename)
}

func TestProjectIndexExposesCallableReturnMetadata(t *testing.T) {
	parsed := map[string][]ast.Node{
		"service.php": parsePHPForProjectIndex(t, `<?php
class Service {}
`),
		"declarations.php": parsePHPForProjectIndex(t, `<?php
class Holder {
    /** @var callable(): Service */
    public $factory;

    /** @return callable(): Service */
    public function make(): callable {}
}

interface FactoryContract {
	/** @var callable(): Service */
	public $factory { get; }

	/** @return callable(): Service */
	public function make(): callable;
}

/** @return callable(): Service */
function makeService(): callable {}
`),
	}
	idx := BuildProjectIndex(parsed)

	property, ok := idx.ResolveProperty("Holder", "factory")
	if !ok || property.Type != "callable" || property.CallableReturnType != "Service" {
		t.Fatalf("unexpected callable property metadata: %#v, %v", property, ok)
	}

	function, ok := idx.ResolveFunction("makeService")
	if !ok || function.ReturnType != "callable" || function.CallableReturnType != "Service" {
		t.Fatalf("unexpected callable global function metadata: %#v, %v", function, ok)
	}

	method, ok := idx.ResolveMethod("Holder", "make")
	if !ok || method.ReturnType != "callable" || method.CallableReturnType != "Service" {
		t.Fatalf("unexpected callable class method metadata: %#v, %v", method, ok)
	}

	interfaceMethod, ok := idx.ResolveMethod("FactoryContract", "make")
	if !ok || interfaceMethod.ReturnType != "callable" || interfaceMethod.CallableReturnType != "Service" {
		t.Fatalf("unexpected callable interface method metadata: %#v, %v", interfaceMethod, ok)
	}

	interfaceProperty, ok := idx.ResolveProperty("FactoryContract", "factory")
	if !ok || interfaceProperty.Type != "callable" || interfaceProperty.CallableReturnType != "Service" {
		t.Fatalf("unexpected callable interface property metadata: %#v, %v", interfaceProperty, ok)
	}
}

func TestProjectIndexKeepsNativePropertyTypeWhenPHPDocAddsGenerics(t *testing.T) {
	parsed := map[string][]ast.Node{
		"collections.php": parsePHPForProjectIndex(t, `<?php
namespace Doctrine\Common\Collections;
interface Collection {}
`),
		"entity.php": parsePHPForProjectIndex(t, `<?php
namespace App;
use Doctrine\Common\Collections\Collection;
class Policy {}
class Entity {
    /** @var Collection<string, Policy> */
    private Collection $users;
}
`),
	}
	idx := BuildProjectIndex(parsed)
	property, ok := idx.ResolveProperty(`App\Entity`, "users")
	if !ok || property.Type != `Doctrine\Common\Collections\Collection` {
		t.Fatalf("native Collection type should win over generic PHPDoc, got %#v, %v", property, ok)
	}
}

func TestProjectIndexAssignsStableIDsToBuiltins(t *testing.T) {
	idx := NewProjectIndex()

	class, ok := idx.ResolveClass("DateTimeImmutable")
	if !ok || class.ID != "class:datetimeimmutable" {
		t.Fatalf("unexpected built-in class metadata: %#v, %v", class, ok)
	}
	if class.Declaration.File == "" {
		t.Fatal("bundled PHP stubs should give built-in classes a declaration file")
	}

	method, ok := idx.ResolveMethod("DateTimeImmutable", "createFromFormat")
	if !ok || method.ID != "method:datetimeimmutable:createfromformat" {
		t.Fatalf("unexpected built-in method metadata: %#v, %v", method, ok)
	}

	function, ok := idx.ResolveFunction("strlen")
	if !ok || function.ID != "function:strlen" {
		t.Fatalf("unexpected built-in function metadata: %#v, %v", function, ok)
	}
	if function.Declaration != (SourceLocation{}) {
		t.Fatalf("built-in function should not have a fake declaration: %#v", function.Declaration)
	}
}

func TestProjectIndexDateTimeModifyUsesPHP83ReturnContract(t *testing.T) {
	idx := NewProjectIndex()
	for _, className := range []string{"DateTime", "DateTimeImmutable"} {
		method, ok := idx.ResolveMethod(className, "modify")
		if !ok {
			t.Fatalf("expected built-in %s::modify() metadata", className)
		}
		if method.ReturnType != className {
			t.Fatalf("%s::modify() return type = %q, want %q", className, method.ReturnType, className)
		}
	}
}

func TestProjectIndexClassifiesBuiltinReferenceParameters(t *testing.T) {
	idx := NewProjectIndex()
	outputOnly := map[string][]string{
		"exec":                        {"output", "result_code"},
		"fscanf":                      {"vars"},
		"getopt":                      {"rest_index"},
		"headers_sent":                {"filename", "line"},
		"passthru":                    {"result_code"},
		"preg_filter":                 {"count"},
		"preg_match_all":              {"matches"},
		"preg_replace":                {"count"},
		"preg_replace_callback":       {"count"},
		"preg_replace_callback_array": {"count"},
		"proc_open":                   {"pipes"},
		"similar_text":                {"percent"},
		"sscanf":                      {"vars"},
		"system":                      {"result_code"},
	}
	for functionName, paramNames := range outputOnly {
		function, ok := idx.ResolveFunction(functionName)
		if !ok {
			t.Fatalf("expected built-in function %s", functionName)
		}
		for _, paramName := range paramNames {
			param, ok := resolvedParamNamed(function.Params, paramName)
			if !ok || !param.IsByRef || !param.IsOut {
				t.Errorf("expected %s($%s) to be output-only, got %#v, %v", functionName, paramName, param, ok)
			}
		}
	}

	inputOutput := map[string]string{
		"array_splice":         "array",
		"array_walk":           "array",
		"array_walk_recursive": "array",
		"arsort":               "array",
		"asort":                "array",
		"krsort":               "array",
		"natcasesort":          "array",
		"natsort":              "array",
		"next":                 "array",
		"prev":                 "array",
		"rsort":                "array",
		"settype":              "var",
		"shuffle":              "array",
		"uasort":               "array",
		"uksort":               "array",
		"usort":                "array",
	}
	for functionName, paramName := range inputOutput {
		function, ok := idx.ResolveFunction(functionName)
		if !ok {
			t.Fatalf("expected built-in function %s", functionName)
		}
		param, ok := resolvedParamNamed(function.Params, paramName)
		if !ok || !param.IsByRef || param.IsOut {
			t.Errorf("expected %s($%s) to be input/output, got %#v, %v", functionName, paramName, param, ok)
		}
	}

	for functionName, paramName := range map[string]string{"fscanf": "vars", "sscanf": "vars"} {
		function, _ := idx.ResolveFunction(functionName)
		param, _ := resolvedParamNamed(function.Params, paramName)
		if !param.IsVariadic || !param.HasDefault {
			t.Errorf("expected %s($%s) to be optional variadic, got %#v", functionName, paramName, param)
		}
	}
}

func resolvedParamNamed(params []ResolvedParam, name string) (ResolvedParam, bool) {
	for _, param := range params {
		if param.Name == name {
			return param, true
		}
	}
	return ResolvedParam{}, false
}

func TestProjectIndexClassLineagePreservesOrderAndDeduplicates(t *testing.T) {
	const source = `<?php
interface Root {}
interface Left extends Root {}
interface Right extends Root {}
trait Shared {}
class Base implements Left, Right { use Shared; }
class Child extends BASE implements RIGHT { use Shared; }
class CycleA extends CycleB {}
class CycleB extends CycleA {}
`
	parsed := map[string][]ast.Node{"lineage.php": parsePHPForProjectIndex(t, source)}
	want := map[string][]string{
		"child":  {"Child", "Base", "Left", "Root", "Right", "Shared"},
		"cyclea": {"CycleA", "CycleB"},
	}

	for i := 0; i < 25; i++ {
		idx := BuildProjectIndex(parsed)
		for query, expected := range want {
			if got := idx.classLineage(query); !reflect.DeepEqual(got, expected) {
				t.Fatalf("iteration %d: classLineage(%q) = %#v, want %#v", i, query, got, expected)
			}
		}
		if got := idx.classLineage("cHiLd"); !reflect.DeepEqual(got, want["child"]) {
			t.Fatalf("iteration %d: mixed-case classLineage = %#v, want %#v", i, got, want["child"])
		}
	}
}

func TestNewProjectIndexClassLineageFallsBackForMutableIndexes(t *testing.T) {
	idx := NewProjectIndex()
	idx.Classes[indexKey("Base")] = ResolvedClass{Name: "Base"}
	idx.Classes[indexKey("Child")] = ResolvedClass{Name: "Child", Extends: []string{"BASE"}}
	idx.Properties[indexKey("Base")] = map[string]ResolvedProperty{
		"state": {Name: "state", Type: "string", DeclaringClass: "Base"},
	}
	idx.ClassConsts[indexKey("Base")] = map[string]ResolvedConstant{
		"kind": {Name: "KIND", DeclaringClass: "Base", Type: "string"},
	}

	if idx.classLineages != nil {
		t.Fatal("NewProjectIndex should not require a precomputed lineage view")
	}
	if got, want := idx.classLineage("cHiLd"), []string{"Child", "Base"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("mutable index classLineage = %#v, want %#v", got, want)
	}
	property, ok := idx.ResolveProperty("CHILD", "$state")
	if !ok || property.DeclaringClass != "Base" || property.Type != "string" {
		t.Fatalf("mutable index inherited property = %#v, %v", property, ok)
	}
	constant, ok := idx.ResolveConstant("child", "kind")
	if !ok || constant.DeclaringClass != "Base" || constant.Name != "KIND" {
		t.Fatalf("mutable index inherited constant = %#v, %v", constant, ok)
	}
}

func TestProjectIndexCachedLineageReducesInheritedResolutionAllocations(t *testing.T) {
	const source = `<?php
class Base {
    public string $state;
    public const KIND = 'base';
}
class Parent extends Base {}
class Child extends Parent {}
`
	parsed := map[string][]ast.Node{"lineage.php": parsePHPForProjectIndex(t, source)}
	cached := BuildProjectIndex(parsed)
	uncached := BuildProjectIndex(parsed)
	uncached.classLineages = nil

	resolve := func(idx *ProjectIndex) {
		property, propertyOK := idx.ResolveProperty("CHILD", "$state")
		constant, constantOK := idx.ResolveConstant("child", "kind")
		if !propertyOK || !constantOK || property.Type != "string" || constant.Name != "KIND" {
			t.Fatalf("inherited resolution failed: property=%#v/%v constant=%#v/%v", property, propertyOK, constant, constantOK)
		}
	}
	resolve(cached)
	resolve(uncached)

	cachedAllocs := testing.AllocsPerRun(1000, func() { resolve(cached) })
	uncachedAllocs := testing.AllocsPerRun(1000, func() { resolve(uncached) })
	t.Logf("cached inherited property/constant resolution allocations: %.2f; uncached: %.2f", cachedAllocs, uncachedAllocs)
	if cachedAllocs >= uncachedAllocs {
		t.Fatalf("cached lineage allocations = %.2f, want less than uncached %.2f", cachedAllocs, uncachedAllocs)
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
