package analyse

import (
	"fmt"
	"github.com/ayanozturk/go-php-parser/ast"
	"strings"
)

func (r *Level0Rule) checkClassModel(filename string, nodes []ast.Node, ctx *AnalysisContext, fileCtx FileTypeContext) []AnalysisIssue {
	var issues []AnalysisIssue
	for _, duplicate := range ctx.Resolver.DuplicateClasses(filename) {
		issues = append(issues, issue(filename, duplicate.Pos, level0ClassModelCode, fmt.Sprintf("Duplicate declaration of class %s.", duplicate.Name)))
	}

	var walk func([]ast.Node, FileTypeContext, string)
	walk = func(nodes []ast.Node, ft FileTypeContext, currentClass string) {
		for _, node := range nodes {
			switch n := node.(type) {
			case *ast.NamespaceNode:
				var cache map[*ast.NamespaceNode]FileTypeContext
				if ctx != nil {
					if ctx.namespaceContextByNode == nil {
						ctx.namespaceContextByNode = make(map[*ast.NamespaceNode]FileTypeContext)
					}
					cache = ctx.namespaceContextByNode
				}
				walk(n.Body, namespaceTypeContext(n, cache), currentClass)
			case *ast.ClassNode:
				className := ft.resolveClassLike(n.Name)
				if hasClassModifier(n, "final") && hasClassModifier(n, "abstract") {
					issues = append(issues, issueSpan(filename, n, level0ClassModelCode, fmt.Sprintf("Class %s cannot be both final and abstract.", className)))
				}
				if n.Extends != "" {
					parentName := ft.resolveClassLike(n.Extends)
					if parent, ok := ctx.Resolver.ResolveClass(parentName); !ok {
						issues = append(issues, issueSpan(filename, n, level0ClassModelCode, fmt.Sprintf("Class %s extends unknown class %s.", className, parentName)))
					} else if parent.Kind != "class" {
						issues = append(issues, issueSpan(filename, n, level0ClassModelCode, fmt.Sprintf("Class %s extends %s %s.", className, parent.Kind, parent.Name)))
					} else {
						if parent.Final {
							issues = append(issues, issueSpan(filename, n, level0ClassModelCode, fmt.Sprintf("Class %s extends final class %s.", className, parent.Name)))
						}
						classReadonly := hasClassModifier(n, "readonly")
						if classReadonly && !parent.Readonly {
							issues = append(issues, issueSpan(filename, n, level0ClassModelCode, fmt.Sprintf("Readonly class %s cannot extend non-readonly class %s.", className, parent.Name)))
						}
						if !classReadonly && parent.Readonly {
							issues = append(issues, issueSpan(filename, n, level0ClassModelCode, fmt.Sprintf("Non-readonly class %s cannot extend readonly class %s.", className, parent.Name)))
						}
					}
				}
				for _, implemented := range n.Implements {
					ifaceName := ft.resolveClassLike(implemented)
					if iface, ok := ctx.Resolver.ResolveClass(ifaceName); !ok {
						issues = append(issues, issueSpan(filename, n, level0ClassModelCode, fmt.Sprintf("Class %s implements unknown interface %s.", className, ifaceName)))
					} else if iface.Kind != "interface" {
						issues = append(issues, issueSpan(filename, n, level0ClassModelCode, fmt.Sprintf("Class %s implements %s %s.", className, iface.Kind, iface.Name)))
					}
				}
				checkClassMethodLegality(filename, className, n, ctx, &issues)
				checkConsistentConstructorLegality(filename, className, n, ctx, &issues)
				checkClassConstantLegality(filename, className, n, ctx, &issues)
				checkReadonlyClassProperties(filename, className, n, ctx, &issues)
				walk(n.Properties, ft, className)
				walk(n.Methods, ft, className)
			case *ast.InterfaceNode:
				interfaceName := ft.resolveClassLike(n.Name)
				for _, parent := range n.Extends {
					parentName := ft.resolveClassLike(parent)
					if resolved, ok := ctx.Resolver.ResolveClass(parentName); !ok {
						issues = append(issues, issueSpan(filename, n, level0ClassModelCode, fmt.Sprintf("Interface %s extends unknown interface %s.", interfaceName, parentName)))
					} else if resolved.Kind != "interface" {
						issues = append(issues, issueSpan(filename, n, level0ClassModelCode, fmt.Sprintf("Interface %s extends %s %s.", interfaceName, resolved.Kind, resolved.Name)))
					}
				}
				checkInterfaceMemberLegality(filename, interfaceName, n, &issues)
			case *ast.TraitUseNode:
				for _, trait := range n.Traits {
					traitName := ft.resolveClassLike(trait)
					if resolved, ok := ctx.Resolver.ResolveClass(traitName); !ok {
						issues = append(issues, issueSpan(filename, n, level0ClassModelCode, fmt.Sprintf("Trait %s not found.", traitName)))
					} else if resolved.Kind != "trait" {
						issues = append(issues, issueSpan(filename, n, level0ClassModelCode, fmt.Sprintf("%s %s used as trait.", titleKind(resolved.Kind), resolved.Name)))
					}
				}
			case *ast.EnumNode:
				enumName := ft.resolveClassLike(n.Name)
				checkEnumLegality(filename, enumName, n, &issues)
				walk(n.Methods, ft, enumName)
			}
		}
	}
	walk(nodes, fileCtx, "")
	return issues
}

