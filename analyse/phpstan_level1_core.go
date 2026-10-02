package analyse

import (
	"fmt"
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
)

const level1CoreCode = "Level1.Core"

// checkLevel1Core collects the remaining level-1 PHPStan checks in one
// context-aware AST pass so this compatibility slice does not add separate
// whole-file walks for constants, unused captures, and nullability checks.
func checkLevel1Core(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
	if ctx == nil {
		ctx = &AnalysisContext{}
	}
	ft := analysisFileTypeContext(ctx, nodes)
	functionNames := make(map[*ast.IdentifierNode]struct{})
	classReferences := make(map[*ast.IdentifierNode]struct{})
	usedByFunction := make(map[*ast.FunctionNode]map[string]struct{})
	functions := make(map[*ast.FunctionNode]struct{})
	functionOrder := make([]*ast.FunctionNode, 0)
	classByFunction := make(map[*ast.FunctionNode]*ast.ClassNode)
	markUsed := func(fn *ast.FunctionNode, name string) {
		if fn == nil || name == "" {
			return
		}
		if usedByFunction[fn] == nil {
			usedByFunction[fn] = make(map[string]struct{})
		}
		usedByFunction[fn][name] = struct{}{}
	}
	var issues []AnalysisIssue
	walkAllWithFileContext(nodes, ft, ctx, func(node ast.Node, class *ast.ClassNode, fn *ast.FunctionNode, nodeFT FileTypeContext) {
		switch n := node.(type) {
		case *ast.FunctionNode:
			if _, seen := functions[n]; !seen {
				functions[n] = struct{}{}
				functionOrder = append(functionOrder, n)
				classByFunction[n] = class
			}
			// A nested closure capture counts as a use in its containing body.
			if fn != nil && n != fn {
				for _, capture := range n.Uses {
					markUsed(fn, capture.Name)
				}
			}
		case *ast.VariableNode:
			// PHPStan's unused-parameter check treats any variable occurrence
			// in the body, including a write target, as a use.
			markUsed(fn, n.Name)
		case *ast.FunctionCallNode:
			if name, ok := n.Name.(*ast.IdentifierNode); ok {
				functionNames[name] = struct{}{}
			}
			callName := strings.ToLower(functionCallName(n))
			if (callName == "isset" || callName == "empty") && fn != nil {
				scope := analysisFunctionScope(ctx, class, fn, nodeFT)
				for _, arg := range n.Args {
					appendLevel1NullableCheck(filename, arg, callName, scope, ctx, &issues)
				}
			}
		case *ast.BinaryExpr:
			if strings.EqualFold(n.Operator, "instanceof") {
				if className, ok := n.Right.(*ast.IdentifierNode); ok {
					classReferences[className] = struct{}{}
				}
			}
			if n.Operator == "??" && fn != nil {
				scope := analysisFunctionScope(ctx, class, fn, nodeFT)
				appendLevel1NullableCheck(filename, n.Left, "coalesce", scope, ctx, &issues)
			}
		case *ast.IdentifierNode:
			if _, isFunctionName := functionNames[n]; isFunctionName || ctx.Resolver == nil {
				return
			}
			if _, isClassReference := classReferences[n]; isClassReference {
				return
			}
			if strings.EqualFold(n.Value, "true") || strings.EqualFold(n.Value, "false") || strings.EqualFold(n.Value, "null") {
				return
			}
			if !globalConstantExists(n.Value, nodeFT, ctx.Resolver) {
				issues = append(issues, AnalysisIssue{Filename: filename, Line: n.Pos.Line, Column: n.Pos.Column, Code: level1CoreCode, Message: fmt.Sprintf("Constant %s not found.", n.Value)})
			}
		}
	})

	for _, fn := range functionOrder {
		used := usedByFunction[fn]
		class := classByFunction[fn]
		if class != nil && strings.EqualFold(fn.Name, "__construct") {
			if classIsAttributeClass(class, ctx.Content) {
				continue
			}
			className := ft.resolveClassLike(class.Name)
			if className == "" {
				className = class.Name
			}
			for _, rawParam := range fn.Params {
				param, ok := rawParam.(*ast.ParamNode)
				if !ok || param.IsPromoted {
					continue
				}
				if _, isUsed := used[param.Name]; isUsed {
					continue
				}
				issues = append(issues, AnalysisIssue{Filename: filename, Line: param.Pos.Line, Column: param.Pos.Column, Code: level1CoreCode, Message: fmt.Sprintf("Constructor of class %s has an unused parameter $%s.", className, param.Name)})
			}
		}
		for _, capture := range fn.Uses {
			if _, isUsed := used[capture.Name]; isUsed {
				continue
			}
			issues = append(issues, AnalysisIssue{Filename: filename, Line: capture.Pos.Line, Column: capture.Pos.Column, Code: level1CoreCode, Message: fmt.Sprintf("Anonymous function has an unused use $%s.", capture.Name)})
		}
	}
	return issues
}

