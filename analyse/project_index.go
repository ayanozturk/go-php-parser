package analyse

import (
	"go-phpcs/ast"
	"strings"
)

type ProjectIndex struct {
	Classes     map[string]ResolvedClass
	Methods     map[string]map[string]ResolvedMethod
	Properties  map[string]map[string]ResolvedProperty
	ClassConsts map[string]map[string]ResolvedConstant
	Functions   map[string]ResolvedFunction
	Constants   map[string]struct{}
	FileTypes   map[string]fileTypeContext
	Duplicates  []DuplicateSymbol
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

func BuildProjectIndex(parsed map[string][]ast.Node) *ProjectIndex {
	idx := NewProjectIndex()
	for filename, nodes := range parsed {
		ft := collectFileTypeContext(nodes)
		idx.FileTypes[filename] = ft
		idx.indexNodes(filename, nodes, ft, "")
	}
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
	class, ok := idx.Classes[indexKey(name)]
	return class, ok
}

func (idx *ProjectIndex) ResolveMethod(className, methodName string) (ResolvedMethod, bool) {
	for _, candidate := range idx.classLineage(className) {
		methods := idx.Methods[indexKey(candidate)]
		if methods == nil {
			continue
		}
		if method, ok := methods[strings.ToLower(methodName)]; ok {
			method.DeclaringClass = candidate
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

func (idx *ProjectIndex) ResolveFunction(name string) (ResolvedFunction, bool) {
	fn, ok := idx.Functions[indexKey(name)]
	return fn, ok
}

func (idx *ProjectIndex) classLineage(className string) []string {
	var out []string
	seen := map[string]struct{}{}
	var walk func(string)
	walk = func(name string) {
		key := indexKey(name)
		if key == "" {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, name)
		class, ok := idx.Classes[key]
		if !ok {
			return
		}
		for _, parent := range class.Extends {
			walk(parent)
		}
		for _, iface := range class.Implements {
			walk(iface)
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
			class := ResolvedClass{
				Name:                  name,
				Extends:               resolvedList(ft, optionalList(n.Extends)),
				Implements:            resolvedList(ft, n.Implements),
				Kind:                  "class",
				Final:                 strings.Contains(n.Modifier, "final"),
				Abstract:              strings.Contains(n.Modifier, "abstract"),
				Readonly:              strings.Contains(n.Modifier, "readonly"),
				ConsistentConstructor: hasPHPStanConsistentConstructorTag(n.PHPDoc),
			}
			idx.addClass(filename, class, n.Pos)
			idx.indexClassMembers(name, n.Properties, n.Methods, n.Constants, ft)
		case *ast.InterfaceNode:
			name := ft.resolveClassLike(n.Name)
			idx.addClass(filename, ResolvedClass{Name: name, Extends: resolvedList(ft, n.Extends), Kind: "interface"}, n.Pos)
			idx.indexInterfaceMembers(name, n.Members, ft)
		case *ast.TraitNode:
			if n.Name != nil {
				name := ft.resolveClassLike(n.Name.Name)
				idx.addClass(filename, ResolvedClass{Name: name, Kind: "trait"}, n.Pos)
				idx.indexNodes(filename, n.Body, ft, name)
			}
		case *ast.EnumNode:
			name := ft.resolveClassLike(n.Name)
			idx.addClass(filename, ResolvedClass{Name: name, Implements: resolvedList(ft, n.Implements), Kind: "enum", Final: true}, n.Pos)
			idx.indexClassMembers(name, nil, n.Methods, nil, ft)
		case *ast.FunctionNode:
			if currentClass != "" {
				idx.addMethod(currentClass, methodFromFunction(currentClass, n, ft))
				continue
			}
			name := ft.resolveClassLike(n.Name)
			idx.addFunction(ResolvedFunction{Name: name, ReturnType: normalizeTypeWithContext(n.ReturnType, ft), Params: paramsFromNodes(n.Params, ft)})
		case *ast.ConstantNode:
			idx.Constants[indexKey(ft.resolveClassLike(n.Name))] = struct{}{}
		}
	}
}

func (idx *ProjectIndex) indexClassMembers(className string, properties, methods, constants []ast.Node, ft fileTypeContext) {
	for _, propNode := range properties {
		switch p := propNode.(type) {
		case *ast.PropertyNode:
			idx.addProperty(className, ResolvedProperty{
				Name:       p.Name,
				Type:       normalizeTypeWithContext(p.TypeHint, ft),
				Visibility: defaultVisibility(p.Visibility),
				IsStatic:   p.IsStatic,
				Readonly:   p.IsReadonly,
			})
		case *ast.TraitUseNode:
			// Trait use is checked by level-0 rules; no index entry needed.
		case *ast.FunctionNode:
			idx.addMethod(className, methodFromFunction(className, p, ft))
		}
	}
	for _, methodNode := range methods {
		if fn, ok := methodNode.(*ast.FunctionNode); ok {
			idx.addMethod(className, methodFromFunction(className, fn, ft))
		}
	}
	for _, constNode := range constants {
		if c, ok := constNode.(*ast.ConstantNode); ok {
			idx.addClassConstant(className, constantFromNode(className, c, ft))
		}
	}
}

func (idx *ProjectIndex) indexInterfaceMembers(className string, members []ast.Node, ft fileTypeContext) {
	for _, member := range members {
		switch m := member.(type) {
		case *ast.InterfaceMethodNode:
			returnType := ""
			if m.ReturnType != nil {
				returnType = m.ReturnType.TokenLiteral()
			}
			idx.addMethod(className, ResolvedMethod{Name: m.Name, DeclaringClass: className, ReturnType: normalizeTypeWithContext(returnType, ft), Params: paramsFromNodes(m.Params, ft), Visibility: "public", Abstract: true})
		case *ast.ConstantNode:
			idx.addClassConstant(className, constantFromNode(className, m, ft))
		}
	}
}

func (idx *ProjectIndex) addClass(filename string, class ResolvedClass, pos ast.Position) {
	key := indexKey(class.Name)
	if _, exists := idx.Classes[key]; exists {
		idx.Duplicates = append(idx.Duplicates, DuplicateSymbol{File: filename, Name: class.Name, Pos: pos})
		return
	}
	idx.Classes[key] = class
}

func (idx *ProjectIndex) addFunction(fn ResolvedFunction) {
	idx.Functions[indexKey(fn.Name)] = fn
}

func (idx *ProjectIndex) addMethod(className string, method ResolvedMethod) {
	key := indexKey(className)
	if idx.Methods[key] == nil {
		idx.Methods[key] = make(map[string]ResolvedMethod)
	}
	idx.Methods[key][strings.ToLower(method.Name)] = method
}

func (idx *ProjectIndex) addProperty(className string, property ResolvedProperty) {
	key := indexKey(className)
	if idx.Properties[key] == nil {
		idx.Properties[key] = make(map[string]ResolvedProperty)
	}
	idx.Properties[key][strings.ToLower(property.Name)] = property
}

func (idx *ProjectIndex) addClassConstant(className string, constant ResolvedConstant) {
	key := indexKey(className)
	if idx.ClassConsts[key] == nil {
		idx.ClassConsts[key] = make(map[string]ResolvedConstant)
	}
	idx.ClassConsts[key][strings.ToLower(constant.Name)] = constant
	idx.Constants[indexKey(className+"::"+constant.Name)] = struct{}{}
}

func methodFromFunction(className string, fn *ast.FunctionNode, ft fileTypeContext) ResolvedMethod {
	return ResolvedMethod{
		Name:           fn.Name,
		DeclaringClass: className,
		ReturnType:     normalizeTypeWithContext(fn.ReturnType, ft),
		Params:         paramsFromNodes(fn.Params, ft),
		Visibility:     functionVisibility(fn),
		IsStatic:       hasModifier(fn.Modifiers, "static"),
		Abstract:       hasModifier(fn.Modifiers, "abstract"),
		Final:          hasModifier(fn.Modifiers, "final"),
	}
}

func constantFromNode(className string, c *ast.ConstantNode, ft fileTypeContext) ResolvedConstant {
	return ResolvedConstant{
		Name:           c.Name,
		DeclaringClass: className,
		Type:           normalizeTypeWithContext(c.Type, ft),
		Visibility:     defaultVisibility(c.Visibility),
		Final:          hasModifier(c.Modifiers, "final"),
	}
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
		params = append(params, ResolvedParam{
			Name:       param.Name,
			Type:       normalizeTypeWithContext(typ, ft),
			HasDefault: param.DefaultValue != nil,
			IsVariadic: param.IsVariadic,
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

func (idx *ProjectIndex) seedBuiltins() {
	for _, class := range []ResolvedClass{
		{Name: "stdClass", Kind: "class"},
		{Name: "Exception", Kind: "class", Extends: []string{"Throwable"}},
		{Name: "Throwable", Kind: "interface"},
		{Name: "Error", Kind: "class", Extends: []string{"Throwable"}},
		{Name: "DateTime", Kind: "class"},
		{Name: "DateTimeImmutable", Kind: "class"},
		{Name: "Closure", Kind: "class", Final: true},
		{Name: "Stringable", Kind: "interface"},
	} {
		idx.Classes[indexKey(class.Name)] = class
	}
	for _, fn := range []ResolvedFunction{
		{Name: "array_map", Params: []ResolvedParam{{Name: "callback"}, {Name: "array"}, {Name: "arrays", IsVariadic: true}}},
		{Name: "array_keys", Params: []ResolvedParam{{Name: "array"}, {Name: "filter_value", HasDefault: true}, {Name: "strict", HasDefault: true}}},
		{Name: "class_exists", Params: []ResolvedParam{{Name: "class"}, {Name: "autoload", HasDefault: true}}},
		{Name: "compact", Params: []ResolvedParam{{Name: "var_name"}, {Name: "var_names", IsVariadic: true}}},
		{Name: "constant", Params: []ResolvedParam{{Name: "name"}}},
		{Name: "define", Params: []ResolvedParam{{Name: "constant_name"}, {Name: "value"}, {Name: "case_insensitive", HasDefault: true}}},
		{Name: "defined", Params: []ResolvedParam{{Name: "constant_name"}}},
		{Name: "die", Params: []ResolvedParam{{Name: "status", HasDefault: true}}},
		{Name: "empty", Params: []ResolvedParam{{Name: "var"}}},
		{Name: "enum_exists", Params: []ResolvedParam{{Name: "enum"}, {Name: "autoload", HasDefault: true}}},
		{Name: "exit", Params: []ResolvedParam{{Name: "status", HasDefault: true}}},
		{Name: "extension_loaded", Params: []ResolvedParam{{Name: "extension"}}},
		{Name: "function_exists", Params: []ResolvedParam{{Name: "function"}}},
		{Name: "interface_exists", Params: []ResolvedParam{{Name: "interface"}, {Name: "autoload", HasDefault: true}}},
		{Name: "isset", Params: []ResolvedParam{{Name: "var"}, {Name: "vars", IsVariadic: true}}},
		{Name: "is_array", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "is_bool", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "is_float", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "is_int", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "is_null", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "is_object", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "is_string", Params: []ResolvedParam{{Name: "value"}}},
		{Name: "method_exists", Params: []ResolvedParam{{Name: "object_or_class"}, {Name: "method"}}},
		{Name: "preg_match", Params: []ResolvedParam{{Name: "pattern"}, {Name: "subject"}, {Name: "matches", HasDefault: true}}},
		{Name: "printf", Params: []ResolvedParam{{Name: "format"}, {Name: "values", IsVariadic: true}}},
		{Name: "sprintf", Params: []ResolvedParam{{Name: "format"}, {Name: "values", IsVariadic: true}}},
		{Name: "str_contains", Params: []ResolvedParam{{Name: "haystack"}, {Name: "needle"}}},
		{Name: "trait_exists", Params: []ResolvedParam{{Name: "trait"}, {Name: "autoload", HasDefault: true}}},
	} {
		idx.Functions[indexKey(fn.Name)] = fn
	}
	for _, constant := range []string{"PHP_VERSION", "PHP_VERSION_ID", "PHP_MAJOR_VERSION", "PHP_MINOR_VERSION", "PHP_OS", "PHP_EOL", "true", "false", "null"} {
		idx.Constants[indexKey(constant)] = struct{}{}
	}
}
