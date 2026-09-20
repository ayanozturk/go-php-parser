package analyse

import "github.com/ayanozturk/go-php-parser/ast"

type byteSpan struct {
	start, end int
}

type preLoweredIndex struct {
	class      map[byteSpan]*ast.ClassNode
	fn         map[byteSpan]*ast.FunctionNode
	methodCall map[byteSpan]*ast.MethodCallNode
	built      bool
}

func ensurePreLoweredIndex(ctx *AnalysisContext, nodes []ast.Node) {
	if ctx == nil {
		return
	}
	if ctx.preLowered != nil && ctx.preLowered.built {
		return
	}
	idx := &preLoweredIndex{
		class:      make(map[byteSpan]*ast.ClassNode),
		fn:         make(map[byteSpan]*ast.FunctionNode),
		methodCall: make(map[byteSpan]*ast.MethodCallNode),
		built:      true,
	}
	if len(nodes) > 0 {
		walkAllWithoutTypeContext(nodes, func(node ast.Node) {
			end := node.GetEndPos()
			if end == (ast.Position{}) {
				return
			}
			start := node.GetPos()
			sp := byteSpan{start: start.Offset, end: end.Offset}
			switch n := node.(type) {
			case *ast.ClassNode:
				idx.class[sp] = n
			case *ast.FunctionNode:
				idx.fn[sp] = n
			case *ast.MethodCallNode:
				idx.methodCall[sp] = n
			}
		})
	}
	ctx.preLowered = idx
}
