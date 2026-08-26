package analyse

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"sort"
	"strings"
)

type ProjectIndex struct {
	Classes         map[string]ResolvedClass
	Methods         map[string]map[string]ResolvedMethod
	Properties      map[string]map[string]ResolvedProperty
	ClassConsts     map[string]map[string]ResolvedConstant
	Functions       map[string]ResolvedFunction
	Constants       map[string]struct{}
	FileTypes       map[string]fileTypeContext
	Duplicates      []DuplicateSymbol
	methodsDeclared map[string][]ResolvedMethod
	classLineages   map[string][]string
}

type DuplicateSymbol struct {
	File string
	Name string
	Pos  ast.Position
}

func NewProjectIndex() *ProjectIndex {
	idx := &ProjectIndex{
		Classes:     make(map[string]ResolvedClass),
		Methods:     make(map[string]map[string]ResolvedMethod),
		Properties:  make(map[string]map[string]ResolvedProperty),
		ClassConsts: make(map[string]map[string]ResolvedConstant),
		Functions:   make(map[string]ResolvedFunction),
		Constants:   make(map[string]struct{}),
		FileTypes:   make(map[string]fileTypeContext),
	}
	idx.seedBuiltins()
	return idx
}

// BuildProjectIndex indexes every parsed file into a single ProjectIndex.
// Files are processed in sorted filename order rather than native Go map
// iteration order (which is randomized per run): symbol registration below
// is order-dependent (addClass keeps the first definition and records
// later same-name definitions as duplicates; addFunction/addMethod/
// addProperty/addClassConstant let the last definition win), so an
// unsorted, randomized iteration order made duplicate-symbol resolution -
// and therefore some diagnostics computed relative to it - vary between
// otherwise-identical runs over the same corpus.
func BuildProjectIndex(parsed map[string][]ast.Node) *ProjectIndex {
	idx := NewProjectIndex()
	filenames := make([]string, 0, len(parsed))
	for filename := range parsed {
		filenames = append(filenames, filename)
	}
	sort.Strings(filenames)
	for _, filename := range filenames {
		nodes := parsed[filename]
		ft := collectFileTypeContext(nodes)
		idx.FileTypes[filename] = ft
		idx.indexNodes(filename, nodes, ft, "")
	}
	idx.methodsDeclared = buildMethodsDeclaredViews(idx)
	idx.classLineages = buildClassLineageViews(idx)
	return idx
}

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
	return idx.resolveMethodWithTemplates(className, methodName, nil, make(map[string]struct{}))
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
		method, ok := idx.ResolveMethod(className, methodName)
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
	if method, found := idx.Methods[key][strings.ToLower(methodName)]; found {
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
	return nil, false
}

