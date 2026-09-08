package analyse

import (
	"strings"
)

func (idx *ProjectIndex) ClassExists(name string) bool {
	_, ok := idx.ResolveClass(name)
	return ok
}

func (idx *ProjectIndex) FunctionExists(name string) bool {
	_, ok := idx.ResolveFunction(name)
	return ok
}

func (idx *ProjectIndex) ConstantExists(name string) bool {
	if _, ok := idx.Constants[indexKey(name)]; ok {
		return true
	}
	if className, constName, ok := strings.Cut(name, "::"); ok {
		_, ok := idx.ResolveConstant(className, constName)
		return ok
	}
	return false
}

func (idx *ProjectIndex) ResolveClass(name string) (ResolvedClass, bool) {
	key := indexKey(name)
	if class, ok := idx.Classes[key]; ok {
		return class, true
	}
	if short := unqualifiedName(key); short != key && isBuiltinClassName(short) {
		class, ok := idx.Classes[short]
		return class, ok
	}
	if class, ok := idx.resolveKnownClassSuffix(key); ok {
		return class, true
	}
	return ResolvedClass{}, false
}

func (idx *ProjectIndex) ResolveMethod(className, methodName string) (ResolvedMethod, bool) {
	if method, found, safe := idx.resolveMethodFromNonGenericLineage(className, methodName); safe {
		if !found {
			return ResolvedMethod{}, false
		}
		// Public callers may mutate the returned parameter metadata. Keep the
		// allocation-light traversal internal, then detach only the mutable slice.
		method.Params = append([]ResolvedParam(nil), method.Params...)
		return method, true
	}
	return idx.resolveMethodWithTemplates(className, methodName, nil, make(map[string]struct{}))
}

func (idx *ProjectIndex) resolveMethodFromNonGenericLineage(className, methodName string) (ResolvedMethod, bool, bool) {
	if idx == nil || idx.classLineages == nil {
		return ResolvedMethod{}, false, false
	}
	class, ok := idx.ResolveClass(className)
	if !ok {
		return ResolvedMethod{}, false, true
	}
	lineage, ok := idx.classLineages[indexKey(class.Name)]
	if !ok {
		return ResolvedMethod{}, false, false
	}
	lowerMethodName := asciiLowerIdent(methodName)
	for _, candidate := range lineage {
		resolved, ok := idx.Classes[indexKey(candidate)]
		if !ok || len(resolved.GenericParents) != 0 {
			return ResolvedMethod{}, false, false
		}
		if method, found := idx.Methods[indexKey(resolved.Name)][lowerMethodName]; found {
			method.DeclaringClass = resolved.Name
			return method, true, true
		}
	}
	return ResolvedMethod{}, false, true
}

// ResolveMethodWithGenerics resolves a method on a generic class instance.
// className is the fully qualified class name (e.g., "Repository").
// typeArguments are the generic type arguments (e.g., ["User"] for Repository<User>).
func (idx *ProjectIndex) ResolveMethodWithGenerics(className, methodName string, typeArguments []string) (ResolvedMethod, bool) {
	class, ok := idx.ResolveClass(className)
	if !ok {
		return ResolvedMethod{}, false
	}
	if len(class.TemplateParams) == 0 {
		if len(typeArguments) == 0 {
			return idx.ResolveMethod(className, methodName)
		}
		bindings := templateBindingsFromAncestors(idx, className, typeArguments)
		if len(bindings) == 0 {
			return idx.ResolveMethod(className, methodName)
		}
		return idx.resolveMethodWithTemplates(className, methodName, bindings, make(map[string]struct{}))
	}
	if len(typeArguments) == 0 {
		return idx.ResolveMethod(className, methodName)
	}

	// Build bindings: T -> User, K -> string, etc.
	bindings := make(map[string]string, len(class.TemplateParams))
	for i, param := range class.TemplateParams {
		if i < len(typeArguments) {
			bindings[param] = typeArguments[i]
		}
	}

	return idx.resolveMethodWithTemplates(className, methodName, bindings, make(map[string]struct{}))
}

