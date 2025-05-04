package style

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// PrintPHPCSStyleOutput prints grouped style issues in PHPCS format to stdout
func PrintPHPCSStyleOutput(issues []StyleIssue) {
	PrintPHPCSStyleOutputToWriter(os.Stdout, issues)
}

// PrintPHPCSStyleOutputToWriter prints grouped style issues in PHPCS format to the given io.Writer
func PrintPHPCSStyleOutputToWriter(w io.Writer, issues []StyleIssue) {
	if len(issues) == 0 {
		fmt.Fprintln(w, "\033[32;1mNo style errors or warnings found. Your code is clean!\033[0m")
		return
	}
	fileMap := groupIssuesByFile(issues)
	files := sortedFileNames(fileMap)
	totalErrors := 0
	for _, file := range files {
		fileIssues := fileMap[file]
		errCount, warnCount, totalLines := countFileIssues(fileIssues)
		totalErrors += errCount
		printFileIssues(w, file, fileIssues, errCount, warnCount, totalLines)
	}
	fmt.Fprintf(w, "Run summary: total style errors found: %d\n", totalErrors)
}

func groupIssuesByFile(issues []StyleIssue) map[string][]StyleIssue {
	fileMap := map[string][]StyleIssue{}
	for _, iss := range issues {
		fileMap[iss.Filename] = append(fileMap[iss.Filename], iss)
	}
	return fileMap
}

func sortedFileNames(fileMap map[string][]StyleIssue) []string {
	files := make([]string, 0, len(fileMap))
	for f := range fileMap {
		files = append(files, f)
	}
	sort.Strings(files)
	return files
}

func countFileIssues(fileIssues []StyleIssue) (errCount, warnCount, totalLines int) {
	lineSet := map[int]struct{}{}
	for _, iss := range fileIssues {
		if iss.Type == Error {
			errCount++
		} else {
			warnCount++
		}
		lineSet[iss.Line] = struct{}{}
	}
	totalLines = len(lineSet)
	return
}

func printFileIssues(w io.Writer, file string, fileIssues []StyleIssue, errCount, warnCount, totalLines int) {
	fmt.Fprintf(w, "FILE: %s\n", file)
	fmt.Fprintln(w, strings.Repeat("-", 80))
	fmt.Fprintf(w, "FOUND %d ERRORS AND %d WARNING%s AFFECTING %d LINE%s\n", errCount, warnCount, plural(warnCount), totalLines, plural(totalLines))
	fmt.Fprintln(w, strings.Repeat("-", 80))
	sort.Slice(fileIssues, func(i, j int) bool {
		if fileIssues[i].Line == fileIssues[j].Line {
			return fileIssues[i].Column < fileIssues[j].Column
		}
		return fileIssues[i].Line < fileIssues[j].Line
	})
	for _, iss := range fileIssues {
		fmt.Fprintf(w, "%4d | %-7s | %s\n", iss.Line, iss.Type, iss.Message)
		if iss.Code != "" {
			fmt.Fprintf(w, "     |         |     (%s)\n", iss.Code)
		}
	}
	fmt.Fprintln(w)
}

// PrintPHPCSStyleIssueToWriter prints a single StyleIssue in PHPCS format to the provided writer.
func PrintPHPCSStyleIssueToWriter(w io.Writer, iss StyleIssue) {
	fmt.Fprintf(w, "%4d | %-7s | %s\n", iss.Line, iss.Type, iss.Message)
	if iss.Code != "" {
		fmt.Fprintf(w, "     |         |     (%s)\n", iss.Code)
	}
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "S"
}
