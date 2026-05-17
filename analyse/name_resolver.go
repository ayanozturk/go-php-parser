package analyse

import (
	"go-phpcs/ast"
	"strings"
)

type fileTypeContext struct {
	namespace string
	aliases   map[string]string
	classes   map[string]ResolvedClass
}

func collectFileTypeContext(nodes []ast.Node) fileTypeContext {
	ctx := fileTypeContext{aliases: make(map[string]string), classes: make(map[string]ResolvedClass)}
	collectFileTypeContextFromNodes(nodes, "", &ctx)
	return ctx
}

func collectFileTypeContextFromNodes(nodes []ast.Node, currentNS string, ctx *fileTypeContext) {
	namespace := currentNS
	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.NamespaceNode:
			if len(n.Body) > 0 {
				if ctx.namespace == "" {
					ctx.namespace = n.Name
				}
				collectFileTypeContextFromNodes(n.Body, n.Name, ctx)
				continue
			}
			namespace = n.Name
			if ctx.namespace == "" {
				ctx.namespace = n.Name
			}
		case *ast.UseNode:
			if n.Type != "" && n.Type != "class" {
				continue
			}
			alias := n.Alias
			if alias == "" {
				alias = unqualifiedTypeName(n.Path)
			}
			ctx.aliases[strings.ToLower(alias)] = strings.TrimPrefix(n.Path, `\`)
		case *ast.ClassNode:
			className := resolveClassLikeInContext(namespace, ctx.aliases, n.Name)
			resolved := ResolvedClass{Name: className}
			if n.Extends != "" {
				resolved.Extends = []string{resolveClassLikeInContext(namespace, ctx.aliases, n.Extends)}
			}
			if len(n.Implements) > 0 {
				resolved.Implements = make([]string, 0, len(n.Implements))
				for _, implemented := range n.Implements {
					resolved.Implements = append(resolved.Implements, resolveClassLikeInContext(namespace, ctx.aliases, implemented))
				}
			}
			ctx.classes[strings.ToLower(strings.TrimPrefix(className, `\`))] = resolved
		case *ast.InterfaceNode:
			interfaceName := resolveClassLikeInContext(namespace, ctx.aliases, n.Name)
			resolved := ResolvedClass{Name: interfaceName}
			if len(n.Extends) > 0 {
				resolved.Extends = make([]string, 0, len(n.Extends))
				for _, parent := range n.Extends {
					resolved.Extends = append(resolved.Extends, resolveClassLikeInContext(namespace, ctx.aliases, parent))
				}
			}
			ctx.classes[strings.ToLower(strings.TrimPrefix(interfaceName, `\`))] = resolved
		}
	}
	if ctx.namespace == "" {
		ctx.namespace = namespace
	}
}

func (ctx fileTypeContext) resolveClassLike(name string) string {
	return resolveClassLikeInContext(ctx.namespace, ctx.aliases, name)
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

func (ctx fileTypeContext) resolveClass(name string) (ResolvedClass, bool) {
	if name == "" {
		return ResolvedClass{}, false
	}
	trimmed := strings.TrimPrefix(strings.TrimSpace(name), `\`)
	if class, ok := ctx.classes[strings.ToLower(trimmed)]; ok {
		return class, true
	}
	resolved := ctx.resolveClassLike(trimmed)
	class, ok := ctx.classes[strings.ToLower(strings.TrimPrefix(resolved, `\`))]
	return class, ok
}

func normalizeTypeWithContext(raw string, ctx fileTypeContext) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	prefix := ""
	if strings.HasPrefix(raw, "?") {
		prefix = "?"
		raw = strings.TrimSpace(strings.TrimPrefix(raw, "?"))
	}

	parts := strings.Split(raw, "|")
	for idx, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		atom, ok := normalizeTypeAtom(part)
		if ok && atom.kind == typeKindClass {
			parts[idx] = ctx.resolveClassLike(part)
			continue
		}
		parts[idx] = part
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