func checkClassMethodLegality(filename, className string, class *ast.ClassNode, ctx *AnalysisContext, issues *[]AnalysisIssue) {
	isAbstractClass := hasClassModifier(class, "abstract")
	for _, methodNode := range class.Methods {
		method, ok := methodNode.(*ast.FunctionNode)
		if !ok {
			continue
		}
		if strings.EqualFold(method.Name, "__construct") && method.ReturnType != "" {
			*issues = append(*issues, issueSpan(filename, method, level0ClassModelCode, fmt.Sprintf("Constructor %s::__construct() cannot have a return type.", className)))
		}
		if hasModifier(method.Modifiers, "abstract") {
			if !isAbstractClass {
				*issues = append(*issues, issueSpan(filename, method, level0ClassModelCode, fmt.Sprintf("Class %s has abstract method %s() but is not abstract.", className, method.Name)))
			}
			if hasModifier(method.Modifiers, "private") {
				*issues = append(*issues, issueSpan(filename, method, level0ClassModelCode, fmt.Sprintf("Abstract method %s::%s() cannot be private.", className, method.Name)))
			}
			if hasModifier(method.Modifiers, "final") {
				*issues = append(*issues, issueSpan(filename, method, level0ClassModelCode, fmt.Sprintf("Abstract method %s::%s() cannot be final.", className, method.Name)))
			}
			continue
		}
		if parentMethod, ok := finalMethodInAncestors(ctx.Resolver, className, method.Name); ok {
			*issues = append(*issues, issueSpan(filename, method, level0ClassModelCode, fmt.Sprintf("Cannot override final method %s::%s().", parentMethod.DeclaringClass, parentMethod.Name)))
		}
	}
	if !isAbstractClass {
		checkRequiredMethodImplementations(filename, className, class.GetPos(), ctx, issues)
	}
}

func checkClassConstantLegality(filename, className string, class *ast.ClassNode, ctx *AnalysisContext, issues *[]AnalysisIssue) {
	for _, constNode := range class.Constants {
		constant, ok := constNode.(*ast.ConstantNode)
		if !ok {
			continue
		}
		if hasModifier(constant.Modifiers, "final") && constant.Visibility == "private" {
			*issues = append(*issues, issueSpan(filename, constant, level0ClassModelCode, fmt.Sprintf("Private constant %s::%s cannot be final.", className, constant.Name)))
		}
		if parentConstant, ok := finalConstantInAncestors(ctx.Resolver, className, constant.Name); ok {
			*issues = append(*issues, issueSpan(filename, constant, level0ClassModelCode, fmt.Sprintf("Cannot override final constant %s::%s.", parentConstant.DeclaringClass, parentConstant.Name)))
		}
	}
}

func checkConsistentConstructorLegality(filename, className string, class *ast.ClassNode, ctx *AnalysisContext, issues *[]AnalysisIssue) {
	if hasPHPStanConsistentConstructorTag(class.PHPDoc) && hasPrivateConstructor(ctx.Resolver, className) && !hasClassModifier(class, "final") {
		*issues = append(*issues, issueSpan(filename, class, level0ClassModelCode, fmt.Sprintf("Class %s has @phpstan-consistent-constructor but its constructor is private.", className)))
	}
	required, ok := consistentConstructorInAncestors(ctx.Resolver, className)
	if !ok {
		return
	}
	implemented, ok := ownConstructor(ctx.Resolver, className)
	if !ok {
		return
	}
	pos := constructorPos(class)
	checkConstructorVisibilityCompatibility(filename, pos, className, required, implemented, issues)
	checkRequiredMethodSignature(filename, pos, className, required, implemented, ctx, issues)
}

