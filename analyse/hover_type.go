package analyse

import "go-phpcs/ast"

func InferTypeAtPosition(nodes []ast.Node, line, column int, ident string, ctx *AnalysisContext) (string, bool) {
	target, ok := InferHoverTargetAtPosition(nodes, line, column, ident, ctx)
	if !ok || target.Type == "" {
		return "", false
	}
	return target.Type, true
}

type HoverTargetKind string

const (
	HoverTargetUnknown  HoverTargetKind = ""
	HoverTargetLiteral  HoverTargetKind = "literal"
	HoverTargetVariable HoverTargetKind = "variable"
	HoverTargetProperty HoverTargetKind = "property"
	HoverTargetMethod   HoverTargetKind = "method"
	HoverTargetFunction HoverTargetKind = "function"
	HoverTargetClass    HoverTargetKind = "class"
)

type HoverTarget struct {
	Type          string
	Kind          HoverTargetKind
	ReceiverClass string
}

func InferHoverTargetAtPosition(nodes []ast.Node, line, column int, ident string, ctx *AnalysisContext) (HoverTarget, bool) {
	if ident == "" {
		return HoverTarget{}, false
	}

	query := hoverTypeQuery{line: line, column: column, ident: ident}
	match := findHoverTypeMatch(nodes, query, ctx)
	if !match.ok {
		return HoverTarget{}, false
	}

	target := HoverTarget{Kind: match.kind, ReceiverClass: match.receiverClass}
	if !match.typ.IsEmpty() {
		target.Type = match.typ.String()
	}
	return target, true
}

type hoverTypeQuery struct {
	line   int
	column int
	ident  string
}

type hoverTypeMatch struct {
	typ           Type
	kind          HoverTargetKind
	receiverClass string
	score         int
	column        int
	ok            bool
}

func findHoverTypeMatch(nodes []ast.Node, query hoverTypeQuery, ctx *AnalysisContext) hoverTypeMatch {
	var best hoverTypeMatch
	fileCtx := collectFileTypeContext(nodes)
	var walk func(node ast.Node, class *ast.ClassNode)

	walk = func(node ast.Node, class *ast.ClassNode) {
		switch n := node.(type) {
		case *ast.ClassNode:
			for _, methodNode := range n.Methods {
				walk(methodNode, n)
			}
		case *ast.FunctionNode:
			scope := newFunctionScope(class, n, fileCtx)
			walkStatementsForHoverTypes(n.Body, scope, ctx, query, &best)
		case *ast.NamespaceNode:
			for _, child := range n.Body {
				walk(child, class)
			}
		}
	}

	for _, node := range nodes {
		walk(node, nil)
	}

	return best
}

func walkStatementsForHoverTypes(nodes []ast.Node, scope *functionScope, ctx *AnalysisContext, query hoverTypeQuery, best *hoverTypeMatch) {
	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.ExpressionStmt:
			walkExprForHoverTypes(n.Expr, scope, ctx, query, best)
			applyExpressionScope(scope, n.Expr, ctx)
		case *ast.AssignmentNode:
			walkExprForHoverTypes(n, scope, ctx, query, best)
			applyAssignmentScope(scope, n, ctx)
		case *ast.ReturnNode:
			walkExprForHoverTypes(n.Expr, scope, ctx, query, best)
		case *ast.IfNode:
			walkExprForHoverTypes(n.Condition, scope, ctx, query, best)
			walkStatementsForHoverTypes(n.Body, scopeForConditionTrue(scope, n.Condition), ctx, query, best)
			for _, elseif := range n.ElseIfs {
				walkExprForHoverTypes(elseif.Condition, scope, ctx, query, best)
				walkStatementsForHoverTypes(elseif.Body, scopeForConditionTrue(scope, elseif.Condition), ctx, query, best)
			}
			if n.Else != nil {
				walkStatementsForHoverTypes(n.Else.Body, scope.clone(), ctx, query, best)
			}
		case *ast.BlockNode:
			walkStatementsForHoverTypes(n.Statements, scope.clone(), ctx, query, best)
		case *ast.WhileNode:
			walkExprForHoverTypes(n.Condition, scope, ctx, query, best)
			walkStatementsForHoverTypes(n.Body, scope.clone(), ctx, query, best)
		case *ast.ForeachNode:
			walkExprForHoverTypes(n.Expr, scope, ctx, query, best)
			walkStatementsForHoverTypes(n.Body, scope.clone(), ctx, query, best)
		}
	}
}

