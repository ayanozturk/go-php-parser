package analyse

import (
	"reflect"
	"sync"
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
)

func TestSemanticSnapshotIsolatedFromInputsAndResolvedValueMutation(t *testing.T) {
	parsed := map[string][]ast.Node{
		"src/Repository.php": parsePHPForProjectIndex(t, `<?php
namespace App;

/** @template T */
class Repository {
    /** @return T */
    public function find(int $id): object {}
}
`),
	}

	snapshot, err := NewSemanticSnapshot(parsed, nil)
	if err != nil {
		t.Fatalf("build snapshot: %v", err)
	}
	parsed["src/Repository.php"] = nil
	parsed["src/Other.php"] = parsePHPForProjectIndex(t, "<?php class Other {}")

	class, ok := snapshot.ResolveClass(`App\Repository`)
	if !ok {
		t.Fatal("expected Repository class in frozen snapshot")
	}
	if class.ID != `class:app\repository` {
		t.Fatalf("unexpected stable class ID %q", class.ID)
	}
	if snapshot.ClassExists("Other") {
		t.Fatal("snapshot changed after the parsed input map was mutated")
	}

	class.TemplateParams[0] = "Mutated"
	classAgain, _ := snapshot.ResolveClass(`APP\REPOSITORY`)
	if !reflect.DeepEqual(classAgain.TemplateParams, []string{"T"}) {
		t.Fatalf("class metadata leaked caller mutation: %#v", classAgain.TemplateParams)
	}
	if classAgain.ID != class.ID {
		t.Fatalf("class ID changed across case-insensitive lookup: %q != %q", classAgain.ID, class.ID)
	}

	method, ok := snapshot.ResolveMethod(`App\Repository`, "find")
	if !ok {
		t.Fatal("expected Repository::find method")
	}
	if method.ID != `method:app\repository:find` {
		t.Fatalf("unexpected stable method ID %q", method.ID)
	}
	method.Params[0].Name = "mutated"
	methodAgain, _ := snapshot.ResolveMethod(`app\repository`, "FIND")
	if methodAgain.Params[0].Name != "id" {
		t.Fatalf("method parameter metadata leaked caller mutation: %#v", methodAgain.Params)
	}

	files := snapshot.Files()
	files[0] = "mutated.php"
	if got := snapshot.Files(); !reflect.DeepEqual(got, []string{"src/Repository.php"}) {
		t.Fatalf("file list leaked caller mutation: %#v", got)
	}
}

func TestSemanticSnapshotFactsValidateAndSortDeterministically(t *testing.T) {
	first := SemanticFact{
		Key:     SemanticFactKey{File: "src/a.php", StartOffset: 20, EndOffset: 24, Kind: FactKindReference},
		Subject: `class:app\repository`,
		Value:   "read",
	}
	second := SemanticFact{
		Key:   SemanticFactKey{File: "src/a.php", StartOffset: 4, EndOffset: 10, Kind: FactKindInferredType},
		Type:  `App\Repository`,
		Value: "non-null",
	}
	otherFile := SemanticFact{
		Key:  SemanticFactKey{File: "src/b.php", StartOffset: 1, EndOffset: 2, Kind: FactKindInferredType},
		Type: "int",
	}

	snapshot, err := NewSemanticSnapshot(nil, []SemanticFact{first, otherFile, second})
	if err != nil {
		t.Fatalf("build snapshot: %v", err)
	}
	got := snapshot.FactsForFile("src/a.php")
	if !reflect.DeepEqual(got, []SemanticFact{second, first}) {
		t.Fatalf("facts not returned in source order:\n got %#v\nwant %#v", got, []SemanticFact{second, first})
	}
	if fact, ok := snapshot.Fact(first.Key); !ok || fact != first {
		t.Fatalf("exact fact lookup failed: %#v, %v", fact, ok)
	}

	if _, err := NewSemanticSnapshot(nil, []SemanticFact{first, first}); err == nil {
		t.Fatal("expected duplicate fact keys to be rejected")
	}
	invalid := SemanticFact{Key: SemanticFactKey{File: "src/a.php", StartOffset: 8, EndOffset: 7, Kind: FactKindReference}}
	if _, err := NewSemanticSnapshot(nil, []SemanticFact{invalid}); err == nil {
		t.Fatal("expected inverted source span to be rejected")
	}
}