func templateBindingsFromAncestors(idx *ProjectIndex, className string, typeArguments []string) map[string]string {
	if idx == nil || className == "" || len(typeArguments) == 0 {
		return nil
	}
	for _, candidate := range idx.classLineage(className) {
		class, ok := idx.ResolveClass(candidate)
		if !ok || len(class.TemplateParams) == 0 {
			continue
		}
		bindings := make(map[string]string, len(class.TemplateParams))
		for i, param := range class.TemplateParams {
			if i < len(typeArguments) {
				bindings[param] = typeArguments[i]
			}
		}
		if len(bindings) > 0 {
			return bindings
		}
	}
	return nil
}

func (idx *ProjectIndex) methodReferenceParams(className, methodName string) ([]ResolvedParam, bool) {
	var seen [32]string
	return idx.methodReferenceParamsSeen(className, methodName, seen[:0])
}

func (idx *ProjectIndex) methodReferenceParamsSeen(className, methodName string, seen []string) ([]ResolvedParam, bool) {
	if idx == nil {
		return nil, false
	}
	if len(seen) == cap(seen) {
		method, ok := idx.resolveMethodView(className, methodName)
		return method.Params, ok
	}
	class, ok := idx.ResolveClass(className)
	if !ok {
		return nil, false
	}
	key := indexKey(class.Name)
	for _, visited := range seen {
		if visited == key {
			return nil, false
		}
	}
	seen = append(seen, key)
	if method, found := idx.Methods[key][asciiLowerIdent(methodName)]; found {
		return method.Params, true
	}
	for _, parentName := range class.Extends {
		if params, found := idx.methodReferenceParamsSeen(parentName, methodName, seen); found {
			return params, true
		}
	}
	for _, parentName := range class.Implements {
		if params, found := idx.methodReferenceParamsSeen(parentName, methodName, seen); found {
			return params, true
		}
	}
	for _, parentName := range class.Traits {
		if params, found := idx.methodReferenceParamsSeen(parentName, methodName, seen); found {
			return params, true
		}
	}
	return nil, false
}

func (idx *ProjectIndex) ResolveOwnMethod(className, methodName string) (ResolvedMethod, bool) {
	method, ok := idx.resolveOwnMethodView(className, methodName)
	if !ok {
		return ResolvedMethod{}, false
	}
	method.Params = append([]ResolvedParam(nil), method.Params...)
	return method, true
}

func (idx *ProjectIndex) resolveOwnMethodView(className, methodName string) (ResolvedMethod, bool) {
	if idx == nil {
		return ResolvedMethod{}, false
	}
	class, ok := idx.ResolveClass(className)
	if !ok {
		return ResolvedMethod{}, false
	}
	method, ok := idx.Methods[indexKey(class.Name)][asciiLowerIdent(methodName)]
	if !ok {
		return ResolvedMethod{}, false
	}
	method.DeclaringClass = class.Name
	return method, true
}

func (idx *ProjectIndex) resolveMethodView(className, methodName string) (ResolvedMethod, bool) {
	if idx == nil {
		return ResolvedMethod{}, false
	}
	var seen [32]string
	return idx.resolveMethodViewSeen(className, asciiLowerIdent(methodName), seen[:0])
}

func (idx *ProjectIndex) resolveMethodViewSeen(className, lowerMethodName string, seen []string) (ResolvedMethod, bool) {
	if len(seen) == cap(seen) {
		visited := make(map[string]struct{}, len(seen)+8)
		for _, key := range seen {
			visited[key] = struct{}{}
		}
		return idx.resolveMethodViewMapped(className, lowerMethodName, visited)
	}
	class, ok := idx.ResolveClass(className)
	if !ok {
		return ResolvedMethod{}, false
	}
	key := indexKey(class.Name)
	for _, visited := range seen {
		if visited == key {
			return ResolvedMethod{}, false
		}
	}
	seen = append(seen, key)
	if method, found := idx.Methods[key][lowerMethodName]; found {
		method.DeclaringClass = class.Name
		return method, true
	}
	for _, parentName := range class.Extends {
		if method, found := idx.resolveMethodViewSeen(parentName, lowerMethodName, seen); found {
			return method, true
		}
	}
	for _, parentName := range class.Implements {
		if method, found := idx.resolveMethodViewSeen(parentName, lowerMethodName, seen); found {
			return method, true
		}
	}
	for _, parentName := range class.Traits {
		if method, found := idx.resolveMethodViewSeen(parentName, lowerMethodName, seen); found {
			return method, true
		}
	}
	return ResolvedMethod{}, false
}

