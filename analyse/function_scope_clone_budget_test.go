package analyse

import (
	"reflect"
	"testing"
	"unsafe"

	"github.com/ayanozturk/go-php-parser/ast"
)

// This fixture deliberately fills both the immutable class metadata and every
// copy-on-write map. It makes a read-only clone's sharing contract observable
// without depending on an analysis pipeline to populate the scope.
func functionScopeCloneBudgetFixture() *functionScope {
	return &functionScope{
		functionScopeContext: &functionScopeContext{
			className: "Example\\Service",
			typeCtx: FileTypeContext{
				Namespace: "Example",
				Aliases:   map[string]string{"Service": "Example\\Service"},
				Classes:   map[string]ResolvedClass{"example\\service": {Name: "Example\\Service"}},
				ClassNodes: map[string]*ast.ClassNode{
					"example\\service": {Name: "Service"},
				},
				Constants: map[string]string{"VERSION": "1"},
			},
			propertyDecls:           map[string]Type{"state": ParseType("string")},
			methods:                 map[string]ResolvedMethod{"run": {Name: "run", ReturnType: "void"}},
			methodReturns:           map[string]Type{"run": ParseType("void")},
			propertyCallableReturns: map[string]Type{"factory": ParseType("Service")},
			classConstantValues:     map[string]string{"VERSION": "1"},
			propertyArrayShapes: map[string]map[string]arrayShapeField{
				"config": {"service": {typ: ParseType("Service")}},
			},
			methodArrayShapes: map[string]map[string]arrayShapeField{
				"run": {"service": {typ: ParseType("Service")}},
			},
		},
		variables:       rootScopeTypeLayer(map[string]Type{"seed": ParseType("int")}),
		properties:      rootScopeTypeLayer(map[string]Type{"state": ParseType("string")}),
		variablesOwned:  true,
		propertiesOwned: true,
		callableReturns: map[string]Type{"factory": ParseType("Service")},
		arrayShapeCallables: map[string]map[string]arrayShapeField{
			"factories": {"service": {callable: ParseType("Service")}},
		},
		arrayIndexKeys: map[string][]string{"key": {"service"}},
		genericContext: map[string]GenericInstance{
			"service": {ClassName: "Repository", TypeArguments: []string{"Service"}},
		},
	}
}

func TestFunctionScopeCloneReadOnlyAllocationAndSizeBudget(t *testing.T) {
	root := functionScopeCloneBudgetFixture()
	clone := root.clone()
	if clone == nil {
		t.Fatal("read-only clone returned nil")
	}

	// A read-only clone should allocate only its shallow scope value. The size
	// log makes the value-layout cost visible while this test guards against
	// accidentally materializing any metadata maps.
	const runs = 200
	var sink *functionScope
	allocs := testing.AllocsPerRun(runs, func() {
		sink = root.clone()
	})
	benchmarkFunctionScopeCloneBudgetSink = sink
	scopeSize := unsafe.Sizeof(*root)
	t.Logf("read-only functionScope.clone: allocs/run=%.2f, value-size=%d bytes", allocs, scopeSize)
	if allocs > 1 {
		t.Fatalf("read-only functionScope.clone allocations/run = %.2f, want at most one scope allocation", allocs)
	}
	if scopeSize > 96 {
		t.Fatalf("functionScope value size = %d bytes, want <= 96", scopeSize)
	}
	if got, want := unsafe.Sizeof(*clone), unsafe.Sizeof(*root); got != want {
		t.Fatalf("clone value size = %d bytes, want %d", got, want)
	}

	if got, want := clone.className, root.className; got != want {
		t.Fatalf("clone class name = %q, want %q", got, want)
	}
	if clone.functionScopeContext != root.functionScopeContext {
		t.Fatal("read-only clone did not share immutable function-scope context")
	}
	assertFunctionScopeCloneBudgetMapShared(t, "property declarations", root.propertyDecls, clone.propertyDecls)
	assertFunctionScopeCloneBudgetMapShared(t, "methods", root.methods, clone.methods)
	assertFunctionScopeCloneBudgetMapShared(t, "method returns", root.methodReturns, clone.methodReturns)
	assertFunctionScopeCloneBudgetMapShared(t, "property callable returns", root.propertyCallableReturns, clone.propertyCallableReturns)
	assertFunctionScopeCloneBudgetMapShared(t, "class constants", root.classConstantValues, clone.classConstantValues)
	assertFunctionScopeCloneBudgetMapShared(t, "property array shapes", root.propertyArrayShapes, clone.propertyArrayShapes)
	assertFunctionScopeCloneBudgetMapShared(t, "method array shapes", root.methodArrayShapes, clone.methodArrayShapes)
	assertFunctionScopeCloneBudgetMapShared(t, "type aliases", root.typeCtx.Aliases, clone.typeCtx.Aliases)
	assertFunctionScopeCloneBudgetMapShared(t, "type classes", root.typeCtx.Classes, clone.typeCtx.Classes)
	assertFunctionScopeCloneBudgetMapShared(t, "type class nodes", root.typeCtx.ClassNodes, clone.typeCtx.ClassNodes)
	assertFunctionScopeCloneBudgetMapShared(t, "type constants", root.typeCtx.Constants, clone.typeCtx.Constants)

	if got, want := clone.propertyArrayShapes["config"], root.propertyArrayShapes["config"]; cloneMapPointer(got) != cloneMapPointer(want) {
		t.Fatal("clone copied nested property array-shape metadata")
	}
	if got, want := clone.methodArrayShapes["run"], root.methodArrayShapes["run"]; cloneMapPointer(got) != cloneMapPointer(want) {
		t.Fatal("clone copied nested method array-shape metadata")
	}
	if got, want := clone.typeCtx.Aliases, root.typeCtx.Aliases; cloneMapPointer(got) != cloneMapPointer(want) {
		t.Fatal("clone copied type-context aliases")
	}
}

