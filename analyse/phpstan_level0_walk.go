package analyse

import "github.com/ayanozturk/go-php-parser/ast"

type walkAllConfig struct {
	collectContext bool
	hasRoot        bool
	root           FileTypeContext
	namespaces     map[*ast.NamespaceNode]FileTypeContext
}

func walkAll(nodes []ast.Node, fn func(ast.Node, *ast.ClassNode, *ast.FunctionNode, FileTypeContext)) {
	walkAllConfigured(nodes, fn, walkAllConfig{collectContext: true})
}

func walkAllWithoutTypeContext(nodes []ast.Node, fn func(ast.Node)) {
	// Alias collection and similar one-off indexers may use this. Production
	// analysis rules must not add another full-tree pass; fuse into
	// ensureArgCallDiagnostics / the snapshot builders instead.
	walkAllConfigured(nodes, func(node ast.Node, _ *ast.ClassNode, _ *ast.FunctionNode, _ FileTypeContext) {
		fn(node)
	}, walkAllConfig{})
}

func walkAllWithFileContext(nodes []ast.Node, ft FileTypeContext, ctx *AnalysisContext, fn func(ast.Node, *ast.ClassNode, *ast.FunctionNode, FileTypeContext)) {
	cfg := walkAllConfig{collectContext: true, hasRoot: true, root: ft}
	if ctx != nil {
		if ctx.namespaceContextByNode == nil {
			ctx.namespaceContextByNode = make(map[*ast.NamespaceNode]FileTypeContext)
		}
		cfg.namespaces = ctx.namespaceContextByNode
	}
	walkAllConfigured(nodes, fn, cfg)
}

func namespaceTypeContext(n *ast.NamespaceNode, cache map[*ast.NamespaceNode]FileTypeContext) FileTypeContext {
	if cached, ok := cache[n]; ok {
		return cached
	}
	nft := CollectFileTypeContext(n.Body)
	if nft.Namespace == "" {
		nft.Namespace = n.Name
	}
	if cache != nil {
		cache[n] = nft
	}
	return nft
}

