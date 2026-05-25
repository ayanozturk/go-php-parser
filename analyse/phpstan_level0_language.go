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
		case *ast.ThrowNode:
			className := resolveThrownClassName(n.Expr, ft)
			if className == "" || isSpecialClassName(className) {
				return
			}
			resolved, ok := ctx.Resolver.ResolveClass(className)
			if !ok {
				return
			}
			if resolved.Kind == "trait" || resolved.Kind == "enum" {
				issues = append(issues, issue(filename, n.GetPos(), level0ClassModelCode, fmt.Sprintf("Cannot throw %s %s.", resolved.Kind, resolved.Name)))
				return
			}
			if !isThrowableClass(className, ctx.Resolver) {
				issues = append(issues, issue(filename, n.GetPos(), level0ClassModelCode, fmt.Sprintf("Invalid type %s to throw.", resolved.Name)))
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