func (idx *ProjectIndex) resolveMethodViewMapped(className, lowerMethodName string, seen map[string]struct{}) (ResolvedMethod, bool) {
	class, ok := idx.ResolveClass(className)
	if !ok {
		return ResolvedMethod{}, false
	}
	key := indexKey(class.Name)
	if _, exists := seen[key]; exists {
		return ResolvedMethod{}, false
	}
	seen[key] = struct{}{}
	if method, found := idx.Methods[key][lowerMethodName]; found {
		method.DeclaringClass = class.Name
		return method, true
	}
	for _, parentName := range class.Extends {
		if method, found := idx.resolveMethodViewMapped(parentName, lowerMethodName, seen); found {
			return method, true
		}
	}
	for _, parentName := range class.Implements {
		if method, found := idx.resolveMethodViewMapped(parentName, lowerMethodName, seen); found {
			return method, true
		}
	}
	for _, parentName := range class.Traits {
		if method, found := idx.resolveMethodViewMapped(parentName, lowerMethodName, seen); found {
			return method, true
		}
	}
	return ResolvedMethod{}, false
}

func (idx *ProjectIndex) MethodsDeclaredBy(className string) []ResolvedMethod {
	if idx == nil {
		return nil
	}
	methods := idx.methodsDeclaredView(className)
	result := make([]ResolvedMethod, len(methods))
	for i := range methods {
		result[i] = methods[i]
		result[i].Params = append([]ResolvedParam(nil), methods[i].Params...)
	}
	return result
}

func (idx *ProjectIndex) rangeMethodsDeclaredBy(className string, visit func(ResolvedMethod) bool) {
	if idx == nil || visit == nil {
		return
	}
	for _, method := range idx.methodsDeclaredView(className) {
		if !visit(method) {
			return
		}
	}
}

func (idx *ProjectIndex) methodsDeclaredView(className string) []ResolvedMethod {
	if idx == nil {
		return nil
	}
	class, ok := idx.ResolveClass(className)
	if !ok {
		return nil
	}
	key := indexKey(class.Name)
	if idx.methodsDeclared != nil {
		return idx.methodsDeclared[key]
	}
	// Mutable indexes constructed directly retain compatibility. Immutable
	// indexes returned by BuildProjectIndex always use the precomputed path.
	return buildMethodsDeclaredView(idx, key, class.Name)
}

func (idx *ProjectIndex) resolveMethodWithTemplates(className, methodName string, bindings map[string]string, seen map[string]struct{}) (ResolvedMethod, bool) {
	class, ok := idx.ResolveClass(className)
	if !ok {
		return ResolvedMethod{}, false
	}
	key := indexKey(class.Name)
	if _, exists := seen[key]; exists {
		return ResolvedMethod{}, false
	}
	seen[key] = struct{}{}
	defer delete(seen, key)
	if method, found := idx.Methods[key][asciiLowerIdent(methodName)]; found {
		// ResolvedMethod is returned by value, but Params is a slice. Clone it
		// before applying call-specific generic bindings so resolution cannot
		// mutate the project index or race with concurrent snapshot readers.
		method.Params = append([]ResolvedParam(nil), method.Params...)
		method.DeclaringClass = class.Name
		method.ReturnType = ApplyTemplateBindings(method.ReturnType, bindings)
		for i := range method.Params {
			method.Params[i].Type = ApplyTemplateBindings(method.Params[i].Type, bindings)
		}
		return method, true
	}
	parents := append(append(append([]string(nil), class.Extends...), class.Implements...), class.Traits...)
	for _, parentName := range parents {
		parent, parentOK := idx.ResolveClass(parentName)
		if !parentOK {
			continue
		}
		parentBindings := map[string]string(nil)
		if relation, relationOK := genericRelationTo(class, parentName); relationOK {
			parentBindings = bindGenericParent(parent, relation, bindings)
		} else if len(bindings) > 0 {
			parentBindings = bindings
		}
		if method, found := idx.resolveMethodWithTemplates(parent.Name, methodName, parentBindings, seen); found {
			return method, true
		}
	}
	return ResolvedMethod{}, false
}

