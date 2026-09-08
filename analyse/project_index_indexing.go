package analyse

import (
	"sort"
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
)

func (idx *ProjectIndex) indexNodes(filename string, nodes []ast.Node, ft FileTypeContext, currentClass string) {
	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.NamespaceNode:
			nft := CollectFileTypeContext(n.Body)
			if nft.Namespace == "" {
				nft.Namespace = n.Name
			}
			idx.indexNodes(filename, n.Body, nft, currentClass)
		case *ast.ClassNode:
			name := ft.resolveClassLike(n.Name)
			templates, templateBounds, genericParents := resolvedGenericMetadata(n.PHPDoc, ft)
			class := ResolvedClass{
				Name:                  name,
				Extends:               resolvedList(ft, optionalList(n.Extends)),
				Implements:            resolvedList(ft, n.Implements),
				TemplateParams:        templates,
				TemplateBounds:        templateBounds,
				GenericParents:        genericParents,
				Traits:                traitUsesFromMembers(n.Properties, ft),
				Kind:                  "class",
				Final:                 strings.Contains(n.Modifier, "final"),
				Abstract:              strings.Contains(n.Modifier, "abstract"),
				Readonly:              strings.Contains(n.Modifier, "readonly"),
				ConsistentConstructor: hasPHPStanConsistentConstructorTag(n.PHPDoc),
			}
			idx.addClass(filename, class, n)
			idx.indexClassMembers(filename, name, n.Properties, n.Methods, n.Constants, ft, n.PHPDoc, templates)
		case *ast.InterfaceNode:
			name := ft.resolveClassLike(n.Name)
			templates, templateBounds, genericParents := resolvedGenericMetadata(n.PHPDoc, ft)
			idx.addClass(filename, ResolvedClass{Name: name, Extends: resolvedList(ft, n.Extends), TemplateParams: templates, TemplateBounds: templateBounds, GenericParents: genericParents, Kind: "interface"}, n)
			idx.indexInterfaceMembers(filename, name, n.Members, ft, n.PHPDoc, templates)
		case *ast.TraitNode:
			if n.Name != nil {
				name := ft.resolveClassLike(n.Name.Name)
				idx.addClass(filename, ResolvedClass{Name: name, Kind: "trait", Traits: traitUsesFromMembers(n.Body, ft)}, n)
				idx.indexClassMembers(filename, name, n.Body, nil, nil, ft, nil, nil)
			}
		case *ast.EnumNode:
			name := ft.resolveClassLike(n.Name)
			idx.addClass(filename, ResolvedClass{Name: name, Implements: resolvedList(ft, n.Implements), Kind: "enum", Final: true}, n)
			idx.indexClassMembers(filename, name, nil, n.Methods, nil, ft, nil, nil)
			for _, enumCase := range n.Cases {
				idx.addClassConstant(name, ResolvedConstant{Name: enumCase.Name, DeclaringClass: name, Declaration: sourceLocation(filename, enumCase), Visibility: "public"})
			}
			idx.addEnumNativeMembers(filename, name, n)
		case *ast.FunctionNode:
			if currentClass != "" {
				idx.addMethod(currentClass, methodFromFunction(filename, currentClass, n, ft, nil, nil))
				continue
			}
			name := ft.resolveClassLike(n.Name)
			returnType := n.ReturnType
			if n.PHPDoc != nil && n.PHPDoc.ReturnType != "" {
				returnType = n.PHPDoc.ReturnType
			}
			returnType = collapsePHPDocConditionalType(returnType, n.ReturnType)
			returnType = expandPHPDocTypeAliases(returnType, phpDocTypeAliasBindings(n.PHPDoc))
			callableReturn := callableReturnType(returnType, ft)
			normalizedReturn := normalizeTypeWithContext(returnType, ft)
			if !callableReturn.IsEmpty() {
				normalizedReturn = "callable"
			}
			fn := ResolvedFunction{Name: name, Declaration: sourceLocation(filename, n), ReturnType: normalizedReturn, CallableReturnType: callableReturn.dnfString(), Params: paramsFromNodesWithPHPDoc(n.Params, n.PHPDoc, ft, nil, phpDocTypeAliasBindings(n.PHPDoc))}
			if n.PHPDoc != nil {
				fn.Deprecated = n.PHPDoc.Deprecated
				fn.DeprecationMessage = n.PHPDoc.DeprecationMessage
			}
			idx.addFunction(fn)
		case *ast.ConstantNode:
			idx.addGlobalConstant(filename, ft.resolveClassLike(n.Name))
		}
	}
}

