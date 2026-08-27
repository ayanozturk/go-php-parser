package analyse

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"strings"
)

type FileTypeContext struct {
	Namespace  string
	Aliases    map[string]string
	Classes    map[string]ResolvedClass
	ClassNodes map[string]*ast.ClassNode
}

func CollectFileTypeContext(nodes []ast.Node) FileTypeContext {
	ctx := FileTypeContext{
		Aliases:    make(map[string]string),
		Classes:    make(map[string]ResolvedClass),
		ClassNodes: make(map[string]*ast.ClassNode),
	}
	collectFileTypeContextFromNodes(nodes, "", &ctx)
	return ctx
}

func collectFileTypeContextFromNodes(nodes []ast.Node, currentNS string, ctx *FileTypeContext) {
	namespace := currentNS
	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.NamespaceNode:
			if len(n.Body) > 0 {
				if ctx.Namespace == "" {
					ctx.Namespace = n.Name
				}
				collectFileTypeContextFromNodes(n.Body, n.Name, ctx)
				continue
			}
			namespace = n.Name
			if ctx.Namespace == "" {
				ctx.Namespace = n.Name
			}
		case *ast.UseNode:
			if n.Type != "" && n.Type != "class" {
				continue
			}
			alias := n.Alias
			if alias == "" {
				alias = unqualifiedTypeName(n.Path)
			}
			ctx.Aliases[strings.ToLower(alias)] = strings.TrimPrefix(n.Path, `\`)
		case *ast.ClassNode:
			className := resolveClassLikeInContext(namespace, ctx.Aliases, n.Name)
			resolved := ResolvedClass{Name: className}
			if n.Extends != "" {
				resolved.Extends = []string{resolveClassLikeInContext(namespace, ctx.Aliases, n.Extends)}
			}
			if len(n.Implements) > 0 {
				resolved.Implements = make([]string, 0, len(n.Implements))
				for _, implemented := range n.Implements {
					resolved.Implements = append(resolved.Implements, resolveClassLikeInContext(namespace, ctx.Aliases, implemented))
				}
			}
			key := strings.ToLower(strings.TrimPrefix(className, `\`))
			ctx.Classes[key] = resolved
			ctx.ClassNodes[key] = n
		case *ast.InterfaceNode:
			interfaceName := resolveClassLikeInContext(namespace, ctx.Aliases, n.Name)
			resolved := ResolvedClass{Name: interfaceName}
			if len(n.Extends) > 0 {
				resolved.Extends = make([]string, 0, len(n.Extends))
				for _, parent := range n.Extends {
					resolved.Extends = append(resolved.Extends, resolveClassLikeInContext(namespace, ctx.Aliases, parent))
				}
			}
			ctx.Classes[strings.ToLower(strings.TrimPrefix(interfaceName, `\`))] = resolved
		}
	}
	if ctx.Namespace == "" {
		ctx.Namespace = namespace
	}
}

func (ctx FileTypeContext) resolveClassLike(name string) string {
	return resolveClassLikeInContext(ctx.Namespace, ctx.Aliases, name)
}

func resolveClassLikeInContext(namespace string, aliases map[string]string, name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}

	lower := strings.ToLower(name)
	if lower == "self" || lower == "static" || lower == "parent" {
		return name
	}
	if strings.HasPrefix(name, `\`) {
		return strings.TrimPrefix(name, `\`)
	}

	firstSegment := name
	remainder := ""
	if idx := strings.Index(name, `\`); idx >= 0 {
		firstSegment = name[:idx]
		remainder = name[idx+1:]
	}
	if target, ok := aliases[strings.ToLower(firstSegment)]; ok {
		if remainder != "" {
			return target + `\` + remainder
		}
		return target
	}
	if namespace != "" {
		return namespace + `\` + name
	}
	return name
}

func (ctx FileTypeContext) resolveClass(name string) (ResolvedClass, bool) {
	if name == "" {
		return ResolvedClass{}, false
	}
	trimmed := strings.TrimPrefix(strings.TrimSpace(name), `\`)
	if class, ok := ctx.Classes[strings.ToLower(trimmed)]; ok {
		return class, true
	}
	resolved := ctx.resolveClassLike(trimmed)
	class, ok := ctx.Classes[strings.ToLower(strings.TrimPrefix(resolved, `\`))]
	return class, ok
}

func normalizeTypeWithContext(raw string, ctx FileTypeContext) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	prefix := ""
	if strings.HasPrefix(raw, "?") {
		prefix = "?"
		raw = strings.TrimSpace(strings.TrimPrefix(raw, "?"))
	}

	parts := splitTopLevelTypes(raw, '|')
	for idx, part := range parts {
		intersectionParts := splitTopLevelTypes(part, '&')
		for intersectionIdx, intersectionPart := range intersectionParts {
			intersectionPart = strings.TrimSpace(intersectionPart)
			if intersectionPart == "" {
				continue
			}
			intersectionPart = canonicalizeDocType(strings.TrimPrefix(intersectionPart, `\`))
			if len(splitTopLevelTypes(intersectionPart, '|')) > 1 {
				intersectionParts[intersectionIdx] = intersectionPart
				continue
			}
			atom, ok := normalizeTypeAtom(intersectionPart)
			if ok && atom.kind == typeKindClass {
				intersectionParts[intersectionIdx] = ctx.resolveClassLike(intersectionPart)
				continue
			}
			intersectionParts[intersectionIdx] = intersectionPart
		}
		parts[idx] = strings.Join(intersectionParts, "&")
	}

	return prefix + strings.Join(parts, "|")
}

func unqualifiedTypeName(name string) string {
	name = strings.TrimPrefix(strings.TrimSpace(name), `\`)
	if idx := strings.LastIndex(name, `\`); idx >= 0 {
		return name[idx+1:]
	}
	return name
}