func classIsAttributeClass(class *ast.ClassNode, source []byte) bool {
	if class == nil || class.Pos.Line <= 0 || len(source) == 0 {
		return false
	}
	lines := strings.Split(string(source), "\n")
	line := class.Pos.Line - 1
	if line >= len(lines) {
		line = len(lines) - 1
	}
	// The attribute can share a line with the class keyword or occupy one or
	// more directly preceding lines. Stop at the first non-attribute line so
	// an earlier class attribute cannot affect this declaration.
	var header strings.Builder
	for i := line; i >= 0; i-- {
		current := strings.TrimSpace(lines[i])
		if i == line {
			column := class.Pos.Column - 1
			if column < 0 {
				column = 0
			}
			if column > len(lines[i]) {
				column = len(lines[i])
			}
			current = lines[i][:column]
		}
		current = strings.TrimSpace(current)
		if current == "" || strings.HasPrefix(current, "#") {
			header.WriteString(current)
			continue
		}
		break
	}
	compact := strings.ToLower(strings.ReplaceAll(header.String(), "\\", ""))
	return strings.Contains(compact, "#[attribute") || strings.Contains(compact, "#[\\attribute")
}

func globalConstantExists(name string, ft FileTypeContext, resolver SymbolResolver) bool {
	if resolver == nil {
		return false
	}
	name = strings.TrimSpace(name)
	if strings.HasPrefix(name, `\`) {
		return resolver.ConstantExists(strings.TrimPrefix(name, `\`))
	}
	if alias, ok := ft.ConstAliases[asciiLowerIdent(name)]; ok {
		return resolver.ConstantExists(alias)
	}
	if ft.Namespace != "" && resolver.ConstantExists(ft.Namespace+`\`+name) {
		return true
	}
	return resolver.ConstantExists(name)
}

func appendLevel1NullableCheck(filename string, expr ast.Node, kind string, scope *functionScope, ctx *AnalysisContext, issues *[]AnalysisIssue) {
	variable, ok := expr.(*ast.VariableNode)
	if !ok || scope == nil {
		return
	}
	typ := inferTypeWithFacts(filename, variable, scope, ctx)
	if typ.IsEmpty() || typ.hasBuiltin("mixed") {
		return
	}
	message := ""
	switch kind {
	case "isset":
		if !typ.hasBuiltin("null") {
			message = fmt.Sprintf("Variable $%s in isset() always exists and is not nullable.", variable.Name)
		}
	case "coalesce":
		if !typ.hasBuiltin("null") {
			message = fmt.Sprintf("Variable $%s on left side of ?? always exists and is not nullable.", variable.Name)
		}
	case "empty":
		if typ.hasBuiltin("null") {
			return
		}
		if definitelyTruthyForEmptyCheck(typ) {
			message = fmt.Sprintf("Variable $%s in empty() always exists and is not falsy.", variable.Name)
		} else if definitelyFalseyForEmptyCheck(typ) {
			message = fmt.Sprintf("Variable $%s in empty() always exists and is always falsy.", variable.Name)
		}
	}
	if message == "" {
		return
	}
	*issues = append(*issues, AnalysisIssue{Filename: filename, Line: expr.GetPos().Line, Column: expr.GetPos().Column, Code: level1CoreCode, Message: message})
}

func definitelyTruthyForEmptyCheck(typ Type) bool {
	if typ.IsEmpty() {
		return false
	}
	for key, atom := range typ.atoms {
		if atom.kind == typeKindClass {
			continue
		}
		switch key {
		case "true", "positive-int", "negative-int", "non-empty-string", "non-empty-array":
		default:
			return false
		}
	}
	return true
}

func definitelyFalseyForEmptyCheck(typ Type) bool {
	if typ.IsEmpty() {
		return false
	}
	for key := range typ.atoms {
		switch key {
		case "false", "null", "empty-string", "empty-array":
		default:
			return false
		}
	}
	return true
}

func init() {
	RegisterAnalysisRuleWithLevel(level1CoreCode, 1, "level1", checkLevel1Core)
}
