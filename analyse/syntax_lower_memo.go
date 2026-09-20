package analyse

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

type redIdentity struct {
	green  *syntax.GreenNode
	offset int
}

type syntaxLowerMemo struct {
	classLike map[redIdentity]*ast.ClassNode
	fnDecl    map[redIdentity]*ast.FunctionNode
	fnLike    map[redIdentity]*ast.FunctionNode
	expr      map[redIdentity]ast.Node
}

func ensureSyntaxLowerMemo(ctx *AnalysisContext) *syntaxLowerMemo {
	if ctx == nil {
		return nil
	}
	if ctx.syntaxLower == nil {
		ctx.syntaxLower = &syntaxLowerMemo{
			classLike: make(map[redIdentity]*ast.ClassNode),
			fnDecl:    make(map[redIdentity]*ast.FunctionNode),
			fnLike:    make(map[redIdentity]*ast.FunctionNode),
			expr:      make(map[redIdentity]ast.Node),
		}
	}
	return ctx.syntaxLower
}

func redID(n *syntax.RedNode) redIdentity {
	if n == nil {
		return redIdentity{}
	}
	return redIdentity{green: n.Green, offset: n.Offset}
}

func memoLowerClassLike(ctx *AnalysisContext, n *syntax.RedNode, file *syntax.File) *ast.ClassNode {
	if n == nil {
		return nil
	}
	if memo := ensureSyntaxLowerMemo(ctx); memo != nil {
		id := redID(n)
		if cls, ok := memo.classLike[id]; ok {
			return cls
		}
		if idx := ctx.preLowered; idx != nil {
			switch n.Kind() {
			case syntax.KindClassDecl, syntax.KindAnonymousClass:
				start, end := syntax.ClassDeclSpanOffsets(n, file)
				if cls, ok := idx.class[byteSpan{start, end}]; ok {
					memo.classLike[id] = cls
					return cls
				}
			}
		}
		cls := syntax.LowerClassLikeContextNode(n, file)
		memo.classLike[id] = cls
		return cls
	}
	return syntax.LowerClassLikeContextNode(n, file)
}

func memoLowerFunctionDecl(ctx *AnalysisContext, n *syntax.RedNode, file *syntax.File) *ast.FunctionNode {
	if n == nil {
		return nil
	}
	if memo := ensureSyntaxLowerMemo(ctx); memo != nil {
		id := redID(n)
		if fn, ok := memo.fnDecl[id]; ok {
			return fn
		}
		if idx := ctx.preLowered; idx != nil {
			start, end := syntax.FunctionDeclSpanOffsets(n, file)
			if fn, ok := idx.fn[byteSpan{start, end}]; ok {
				memo.fnDecl[id] = fn
				return fn
			}
		}
		fn := syntax.LowerFunctionDeclNode(n, file)
		memo.fnDecl[id] = fn
		return fn
	}
	return syntax.LowerFunctionDeclNode(n, file)
}

func memoLowerFunctionLike(ctx *AnalysisContext, n *syntax.RedNode, file *syntax.File) *ast.FunctionNode {
	if n == nil {
		return nil
	}
	if memo := ensureSyntaxLowerMemo(ctx); memo != nil {
		id := redID(n)
		if fn, ok := memo.fnLike[id]; ok {
			return fn
		}
		if idx := ctx.preLowered; idx != nil {
			start, end := syntax.FunctionDeclSpanOffsets(n, file)
			if fn, ok := idx.fn[byteSpan{start, end}]; ok {
				memo.fnLike[id] = fn
				return fn
			}
		}
		fn := syntax.LowerFunctionLikeContextNode(n, file)
		memo.fnLike[id] = fn
		return fn
	}
	return syntax.LowerFunctionLikeContextNode(n, file)
}

func memoLowerExpr(ctx *AnalysisContext, n *syntax.RedNode, file *syntax.File) ast.Node {
	if n == nil {
		return nil
	}
	if memo := ensureSyntaxLowerMemo(ctx); memo != nil {
		id := redID(n)
		if v, ok := memo.expr[id]; ok {
			return v
		}
		if idx := ctx.preLowered; idx != nil && n.Kind() == syntax.KindCallExpr {
			start, end := syntax.ContentSpanOffsets(n, file)
			if mc, ok := idx.methodCall[byteSpan{start, end}]; ok {
				memo.expr[id] = mc
				return mc
			}
		}
		v := syntax.LowerExprNode(n, file)
		memo.expr[id] = v
		return v
	}
	return syntax.LowerExprNode(n, file)
}
