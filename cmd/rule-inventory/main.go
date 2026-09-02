package main

import (
	"fmt"
	"strings"

	"github.com/ayanozturk/go-php-parser/analyse"
)

const (
	maxPHPStanLevel = 10
	tableStart      = "<!-- analysis-rule-level-table:start -->"
	tableEnd        = "<!-- analysis-rule-level-table:end -->"
)

func main() {
	fmt.Println(renderRuleLevelTable(analyse.ListRegisteredAnalysisRuleMetadata()))
}

func renderRuleLevelTable(rules []analyse.AnalysisRuleMeta) string {
	introduced := make([]int, maxPHPStanLevel+1)
	unlevelled := 0
	for _, rule := range rules {
		if rule.Level < 0 || rule.Level > maxPHPStanLevel {
			unlevelled++
			continue
		}
		introduced[rule.Level]++
	}

	var builder strings.Builder
	builder.WriteString(tableStart + "\n")
	builder.WriteString("| PHPStan level | Rules introduced | Cumulative levelled rules | Detail |\n")
	builder.WriteString("| ---: | ---: | ---: | --- |\n")
	cumulative := 0
	for level, count := range introduced {
		cumulative += count
		fmt.Fprintf(&builder, "| %d | %d | %d | [Level %d rules](docs/rules/level-%d.md) |\n", level, count, cumulative, level, level)
	}
	fmt.Fprintf(&builder, "| Unlevelled | %d | %d total registered | [Unlevelled rules](docs/rules/unlevelled.md) |\n", unlevelled, cumulative+unlevelled)
	builder.WriteString(tableEnd)
	return builder.String()
}
