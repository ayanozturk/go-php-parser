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
	if ctx.Resolver != snapshot || ctx.Facts != snapshot || ctx.Flow != snapshot || ctx.VariableFlow != snapshot {
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
	function := parsed[filename][0].(*ast.FunctionNode)
	functionReturn := function.Body[1].(*ast.ReturnNode)
	functionFact, ok := snapshot.Fact(inferredTypeFactKey(filename, functionReturn.Expr))
	if !ok || functionFact.Type != "string" || functionFact.Subject != "function:answer" {
		t.Fatalf("unexpected function return fact: %#v, %v", functionFact, ok)
	}
	class := parsed[filename][1].(*ast.ClassNode)
	method := class.Methods[0].(*ast.FunctionNode)
	methodReturn := method.Body[0].(*ast.ReturnNode)
	methodFact, ok := snapshot.Fact(inferredTypeFactKey(filename, methodReturn.Expr))
	if !ok || methodFact.Type != "int" || methodFact.Subject != "method:provider:count" {
		t.Fatalf("unexpected method return fact: %#v, %v", methodFact, ok)
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

func TestSemanticSnapshotGeneratesScopeAwareArgumentTypeFacts(t *testing.T) {
	const filename = "src/Example.php"
	nodes, argument := parseMethodArgumentFactFixtureSource(t, `<?php
class Example {
    public function takesString(string $value): void {}

    public function run(): void {
        $value = "ok";
        $this->takesString($value);
    }
}
`)

	snapshot, err := NewSemanticSnapshot(map[string][]ast.Node{filename: nodes}, nil)
	if err != nil {
		t.Fatalf("build snapshot: %v", err)
	}
	fact, ok := snapshot.Fact(inferredTypeFactKey(filename, argument))
	if !ok {
		t.Fatal("expected generated inferred-type fact for method argument")
	}
	if fact.Type != "string" || fact.Subject != "method:example:run" {
		t.Fatalf("unexpected generated argument fact: %#v", fact)
	}
}

func TestSemanticSnapshotGeneratesBranchRefinedArgumentTypeFacts(t *testing.T) {
	const filename = "src/Example.php"
	nodes := parsePHPForProjectIndex(t, `<?php
class Example {
    public function takesString(string $value): void {}

    public function run(mixed $value): void {
        if (is_string($value)) {
            $this->takesString($value);
        }
    }
}
`)
	class := nodes[0].(*ast.ClassNode)
	run := class.Methods[1].(*ast.FunctionNode)
	branch := run.Body[0].(*ast.IfNode)
	call := branch.Body[0].(*ast.ExpressionStmt).Expr.(*ast.MethodCallNode)
	argument := argumentValue(call.Args[0])

	snapshot, err := NewSemanticSnapshot(map[string][]ast.Node{filename: nodes}, nil)
	if err != nil {
		t.Fatalf("build snapshot: %v", err)
	}
	fact, ok := snapshot.Fact(inferredTypeFactKey(filename, argument))
	if !ok || fact.Type != "string" {
		t.Fatalf("expected branch-refined string argument fact, got %#v, %v", fact, ok)
	}
}

func TestSemanticSnapshotGeneratesScopeAwareAssignmentTypeFacts(t *testing.T) {
	const filename = "src/Example.php"
	nodes, rhs := parsePropertyAssignmentFactFixture(t, `<?php
class Example {
    private int $count;

    public function run(): void {
        $value = 42;
        $this->count = $value;
    }
}
`)

	snapshot, err := NewSemanticSnapshot(map[string][]ast.Node{filename: nodes}, nil)
	if err != nil {
		t.Fatalf("build snapshot: %v", err)
	}
	fact, ok := snapshot.Fact(inferredTypeFactKey(filename, rhs))
	if !ok {
		t.Fatal("expected generated inferred-type fact for assignment RHS")
	}
	if fact.Type != "int" || fact.Subject != "method:example:run" {
		t.Fatalf("unexpected generated assignment fact: %#v", fact)
	}

	issues := (&PropertyTypeRule{}).CheckIssues(nodes, filename, snapshot.NewAnalysisContext())
	if hasPropertyTypeIssue(issues) {
		t.Fatalf("generated assignment fact changed compatible property diagnostics: %#v", issues)
	}
}

func TestSemanticSnapshotGeneratesReceiverTypeFacts(t *testing.T) {
	const filename = "src/Example.php"
	nodes := parsePHPForProjectIndex(t, `<?php
class Holder {
    public int $count;
    public function accept(int $value): void {}
}

class Example {
    public function run(): void {
        $holder = new Holder();
        $holder->accept(1);
        $holder->count = 1;
    }
}
`)
	class := nodes[1].(*ast.ClassNode)
	method := class.Methods[0].(*ast.FunctionNode)
	methodCall := method.Body[1].(*ast.ExpressionStmt).Expr.(*ast.MethodCallNode)
	propertyAssignment := method.Body[2].(*ast.ExpressionStmt).Expr.(*ast.AssignmentNode)
	propertyReceiver := propertyAssignment.Left.(*ast.PropertyFetchNode).Object

	snapshot, err := NewSemanticSnapshot(map[string][]ast.Node{filename: nodes}, nil)
	if err != nil {
		t.Fatalf("build snapshot: %v", err)
	}
	for label, receiver := range map[string]ast.Node{"method": methodCall.Object, "property": propertyReceiver} {
		fact, ok := snapshot.Fact(inferredTypeFactKey(filename, receiver))
		if !ok || fact.Type != "Holder" || fact.Subject != "method:example:run" {
			t.Fatalf("unexpected %s receiver fact: %#v, %v", label, fact, ok)
		}
	}
}

func TestSemanticSnapshotGeneratesHoverAndConditionTypeFacts(t *testing.T) {
	const filename = "src/Example.php"
	nodes := parsePHPForProjectIndex(t, `<?php
class Service {
    public function ready(): bool {}
}

class Example {
    public function run(Service $service): void {
        $label = "waiting";
        if ($service->ready()) {
            $label = "ready";
        }
    }
}
`)
	class := nodes[1].(*ast.ClassNode)
	method := class.Methods[0].(*ast.FunctionNode)
	assignment := method.Body[0].(*ast.ExpressionStmt).Expr.(*ast.AssignmentNode)
	variable := assignment.Left.(*ast.VariableNode)
	condition := method.Body[1].(*ast.IfNode).Condition.(*ast.MethodCallNode)

	snapshot, err := NewSemanticSnapshot(map[string][]ast.Node{filename: nodes}, nil)
	if err != nil {
		t.Fatalf("build snapshot: %v", err)
	}
	checks := []struct {
		label string
		node  ast.Node
		typ   string
	}{
		{label: "assignment target", node: variable, typ: "string"},
		{label: "condition", node: condition, typ: "bool"},
	}
	for _, check := range checks {
		fact, ok := snapshot.Fact(inferredTypeFactKey(filename, check.node))
		if !ok || fact.Type != check.typ || fact.Subject != "method:example:run" {
			t.Fatalf("unexpected %s fact: %#v, %v", check.label, fact, ok)
		}
	}
}

func TestSemanticSnapshotExposesDefensiveOwnMemberQueries(t *testing.T) {
	const filename = "src/Models.php"
	nodes := parsePHPForProjectIndex(t, `<?php
class BaseModel {
    final public const KIND = "base";
    final public function locked(int $id): void {}
}

class ChildModel extends BaseModel {
    public function zeta(): void {}
    public function alpha(string $name): void {}
}
`)
	snapshot, err := NewSemanticSnapshot(map[string][]ast.Node{filename: nodes}, nil)
	if err != nil {
		t.Fatalf("build snapshot: %v", err)
	}

	if _, ok := snapshot.ResolveOwnMethod("ChildModel", "locked"); ok {
		t.Fatal("inherited method must not resolve as a ChildModel declaration")
	}
	if inherited, ok := snapshot.ResolveMethod("ChildModel", "locked"); !ok || inherited.DeclaringClass != "BaseModel" {
		t.Fatalf("expected inherited BaseModel::locked, got %#v, %v", inherited, ok)
	}
	methods := snapshot.MethodsDeclaredBy("ChildModel")
	if len(methods) != 2 || methods[0].Name != "alpha" || methods[1].Name != "zeta" {
		t.Fatalf("expected deterministic declared methods [alpha zeta], got %#v", methods)
	}
	methods[0].Params[0].Type = "mutated"
	again := snapshot.MethodsDeclaredBy("ChildModel")
	if again[0].Params[0].Type != "string" {
		t.Fatalf("declared method params leaked caller mutation: %#v", again[0])
	}

	if _, ok := snapshot.ResolveOwnConstant("ChildModel", "KIND"); ok {
		t.Fatal("inherited constant must not resolve as a ChildModel declaration")
	}
	if inherited, ok := snapshot.ResolveConstant("ChildModel", "KIND"); !ok || inherited.DeclaringClass != "BaseModel" {
		t.Fatalf("expected inherited BaseModel::KIND, got %#v, %v", inherited, ok)
	}
}

func TestSemanticSnapshotFiltersAndCopiesDuplicateClasses(t *testing.T) {
	parsed := map[string][]ast.Node{
		"a.php": parsePHPForProjectIndex(t, "<?php class Duplicate {}"),
		"b.php": parsePHPForProjectIndex(t, "<?php class Duplicate {}"),
	}
	snapshot, err := NewSemanticSnapshot(parsed, nil)
	if err != nil {
		t.Fatalf("build snapshot: %v", err)
	}
	if duplicates := snapshot.DuplicateClasses("a.php"); len(duplicates) != 0 {
		t.Fatalf("first declaration must not be reported as duplicate: %#v", duplicates)
	}
	duplicates := snapshot.DuplicateClasses("b.php")
	if len(duplicates) != 1 || duplicates[0].Name != "Duplicate" || duplicates[0].File != "b.php" {
		t.Fatalf("unexpected filtered duplicates: %#v", duplicates)
	}
	duplicates[0].Name = "mutated"
	if again := snapshot.DuplicateClasses("b.php"); len(again) != 1 || again[0].Name != "Duplicate" {
		t.Fatalf("duplicate metadata leaked caller mutation: %#v", again)
	}
}
