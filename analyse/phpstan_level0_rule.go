package analyse

import (
	"fmt"
	"go-phpcs/ast"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	level0SymbolsCode    = "PHPStan.Level0.Symbols"
	level0ClassModelCode = "PHPStan.Level0.ClassModel"
	level0InvocationCode = "PHPStan.Level0.Invocation"
	level0VariablesCode  = "PHPStan.Level0.Variables"
	level0LanguageCode   = "PHPStan.Level0.Language"
)

type PHPStanLevel0Rule struct{}

func (r *PHPStanLevel0Rule) CheckIssues(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
	ctx = ensureLevel0Context(filename, nodes, ctx)
	fileCtx := analysisFileTypeContext(ctx, nodes)
	var issues []AnalysisIssue
	issues = append(issues, r.checkClassModel(filename, nodes, ctx, fileCtx)...)
	issues = append(issues, r.checkTypeReferences(filename, nodes, ctx, fileCtx)...)
	issues = append(issues, r.checkSymbolsAndCalls(filename, nodes, ctx, fileCtx)...)
	issues = append(issues, r.checkUndefinedVariables(filename, nodes, ctx, fileCtx)...)
	issues = append(issues, r.checkLanguage(filename, nodes, ctx, fileCtx)...)
	return issues
}

func ensureLevel0Context(filename string, nodes []ast.Node, ctx *AnalysisContext) *AnalysisContext {
	if ctx == nil {
		ctx = &AnalysisContext{}
	}
	if ctx.Project == nil {
		ctx.Project = BuildProjectIndex(map[string][]ast.Node{filename: nodes})
	}
	if ctx.Resolver == nil {
		ctx.Resolver = ctx.Project
	}
	return ctx
}

func (r *PHPStanLevel0Rule) checkClassModel(filename string, nodes []ast.Node, ctx *AnalysisContext, fileCtx fileTypeContext) []AnalysisIssue {
	var issues []AnalysisIssue
	for _, duplicate := range ctx.Project.Duplicates {
		if duplicate.File == filename {
			issues = append(issues, issue(filename, duplicate.Pos, level0ClassModelCode, fmt.Sprintf("Duplicate declaration of class %s.", duplicate.Name)))
		}
	}

	var walk func([]ast.Node, fileTypeContext, string)
	walk = func(nodes []ast.Node, ft fileTypeContext, currentClass string) {
		for _, node := range nodes {
			switch n := node.(type) {
			case *ast.NamespaceNode:
				nft := collectFileTypeContext(n.Body)
				if nft.namespace == "" {
					nft.namespace = n.Name
				}
				walk(n.Body, nft, currentClass)
			case *ast.ClassNode:
				className := ft.resolveClassLike(n.Name)
				if n.Extends != "" {
					parentName := ft.resolveClassLike(n.Extends)
					if parent, ok := ctx.Resolver.ResolveClass(parentName); !ok {
						issues = append(issues, issue(filename, n.GetPos(), level0ClassModelCode, fmt.Sprintf("Class %s extends unknown class %s.", className, parentName)))
					} else if parent.Kind != "class" {
						issues = append(issues, issue(filename, n.GetPos(), level0ClassModelCode, fmt.Sprintf("Class %s extends %s %s.", className, parent.Kind, parent.Name)))
					} else if parent.Final {
						issues = append(issues, issue(filename, n.GetPos(), level0ClassModelCode, fmt.Sprintf("Class %s extends final class %s.", className, parent.Name)))
					}
				}
				for _, implemented := range n.Implements {
					ifaceName := ft.resolveClassLike(implemented)
					if iface, ok := ctx.Resolver.ResolveClass(ifaceName); !ok {
						issues = append(issues, issue(filename, n.GetPos(), level0ClassModelCode, fmt.Sprintf("Class %s implements unknown interface %s.", className, ifaceName)))
					} else if iface.Kind != "interface" {
						issues = append(issues, issue(filename, n.GetPos(), level0ClassModelCode, fmt.Sprintf("Class %s implements %s %s.", className, iface.Kind, iface.Name)))
					}
				}
				walk(n.Properties, ft, className)
				walk(n.Methods, ft, className)
			case *ast.InterfaceNode:
				interfaceName := ft.resolveClassLike(n.Name)
				for _, parent := range n.Extends {
					parentName := ft.resolveClassLike(parent)
					if resolved, ok := ctx.Resolver.ResolveClass(parentName); !ok {
						issues = append(issues, issue(filename, n.GetPos(), level0ClassModelCode, fmt.Sprintf("Interface %s extends unknown interface %s.", interfaceName, parentName)))
					} else if resolved.Kind != "interface" {
						issues = append(issues, issue(filename, n.GetPos(), level0ClassModelCode, fmt.Sprintf("Interface %s extends %s %s.", interfaceName, resolved.Kind, resolved.Name)))
					}
				}
			case *ast.TraitUseNode:
				for _, trait := range n.Traits {
					traitName := ft.resolveClassLike(trait)
					if resolved, ok := ctx.Resolver.ResolveClass(traitName); !ok {
						issues = append(issues, issue(filename, n.GetPos(), level0ClassModelCode, fmt.Sprintf("Trait %s not found.", traitName)))
					} else if resolved.Kind != "trait" {
						issues = append(issues, issue(filename, n.GetPos(), level0ClassModelCode, fmt.Sprintf("%s %s used as trait.", titleKind(resolved.Kind), resolved.Name)))
					}
				}
			}
		}
	}
	walk(nodes, fileCtx, "")
	return issues
}

