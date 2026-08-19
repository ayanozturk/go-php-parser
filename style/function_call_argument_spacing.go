package style

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"strings"
)

type FunctionCallArgumentSpacingChecker struct{}

var nonCallKeywords = map[string]struct{}{
	"if":           {},
	"elseif":       {},
	"else":         {},
	"for":          {},
	"foreach":      {},
	"while":        {},
	"do":           {},
	"switch":       {},
	"case":         {},
	"declare":      {},
	"catch":        {},
	"finally":      {},
	"match":        {},
	"function":     {},
	"fn":           {},
	"array":        {},
	"list":         {},
	"isset":        {},
	"unset":        {},
	"empty":        {},
	"eval":         {},
	"exit":         {},
	"die":          {},
	"include":      {},
	"include_once": {},
	"require":      {},
	"require_once": {},
	"print":        {},
	"echo":         {},
	"return":       {},
	"throw":        {},
	"yield":        {},
	"new":          {},
	"clone":        {},
	"use":          {},
	"and":          {},
	"or":           {},
	"xor":          {},
	"as":           {},
	"instanceof":   {},
}

func isNonCallKeyword(name string) bool {
	_, ok := nonCallKeywords[name]
	return ok
}

func isCommentOnlyLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "//") ||
		strings.HasPrefix(trimmed, "#") ||
		strings.HasPrefix(trimmed, "/*") ||
		strings.HasPrefix(trimmed, "*") ||
		strings.HasPrefix(trimmed, "*/")
}

// Detects bad comma spacing without regex:
// 1) space before comma: foo(a , b)
// 2) two or more spaces after comma: foo(a,  b)
// 3) no space after comma: foo(a,b)
// Returns whether an issue was found and the comma's 0-based offset in args.
func hasBadCommaSpacing(args string) (bool, int) {
	parenDepth := 0
	bracketDepth := 0
	braceDepth := 0
	inSingleQuote := false
	inDoubleQuote := false
	inBacktick := false
	escaped := false

	for i := 0; i < len(args); i++ {
		ch := args[i]

		if escaped {
			escaped = false
			continue
		}

		if inSingleQuote {
			if ch == '\\' {
				escaped = true
			} else if ch == '\'' {
				inSingleQuote = false
			}
			continue
		}

		if inDoubleQuote {
			if ch == '\\' {
				escaped = true
			} else if ch == '"' {
				inDoubleQuote = false
			}
			continue
		}

		if inBacktick {
			if ch == '\\' {
				escaped = true
			} else if ch == '`' {
				inBacktick = false
			}
			continue
		}

		switch ch {
		case '\'':
			inSingleQuote = true
			continue
		case '"':
			inDoubleQuote = true
			continue
		case '`':
			inBacktick = true
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
		case '/':
			if i+1 < len(args) && args[i+1] == '/' {
				return false, -1
			}
			if i+1 < len(args) && args[i+1] == '*' {
				end := strings.Index(args[i+2:], "*/")
				if end != -1 {
					i = i + 2 + end + 1
					continue
				}
				return false, -1
			}
		}

		if parenDepth > 0 || bracketDepth > 0 || braceDepth > 0 {
			continue
		}

		if ch == ',' {
			if i > 0 && (args[i-1] == ' ' || args[i-1] == '\t') {
				return true, i
			}

			j := i + 1
			spaceCount := 0
			for j < len(args) && (args[j] == ' ' || args[j] == '\t') {
				spaceCount++
				j++
			}
			if spaceCount >= 2 {
				return true, i
			}
			if j >= len(args) {
				continue
			}
			if spaceCount == 0 {
				return true, i
			}
		}
	}

	return false, -1
}