func walkExprForHoverTypes(node ast.Node, scope *functionScope, ctx *AnalysisContext, query hoverTypeQuery, best *hoverTypeMatch) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *ast.StringLiteral, *ast.InterpolatedStringLiteral, *ast.StringNode,
		*ast.IntegerLiteral, *ast.IntegerNode,
		*ast.FloatLiteral, *ast.FloatNode,
		*ast.BooleanLiteral, *ast.BooleanNode,
		*ast.NullLiteral, *ast.NullNode:
		considerHoverTypeMatch(n, n.TokenLiteral(), inferType(n, scope, ctx), HoverTargetLiteral, "", query, best)
	case *ast.VariableNode:
		considerHoverTypeMatch(n, n.Name, inferType(n, scope, ctx), HoverTargetVariable, "", query, best)
	case *ast.PropertyFetchNode:
		considerHoverTypeMatch(n, n.Property, inferType(n, scope, ctx), HoverTargetProperty, hoverReceiverClass(n.Object, scope, ctx), query, best)
		walkExprForHoverTypes(n.Object, scope, ctx, query, best)
	case *ast.MethodCallNode:
		considerHoverTypeMatch(n, n.Method, inferType(n, scope, ctx), HoverTargetMethod, hoverReceiverClass(n.Object, scope, ctx), query, best)
		walkExprForHoverTypes(n.Object, scope, ctx, query, best)
		for _, arg := range n.Args {
			walkExprForHoverTypes(argumentValue(arg), scope, ctx, query, best)
		}
	case *ast.FunctionCallNode:
		if identNode, ok := n.Name.(*ast.IdentifierNode); ok {
			considerHoverTypeMatch(n, identNode.Value, inferType(n, scope, ctx), HoverTargetFunction, "", query, best)
		}
		for _, arg := range n.Args {
			walkExprForHoverTypes(argumentValue(arg), scope, ctx, query, best)
		}
	case *ast.NewNode:
		className := n.ClassName
		if className == "" {
			if identNode, ok := n.ClassExpr.(*ast.IdentifierNode); ok {
				className = identNode.Value
			}
		}
		if className != "" {
			considerHoverTypeMatch(n, className, inferType(n, scope, ctx), HoverTargetClass, "", query, best)
		}
		for _, arg := range n.Args {
			walkExprForHoverTypes(argumentValue(arg), scope, ctx, query, best)
		}
	case *ast.AssignmentNode:
		walkExprForHoverTypes(n.Left, scope, ctx, query, best)
		walkExprForHoverTypes(n.Right, scope, ctx, query, best)
	case *ast.BinaryExpr:
		walkExprForHoverTypes(n.Left, scope, ctx, query, best)
		walkExprForHoverTypes(n.Right, scope, ctx, query, best)
	case *ast.ConcatNode:
		for _, part := range n.Parts {
			walkExprForHoverTypes(part, scope, ctx, query, best)
		}
	case *ast.TernaryExpr:
		walkExprForHoverTypes(n.Condition, scope, ctx, query, best)
		walkExprForHoverTypes(n.IfTrue, scope, ctx, query, best)
		walkExprForHoverTypes(n.IfFalse, scope, ctx, query, best)
	case *ast.NamedArgumentNode:
		walkExprForHoverTypes(n.Value, scope, ctx, query, best)
	case *ast.UnpackedArgumentNode:
		walkExprForHoverTypes(n.Expr, scope, ctx, query, best)
	}
}

func considerHoverTypeMatch(node ast.Node, ident string, typ Type, kind HoverTargetKind, receiverClass string, query hoverTypeQuery, best *hoverTypeMatch) {
	if node == nil || typ.IsEmpty() || ident != query.ident {
		return
	}
	pos := node.GetPos()
	if pos.Line != query.line {
		return
	}
	score := 0
	if pos.Column <= query.column {
		score = query.column - pos.Column
	} else {
		score = pos.Column - query.column + 1000
	}
	if !best.ok || score < best.score || (score == best.score && pos.Column > best.column) {
		best.typ = typ
		best.kind = kind
		best.receiverClass = receiverClass
		best.score = score
		best.column = pos.Column
		best.ok = true
	}
}

func receiverClassName(typ Type) string {
	className, ok := typ.SingleClassName()
	if !ok {
		return ""
	}
	return className
}

func hoverReceiverClass(node ast.Node, scope *functionScope, ctx *AnalysisContext) string {
	if variable, ok := node.(*ast.VariableNode); ok && variable.Name == "this" && scope != nil {
		return scope.className
	}
	return receiverClassName(inferType(node, scope, ctx))
}
