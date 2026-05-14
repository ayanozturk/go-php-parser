package analyse

import "go-phpcs/overrides"

func FilterIssues(issues []AnalysisIssue, matcher *overrides.Compiled) []AnalysisIssue {
	if matcher == nil {
		return issues
	}

	filtered := make([]AnalysisIssue, 0, len(issues))
	for _, issue := range issues {
		if matcher.IgnoreIssue(issue.Code, issue.SubjectKind, issue.SubjectName) {
			continue
		}
		filtered = append(filtered, issue)
	}
	return filtered
}