// findMatchingParen finds the index of the matching closing parenthesis for line[openParenIdx],
// respecting strings, escapes, and nested brackets/braces/parentheses.
func findMatchingParen(line string, openParenIdx int) int {
	parenDepth := 1
	bracketDepth := 0
	braceDepth := 0
	inSingleQuote := false
	inDoubleQuote := false
	inBacktick := false
	escaped := false

	for j := openParenIdx + 1; j < len(line); j++ {
		ch := line[j]

		if escaped {
			escaped = false
			continue
		}

		if inSingleQuote {
			if ch == '\\' {
				escaped = true
			} else if ch == '\'' {
				inSingleQuote = false
			}
			continue
		}

		if inDoubleQuote {
			if ch == '\\' {
				escaped = true
			} else if ch == '"' {
				inDoubleQuote = false
			}
			continue
		}

		if inBacktick {
			if ch == '\\' {
				escaped = true
			} else if ch == '`' {
				inBacktick = false
			}
			continue
		}

		switch ch {
		case '\'':
			inSingleQuote = true
			continue
		case '"':
			inDoubleQuote = true
			continue
		case '`':
			inBacktick = true
			continue
		case '(':
			parenDepth++
			continue
		case ')':
			parenDepth--
			if parenDepth == 0 {
				return j
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
		case '/':
			if j+1 < len(line) && line[j+1] == '/' {
				return -1
			}
			if j+1 < len(line) && line[j+1] == '*' {
				end := strings.Index(line[j+2:], "*/")
				if end != -1 {
					j = j + 2 + end + 1
					continue
				}
				return -1
			}
		case '#':
			return -1
		}
	}
	return -1
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

		inSingleQuote := false
		inDoubleQuote := false
		inBacktick := false
		escaped := false

		for idx := 0; idx < len(line); {
			if escaped {
				escaped = false
				idx++
				continue
			}

			if inSingleQuote {
				if line[idx] == '\\' {
					escaped = true
				} else if line[idx] == '\'' {
					inSingleQuote = false
				}
				idx++
				continue
			}

			if inDoubleQuote {
				if line[idx] == '\\' {
					escaped = true
				} else if line[idx] == '"' {
					inDoubleQuote = false
				}
				idx++
				continue
			}

			if inBacktick {
				if line[idx] == '\\' {
					escaped = true
				} else if line[idx] == '`' {
					inBacktick = false
				}
				idx++
				continue
			}

			if line[idx] == '\'' {
				inSingleQuote = true
				idx++
				continue
			}
			if line[idx] == '"' {
				inDoubleQuote = true
				idx++
				continue
			}
			if line[idx] == '`' {
				inBacktick = true
				idx++
				continue
			}

			// Inline comment
			if line[idx] == '/' && idx+1 < len(line) && line[idx+1] == '/' {
				break
			}
			if line[idx] == '#' {
				break
			}
			if line[idx] == '/' && idx+1 < len(line) && line[idx+1] == '*' {
				end := strings.Index(line[idx+2:], "*/")
				if end != -1 {
					idx = idx + 2 + end + 2
					continue
				}
				break
			}

			start := idx
			for idx < len(line) && (isIdentChar(line[idx]) || (idx > start && isDigit(line[idx]))) {
				idx++
			}

			if start != idx {
				funcName := line[start:idx]
				parenIdx := idx
				for parenIdx < len(line) && (line[parenIdx] == ' ' || line[parenIdx] == '\t') {
					parenIdx++
				}

				if parenIdx < len(line) && line[parenIdx] == '(' {
					if !isNonCallKeyword(funcName) {
						beforeIdent := strings.TrimSpace(line[:start])
						isDecl := strings.HasSuffix(beforeIdent, "function") || strings.HasSuffix(beforeIdent, "fn")
						if !isDecl {
							closeParenIdx := findMatchingParen(line, parenIdx)
							if closeParenIdx != -1 {
								args := line[parenIdx+1 : closeParenIdx]
								if hasBad, commaOffset := hasBadCommaSpacing(args); hasBad {
									col := parenIdx + 1 + commaOffset + 1
									issues = append(issues, StyleIssue{
										Filename: filename,
										Line:     i + 1,
										Column:   col,
										Type:     Error,
										Fixable:  true,
										Message:  "Incorrect spacing between function call arguments",
										Code:     "Generic.Functions.FunctionCallArgumentSpacing",
									})
								}
								idx = closeParenIdx + 1
								continue
							}
						}
					}
				}
				idx = parenIdx
				continue
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

func (f FunctionCallArgumentSpacingFixer) Fix(content string) (fixedContent string) {
	fixedContent = content
	defer func() {
		if r := recover(); r != nil {
			fixedContent = content
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

	fixedContent = strings.Join(lines, "\n")
	return fixedContent
}

func fixFunctionCallSpacingInLine(line string) string {
	out := getBuilder()
	inSingleQuote := false
	inDoubleQuote := false
	inBacktick := false
	escaped := false

	for i := 0; i < len(line); {
		if escaped {
			escaped = false
			out.WriteByte(line[i])
			i++
			continue
		}

		if inSingleQuote {
			if line[i] == '\\' {
				escaped = true
			} else if line[i] == '\'' {
				inSingleQuote = false
			}
			out.WriteByte(line[i])
			i++
			continue
		}

		if inDoubleQuote {
			if line[i] == '\\' {
				escaped = true
			} else if line[i] == '"' {
				inDoubleQuote = false
			}
			out.WriteByte(line[i])
			i++
			continue
		}

		if inBacktick {
			if line[i] == '\\' {
				escaped = true
			} else if line[i] == '`' {
				inBacktick = false
			}
			out.WriteByte(line[i])
			i++
			continue
		}

		if line[i] == '\'' {
			inSingleQuote = true
			out.WriteByte(line[i])
			i++
			continue
		}
		if line[i] == '"' {
			inDoubleQuote = true
			out.WriteByte(line[i])
			i++
			continue
		}
		if line[i] == '`' {
			inBacktick = true
			out.WriteByte(line[i])
			i++
			continue
		}

		if line[i] == '/' && i+1 < len(line) && line[i+1] == '/' {
			out.WriteString(line[i:])
			break
		}
		if line[i] == '#' {
			out.WriteString(line[i:])
			break
		}

		start := i
		for i < len(line) && (isIdentChar(line[i]) || (i > start && isDigit(line[i]))) {
			i++
		}

		if start != i {
			funcName := line[start:i]
			parenIdx := i
			for parenIdx < len(line) && (line[parenIdx] == ' ' || line[parenIdx] == '\t') {
				parenIdx++
			}

			if parenIdx < len(line) && line[parenIdx] == '(' && !isNonCallKeyword(funcName) {
				beforeIdent := strings.TrimSpace(line[:start])
				isDecl := strings.HasSuffix(beforeIdent, "function") || strings.HasSuffix(beforeIdent, "fn")
				if !isDecl {
					closeParenIdx := findMatchingParen(line, parenIdx)
					if closeParenIdx != -1 {
						args := line[parenIdx+1 : closeParenIdx]
						fixedArgs := fixArgumentSpacing(args)
						out.WriteString(line[start:parenIdx] + "(" + fixedArgs + ")")
						i = closeParenIdx + 1
						continue
					}
				}
			}
			out.WriteString(line[start:i])
			continue
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
		inBacktick    bool
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

		if inBacktick {
			if i < len(args) && args[i] == '\\' {
				escaped = true
				continue
			}
			if i < len(args) && args[i] == '`' {
				inBacktick = false
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
			case '`':
				inBacktick = true
				continue
			case '(':
				parenDepth++
			case ')':
				if parenDepth > 0 {
					parenDepth--
				}
			case '[':
				bracketDepth++
			case ']':
				if bracketDepth > 0 {
					bracketDepth--
				}
			case '{':
				braceDepth++
			case '}':
				if braceDepth > 0 {
					braceDepth--
				}
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
