package analyse

import (
	"fmt"
	"go-phpcs/ast"
	"strings"
)

func (r *PHPStanLevel0Rule) checkClassModel(filename string, nodes []ast.Node, ctx *AnalysisContext, fileCtx fileTypeContext) []AnalysisIssue {
	var issues []AnalysisIssue
	for _, duplicate := range ctx.Project.Duplicates {
		if duplicate.File == filename {
			issues = append(issues, issue(filename, duplicate.Pos, level0ClassModelCode, fmt.Sprintf("Duplicate declaration of class %s.", duplicate.Name)))
		}
	}

	var walk func([]ast.Node, fileTypeContext, string)
	walk = func(nodes []ast.Node, ft fileTypeContext, currentClass string) {
		for _, node := range nodes {
			switch n := node.(type) {
			case *ast.NamespaceNode:
				nft := collectFileTypeContext(n.Body)
				if nft.namespace == "" {
					nft.namespace = n.Name
				}
				walk(n.Body, nft, currentClass)
			case *ast.ClassNode:
				className := ft.resolveClassLike(n.Name)
				if hasClassModifier(n, "final") && hasClassModifier(n, "abstract") {
					issues = append(issues, issue(filename, n.GetPos(), level0ClassModelCode, fmt.Sprintf("Class %s cannot be both final and abstract.", className)))
				}
				if n.Extends != "" {
					parentName := ft.resolveClassLike(n.Extends)
					if parent, ok := ctx.Resolver.ResolveClass(parentName); !ok {
						issues = append(issues, issue(filename, n.GetPos(), level0ClassModelCode, fmt.Sprintf("Class %s extends unknown class %s.", className, parentName)))
					} else if parent.Kind != "class" {
						issues = append(issues, issue(filename, n.GetPos(), level0ClassModelCode, fmt.Sprintf("Class %s extends %s %s.", className, parent.Kind, parent.Name)))
					} else {
						if parent.Final {
							issues = append(issues, issue(filename, n.GetPos(), level0ClassModelCode, fmt.Sprintf("Class %s extends final class %s.", className, parent.Name)))
						}
						classReadonly := hasClassModifier(n, "readonly")
						if classReadonly && !parent.Readonly {
							issues = append(issues, issue(filename, n.GetPos(), level0ClassModelCode, fmt.Sprintf("Readonly class %s cannot extend non-readonly class %s.", className, parent.Name)))
						}
						if !classReadonly && parent.Readonly {
							issues = append(issues, issue(filename, n.GetPos(), level0ClassModelCode, fmt.Sprintf("Non-readonly class %s cannot extend readonly class %s.", className, parent.Name)))
						}
					}
				}
				for _, implemented := range n.Implements {
					ifaceName := ft.resolveClassLike(implemented)
					if iface, ok := ctx.Resolver.ResolveClass(ifaceName); !ok {
						issues = append(issues, issue(filename, n.GetPos(), level0ClassModelCode, fmt.Sprintf("Class %s implements unknown interface %s.", className, ifaceName)))
					} else if iface.Kind != "interface" {
						issues = append(issues, issue(filename, n.GetPos(), level0ClassModelCode, fmt.Sprintf("Class %s implements %s %s.", className, iface.Kind, iface.Name)))
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
						issues = append(issues, issue(filename, n.GetPos(), level0ClassModelCode, fmt.Sprintf("Interface %s extends unknown interface %s.", interfaceName, parentName)))
					} else if resolved.Kind != "interface" {
						issues = append(issues, issue(filename, n.GetPos(), level0ClassModelCode, fmt.Sprintf("Interface %s extends %s %s.", interfaceName, resolved.Kind, resolved.Name)))
					}
				}
				checkInterfaceMemberLegality(filename, interfaceName, n, &issues)
			case *ast.TraitUseNode:
				for _, trait := range n.Traits {
					traitName := ft.resolveClassLike(trait)
					if resolved, ok := ctx.Resolver.ResolveClass(traitName); !ok {
						issues = append(issues, issue(filename, n.GetPos(), level0ClassModelCode, fmt.Sprintf("Trait %s not found.", traitName)))
					} else if resolved.Kind != "trait" {
						issues = append(issues, issue(filename, n.GetPos(), level0ClassModelCode, fmt.Sprintf("%s %s used as trait.", titleKind(resolved.Kind), resolved.Name)))
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
			*issues = append(*issues, issue(filename, method.GetPos(), level0ClassModelCode, fmt.Sprintf("Constructor %s::__construct() cannot have a return type.", className)))
		}
		if hasModifier(method.Modifiers, "abstract") {
			if !isAbstractClass {
				*issues = append(*issues, issue(filename, method.GetPos(), level0ClassModelCode, fmt.Sprintf("Class %s has abstract method %s() but is not abstract.", className, method.Name)))
			}
			if hasModifier(method.Modifiers, "private") {
				*issues = append(*issues, issue(filename, method.GetPos(), level0ClassModelCode, fmt.Sprintf("Abstract method %s::%s() cannot be private.", className, method.Name)))
			}
			if hasModifier(method.Modifiers, "final") {
				*issues = append(*issues, issue(filename, method.GetPos(), level0ClassModelCode, fmt.Sprintf("Abstract method %s::%s() cannot be final.", className, method.Name)))
			}
			continue
		}
		if parentMethod, ok := finalMethodInAncestors(ctx.Project, className, method.Name); ok {
			*issues = append(*issues, issue(filename, method.GetPos(), level0ClassModelCode, fmt.Sprintf("Cannot override final method %s::%s().", parentMethod.DeclaringClass, parentMethod.Name)))
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
			*issues = append(*issues, issue(filename, constant.GetPos(), level0ClassModelCode, fmt.Sprintf("Private constant %s::%s cannot be final.", className, constant.Name)))
		}
		if parentConstant, ok := finalConstantInAncestors(ctx.Project, className, constant.Name); ok {
			*issues = append(*issues, issue(filename, constant.GetPos(), level0ClassModelCode, fmt.Sprintf("Cannot override final constant %s::%s.", parentConstant.DeclaringClass, parentConstant.Name)))
		}
	}
}

func checkConsistentConstructorLegality(filename, className string, class *ast.ClassNode, ctx *AnalysisContext, issues *[]AnalysisIssue) {
	if hasPHPStanConsistentConstructorTag(class.PHPDoc) && hasPrivateConstructor(ctx.Project, className) && !hasClassModifier(class, "final") {
		*issues = append(*issues, issue(filename, class.GetPos(), level0ClassModelCode, fmt.Sprintf("Class %s has @phpstan-consistent-constructor but its constructor is private.", className)))
	}
	required, ok := consistentConstructorInAncestors(ctx.Project, className)
	if !ok {
		return
	}
	implemented, ok := ownConstructor(ctx.Project, className)
	if !ok {
		return
	}
	pos := constructorPos(class)
	checkConstructorVisibilityCompatibility(filename, pos, className, required, implemented, issues)
	checkRequiredMethodSignature(filename, pos, className, required, implemented, issues)
}

func checkInterfaceMemberLegality(filename, interfaceName string, iface *ast.InterfaceNode, issues *[]AnalysisIssue) {
	for _, member := range iface.Members {
		switch n := member.(type) {
		case *ast.InterfaceMethodNode:
			if n.Visibility != "" && n.Visibility != "public" {
				*issues = append(*issues, issue(filename, n.GetPos(), level0ClassModelCode, fmt.Sprintf("Interface method %s::%s() must be public.", interfaceName, n.Name)))
			}
		case *ast.ConstantNode:
			if n.Visibility != "" && n.Visibility != "public" {
				*issues = append(*issues, issue(filename, n.GetPos(), level0ClassModelCode, fmt.Sprintf("Interface constant %s::%s must be public.", interfaceName, n.Name)))
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
		if classReadonly && !property.IsReadonly {
			*issues = append(*issues, issue(filename, property.GetPos(), level0ClassModelCode, fmt.Sprintf("Readonly class %s cannot have non-readonly property $%s.", className, property.Name)))
		}
		if class.Extends != "" {
			parentName := ""
			if ctx.Project != nil {
				if resolved, ok := ctx.Project.ResolveClass(className); ok && len(resolved.Extends) > 0 {
					parentName = resolved.Extends[0]
				}
			}
			if parentName == "" {
				continue
			}
			if parentProperty, ok := ctx.Resolver.ResolveProperty(parentName, property.Name); ok && parentProperty.Readonly && !property.IsReadonly {
				*issues = append(*issues, issue(filename, property.GetPos(), level0ClassModelCode, fmt.Sprintf("Property %s::$%s overriding readonly property must be readonly.", className, property.Name)))
			}
		}
	}
	for _, methodNode := range class.Methods {
		fn, ok := methodNode.(*ast.FunctionNode)
		if !ok || !strings.EqualFold(fn.Name, "__construct") {
			continue
		}
		for _, paramNode := range fn.Params {
			param, ok := paramNode.(*ast.ParamNode)
			if !ok || !param.IsPromoted {
				continue
			}
			if classReadonly && !param.IsReadonly {
				*issues = append(*issues, issue(filename, param.GetPos(), level0ClassModelCode, fmt.Sprintf("Readonly class %s cannot have non-readonly property $%s.", className, param.Name)))
			}
		}
	}
}

func finalMethodInAncestors(project *ProjectIndex, className, methodName string) (ResolvedMethod, bool) {
	if project == nil {
		return ResolvedMethod{}, false
	}
	class, ok := project.ResolveClass(className)
	if !ok {
		return ResolvedMethod{}, false
	}
	for _, parent := range class.Extends {
		if method, ok := project.Methods[indexKey(parent)][strings.ToLower(methodName)]; ok && method.Final {
			return method, true
		}
		if method, ok := finalMethodInAncestors(project, parent, methodName); ok {
			return method, true
		}
	}
	return ResolvedMethod{}, false
}

func finalConstantInAncestors(project *ProjectIndex, className, constName string) (ResolvedConstant, bool) {
	if project == nil {
		return ResolvedConstant{}, false
	}
	class, ok := project.ResolveClass(className)
	if !ok {
		return ResolvedConstant{}, false
	}
	for _, parent := range class.Extends {
		if constant, ok := project.ClassConsts[indexKey(parent)][strings.ToLower(constName)]; ok && constant.Final && constant.Visibility != "private" {
			constant.DeclaringClass = parent
			return constant, true
		}
		if constant, ok := finalConstantInAncestors(project, parent, constName); ok {
			return constant, true
		}
	}
	return ResolvedConstant{}, false
}

func consistentConstructorInAncestors(project *ProjectIndex, className string) (ResolvedMethod, bool) {
	if project == nil {
		return ResolvedMethod{}, false
	}
	class, ok := project.ResolveClass(className)
	if !ok {
		return ResolvedMethod{}, false
	}
	for _, parent := range class.Extends {
		if parentClass, ok := project.ResolveClass(parent); ok && parentClass.ConsistentConstructor {
			constructor, ok := project.Methods[indexKey(parent)]["__construct"]
			if !ok {
				constructor = ResolvedMethod{Name: "__construct", DeclaringClass: parent, Visibility: "public"}
			}
			constructor.DeclaringClass = parent
			return constructor, true
		}
		if constructor, ok := consistentConstructorInAncestors(project, parent); ok {
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

func ownConstructor(project *ProjectIndex, className string) (ResolvedMethod, bool) {
	if project == nil {
		return ResolvedMethod{}, false
	}
	methods := project.Methods[indexKey(className)]
	if methods == nil {
		return ResolvedMethod{}, false
	}
	method, ok := methods["__construct"]
	return method, ok
}

func hasPrivateConstructor(project *ProjectIndex, className string) bool {
	constructor, ok := ownConstructor(project, className)
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
	project := ctx.Project
	if project == nil {
		return
	}
	class, ok := project.ResolveClass(className)
	if !ok || class.Kind != "class" {
		return
	}
	required := map[string]ResolvedMethod{}
	for _, iface := range class.Implements {
		collectAbstractMethods(project, iface, required)
	}
	for _, parent := range class.Extends {
		collectUnimplementedParentAbstractMethods(project, parent, required)
	}
	for _, method := range required {
		implemented, ok := findConcreteClassMethod(project, className, method.Name)
		if !ok {
			*issues = append(*issues, issue(filename, pos, level0ClassModelCode, fmt.Sprintf("Class %s must implement method %s().", className, method.Name)))
			continue
		}
		if method.Visibility == "public" && implemented.Visibility != "public" {
			*issues = append(*issues, issue(filename, pos, level0ClassModelCode, fmt.Sprintf("Method %s::%s() implementing interface method must be public.", className, method.Name)))
		}
		checkRequiredMethodSignature(filename, pos, className, method, implemented, issues)
	}
}

func checkRequiredMethodSignature(filename string, pos ast.Position, className string, required, implemented ResolvedMethod, issues *[]AnalysisIssue) {
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
	if required.ReturnType != "" && implemented.ReturnType != "" && !strings.EqualFold(required.ReturnType, implemented.ReturnType) {
		*issues = append(*issues, issue(filename, pos, level0ClassModelCode, fmt.Sprintf("Return type %s of method %s::%s() is not compatible with inherited return type %s.", implemented.ReturnType, className, implemented.Name, required.ReturnType)))
	}
}

func collectAbstractMethods(project *ProjectIndex, className string, out map[string]ResolvedMethod) {
	class, ok := project.ResolveClass(className)
	if !ok {
		return
	}
	for _, parent := range class.Extends {
		collectAbstractMethods(project, parent, out)
	}
	for _, iface := range class.Implements {
		collectAbstractMethods(project, iface, out)
	}
	for _, method := range project.Methods[indexKey(className)] {
		if method.Abstract {
			out[strings.ToLower(method.Name)] = method
		}
	}
}

func collectUnimplementedParentAbstractMethods(project *ProjectIndex, className string, out map[string]ResolvedMethod) {
	class, ok := project.ResolveClass(className)
	if !ok || class.Kind != "class" {
		return
	}
	for _, parent := range class.Extends {
		collectUnimplementedParentAbstractMethods(project, parent, out)
	}
	for _, method := range project.Methods[indexKey(className)] {
		key := strings.ToLower(method.Name)
		if method.Abstract {
			out[key] = method
		} else {
			delete(out, key)
		}
	}
}

func findConcreteClassMethod(project *ProjectIndex, className, methodName string) (ResolvedMethod, bool) {
	seen := map[string]struct{}{}
	for className != "" {
		key := indexKey(className)
		if _, ok := seen[key]; ok {
			return ResolvedMethod{}, false
		}
		seen[key] = struct{}{}
		if method, ok := project.Methods[key][strings.ToLower(methodName)]; ok && !method.Abstract {
			return method, true
		}
		class, ok := project.ResolveClass(className)
		if !ok || len(class.Extends) == 0 {
			return ResolvedMethod{}, false
		}
		className = class.Extends[0]
	}
	return ResolvedMethod{}, false
}
