package command

import (
	"fmt"
	"sort"
	"testing"

	"github.com/ayanozturk/go-php-parser/syntax"
)

// TestNoLevelStylePathProducesLevelDiagnostics exercises the no-`--level`
// `style` command path end-to-end: runAnalysis with configuredAnalysisLevel==nil
// and project==nil (the exact shape used at ProcessFileWithErrors' commandName
// == "style" call site). It locks in the Level0/Level1 diagnostics this path
// emits so the CST-direct migration of runAnalysis (which now always sets
// ctx.Content) stays behaviour-identical to the old ast-fallback path this
// branch used to take.
func TestNoLevelStylePathProducesLevelDiagnostics(t *testing.T) {
	prev := configuredAnalysisLevel
	configuredAnalysisLevel = nil
	defer func() { configuredAnalysisLevel = prev }()

	cases := []struct {
		name string
		src  string
		want []string
	}{
		{"missing_parent.php", "<?php\nclass C extends Missing {}\n", []string{
			"Level0.ClassModel|2:1|Class C extends unknown class Missing.",
		}},
		{"undef_func.php", "<?php\nfoo();\n", []string{
			"Level0.Symbols|2:1|Function foo not found.",
		}},
		{"dup_key.php", "<?php\n$a = [1 => 'x', 1 => 'y'];\n", []string{
			"Level0.Language|2:17|Array has 1 duplicate key.",
		}},
		{"final_abstract.php", "<?php\nfinal abstract class C {}\n", []string{
			"Level0.ClassModel|2:1|Class C cannot be both final and abstract.",
		}},
		{"clean.php", "<?php\nclass C {}\n", nil},
		{"bad_regex.php", "<?php\npreg_match('/[/', $s);\n", []string{
			"Level0.Language|2:1|Regex pattern is invalid: error parsing regexp: missing closing ]: `[`",
			"Level1.Variables|2:19|Variable $s might not be defined.",
		}},
		{"void_cast.php", "<?php\n$x = (unset) $y;\n", []string{
			"Level0.Language|2:6|Cannot cast to unset.",
			"Level1.Variables|2:14|Variable $y might not be defined.",
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			content := []byte(tc.src)
			nodes, _ := syntax.ParseAST(content)
			issues := runAnalysis(tc.name, nodes, content, nil)
			got := make([]string, 0, len(issues))
			for _, is := range issues {
				got = append(got, fmt.Sprintf("%s|%d:%d|%s", is.Code, is.Line, is.Column, is.Message))
			}
			sort.Strings(got)
			want := append([]string(nil), tc.want...)
			sort.Strings(want)
			if len(got) != len(want) {
				t.Fatalf("%s: got %d issues %v, want %d %v", tc.name, len(got), got, len(want), want)
			}
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("%s: issue %d = %q, want %q", tc.name, i, got[i], want[i])
				}
			}
		})
	}
}
