package analyse

import "go-phpcs/ast"

type SymbolResolver interface {
	ClassExists(name string) bool
	ResolveClass(name string) (ResolvedClass, bool)
	ResolveMethod(className, methodName string) (ResolvedMethod, bool)
	ResolveProperty(className, propertyName string) (ResolvedProperty, bool)
}

type ResolvedClass struct {
	Name       string
	Extends    []string
	Implements []string
}

type ResolvedMethod struct {
	Name       string
	ReturnType string
	Params     []ResolvedParam
}

type ResolvedProperty struct {
	Name string
	Type string
}

type ResolvedParam struct {
	Name       string
	Type       string
	HasDefault bool
	IsVariadic bool
}

type AnalysisContext struct {
	Resolver SymbolResolver

	fileTypeContext     fileTypeContext
	hasFileTypeContext  bool
	functionScopeByNode map[*ast.FunctionNode]*functionScope
	classScopeByNode    map[*ast.ClassNode]classScopeData
}

func analysisFileTypeContext(ctx *AnalysisContext, nodes []ast.Node) fileTypeContext {
	if ctx == nil {
		return collectFileTypeContext(nodes)
	}
	if !ctx.hasFileTypeContext {
		ctx.fileTypeContext = collectFileTypeContext(nodes)
		ctx.hasFileTypeContext = true
	}
	return ctx.fileTypeContext
}

func analysisFunctionScope(ctx *AnalysisContext, class *ast.ClassNode, fn *ast.FunctionNode, typeCtx fileTypeContext) *functionScope {
	if ctx == nil || fn == nil {
		return newFunctionScope(class, fn, typeCtx)
	}
	if ctx.functionScopeByNode == nil {
		ctx.functionScopeByNode = make(map[*ast.FunctionNode]*functionScope)
	}
	if scope, ok := ctx.functionScopeByNode[fn]; ok {
		return scope.clone()
	}
	scope := newFunctionScopeWithContext(ctx, class, fn, typeCtx)
	ctx.functionScopeByNode[fn] = scope
	return scope.clone()
}

func analysisClassScopeData(ctx *AnalysisContext, class *ast.ClassNode, typeCtx fileTypeContext) classScopeData {
	if class == nil {
		return classScopeData{}
	}
	if ctx == nil {
		return buildClassScopeData(class, typeCtx)
	}
	if ctx.classScopeByNode == nil {
		ctx.classScopeByNode = make(map[*ast.ClassNode]classScopeData)
	}
	if data, ok := ctx.classScopeByNode[class]; ok {
		return data
	}
	data := buildClassScopeData(class, typeCtx)
	ctx.classScopeByNode[class] = data
	return data
}
