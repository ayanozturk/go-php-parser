package analyse

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/lexer"
	"github.com/ayanozturk/go-php-parser/parser"
	"reflect"
	"testing"
)

func TestFunctionScopeCloneSharesReadOnlyMaps(t *testing.T) {
	original := seededFunctionScope()
	clone := original.clone()

	if clone == nil {
		t.Fatal("clone returned nil")
	}
	if clone == original {
		t.Fatal("clone returned the original scope")
	}
	if !original.variablesShared || !clone.variablesShared {
		t.Fatalf("variable maps are not marked shared after clone: original=%v clone=%v", original.variablesShared, clone.variablesShared)
	}
	if !original.propertiesShared || !clone.propertiesShared {
		t.Fatalf("property maps are not marked shared after clone: original=%v clone=%v", original.propertiesShared, clone.propertiesShared)
	}
	if got, want := clone.variables["seed"].String(), "int"; got != want {
		t.Fatalf("clone variable seed = %q, want %q", got, want)
	}
	if got, want := clone.properties["state"].String(), "string"; got != want {
		t.Fatalf("clone property state = %q, want %q", got, want)
	}

	assertFunctionScopeMapsShareBacking(t, original, clone)

	allocs := testing.AllocsPerRun(100, func() {
		readOnly := original.clone()
		if readOnly.variables["seed"].IsEmpty() || readOnly.properties["state"].IsEmpty() {
			t.Fatal("read-only clone lost seeded state")
		}
	})
	t.Logf("read-only functionScope.clone allocations/run: %.2f", allocs)
}

func TestFunctionScopeCloneFirstVariableWriteIsolatesClone(t *testing.T) {
	original := seededFunctionScope()
	clone := original.clone()

	clone.setVariable("seed", ParseType("bool"))
	clone.setVariable("branch", ParseType("float"))

	assertScopeType(t, clone.variables, "seed", "bool")
	assertScopeType(t, clone.variables, "branch", "float")
	assertScopeType(t, original.variables, "seed", "int")
	if _, ok := original.variables["branch"]; ok {
		t.Fatal("clone variable write leaked into original")
	}
	if functionScopeMapPointer(clone.variables) == functionScopeMapPointer(original.variables) {
		t.Fatal("clone variable write did not detach the variable map")
	}
	if clone.variablesShared || !original.variablesShared {
		t.Fatalf("variable sharing flags after clone write = clone=%v original=%v", clone.variablesShared, original.variablesShared)
	}
	if functionScopeMapPointer(clone.properties) != functionScopeMapPointer(original.properties) {
		t.Fatal("unmodified property map detached during variable write")
	}
}

func TestFunctionScopeCloneFirstPropertyWriteIsolatesClone(t *testing.T) {
	original := seededFunctionScope()
	clone := original.clone()

	clone.setProperty("state", ParseType("bool"))
	clone.setProperty("branch", ParseType("float"))

	assertScopeType(t, clone.properties, "state", "bool")
	assertScopeType(t, clone.properties, "branch", "float")
	assertScopeType(t, original.properties, "state", "string")
	if _, ok := original.properties["branch"]; ok {
		t.Fatal("clone property write leaked into original")
	}
	if functionScopeMapPointer(clone.properties) == functionScopeMapPointer(original.properties) {
		t.Fatal("clone property write did not detach the property map")
	}
	if clone.propertiesShared || !original.propertiesShared {
		t.Fatalf("property sharing flags after clone write = clone=%v original=%v", clone.propertiesShared, original.propertiesShared)
	}
	if functionScopeMapPointer(clone.variables) != functionScopeMapPointer(original.variables) {
		t.Fatal("unmodified variable map detached during property write")
	}
}

func TestFunctionScopeOriginalWriteAfterCloneIsolatesOriginal(t *testing.T) {
	original := seededFunctionScope()
	clone := original.clone()

	original.setVariable("seed", ParseType("float"))
	original.setVariable("originalOnly", ParseType("bool"))
	original.setProperty("state", ParseType("bool"))
	original.setProperty("originalOnly", ParseType("int"))

	assertScopeType(t, original.variables, "seed", "float")
	assertScopeType(t, original.variables, "originalOnly", "bool")
	assertScopeType(t, original.properties, "state", "bool")
	assertScopeType(t, original.properties, "originalOnly", "int")
	assertScopeType(t, clone.variables, "seed", "int")
	assertScopeType(t, clone.properties, "state", "string")
	if _, ok := clone.variables["originalOnly"]; ok {
		t.Fatal("original variable write leaked into clone")
	}
	if _, ok := clone.properties["originalOnly"]; ok {
		t.Fatal("original property write leaked into clone")
	}
	if functionScopeMapPointer(clone.variables) == functionScopeMapPointer(original.variables) {
		t.Fatal("original variable write did not detach the original map")
	}
	if functionScopeMapPointer(clone.properties) == functionScopeMapPointer(original.properties) {
		t.Fatal("original property write did not detach the original map")
	}
	if original.variablesShared || !clone.variablesShared {
		t.Fatalf("variable sharing flags after original write = original=%v clone=%v", original.variablesShared, clone.variablesShared)
	}
	if original.propertiesShared || !clone.propertiesShared {
		t.Fatalf("property sharing flags after original write = original=%v clone=%v", original.propertiesShared, clone.propertiesShared)
	}
}

