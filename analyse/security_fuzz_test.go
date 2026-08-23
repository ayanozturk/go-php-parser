package analyse

import (
	"strings"
	"testing"

	"github.com/ayanozturk/go-php-parser/lexer"
	"github.com/ayanozturk/go-php-parser/parser"
)

const maxAnalysisFuzzBytes = 32 << 10

var typeFuzzSeeds = []string{
	"",
	"int|string|null",
	"array<string, list<int>>",
	"array{open: string, nested: array<int, string>}",
	"(Alpha&Beta)|null",
	"class-string<Entity>",
	strings.Repeat("Nested<", 64) + "Value" + strings.Repeat(">", 64),
	string([]byte{0xff, 0xfe, 0x00}),
}

func FuzzParseType(f *testing.F) {
	for _, seed := range typeFuzzSeeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, raw string) {
		if len(raw) > maxAnalysisFuzzBytes {
			t.Skip()
		}

		first := ParseType(raw).String()
		if second := ParseType(raw).String(); second != first {
			t.Fatalf("type parsing is not deterministic: first %q, second %q", first, second)
		}
	})
}

func FuzzMalformedPHPDocRuleExecution(f *testing.F) {
	for _, seed := range typeFuzzSeeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, raw string) {
		if len(raw) > maxAnalysisFuzzBytes {
			t.Skip()
		}

		source := "<?php\n/**\n * @param " + raw + " $value\n * @return " + raw + "\n */\nfunction securityFuzz($value) { return $value; }\n"
		p := parser.New(lexer.New(source), false)
		nodes := p.Parse()
		for _, parseErr := range p.Errors() {
			if strings.HasPrefix(parseErr, "Parser panic:") {
				t.Fatalf("parser recovered an internal panic: %s", parseErr)
			}
		}
		_ = RunAnalysisRules("security-fuzz.php", nodes)
	})
}
