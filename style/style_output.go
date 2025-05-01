package style

import (
	"fmt"
	"sort"
	"strings"
)

// PrintPHPCSStyleOutput prints grouped style issues in PHPCS format
func PrintPHPCSStyleOutput(issues []StyleIssue) {
	if len(issues) == 0 {
		return
	}
	// Group by file
	fileMap := map[string][]StyleIssue{}
	for _, iss := range issues {
		fileMap[iss.Filename] = append(fileMap[iss.Filename], iss)
	}

	files := make([]string, 0, len(fileMap))
	for f := range fileMap {
		files = append(files, f)
	}
	sort.Strings(files)

	for _, file := range files {
		fileIssues := fileMap[file]
		errCount, warnCount := 0, 0
		lineSet := map[int]struct{}{}
		for _, iss := range fileIssues {
			if iss.Type == Error {
				errCount++
			} else {
				warnCount++
			}
			lineSet[iss.Line] = struct{}{}
		}
		totalLines := len(lineSet)
		fmt.Printf("FILE: %s\n", file)
		fmt.Println(strings.Repeat("-", 80))
		fmt.Printf("FOUND %d ERRORS AND %d WARNING%s AFFECTING %d LINE%s\n", errCount, warnCount, plural(warnCount), totalLines, plural(totalLines))
		fmt.Println(strings.Repeat("-", 80))
		// Sort by line
		sort.Slice(fileIssues, func(i, j int) bool {
			if fileIssues[i].Line == fileIssues[j].Line {
				return fileIssues[i].Column < fileIssues[j].Column
			}
			return fileIssues[i].Line < fileIssues[j].Line
		})
		for _, iss := range fileIssues {
			fix := "[ ]"
			if iss.Fixable {
				fix = "[x]"
			}
			fmt.Printf("%4d | %-7s | %s %s\n", iss.Line, iss.Type, fix, iss.Message)
			if iss.Code != "" {
				fmt.Printf("     |         |     (%s)\n", iss.Code)
			}
		}
		fmt.Println()
	}
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "S"
}