func checkInterfaceMemberLegality(filename, interfaceName string, iface *ast.InterfaceNode, issues *[]AnalysisIssue) {
	for _, member := range iface.Members {
		switch n := member.(type) {
		case *ast.InterfaceMethodNode:
			if n.Visibility != "" && n.Visibility != "public" {
				*issues = append(*issues, issueSpan(filename, n, level0ClassModelCode, fmt.Sprintf("Interface method %s::%s() must be public.", interfaceName, n.Name)))
			}
		case *ast.ConstantNode:
			if n.Visibility != "" && n.Visibility != "public" {
				*issues = append(*issues, issueSpan(filename, n, level0ClassModelCode, fmt.Sprintf("Interface constant %s::%s must be public.", interfaceName, n.Name)))
			}
		}
	}
}

func checkReadonlyClassProperties(filename, className string, class *ast.ClassNode, ctx *AnalysisContext, issues *[]AnalysisIssue) {
	classReadonly := hasClassModifier(class, "readonly")
	for _, propNode := range class.Properties {
		property, ok := propNode.(*ast.PropertyNode)
		if !ok {
			continue
		}
		if class.Extends != "" {
			parentName := ""
			if resolved, ok := ctx.Resolver.ResolveClass(className); ok && len(resolved.Extends) > 0 {
				parentName = resolved.Extends[0]
			}
			if parentName == "" {
				continue
			}
			if parentProperty, ok := ctx.Resolver.ResolveProperty(parentName, property.Name); ok && parentProperty.Readonly && !property.IsReadonly && !classReadonly {
				*issues = append(*issues, issueSpan(filename, property, level0ClassModelCode, fmt.Sprintf("Property %s::$%s overriding readonly property must be readonly.", className, property.Name)))
			}
		}
	}
}

// ancestorRecursionGuard bounds recursive ancestor walks so a self-referential
// or cyclic "extends" chain in indexed (possibly invalid) PHP source cannot
// cause unbounded recursion and crash the analyser with a stack overflow.
func finalMethodInAncestors(resolver SymbolResolver, className, methodName string) (ResolvedMethod, bool) {
	return finalMethodInAncestorsSeen(resolver, className, methodName, map[string]struct{}{})
}

func finalMethodInAncestorsSeen(resolver SymbolResolver, className, methodName string, seen map[string]struct{}) (ResolvedMethod, bool) {
	if resolver == nil {
		return ResolvedMethod{}, false
	}
	key := indexKey(className)
	if _, visited := seen[key]; visited {
		return ResolvedMethod{}, false
	}
	seen[key] = struct{}{}
	class, ok := resolver.ResolveClass(className)
	if !ok {
		return ResolvedMethod{}, false
	}
	for _, parent := range class.Extends {
		if method, ok := resolveOwnMethodView(resolver, parent, methodName); ok && method.Final {
			return method, true
		}
		if method, ok := finalMethodInAncestorsSeen(resolver, parent, methodName, seen); ok {
			return method, true
		}
	}
	return ResolvedMethod{}, false
}

func finalConstantInAncestors(resolver SymbolResolver, className, constName string) (ResolvedConstant, bool) {
	return finalConstantInAncestorsSeen(resolver, className, constName, map[string]struct{}{})
}

func finalConstantInAncestorsSeen(resolver SymbolResolver, className, constName string, seen map[string]struct{}) (ResolvedConstant, bool) {
	if resolver == nil {
		return ResolvedConstant{}, false
	}
	key := indexKey(className)
	if _, visited := seen[key]; visited {
		return ResolvedConstant{}, false
	}
	seen[key] = struct{}{}
	class, ok := resolver.ResolveClass(className)
	if !ok {
		return ResolvedConstant{}, false
	}
	for _, parent := range class.Extends {
		if constant, ok := resolver.ResolveOwnConstant(parent, constName); ok && constant.Final && constant.Visibility != "private" {
			return constant, true
		}
		if constant, ok := finalConstantInAncestorsSeen(resolver, parent, constName, seen); ok {
			return constant, true
		}
	}
	return ResolvedConstant{}, false
}

func consistentConstructorInAncestors(resolver SymbolResolver, className string) (ResolvedMethod, bool) {
	return consistentConstructorInAncestorsSeen(resolver, className, map[string]struct{}{})
}