// indexPromotedProperties registers PHP 8 constructor-promoted parameters
// (e.g. `private readonly Foo $x` in a __construct signature) as properties,
// mirroring promotedClassProperties' handling for return-type inference so
// property-existence checks don't false-positive on them.
func (idx *ProjectIndex) indexPromotedProperties(filename, className string, constructor *ast.FunctionNode, ft FileTypeContext) {
	for _, paramNode := range constructor.Params {
		param, ok := paramNode.(*ast.ParamNode)
		if !ok || !param.IsPromoted {
			continue
		}
		idx.addProperty(className, ResolvedProperty{
			Declaration: sourceLocation(filename, param),
			Name:        param.Name,
			Type:        normalizeTypeWithContext(param.TypeHint, ft),
			Visibility:  defaultVisibility(param.Visibility),
			Readonly:    param.IsReadonly,
		})
	}
}

func (idx *ProjectIndex) indexClassMembers(filename, className string, properties, methods, constants []ast.Node, ft FileTypeContext, classDoc *ast.PHPDocNode, templateParams []string) {
	for _, propNode := range properties {
		switch p := propNode.(type) {
		case *ast.PropertyNode:
			rawType := p.TypeHint
			docType := ""
			if p.PHPDoc != nil {
				docType = p.PHPDoc.VarType
			}
			callableReturn := callableReturnType(docType, ft)
			if callableReturn.IsEmpty() {
				callableReturn = callableReturnType(rawType, ft)
			}
			normalizedType := richerGenericType(rawType, docType, ft)
			if !callableReturn.IsEmpty() {
				normalizedType = "callable"
			}
			idx.addProperty(className, ResolvedProperty{
				Declaration:        sourceLocation(filename, p),
				Name:               p.Name,
				Type:               normalizedType,
				CallableReturnType: callableReturn.dnfString(),
				Visibility:         defaultVisibility(p.Visibility),
				IsStatic:           p.IsStatic,
				Readonly:           p.IsReadonly,
			})
		case *ast.TraitUseNode:
			// Trait use is checked by level-0 rules; no index entry needed.
		case *ast.FunctionNode:
			idx.addMethod(className, methodFromFunction(filename, className, p, ft, classDoc, templateParams))
		}
	}
	for _, methodNode := range methods {
		fn, ok := methodNode.(*ast.FunctionNode)
		if !ok {
			continue
		}
		idx.addMethod(className, methodFromFunction(filename, className, fn, ft, classDoc, templateParams))
		if strings.EqualFold(fn.Name, "__construct") {
			idx.indexPromotedProperties(filename, className, fn, ft)
		}
	}
	for _, constNode := range constants {
		if c, ok := constNode.(*ast.ConstantNode); ok {
			idx.addClassConstant(className, constantFromNode(filename, className, c, ft))
		}
	}
}

