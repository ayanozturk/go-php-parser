package style

import (
	"go-phpcs/ast"
	"go-phpcs/style/helper"
)

const methodVisibilityDeclaredCode = "PSR12.Methods.VisibilityDeclared"

type MethodVisibilityDeclaredChecker struct{}

func (c *MethodVisibilityDeclaredChecker) CheckIssues(lines []string, filename string) []StyleIssue {
	var issues []StyleIssue
	inClass := false
	braceDepth := 0
	commentState := &helper.CommentState{}
	for i, line := range lines {
		codeLine := codeOnlyLine(line, commentState)
		trimmed := helper.TrimWhitespace(codeLine)
		if shouldSkipLine(trimmed) {
			continue
		}
		if helper.IsClassDeclaration(trimmed) {
			inClass = true
			continue
		}
		if inClass {
			braceDepth, inClass = updateBraceState(trimmed, braceDepth, inClass)
			if !inClass {
				continue
			}
			if isMethodDeclaration(trimmed) && !hasVisibility(trimmed) {
				issues = append(issues, StyleIssue{
					Filename: filename,
					Line:     i + 1,
					Type:     Error,
					Fixable:  false,
					Message:  "Visibility must be declared on all class methods",
					Code:     methodVisibilityDeclaredCode,
				})
			}
		}
	}
	return issues
}

func shouldSkipLine(line string) bool {
	return len(line) > 0 && (line[0] == '/' || line[0] == '*')
}

func codeOnlyLine(line string, commentState *helper.CommentState) string {
	if helper.HandleHeredocEnd(line, commentState) {
		return ""
	}

	quoteState := &helper.QuoteState{}
	out := []byte(line)
	for j := 0; j < len(line); {
		if commentState.InBlockComment {
			out[j] = ' '
			next := helper.HandleBlockComment(line, j, commentState)
			for k := j + 1; k < next && k < len(out); k++ {
				out[k] = ' '
			}
			j = next
			continue
		}

		if !quoteState.InSingle && !quoteState.InDouble {
			if j+1 < len(line) && line[j] == '/' && line[j+1] == '/' {
				blankBytes(out, j, len(out))
				break
			}
			if line[j] == '#' {
				blankBytes(out, j, len(out))
				break
			}
			if next := helper.HandleBlockComment(line, j, commentState); next != j {
				blankBytes(out, j, next)
				j = next
				continue
			}
			if next := helper.HandleHeredocStart(line, j, commentState); next != j {
				blankBytes(out, j, len(out))
				break
			}
		}

		if quoteState.InSingle || quoteState.InDouble {
			out[j] = ' '
		}
		next := helper.HandleQuotes(line, j, quoteState)
		if next != j {
			blankBytes(out, j, next)
			j = next
			continue
		}
		j++
	}

	return string(out)
}

func blankBytes(out []byte, start int, end int) {
	for i := start; i < end && i < len(out); i++ {
		out[i] = ' '
	}
}

func updateBraceState(line string, braceDepth int, inClass bool) (int, bool) {
	if line == "{" {
		braceDepth++
		return braceDepth, inClass
	}
	if line == "}" {
		braceDepth--
		if braceDepth <= 0 {
			return braceDepth, false
		}
		return braceDepth, inClass
	}
	return braceDepth, inClass
}

func isMethodDeclaration(line string) bool {
	idx := helper.IndexOfWord(line, "function")
	if idx == -1 {
		return false
	}
	// `$function` — the word "function" is preceded by `$`, making it a
	// variable name, not the keyword.
	if idx > 0 && line[idx-1] == '$' {
		return false
	}
	n := idx + len("function")
	for n < len(line) && (line[n] == ' ' || line[n] == '\t') {
		n++
	}
	if n < len(line) && line[n] == '(' {
		return false
	}
	return true
}

func hasVisibility(line string) bool {
	return helper.HasWord(line, "public") || helper.HasWord(line, "protected") || helper.HasWord(line, "private")
}

func init() {
	RegisterRule(methodVisibilityDeclaredCode, func(filename string, content []byte, _ []ast.Node) []StyleIssue {
		lines := SplitLinesCached(content)
		checker := &MethodVisibilityDeclaredChecker{}
		return checker.CheckIssues(lines, filename)
	})
}
