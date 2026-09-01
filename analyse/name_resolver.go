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
	Constants  map[string]string
}

func CollectFileTypeContext(nodes []ast.Node) FileTypeContext {
	ctx := FileTypeContext{
		Aliases:    make(map[string]string),
		Classes:    make(map[string]ResolvedClass),
		ClassNodes: make(map[string]*ast.ClassNode),
		Constants:  make(map[string]string),
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
			ctx.Aliases[asciiLowerIdent(alias)] = strings.TrimPrefix(n.Path, `\`)
		case *ast.ConstantNode:
			if key, ok := literalArrayKey(n.Value); ok {
				ctx.Constants[n.Name] = key
			}
		case *ast.BlockNode:
			collectFileTypeContextFromNodes(n.Statements, namespace, ctx)
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
			key := asciiLowerIdent(strings.TrimPrefix(className, `\`))
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
			ctx.Classes[asciiLowerIdent(strings.TrimPrefix(interfaceName, `\`))] = resolved
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

	lower := asciiLowerIdent(name)
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
	if target, ok := aliases[asciiLowerIdent(firstSegment)]; ok {
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
	if class, ok := ctx.Classes[asciiLowerIdent(trimmed)]; ok {
		return class, true
	}
	resolved := ctx.resolveClassLike(trimmed)
	class, ok := ctx.Classes[asciiLowerIdent(strings.TrimPrefix(resolved, `\`))]
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

	return prefix + normalizeTypeExpressionWithContext(raw, ctx)
}

func normalizeTypeExpressionWithContext(raw string, ctx FileTypeContext) string {
	raw = stripBalancedOuterTypeParens(strings.TrimSpace(raw))
	if raw == "" {
		return ""
	}
	if parts := splitTopLevelTypes(raw, '|'); len(parts) > 1 {
		for idx, part := range parts {
			normalized := normalizeTypeExpressionWithContext(part, ctx)
			if len(splitTopLevelTypes(normalized, '&')) > 1 {
				normalized = "(" + normalized + ")"
			}
			parts[idx] = normalized
		}
		return strings.Join(parts, "|")
	}
	if parts := splitTopLevelTypes(raw, '&'); len(parts) > 1 {
		for idx, part := range parts {
			parts[idx] = normalizeTypeExpressionWithContext(part, ctx)
		}
		return strings.Join(parts, "&")
	}
	if instance, ok := parseGenericTypeFromString(raw); ok {
		name := normalizeTypeExpressionWithContext(instance.ClassName, ctx)
		args := make([]string, len(instance.TypeArguments))
		for idx, argument := range instance.TypeArguments {
			args[idx] = normalizeTypeExpressionWithContext(argument, ctx)
		}
		return name + "<" + strings.Join(args, ", ") + ">"
	}

	canonical := canonicalizeDocType(strings.TrimPrefix(raw, `\`))
	if len(splitTopLevelTypes(canonical, '|')) > 1 || len(splitTopLevelTypes(canonical, '&')) > 1 {
		return normalizeTypeExpressionWithContext(canonical, ctx)
	}
	atom, ok := normalizeTypeAtom(canonical)
	if ok && atom.kind == typeKindClass {
		return ctx.resolveClassLike(canonical)
	}
	return canonical
}

func unqualifiedTypeName(name string) string {
	name = strings.TrimPrefix(strings.TrimSpace(name), `\`)
	if idx := strings.LastIndex(name, `\`); idx >= 0 {
		return name[idx+1:]
	}
	return name
}
