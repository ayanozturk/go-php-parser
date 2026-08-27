package analyse

import "github.com/ayanozturk/go-php-parser/ast"

func InferTypeAtPosition(nodes []ast.Node, line, column int, ident string, ctx *AnalysisContext) (string, bool) {
	return InferTypeAtPositionWithFilename(nodes, "", line, column, ident, ctx)
}

// InferTypeAtPositionWithFilename resolves a hover type against semantic facts
// from the same named file while preserving InferTypeAtPosition's fallback API.
func InferTypeAtPositionWithFilename(nodes []ast.Node, filename string, line, column int, ident string, ctx *AnalysisContext) (string, bool) {
	target, ok := InferHoverTargetAtPositionWithFilename(nodes, filename, line, column, ident, ctx)
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
	return InferHoverTargetAtPositionWithFilename(nodes, "", line, column, ident, ctx)
}

// InferHoverTargetAtPositionWithFilename resolves exact-span semantic facts
// for filename and falls back to live inference when no matching fact exists.
func InferHoverTargetAtPositionWithFilename(nodes []ast.Node, filename string, line, column int, ident string, ctx *AnalysisContext) (HoverTarget, bool) {
	if ident == "" {
		return HoverTarget{}, false
	}
	if ident == "new" {
		return HoverTarget{}, false
	}

	query := hoverTypeQuery{filename: filename, line: line, column: column, ident: ident}
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
	filename string
	line     int
	column   int
	ident    string
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
	fileCtx := CollectFileTypeContext(nodes)
	var walk func(node ast.Node, class *ast.ClassNode)

	walk = func(node ast.Node, class *ast.ClassNode) {
		switch n := node.(type) {
		case *ast.ClassNode:
			for _, methodNode := range n.Methods {
				walk(methodNode, n)
			}
		case *ast.FunctionNode:
			scope := analysisFunctionScope(ctx, class, n, fileCtx)
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
			walkExprForHoverTypes(n.Right, scope, ctx, query, best)
			applyAssignmentScope(scope, n, ctx)
			walkExprForHoverTypes(n.Left, scope, ctx, query, best)
		case *ast.ReturnNode:
			walkExprForHoverTypes(n.Expr, scope, ctx, query, best)
		case *ast.BreakNode:
			walkExprForHoverTypes(n.Level, scope, ctx, query, best)
		case *ast.ContinueNode:
			walkExprForHoverTypes(n.Level, scope, ctx, query, best)
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
			applyTerminatingIfFalseScope(scope, n)
			applyLazyInitPropertyScope(scope, n, ctx)
		case *ast.BlockNode:
			walkStatementsForHoverTypes(n.Statements, scope.clone(), ctx, query, best)
		case *ast.WhileNode:
			walkExprForHoverTypes(n.Condition, scope, ctx, query, best)
			walkStatementsForHoverTypes(n.Body, scope.clone(), ctx, query, best)
		case *ast.DoWhileNode:
			loopScope := scope.clone()
			walkStatementsForHoverTypes(n.Body, loopScope, ctx, query, best)
			walkExprForHoverTypes(n.Condition, loopScope, ctx, query, best)
		case *ast.ForNode:
			for _, expression := range n.Init {
				walkExprForHoverTypes(expression, scope, ctx, query, best)
				applyExpressionScope(scope, expression, ctx)
			}
			loopScope := scope.clone()
			for _, condition := range n.Conditions {
				walkExprForHoverTypes(condition, loopScope, ctx, query, best)
			}
			walkStatementsForHoverTypes(n.Body, loopScope, ctx, query, best)
			for _, update := range n.Updates {
				walkExprForHoverTypes(update, loopScope, ctx, query, best)
			}
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
		considerHoverTypeMatch(n, n.TokenLiteral(), inferTypeWithFacts(query.filename, n, scope, ctx), HoverTargetLiteral, "", query, best)
	case *ast.VariableNode:
		considerHoverTypeMatch(n, n.Name, inferTypeWithFacts(query.filename, n, scope, ctx), HoverTargetVariable, "", query, best)
	case *ast.PropertyFetchNode:
		considerHoverTypeMatch(n, n.Property, inferTypeWithFacts(query.filename, n, scope, ctx), HoverTargetProperty, hoverReceiverClass(n.Object, scope, ctx, query.filename), query, best)
		walkExprForHoverTypes(n.Object, scope, ctx, query, best)
	case *ast.MethodCallNode:
		considerHoverTypeMatch(n, n.Method, inferTypeWithFacts(query.filename, n, scope, ctx), HoverTargetMethod, hoverReceiverClass(n.Object, scope, ctx, query.filename), query, best)
		walkExprForHoverTypes(n.Object, scope, ctx, query, best)
		for _, arg := range n.Args {
			walkExprForHoverTypes(argumentValue(arg), scope, ctx, query, best)
		}
	case *ast.FunctionCallNode:
		if identNode, ok := n.Name.(*ast.IdentifierNode); ok {
			considerHoverTypeMatch(n, identNode.Value, inferTypeWithFacts(query.filename, n, scope, ctx), HoverTargetFunction, "", query, best)
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
			considerHoverTypeMatch(n, className, inferTypeWithFacts(query.filename, n, scope, ctx), HoverTargetClass, "", query, best)
		}
		for _, arg := range n.Args {
			walkExprForHoverTypes(argumentValue(arg), scope, ctx, query, best)
		}
	case *ast.AssignmentNode:
		walkExprForHoverTypes(n.Right, scope, ctx, query, best)
		assignedScope := scope.clone()
		applyAssignmentScope(assignedScope, n, ctx)
		walkExprForHoverTypes(n.Left, assignedScope, ctx, query, best)
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

func hoverReceiverClass(node ast.Node, scope *functionScope, ctx *AnalysisContext, filename string) string {
	if variable, ok := node.(*ast.VariableNode); ok && variable.Name == "this" && scope != nil {
		return scope.className
	}
	return receiverClassName(inferTypeWithFacts(filename, node, scope, ctx))
}
