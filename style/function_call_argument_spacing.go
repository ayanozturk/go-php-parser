package style

import (
	"fmt"
	"go-phpcs/ast"
	"os"
	"strings"
)

type FunctionCallArgumentSpacingChecker struct{}

func isCommentOnlyLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "//") ||
		strings.HasPrefix(trimmed, "#") ||
		strings.HasPrefix(trimmed, "/*") ||
		strings.HasPrefix(trimmed, "*") ||
		strings.HasPrefix(trimmed, "*/")
}

// Detects bad comma spacing without regex: any of
// 1) one or more spaces before comma
// 2) two or more spaces after comma
// 3) no space after comma (next is non-space)
func hasBadCommaSpacing(args string) bool {
	parenDepth := 0
	bracketDepth := 0
	braceDepth := 0
	inSingleQuote := false
	inDoubleQuote := false
	escaped := false

	for i := 0; i < len(args); i++ {
		if escaped {
			escaped = false
			continue
		}

		if inSingleQuote {
			if args[i] == '\\' {
				escaped = true
				continue
			}
			if args[i] == '\'' {
				inSingleQuote = false
			}
			continue
		}

		if inDoubleQuote {
			if args[i] == '\\' {
				escaped = true
				continue
			}
			if args[i] == '"' {
				inDoubleQuote = false
			}
			continue
		}

		switch args[i] {
		case '\'':
			inSingleQuote = true
			continue
		case '"':
			inDoubleQuote = true
			continue
		case '(':
			parenDepth++
			continue
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
			continue
		case '[':
			bracketDepth++
			continue
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
			continue
		case '{':
			braceDepth++
			continue
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
			continue
		}

		if parenDepth > 0 || bracketDepth > 0 || braceDepth > 0 {
			continue
		}

		if args[i] == ',' {
			if i > 0 && (args[i-1] == ' ' || args[i-1] == '\t') {
				return true
			}

			j := i + 1
			spaceCount := 0
			for j < len(args) && (args[j] == ' ' || args[j] == '\t') {
				spaceCount++
				j++
			}
			if spaceCount >= 2 {
				return true
			}
			if j < len(args) && spaceCount == 0 {
				return true
			}
		}
	}

	return false
}

func (c *FunctionCallArgumentSpacingChecker) CheckIssues(lines []string, filename string) []StyleIssue {
	var issues []StyleIssue
	for i, line := range lines {
		if isCommentOnlyLine(line) {
			continue
		}
		if !strings.Contains(line, "(") || !strings.Contains(line, ",") {
			continue
		}

		for idx := 0; idx < len(line); {
			start := idx
			for idx < len(line) && (isIdentChar(line[idx]) || (idx > start && isDigit(line[idx]))) {
				idx++
			}
			if idx < len(line) && line[idx] == '(' && start != idx {
				parenDepth := 1
				j := idx + 1
				for ; j < len(line) && parenDepth > 0; j++ {
					if line[j] == '(' {
						parenDepth++
					} else if line[j] == ')' {
						parenDepth--
					}
				}
				if parenDepth == 0 {
					argsStart := idx + 1
					argsEnd := j - 1
					if argsEnd >= argsStart && argsEnd <= len(line) {
						args := line[argsStart:argsEnd]
						if hasBadCommaSpacing(args) {
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
					idx = j
					continue
				}
				break
			}
			idx++
		}
	}

	return issues
}

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
		if isCommentOnlyLine(line) {
			continue
		}
		fixed := fixFunctionCallSpacingInLine(line)
		lines[i] = fixed
	}

	return strings.Join(lines, "\n")
}

func fixFunctionCallSpacingInLine(line string) string {
	out := getBuilder()
	for i := 0; i < len(line); {
		start := i
		for i < len(line) && (isIdentChar(line[i]) || (i > start && isDigit(line[i]))) {
			i++
		}
		if start != i && (i >= len(line) || line[i] != '(') {
			out.WriteString(line[start:i])
			continue
		}
		if i < len(line) && line[i] == '(' && start != i {
			funcName := line[start:i]
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
				args := ""
				if j-1 >= i+1 && j-1 <= len(line) {
					args = line[i+1 : j-1]
				}
				fixedArgs := fixArgumentSpacing(args)
				out.WriteString(funcName + "(" + fixedArgs + ")")
				i = j
				continue
			}
			out.WriteString(line[start:])
			break
		}
		out.WriteByte(line[i])
		i++
	}
	result := out.String()
	putBuilder(out)
	return result
}

func isIdentChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

// Splits arguments at the top level, respecting parentheses, quotes, and unpacked arguments.
func splitFunctionArguments(args string) []string {
	var (
		result        []string
		parenDepth    int
		bracketDepth  int
		braceDepth    int
		start         int
		inUnpack      bool
		inSingleQuote bool
		inDoubleQuote bool
		escaped       bool
	)

	for i := 0; i <= len(args); i++ {
		if i < len(args) && args[i] == '.' && i+2 < len(args) && args[i+1] == '.' && args[i+2] == '.' {
			inUnpack = true
			i += 2
			continue
		}

		if escaped {
			escaped = false
			continue
		}

		if inSingleQuote {
			if i < len(args) && args[i] == '\\' {
				escaped = true
				continue
			}
			if i < len(args) && args[i] == '\'' {
				inSingleQuote = false
			}
			continue
		}

		if inDoubleQuote {
			if i < len(args) && args[i] == '\\' {
				escaped = true
				continue
			}
			if i < len(args) && args[i] == '"' {
				inDoubleQuote = false
			}
			continue
		}

		if i < len(args) {
			switch args[i] {
			case '\'':
				inSingleQuote = true
				continue
			case '"':
				inDoubleQuote = true
				continue
			case '(':
				parenDepth++
			case ')':
				parenDepth--
			case '[':
				bracketDepth++
			case ']':
				bracketDepth--
			case '{':
				braceDepth++
			case '}':
				braceDepth--
			}
		}

		if i == len(args) || (parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 && i < len(args) && args[i] == ',') {
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
		lines := SplitLinesCached(content)
		checker := &FunctionCallArgumentSpacingChecker{}
		return checker.CheckIssues(lines, filename)
	})
	RegisterFixer(FunctionCallArgumentSpacingFixer{})
}
