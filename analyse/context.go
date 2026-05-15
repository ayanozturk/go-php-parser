package analyse

type SymbolResolver interface {
	ClassExists(name string) bool
	ResolveClass(name string) (ResolvedClass, bool)
	ResolveMethod(className, methodName string) (ResolvedMethod, bool)
	ResolveProperty(className, propertyName string) (ResolvedProperty, bool)
}

type ResolvedClass struct {
	Name    string
	Extends []string
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
	Name string
	Type string
}

type AnalysisContext struct {
	Resolver SymbolResolver
}
