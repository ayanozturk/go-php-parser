package analyse

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"strings"
)

type SymbolResolver interface {
	ClassExists(name string) bool
	FunctionExists(name string) bool
	ConstantExists(name string) bool
	ResolveClass(name string) (ResolvedClass, bool)
	ResolveMethod(className, methodName string) (ResolvedMethod, bool)
	ResolveOwnMethod(className, methodName string) (ResolvedMethod, bool)
	MethodsDeclaredBy(className string) []ResolvedMethod
	ResolveProperty(className, propertyName string) (ResolvedProperty, bool)
	ResolveConstant(className, constantName string) (ResolvedConstant, bool)
	ResolveOwnConstant(className, constantName string) (ResolvedConstant, bool)
	ResolveFunction(name string) (ResolvedFunction, bool)
	DuplicateClasses(filename string) []DuplicateSymbol
}

type methodsDeclaredRanger interface {
	rangeMethodsDeclaredBy(className string, visit func(ResolvedMethod) bool)
}

func rangeMethodsDeclaredBy(resolver SymbolResolver, className string, visit func(ResolvedMethod) bool) {
	if resolver == nil || visit == nil {
		return
	}
	if ranger, ok := resolver.(methodsDeclaredRanger); ok {
		ranger.rangeMethodsDeclaredBy(className, visit)
		return
	}
	for _, method := range resolver.MethodsDeclaredBy(className) {
		if !visit(method) {
			return
		}
	}
}

type ResolvedClass struct {
	ID                    SymbolID
	Declaration           SourceLocation
	Name                  string
	Extends               []string
	Implements            []string
	TemplateParams        []string
	GenericParents        []ResolvedGenericParent
	Traits                []string
	Kind                  string
	Final                 bool
	Abstract              bool
	Readonly              bool
	ConsistentConstructor bool
}

// ResolvedGenericParent binds a class-like inheritance target to the type
// arguments supplied by an @extends or @implements PHPDoc annotation.
type ResolvedGenericParent struct {
	Name          string
	TypeArguments []string
}

type ResolvedMethod struct {
	ID             SymbolID
	Declaration    SourceLocation
	Name           string
	DeclaringClass string
	ReturnType     string
	Params         []ResolvedParam
	Visibility     string
	IsStatic       bool
	Abstract       bool
	Final          bool
}

type ResolvedProperty struct {
	ID             SymbolID
	Declaration    SourceLocation
	DeclaringClass string
	Name           string
	Type           string
	Visibility     string
	IsStatic       bool
	Readonly       bool
}

type ResolvedConstant struct {
	ID             SymbolID
	Declaration    SourceLocation
	Name           string
	DeclaringClass string
	Type           string
	Visibility     string
	Final          bool
}

type ResolvedFunction struct {
	ID          SymbolID
	Declaration SourceLocation
	Name        string
	ReturnType  string
	Params      []ResolvedParam
}

type ResolvedParam struct {
	Name       string
	Type       string
	HasDefault bool
	IsVariadic bool
	IsByRef    bool
	IsOut      bool // reference argument is defined without requiring an input read
}

type AnalysisContext struct {
	Resolver           SymbolResolver
	Facts              SemanticFactReader
	Flow               FlowGraphReader
	VariableFlow       VariableFlowReader
	PHPVersion         string
	AnalysisLevel      *int
	DisabledIssueCodes map[string]bool

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

func analysisClassScopeDataByName(ctx *AnalysisContext, className string, typeCtx fileTypeContext) (classScopeData, bool) {
	className = strings.TrimPrefix(strings.TrimSpace(className), `\`)
	if className == "" {
		return classScopeData{}, false
	}
	class, ok := typeCtx.classNodes[strings.ToLower(className)]
	if !ok {
		resolved := typeCtx.resolveClassLike(className)
		class, ok = typeCtx.classNodes[strings.ToLower(strings.TrimPrefix(resolved, `\`))]
		if !ok {
			return classScopeData{}, false
		}
	}
	return analysisClassScopeData(ctx, class, typeCtx), true
}