func consistentConstructorInAncestorsSeen(resolver SymbolResolver, className string, seen map[string]struct{}) (ResolvedMethod, bool) {
	if resolver == nil {
		return ResolvedMethod{}, false
	}
	key := indexKey(className)
	if _, visited := seen[key]; visited {
		return ResolvedMethod{}, false
	}
	seen[key] = struct{}{}
	class, ok := resolver.ResolveClass(className)
	if !ok {
		return ResolvedMethod{}, false
	}
	for _, parent := range class.Extends {
		if parentClass, ok := resolver.ResolveClass(parent); ok && parentClass.ConsistentConstructor {
			constructor, ok := resolveOwnMethodView(resolver, parent, "__construct")
			if !ok {
				constructor = ResolvedMethod{Name: "__construct", DeclaringClass: parent, Visibility: "public"}
			}
			constructor.DeclaringClass = parent
			return constructor, true
		}
		if constructor, ok := consistentConstructorInAncestorsSeen(resolver, parent, seen); ok {
			return constructor, true
		}
	}
	return ResolvedMethod{}, false
}

func hasClassModifier(class *ast.ClassNode, modifier string) bool {
	for _, part := range strings.Fields(class.Modifier) {
		if strings.EqualFold(part, modifier) {
			return true
		}
	}
	return false
}

func hasPHPStanConsistentConstructorTag(doc *ast.PHPDocNode) bool {
	return doc != nil && strings.Contains(doc.RawContent, "@phpstan-consistent-constructor")
}

func ownConstructor(resolver SymbolResolver, className string) (ResolvedMethod, bool) {
	if resolver == nil {
		return ResolvedMethod{}, false
	}
	return resolveOwnMethodView(resolver, className, "__construct")
}

func hasPrivateConstructor(resolver SymbolResolver, className string) bool {
	constructor, ok := ownConstructor(resolver, className)
	return ok && constructor.Visibility == "private"
}

func constructorPos(class *ast.ClassNode) ast.Position {
	for _, methodNode := range class.Methods {
		method, ok := methodNode.(*ast.FunctionNode)
		if ok && strings.EqualFold(method.Name, "__construct") {
			return method.GetPos()
		}
	}
	return class.GetPos()
}

func checkConstructorVisibilityCompatibility(filename string, pos ast.Position, className string, required, implemented ResolvedMethod, issues *[]AnalysisIssue) {
	if visibilityRank(implemented.Visibility) < visibilityRank(required.Visibility) {
		*issues = append(*issues, issue(filename, pos, level0ClassModelCode, fmt.Sprintf("Constructor %s::__construct() visibility must be at least as visible as %s::__construct().", className, required.DeclaringClass)))
	}
}

func visibilityRank(visibility string) int {
	switch visibility {
	case "private":
		return 1
	case "protected":
		return 2
	default:
		return 3
	}
}

func checkRequiredMethodImplementations(filename, className string, pos ast.Position, ctx *AnalysisContext, issues *[]AnalysisIssue) {
	resolver := ctx.Resolver
	if resolver == nil {
		return
	}
	class, ok := resolver.ResolveClass(className)
	if !ok || class.Kind != "class" {
		return
	}
	required := map[string]ResolvedMethod{}
	for _, iface := range class.Implements {
		collectAbstractMethods(resolver, iface, required)
	}
	for _, parent := range class.Extends {
		collectUnimplementedParentAbstractMethods(resolver, parent, required)
	}
	for _, method := range required {
		implemented, ok := findConcreteClassMethod(resolver, className, method.Name)
		if !ok {
			*issues = append(*issues, issue(filename, pos, level0ClassModelCode, fmt.Sprintf("Class %s must implement method %s().", className, method.Name)))
			continue
		}
		if method.Visibility == "public" && implemented.Visibility != "public" {
			*issues = append(*issues, issue(filename, pos, level0ClassModelCode, fmt.Sprintf("Method %s::%s() implementing interface method must be public.", className, method.Name)))
		}
		checkRequiredMethodSignature(filename, pos, className, method, implemented, ctx, issues)
	}
}

