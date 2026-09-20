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
		fn := syntax.LowerFunctionLikeContextNode(n, file)
		memo.fnLike[id] = fn
		return fn
	}
	return syntax.LowerFunctionLikeContextNode(n, file)
}