func (idx *ProjectIndex) indexInterfaceMembers(filename, className string, members []ast.Node, ft FileTypeContext, classDoc *ast.PHPDocNode, templateParams []string) {
	for _, member := range members {
		switch m := member.(type) {
		case *ast.InterfaceMethodNode:
			nativeReturn := ""
			if m.ReturnType != nil {
				nativeReturn = m.ReturnType.TokenLiteral()
			}
			returnType := nativeReturn
			if m.PHPDoc != nil && m.PHPDoc.ReturnType != "" {
				returnType = m.PHPDoc.ReturnType
			}
			returnType = collapsePHPDocConditionalType(returnType, nativeReturn)
			aliases := phpDocTypeAliasBindings(classDoc, m.PHPDoc)
			returnType = expandPHPDocTypeAliases(returnType, aliases)
			templates := templateNames(templateParams)
			if m.PHPDoc != nil {
				for _, template := range m.PHPDoc.Templates {
					templates = mergeTemplateNames(templates, []string{template.Name})
				}
			}
			for name := range aliases {
				templates = mergeTemplateNames(templates, []string{name})
			}
			callableReturn := callableReturnType(returnType, ft)
			normalizedReturn := normalizeTemplateAwareType(returnType, ft, templates)
			if !callableReturn.IsEmpty() {
				normalizedReturn = "callable"
			}
			nativeReturnType := ""
			if nativeReturn != "" {
				nativeReturnType = normalizeTemplateAwareType(nativeReturn, ft, templates)
			}
			idx.addMethod(className, ResolvedMethod{Name: m.Name, DeclaringClass: className, Declaration: sourceLocation(filename, m), ReturnType: normalizedReturn, NativeReturnType: nativeReturnType, CallableReturnType: callableReturn.dnfString(), Params: paramsFromNodesWithPHPDoc(m.Params, m.PHPDoc, ft, templates, aliases), Visibility: "public", Abstract: true})
		case *ast.PropertyNode:
			rawType := m.TypeHint
			docType := ""
			if m.PHPDoc != nil {
				docType = m.PHPDoc.VarType
			}
			callableReturn := callableReturnType(docType, ft)
			if callableReturn.IsEmpty() {
				callableReturn = callableReturnType(rawType, ft)
			}
			normalizedType := richerGenericType(rawType, docType, ft)
			if !callableReturn.IsEmpty() {
				normalizedType = "callable"
			}
			idx.addProperty(className, ResolvedProperty{Name: m.Name, DeclaringClass: className, Declaration: sourceLocation(filename, m), Type: normalizedType, CallableReturnType: callableReturn.dnfString(), Visibility: "public", Readonly: m.IsReadonly})
		case *ast.ConstantNode:
			idx.addClassConstant(className, constantFromNode(filename, className, m, ft))
		}
	}
}

func (idx *ProjectIndex) addClass(filename string, class ResolvedClass, node ast.Node) {
	key := indexKey(class.Name)
	class.ID = stableSymbolID("class", "", class.Name)
	class.Declaration = sourceLocation(filename, node)
	_, exists := idx.Classes[key]
	if exists {
		idx.collidingDefinitions[classDefinitionKey(class.Name)] = struct{}{}
		idx.Duplicates = append(idx.Duplicates, DuplicateSymbol{File: filename, Name: class.Name, Pos: node.GetPos()})
		return
	}
	idx.Classes[key] = class
	idx.classLineages = nil

	// Track class → file mapping for incremental analysis
	if idx.fileClasses[filename] == nil {
		idx.fileClasses[filename] = make(map[string]struct{})
	}
	idx.fileClasses[filename][class.Name] = struct{}{}
}

func (idx *ProjectIndex) addFunction(fn ResolvedFunction) {
	fn.ID = stableSymbolID("function", "", fn.Name)
	key := indexKey(fn.Name)
	_, exists := idx.Functions[key]
	if exists {
		idx.collidingDefinitions[functionDefinitionKey(fn.Name)] = struct{}{}
	}
	idx.Functions[key] = fn
}

func (idx *ProjectIndex) addMethod(className string, method ResolvedMethod) {
	idx.addMethodMaybe(className, method, true)
}

func (idx *ProjectIndex) addMethodIfMissing(className string, method ResolvedMethod) {
	idx.addMethodMaybe(className, method, false)
}

