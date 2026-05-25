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
					} else if parent.Final {
						issues = append(issues, issue(filename, n.GetPos(), level0ClassModelCode, fmt.Sprintf("Class %s extends final class %s.", className, parent.Name)))
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
				checkClassMethodLegality(filename, className, n, &issues)
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
			}
		}
	}
	walk(nodes, fileCtx, "")
	return issues
}

func checkClassMethodLegality(filename, className string, class *ast.ClassNode, issues *[]AnalysisIssue) {
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
		}
	}
}

func checkInterfaceMemberLegality(filename, interfaceName string, iface *ast.InterfaceNode, issues *[]AnalysisIssue) {
	for _, member := range iface.Members {
		method, ok := member.(*ast.InterfaceMethodNode)
		if !ok {
			continue
		}
		if method.Visibility != "" && method.Visibility != "public" {
			*issues = append(*issues, issue(filename, method.GetPos(), level0ClassModelCode, fmt.Sprintf("Interface method %s::%s() must be public.", interfaceName, method.Name)))
		}
	}
}

func hasClassModifier(class *ast.ClassNode, modifier string) bool {
	for _, part := range strings.Fields(class.Modifier) {
		if strings.EqualFold(part, modifier) {
			return true
		}
	}
	return false
}
