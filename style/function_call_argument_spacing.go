package style

import (
	"go-phpcs/ast"
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
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = fixFunctionCallSpacingInLine(line)
	}
	return strings.Join(lines, "\n")
}

func fixFunctionCallSpacingInLine(line string) string {
	// Preserve all leading/trailing whitespace
	leading := ""
	trailing := ""
	core := line
	for len(core) > 0 && (core[0] == ' ' || core[0] == '\t') {
		leading += string(core[0])
		core = core[1:]
	}
	for len(core) > 0 && (core[len(core)-1] == ' ' || core[len(core)-1] == '\t') {
		trailing = string(core[len(core)-1]) + trailing
		core = core[:len(core)-1]
	}
	// First, recursively fix all nested calls
	core = fixAllFunctionCallArgumentSpacing(core)
	// Then, fix all top-level calls in the line
	core = funcCallRegex.ReplaceAllStringFunc(core, func(match string) string {
		parts := funcCallRegex.FindStringSubmatch(match)
		if len(parts) < 3 {
			return match
		}
		funcName := parts[1]
		args := parts[2]
		args = fixArgumentSpacing(args)
		return funcName + "(" + args + ")"
	})
	return leading + core + trailing
}

// Recursively fix all function call argument spacing in a string
func fixAllFunctionCallArgumentSpacing(s string) string {
	// Recursively fix all nested calls, then fix the outer call's arguments
	return funcCallRegex.ReplaceAllStringFunc(s, func(match string) string {
		parts := funcCallRegex.FindStringSubmatch(match)
		if len(parts) < 3 {
			return match
		}
		funcName := parts[1]
		args := parts[2]
		// Recursively fix nested calls in the arguments
		args = fixAllFunctionCallArgumentSpacing(args)
		// Now fix the argument spacing at this (outer) level
		args = fixArgumentSpacing(args)
		return funcName + "(" + args + ")"
	})
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