func (idx *ProjectIndex) ResolveOwnMethod(className, methodName string) (ResolvedMethod, bool) {
	if idx == nil {
		return ResolvedMethod{}, false
	}
	class, ok := idx.ResolveClass(className)
	if !ok {
		return ResolvedMethod{}, false
	}
	method, ok := idx.Methods[indexKey(class.Name)][strings.ToLower(methodName)]
	if !ok {
		return ResolvedMethod{}, false
	}
	method.DeclaringClass = class.Name
	method.Params = append([]ResolvedParam(nil), method.Params...)
	return method, true
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
	if method, found := idx.Methods[key][strings.ToLower(methodName)]; found {
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
	parents := append(append([]string(nil), class.Extends...), class.Implements...)
	for _, parentName := range parents {
		parent, parentOK := idx.ResolveClass(parentName)
		if !parentOK {
			continue
		}
		parentBindings := map[string]string(nil)
		if relation, relationOK := genericRelationTo(class, parentName); relationOK {
			parentBindings = bindGenericParent(parent, relation, bindings)
		}
		if method, found := idx.resolveMethodWithTemplates(parent.Name, methodName, parentBindings, seen); found {
			return method, true
		}
	}
	return ResolvedMethod{}, false
}

func (idx *ProjectIndex) ResolveProperty(className, propertyName string) (ResolvedProperty, bool) {
	if class, ok := idx.ResolveClass(className); ok && class.Kind == "enum" && strings.EqualFold(strings.TrimPrefix(propertyName, "$"), "value") {
		return ResolvedProperty{ID: stableSymbolID("property", class.Name, "value"), DeclaringClass: class.Name, Name: "value", Visibility: "public", Readonly: true}, true
	}
	for _, candidate := range idx.classLineage(className) {
		properties := idx.Properties[indexKey(candidate)]
		if properties == nil {
			continue
		}
		if property, ok := properties[strings.ToLower(strings.TrimPrefix(propertyName, "$"))]; ok {
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
		if constant, ok := constants[strings.ToLower(constantName)]; ok {
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
	constant, ok := idx.ClassConsts[indexKey(class.Name)][strings.ToLower(constantName)]
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

func (idx *ProjectIndex) indexNodes(filename string, nodes []ast.Node, ft fileTypeContext, currentClass string) {
	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.NamespaceNode:
			nft := collectFileTypeContext(n.Body)
			if nft.namespace == "" {
				nft.namespace = n.Name
			}
			idx.indexNodes(filename, n.Body, nft, currentClass)
		case *ast.ClassNode:
			name := ft.resolveClassLike(n.Name)
			templates, genericParents := resolvedGenericMetadata(n.PHPDoc, ft)
			class := ResolvedClass{
				Name:                  name,
				Extends:               resolvedList(ft, optionalList(n.Extends)),
				Implements:            resolvedList(ft, n.Implements),
				TemplateParams:        templates,
				GenericParents:        genericParents,
				Traits:                traitUsesFromMembers(n.Properties, ft),
				Kind:                  "class",
				Final:                 strings.Contains(n.Modifier, "final"),
				Abstract:              strings.Contains(n.Modifier, "abstract"),
				Readonly:              strings.Contains(n.Modifier, "readonly"),
				ConsistentConstructor: hasPHPStanConsistentConstructorTag(n.PHPDoc),
			}
			idx.addClass(filename, class, n)
			idx.indexClassMembers(filename, name, n.Properties, n.Methods, n.Constants, ft, templates)
		case *ast.InterfaceNode:
			name := ft.resolveClassLike(n.Name)
			templates, genericParents := resolvedGenericMetadata(n.PHPDoc, ft)
			idx.addClass(filename, ResolvedClass{Name: name, Extends: resolvedList(ft, n.Extends), TemplateParams: templates, GenericParents: genericParents, Kind: "interface"}, n)
			idx.indexInterfaceMembers(filename, name, n.Members, ft, templates)
		case *ast.TraitNode:
			if n.Name != nil {
				name := ft.resolveClassLike(n.Name.Name)
				idx.addClass(filename, ResolvedClass{Name: name, Kind: "trait"}, n)
				idx.indexClassMembers(filename, name, n.Body, nil, nil, ft, nil)
			}
		case *ast.EnumNode:
			name := ft.resolveClassLike(n.Name)
			idx.addClass(filename, ResolvedClass{Name: name, Implements: resolvedList(ft, n.Implements), Kind: "enum", Final: true}, n)
			idx.indexClassMembers(filename, name, nil, n.Methods, nil, ft, nil)
			for _, enumCase := range n.Cases {
				idx.addClassConstant(name, ResolvedConstant{Name: enumCase.Name, DeclaringClass: name, Declaration: sourceLocation(filename, enumCase), Visibility: "public"})
			}
			declaration := sourceLocation(filename, n)
			idx.addMethod(name, ResolvedMethod{Name: "cases", DeclaringClass: name, Declaration: declaration, ReturnType: "array", Visibility: "public", IsStatic: true})
			idx.addMethod(name, ResolvedMethod{Name: "from", DeclaringClass: name, Declaration: declaration, ReturnType: name, Params: []ResolvedParam{{Name: "value"}}, Visibility: "public", IsStatic: true})
			idx.addMethod(name, ResolvedMethod{Name: "tryFrom", DeclaringClass: name, Declaration: declaration, ReturnType: "?" + name, Params: []ResolvedParam{{Name: "value"}}, Visibility: "public", IsStatic: true})
		case *ast.FunctionNode:
			if currentClass != "" {
				idx.addMethod(currentClass, methodFromFunction(filename, currentClass, n, ft, nil))
				continue
			}
			name := ft.resolveClassLike(n.Name)
			idx.addFunction(ResolvedFunction{Name: name, Declaration: sourceLocation(filename, n), ReturnType: normalizeTypeWithContext(n.ReturnType, ft), Params: paramsFromNodes(n.Params, ft)})
		case *ast.ConstantNode:
			idx.Constants[indexKey(ft.resolveClassLike(n.Name))] = struct{}{}
		}
	}
}

func (idx *ProjectIndex) indexClassMembers(filename, className string, properties, methods, constants []ast.Node, ft fileTypeContext, templateParams []string) {
	for _, propNode := range properties {
		switch p := propNode.(type) {
		case *ast.PropertyNode:
			idx.addProperty(className, ResolvedProperty{
				Declaration: sourceLocation(filename, p),
				Name:        p.Name,
				Type:        normalizeTypeWithContext(p.TypeHint, ft),
				Visibility:  defaultVisibility(p.Visibility),
				IsStatic:    p.IsStatic,
				Readonly:    p.IsReadonly,
			})
		case *ast.TraitUseNode:
			// Trait use is checked by level-0 rules; no index entry needed.
		case *ast.FunctionNode:
			idx.addMethod(className, methodFromFunction(filename, className, p, ft, templateParams))
		}
	}
	for _, methodNode := range methods {
		if fn, ok := methodNode.(*ast.FunctionNode); ok {
			idx.addMethod(className, methodFromFunction(filename, className, fn, ft, templateParams))
		}
	}
	for _, constNode := range constants {
		if c, ok := constNode.(*ast.ConstantNode); ok {
			idx.addClassConstant(className, constantFromNode(filename, className, c, ft))
		}
	}
}

func (idx *ProjectIndex) indexInterfaceMembers(filename, className string, members []ast.Node, ft fileTypeContext, templateParams []string) {
	templates := templateNames(templateParams)
	for _, member := range members {
		switch m := member.(type) {
		case *ast.InterfaceMethodNode:
			returnType := ""
			if m.ReturnType != nil {
				returnType = m.ReturnType.TokenLiteral()
			}
			if m.PHPDoc != nil && m.PHPDoc.ReturnType != "" {
				returnType = m.PHPDoc.ReturnType
			}
			idx.addMethod(className, ResolvedMethod{Name: m.Name, DeclaringClass: className, Declaration: sourceLocation(filename, m), ReturnType: normalizeTemplateAwareType(returnType, ft, templates), Params: paramsFromNodesWithPHPDoc(m.Params, m.PHPDoc, ft, templates), Visibility: "public", Abstract: true})
		case *ast.ConstantNode:
			idx.addClassConstant(className, constantFromNode(filename, className, m, ft))
		}
	}
}

func (idx *ProjectIndex) addClass(filename string, class ResolvedClass, node ast.Node) {
	key := indexKey(class.Name)
	if _, exists := idx.Classes[key]; exists {
		idx.Duplicates = append(idx.Duplicates, DuplicateSymbol{File: filename, Name: class.Name, Pos: node.GetPos()})
		return
	}
	class.ID = stableSymbolID("class", "", class.Name)
	class.Declaration = sourceLocation(filename, node)
	idx.Classes[key] = class
	idx.classLineages = nil
}

func (idx *ProjectIndex) addFunction(fn ResolvedFunction) {
	fn.ID = stableSymbolID("function", "", fn.Name)
	idx.Functions[indexKey(fn.Name)] = fn
}

func (idx *ProjectIndex) addMethod(className string, method ResolvedMethod) {
	key := indexKey(className)
	method.DeclaringClass = className
	method.ID = stableSymbolID("method", className, method.Name)
	if idx.Methods[key] == nil {
		idx.Methods[key] = make(map[string]ResolvedMethod)
	}
	idx.Methods[key][strings.ToLower(method.Name)] = method
	idx.methodsDeclared = nil
}

func buildMethodsDeclaredViews(project *ProjectIndex) map[string][]ResolvedMethod {
	views := make(map[string][]ResolvedMethod, len(project.Methods))
	for classKey := range project.Methods {
		className := classKey
		if class, ok := project.Classes[classKey]; ok {
			className = class.Name
		}
		views[classKey] = buildMethodsDeclaredView(project, classKey, className)
	}
	return views
}

func buildMethodsDeclaredView(project *ProjectIndex, classKey, className string) []ResolvedMethod {
	methodMap := project.Methods[classKey]
	methodKeys := make([]string, 0, len(methodMap))
	for methodKey := range methodMap {
		methodKeys = append(methodKeys, methodKey)
	}
	sort.Strings(methodKeys)
	methods := make([]ResolvedMethod, 0, len(methodKeys))
	for _, methodKey := range methodKeys {
		method := methodMap[methodKey]
		method.DeclaringClass = className
		if method.ID == "" {
			method.ID = stableSymbolID("method", className, method.Name)
		}
		methods = append(methods, method)
	}
	return methods
}

func (idx *ProjectIndex) addProperty(className string, property ResolvedProperty) {
	key := indexKey(className)
	property.DeclaringClass = className
	property.ID = stableSymbolID("property", className, strings.TrimPrefix(property.Name, "$"))
	if idx.Properties[key] == nil {
		idx.Properties[key] = make(map[string]ResolvedProperty)
	}
	idx.Properties[key][strings.ToLower(property.Name)] = property
}

func (idx *ProjectIndex) addClassConstant(className string, constant ResolvedConstant) {
	key := indexKey(className)
	constant.DeclaringClass = className
	constant.ID = stableSymbolID("constant", className, constant.Name)
	if idx.ClassConsts[key] == nil {
		idx.ClassConsts[key] = make(map[string]ResolvedConstant)
	}
	idx.ClassConsts[key][strings.ToLower(constant.Name)] = constant
	idx.Constants[indexKey(className+"::"+constant.Name)] = struct{}{}
}

func methodFromFunction(filename, className string, fn *ast.FunctionNode, ft fileTypeContext, templateParams []string) ResolvedMethod {
	returnType := fn.ReturnType
	if fn.PHPDoc != nil && fn.PHPDoc.ReturnType != "" {
		returnType = fn.PHPDoc.ReturnType
	}
	templates := templateNames(templateParams)
	return ResolvedMethod{
		Name:           fn.Name,
		DeclaringClass: className,
		Declaration:    sourceLocation(filename, fn),
		ReturnType:     normalizeTemplateAwareType(returnType, ft, templates),
		Params:         paramsFromNodesWithPHPDoc(fn.Params, fn.PHPDoc, ft, templates),
		Visibility:     functionVisibility(fn),
		IsStatic:       hasModifier(fn.Modifiers, "static"),
		Abstract:       hasModifier(fn.Modifiers, "abstract"),
		Final:          hasModifier(fn.Modifiers, "final"),
	}
}

func resolvedGenericMetadata(doc *ast.PHPDocNode, ft fileTypeContext) ([]string, []ResolvedGenericParent) {
	if doc == nil {
		return nil, nil
	}
	templates := make([]string, 0, len(doc.Templates))
	for _, template := range doc.Templates {
		templates = append(templates, template.Name)
	}
	templateSet := templateNames(templates)
	references := append(append([]ast.PHPDocTypeReference(nil), doc.Extends...), doc.Implements...)
	parents := make([]ResolvedGenericParent, 0, len(references))
	for _, ref := range references {
		parent := ResolvedGenericParent{Name: ft.resolveClassLike(ref.Name)}
		for _, argument := range ref.TypeArguments {
			parent.TypeArguments = append(parent.TypeArguments, normalizeTemplateAwareType(argument, ft, templateSet))
		}
		parents = append(parents, parent)
	}
	return templates, parents
}

func constantFromNode(filename, className string, c *ast.ConstantNode, ft fileTypeContext) ResolvedConstant {
	return ResolvedConstant{
		Name:           c.Name,
		DeclaringClass: className,
		Declaration:    sourceLocation(filename, c),
		Type:           normalizeTypeWithContext(c.Type, ft),
		Visibility:     defaultVisibility(c.Visibility),
		Final:          hasModifier(c.Modifiers, "final"),
	}
}

func sourceLocation(filename string, node ast.Node) SourceLocation {
	if node == nil {
		return SourceLocation{}
	}
	return SourceLocation{File: filename, Start: node.GetPos(), End: node.GetEndPos()}
}

func functionVisibility(fn *ast.FunctionNode) string {
	if fn.Visibility != "" {
		return defaultVisibility(fn.Visibility)
	}
	if hasModifier(fn.Modifiers, "private") {
		return "private"
	}
	if hasModifier(fn.Modifiers, "protected") {
		return "protected"
	}
	return "public"
}

func paramsFromNodes(nodes []ast.Node, ft fileTypeContext) []ResolvedParam {
	return paramsFromNodesWithPHPDoc(nodes, nil, ft, nil)
}

func paramsFromNodesWithPHPDoc(nodes []ast.Node, doc *ast.PHPDocNode, ft fileTypeContext, templates map[string]struct{}) []ResolvedParam {
	params := make([]ResolvedParam, 0, len(nodes))
	for _, node := range nodes {
		param, ok := node.(*ast.ParamNode)
		if !ok {
			continue
		}
		typ := param.TypeHint
		if typ == "" && param.UnionType != nil {
			typ = param.UnionType.TokenLiteral()
		}
		if doc != nil {
			if documented := doc.GetParamTypeFromPHPDoc(param.Name); documented != "" {
				typ = documented
			}
		}
		params = append(params, ResolvedParam{
			Name:       param.Name,
			Type:       normalizeTemplateAwareType(typ, ft, templates),
			HasDefault: param.DefaultValue != nil,
			IsVariadic: param.IsVariadic,
			IsByRef:    param.IsByRef,
			IsOut:      param.IsByRef,
		})
	}
	return params
}

func resolvedList(ft fileTypeContext, names []string) []string {
	out := make([]string, 0, len(names))
	for _, name := range names {
		if strings.TrimSpace(name) == "" {
			continue
		}
		out = append(out, ft.resolveClassLike(name))
	}
	return out
}

func traitUsesFromMembers(members []ast.Node, ft fileTypeContext) []string {
	var traits []string
	for _, member := range members {
		use, ok := member.(*ast.TraitUseNode)
		if !ok {
			continue
		}
		for _, trait := range use.Traits {
			traits = append(traits, ft.resolveClassLike(trait))
		}
	}
	return traits
}

func optionalList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return []string{value}
}

func hasModifier(modifiers []string, wanted string) bool {
	for _, modifier := range modifiers {
		if strings.EqualFold(modifier, wanted) {
			return true
		}
	}
	return false
}

func defaultVisibility(visibility string) string {
	if visibility == "" {
		return "public"
	}
	return visibility
}

func indexKey(name string) string {
	return strings.ToLower(strings.TrimPrefix(strings.TrimSpace(name), `\`))
}

func unqualifiedName(name string) string {
	name = strings.TrimPrefix(strings.TrimSpace(name), `\`)
	if i := strings.LastIndex(name, `\`); i >= 0 {
		return name[i+1:]
	}
	return name
}

var builtinClassNames = map[string]struct{}{
	"arrayaccess":         {},
	"arrayiterator":       {},
	"arrayobject":         {},
	"closure":             {},
	"countable":           {},
	"dateinterval":        {},
	"datetime":            {},
	"datetimeimmutable":   {},
	"datetimeinterface":   {},
	"datetimezone":        {},
	"error":               {},
	"exception":           {},
	"generator":           {},
	"iterator":            {},
	"iteratoraggregate":   {},
	"jsonexception":       {},
	"jsonserializable":    {},
	"reflectionclass":     {},
	"reflectionexception": {},
	"reflectionfunction":  {},
	"reflectionmethod":    {},
	"reflectionnamedtype": {},
	"reflectionobject":    {},
	"reflectionparameter": {},
	"reflectionproperty":  {},
	"sensitiveparameter":  {},
	"simplexmlelement":    {},
	"stdclass":            {},
	"stringable":          {},
	"throwable":           {},
	"traversable":         {},
	"valueerror":          {},
}

func isBuiltinClassName(name string) bool {
	_, ok := builtinClassNames[indexKey(name)]
	return ok
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

func (idx *ProjectIndex) seedBuiltins() {
	for _, class := range []ResolvedClass{
		{Name: "stdClass", Kind: "class"},
		{Name: "Exception", Kind: "class", Extends: []string{"Throwable"}},
		{Name: "Throwable", Kind: "interface"},
		{Name: "Error", Kind: "class", Extends: []string{"Throwable"}},
		{Name: "DateTime", Kind: "class", Implements: []string{"DateTimeInterface"}},
		{Name: "DateTimeImmutable", Kind: "class", Implements: []string{"DateTimeInterface"}},
		{Name: "DateTimeInterface", Kind: "interface"},
		{Name: "DateTimeZone", Kind: "class"},
		{Name: "Closure", Kind: "class", Final: true},
		{Name: "Stringable", Kind: "interface"},
		{Name: "ReflectionClass", Kind: "class"},
		{Name: "ReflectionException", Kind: "class", Extends: []string{"Exception"}},
		{Name: "ReflectionFunction", Kind: "class"},
		{Name: "ReflectionMethod", Kind: "class"},
		{Name: "ReflectionNamedType", Kind: "class"},
		{Name: "ReflectionObject", Kind: "class", Extends: []string{"ReflectionClass"}},
		{Name: "ReflectionParameter", Kind: "class"},
		{Name: "ReflectionProperty", Kind: "class"},
		{Name: "ArrayAccess", Kind: "interface"},
		{Name: "ArrayIterator", Kind: "class", Implements: []string{"Iterator", "Traversable"}},
		{Name: "ArrayObject", Kind: "class", Implements: []string{"IteratorAggregate", "Traversable"}},
		{Name: "Countable", Kind: "interface"},
		{Name: "DateInterval", Kind: "class"},
		{Name: "Generator", Kind: "class", Final: true},
		{Name: "Iterator", Kind: "interface", Extends: []string{"Traversable"}},
		{Name: "IteratorAggregate", Kind: "interface", Extends: []string{"Traversable"}},
		{Name: "JsonException", Kind: "class", Extends: []string{"Exception"}},
		{Name: "JsonSerializable", Kind: "interface"},
		{Name: "SensitiveParameter", Kind: "class", Final: true},
		{Name: "SimpleXMLElement", Kind: "class"},
		{Name: "Traversable", Kind: "interface"},
		{Name: "ValueError", Kind: "class", Extends: []string{"Error"}},
	} {
		class.ID = stableSymbolID("class", "", class.Name)
		idx.Classes[indexKey(class.Name)] = class
	}
	for _, className := range []string{"DateTime", "DateTimeImmutable"} {
		idx.addMethod(className, ResolvedMethod{Name: "createFromFormat", DeclaringClass: className, ReturnType: className + "|false", Params: []ResolvedParam{{Name: "format"}, {Name: "datetime"}, {Name: "timezone", HasDefault: true}}, Visibility: "public", IsStatic: true})
		idx.addMethod(className, ResolvedMethod{Name: "createFromInterface", DeclaringClass: className, ReturnType: className, Params: []ResolvedParam{{Name: "object"}}, Visibility: "public", IsStatic: true})
		idx.addMethod(className, ResolvedMethod{Name: "getLastErrors", DeclaringClass: className, ReturnType: "array|false", Visibility: "public", IsStatic: true})
	}
	idx.addMethod("DateTime", ResolvedMethod{Name: "createFromImmutable", DeclaringClass: "DateTime", ReturnType: "DateTime", Params: []ResolvedParam{{Name: "object"}}, Visibility: "public", IsStatic: true})
	idx.addMethod("DateTimeImmutable", ResolvedMethod{Name: "createFromMutable", DeclaringClass: "DateTimeImmutable", ReturnType: "DateTimeImmutable", Params: []ResolvedParam{{Name: "object"}}, Visibility: "public", IsStatic: true})
	idx.addMethod("Closure", ResolvedMethod{Name: "fromCallable", DeclaringClass: "Closure", ReturnType: "Closure", Params: []ResolvedParam{{Name: "callback"}}, Visibility: "public", IsStatic: true})
	idx.addMethod("DateTimeZone", ResolvedMethod{Name: "__construct", DeclaringClass: "DateTimeZone", Params: []ResolvedParam{{Name: "timezone"}}, Visibility: "public"})
	idx.addMethod("DateInterval", ResolvedMethod{Name: "__construct", DeclaringClass: "DateInterval", Params: []ResolvedParam{{Name: "duration"}}, Visibility: "public"})
	idx.addMethod("ArrayObject", ResolvedMethod{Name: "__construct", DeclaringClass: "ArrayObject", Params: []ResolvedParam{{Name: "array", HasDefault: true}, {Name: "flags", HasDefault: true}, {Name: "iteratorClass", HasDefault: true}}, Visibility: "public"})
	idx.addMethod("Error", ResolvedMethod{Name: "__construct", DeclaringClass: "Error", Params: []ResolvedParam{{Name: "message", HasDefault: true}, {Name: "code", HasDefault: true}, {Name: "previous", HasDefault: true}}, Visibility: "public"})
	idx.addMethod("Exception", ResolvedMethod{Name: "__construct", DeclaringClass: "Exception", Params: []ResolvedParam{{Name: "message", HasDefault: true}, {Name: "code", HasDefault: true}, {Name: "previous", HasDefault: true}}, Visibility: "public"})
	idx.addMethod("ReflectionClass", ResolvedMethod{Name: "__construct", DeclaringClass: "ReflectionClass", Params: []ResolvedParam{{Name: "objectOrClass"}}, Visibility: "public"})
	idx.addMethod("ReflectionMethod", ResolvedMethod{Name: "__construct", DeclaringClass: "ReflectionMethod", Params: []ResolvedParam{{Name: "objectOrMethod"}, {Name: "method", HasDefault: true}}, Visibility: "public"})
	idx.addMethod("ReflectionProperty", ResolvedMethod{Name: "__construct", DeclaringClass: "ReflectionProperty", Params: []ResolvedParam{{Name: "class"}, {Name: "property"}}, Visibility: "public"})
	idx.addMethod("ReflectionObject", ResolvedMethod{Name: "__construct", DeclaringClass: "ReflectionObject", Params: []ResolvedParam{{Name: "object"}}, Visibility: "public"})
	for _, constant := range []string{"ATOM", "COOKIE", "ISO8601", "RFC822", "RFC850", "RFC1036", "RFC1123", "RFC7231", "RFC2822", "RFC3339", "RFC3339_EXTENDED", "RSS", "W3C"} {
		idx.addClassConstant("DateTime", ResolvedConstant{Name: constant, DeclaringClass: "DateTime", Visibility: "public"})
		idx.addClassConstant("DateTimeImmutable", ResolvedConstant{Name: constant, DeclaringClass: "DateTimeImmutable", Visibility: "public"})
		idx.addClassConstant("DateTimeInterface", ResolvedConstant{Name: constant, DeclaringClass: "DateTimeInterface", Visibility: "public"})
	}
	for _, fn := range []ResolvedFunction{
		{Name: "abs", Params: []ResolvedParam{{Name: "num"}}},
		{Name: "addslashes", Params: []ResolvedParam{{Name: "string"}}},
		{Name: "array_any", Params: []ResolvedParam{{Name: "array"}, {Name: "callback"}}},
		{Name: "array_chunk", Params: []ResolvedParam{{Name: "array"}, {Name: "length"}, {Name: "preserve_keys", HasDefault: true}}},
		{Name: "array_column", Params: []ResolvedParam{{Name: "array"}, {Name: "column_key"}, {Name: "index_key", HasDefault: true}}},
		{Name: "array_diff", Params: []ResolvedParam{{Name: "array"}, {Name: "arrays", IsVariadic: true}}},
		{Name: "array_fill", Params: []ResolvedParam{{Name: "start_index"}, {Name: "count"}, {Name: "value"}}},
		{Name: "array_fill_keys", Params: []ResolvedParam{{Name: "keys"}, {Name: "value"}}},
		{Name: "array_filter", Params: []ResolvedParam{{Name: "array"}, {Name: "callback", HasDefault: true}, {Name: "mode", HasDefault: true}}},
		{Name: "array_intersect_assoc", Params: []ResolvedParam{{Name: "array"}, {Name: "arrays", IsVariadic: true}}},
		{Name: "array_intersect_key", Params: []ResolvedParam{{Name: "array"}, {Name: "arrays", IsVariadic: true}}},
		{Name: "array_key_exists", Params: []ResolvedParam{{Name: "key"}, {Name: "array"}}},
		{Name: "array_key_first", Params: []ResolvedParam{{Name: "array"}}},
		{Name: "array_map", Params: []ResolvedParam{{Name: "callback"}, {Name: "array"}, {Name: "arrays", IsVariadic: true}}},
		{Name: "array_merge", Params: []ResolvedParam{{Name: "arrays", IsVariadic: true}}},
		{Name: "array_merge_recursive", Params: []ResolvedParam{{Name: "arrays", IsVariadic: true}}},
		{Name: "array_pop", Params: []ResolvedParam{{Name: "array", IsByRef: true}}},
		{Name: "array_push", Params: []ResolvedParam{{Name: "array", IsByRef: true}, {Name: "values", IsVariadic: true}}},
		{Name: "array_reduce", Params: []ResolvedParam{{Name: "array"}, {Name: "callback"}, {Name: "initial", HasDefault: true}}},
		{Name: "array_search", Params: []ResolvedParam{{Name: "needle"}, {Name: "haystack"}, {Name: "strict", HasDefault: true}}},
		{Name: "array_shift", Params: []ResolvedParam{{Name: "array", IsByRef: true}}},
		{Name: "array_keys", Params: []ResolvedParam{{Name: "array"}, {Name: "filter_value", HasDefault: true}, {Name: "strict", HasDefault: true}}},
		{Name: "array_slice", Params: []ResolvedParam{{Name: "array"}, {Name: "offset"}, {Name: "length", HasDefault: true}, {Name: "preserve_keys", HasDefault: true}}},
		{Name: "array_unique", Params: []ResolvedParam{{Name: "array"}, {Name: "flags", HasDefault: true}}},
		{Name: "array_unshift", Params: []ResolvedParam{{Name: "array", IsByRef: true}, {Name: "values", IsVariadic: true}}},
		{Name: "array_sum", Params: []ResolvedParam{{Name: "array"}}},
		{Name: "array_values", Params: []ResolvedParam{{Name: "array"}}},
		{Name: "assert", Params: []ResolvedParam{{Name: "assertion"}, {Name: "description", HasDefault: true}}},
		{Name: "base64_encode", Params: []ResolvedParam{{Name: "string"}}},
		{Name: "basename", Params: []ResolvedParam{{Name: "path"}, {Name: "suffix", HasDefault: true}}},
		{Name: "bin2hex", Params: []ResolvedParam{{Name: "string"}}},
		{Name: "ceil", Params: []ResolvedParam{{Name: "num"}}},
		{Name: "checkdate", Params: []ResolvedParam{{Name: "month"}, {Name: "day"}, {Name: "year"}}},
		{Name: "class_exists", Params: []ResolvedParam{{Name: "class"}, {Name: "autoload", HasDefault: true}}},
		{Name: "compact", Params: []ResolvedParam{{Name: "var_name"}, {Name: "var_names", IsVariadic: true}}},
		{Name: "constant", Params: []ResolvedParam{{Name: "name"}}},
		{Name: "count", Params: []ResolvedParam{{Name: "value"}, {Name: "mode", HasDefault: true}}},
		{Name: "crc32", Params: []ResolvedParam{{Name: "string"}}},
		{Name: "define", Params: []ResolvedParam{{Name: "constant_name"}, {Name: "value"}, {Name: "case_insensitive", HasDefault: true}}},
		{Name: "defined", Params: []ResolvedParam{{Name: "constant_name"}}},
		{Name: "die", Params: []ResolvedParam{{Name: "status", HasDefault: true}}},
		{Name: "dirname", Params: []ResolvedParam{{Name: "path"}, {Name: "levels", HasDefault: true}}},
		{Name: "empty", Params: []ResolvedParam{{Name: "var"}}},
		{Name: "enum_exists", Params: []ResolvedParam{{Name: "enum"}, {Name: "autoload", HasDefault: true}}},
		{Name: "end", Params: []ResolvedParam{{Name: "array", IsByRef: true}}},
		{Name: "eval", Params: []ResolvedParam{{Name: "code"}}},
		{Name: "exit", Params: []ResolvedParam{{Name: "status", HasDefault: true}}},
		{Name: "explode", Params: []ResolvedParam{{Name: "separator"}, {Name: "string"}, {Name: "limit", HasDefault: true}}},
		{Name: "extract", Params: []ResolvedParam{{Name: "array"}, {Name: "flags", HasDefault: true}, {Name: "prefix", HasDefault: true}}},
		{Name: "extension_loaded", Params: []ResolvedParam{{Name: "extension"}}},
		{Name: "file_exists", Params: []ResolvedParam{{Name: "filename"}}},
		{Name: "filter_var", Params: []ResolvedParam{{Name: "value"}, {Name: "filter", HasDefault: true}, {Name: "options", HasDefault: true}}},
		{Name: "floor", Params: []ResolvedParam{{Name: "num"}}},
		{Name: "fpassthru", Params: []ResolvedParam{{Name: "stream"}}},
		{Name: "func_get_args"},
		{Name: "function_exists", Params: []ResolvedParam{{Name: "function"}}},
		{Name: "get_class", Params: []ResolvedParam{{Name: "object", HasDefault: true}}},
		{Name: "get_object_vars", Params: []ResolvedParam{{Name: "object"}}},
		{Name: "getenv", Params: []ResolvedParam{{Name: "name", HasDefault: true}, {Name: "local_only", HasDefault: true}}},
		{Name: "glob", Params: []ResolvedParam{{Name: "pattern"}, {Name: "flags", HasDefault: true}}},
		{Name: "hash", Params: []ResolvedParam{{Name: "algo"}, {Name: "data"}, {Name: "binary", HasDefault: true}, {Name: "options", HasDefault: true}}},
		{Name: "http_build_query", Params: []ResolvedParam{{Name: "data"}, {Name: "numeric_prefix", HasDefault: true}, {Name: "arg_separator", HasDefault: true}, {Name: "encoding_type", HasDefault: true}}},
		{Name: "htmlspecialchars", Params: []ResolvedParam{{Name: "string"}, {Name: "flags", HasDefault: true}, {Name: "encoding", HasDefault: true}, {Name: "double_encode", HasDefault: true}}},
		{Name: "implode", Params: []ResolvedParam{{Name: "separator"}, {Name: "array", HasDefault: true}}},
		{Name: "in_array", Params: []ResolvedParam{{Name: "needle"}, {Name: "haystack"}, {Name: "strict", HasDefault: true}}},
		{Name: "intdiv", Params: []ResolvedParam{{Name: "num1"}, {Name: "num2"}}},
		{Name: "intval", Params: []ResolvedParam{{Name: "value"}, {Name: "base", HasDefault: true}}},
		{Name: "interface_exists", Params: []ResolvedParam{{Name: "interface"}, {Name: "autoload", HasDefault: true}}},
		{Name: "isset", Params: []ResolvedParam{{Name: "var"}, {Name: "vars", IsVariadic: true}}},
		{Name: "is_array", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "is_bool", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "is_callable", Params: []ResolvedParam{{Name: "value"}, {Name: "syntax_only", HasDefault: true}, {Name: "callable_name", HasDefault: true, IsByRef: true, IsOut: true}}},
		{Name: "is_countable", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "is_dir", Params: []ResolvedParam{{Name: "filename"}}},
		{Name: "is_file", Params: []ResolvedParam{{Name: "filename"}}},
		{Name: "is_finite", Params: []ResolvedParam{{Name: "num"}}},
		{Name: "is_float", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "is_int", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "is_null", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "is_numeric", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "is_object", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "is_resource", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "is_scalar", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "is_string", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "iterator_count", Params: []ResolvedParam{{Name: "iterator"}}},
		{Name: "iterator_to_array", Params: []ResolvedParam{{Name: "iterator"}, {Name: "preserve_keys", HasDefault: true}}},
		{Name: "json_last_error"},
		{Name: "ksort", Params: []ResolvedParam{{Name: "array", IsByRef: true}, {Name: "flags", HasDefault: true}}},
		{Name: "lcfirst", Params: []ResolvedParam{{Name: "string"}}},
		{Name: "libxml_clear_errors"},
		{Name: "libxml_get_errors"},
		{Name: "libxml_use_internal_errors", Params: []ResolvedParam{{Name: "use_errors", HasDefault: true}}},
		{Name: "ltrim", Params: []ResolvedParam{{Name: "string"}, {Name: "characters", HasDefault: true}}},
		{Name: "max", Params: []ResolvedParam{{Name: "value"}, {Name: "values", IsVariadic: true}}},
		{Name: "mb_strtoupper", Params: []ResolvedParam{{Name: "string"}, {Name: "encoding", HasDefault: true}}},
		{Name: "mb_strlen", Params: []ResolvedParam{{Name: "string"}, {Name: "encoding", HasDefault: true}}},
		{Name: "mb_substr", Params: []ResolvedParam{{Name: "string"}, {Name: "start"}, {Name: "length", HasDefault: true}, {Name: "encoding", HasDefault: true}}},
		{Name: "md5", Params: []ResolvedParam{{Name: "string"}, {Name: "binary", HasDefault: true}}},
		{Name: "method_exists", Params: []ResolvedParam{{Name: "object_or_class"}, {Name: "method"}}},
		{Name: "microtime", Params: []ResolvedParam{{Name: "as_float", HasDefault: true}}},
		{Name: "min", Params: []ResolvedParam{{Name: "value"}, {Name: "values", IsVariadic: true}}},
		{Name: "number_format", Params: []ResolvedParam{{Name: "num"}, {Name: "decimals", HasDefault: true}, {Name: "decimal_separator", HasDefault: true}, {Name: "thousands_separator", HasDefault: true}}},
		{Name: "parse_str", Params: []ResolvedParam{{Name: "string"}, {Name: "result", IsByRef: true, IsOut: true}}},
		{Name: "pathinfo", Params: []ResolvedParam{{Name: "path"}, {Name: "flags", HasDefault: true}}},
		{Name: "preg_match", Params: []ResolvedParam{{Name: "pattern"}, {Name: "subject"}, {Name: "matches", HasDefault: true, IsByRef: true, IsOut: true}}},
		{Name: "preg_quote", Params: []ResolvedParam{{Name: "str"}, {Name: "delimiter", HasDefault: true}}},
		{Name: "printf", Params: []ResolvedParam{{Name: "format"}, {Name: "values", IsVariadic: true}}},
		{Name: "random_bytes", Params: []ResolvedParam{{Name: "length"}}},
		{Name: "reset", Params: []ResolvedParam{{Name: "array", IsByRef: true}}},
		{Name: "range", Params: []ResolvedParam{{Name: "start"}, {Name: "end"}, {Name: "step", HasDefault: true}}},
		{Name: "round", Params: []ResolvedParam{{Name: "num"}, {Name: "precision", HasDefault: true}, {Name: "mode", HasDefault: true}}},
		{Name: "rtrim", Params: []ResolvedParam{{Name: "string"}, {Name: "characters", HasDefault: true}}},
		{Name: "serialize", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "sha1", Params: []ResolvedParam{{Name: "string"}, {Name: "binary", HasDefault: true}}},
		{Name: "sort", Params: []ResolvedParam{{Name: "array", IsByRef: true}, {Name: "flags", HasDefault: true}}},
		{Name: "sprintf", Params: []ResolvedParam{{Name: "format"}, {Name: "values", IsVariadic: true}}},
		{Name: "stripos", Params: []ResolvedParam{{Name: "haystack"}, {Name: "needle"}, {Name: "offset", HasDefault: true}}},
		{Name: "str_contains", Params: []ResolvedParam{{Name: "haystack"}, {Name: "needle"}}},
		{Name: "str_ends_with", Params: []ResolvedParam{{Name: "haystack"}, {Name: "needle"}}},
		{Name: "str_pad", Params: []ResolvedParam{{Name: "string"}, {Name: "length"}, {Name: "pad_string", HasDefault: true}, {Name: "pad_type", HasDefault: true}}},
		{Name: "str_repeat", Params: []ResolvedParam{{Name: "string"}, {Name: "times"}}},
		{Name: "str_replace", Params: []ResolvedParam{{Name: "search"}, {Name: "replace"}, {Name: "subject"}, {Name: "count", HasDefault: true, IsByRef: true, IsOut: true}}},
		{Name: "str_starts_with", Params: []ResolvedParam{{Name: "haystack"}, {Name: "needle"}}},
		{Name: "strcasecmp", Params: []ResolvedParam{{Name: "string1"}, {Name: "string2"}}},
		{Name: "strcmp", Params: []ResolvedParam{{Name: "string1"}, {Name: "string2"}}},
		{Name: "strlen", Params: []ResolvedParam{{Name: "string"}}},
		{Name: "strpos", Params: []ResolvedParam{{Name: "haystack"}, {Name: "needle"}, {Name: "offset", HasDefault: true}}},
		{Name: "strrpos", Params: []ResolvedParam{{Name: "haystack"}, {Name: "needle"}, {Name: "offset", HasDefault: true}}},
		{Name: "strtolower", Params: []ResolvedParam{{Name: "string"}}},
		{Name: "strtoupper", Params: []ResolvedParam{{Name: "string"}}},
		{Name: "strtr", Params: []ResolvedParam{{Name: "string"}, {Name: "from"}, {Name: "to", HasDefault: true}}},
		{Name: "substr", Params: []ResolvedParam{{Name: "string"}, {Name: "offset"}, {Name: "length", HasDefault: true}}},
		{Name: "substr_count", Params: []ResolvedParam{{Name: "haystack"}, {Name: "needle"}, {Name: "offset", HasDefault: true}, {Name: "length", HasDefault: true}}},
		{Name: "sys_get_temp_dir"},
		{Name: "time"},
		{Name: "trait_exists", Params: []ResolvedParam{{Name: "trait"}, {Name: "autoload", HasDefault: true}}},
		{Name: "trim", Params: []ResolvedParam{{Name: "string"}, {Name: "characters", HasDefault: true}}},
		{Name: "trigger_error", Params: []ResolvedParam{{Name: "message"}, {Name: "error_level", HasDefault: true}}},
		{Name: "uasort", Params: []ResolvedParam{{Name: "array"}, {Name: "callback"}}},
		{Name: "ucfirst", Params: []ResolvedParam{{Name: "string"}}},
		{Name: "ucwords", Params: []ResolvedParam{{Name: "string"}, {Name: "separators", HasDefault: true}}},
		{Name: "uniqid", Params: []ResolvedParam{{Name: "prefix", HasDefault: true}, {Name: "more_entropy", HasDefault: true}}},
		{Name: "urlencode", Params: []ResolvedParam{{Name: "string"}}},
		{Name: "unset", Params: []ResolvedParam{{Name: "var"}, {Name: "vars", IsVariadic: true}}},
		{Name: "usleep", Params: []ResolvedParam{{Name: "microseconds"}}},
		{Name: "usort", Params: []ResolvedParam{{Name: "array"}, {Name: "callback"}}},
	} {
		idx.addFunction(fn)
	}
	for _, constant := range []string{"PHP_VERSION", "PHP_VERSION_ID", "PHP_MAJOR_VERSION", "PHP_MINOR_VERSION", "PHP_OS", "PHP_EOL", "true", "false", "null"} {
		idx.Constants[indexKey(constant)] = struct{}{}
	}
}