func TestFunctionScopeCloneAllMutableStateWritesAreIsolated(t *testing.T) {
	root := functionScopeCloneBudgetFixture()
	clone := root.clone()

	clone.setVariable("seed", ParseType("bool"))
	clone.setVariable("branch", ParseType("float"))
	clone.setProperty("state", ParseType("bool"))
	clone.setProperty("branch", ParseType("float"))
	clone.setCallableReturn("factory", ParseType("OtherService"))
	clone.setCallableReturn("cloneOnly", ParseType("bool"))
	clone.setArrayShapeCallables("factories", map[string]arrayShapeField{"service": {callable: ParseType("OtherService")}})
	clone.setArrayShapeCallables("cloneOnly", map[string]arrayShapeField{"service": {callable: ParseType("bool")}})
	clone.setArrayIndexKeys("key", []string{"other"})
	clone.setArrayIndexKeys("cloneOnly", []string{"inner"})
	clone.setGenericContext("service", GenericInstance{ClassName: "OtherRepository", TypeArguments: []string{"OtherService"}})
	clone.setGenericContext("cloneOnly", GenericInstance{ClassName: "Repository", TypeArguments: []string{"bool"}})

	assertCloneBudgetVariable(t, clone, "seed", "bool")
	assertCloneBudgetVariable(t, clone, "branch", "float")
	assertCloneBudgetVariable(t, root, "seed", "int")
	assertCloneBudgetVariableMissing(t, root, "branch")
	assertCloneBudgetProperty(t, clone, "state", "bool")
	assertCloneBudgetProperty(t, clone, "branch", "float")
	assertCloneBudgetProperty(t, root, "state", "string")
	assertCloneBudgetPropertyMissing(t, root, "branch")
	assertScopeType(t, root.callableReturns, "factory", "Service")
	assertScopeType(t, clone.callableReturns, "factory", "OtherService")
	assertScopeType(t, clone.callableReturns, "cloneOnly", "bool")
	if _, ok := root.callableReturns["cloneOnly"]; ok {
		t.Fatal("clone-only callable return leaked into root")
	}
	if got := root.arrayShapeCallables["factories"]["service"].callable.String(); got != "Service" {
		t.Fatalf("root array-shape callable = %q, want Service", got)
	}
	if _, ok := root.arrayShapeCallables["cloneOnly"]; ok {
		t.Fatal("clone-only array-shape callable leaked into root")
	}
	if got := root.arrayIndexKeys["key"][0]; got != "service" {
		t.Fatalf("root array-index key = %q, want service", got)
	}
	if _, ok := root.arrayIndexKeys["cloneOnly"]; ok {
		t.Fatal("clone-only array-index key leaked into root")
	}
	if got := root.genericContext["service"].ClassName; got != "Repository" {
		t.Fatalf("root generic class = %q, want Repository", got)
	}
	if _, ok := root.genericContext["cloneOnly"]; ok {
		t.Fatal("clone-only generic context leaked into root")
	}
	if got := clone.arrayShapeCallables["factories"]["service"].callable.String(); got != "OtherService" {
		t.Fatalf("clone array-shape callable = %q, want OtherService", got)
	}
	if got := clone.arrayIndexKeys["key"][0]; got != "other" {
		t.Fatalf("clone array-index key = %q, want other", got)
	}
	if got := clone.genericContext["service"].ClassName; got != "OtherRepository" {
		t.Fatalf("clone generic class = %q, want OtherRepository", got)
	}
	if got := clone.genericContext["service"].TypeArguments[0]; got != "OtherService" {
		t.Fatalf("clone generic type argument = %q, want OtherService", got)
	}

	if clone.variables == root.variables || clone.properties == root.properties {
		t.Fatal("variable/property writes did not detach clone layers")
	}
	if cloneMapPointer(clone.callableReturns) == cloneMapPointer(root.callableReturns) {
		t.Fatal("callable-return write did not detach clone map")
	}
	if cloneMapPointer(clone.arrayShapeCallables) == cloneMapPointer(root.arrayShapeCallables) {
		t.Fatal("array-shape write did not detach clone map")
	}
	if cloneMapPointer(clone.arrayIndexKeys) == cloneMapPointer(root.arrayIndexKeys) {
		t.Fatal("array-index write did not detach clone map")
	}
	if cloneMapPointer(clone.genericContext) == cloneMapPointer(root.genericContext) {
		t.Fatal("generic-context write did not detach clone map")
	}

	if got := clone.variables.values; len(got) != 1 {
		t.Fatalf("clone variable delta map size = %d, want one entry", len(got))
	}
	if !clone.variables.hasOne {
		t.Fatal("clone variable delta did not retain its first write as a compact slot")
	}
	if got := len(root.variables.values); got != 1 {
		t.Fatalf("root variable map size = %d, want one entry", got)
	}
	if got := clone.properties.values; len(got) != 1 {
		t.Fatalf("clone property delta map size = %d, want one entry", len(got))
	}
	if !clone.properties.hasOne {
		t.Fatal("clone property delta did not retain its first write as a compact slot")
	}
	if got := len(root.properties.values); got != 1 {
		t.Fatalf("root property map size = %d, want one entry", got)
	}
}

