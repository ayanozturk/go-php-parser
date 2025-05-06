package style

import (
	"fmt"
	"go-phpcs/ast"
	"os"
	"regexp"
	"strings"
)

type FunctionCallArgumentSpacingChecker struct{}

var (
	funcCallRegex   = regexp.MustCompile(`([a-zA-Z_][a-zA-Z0-9_]*)\s*\(([^)]*)\)`)
	badCommaSpacing = regexp.MustCompile(`,\s{2,}|\s+,|,\S`)
)

func (c *FunctionCallArgumentSpacingChecker) CheckIssues(lines []string, filename string) []StyleIssue {
	var issues []StyleIssue
	for i, line := range lines {
		matches := funcCallRegex.FindAllStringSubmatchIndex(line, -1)
		for _, m := range matches {
			argsStart, argsEnd := m[4], m[5]
			args := line[argsStart:argsEnd]
			if badCommaSpacing.MatchString(args) {
				issues = append(issues, StyleIssue{
					Filename: filename,
					Line:     i + 1,
					Type:     Error,
					Fixable:  true,
					Message:  "Incorrect spacing between function call arguments",
					Code:     "Generic.Functions.FunctionCallArgumentSpacing",
				})
			}
		}
	}
	return issues
}

// Fixer for Generic.Functions.FunctionCallArgumentSpacing

type FunctionCallArgumentSpacingFixer struct{}

func (f FunctionCallArgumentSpacingFixer) Code() string {
	return "Generic.Functions.FunctionCallArgumentSpacing"
}

func (f FunctionCallArgumentSpacingFixer) Fix(content string) string {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "[PANIC] in FunctionCallArgumentSpacingFixer.Fix: %v\n", r)
			fmt.Fprintf(os.Stderr, "[PANIC] content: %q\n", content)
		}
	}()
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		fixed := fixFunctionCallSpacingInLine(line)
		if fixed != line {
			// Debug print to stderr with file and line info
			fmt.Fprintf(os.Stderr, "[DEBUG] FunctionCallArgumentSpacingFixer: line %d changed in Fix\nOriginal: %q\nFixed:   %q\n", i+1, line, fixed)
		}
		lines[i] = fixed
	}
	return strings.Join(lines, "\n")
}

// Parenthesis-aware function call fixer for a line
func fixFunctionCallSpacingInLine(line string) string {
	var out strings.Builder
	for i := 0; i < len(line); {
		// Find function name
		start := i
		for i < len(line) && (isIdentChar(line[i]) || (i > start && isDigit(line[i]))) {
			i++
		}
		if i < len(line) && line[i] == '(' && start != i {
			funcName := line[start:i]
			// Find matching closing parenthesis
			parenDepth := 1
			j := i + 1
			for ; j < len(line) && parenDepth > 0; j++ {
				if line[j] == '(' {
					parenDepth++
				} else if line[j] == ')' {
					parenDepth--
				}
			}
			if parenDepth == 0 {
				// Only extract args if j-1 >= i+1 and j-1 <= len(line)
				var args string
				if j-1 >= i+1 && j-1 <= len(line) {
					args = line[i+1 : j-1]
				} else {
					args = ""
				}
				fixedArgs := fixArgumentSpacing(args)
				out.WriteString(funcName + "(" + fixedArgs + ")")
				i = j
				continue
			} else {
				// No matching closing parenthesis, copy the rest and break
				out.WriteString(line[start:])
				break
			}
		}
		// Not a function call, just copy
		out.WriteByte(line[i])
		i++
	}
	return out.String()
}

func isIdentChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

// Splits arguments at the top level, respecting parentheses and unpacked arguments
func splitFunctionArguments(args string) []string {
	var (
		result     []string
		parenDepth int
		start      int
		inUnpack   bool
	)
	for i := 0; i <= len(args); i++ {
		if i < len(args) && args[i] == '.' && i+2 < len(args) && args[i+1] == '.' && args[i+2] == '.' {
			inUnpack = true
			i += 2
			continue
		}
		if i < len(args) {
			if args[i] == '(' {
				parenDepth++
			} else if args[i] == ')' {
				parenDepth--
			}
		}
		if i == len(args) || (parenDepth == 0 && i < len(args) && args[i] == ',') {
			arg := args[start:i]
			if inUnpack {
				arg = "..." + arg
				inUnpack = false
			}
			result = append(result, arg)
			start = i + 1
		}
	}
	return result
}

func fixArgumentSpacing(args string) string {
	if args == "" {
		return ""
	}
	// Preserve leading/trailing spaces inside the argument list
	leading := ""
	trailing := ""
	core := args
	for len(core) > 0 && (core[0] == ' ' || core[0] == '\t') {
		leading += string(core[0])
		core = core[1:]
	}
	for len(core) > 0 && (core[len(core)-1] == ' ' || core[len(core)-1] == '\t') {
		trailing = string(core[len(core)-1]) + trailing
		core = core[:len(core)-1]
	}
	parts := splitFunctionArguments(core)
	for i, arg := range parts {
		arg = strings.TrimSpace(arg)
		if strings.Contains(arg, "(") {
			arg = fixFunctionCallSpacingInLine(arg)
		}
		parts[i] = arg
	}
	return leading + strings.Join(parts, ", ") + trailing
}

func init() {
	RegisterRule("Generic.Functions.FunctionCallArgumentSpacing", func(filename string, content []byte, _ []ast.Node) []StyleIssue {
		lines := strings.Split(string(content), "\n")
		checker := &FunctionCallArgumentSpacingChecker{}
		return checker.CheckIssues(lines, filename)
	})
	RegisterFixer(FunctionCallArgumentSpacingFixer{})
}
