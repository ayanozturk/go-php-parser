package analyse

import (
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

// GenericInstance represents a concrete instantiation of a generic class.
// E.g., Collection<User> has ClassName="Collection", TypeArguments=["User"]
type GenericInstance struct {
	ClassName     string
	TypeArguments []string
}

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

// methodReferenceParamResolver is an internal allocation-light query for
// analyses that only inspect parameter reference metadata. Implementations
// return immutable index-owned params; callers must not retain or mutate them.
type methodReferenceParamResolver interface {
	methodReferenceParams(className, methodName string) ([]ResolvedParam, bool)
}

// functionViewResolver is an internal allocation-light query for analyses
// that only read function metadata. Implementations return immutable
// index-owned parameter storage; callers must not retain or mutate it.
type functionViewResolver interface {
	resolveFunctionView(name string) (ResolvedFunction, bool)
}

// methodViewResolver is an internal allocation-light query for analyses that
// only read method metadata. Implementations return immutable index-owned
// parameter storage and skip generic template rewriting; callers must not
// retain or mutate the result, and type-sensitive inference must keep using
// ResolveMethod / ResolveMethodWithGenerics.
type methodViewResolver interface {
	resolveMethodView(className, methodName string) (ResolvedMethod, bool)
	resolveOwnMethodView(className, methodName string) (ResolvedMethod, bool)
}

type genericMethodResolver interface {
	ResolveMethodWithGenerics(className, methodName string, typeArguments []string) (ResolvedMethod, bool)
}

func resolveMethodWithGenerics(resolver SymbolResolver, className, methodName string, typeArguments []string) (ResolvedMethod, bool) {
	if resolver == nil {
		return ResolvedMethod{}, false
	}
	if len(typeArguments) > 0 {
		if generic, ok := resolver.(genericMethodResolver); ok {
			return generic.ResolveMethodWithGenerics(className, methodName, typeArguments)
		}
	}
	return resolver.ResolveMethod(className, methodName)
}

func resolveFunctionView(resolver SymbolResolver, name string) (ResolvedFunction, bool) {
	if resolver == nil {
		return ResolvedFunction{}, false
	}
	if viewResolver, ok := resolver.(functionViewResolver); ok {
		return viewResolver.resolveFunctionView(name)
	}
	return resolver.ResolveFunction(name)
}

func resolveMethodView(resolver SymbolResolver, className, methodName string) (ResolvedMethod, bool) {
	if resolver == nil {
		return ResolvedMethod{}, false
	}
	if viewResolver, ok := resolver.(methodViewResolver); ok {
		return viewResolver.resolveMethodView(className, methodName)
	}
	return resolver.ResolveMethod(className, methodName)
}

func resolveOwnMethodView(resolver SymbolResolver, className, methodName string) (ResolvedMethod, bool) {
	if resolver == nil {
		return ResolvedMethod{}, false
	}
	if viewResolver, ok := resolver.(methodViewResolver); ok {
		return viewResolver.resolveOwnMethodView(className, methodName)
	}
	return resolver.ResolveOwnMethod(className, methodName)
}

func resolveMethodReferenceParams(resolver SymbolResolver, className, methodName string) ([]ResolvedParam, bool) {
	if resolver == nil {
		return nil, false
	}
	if referenceResolver, ok := resolver.(methodReferenceParamResolver); ok {
		return referenceResolver.methodReferenceParams(className, methodName)
	}
	method, ok := resolver.ResolveMethod(className, methodName)
	if !ok {
		return nil, false
	}
	return method.Params, true
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
	TemplateBounds        []string
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
	// NativeReturnType holds only the PHP-declared return type (empty when
	// the method has no native return type, even if PHPDoc declares one).
	// PHP's return-type covariance rules apply solely to native
	// declarations, so LSP compatibility checks must use this field rather
	// than ReturnType, which PHPDoc can override for inference purposes.
	NativeReturnType   string
	CallableReturnType string
	Params             []ResolvedParam
	Visibility         string
	IsStatic           bool
	Abstract           bool
	Final              bool
	Deprecated         bool
	DeprecationMessage string
}

type ResolvedProperty struct {
	ID                 SymbolID
	Declaration        SourceLocation
	DeclaringClass     string
	Name               string
	Type               string
	CallableReturnType string
	Visibility         string
	IsStatic           bool
	Readonly           bool
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
	ID                 SymbolID
	Declaration        SourceLocation
	Name               string
	ReturnType         string
	CallableReturnType string
	Params             []ResolvedParam
	Deprecated         bool
	DeprecationMessage string
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

	// Content, when set by the caller, is the raw PHP source for filename and
	// selects the production CST-direct rule paths. The registry runner reuses
	// Parsed across the fused diagnostics, call/assignment checks, and
	// unreachable-code analysis. Shared per-file semantic facts remain
	// available to individual rules. Left nil, AST-based compatibility paths
	// remain available to tests and tools that supply nodes without source.
	Content []byte

	// Parsed is an optional shared *syntax.ParseResult for this file.
	// When set (or filled by sharedParseResult), CST-direct rules must
	// reuse it instead of calling syntax.Parse again. Callers that already
	// parsed for LowerAST should set Parsed to avoid a second parse.
	Parsed *syntax.ParseResult

	syntaxLower *syntaxLowerMemo // cross-walk CST Lower* memo for this file
	preLowered  *preLoweredIndex // ingest []ast.Node span index for CST Lower* reuse

	syntaxRootFt     FileTypeContext // CollectFileTypeContextFromSyntax(root), once per parse
	hasSyntaxRootFt  bool

	FileTypeContext     FileTypeContext
	hasFileTypeContext  bool
	phpDocTypeAliases   map[string]struct{}
	functionScopeByNode map[*ast.FunctionNode]*functionScope
	classScopeByNode    map[*ast.ClassNode]classScopeData

	// Cached once per file so the level-2/7/8 method-receiver rules share one
	// flow-sensitive walk and one type inference per call site.
	methodReceiverIssues    []AnalysisIssue
	hasMethodReceiverIssues bool
	namespaceContextByNode  map[*ast.NamespaceNode]FileTypeContext

	argTypeIssues         []AnalysisIssue
	argCountIssues        []AnalysisIssue
	argCountSink          *[]AnalysisIssue
	deprecatedCallIssues  []AnalysisIssue
	deprecatedCallSink    *[]AnalysisIssue
	deprecatedCallSeen    map[ast.Node]struct{}
	hasArgCallDiagnostics bool

	reflectionGuards             reflectionGuards
	hasReflectionGuards          bool
	level0Issues                 []AnalysisIssue
	level0PropertyCallableIssues []AnalysisIssue
	hasLevel0Issues              bool
	emptyStatementIssues         []AnalysisIssue

	methodVisibilityIssues  []AnalysisIssue
	throwTypeIssues         []AnalysisIssue
	returnTypeIssues        []AnalysisIssue
	phpDocIssues            []AnalysisIssue
	missingTypeIssues       []AnalysisIssue
	hasStructuralIssues     bool
	hasReturnTypeIssues     bool
	assignmentTypeIssues    []AnalysisIssue
	assignmentTypeSink      *[]AnalysisIssue
	hasAssignmentTypeIssues bool
}

// releaseEphemeralAnalysisState clears per-file caches populated during
// RunAnalysisRulesWithContext so callers can reuse a long-lived AnalysisContext
// shell (Resolver, Facts, Flow, etc.) without retaining parse trees or issue slices.
func releaseEphemeralAnalysisState(ctx *AnalysisContext) {
	if ctx == nil {
		return
	}
	ctx.Content = nil
	ctx.Parsed = nil
	ctx.syntaxLower = nil
	ctx.preLowered = nil
	ctx.hasSyntaxRootFt = false
	ctx.syntaxRootFt = FileTypeContext{}
	ctx.hasFileTypeContext = false
	ctx.FileTypeContext = FileTypeContext{}
	ctx.phpDocTypeAliases = nil
	ctx.functionScopeByNode = nil
	ctx.classScopeByNode = nil
	ctx.namespaceContextByNode = nil

	ctx.methodReceiverIssues = nil
	ctx.hasMethodReceiverIssues = false
	ctx.argTypeIssues = nil
	ctx.argCountIssues = nil
	ctx.argCountSink = nil
	ctx.deprecatedCallIssues = nil
	ctx.deprecatedCallSink = nil
	ctx.deprecatedCallSeen = nil
	ctx.hasArgCallDiagnostics = false

	ctx.reflectionGuards = reflectionGuards{}
	ctx.hasReflectionGuards = false
	ctx.level0Issues = nil
	ctx.level0PropertyCallableIssues = nil
	ctx.hasLevel0Issues = false
	ctx.emptyStatementIssues = nil

	ctx.methodVisibilityIssues = nil
	ctx.throwTypeIssues = nil
	ctx.returnTypeIssues = nil
	ctx.phpDocIssues = nil
	ctx.missingTypeIssues = nil
	ctx.hasStructuralIssues = false
	ctx.hasReturnTypeIssues = false
	ctx.assignmentTypeIssues = nil
	ctx.assignmentTypeSink = nil
	ctx.hasAssignmentTypeIssues = false
}

func analysisLevelAtLeast(ctx *AnalysisContext, level int) bool {
	if ctx == nil || ctx.AnalysisLevel == nil {
		return true
	}
	return *ctx.AnalysisLevel >= level
}

func analysisFileTypeContext(ctx *AnalysisContext, nodes []ast.Node) FileTypeContext {
	if ctx == nil {
		return CollectFileTypeContext(nodes)
	}
	if !ctx.hasFileTypeContext {
		ctx.FileTypeContext = CollectFileTypeContext(nodes)
		ctx.hasFileTypeContext = true
	}
	return ctx.FileTypeContext
}

func analysisFunctionScope(ctx *AnalysisContext, class *ast.ClassNode, fn *ast.FunctionNode, typeCtx FileTypeContext) *functionScope {
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

func analysisClassScopeData(ctx *AnalysisContext, class *ast.ClassNode, typeCtx FileTypeContext) classScopeData {
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

func analysisClassScopeDataByName(ctx *AnalysisContext, className string, typeCtx FileTypeContext) (classScopeData, bool) {
	className = strings.TrimPrefix(strings.TrimSpace(className), `\`)
	if className == "" {
		return classScopeData{}, false
	}
	class, ok := typeCtx.ClassNodes[asciiLowerIdent(className)]
	if !ok {
		resolved := typeCtx.resolveClassLike(className)
		class, ok = typeCtx.ClassNodes[asciiLowerIdent(strings.TrimPrefix(resolved, `\`))]
		if !ok {
			return classScopeData{}, false
		}
	}
	return analysisClassScopeData(ctx, class, typeCtx), true
}