func (idx *ProjectIndex) ResolveProperty(className, propertyName string) (ResolvedProperty, bool) {
	for _, candidate := range idx.classLineage(className) {
		properties := idx.Properties[indexKey(candidate)]
		if properties == nil {
			continue
		}
		if property, ok := properties[asciiLowerIdent(strings.TrimPrefix(propertyName, "$"))]; ok {
			return property, true
		}
	}
	return ResolvedProperty{}, false
}

func (idx *ProjectIndex) ResolveConstant(className, constantName string) (ResolvedConstant, bool) {
	for _, candidate := range idx.classLineage(className) {
		constants := idx.ClassConsts[indexKey(candidate)]
		if constants == nil {
			continue
		}
		if constant, ok := constants[asciiLowerIdent(constantName)]; ok {
			constant.DeclaringClass = candidate
			return constant, true
		}
	}
	return ResolvedConstant{}, false
}

func (idx *ProjectIndex) ResolveOwnConstant(className, constantName string) (ResolvedConstant, bool) {
	if idx == nil {
		return ResolvedConstant{}, false
	}
	class, ok := idx.ResolveClass(className)
	if !ok {
		return ResolvedConstant{}, false
	}
	constant, ok := idx.ClassConsts[indexKey(class.Name)][asciiLowerIdent(constantName)]
	if !ok {
		return ResolvedConstant{}, false
	}
	constant.DeclaringClass = class.Name
	return constant, true
}

func (idx *ProjectIndex) DuplicateClasses(filename string) []DuplicateSymbol {
	if idx == nil {
		return nil
	}
	duplicates := make([]DuplicateSymbol, 0)
	for _, duplicate := range idx.Duplicates {
		if filename == "" || duplicate.File == filename {
			duplicates = append(duplicates, duplicate)
		}
	}
	return duplicates
}

func (idx *ProjectIndex) ResolveFunction(name string) (ResolvedFunction, bool) {
	fn, ok := idx.Functions[indexKey(name)]
	return fn, ok
}

func (idx *ProjectIndex) resolveFunctionView(name string) (ResolvedFunction, bool) {
	return idx.ResolveFunction(name)
}

func (idx *ProjectIndex) classLineage(className string) []string {
	if idx == nil {
		return nil
	}
	key := indexKey(className)
	if class, ok := idx.ResolveClass(className); ok {
		key = indexKey(class.Name)
	}
	if lineage, ok := idx.classLineages[key]; ok {
		return lineage
	}
	// Mutable indexes constructed directly retain compatibility. Immutable
	// indexes returned by BuildProjectIndex always use the precomputed path.
	return buildClassLineage(idx, className)
}

func buildClassLineageViews(idx *ProjectIndex) map[string][]string {
	views := make(map[string][]string, len(idx.Classes))
	for key, class := range idx.Classes {
		views[key] = buildClassLineage(idx, class.Name)
	}
	return views
}

func buildClassLineage(idx *ProjectIndex, className string) []string {
	var out []string
	seen := map[string]struct{}{}
	var walk func(string)
	walk = func(name string) {
		key := indexKey(name)
		if key == "" {
			return
		}
		class, ok := idx.ResolveClass(name)
		if ok {
			key = indexKey(class.Name)
			name = class.Name
		}
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		out = append(out, name)
		if !ok {
			return
		}
		for _, parent := range class.Extends {
			walk(parent)
		}
		for _, iface := range class.Implements {
			walk(iface)
		}
		for _, trait := range class.Traits {
			walk(trait)
		}
	}
	walk(className)
	return out
}

func (idx *ProjectIndex) resolveKnownClassSuffix(key string) (ResolvedClass, bool) {
	parts := strings.Split(strings.TrimPrefix(key, `\`), `\`)
	for i := 1; i < len(parts)-1; i++ {
		suffix := strings.Join(parts[i:], `\`)
		if class, ok := idx.Classes[suffix]; ok && strings.Contains(suffix, `\`) {
			return class, true
		}
	}
	return ResolvedClass{}, false
}
