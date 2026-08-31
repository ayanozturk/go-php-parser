package analyse

import (
	"fmt"
	"github.com/ayanozturk/go-php-parser/ast"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

func (r *PHPStanLevel0Rule) checkLanguage(filename string, nodes []ast.Node, ctx *AnalysisContext, fileCtx FileTypeContext) []AnalysisIssue {
	var issues []AnalysisIssue
	labels := map[string]struct{}{}
	var gotos []*ast.GotoNode
	walkAllWithFileContext(nodes, fileCtx, ctx, func(node ast.Node, class *ast.ClassNode, _ *ast.FunctionNode, ft FileTypeContext) {
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
					issues = append(issues, issueSpan(filename, item, level0LanguageCode, fmt.Sprintf("Array has %s duplicate key.", key)))
					continue
				}
				seen[key] = item.GetPos()
			}
		case *ast.UnaryExpr:
			switch n.Operator {
			case "include", "include_once", "require", "require_once":
				if path, ok := stringLiteralValue(n.Operand); ok {
					if _, err := os.Stat(resolveIncludePath(filename, path)); err != nil {
						issues = append(issues, issueSpan(filename, n, level0LanguageCode, fmt.Sprintf("Path in %s() \"%s\" is not a file or it does not exist.", n.Operator, path)))
					}
				}
			case "++", "--":
				if !isWritableExpr(n.Operand) {
					issues = append(issues, issueSpan(filename, n, level0LanguageCode, fmt.Sprintf("Cannot use %s on non-variable expression.", n.Operator)))
				}
			}
		case *ast.TypeCastNode:
			if strings.EqualFold(n.Type, "unset") || strings.EqualFold(n.Type, "void") {
				issues = append(issues, issueSpan(filename, n, level0LanguageCode, fmt.Sprintf("Cannot cast to %s.", n.Type)))
			}
		case *ast.FunctionCallNode:
			name := strings.ToLower(functionCallName(n))
			if name == "preg_match" && len(n.Args) > 0 {
				if pattern, ok := stringLiteralValue(argumentValue(n.Args[0])); ok {
					if _, err := regexp.Compile(extractRegexpBody(pattern)); err != nil {
						issues = append(issues, issueSpan(filename, n, level0LanguageCode, fmt.Sprintf("Regex pattern is invalid: %s", err.Error())))
					}
				}
			}
			if (name == "printf" || name == "sprintf") && len(n.Args) > 0 {
				if format, ok := stringLiteralValue(argumentValue(n.Args[0])); ok {
					required := countPrintfPlaceholders(format)
					if required > len(n.Args)-1 {
						issues = append(issues, issueSpan(filename, n, level0InvocationCode, fmt.Sprintf("Call to function %s contains %d placeholders, %d values given.", name, required, len(n.Args)-1)))
					}
				}
			}
		}
	})
	for _, goTo := range gotos {
		if _, ok := labels[goTo.Label]; !ok {
			issues = append(issues, issueSpan(filename, goTo, level0LanguageCode, fmt.Sprintf("Goto to undefined label %s.", goTo.Label)))
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

// countPrintfPlaceholders returns the number of arguments a printf/sprintf
// format string requires. Plain specifiers (%s, %d, ...) are counted
// one-for-one. Positional specifiers (%1$s, %2$d, ...) reference an
// argument by index and may repeat; when any are present the required
// count is the highest index referenced, not the number of occurrences.
func countPrintfPlaceholders(format string) int {
	count := 0
	maxPositional := 0
	usesPositional := false
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			continue
		}
		if i+1 < len(format) && format[i+1] == '%' {
			i++ // literal "%%"
			continue
		}

		j := i + 1
		digitsStart := j
		for j < len(format) && format[j] >= '0' && format[j] <= '9' {
			j++
		}
		if j > digitsStart && j < len(format) && format[j] == '$' {
			if n, err := strconv.Atoi(format[digitsStart:j]); err == nil && n > maxPositional {
				maxPositional = n
			}
			usesPositional = true
			i = j // resume after "N$"
		}

		count++
	}
	if usesPositional {
		return maxPositional
	}
	return count
}
