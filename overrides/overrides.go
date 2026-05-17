package overrides

import (
	"fmt"
	"regexp"
	"strings"
)

type RuleOverride struct {
	Classes []string `yaml:"classes" json:"classes"`
}

type RuleOverrides map[string]RuleOverride

type Compiled struct {
	rules map[string]compiledRuleOverride
}

type compiledRuleOverride struct {
	classPatterns []*regexp.Regexp
}

func Compile(raw RuleOverrides) (*Compiled, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	compiled := &Compiled{rules: make(map[string]compiledRuleOverride, len(raw))}
	for code, override := range raw {
		classPatterns, err := compilePatterns(override.Classes)
		if err != nil {
			return nil, fmt.Errorf("compile override for %s: %w", code, err)
		}
		if len(classPatterns) == 0 {
			continue
		}
		compiled.rules[code] = compiledRuleOverride{classPatterns: classPatterns}
	}
	if len(compiled.rules) == 0 {
		return nil, nil
	}

	return compiled, nil
}

func (c *Compiled) IgnoreIssue(ruleCode, subjectKind, subjectName string) bool {
	if c == nil || subjectName == "" {
		return false
	}

	rule, ok := c.rules[ruleCode]
	if !ok {
		return false
	}

	switch subjectKind {
	case "class":
		return matchesAny(rule.classPatterns, subjectName)
	default:
		return false
	}
}

func compilePatterns(patterns []string) ([]*regexp.Regexp, error) {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		normalized := normalizePattern(pattern)
		if normalized == "" {
			continue
		}
		re, err := regexp.Compile(normalized)
		if err != nil {
			return nil, fmt.Errorf("invalid regex %q: %w", pattern, err)
		}
		compiled = append(compiled, re)
	}
	return compiled, nil
}

func normalizePattern(pattern string) string {
	trimmed := strings.TrimSpace(pattern)
	if trimmed == "" {
		return ""
	}
	if len(trimmed) >= 2 && strings.HasPrefix(trimmed, "/") && strings.HasSuffix(trimmed, "/") {
		return trimmed[1 : len(trimmed)-1]
	}
	return "^" + regexp.QuoteMeta(trimmed) + "$"
}

func matchesAny(patterns []*regexp.Regexp, value string) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(value) {
			return true
		}
	}
	return false
}