func (idx *ProjectIndex) addMethodMaybe(className string, method ResolvedMethod, overwrite bool) {
	key := indexKey(className)
	method.DeclaringClass = className
	method.ID = stableSymbolID("method", className, method.Name)
	if idx.Methods[key] == nil {
		idx.Methods[key] = make(map[string]ResolvedMethod)
	}
	methodKey := asciiLowerIdent(method.Name)
	_, exists := idx.Methods[key][methodKey]
	if exists && !overwrite {
		return
	}
	if exists {
		idx.collidingDefinitions[memberDefinitionKey("method", key, methodKey)] = struct{}{}
	}
	idx.Methods[key][methodKey] = method
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

func (idx *ProjectIndex) addEnumNativeMembers(filename, enumName string, enum *ast.EnumNode) {
	declaration := sourceLocation(filename, enum)
	idx.addProperty(enumName, ResolvedProperty{
		Name:           "name",
		DeclaringClass: enumName,
		Declaration:    declaration,
		Type:           "string",
		Visibility:     "public",
		Readonly:       true,
	})
	idx.addMethod(enumName, ResolvedMethod{
		Name:           "cases",
		DeclaringClass: enumName,
		Declaration:    declaration,
		ReturnType:     "array",
		Visibility:     "public",
		IsStatic:       true,
	})
	backing := enumBackingType(enum.BackedBy)
	if backing == "" {
		return
	}
	idx.addProperty(enumName, ResolvedProperty{
		Name:           "value",
		DeclaringClass: enumName,
		Declaration:    declaration,
		Type:           backing,
		Visibility:     "public",
		Readonly:       true,
	})
	idx.addMethod(enumName, ResolvedMethod{
		Name:           "from",
		DeclaringClass: enumName,
		Declaration:    declaration,
		ReturnType:     enumName,
		Params:         []ResolvedParam{{Name: "value"}},
		Visibility:     "public",
		IsStatic:       true,
	})
	idx.addMethod(enumName, ResolvedMethod{
		Name:           "tryFrom",
		DeclaringClass: enumName,
		Declaration:    declaration,
		ReturnType:     "?" + enumName,
		Params:         []ResolvedParam{{Name: "value"}},
		Visibility:     "public",
		IsStatic:       true,
	})
}

func enumBackingType(backedBy string) string {
	switch asciiLowerIdent(strings.TrimSpace(backedBy)) {
	case "int":
		return "int"
	case "string":
		return "string"
	default:
		return ""
	}
}

func (idx *ProjectIndex) addProperty(className string, property ResolvedProperty) {
	key := indexKey(className)
	property.DeclaringClass = className
	property.ID = stableSymbolID("property", className, strings.TrimPrefix(property.Name, "$"))
	if idx.Properties[key] == nil {
		idx.Properties[key] = make(map[string]ResolvedProperty)
	}
	propertyKey := asciiLowerIdent(property.Name)
	_, exists := idx.Properties[key][propertyKey]
	if exists {
		idx.collidingDefinitions[memberDefinitionKey("property", key, propertyKey)] = struct{}{}
	}
	idx.Properties[key][propertyKey] = property
}

func (idx *ProjectIndex) addClassConstant(className string, constant ResolvedConstant) {
	idx.addClassConstantMaybe(className, constant, true)
}

func (idx *ProjectIndex) addClassConstantIfMissing(className string, constant ResolvedConstant) {
	idx.addClassConstantMaybe(className, constant, false)
}

func (idx *ProjectIndex) addClassConstantMaybe(className string, constant ResolvedConstant, overwrite bool) {
	key := indexKey(className)
	constant.DeclaringClass = className
	constant.ID = stableSymbolID("constant", className, constant.Name)
	if idx.ClassConsts[key] == nil {
		idx.ClassConsts[key] = make(map[string]ResolvedConstant)
	}
	constantKey := asciiLowerIdent(constant.Name)
	_, exists := idx.ClassConsts[key][constantKey]
	if exists && !overwrite {
		return
	}
	if exists {
		idx.collidingDefinitions[memberDefinitionKey("class-constant", key, constantKey)] = struct{}{}
	}
	idx.ClassConsts[key][constantKey] = constant
	idx.Constants[indexKey(className+"::"+constant.Name)] = struct{}{}
}

func (idx *ProjectIndex) addGlobalConstant(filename, name string) {
	key := indexKey(name)
	_, exists := idx.globalConstantFiles[key]
	if exists {
		idx.collidingDefinitions[globalConstantDefinitionKey(key)] = struct{}{}
	}
	idx.Constants[key] = struct{}{}
	idx.globalConstantFiles[key] = filename
}

func phpDocTypeAliasBindings(docs ...*ast.PHPDocNode) map[string]string {
	var bindings map[string]string
	for _, doc := range docs {
		if doc == nil || len(doc.TypeAliases) == 0 {
			continue
		}
		if bindings == nil {
			bindings = make(map[string]string, len(doc.TypeAliases))
		}
		for _, alias := range doc.TypeAliases {
			name := strings.TrimSpace(alias.Name)
			typ := strings.TrimSpace(alias.Type)
			if name == "" || typ == "" {
				continue
			}
			bindings[name] = typ
		}
	}
	return bindings
}

func methodFromFunction(filename, className string, fn *ast.FunctionNode, ft FileTypeContext, classDoc *ast.PHPDocNode, templateParams []string) ResolvedMethod {
	aliases := phpDocTypeAliasBindings(classDoc, fn.PHPDoc)
	returnType := fn.ReturnType
	if fn.PHPDoc != nil && fn.PHPDoc.ReturnType != "" {
		returnType = fn.PHPDoc.ReturnType
	}
	returnType = collapsePHPDocConditionalType(returnType, fn.ReturnType)
	returnType = expandPHPDocTypeAliases(returnType, aliases)
	templates := mergeTemplateNames(templateNames(templateParams), nil)
	if fn.PHPDoc != nil {
		for _, template := range fn.PHPDoc.Templates {
			templates = mergeTemplateNames(templates, []string{template.Name})
		}
	}
	for name := range aliases {
		templates = mergeTemplateNames(templates, []string{name})
	}
	callableReturn := callableReturnType(returnType, ft)
	normalizedReturn := normalizeTemplateAwareType(returnType, ft, templates)
	if !callableReturn.IsEmpty() {
		normalizedReturn = "callable"
	}
	nativeReturnType := ""
	if fn.ReturnType != "" {
		nativeReturnType = normalizeTemplateAwareType(fn.ReturnType, ft, templates)
	}
	method := ResolvedMethod{
		Name:               fn.Name,
		DeclaringClass:     className,
		Declaration:        sourceLocation(filename, fn),
		ReturnType:         normalizedReturn,
		NativeReturnType:   nativeReturnType,
		CallableReturnType: callableReturn.dnfString(),
		Params:             paramsFromNodesWithPHPDoc(fn.Params, fn.PHPDoc, ft, templates, aliases),
		Visibility:         functionVisibility(fn),
		IsStatic:           hasModifier(fn.Modifiers, "static"),
		Abstract:           hasModifier(fn.Modifiers, "abstract"),
		Final:              hasModifier(fn.Modifiers, "final"),
	}
	if fn.PHPDoc != nil {
		method.Deprecated = fn.PHPDoc.Deprecated
		method.DeprecationMessage = fn.PHPDoc.DeprecationMessage
	}
	return method
}

func resolvedGenericMetadata(doc *ast.PHPDocNode, ft FileTypeContext) ([]string, []string, []ResolvedGenericParent) {
	if doc == nil {
		return nil, nil, nil
	}
	templates := make([]string, 0, len(doc.Templates))
	for _, template := range doc.Templates {
		templates = append(templates, template.Name)
	}
	templateSet := templateNames(templates)
	templateBounds := make([]string, 0, len(doc.Templates))
	for _, template := range doc.Templates {
		templateBounds = append(templateBounds, normalizeTemplateAwareType(template.Bound, ft, templateSet))
	}
	references := append(append([]ast.PHPDocTypeReference(nil), doc.Extends...), doc.Implements...)
	parents := make([]ResolvedGenericParent, 0, len(references))
	for _, ref := range references {
		parent := ResolvedGenericParent{Name: ft.resolveClassLike(ref.Name)}
		for _, argument := range ref.TypeArguments {
			parent.TypeArguments = append(parent.TypeArguments, normalizeTemplateAwareType(argument, ft, templateSet))
		}
		parents = append(parents, parent)
	}
	return templates, templateBounds, parents
}

func constantFromNode(filename, className string, c *ast.ConstantNode, ft FileTypeContext) ResolvedConstant {
	typ := c.Type
	if typ == "" && c.PHPDoc != nil && c.PHPDoc.VarType != "" {
		typ = c.PHPDoc.VarType
	}
	return ResolvedConstant{
		Name:           c.Name,
		DeclaringClass: className,
		Declaration:    sourceLocation(filename, c),
		Type:           normalizeTypeWithContext(typ, ft),
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

func paramsFromNodes(nodes []ast.Node, ft FileTypeContext) []ResolvedParam {
	return paramsFromNodesWithPHPDoc(nodes, nil, ft, nil, nil)
}

func paramsFromNodesWithPHPDoc(nodes []ast.Node, doc *ast.PHPDocNode, ft FileTypeContext, templates map[string]struct{}, aliases map[string]string) []ResolvedParam {
	var callableTemplates map[string]struct{}
	if doc != nil {
		for _, template := range doc.Templates {
			if callableTemplates == nil {
				callableTemplates = make(map[string]struct{}, len(doc.Templates))
			}
			callableTemplates[asciiLowerIdent(template.Name)] = struct{}{}
		}
	}
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
		native := typ
		if doc != nil {
			if documented := doc.GetParamTypeFromPHPDoc(param.Name); documented != "" {
				typ = documentedParamTypePreservingNativeNull(native, documented)
			}
		}
		typ = expandPHPDocTypeAliases(typ, aliases)
		// Composite types that mention call-site templates stay mixed until
		// argument inference can bind them. Bare template names and
		// class-string<T> stay so find(Foo::class) can substitute T.
		if phpDocUsesTemplate(typ, callableTemplates) && !isKnownTemplateName(typ, templates) && !isKnownTemplateName(typ, callableTemplates) && !isClassStringOfKnownTemplate(typ, templates, callableTemplates) {
			typ = "mixed"
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

func documentedParamTypePreservingNativeNull(native, documented string) string {
	if documented == "" {
		return native
	}
	if phpTypeIncludesNull(native) && !phpTypeIncludesNull(documented) {
		return documented + "|null"
	}
	return documented
}

func phpTypeIncludesNull(raw string) bool {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "?") {
		return true
	}
	for _, part := range splitTopLevelTypes(raw, '|') {
		if asciiLowerIdent(strings.TrimPrefix(strings.TrimSpace(part), `\`)) == "null" {
			return true
		}
	}
	return false
}

func resolvedList(ft FileTypeContext, names []string) []string {
	out := make([]string, 0, len(names))
	for _, name := range names {
		if strings.TrimSpace(name) == "" {
			continue
		}
		out = append(out, ft.resolveClassLike(name))
	}
	return out
}

func traitUsesFromMembers(members []ast.Node, ft FileTypeContext) []string {
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
	return asciiLowerIdent(strings.TrimPrefix(strings.TrimSpace(name), `\`))
}

func unqualifiedName(name string) string {
	name = strings.TrimPrefix(strings.TrimSpace(name), `\`)
	if i := strings.LastIndex(name, `\`); i >= 0 {
		return name[i+1:]
	}
	return name
}