func TestFunctionScopeChainedAndSiblingClonesRemainIndependent(t *testing.T) {
	root := seededFunctionScope()
	parent := root.clone()
	left := parent.clone()
	right := parent.clone()

	left.setVariable("branch", ParseType("bool"))
	right.setVariable("branch", ParseType("float"))
	parent.setProperty("branch", ParseType("int"))
	root.setVariable("rootOnly", ParseType("string"))

	assertScopeType(t, left.variables, "branch", "bool")
	assertScopeType(t, right.variables, "branch", "float")
	assertScopeType(t, parent.properties, "branch", "int")
	if _, ok := parent.variables["branch"]; ok {
		t.Fatal("sibling variable write leaked into parent")
	}
	if _, ok := left.variables["rootOnly"]; ok {
		t.Fatal("root variable write leaked into left sibling")
	}
	if _, ok := right.variables["rootOnly"]; ok {
		t.Fatal("root variable write leaked into right sibling")
	}
	if _, ok := left.properties["branch"]; ok {
		t.Fatal("parent property write leaked into left sibling")
	}
	if _, ok := right.properties["branch"]; ok {
		t.Fatal("parent property write leaked into right sibling")
	}
	assertScopeType(t, root.variables, "seed", "int")
	if _, ok := root.variables["branch"]; ok {
		t.Fatal("descendant variable write leaked into root")
	}
}

func TestFunctionScopeCallableReturnClonesRemainIndependent(t *testing.T) {
	root := &functionScope{callableReturns: map[string]Type{"factory": ParseType("InitialService")}}
	parent := root.clone()
	left := parent.clone()
	right := parent.clone()

	left.setCallableReturn("factory", ParseType("LeftService"))
	left.setCallableReturn("leftOnly", ParseType("bool"))
	right.setCallableReturn("factory", ParseType("RightService"))
	right.setCallableReturn("rightOnly", ParseType("int"))
	parent.clearCallableReturn("factory")

	assertScopeType(t, root.callableReturns, "factory", "InitialService")
	assertScopeType(t, left.callableReturns, "factory", "LeftService")
	assertScopeType(t, right.callableReturns, "factory", "RightService")
	if _, ok := parent.callableReturns["factory"]; ok {
		t.Fatal("parent callable return deletion leaked into root or siblings")
	}
	if _, ok := left.callableReturns["rightOnly"]; ok {
		t.Fatal("right sibling callable return leaked into left sibling")
	}
	if _, ok := right.callableReturns["leftOnly"]; ok {
		t.Fatal("left sibling callable return leaked into right sibling")
	}
	if functionScopeMapPointer(left.callableReturns) == functionScopeMapPointer(right.callableReturns) {
		t.Fatal("callable return sibling writes did not detach their maps")
	}
}

func TestFunctionScopeClassStringMetadataClonesRemainIndependent(t *testing.T) {
	root := &functionScope{genericContext: map[string]GenericInstance{
		"class": {ClassName: "class-string", TypeArguments: []string{"InitialService"}},
	}}
	left := root.clone()
	right := root.clone()

	delete(left.genericContext, "class")
	left.genericContext["left"] = GenericInstance{ClassName: "class-string", TypeArguments: []string{"LeftService"}}

	if target, ok := classStringTarget(root, "class"); !ok || target.String() != "InitialService" {
		t.Fatalf("root class-string target changed through clone: %q, %v", target.String(), ok)
	}
	if target, ok := classStringTarget(right, "class"); !ok || target.String() != "InitialService" {
		t.Fatalf("sibling class-string target changed through clone: %q, %v", target.String(), ok)
	}
	if _, ok := left.genericContext["class"]; ok {
		t.Fatal("left clone retained deleted class-string metadata")
	}
}

func TestFunctionScopeCloneNilIsSafe(t *testing.T) {
	var scope *functionScope
	if got := scope.clone(); got != nil {
		t.Fatalf("nil clone = %#v, want nil", got)
	}
}

