package analyse

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/lexer"
	"github.com/ayanozturk/go-php-parser/parser"
)

func TestFunctionScopeCloneSharesReadOnlyLayers(t *testing.T) {
	original := seededFunctionScope()
	clone := original.clone()

	if clone == nil {
		t.Fatal("clone returned nil")
	}
	if clone == original {
		t.Fatal("clone returned the original scope")
	}
	if original.variablesOwned || clone.variablesOwned {
		t.Fatalf("variable layers are not marked shared after clone: originalOwned=%v cloneOwned=%v", original.variablesOwned, clone.variablesOwned)
	}
	if original.propertiesOwned || clone.propertiesOwned {
		t.Fatalf("property layers are not marked shared after clone: originalOwned=%v cloneOwned=%v", original.propertiesOwned, clone.propertiesOwned)
	}
	if clone.variables != original.variables {
		t.Fatal("read-only clone did not share the variable layer")
	}
	if clone.properties != original.properties {
		t.Fatal("read-only clone did not share the property layer")
	}
	assertScopeVariable(t, clone, "seed", "int")
	assertScopeProperty(t, clone, "state", "string")

	assertFunctionScopeLayersShareBaseMaps(t, original, clone)

	allocs := testing.AllocsPerRun(100, func() {
		readOnly := original.clone()
		if typ, ok := readOnly.variable("seed"); !ok || typ.IsEmpty() {
			t.Fatal("read-only clone lost seeded variable state")
		}
		if typ, ok := readOnly.property("state"); !ok || typ.IsEmpty() {
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

	assertScopeVariable(t, clone, "seed", "bool")
	assertScopeVariable(t, clone, "branch", "float")
	assertScopeVariable(t, original, "seed", "int")
	assertScopeVariableMissing(t, original, "branch")
	if clone.variables == original.variables || clone.variables.parent != original.variables {
		t.Fatal("clone variable write did not add a delta layer")
	}
	if !clone.variablesOwned || original.variablesOwned {
		t.Fatalf("variable ownership after clone write = clone=%v original=%v", clone.variablesOwned, original.variablesOwned)
	}
	if clone.properties != original.properties {
		t.Fatal("unmodified property layer detached during variable write")
	}
}

func TestFunctionScopeCloneFirstPropertyWriteIsolatesClone(t *testing.T) {
	original := seededFunctionScope()
	clone := original.clone()

	clone.setProperty("state", ParseType("bool"))
	clone.setProperty("branch", ParseType("float"))

	assertScopeProperty(t, clone, "state", "bool")
	assertScopeProperty(t, clone, "branch", "float")
	assertScopeProperty(t, original, "state", "string")
	assertScopePropertyMissing(t, original, "branch")
	if clone.properties == original.properties || clone.properties.parent != original.properties {
		t.Fatal("clone property write did not add a delta layer")
	}
	if !clone.propertiesOwned || original.propertiesOwned {
		t.Fatalf("property ownership after clone write = clone=%v original=%v", clone.propertiesOwned, original.propertiesOwned)
	}
	if clone.variables != original.variables {
		t.Fatal("unmodified variable layer detached during property write")
	}
}

func TestFunctionScopeOriginalWriteAfterCloneIsolatesOriginal(t *testing.T) {
	original := seededFunctionScope()
	clone := original.clone()

	original.setVariable("seed", ParseType("float"))
	original.setVariable("originalOnly", ParseType("bool"))
	original.setProperty("state", ParseType("bool"))
	original.setProperty("originalOnly", ParseType("int"))

	assertScopeVariable(t, original, "seed", "float")
	assertScopeVariable(t, original, "originalOnly", "bool")
	assertScopeProperty(t, original, "state", "bool")
	assertScopeProperty(t, original, "originalOnly", "int")
	assertScopeVariable(t, clone, "seed", "int")
	assertScopeProperty(t, clone, "state", "string")
	assertScopeVariableMissing(t, clone, "originalOnly")
	assertScopePropertyMissing(t, clone, "originalOnly")
	if original.variables == clone.variables || original.variables.parent != clone.variables {
		t.Fatal("original variable write did not add a delta layer")
	}
	if original.properties == clone.properties || original.properties.parent != clone.properties {
		t.Fatal("original property write did not add a delta layer")
	}
	if !original.variablesOwned || clone.variablesOwned {
		t.Fatalf("variable ownership after original write = original=%v clone=%v", original.variablesOwned, clone.variablesOwned)
	}
	if !original.propertiesOwned || clone.propertiesOwned {
		t.Fatalf("property ownership after original write = original=%v clone=%v", original.propertiesOwned, clone.propertiesOwned)
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

	assertScopeVariable(t, left, "branch", "bool")
	assertScopeVariable(t, right, "branch", "float")
	assertScopeProperty(t, parent, "branch", "int")
	assertScopeVariableMissing(t, parent, "branch")
	assertScopeVariableMissing(t, left, "rootOnly")
	assertScopeVariableMissing(t, right, "rootOnly")
	assertScopePropertyMissing(t, left, "branch")
	assertScopePropertyMissing(t, right, "branch")
	assertScopeVariable(t, root, "seed", "int")
	assertScopeVariableMissing(t, root, "branch")
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

func TestFunctionScopeClearMissingCallableReturnPreservesSharing(t *testing.T) {
	root := &functionScope{callableReturns: map[string]Type{"factory": ParseType("InitialService")}}
	clone := root.clone()
	before := functionScopeMapPointer(clone.callableReturns)

	clone.clearCallableReturn("missing")

	if !root.callablesShared || !clone.callablesShared {
		t.Fatalf("missing callable return clear changed sharing flags: root=%v clone=%v", root.callablesShared, clone.callablesShared)
	}
	if got := functionScopeMapPointer(clone.callableReturns); got != before {
		t.Fatalf("missing callable return clear detached backing map: before=%x after=%x", before, got)
	}
	assertScopeType(t, root.callableReturns, "factory", "InitialService")
	assertScopeType(t, clone.callableReturns, "factory", "InitialService")
}

func TestFunctionScopeClassStringMetadataClonesRemainIndependent(t *testing.T) {
	root := &functionScope{genericContext: map[string]GenericInstance{
		"class": {ClassName: "class-string", TypeArguments: []string{"InitialService"}},
	}}
	left := root.clone()
	right := root.clone()
	shared := functionScopeMapPointer(root.genericContext)

	if !root.genericContextShared || !left.genericContextShared || !right.genericContextShared {
		t.Fatalf("generic context maps are not marked shared after clone: root=%v left=%v right=%v", root.genericContextShared, left.genericContextShared, right.genericContextShared)
	}
	if functionScopeMapPointer(root.genericContext) != functionScopeMapPointer(left.genericContext) || functionScopeMapPointer(root.genericContext) != functionScopeMapPointer(right.genericContext) {
		t.Fatal("read-only generic context clones did not share their backing map")
	}

	right.clearGenericContext("missing")
	if got := functionScopeMapPointer(right.genericContext); got != shared || !right.genericContextShared {
		t.Fatalf("missing generic context clear detached shared map: got=%x want=%x shared=%v", got, shared, right.genericContextShared)
	}

	root.setGenericContext("rootOnly", GenericInstance{ClassName: "class-string", TypeArguments: []string{"RootService"}})
	left.clearGenericContext("class")
	left.setGenericContext("left", GenericInstance{ClassName: "class-string", TypeArguments: []string{"LeftService"}})

	if target, ok := classStringTarget(root, "class"); !ok || target.String() != "InitialService" {
		t.Fatalf("root class-string target changed through clone: %q, %v", target.String(), ok)
	}
	if target, ok := classStringTarget(right, "class"); !ok || target.String() != "InitialService" {
		t.Fatalf("sibling class-string target changed through clone: %q, %v", target.String(), ok)
	}
	if _, ok := left.genericContext["class"]; ok {
		t.Fatal("left clone retained deleted class-string metadata")
	}
	if _, ok := left.genericContext["rootOnly"]; ok {
		t.Fatal("root generic context write leaked into left clone")
	}
	if _, ok := right.genericContext["rootOnly"]; ok {
		t.Fatal("root generic context write leaked into right sibling")
	}
	if functionScopeMapPointer(left.genericContext) == shared || functionScopeMapPointer(root.genericContext) == shared {
		t.Fatal("generic context write or delete did not detach the modified scope")
	}
	if functionScopeMapPointer(right.genericContext) != shared {
		t.Fatal("read-only sibling detached from the original generic context map")
	}
}

func TestFunctionScopeGenericContextCopiesInputTypeArguments(t *testing.T) {
	scope := &functionScope{}
	typeArguments := []string{"InitialService"}
	scope.setGenericContext("class", GenericInstance{ClassName: "class-string", TypeArguments: typeArguments})

	typeArguments[0] = "MutatedService"
	if target, ok := classStringTarget(scope, "class"); !ok || target.String() != "InitialService" {
		t.Fatalf("generic context retained caller-owned type arguments: %q, %v", target.String(), ok)
	}
}

func TestFunctionScopeArrayShapeCallablesClonesRemainIndependent(t *testing.T) {
	root := &functionScope{arrayShapeCallables: map[string]map[string]arrayShapeField{
		"factories": {"service": {callable: ParseType("InitialService")}},
	}}
	parent := root.clone()
	left := parent.clone()
	right := parent.clone()

	left.setArrayShapeCallables("factories", map[string]arrayShapeField{"service": {callable: ParseType("LeftService")}})
	left.setArrayShapeCallables("leftOnly", map[string]arrayShapeField{"service": {callable: ParseType("bool")}})
	right.setArrayShapeCallables("factories", map[string]arrayShapeField{"service": {callable: ParseType("RightService")}})
	parent.clearArrayShapeCallables("factories")

	if got := root.arrayShapeCallables["factories"]["service"].callable.String(); got != "InitialService" {
		t.Fatalf("root array-shape callable changed through clone: %q", got)
	}
	if got := left.arrayShapeCallables["factories"]["service"].callable.String(); got != "LeftService" {
		t.Fatalf("left array-shape callable = %q, want LeftService", got)
	}
	if got := right.arrayShapeCallables["factories"]["service"].callable.String(); got != "RightService" {
		t.Fatalf("right array-shape callable = %q, want RightService", got)
	}
	if _, ok := parent.arrayShapeCallables["factories"]; ok {
		t.Fatal("parent array-shape deletion leaked into root or siblings")
	}
	if _, ok := left.arrayShapeCallables["rightOnly"]; ok {
		t.Fatal("right sibling array-shape leaked into left sibling")
	}
	if _, ok := right.arrayShapeCallables["leftOnly"]; ok {
		t.Fatal("left sibling array-shape leaked into right sibling")
	}
}

func TestFunctionScopeClearMissingArrayShapeCallablePreservesSharing(t *testing.T) {
	root := &functionScope{arrayShapeCallables: map[string]map[string]arrayShapeField{
		"factories": {"service": {callable: ParseType("InitialService")}},
	}}
	clone := root.clone()
	before := functionScopeMapPointer(clone.arrayShapeCallables)

	clone.clearArrayShapeCallables("missing")

	if !root.arrayShapesShared || !clone.arrayShapesShared {
		t.Fatalf("missing array-shape clear changed sharing flags: root=%v clone=%v", root.arrayShapesShared, clone.arrayShapesShared)
	}
	if got := functionScopeMapPointer(clone.arrayShapeCallables); got != before {
		t.Fatalf("missing array-shape clear detached backing map: before=%x after=%x", before, got)
	}
	if got := clone.arrayShapeCallables["factories"]["service"].callable.String(); got != "InitialService" {
		t.Fatalf("missing array-shape clear changed callable: %q", got)
	}
}

func TestFunctionScopeArrayIndexKeysCloneIndependently(t *testing.T) {
	root := &functionScope{arrayIndexKeys: map[string][]string{"key": {"service"}}}
	parent := root.clone()
	left := parent.clone()
	right := parent.clone()
	shared := functionScopeMapPointer(root.arrayIndexKeys)

	if !root.arrayIndexKeysShared || !parent.arrayIndexKeysShared || !left.arrayIndexKeysShared || !right.arrayIndexKeysShared {
		t.Fatal("array-index key maps are not marked shared after clone")
	}
	if functionScopeMapPointer(parent.arrayIndexKeys) != shared || functionScopeMapPointer(left.arrayIndexKeys) != shared || functionScopeMapPointer(right.arrayIndexKeys) != shared {
		t.Fatal("read-only array-index clones did not share their backing map")
	}
	right.clearArrayIndexKeys("missing")
	if got := functionScopeMapPointer(right.arrayIndexKeys); got != shared || !right.arrayIndexKeysShared {
		t.Fatalf("missing array-index clear detached shared map: got=%x want=%x shared=%v", got, shared, right.arrayIndexKeysShared)
	}

	left.setArrayIndexKeys("key", []string{"left"})
	left.setArrayIndexKeys("leftOnly", []string{"inner"})
	right.setArrayIndexKeys("key", []string{"right"})
	parent.clearArrayIndexKeys("key")

	if got := strings.Join(root.arrayIndexKeys["key"], ","); got != "service" {
		t.Fatalf("root array-index keys changed through clone: %q", got)
	}
	if got := strings.Join(left.arrayIndexKeys["key"], ","); got != "left" {
		t.Fatalf("left array-index keys = %q, want left", got)
	}
	if got := strings.Join(right.arrayIndexKeys["key"], ","); got != "right" {
		t.Fatalf("right array-index keys = %q, want right", got)
	}
	if _, ok := parent.arrayIndexKeys["key"]; ok {
		t.Fatal("parent array-index deletion leaked into root or siblings")
	}
	if _, ok := left.arrayIndexKeys["rightOnly"]; ok {
		t.Fatal("right sibling array-index leaked into left sibling")
	}
	if _, ok := right.arrayIndexKeys["leftOnly"]; ok {
		t.Fatal("left sibling array-index leaked into right sibling")
	}
	if functionScopeMapPointer(root.arrayIndexKeys) != shared {
		t.Fatal("root array-index map detached without a write")
	}
	for name, scope := range map[string]*functionScope{"parent": parent, "left": left, "right": right} {
		if functionScopeMapPointer(scope.arrayIndexKeys) == shared {
			t.Fatalf("%s array-index mutation did not detach its map", name)
		}
	}
}

func TestFunctionScopeArrayIndexKeysCopyInputAndLookupResult(t *testing.T) {
	scope := &functionScope{}
	keys := []string{"service"}
	scope.setArrayIndexKeys("key", keys)
	keys[0] = "mutated"

	lookup := resolveArrayIndex(&ast.VariableNode{Name: "key"}, scope)
	if got := strings.Join(lookup.keys, ","); got != "service" {
		t.Fatalf("array-index keys retained caller mutation: %q", got)
	}
	lookup.keys[0] = "changed"
	if got := strings.Join(scope.arrayIndexKeys["key"], ","); got != "service" {
		t.Fatalf("array-index lookup exposed stored slice: %q", got)
	}
}

func TestFunctionScopeReadOnlyCloneDoesNotCopyMetadataMaps(t *testing.T) {
	root := &functionScope{
		arrayIndexKeys: map[string][]string{"key": {"service"}},
		genericContext: map[string]GenericInstance{"class": {ClassName: "class-string", TypeArguments: []string{"Service"}}},
	}
	arrayIndexKeys := functionScopeMapPointer(root.arrayIndexKeys)
	genericContext := functionScopeMapPointer(root.genericContext)

	allocs := testing.AllocsPerRun(100, func() {
		clone := root.clone()
		benchmarkFunctionScopeSink = clone
		if functionScopeMapPointer(clone.arrayIndexKeys) != arrayIndexKeys || functionScopeMapPointer(clone.genericContext) != genericContext {
			t.Fatal("read-only clone copied metadata maps")
		}
	})
	if allocs > 1 {
		t.Fatalf("read-only metadata clone allocations/run = %.2f, want at most the scope allocation", allocs)
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
	if first.propertiesOwned || second.propertiesOwned {
		t.Fatalf("new scopes should share cached class properties: firstOwned=%v secondOwned=%v", first.propertiesOwned, second.propertiesOwned)
	}
	if first.properties == second.properties {
		t.Fatal("new scopes unexpectedly share mutable property layer")
	}
	if functionScopeLayerBaseMapPointer(first.properties) != functionScopeLayerBaseMapPointer(second.properties) {
		t.Fatal("new scopes did not share cached class property map")
	}
	assertScopeProperty(t, first, "state", "string")
	assertScopeProperty(t, second, "count", "int")

	first.setProperty("state", ParseType("bool"))
	if !first.propertiesOwned {
		t.Fatal("first scope remained unowned after its first property write")
	}
	if second.propertiesOwned {
		t.Fatal("second scope detached before it was written")
	}
	assertScopeProperty(t, first, "state", "bool")
	assertScopeProperty(t, second, "state", "string")
	if first.properties.parent == nil || functionScopeLayerBaseMapPointer(first.properties) != functionScopeLayerBaseMapPointer(second.properties) {
		t.Fatal("first property write did not retain the cached base layer")
	}

	second.setProperty("count", ParseType("string"))
	if !second.propertiesOwned {
		t.Fatal("second scope remained unowned after its first property write")
	}
	assertScopeProperty(t, second, "count", "string")
	assertScopeProperty(t, first, "count", "int")

	third := newFunctionScopeWithContext(ctx, class, method, typeCtx)
	if third.propertiesOwned {
		t.Fatal("new scope did not share the cached class properties after sibling writes")
	}
	assertScopeProperty(t, third, "state", "string")
	assertScopeProperty(t, third, "count", "int")
	if got, want := functionScopeLayerBaseMapPointer(third.properties), functionScopeLayerBaseMapPointer(first.properties); got != want {
		t.Fatalf("new scope cached class property base = %x, want %x", got, want)
	}
}

func TestFunctionScopeCloneFirstWritesUseSmallDeltaLayers(t *testing.T) {
	const baseEntries = 256
	variables := make(map[string]Type, baseEntries)
	properties := make(map[string]Type, baseEntries)
	for i := 0; i < baseEntries; i++ {
		name := fmt.Sprintf("entry%d", i)
		variables[name] = ParseType("int")
		properties[name] = ParseType("string")
	}
	original := &functionScope{
		variables:       rootScopeTypeLayer(variables),
		properties:      rootScopeTypeLayer(properties),
		variablesOwned:  true,
		propertiesOwned: true,
	}
	clone := original.clone()
	variableBase := original.variables
	propertyBase := original.properties

	clone.setVariable("written", ParseType("bool"))
	clone.setProperty("written", ParseType("float"))

	if clone.variables.parent != variableBase || clone.properties.parent != propertyBase {
		t.Fatal("first writes did not retain the original layers as their bases")
	}
	if got := scopeTypeLayerLocalSize(clone.variables); got != 1 {
		t.Fatalf("first variable write materialized %d entries, want one delta entry", got)
	}
	if got := scopeTypeLayerLocalSize(clone.properties); got != 1 {
		t.Fatalf("first property write materialized %d entries, want one delta entry", got)
	}
	if got := len(variableBase.values); got != baseEntries {
		t.Fatalf("variable base changed size after clone write: got %d, want %d", got, baseEntries)
	}
	if got := len(propertyBase.values); got != baseEntries {
		t.Fatalf("property base changed size after clone write: got %d, want %d", got, baseEntries)
	}
	assertScopeVariable(t, clone, "entry0", "int")
	assertScopeProperty(t, clone, "entry0", "string")
}

func TestFunctionScopeSequentialCloneWritesKeepLayerDepthBounded(t *testing.T) {
	scope := seededFunctionScope()
	const writeCount = 256
	for i := 0; i < writeCount; i++ {
		previous := scope.clone()
		name := fmt.Sprintf("branch%d", i)
		scope.setVariable(name, ParseType("bool"))
		scope.setProperty(name, ParseType("float"))
		assertScopeVariableMissing(t, previous, name)
		assertScopePropertyMissing(t, previous, name)
		if scope.variables.depth > maxScopeTypeLayerDepth || scope.properties.depth > maxScopeTypeLayerDepth {
			t.Fatalf("write %d exceeded layer depth bound: variables=%d properties=%d", i, scope.variables.depth, scope.properties.depth)
		}
	}
	assertScopeVariable(t, scope, "seed", "int")
	assertScopeVariable(t, scope, "branch255", "bool")
	assertScopeProperty(t, scope, "state", "string")
	assertScopeProperty(t, scope, "branch255", "float")
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
		variables:       rootScopeTypeLayer(map[string]Type{"seed": ParseType("int")}),
		properties:      rootScopeTypeLayer(map[string]Type{"state": ParseType("string")}),
		variablesOwned:  true,
		propertiesOwned: true,
	}
}

func assertScopeVariable(t *testing.T, scope *functionScope, name, want string) {
	t.Helper()
	typ, ok := scope.variable(name)
	if !ok {
		t.Fatalf("scope is missing variable %q", name)
	}
	if got := typ.String(); got != want {
		t.Fatalf("variable %q = %q, want %q", name, got, want)
	}
}

func assertScopeProperty(t *testing.T, scope *functionScope, name, want string) {
	t.Helper()
	typ, ok := scope.property(name)
	if !ok {
		t.Fatalf("scope is missing property %q", name)
	}
	if got := typ.String(); got != want {
		t.Fatalf("property %q = %q, want %q", name, got, want)
	}
}

func assertScopeVariableMissing(t *testing.T, scope *functionScope, name string) {
	t.Helper()
	if _, ok := scope.variable(name); ok {
		t.Fatalf("scope unexpectedly contains variable %q", name)
	}
}

func assertScopePropertyMissing(t *testing.T, scope *functionScope, name string) {
	t.Helper()
	if _, ok := scope.property(name); ok {
		t.Fatalf("scope unexpectedly contains property %q", name)
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

func assertFunctionScopeLayersShareBaseMaps(t *testing.T, first, second *functionScope) {
	t.Helper()
	if got, want := functionScopeLayerBaseMapPointer(first.variables), functionScopeLayerBaseMapPointer(second.variables); got != want {
		t.Fatalf("variable base map pointers differ: %x != %x", got, want)
	}
	if got, want := functionScopeLayerBaseMapPointer(first.properties), functionScopeLayerBaseMapPointer(second.properties); got != want {
		t.Fatalf("property base map pointers differ: %x != %x", got, want)
	}
}

func functionScopeLayerBaseMapPointer(layer *scopeTypeLayer) uintptr {
	if layer == nil {
		return 0
	}
	for layer.parent != nil {
		layer = layer.parent
	}
	return functionScopeMapPointer(layer.values)
}

func scopeTypeLayerLocalSize(layer *scopeTypeLayer) int {
	if layer == nil {
		return 0
	}
	size := len(layer.values)
	if layer.hasOne {
		size++
	}
	return size
}

func functionScopeMapPointer(values any) uintptr {
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