func checkRequiredMethodSignature(filename string, pos ast.Position, className string, required, implemented ResolvedMethod, ctx *AnalysisContext, issues *[]AnalysisIssue) {
	requiredMin, requiredMax, requiredVariadic := parameterBounds(required.Params)
	implementedMin, implementedMax, implementedVariadic := parameterBounds(implemented.Params)
	if implementedMin > requiredMin {
		*issues = append(*issues, issue(filename, pos, level0ClassModelCode, fmt.Sprintf("Method %s::%s() requires more required parameters than the inherited method.", className, implemented.Name)))
	}
	if !implementedVariadic && (requiredVariadic || implementedMax < requiredMax) {
		*issues = append(*issues, issue(filename, pos, level0ClassModelCode, fmt.Sprintf("Method %s::%s() accepts fewer parameters than the inherited method.", className, implemented.Name)))
	}
	for idx, requiredParam := range required.Params {
		if idx >= len(implemented.Params) {
			break
		}
		implementedParam := implemented.Params[idx]
		if requiredParam.Name != "" && implementedParam.Name != "" && requiredParam.Name != implementedParam.Name {
			*issues = append(*issues, issue(filename, pos, level0ClassModelCode, fmt.Sprintf("Parameter %d of method %s::%s() is named $%s, expected $%s.", idx+1, className, implemented.Name, implementedParam.Name, requiredParam.Name)))
		}
	}
	// Native return-type covariance is a PHP engine (fatal-error) rule, not a
	// PHPDoc one: PHP only enforces it when both the ancestor and the
	// implementation declare a native return type. Comparing PHPDoc-derived
	// types here would flag implementations that PHP itself accepts, e.g. an
	// ancestor with only a PHPDoc `@return X[]` imposes no native contract.
	if required.NativeReturnType != "" && implemented.NativeReturnType != "" && !returnTypeCompatible(required.NativeReturnType, implemented.NativeReturnType, ctx) {
		*issues = append(*issues, issue(filename, pos, level0ClassModelCode, fmt.Sprintf("Return type %s of method %s::%s() is not compatible with inherited return type %s.", implemented.NativeReturnType, className, implemented.Name, required.NativeReturnType)))
	}
}

func returnTypeCompatible(required, implemented string, ctx *AnalysisContext) bool {
	if strings.EqualFold(required, implemented) {
		return true
	}
	return ParseType(required).AcceptsWithContext(ParseType(implemented), nil, ctx)
}

func collectAbstractMethods(resolver SymbolResolver, className string, out map[string]ResolvedMethod) {
	collectAbstractMethodsSeen(resolver, className, out, map[string]struct{}{})
}

func collectAbstractMethodsSeen(resolver SymbolResolver, className string, out map[string]ResolvedMethod, seen map[string]struct{}) {
	key := indexKey(className)
	if _, visited := seen[key]; visited {
		return
	}
	seen[key] = struct{}{}
	class, ok := resolver.ResolveClass(className)
	if !ok {
		return
	}
	for _, parent := range class.Extends {
		collectAbstractMethodsSeen(resolver, parent, out, seen)
	}
	for _, iface := range class.Implements {
		collectAbstractMethodsSeen(resolver, iface, out, seen)
	}
	rangeMethodsDeclaredBy(resolver, className, func(method ResolvedMethod) bool {
		if method.Abstract {
			out[asciiLowerIdent(method.Name)] = method
		}
		return true
	})
}

func collectUnimplementedParentAbstractMethods(resolver SymbolResolver, className string, out map[string]ResolvedMethod) {
	collectUnimplementedParentAbstractMethodsSeen(resolver, className, out, map[string]struct{}{})
}

func collectUnimplementedParentAbstractMethodsSeen(resolver SymbolResolver, className string, out map[string]ResolvedMethod, seen map[string]struct{}) {
	key := indexKey(className)
	if _, visited := seen[key]; visited {
		return
	}
	seen[key] = struct{}{}
	class, ok := resolver.ResolveClass(className)
	if !ok || class.Kind != "class" {
		return
	}
	for _, parent := range class.Extends {
		collectUnimplementedParentAbstractMethodsSeen(resolver, parent, out, seen)
	}
	rangeMethodsDeclaredBy(resolver, className, func(method ResolvedMethod) bool {
		key := asciiLowerIdent(method.Name)
		if method.Abstract {
			out[key] = method
		} else {
			delete(out, key)
		}
		return true
	})
}

func findConcreteClassMethod(resolver SymbolResolver, className, methodName string) (ResolvedMethod, bool) {
	seen := map[string]struct{}{}
	for className != "" {
		key := indexKey(className)
		if _, ok := seen[key]; ok {
			return ResolvedMethod{}, false
		}
		seen[key] = struct{}{}
		if method, ok := resolveOwnMethodView(resolver, className, methodName); ok && !method.Abstract {
			return method, true
		}
		class, ok := resolver.ResolveClass(className)
		if !ok || len(class.Extends) == 0 {
			return ResolvedMethod{}, false
		}
		className = class.Extends[0]
	}
	return ResolvedMethod{}, false
}
