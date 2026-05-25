package analyse

import (
	"fmt"
	"go-phpcs/ast"
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
