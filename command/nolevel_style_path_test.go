package command

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/ayanozturk/go-php-parser/syntax"
)

// TestNoLevelStylePathSuppressesResolverDependentRules is the regression lock
// for the resolver-semantics finding: the no-`--level` style path
// (runAnalysis with configuredAnalysisLevel==nil and project==nil) must leave
// ctx.Resolver nil, exactly as the historic RunAnalysisRules(...) call did.
// Building a Resolver there eagerly activates resolver-dependent rules
// (A.DEPRECATED.CALL, A.ARG.TYPE, A.PROP.TYPE, ...) that previously stayed
// suppressed. Each fixture is same-file resolvable, so with a Resolver present
// the named rule WOULD fire (verified out of band against the resolver-present
// branch); with the correct nil-Resolver semantics it must not.
func TestNoLevelStylePathSuppressesResolverDependentRules(t *testing.T) {
	prev := configuredAnalysisLevel
	configuredAnalysisLevel = nil
	defer func() { configuredAnalysisLevel = prev }()

	cases := []struct {
		name       string
		src        string
		absentCode string
	}{
		{"deprecated_fn.php", "<?php\n/** @deprecated use bar */\nfunction foo() {}\nfoo();\n", "A.DEPRECATED.CALL"},
		{"arg_type_fn.php", "<?php\nfunction takesInt(int $x) {}\ntakesInt('str');\n", "A.ARG.TYPE"},
		{"prop_type_obj.php", "<?php\nclass A { public int $n; }\nfunction f(A $a) { $a->n = 'x'; }\n", "A.PROP.TYPE"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			content := []byte(tc.src)
			nodes, diags := syntax.ParseAST(content)
			if len(diags) > 0 {
				t.Fatalf("parse: %v", diags)
			}
			issues := runAnalysis(tc.name, nodes, content, nil)
			ran := false
			for _, is := range issues {
				if is.Code == tc.absentCode {
					t.Fatalf("%s: resolver-dependent rule %s must stay suppressed on the no-level style path (nil Resolver); got issues=%+v", tc.name, tc.absentCode, issues)
				}
				if strings.HasPrefix(is.Code, "Level6.") || is.Code == "PSR1.Files.SideEffects" {
					ran = true
				}
			}
			if !ran {
				t.Fatalf("%s: expected in-file diagnostics proving analysis ran, got none: %+v", tc.name, issues)
			}
		})
	}
}

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
