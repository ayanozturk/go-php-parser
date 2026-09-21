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
	classLike       map[redIdentity]*ast.ClassNode
	fnDecl          map[redIdentity]*ast.FunctionNode
	fnLike          map[redIdentity]*ast.FunctionNode
	expr            map[redIdentity]ast.Node
	propertyDecl    map[redIdentity][]ast.Node
	interfaceMethod map[redIdentity]*ast.InterfaceMethodNode
}

func ensureSyntaxLowerMemo(ctx *AnalysisContext) *syntaxLowerMemo {
	if ctx == nil {
		return nil
	}
	if ctx.syntaxLower == nil {
		ctx.syntaxLower = &syntaxLowerMemo{
			classLike:       make(map[redIdentity]*ast.ClassNode),
			fnDecl:          make(map[redIdentity]*ast.FunctionNode),
			fnLike:          make(map[redIdentity]*ast.FunctionNode),
			expr:            make(map[redIdentity]ast.Node),
			propertyDecl:    make(map[redIdentity][]ast.Node),
			interfaceMethod: make(map[redIdentity]*ast.InterfaceMethodNode),
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
			recordMemoLower(MemoBucketClassLike, MemoOutcomeHit)
			return cls
		}
		if idx := ctx.preLowered; idx != nil {
			switch n.Kind() {
			case syntax.KindClassDecl, syntax.KindAnonymousClass:
				start, end := syntax.ClassDeclSpanOffsets(n, file)
				if cls, ok := idx.class[byteSpan{start, end}]; ok {
					memo.classLike[id] = cls
					recordMemoLower(MemoBucketClassLike, MemoOutcomeBridge)
					return cls
				}
			}
		}
		cls := syntax.LowerClassLikeContextNode(n, file)
		memo.classLike[id] = cls
		recordMemoLower(MemoBucketClassLike, MemoOutcomeMiss)
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
			recordMemoLower(MemoBucketFnDecl, MemoOutcomeHit)
			return fn
		}
		if idx := ctx.preLowered; idx != nil {
			start, end := syntax.FunctionDeclSpanOffsets(n, file)
			if fn, ok := idx.fn[byteSpan{start, end}]; ok {
				memo.fnDecl[id] = fn
				recordMemoLower(MemoBucketFnDecl, MemoOutcomeBridge)
				return fn
			}
		}
		fn := syntax.LowerFunctionDeclNode(n, file)
		memo.fnDecl[id] = fn
		recordMemoLower(MemoBucketFnDecl, MemoOutcomeMiss)
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
			recordMemoLower(MemoBucketFnLike, MemoOutcomeHit)
			return fn
		}
		if idx := ctx.preLowered; idx != nil {
			start, end := syntax.FunctionDeclSpanOffsets(n, file)
			if fn, ok := idx.fn[byteSpan{start, end}]; ok {
				memo.fnLike[id] = fn
				recordMemoLower(MemoBucketFnLike, MemoOutcomeBridge)
				return fn
			}
		}
		fn := syntax.LowerFunctionLikeContextNode(n, file)
		memo.fnLike[id] = fn
		recordMemoLower(MemoBucketFnLike, MemoOutcomeMiss)
		return fn
	}
	return syntax.LowerFunctionLikeContextNode(n, file)
}

func memoLowerPropertyDecl(ctx *AnalysisContext, n *syntax.RedNode, file *syntax.File) []ast.Node {
	if n == nil {
		return nil
	}
	if memo := ensureSyntaxLowerMemo(ctx); memo != nil {
		id := redID(n)
		if props, ok := memo.propertyDecl[id]; ok {
			recordMemoLower(MemoBucketPropertyDecl, MemoOutcomeHit)
			return props
		}
		props := syntax.LowerPropertyDeclNode(n, file)
		memo.propertyDecl[id] = props
		recordMemoLower(MemoBucketPropertyDecl, MemoOutcomeMiss)
		return props
	}
	return syntax.LowerPropertyDeclNode(n, file)
}

func memoLowerInterfaceMethod(ctx *AnalysisContext, n *syntax.RedNode, file *syntax.File) *ast.InterfaceMethodNode {
	if n == nil {
		return nil
	}
	if memo := ensureSyntaxLowerMemo(ctx); memo != nil {
		id := redID(n)
		if im, ok := memo.interfaceMethod[id]; ok {
			recordMemoLower(MemoBucketInterfaceMethod, MemoOutcomeHit)
			return im
		}
		im := syntax.LowerInterfaceMethodDeclNode(n, file)
		memo.interfaceMethod[id] = im
		recordMemoLower(MemoBucketInterfaceMethod, MemoOutcomeMiss)
		return im
	}
	return syntax.LowerInterfaceMethodDeclNode(n, file)
}

func memoLowerExpr(ctx *AnalysisContext, n *syntax.RedNode, file *syntax.File) ast.Node {
	if n == nil {
		return nil
	}
	if memo := ensureSyntaxLowerMemo(ctx); memo != nil {
		id := redID(n)
		if v, ok := memo.expr[id]; ok {
			recordMemoLower(MemoBucketExpr, MemoOutcomeHit)
			return v
		}
		if idx := ctx.preLowered; idx != nil && n.Kind() == syntax.KindCallExpr {
			start, end := syntax.ContentSpanOffsets(n, file)
			if mc, ok := idx.methodCall[byteSpan{start, end}]; ok {
				memo.expr[id] = mc
				recordMemoLower(MemoBucketExpr, MemoOutcomeBridge)
				return mc
			}
		}
		if n.Kind() == syntax.KindCallExpr {
			if v, ok := tryCSTCallExprForMemo(n, file); ok {
				memo.expr[id] = v
				recordMemoLower(MemoBucketExpr, MemoOutcomeCST)
				return v
			}
		}
		if n.Kind() == syntax.KindNewExpr {
			if v, ok := tryCSTNewExprForMemo(n); ok {
				memo.expr[id] = v
				recordMemoLower(MemoBucketExpr, MemoOutcomeCST)
				return v
			}
		}
		switch n.Kind() {
		case syntax.KindMemberAccessExpr, syntax.KindNullsafeMemberAccessExpr:
			if v, ok := tryCSTMemberAccessForMemo(n); ok {
				memo.expr[id] = v
				recordMemoLower(MemoBucketExpr, MemoOutcomeCST)
				return v
			}
		case syntax.KindStaticMemberAccessExpr:
			if v, ok := tryCSTStaticMemberForMemo(n); ok {
				memo.expr[id] = v
				recordMemoLower(MemoBucketExpr, MemoOutcomeCST)
				return v
			}
		}
		v := syntax.LowerExprNode(n, file)
		memo.expr[id] = v
		recordMemoLower(MemoBucketExpr, MemoOutcomeMiss)
		recordMemoLowerExprMiss(n.Kind())
		return v
	}
	return syntax.LowerExprNode(n, file)
}
