package style

import "go-phpcs/overrides"

func FilterIssues(issues []StyleIssue, matcher *overrides.Compiled) []StyleIssue {
	if matcher == nil {
		return issues
	}

	filtered := make([]StyleIssue, 0, len(issues))
	for _, issue := range issues {
		if matcher.IgnoreIssue(issue.Code, issue.SubjectKind, issue.SubjectName) {
			continue
		}
		filtered = append(filtered, issue)
	}
	return filtered
}