var benchmarkFunctionScopeCloneBudgetSink *functionScope

func assertFunctionScopeCloneBudgetMapShared(t *testing.T, name string, original, clone any) {
	t.Helper()
	if got, want := cloneMapPointer(clone), cloneMapPointer(original); got != want {
		t.Fatalf("read-only clone copied %s: clone=%x root=%x", name, got, want)
	}
}

func cloneMapPointer(value any) uintptr {
	if value == nil {
		return 0
	}
	return reflect.ValueOf(value).Pointer()
}

func assertCloneBudgetVariable(t *testing.T, scope *functionScope, name, want string) {
	t.Helper()
	typ, ok := scope.variable(name)
	if !ok {
		t.Fatalf("scope is missing variable %q", name)
	}
	if got := typ.String(); got != want {
		t.Fatalf("variable %q = %q, want %q", name, got, want)
	}
}

func assertCloneBudgetVariableMissing(t *testing.T, scope *functionScope, name string) {
	t.Helper()
	if _, ok := scope.variable(name); ok {
		t.Fatalf("scope unexpectedly contains variable %q", name)
	}
}

func assertCloneBudgetProperty(t *testing.T, scope *functionScope, name, want string) {
	t.Helper()
	typ, ok := scope.property(name)
	if !ok {
		t.Fatalf("scope is missing property %q", name)
	}
	if got := typ.String(); got != want {
		t.Fatalf("property %q = %q, want %q", name, got, want)
	}
}

func assertCloneBudgetPropertyMissing(t *testing.T, scope *functionScope, name string) {
	t.Helper()
	if _, ok := scope.property(name); ok {
		t.Fatalf("scope unexpectedly contains property %q", name)
	}
}
