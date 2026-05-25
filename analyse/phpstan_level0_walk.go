package analyse

import "go-phpcs/ast"

func walkAll(nodes []ast.Node, fn func(ast.Node, *ast.ClassNode, fileTypeContext)) {
	var walk func(ast.Node, *ast.ClassNode, fileTypeContext)
	walk = func(node ast.Node, class *ast.ClassNode, ft fileTypeContext) {
		if node == nil {
			return
		}
		fn(node, class, ft)
		switch n := node.(type) {
		case *ast.NamespaceNode:
			nft := collectFileTypeContext(n.Body)
			if nft.namespace == "" {
				nft.namespace = n.Name
			}
			for _, child := range n.Body {
				walk(child, class, nft)
			}
		case *ast.ClassNode:
			cft := ft
			for _, child := range n.Properties {
				walk(child, n, cft)
			}
			for _, child := range n.Methods {
				walk(child, n, cft)
			}
		case *ast.FunctionNode:
			for _, param := range n.Params {
				walk(param, class, ft)
			}
			for _, child := range n.Body {
				walk(child, class, ft)
			}
		case *ast.InterfaceNode:
			for _, child := range n.Members {
				walk(child, class, ft)
			}
		case *ast.TraitNode:
			for _, child := range n.Body {
				walk(child, class, ft)
			}
		case *ast.EnumNode:
			for _, child := range n.Methods {
				walk(child, class, ft)
			}
		case *ast.ExpressionStmt:
			walk(n.Expr, class, ft)
		case *ast.AssignmentNode:
			walk(n.Left, class, ft)
			walk(n.Right, class, ft)
		case *ast.ReturnNode:
			walk(n.Expr, class, ft)
		case *ast.ThrowNode:
			walk(n.Expr, class, ft)
		case *ast.IfNode:
			walk(n.Condition, class, ft)
			for _, child := range n.Body {
				walk(child, class, ft)
			}
			for _, elseif := range n.ElseIfs {
				walk(elseif.Condition, class, ft)
				for _, child := range elseif.Body {
					walk(child, class, ft)
				}
			}
			if n.Else != nil {
				for _, child := range n.Else.Body {
					walk(child, class, ft)
				}
			}
		case *ast.WhileNode:
			walk(n.Condition, class, ft)
			for _, child := range n.Body {
				walk(child, class, ft)
			}
		case *ast.ForeachNode:
			walk(n.Expr, class, ft)
			walk(n.KeyVar, class, ft)
			walk(n.ValueVar, class, ft)
			for _, child := range n.Body {
				walk(child, class, ft)
			}
		case *ast.TryNode:
			for _, child := range n.Body {
				walk(child, class, ft)
			}
			for _, catchNode := range n.Catches {
				walk(catchNode, class, ft)
			}
			for _, child := range n.Finally {
				walk(child, class, ft)
			}
		case *ast.CatchNode:
			for _, child := range n.Body {
				walk(child, class, ft)
			}
		case *ast.AttributeNode:
			for _, arg := range n.Arguments {
				walk(arg, class, ft)
			}
		case *ast.StaticVarDeclNode:
			for _, entry := range n.Vars {
				walk(entry.Init, class, ft)
			}
		case *ast.FunctionCallNode:
			walk(n.Name, class, ft)
			for _, arg := range n.Args {
				walk(arg, class, ft)
			}
		case *ast.MethodCallNode:
			walk(n.Object, class, ft)
			for _, arg := range n.Args {
				walk(arg, class, ft)
			}
		case *ast.NewNode:
			walk(n.ClassExpr, class, ft)
			for _, arg := range n.Args {
				walk(arg, class, ft)
			}
		case *ast.NamedArgumentNode:
			walk(n.Value, class, ft)
		case *ast.UnpackedArgumentNode:
			walk(n.Expr, class, ft)
		case *ast.BinaryExpr:
			walk(n.Left, class, ft)
			walk(n.Right, class, ft)
		case *ast.UnaryExpr:
			walk(n.Operand, class, ft)
		case *ast.TernaryExpr:
			walk(n.Condition, class, ft)
			walk(n.IfTrue, class, ft)
			walk(n.IfFalse, class, ft)
		case *ast.ArrayNode:
			for _, child := range n.Elements {
				walk(child, class, ft)
			}
		case *ast.ArrayItemNode:
			walk(n.Key, class, ft)
			walk(n.Value, class, ft)
		case *ast.ArrayAccessNode:
			walk(n.Var, class, ft)
			walk(n.Index, class, ft)
		case *ast.PropertyFetchNode:
			walk(n.Object, class, ft)
		case *ast.ConcatNode:
			for _, child := range n.Parts {
				walk(child, class, ft)
			}
		case *ast.MatchNode:
			walk(n.Condition, class, ft)
			for _, arm := range n.Arms {
				for _, condition := range arm.Conditions {
					walk(condition, class, ft)
				}
				walk(arm.Body, class, ft)
			}
		case *ast.ArrowFunctionNode:
			for _, param := range n.Params {
				walk(param, class, ft)
			}
			walk(n.Expr, class, ft)
		}
	}
	ft := collectFileTypeContext(nodes)
	for _, node := range nodes {
		walk(node, nil, ft)
	}
}
