package parser

import (
	"context"
	"strings"
	"testing"

	"github.com/ayanozturk/go-php-parser/lexer"
)

const maxFuzzSourceBytes = 64 << 10

var malformedPHPSeeds = []string{
	"",
	"<?php",
	"<?php if () {",
	"<?php function broken(] { return (((; }",
	"<?phpA[0(00",
	"<?php $value = <<<'TXT'\nunterminated",
	"<?php /* unterminated comment",
	"<?php class Example { public function run(): never { throw new \\RuntimeException(); }",
	string([]byte("<?php \xff\xfe\x00")),
}

func FuzzParserMalformedPHP(f *testing.F) {
	for _, seed := range malformedPHPSeeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, source string) {
		if len(source) > maxFuzzSourceBytes {
			t.Skip()
		}

		p := New(lexer.New(source), false)
		_ = p.Parse()
		assertNoRecoveredParserPanic(t, p.Errors())
	})
}

func FuzzParserCancellation(f *testing.F) {
	for _, seed := range malformedPHPSeeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, source string) {
		if len(source) > maxFuzzSourceBytes {
			t.Skip()
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		p := New(lexer.New("<?php\n$sentinel = 1;\n"+source), false)
		p.Ctx = ctx
		_ = p.Parse()

		errors := p.Errors()
		assertNoRecoveredParserPanic(t, errors)
		for _, parseErr := range errors {
			if strings.Contains(parseErr, "parser context cancelled: context canceled") {
				return
			}
		}
		t.Fatalf("pre-cancelled parse did not report cancellation: %v", errors)
	})
}

func assertNoRecoveredParserPanic(t *testing.T, errors []string) {
	t.Helper()
	for _, parseErr := range errors {
		if strings.HasPrefix(parseErr, "Parser panic:") {
			t.Fatalf("parser recovered an internal panic: %s", parseErr)
		}
	}
}