func TestNewFunctionScopeWithContextSharesAndDetachesCachedClassProperties(t *testing.T) {
	class, method, typeCtx := functionScopeClassFixture(t)
	ctx := &AnalysisContext{}

	first := newFunctionScopeWithContext(ctx, class, method, typeCtx)
	second := newFunctionScopeWithContext(ctx, class, method, typeCtx)
	if !first.propertiesShared || !second.propertiesShared {
		t.Fatalf("new scopes should share cached class properties: first=%v second=%v", first.propertiesShared, second.propertiesShared)
	}
	if functionScopeMapPointer(first.properties) != functionScopeMapPointer(second.properties) {
		t.Fatal("new scopes did not share cached class property map")
	}
	assertScopeType(t, first.properties, "state", "string")
	assertScopeType(t, second.properties, "count", "int")

	first.setProperty("state", ParseType("bool"))
	if first.propertiesShared {
		t.Fatal("first scope remained shared after its first property write")
	}
	if !second.propertiesShared {
		t.Fatal("second scope detached before it was written")
	}
	assertScopeType(t, first.properties, "state", "bool")
	assertScopeType(t, second.properties, "state", "string")
	if functionScopeMapPointer(first.properties) == functionScopeMapPointer(second.properties) {
		t.Fatal("first property write did not detach the scope")
	}

	second.setProperty("count", ParseType("string"))
	if second.propertiesShared {
		t.Fatal("second scope remained shared after its first property write")
	}
	assertScopeType(t, second.properties, "count", "string")
	assertScopeType(t, first.properties, "count", "int")

	third := newFunctionScopeWithContext(ctx, class, method, typeCtx)
	if !third.propertiesShared {
		t.Fatal("new scope did not share the cached class properties after sibling writes")
	}
	assertScopeType(t, third.properties, "state", "string")
	assertScopeType(t, third.properties, "count", "int")
}

func BenchmarkFunctionScopeCloneReadOnly(b *testing.B) {
	scope := seededFunctionScope()
	var sink *functionScope
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sink = scope.clone()
	}
	benchmarkFunctionScopeSink = sink
}

func BenchmarkFunctionScopeCloneAndWrite(b *testing.B) {
	scope := seededFunctionScope()
	variableType := ParseType("bool")
	propertyType := ParseType("int")
	var sink *functionScope
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sink = scope.clone()
		sink.setVariable("branch", variableType)
		sink.setProperty("branch", propertyType)
	}
	benchmarkFunctionScopeSink = sink
}

func BenchmarkNewFunctionScopeWithContextCachedClassProperties(b *testing.B) {
	class, method, typeCtx := functionScopeClassFixture(b)
	ctx := &AnalysisContext{}
	var sink *functionScope
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sink = newFunctionScopeWithContext(ctx, class, method, typeCtx)
	}
	benchmarkFunctionScopeSink = sink
}

var benchmarkFunctionScopeSink *functionScope

func seededFunctionScope() *functionScope {
	return &functionScope{
		variables:  map[string]Type{"seed": ParseType("int")},
		properties: map[string]Type{"state": ParseType("string")},
	}
}

func assertScopeType(t *testing.T, values map[string]Type, name, want string) {
	t.Helper()
	typ, ok := values[name]
	if !ok {
		t.Fatalf("scope is missing %q", name)
	}
	if got := typ.String(); got != want {
		t.Fatalf("scope[%q] = %q, want %q", name, got, want)
	}
}

func assertFunctionScopeMapsShareBacking(t *testing.T, first, second *functionScope) {
	t.Helper()
	if got, want := functionScopeMapPointer(first.variables), functionScopeMapPointer(second.variables); got != want {
		t.Fatalf("variable map pointers differ: %x != %x", got, want)
	}
	if got, want := functionScopeMapPointer(first.properties), functionScopeMapPointer(second.properties); got != want {
		t.Fatalf("property map pointers differ: %x != %x", got, want)
	}
}

func functionScopeMapPointer(values map[string]Type) uintptr {
	if values == nil {
		return 0
	}
	return reflect.ValueOf(values).Pointer()
}

func functionScopeClassFixture(tb testing.TB) (*ast.ClassNode, *ast.FunctionNode, FileTypeContext) {
	tb.Helper()
	const source = `<?php
class CachedProperties {
    public string $state;
    protected int $count;

    public function run(string $input): void {}
}
`
	p := parser.New(lexer.New(source), false)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		tb.Fatalf("parse errors: %v", p.Errors())
	}
	for _, node := range nodes {
		class, ok := node.(*ast.ClassNode)
		if !ok {
			continue
		}
		for _, member := range class.Methods {
			method, ok := member.(*ast.FunctionNode)
			if ok {
				return class, method, CollectFileTypeContext(nodes)
			}
		}
	}
	tb.Fatal("fixture did not contain a class method")
	return nil, nil, FileTypeContext{}
}