func (r *PHPStanLevel0Rule) checkTypeReferences(filename string, nodes []ast.Node, ctx *AnalysisContext, fileCtx fileTypeContext) []AnalysisIssue {
	var issues []AnalysisIssue
	walkAll(nodes, func(node ast.Node, class *ast.ClassNode, ft fileTypeContext) {
		switch n := node.(type) {
		case *ast.UseNode:
			switch n.Type {
			case "function":
				if !ctx.Resolver.FunctionExists(strings.TrimPrefix(n.Path, `\`)) {
					issues = append(issues, issue(filename, n.GetPos(), level0SymbolsCode, fmt.Sprintf("Used function %s not found.", n.Path)))
				}
			case "const":
				if !ctx.Resolver.ConstantExists(strings.TrimPrefix(n.Path, `\`)) {
					issues = append(issues, issue(filename, n.GetPos(), level0SymbolsCode, fmt.Sprintf("Used constant %s not found.", n.Path)))
				}
			default:
				name := strings.TrimPrefix(n.Path, `\`)
				if _, ok := ctx.Resolver.ResolveClass(name); !ok {
					issues = append(issues, issue(filename, n.GetPos(), level0SymbolsCode, fmt.Sprintf("Used class %s not found.", name)))
				}
			}
		case *ast.FunctionNode:
			for _, param := range n.Params {
				if p, ok := param.(*ast.ParamNode); ok {
					checkTypeReference(filename, p.GetPos(), "Parameter $"+p.Name, paramTypeName(p), ft, ctx, &issues)
				}
			}
			checkTypeReference(filename, n.GetPos(), "Return type", n.ReturnType, ft, ctx, &issues)
		case *ast.InterfaceMethodNode:
			for _, param := range n.Params {
				if p, ok := param.(*ast.ParamNode); ok {
					checkTypeReference(filename, p.GetPos(), "Parameter $"+p.Name, paramTypeName(p), ft, ctx, &issues)
				}
			}
			if n.ReturnType != nil {
				checkTypeReference(filename, n.GetPos(), "Return type", n.ReturnType.TokenLiteral(), ft, ctx, &issues)
			}
		case *ast.PropertyNode:
			checkTypeReference(filename, n.GetPos(), "Property $"+n.Name, n.TypeHint, ft, ctx, &issues)
		case *ast.ConstantNode:
			checkTypeReference(filename, n.GetPos(), "Constant "+n.Name, n.Type, ft, ctx, &issues)
		case *ast.CatchNode:
			for _, catchType := range n.Types {
				name := ft.resolveClassLike(catchType)
				resolved, ok := ctx.Resolver.ResolveClass(name)
				if !ok {
					issues = append(issues, issue(filename, n.GetPos(), level0SymbolsCode, fmt.Sprintf("Caught class %s not found.", name)))
					continue
				}
				if resolved.Kind == "trait" || resolved.Kind == "enum" {
					issues = append(issues, issue(filename, n.GetPos(), level0ClassModelCode, fmt.Sprintf("Caught %s %s is not throwable.", resolved.Kind, resolved.Name)))
				}
			}
		case *ast.AttributeNode:
			name := ft.resolveClassLike(n.Name)
			resolved, ok := ctx.Resolver.ResolveClass(name)
			if !ok {
				issues = append(issues, issue(filename, n.GetPos(), level0SymbolsCode, fmt.Sprintf("Attribute class %s not found.", name)))
				return
			}
			checkCallArguments(filename, n.GetPos(), "Attribute class "+resolved.Name+" constructor", "__construct", n.Arguments, constructorFor(resolved.Name, ctx), &issues)
		}
	})
	return issues
}

func (r *PHPStanLevel0Rule) checkSymbolsAndCalls(filename string, nodes []ast.Node, ctx *AnalysisContext, fileCtx fileTypeContext) []AnalysisIssue {
	var issues []AnalysisIssue
	walkAll(nodes, func(node ast.Node, class *ast.ClassNode, ft fileTypeContext) {
		switch n := node.(type) {
		case *ast.NewNode:
			className := resolveNewClassName(n, ft)
			if className == "" || isSpecialClassName(className) {
				return
			}
			resolved, ok := ctx.Resolver.ResolveClass(className)
			if !ok {
				issues = append(issues, issue(filename, n.GetPos(), level0SymbolsCode, fmt.Sprintf("Instantiated class %s not found.", className)))
				return
			}
			switch resolved.Kind {
			case "interface", "trait", "enum":
				issues = append(issues, issue(filename, n.GetPos(), level0ClassModelCode, fmt.Sprintf("Cannot instantiate %s %s.", resolved.Kind, resolved.Name)))
			}
			if resolved.Abstract {
				issues = append(issues, issue(filename, n.GetPos(), level0ClassModelCode, fmt.Sprintf("Instantiated class %s is abstract.", resolved.Name)))
			}
			checkCallArguments(filename, n.GetPos(), "Class "+resolved.Name+" constructor", "__construct", n.Args, constructorFor(resolved.Name, ctx), &issues)
		case *ast.FunctionCallNode:
			name := functionCallName(n)
			if name == "" {
				return
			}
			if className, methodName, ok := strings.Cut(name, "::"); ok {
				resolvedClass := resolveClassLikeForCall(className, class, ft)
				if !isSpecialClassName(resolvedClass) {
					if _, ok := ctx.Resolver.ResolveClass(resolvedClass); !ok {
						issues = append(issues, issue(filename, n.GetPos(), level0SymbolsCode, fmt.Sprintf("Call to static method %s() on an unknown class %s.", methodName, resolvedClass)))
						return
					}
				}
				method, ok := ctx.Resolver.ResolveMethod(resolvedClass, methodName)
				if !ok {
					issues = append(issues, issue(filename, n.GetPos(), level0SymbolsCode, fmt.Sprintf("Call to an undefined static method %s::%s().", resolvedClass, methodName)))
					return
				}
				checkMethodVisibility(filename, n.GetPos(), method, resolvedClass, class, true, &issues)
				checkCallArguments(filename, n.GetPos(), "Static method "+resolvedClass+"::"+method.Name+"()", method.Name, n.Args, method, &issues)
				return
			}
			resolvedName := resolveFunctionNameForCall(name, ft, ctx)
			if !ctx.Resolver.FunctionExists(resolvedName) {
				issues = append(issues, issue(filename, n.GetPos(), level0SymbolsCode, fmt.Sprintf("Function %s not found.", name)))
				return
			}
			if fn, ok := ctx.Resolver.ResolveFunction(resolvedName); ok {
				checkCallArguments(filename, n.GetPos(), "Function "+fn.Name, fn.Name, n.Args, ResolvedMethod{Name: fn.Name, Params: fn.Params}, &issues)
			}
		case *ast.MethodCallNode:
			if receiver, ok := n.Object.(*ast.VariableNode); ok && receiver.Name == "this" {
				className := currentClassName(class, ft)
				if className == "" {
					issues = append(issues, issue(filename, n.GetPos(), level0SymbolsCode, fmt.Sprintf("Undefined variable: $%s", receiver.Name)))
					return
				}
				method, ok := ctx.Resolver.ResolveMethod(className, n.Method)
				if !ok {
					issues = append(issues, issue(filename, n.GetPos(), level0SymbolsCode, fmt.Sprintf("Call to an undefined method %s::%s().", className, n.Method)))
					return
				}
				checkMethodVisibility(filename, n.GetPos(), method, className, class, false, &issues)
				checkCallArguments(filename, n.GetPos(), "Method "+className+"::"+method.Name+"()", method.Name, n.Args, method, &issues)
			}
		case *ast.ClassConstFetchNode:
			className := resolveClassLikeForCall(n.Class, class, ft)
			if isSpecialClassName(className) || strings.HasPrefix(className, "$") {
				return
			}
			resolvedClass, ok := ctx.Resolver.ResolveClass(className)
			if !ok {
				issues = append(issues, issue(filename, n.GetPos(), level0SymbolsCode, fmt.Sprintf("Access to constant %s::%s on an unknown class %s.", className, n.Const, className)))
				return
			}
			if strings.HasPrefix(n.Const, "$") {
				propertyName := strings.TrimPrefix(n.Const, "$")
				property, ok := ctx.Resolver.ResolveProperty(className, propertyName)
				if !ok {
					issues = append(issues, issue(filename, n.GetPos(), level0SymbolsCode, fmt.Sprintf("Access to undefined static property %s::$%s.", className, propertyName)))
					return
				}
				if !property.IsStatic {
					issues = append(issues, issue(filename, n.GetPos(), level0SymbolsCode, fmt.Sprintf("Static access to instance property %s::$%s.", resolvedClass.Name, property.Name)))
				}
				return
			}
			if n.Const != "class" && !ctx.Resolver.ConstantExists(className+"::"+n.Const) {
				issues = append(issues, issue(filename, n.GetPos(), level0SymbolsCode, fmt.Sprintf("Access to undefined constant %s::%s.", className, n.Const)))
			}
		case *ast.PropertyFetchNode:
			receiver, ok := n.Object.(*ast.VariableNode)
			if !ok || receiver.Name != "this" {
				return
			}
			className := currentClassName(class, ft)
			if className == "" {
				issues = append(issues, issue(filename, n.GetPos(), level0SymbolsCode, "Undefined variable: $this"))
				return
			}
			if _, ok := ctx.Resolver.ResolveProperty(className, n.Property); !ok {
				issues = append(issues, issue(filename, n.GetPos(), level0SymbolsCode, fmt.Sprintf("Access to an undefined property %s::$%s.", className, n.Property)))
			}
		}
	})
	return issues
}

func (r *PHPStanLevel0Rule) checkUndefinedVariables(filename string, nodes []ast.Node, ctx *AnalysisContext, fileCtx fileTypeContext) []AnalysisIssue {
	var issues []AnalysisIssue
	var walkStatements func([]ast.Node, map[string]bool, *ast.ClassNode, bool) map[string]bool
	walkStatements = func(stmts []ast.Node, defined map[string]bool, class *ast.ClassNode, inFunction bool) map[string]bool {
		for _, stmt := range stmts {
			switch n := stmt.(type) {
			case *ast.FunctionNode:
				local := map[string]bool{}
				for _, param := range n.Params {
					if p, ok := param.(*ast.ParamNode); ok {
						local[p.Name] = true
					}
				}
				if class != nil && !hasModifier(n.Modifiers, "static") {
					local["this"] = true
				}
				walkStatements(n.Body, local, class, true)
			case *ast.ClassNode:
				for _, method := range n.Methods {
					if fn, ok := method.(*ast.FunctionNode); ok {
						walkStatements([]ast.Node{fn}, defined, n, false)
					}
				}
			case *ast.NamespaceNode:
				defined = walkStatements(n.Body, defined, class, inFunction)
			case *ast.StaticVarDeclNode:
				for _, entry := range n.Vars {
					if entry.Init != nil {
						checkExprVars(filename, entry.Init, defined, &issues)
					}
					defined[entry.Name] = true
				}
			case *ast.AssignmentNode:
				checkExprVars(filename, n.Right, defined, &issues)
				defineAssignmentTarget(n.Left, defined)
			case *ast.ExpressionStmt:
				if assign, ok := n.Expr.(*ast.AssignmentNode); ok {
					checkExprVars(filename, assign.Right, defined, &issues)
					defineAssignmentTarget(assign.Left, defined)
				} else {
					checkExprVars(filename, n.Expr, defined, &issues)
				}
			case *ast.ReturnNode:
				checkExprVars(filename, n.Expr, defined, &issues)
			case *ast.ThrowNode:
				checkExprVars(filename, n.Expr, defined, &issues)
			case *ast.IfNode:
				checkExprVars(filename, n.Condition, defined, &issues)
				before := cloneBoolMap(defined)
				thenDefined := walkStatements(n.Body, cloneBoolMap(defined), class, inFunction)
				branchUnion := cloneBoolMap(before)
				for k := range thenDefined {
					branchUnion[k] = true
				}
				for _, elseif := range n.ElseIfs {
					checkExprVars(filename, elseif.Condition, before, &issues)
					ed := walkStatements(elseif.Body, cloneBoolMap(before), class, inFunction)
					for k := range ed {
						branchUnion[k] = true
					}
				}
				if n.Else != nil {
					ed := walkStatements(n.Else.Body, cloneBoolMap(before), class, inFunction)
					for k := range ed {
						branchUnion[k] = true
					}
				}
				defined = branchUnion
			case *ast.ForeachNode:
				checkExprVars(filename, n.Expr, defined, &issues)
				defineAssignmentTarget(n.KeyVar, defined)
				defineAssignmentTarget(n.ValueVar, defined)
				defined = walkStatements(n.Body, defined, class, inFunction)
			case *ast.TryNode:
				defined = walkStatements(n.Body, defined, class, inFunction)
				for _, catchNode := range n.Catches {
					catchDefined := cloneBoolMap(defined)
					if catchNode.Variable != "" {
						catchDefined[strings.TrimPrefix(catchNode.Variable, "$")] = true
					}
					walkStatements(catchNode.Body, catchDefined, class, inFunction)
				}
				defined = walkStatements(n.Finally, defined, class, inFunction)
			}
		}
		return defined
	}
	defined := map[string]bool{"GLOBALS": true, "_SERVER": true, "_GET": true, "_POST": true, "_FILES": true, "_COOKIE": true, "_SESSION": true, "_REQUEST": true, "_ENV": true, "argc": true, "argv": true}
	walkStatements(nodes, defined, nil, false)
	return issues
}

func (r *PHPStanLevel0Rule) checkLanguage(filename string, nodes []ast.Node, ctx *AnalysisContext, fileCtx fileTypeContext) []AnalysisIssue {
	var issues []AnalysisIssue
	labels := map[string]struct{}{}
	var gotos []*ast.GotoNode
	walkAll(nodes, func(node ast.Node, class *ast.ClassNode, ft fileTypeContext) {
		switch n := node.(type) {
		case *ast.LabelNode:
			labels[n.Name] = struct{}{}
		case *ast.GotoNode:
			gotos = append(gotos, n)
		case *ast.ArrayNode:
			seen := map[string]ast.Position{}
			for _, element := range n.Elements {
				item, ok := element.(*ast.ArrayItemNode)
				if !ok || item.Key == nil {
					continue
				}
				key, ok := literalKey(item.Key)
				if !ok {
					continue
				}
				if first, exists := seen[key]; exists {
					_ = first
					issues = append(issues, issue(filename, item.GetPos(), level0LanguageCode, fmt.Sprintf("Array has %s duplicate key.", key)))
					continue
				}
				seen[key] = item.GetPos()
			}
		case *ast.UnaryExpr:
			switch n.Operator {
			case "include", "include_once", "require", "require_once":
				if path, ok := stringLiteralValue(n.Operand); ok {
					if _, err := os.Stat(resolveIncludePath(filename, path)); err != nil {
						issues = append(issues, issue(filename, n.GetPos(), level0LanguageCode, fmt.Sprintf("Path in %s() \"%s\" is not a file or it does not exist.", n.Operator, path)))
					}
				}
			case "++", "--":
				if !isWritableExpr(n.Operand) {
					issues = append(issues, issue(filename, n.GetPos(), level0LanguageCode, fmt.Sprintf("Cannot use %s on non-variable expression.", n.Operator)))
				}
			}
		case *ast.TypeCastNode:
			if strings.EqualFold(n.Type, "unset") || strings.EqualFold(n.Type, "void") {
				issues = append(issues, issue(filename, n.GetPos(), level0LanguageCode, fmt.Sprintf("Cannot cast to %s.", n.Type)))
			}
		case *ast.FunctionCallNode:
			name := strings.ToLower(functionCallName(n))
			if name == "preg_match" && len(n.Args) > 0 {
				if pattern, ok := stringLiteralValue(argumentValue(n.Args[0])); ok {
					if _, err := regexp.Compile(extractRegexpBody(pattern)); err != nil {
						issues = append(issues, issue(filename, n.GetPos(), level0LanguageCode, fmt.Sprintf("Regex pattern is invalid: %s", err.Error())))
					}
				}
			}
			if (name == "printf" || name == "sprintf") && len(n.Args) > 0 {
				if format, ok := stringLiteralValue(argumentValue(n.Args[0])); ok {
					required := countPrintfPlaceholders(format)
					if required > len(n.Args)-1 {
						issues = append(issues, issue(filename, n.GetPos(), level0InvocationCode, fmt.Sprintf("Call to function %s contains %d placeholders, %d values given.", name, required, len(n.Args)-1)))
					}
				}
			}
		}
	})
	for _, goTo := range gotos {
		if _, ok := labels[goTo.Label]; !ok {
			issues = append(issues, issue(filename, goTo.GetPos(), level0LanguageCode, fmt.Sprintf("Goto to undefined label %s.", goTo.Label)))
		}
	}
	return issues
}

func walkAll(nodes []ast.Node, fn func(ast.Node, *ast.ClassNode, fileTypeContext)) {
	var walk func(ast.Node, *ast.ClassNode, fileTypeContext)
	walk = func(node ast.Node, class *ast.ClassNode, ft fileTypeContext) {
		if node == nil {
			return
		}
		fn(node, class, ft)
		switch n := node.(type) {
		case *ast.NamespaceNode:
			nft := collectFileTypeContext(n.Body)
			if nft.namespace == "" {
				nft.namespace = n.Name
			}
			for _, child := range n.Body {
				walk(child, class, nft)
			}
		case *ast.ClassNode:
			cft := ft
			for _, child := range n.Properties {
				walk(child, n, cft)
			}
			for _, child := range n.Methods {
				walk(child, n, cft)
			}
		case *ast.FunctionNode:
			for _, param := range n.Params {
				walk(param, class, ft)
			}
			for _, child := range n.Body {
				walk(child, class, ft)
			}
		case *ast.InterfaceNode:
			for _, child := range n.Members {
				walk(child, class, ft)
			}
		case *ast.TraitNode:
			for _, child := range n.Body {
				walk(child, class, ft)
			}
		case *ast.EnumNode:
			for _, child := range n.Methods {
				walk(child, class, ft)
			}
		case *ast.ExpressionStmt:
			walk(n.Expr, class, ft)
		case *ast.AssignmentNode:
			walk(n.Left, class, ft)
			walk(n.Right, class, ft)
		case *ast.ReturnNode:
			walk(n.Expr, class, ft)
		case *ast.ThrowNode:
			walk(n.Expr, class, ft)
		case *ast.IfNode:
			walk(n.Condition, class, ft)
			for _, child := range n.Body {
				walk(child, class, ft)
			}
			for _, elseif := range n.ElseIfs {
				walk(elseif.Condition, class, ft)
				for _, child := range elseif.Body {
					walk(child, class, ft)
				}
			}
			if n.Else != nil {
				for _, child := range n.Else.Body {
					walk(child, class, ft)
				}
			}
		case *ast.WhileNode:
			walk(n.Condition, class, ft)
			for _, child := range n.Body {
				walk(child, class, ft)
			}
		case *ast.ForeachNode:
			walk(n.Expr, class, ft)
			walk(n.KeyVar, class, ft)
			walk(n.ValueVar, class, ft)
			for _, child := range n.Body {
				walk(child, class, ft)
			}
		case *ast.TryNode:
			for _, child := range n.Body {
				walk(child, class, ft)
			}
			for _, catchNode := range n.Catches {
				walk(catchNode, class, ft)
			}
			for _, child := range n.Finally {
				walk(child, class, ft)
			}
		case *ast.CatchNode:
			for _, child := range n.Body {
				walk(child, class, ft)
			}
		case *ast.AttributeNode:
			for _, arg := range n.Arguments {
				walk(arg, class, ft)
			}
		case *ast.StaticVarDeclNode:
			for _, entry := range n.Vars {
				walk(entry.Init, class, ft)
			}
		case *ast.FunctionCallNode:
			walk(n.Name, class, ft)
			for _, arg := range n.Args {
				walk(arg, class, ft)
			}
		case *ast.MethodCallNode:
			walk(n.Object, class, ft)
			for _, arg := range n.Args {
				walk(arg, class, ft)
			}
		case *ast.NewNode:
			walk(n.ClassExpr, class, ft)
			for _, arg := range n.Args {
				walk(arg, class, ft)
			}
		case *ast.NamedArgumentNode:
			walk(n.Value, class, ft)
		case *ast.UnpackedArgumentNode:
			walk(n.Expr, class, ft)
		case *ast.BinaryExpr:
			walk(n.Left, class, ft)
			walk(n.Right, class, ft)
		case *ast.UnaryExpr:
			walk(n.Operand, class, ft)
		case *ast.TernaryExpr:
			walk(n.Condition, class, ft)
			walk(n.IfTrue, class, ft)
			walk(n.IfFalse, class, ft)
		case *ast.ArrayNode:
			for _, child := range n.Elements {
				walk(child, class, ft)
			}
		case *ast.ArrayItemNode:
			walk(n.Key, class, ft)
			walk(n.Value, class, ft)
		case *ast.ArrayAccessNode:
			walk(n.Var, class, ft)
			walk(n.Index, class, ft)
		case *ast.PropertyFetchNode:
			walk(n.Object, class, ft)
		case *ast.ConcatNode:
			for _, child := range n.Parts {
				walk(child, class, ft)
			}
		case *ast.MatchNode:
			walk(n.Condition, class, ft)
			for _, arm := range n.Arms {
				for _, condition := range arm.Conditions {
					walk(condition, class, ft)
				}
				walk(arm.Body, class, ft)
			}
		case *ast.ArrowFunctionNode:
			for _, param := range n.Params {
				walk(param, class, ft)
			}
			walk(n.Expr, class, ft)
		}
	}
	ft := collectFileTypeContext(nodes)
	for _, node := range nodes {
		walk(node, nil, ft)
	}
}

func checkTypeReference(filename string, pos ast.Position, subject, raw string, ft fileTypeContext, ctx *AnalysisContext, issues *[]AnalysisIssue) {
	for _, name := range referencedClassTypes(raw, ft) {
		if isSpecialClassName(name) {
			continue
		}
		if _, ok := ctx.Resolver.ResolveClass(name); !ok {
			*issues = append(*issues, issue(filename, pos, level0SymbolsCode, fmt.Sprintf("%s references unknown class %s.", subject, name)))
		}
	}
}

func referencedClassTypes(raw string, ft fileTypeContext) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if strings.HasPrefix(raw, "?") {
		raw = strings.TrimSpace(strings.TrimPrefix(raw, "?"))
	}
	var refs []string
	for _, unionPart := range splitTopLevelTypes(raw, '|') {
		for _, part := range splitTopLevelTypes(unionPart, '&') {
			part = strings.TrimSpace(part)
			part = strings.Trim(part, "()")
			if part == "" {
				continue
			}
			canonical := canonicalizeDocType(strings.TrimPrefix(part, `\`))
			if atom, ok := normalizeTypeAtom(canonical); ok {
				if atom.kind == typeKindBuiltin {
					continue
				}
				part = atom.display
			}
			if strings.ContainsAny(part, "$[]{}") {
				continue
			}
			refs = append(refs, ft.resolveClassLike(part))
		}
	}
	return refs
}

func paramTypeName(param *ast.ParamNode) string {
	if param.TypeHint != "" {
		return param.TypeHint
	}
	if param.UnionType != nil {
		return param.UnionType.TokenLiteral()
	}
	return ""
}

func checkCallArguments(filename string, pos ast.Position, target, name string, args []ast.Node, method ResolvedMethod, issues *[]AnalysisIssue) {
	if method.Name == "" && len(method.Params) == 0 {
		return
	}
	actualCount, hasUnpacked := countCallArguments(args)
	if !hasUnpacked {
		required, max, variadic := parameterBounds(method.Params)
		if actualCount < required {
			*issues = append(*issues, issue(filename, pos, level0InvocationCode, fmt.Sprintf("%s invoked with %d %s, at least %d required.", target, actualCount, pluralizeParameters(actualCount), required)))
		} else if !variadic && actualCount > max {
			*issues = append(*issues, issue(filename, pos, level0InvocationCode, fmt.Sprintf("%s invoked with %d %s, at most %d allowed.", target, actualCount, pluralizeParameters(actualCount), max)))
		}
	}
	checkNamedArguments(filename, pos, name, args, method.Params, issues)
}

func checkNamedArguments(filename string, pos ast.Position, name string, args []ast.Node, params []ResolvedParam, issues *[]AnalysisIssue) {
	seenNamed := false
	seenUnpacked := false
	used := map[string]struct{}{}
	paramsByName := map[string]ResolvedParam{}
	var variadic bool
	for _, param := range params {
		paramsByName[param.Name] = param
		if param.IsVariadic {
			variadic = true
		}
	}
	for _, arg := range args {
		switch a := arg.(type) {
		case *ast.NamedArgumentNode:
			seenNamed = true
			if _, ok := paramsByName[a.Name]; !ok && !variadic {
				*issues = append(*issues, issue(filename, a.GetPos(), level0InvocationCode, fmt.Sprintf("Unknown parameter $%s in call to %s.", a.Name, name)))
			}
			if _, exists := used[a.Name]; exists {
				*issues = append(*issues, issue(filename, a.GetPos(), level0InvocationCode, fmt.Sprintf("Argument for parameter $%s has already been passed.", a.Name)))
			}
			used[a.Name] = struct{}{}
		case *ast.UnpackedArgumentNode:
			if seenNamed {
				*issues = append(*issues, issue(filename, a.GetPos(), level0InvocationCode, "Named argument cannot be followed by an unpacked (...) argument."))
			}
			seenUnpacked = true
		default:
			if seenNamed {
				*issues = append(*issues, issue(filename, arg.GetPos(), level0InvocationCode, "Named argument cannot be followed by a positional argument."))
			}
			if seenUnpacked {
				*issues = append(*issues, issue(filename, arg.GetPos(), level0InvocationCode, "Unpacked argument (...) cannot be followed by a non-unpacked argument."))
			}
		}
	}
}

func parameterBounds(params []ResolvedParam) (int, int, bool) {
	required := 0
	max := 0
	variadic := false
	for _, param := range params {
		if param.IsVariadic {
			variadic = true
			continue
		}
		max++
		if !param.HasDefault {
			required++
		}
	}
	return required, max, variadic
}

func constructorFor(className string, ctx *AnalysisContext) ResolvedMethod {
	method, ok := ctx.Resolver.ResolveMethod(className, "__construct")
	if !ok {
		return ResolvedMethod{Name: "__construct"}
	}
	return method
}

func checkMethodVisibility(filename string, pos ast.Position, method ResolvedMethod, className string, currentClass *ast.ClassNode, static bool, issues *[]AnalysisIssue) {
	if method.Visibility == "private" && (currentClass == nil || !strings.EqualFold(currentClass.Name, className)) {
		*issues = append(*issues, issue(filename, pos, level0InvocationCode, fmt.Sprintf("Call to private method %s() of class %s.", method.Name, className)))
	}
	if method.Visibility == "protected" && currentClass == nil {
		*issues = append(*issues, issue(filename, pos, level0InvocationCode, fmt.Sprintf("Call to protected method %s() of class %s.", method.Name, className)))
	}
	if static && !method.IsStatic {
		*issues = append(*issues, issue(filename, pos, level0InvocationCode, fmt.Sprintf("Static call to instance method %s::%s().", className, method.Name)))
	}
}

func checkExprVars(filename string, node ast.Node, defined map[string]bool, issues *[]AnalysisIssue) {
	if node == nil {
		return
	}
	switch n := node.(type) {
	case *ast.VariableNode:
		if !defined[n.Name] {
			*issues = append(*issues, issue(filename, n.GetPos(), level0VariablesCode, fmt.Sprintf("Undefined variable: $%s", n.Name)))
		}
	case *ast.AssignmentNode:
		checkExprVars(filename, n.Right, defined, issues)
		defineAssignmentTarget(n.Left, defined)
	case *ast.FunctionCallNode:
		name := strings.ToLower(functionCallName(n))
		if name == "isset" || name == "empty" {
			return
		}
		for _, arg := range n.Args {
			checkExprVars(filename, argumentValue(arg), defined, issues)
		}
	case *ast.MethodCallNode:
		checkExprVars(filename, n.Object, defined, issues)
		for _, arg := range n.Args {
			checkExprVars(filename, argumentValue(arg), defined, issues)
		}
	case *ast.NewNode:
		for _, arg := range n.Args {
			checkExprVars(filename, argumentValue(arg), defined, issues)
		}
	case *ast.PropertyFetchNode:
		checkExprVars(filename, n.Object, defined, issues)
	case *ast.ArrayAccessNode:
		checkExprVars(filename, n.Var, defined, issues)
		checkExprVars(filename, n.Index, defined, issues)
	case *ast.BinaryExpr:
		checkExprVars(filename, n.Left, defined, issues)
		checkExprVars(filename, n.Right, defined, issues)
	case *ast.UnaryExpr:
		checkExprVars(filename, n.Operand, defined, issues)
	case *ast.TernaryExpr:
		checkExprVars(filename, n.Condition, defined, issues)
		checkExprVars(filename, n.IfTrue, defined, issues)
		checkExprVars(filename, n.IfFalse, defined, issues)
	case *ast.ArrayNode:
		for _, element := range n.Elements {
			checkExprVars(filename, element, defined, issues)
		}
	case *ast.ArrayItemNode:
		checkExprVars(filename, n.Key, defined, issues)
		checkExprVars(filename, n.Value, defined, issues)
	case *ast.NamedArgumentNode:
		checkExprVars(filename, n.Value, defined, issues)
	case *ast.UnpackedArgumentNode:
		checkExprVars(filename, n.Expr, defined, issues)
	case *ast.ConcatNode:
		for _, part := range n.Parts {
			checkExprVars(filename, part, defined, issues)
		}
	}
}

func defineAssignmentTarget(node ast.Node, defined map[string]bool) {
	switch n := node.(type) {
	case *ast.VariableNode:
		defined[n.Name] = true
	case *ast.ArrayAccessNode:
		defineAssignmentTarget(n.Var, defined)
	case *ast.PropertyFetchNode:
		defineAssignmentTarget(n.Object, defined)
	}
}

func functionCallName(call *ast.FunctionCallNode) string {
	switch name := call.Name.(type) {
	case *ast.IdentifierNode:
		return strings.TrimPrefix(name.Value, `\`)
	case *ast.Identifier:
		return strings.TrimPrefix(name.Name, `\`)
	}
	return ""
}

func resolveNewClassName(node *ast.NewNode, ft fileTypeContext) string {
	if node.ClassName != "" {
		return ft.resolveClassLike(node.ClassName)
	}
	if ident, ok := node.ClassExpr.(*ast.IdentifierNode); ok {
		return ft.resolveClassLike(ident.Value)
	}
	return ""
}

func resolveClassLikeForCall(name string, current *ast.ClassNode, ft fileTypeContext) string {
	switch strings.ToLower(strings.TrimPrefix(name, `\`)) {
	case "self", "static":
		if current != nil {
			return ft.resolveClassLike(current.Name)
		}
	case "parent":
		if current != nil {
			if class, ok := ft.resolveClass(ft.resolveClassLike(current.Name)); ok && len(class.Extends) > 0 {
				return class.Extends[0]
			}
		}
	}
	return ft.resolveClassLike(name)
}

func resolveFunctionNameForCall(name string, ft fileTypeContext, ctx *AnalysisContext) string {
	name = strings.TrimPrefix(strings.TrimSpace(name), `\`)
	if name == "" || strings.Contains(name, "::") {
		return name
	}
	if ctx != nil && ctx.Resolver != nil && ctx.Resolver.FunctionExists(name) {
		return name
	}
	resolved := ft.resolveClassLike(name)
	if ctx != nil && ctx.Resolver != nil && ctx.Resolver.FunctionExists(resolved) {
		return resolved
	}
	return name
}

func currentClassName(class *ast.ClassNode, ft fileTypeContext) string {
	if class == nil {
		return ""
	}
	return ft.resolveClassLike(class.Name)
}

func isSpecialClassName(name string) bool {
	switch strings.ToLower(strings.TrimPrefix(name, `\`)) {
	case "", "self", "static", "parent":
		return true
	default:
		return false
	}
}

func isWritableExpr(node ast.Node) bool {
	switch node.(type) {
	case *ast.VariableNode, *ast.ArrayAccessNode, *ast.PropertyFetchNode:
		return true
	default:
		return false
	}
}

func titleKind(kind string) string {
	if kind == "" {
		return ""
	}
	return strings.ToUpper(kind[:1]) + kind[1:]
}

func literalKey(node ast.Node) (string, bool) {
	switch n := node.(type) {
	case *ast.StringLiteral:
		return strconv.Quote(n.Value), true
	case *ast.StringNode:
		return strconv.Quote(n.Value), true
	case *ast.IntegerLiteral:
		return strconv.FormatInt(n.Value, 10), true
	case *ast.IntegerNode:
		return strconv.FormatInt(n.Value, 10), true
	}
	return "", false
}

func stringLiteralValue(node ast.Node) (string, bool) {
	switch n := node.(type) {
	case *ast.StringLiteral:
		return n.Value, true
	case *ast.StringNode:
		return n.Value, true
	}
	return "", false
}

func resolveIncludePath(filename, include string) string {
	if filepath.IsAbs(include) {
		return include
	}
	return filepath.Join(filepath.Dir(filename), include)
}

func extractRegexpBody(pattern string) string {
	if len(pattern) < 2 {
		return pattern
	}
	delimiter := pattern[0]
	end := strings.LastIndexByte(pattern[1:], delimiter)
	if end < 0 {
		return pattern
	}
	return pattern[1 : end+1]
}

func countPrintfPlaceholders(format string) int {
	count := 0
	escaped := false
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			escaped = false
			continue
		}
		if escaped {
			escaped = false
			continue
		}
		if i+1 < len(format) && format[i+1] == '%' {
			escaped = true
			continue
		}
		count++
	}
	return count
}

func cloneBoolMap(in map[string]bool) map[string]bool {
	out := make(map[string]bool, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func issue(filename string, pos ast.Position, code, message string) AnalysisIssue {
	return AnalysisIssue{Filename: filename, Line: pos.Line, Column: pos.Column, Code: code, Message: message}
}

func init() {
	RegisterAnalysisRuleWithLevel(level0SymbolsCode, 0, "phpstan.level0", func(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
		return (&PHPStanLevel0Rule{}).CheckIssues(filename, nodes, ctx)
	})
}
