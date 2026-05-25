package analyse

import (
	"fmt"
	"go-phpcs/ast"
	"strings"
)

func (r *PHPStanLevel0Rule) checkTypeReferences(filename string, nodes []ast.Node, ctx *AnalysisContext, fileCtx fileTypeContext) []AnalysisIssue {
	var issues []AnalysisIssue
	walkAll(nodes, func(node ast.Node, class *ast.ClassNode, ft fileTypeContext) {
		switch n := node.(type) {
		case *ast.UseNode:
			switch n.Type {
			case "function":
				if !ctx.Resolver.FunctionExists(strings.TrimPrefix(n.Path, `\`)) {
					issues = append(issues, issue(filename, n.GetPos(), level0SymbolsCode, fmt.Sprintf("Used function %s not found.", n.Path)))
				}
			case "const":
				if !ctx.Resolver.ConstantExists(strings.TrimPrefix(n.Path, `\`)) {
					issues = append(issues, issue(filename, n.GetPos(), level0SymbolsCode, fmt.Sprintf("Used constant %s not found.", n.Path)))
				}
			default:
				name := strings.TrimPrefix(n.Path, `\`)
				if _, ok := ctx.Resolver.ResolveClass(name); !ok {
					issues = append(issues, issue(filename, n.GetPos(), level0SymbolsCode, fmt.Sprintf("Used class %s not found.", name)))
				}
			}
		case *ast.FunctionNode:
			for _, param := range n.Params {
				if p, ok := param.(*ast.ParamNode); ok {
					checkTypeReference(filename, p.GetPos(), "Parameter $"+p.Name, paramTypeName(p), ft, ctx, &issues)
				}
			}
			checkTypeReference(filename, n.GetPos(), "Return type", n.ReturnType, ft, ctx, &issues)
		case *ast.InterfaceMethodNode:
			for _, param := range n.Params {
				if p, ok := param.(*ast.ParamNode); ok {
					checkTypeReference(filename, p.GetPos(), "Parameter $"+p.Name, paramTypeName(p), ft, ctx, &issues)
				}
			}
			if n.ReturnType != nil {
				checkTypeReference(filename, n.GetPos(), "Return type", n.ReturnType.TokenLiteral(), ft, ctx, &issues)
			}
		case *ast.PropertyNode:
			checkTypeReference(filename, n.GetPos(), "Property $"+n.Name, n.TypeHint, ft, ctx, &issues)
		case *ast.ConstantNode:
			checkTypeReference(filename, n.GetPos(), "Constant "+n.Name, n.Type, ft, ctx, &issues)
		case *ast.CatchNode:
			for _, catchType := range n.Types {
				name := ft.resolveClassLike(catchType)
				resolved, ok := ctx.Resolver.ResolveClass(name)
				if !ok {
					issues = append(issues, issue(filename, n.GetPos(), level0SymbolsCode, fmt.Sprintf("Caught class %s not found.", name)))
					continue
				}
				if resolved.Kind == "trait" || resolved.Kind == "enum" {
					issues = append(issues, issue(filename, n.GetPos(), level0ClassModelCode, fmt.Sprintf("Caught %s %s is not throwable.", resolved.Kind, resolved.Name)))
				}
			}
		case *ast.AttributeNode:
			name := ft.resolveClassLike(n.Name)
			resolved, ok := ctx.Resolver.ResolveClass(name)
			if !ok {
				issues = append(issues, issue(filename, n.GetPos(), level0SymbolsCode, fmt.Sprintf("Attribute class %s not found.", name)))
				return
			}
			checkCallArguments(filename, n.GetPos(), "Attribute class "+resolved.Name+" constructor", "__construct", n.Arguments, constructorFor(resolved.Name, ctx), &issues)
		}
	})
	return issues
}

func checkTypeReference(filename string, pos ast.Position, subject, raw string, ft fileTypeContext, ctx *AnalysisContext, issues *[]AnalysisIssue) {
	for _, name := range referencedClassTypes(raw, ft) {
		if isSpecialClassName(name) {
			continue
		}
		if _, ok := ctx.Resolver.ResolveClass(name); !ok {
			*issues = append(*issues, issue(filename, pos, level0SymbolsCode, fmt.Sprintf("%s references unknown class %s.", subject, name)))
		}
	}
}

func referencedClassTypes(raw string, ft fileTypeContext) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if strings.HasPrefix(raw, "?") {
		raw = strings.TrimSpace(strings.TrimPrefix(raw, "?"))
	}
	var refs []string
	for _, unionPart := range splitTopLevelTypes(raw, '|') {
		for _, part := range splitTopLevelTypes(unionPart, '&') {
			part = strings.TrimSpace(part)
			part = strings.Trim(part, "()")
			if part == "" {
				continue
			}
			canonical := canonicalizeDocType(strings.TrimPrefix(part, `\`))
			if atom, ok := normalizeTypeAtom(canonical); ok {
				if atom.kind == typeKindBuiltin {
					continue
				}
				part = atom.display
			}
			if strings.ContainsAny(part, "$[]{}") {
				continue
			}
			refs = append(refs, ft.resolveClassLike(part))
		}
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
