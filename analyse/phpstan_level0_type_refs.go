package analyse

import (
	"fmt"
	"github.com/ayanozturk/go-php-parser/ast"
	"strings"
)

func (r *Level0Rule) checkTypeReferences(filename string, nodes []ast.Node, ctx *AnalysisContext, fileCtx FileTypeContext) []AnalysisIssue {
	var issues []AnalysisIssue
	guards := collectReflectionGuards(nodes, ctx, fileCtx)
	walkAllWithFileContext(nodes, fileCtx, ctx, func(node ast.Node, _ *ast.ClassNode, _ *ast.FunctionNode, ft FileTypeContext) {
		checkTypeReferenceOnNode(filename, node, ft, ctx, guards, &issues)
	})
	return issues
}

func checkTypeReferenceOnNode(filename string, node ast.Node, ft FileTypeContext, ctx *AnalysisContext, guards reflectionGuards, issues *[]AnalysisIssue) {
	switch n := node.(type) {
	case *ast.UseNode:
		switch n.Type {
		case "function":
			name := strings.TrimPrefix(n.Path, `\`)
			if !ctx.Resolver.FunctionExists(name) {
				if guards.hasFunction(name) {
					return
				}
				*issues = append(*issues, issueSpan(filename, n, level0SymbolsCode, fmt.Sprintf("Used function %s not found.", n.Path)))
			}
		case "const":
			name := strings.TrimPrefix(n.Path, `\`)
			if !ctx.Resolver.ConstantExists(name) {
				if guards.hasConstant(name) {
					return
				}
				*issues = append(*issues, issueSpan(filename, n, level0SymbolsCode, fmt.Sprintf("Used constant %s not found.", n.Path)))
			}
		default:
			// Class imports can legally alias a namespace prefix, especially for attributes
			// such as "use Doctrine\ORM\Mapping as ORM; #[ORM\Entity]".
			// Concrete class references are checked at their use sites.
		}
	case *ast.FunctionNode:
		for _, param := range n.Params {
			if p, ok := param.(*ast.ParamNode); ok {
				checkTypeReference(filename, p.GetPos(), "Parameter $"+p.Name, paramTypeName(p), ft, ctx, guards, issues)
			}
		}
		checkTypeReference(filename, n.GetPos(), "Return type", n.ReturnType, ft, ctx, guards, issues)
	case *ast.InterfaceMethodNode:
		for _, param := range n.Params {
			if p, ok := param.(*ast.ParamNode); ok {
				checkTypeReference(filename, p.GetPos(), "Parameter $"+p.Name, paramTypeName(p), ft, ctx, guards, issues)
			}
		}
		if n.ReturnType != nil {
			checkTypeReference(filename, n.GetPos(), "Return type", n.ReturnType.TokenLiteral(), ft, ctx, guards, issues)
		}
	case *ast.PropertyNode:
		checkTypeReference(filename, n.GetPos(), "Property $"+n.Name, n.TypeHint, ft, ctx, guards, issues)
	case *ast.ConstantNode:
		checkTypeReference(filename, n.GetPos(), "Constant "+n.Name, n.Type, ft, ctx, guards, issues)
	case *ast.CatchNode:
		for _, catchType := range n.Types {
			name := ft.resolveClassLike(catchType)
			resolved, ok := ctx.Resolver.ResolveClass(name)
			if !ok {
				if guards.hasClass(name) {
					continue
				}
				*issues = append(*issues, issueSpan(filename, n, level0SymbolsCode, fmt.Sprintf("Caught class %s not found.", name)))
				continue
			}
			if resolved.Kind == "trait" || resolved.Kind == "enum" {
				*issues = append(*issues, issueSpan(filename, n, level0ClassModelCode, fmt.Sprintf("Caught %s %s is not throwable.", resolved.Kind, resolved.Name)))
			}
		}
	case *ast.AttributeNode:
		name := ft.resolveClassLike(n.Name)
		_, ok := ctx.Resolver.ResolveClass(name)
		if !ok {
			if guards.hasClass(name) {
				return
			}
			*issues = append(*issues, issueSpan(filename, n, level0SymbolsCode, fmt.Sprintf("Attribute class %s not found.", name)))
			return
		}
	}
}

func checkTypeReference(filename string, pos ast.Position, subject, raw string, ft FileTypeContext, ctx *AnalysisContext, guards reflectionGuards, issues *[]AnalysisIssue) {
	for _, name := range referencedClassTypes(raw, ft) {
		if isSpecialClassName(name) {
			continue
		}
		if _, ok := ctx.Resolver.ResolveClass(name); !ok {
			if guards.hasClass(name) {
				continue
			}
			*issues = append(*issues, issue(filename, pos, level0SymbolsCode, fmt.Sprintf("%s references unknown class %s.", subject, name)))
		}
	}
}

func referencedClassTypes(raw string, ft FileTypeContext) []string {
	typ := ParseType(raw)
	if typ.IsEmpty() {
		return nil
	}
	var refs []string
	for _, atom := range typ.sortedAtoms() {
		if atom.kind != typeKindClass || strings.ContainsAny(atom.display, "$[]{}") {
			continue
		}
		refs = append(refs, ft.resolveClassLike(atom.display))
	}
	return refs
}

func paramTypeName(param *ast.ParamNode) string {
	if param.TypeHint != "" {
		return param.TypeHint
	}
	if param.UnionType != nil {
		return param.UnionType.TokenLiteral()
	}
	return ""
}