func walkAllConfigured(nodes []ast.Node, fn func(ast.Node, *ast.ClassNode, *ast.FunctionNode, FileTypeContext), cfg walkAllConfig) {
	var walk func(ast.Node, *ast.ClassNode, *ast.FunctionNode, FileTypeContext)
	walk = func(node ast.Node, class *ast.ClassNode, currentFn *ast.FunctionNode, ft FileTypeContext) {
		if node == nil {
			return
		}
		fn(node, class, currentFn, ft)
		switch n := node.(type) {
		case *ast.NamespaceNode:
			nft := ft
			if cfg.collectContext {
				nft = namespaceTypeContext(n, cfg.namespaces)
			}
			for _, child := range n.Body {
				walk(child, class, currentFn, nft)
			}
		case *ast.ClassNode:
			cft := ft
			for _, child := range n.Properties {
				walk(child, n, currentFn, cft)
			}
			for _, child := range n.Methods {
				walk(child, n, currentFn, cft)
			}
		case *ast.FunctionNode:
			for _, param := range n.Params {
				walk(param, class, currentFn, ft)
			}
			for _, child := range n.Body {
				walk(child, class, n, ft)
			}
		case *ast.InterfaceNode:
			for _, child := range n.Members {
				walk(child, class, currentFn, ft)
			}
		case *ast.TraitNode:
			traitClass := class
			if n.Name != nil {
				traitClass = &ast.ClassNode{Name: n.Name.Name}
			}
			for _, child := range n.Body {
				walk(child, traitClass, currentFn, ft)
			}
		case *ast.EnumNode:
			enumClass := &ast.ClassNode{Name: n.Name}
			for _, child := range n.Methods {
				walk(child, enumClass, currentFn, ft)
			}
		case *ast.ExpressionStmt:
			walk(n.Expr, class, currentFn, ft)
		case *ast.AssignmentNode:
			walk(n.Left, class, currentFn, ft)
			walk(n.Right, class, currentFn, ft)
		case *ast.ReturnNode:
			walk(n.Expr, class, currentFn, ft)
		case *ast.BreakNode:
			walk(n.Level, class, currentFn, ft)
		case *ast.ContinueNode:
			walk(n.Level, class, currentFn, ft)
		case *ast.ThrowNode:
			walk(n.Expr, class, currentFn, ft)
		case *ast.IfNode:
			walk(n.Condition, class, currentFn, ft)
			for _, child := range n.Body {
				walk(child, class, currentFn, ft)
			}
			for _, elseif := range n.ElseIfs {
				walk(elseif.Condition, class, currentFn, ft)
				for _, child := range elseif.Body {
					walk(child, class, currentFn, ft)
				}
			}
			if n.Else != nil {
				for _, child := range n.Else.Body {
					walk(child, class, currentFn, ft)
				}
			}
		case *ast.WhileNode:
			walk(n.Condition, class, currentFn, ft)
			for _, child := range n.Body {
				walk(child, class, currentFn, ft)
			}
		case *ast.DoWhileNode:
			for _, child := range n.Body {
				walk(child, class, currentFn, ft)
			}
			walk(n.Condition, class, currentFn, ft)
		case *ast.ForNode:
			for _, expression := range n.Init {
				walk(expression, class, currentFn, ft)
			}
			for _, condition := range n.Conditions {
				walk(condition, class, currentFn, ft)
			}
			for _, update := range n.Updates {
				walk(update, class, currentFn, ft)
			}
			for _, child := range n.Body {
				walk(child, class, currentFn, ft)
			}
		case *ast.ForeachNode:
			walk(n.Expr, class, currentFn, ft)
			walk(n.KeyVar, class, currentFn, ft)
			walk(n.ValueVar, class, currentFn, ft)
			for _, child := range n.Body {
				walk(child, class, currentFn, ft)
			}
		case *ast.TryNode:
			for _, child := range n.Body {
				walk(child, class, currentFn, ft)
			}
			for _, catchNode := range n.Catches {
				walk(catchNode, class, currentFn, ft)
			}
			for _, child := range n.Finally {
				walk(child, class, currentFn, ft)
			}
		case *ast.CatchNode:
			for _, child := range n.Body {
				walk(child, class, currentFn, ft)
			}
		case *ast.AttributeNode:
			for _, arg := range n.Arguments {
				walk(arg, class, currentFn, ft)
			}
		case *ast.StaticVarDeclNode:
			for _, entry := range n.Vars {
				walk(entry.Init, class, currentFn, ft)
			}
		case *ast.FunctionCallNode:
			walk(n.Name, class, currentFn, ft)
			for _, arg := range n.Args {
				walk(arg, class, currentFn, ft)
			}
		case *ast.MethodCallNode:
			walk(n.Object, class, currentFn, ft)
			for _, arg := range n.Args {
				walk(arg, class, currentFn, ft)
			}
		case *ast.NewNode:
			walk(n.ClassExpr, class, currentFn, ft)
			for _, arg := range n.Args {
				walk(arg, class, currentFn, ft)
			}
		case *ast.NamedArgumentNode:
			walk(n.Value, class, currentFn, ft)
		case *ast.UnpackedArgumentNode:
			walk(n.Expr, class, currentFn, ft)
		case *ast.BinaryExpr:
			walk(n.Left, class, currentFn, ft)
			walk(n.Right, class, currentFn, ft)
		case *ast.UnaryExpr:
			walk(n.Operand, class, currentFn, ft)
		case *ast.TernaryExpr:
			walk(n.Condition, class, currentFn, ft)
			walk(n.IfTrue, class, currentFn, ft)
			walk(n.IfFalse, class, currentFn, ft)
		case *ast.ArrayNode:
			for _, child := range n.Elements {
				walk(child, class, currentFn, ft)
			}
		case *ast.ArrayItemNode:
			walk(n.Key, class, currentFn, ft)
			walk(n.Value, class, currentFn, ft)
		case *ast.ArrayAccessNode:
			walk(n.Var, class, currentFn, ft)
			walk(n.Index, class, currentFn, ft)
		case *ast.PropertyFetchNode:
			walk(n.Object, class, currentFn, ft)
		case *ast.ConcatNode:
			for _, child := range n.Parts {
				walk(child, class, currentFn, ft)
			}
		case *ast.MatchNode:
			walk(n.Condition, class, currentFn, ft)
			for _, arm := range n.Arms {
				for _, condition := range arm.Conditions {
					walk(condition, class, currentFn, ft)
				}
				walk(arm.Body, class, currentFn, ft)
			}
		case *ast.ArrowFunctionNode:
			for _, param := range n.Params {
				walk(param, class, currentFn, ft)
			}
			walk(n.Expr, class, currentFn, ft)
		}
	}
	ft := FileTypeContext{}
	if cfg.hasRoot {
		ft = cfg.root
	} else if cfg.collectContext {
		ft = CollectFileTypeContext(nodes)
	}
	for _, node := range nodes {
		walk(node, nil, nil, ft)
	}
}

func isStaticMethod(fn *ast.FunctionNode) bool {
	return fn != nil && hasModifier(fn.Modifiers, "static")
}