func TestSemanticSnapshotSupportsConcurrentReads(t *testing.T) {
	parsed := map[string][]ast.Node{
		"src/Model.php": parsePHPForProjectIndex(t, `<?php
/** @template T */
class Model {
    public string $name;
    public const KIND = 'model';
    /** @param T $value @return T */
    public function label($value) {}
}
function load_model(int $id): Model {}
`),
	}
	snapshot, err := NewSemanticSnapshot(parsed, []SemanticFact{{
		Key:  SemanticFactKey{File: "src/Model.php", StartOffset: 6, EndOffset: 11, Kind: FactKindInferredType},
		Type: "Model",
	}})
	if err != nil {
		t.Fatalf("build snapshot: %v", err)
	}
	ctx := snapshot.NewAnalysisContext()
	if ctx.Resolver != snapshot || ctx.Facts != snapshot || ctx.Project == nil {
		t.Fatalf("analysis context did not retain the snapshot contracts: %#v", ctx)
	}

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if class, ok := snapshot.ResolveClass("Model"); !ok || class.ID != "class:model" {
				t.Errorf("unexpected class result: %#v, %v", class, ok)
			}
			if method, ok := snapshot.ResolveMethod("Model", "label"); !ok || method.ID != "method:model:label" {
				t.Errorf("unexpected method result: %#v, %v", method, ok)
			}
			if property, ok := snapshot.ResolveProperty("Model", "name"); !ok || property.ID != "property:model:name" || property.DeclaringClass != "Model" {
				t.Errorf("unexpected property result: %#v, %v", property, ok)
			}
			if fn, ok := snapshot.ResolveFunction("load_model"); !ok || fn.ID != "function:load_model" {
				t.Errorf("unexpected function result: %#v, %v", fn, ok)
			}
			if constant, ok := snapshot.ResolveConstant("Model", "KIND"); !ok || constant.ID != "constant:model:kind" {
				t.Errorf("unexpected constant result: %#v, %v", constant, ok)
			}
			if len(snapshot.FactsForFile("src/Model.php")) != 1 {
				t.Error("expected one semantic fact")
			}
		}()
	}
	wg.Wait()
}

func TestSemanticSnapshotGeneratesScopeAwareReturnTypeFacts(t *testing.T) {
	const filename = "src/Answers.php"
	parsed := map[string][]ast.Node{
		filename: parsePHPForProjectIndex(t, `<?php
function answer(): string {
    $value = 'ok';
    return $value;
}

class Provider {
    public function count(): int {
        return 42;
    }
}
`),
	}

	snapshot, err := NewSemanticSnapshot(parsed, nil)
	if err != nil {
		t.Fatalf("build snapshot: %v", err)
	}
	facts := snapshot.FactsForFile(filename)
	if len(facts) != 2 {
		t.Fatalf("expected two generated return facts, got %#v", facts)
	}
	if facts[0].Key.Kind != FactKindInferredType || facts[0].Type != "string" || facts[0].Subject != "function:answer" {
		t.Fatalf("unexpected function return fact: %#v", facts[0])
	}
	if facts[1].Key.Kind != FactKindInferredType || facts[1].Type != "int" || facts[1].Subject != "method:provider:count" {
		t.Fatalf("unexpected method return fact: %#v", facts[1])
	}

	ctx := snapshot.NewAnalysisContext()
	if issues := (&ReturnTypeRule{}).CheckIssues(parsed[filename], filename, ctx); hasReturnTypeIssue(issues) {
		t.Fatalf("generated facts changed compatible return diagnostics: %#v", issues)
	}
}

func TestSemanticSnapshotKeepsExplicitInferredTypeFact(t *testing.T) {
	const filename = "src/Answer.php"
	nodes, expression := parseReturnFactFixture(t, `<?php
function answer(): string {
    return 42;
}
`)
	key := inferredTypeFactKey(filename, expression)
	explicit := SemanticFact{Key: key, Type: "string", Value: "external"}
	snapshot, err := NewSemanticSnapshot(map[string][]ast.Node{filename: nodes}, []SemanticFact{explicit})
	if err != nil {
		t.Fatalf("build snapshot: %v", err)
	}
	if got, ok := snapshot.Fact(key); !ok || got != explicit {
		t.Fatalf("generated int fact overwrote explicit fact: %#v, %v", got, ok)
	}
}
